package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/startower-observability/blackcat/llm"
	"github.com/startower-observability/blackcat/types"
)

const (
	// DefaultGeminiBaseURL is the base URL for the official Gemini API.
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiBackend implements llm.Backend using the official Google Gemini API.
// It reuses the Gemini codec (EncodeMessages/DecodeResponse) for wire format
// and handles HTTP transport + API key authentication.
type GeminiBackend struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	temp    *float64
	maxTok  *int
}

// NewGeminiBackend creates a new Gemini Backend from BackendConfig.
func NewGeminiBackend(cfg llm.BackendConfig) (llm.Backend, error) {
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.TokenSource != nil {
		token, err := cfg.TokenSource()
		if err != nil {
			return nil, fmt.Errorf("gemini: get API key: %w", err)
		}
		apiKey = token
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: API key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	b := &GeminiBackend{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{},
	}
	if cfg.Temperature > 0 {
		t := cfg.Temperature
		b.temp = &t
	}
	if cfg.MaxTokens > 0 {
		m := cfg.MaxTokens
		b.maxTok = &m
	}

	return b, nil
}

// Chat sends a non-streaming request to the Gemini API.
func (b *GeminiBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	geminiReq := EncodeMessages(messages, tools)
	b.applyGenConfig(geminiReq)

	endpoint := fmt.Sprintf("%s/models/%s:generateContent", b.baseURL, b.model)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}

	return DecodeResponse(&geminiResp, b.model)
}

// Stream sends a streaming request to the Gemini API and returns a channel of chunks.
func (b *GeminiBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	geminiReq := EncodeMessages(messages, tools)
	b.applyGenConfig(geminiReq)

	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", b.baseURL, b.model)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: stream HTTP %d: %s", resp.StatusCode, string(body))
	}

	chunks := make(chan types.Chunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE format: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- types.Chunk{Done: true}
				return
			}

			var geminiResp GeminiResponse
			if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
				continue
			}

			llmResp, err := DecodeResponse(&geminiResp, b.model)
			if err != nil {
				continue
			}

			if llmResp.Content != "" {
				chunks <- types.Chunk{Content: llmResp.Content}
			}
			if len(llmResp.ToolCalls) > 0 {
				chunks <- types.Chunk{ToolCalls: llmResp.ToolCalls}
			}
		}

		chunks <- types.Chunk{Done: true}
	}()

	return chunks, nil
}

// Info returns metadata about this backend.
func (b *GeminiBackend) Info() llm.BackendInfo {
	return llm.BackendInfo{
		Name:       "gemini",
		Models:     []string{b.model},
		AuthMethod: "api-key",
	}
}

// applyGenConfig sets generation parameters on the request if configured.
func (b *GeminiBackend) applyGenConfig(req *GeminiRequest) {
	if b.temp != nil || b.maxTok != nil {
		if req.GenerationConfig == nil {
			req.GenerationConfig = &GeminiGenConfig{}
		}
		if b.temp != nil {
			req.GenerationConfig.Temperature = b.temp
		}
		if b.maxTok != nil {
			req.GenerationConfig.MaxOutputTokens = b.maxTok
		}
	}
}
