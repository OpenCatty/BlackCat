// Package antigravity provides the Google Antigravity (Cloud Code) LLM backend
// for BlackCat. It wraps Gemini wire format in a Cloud Code envelope and
// authenticates via Google OAuth PKCE flow.
//
// WARNING: Antigravity uses an internal Google API that may violate Google's
// Terms of Service. Use at your own risk. Users must explicitly accept the ToS
// risk via config (oauth.antigravity.acceptedToS = true).
package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/llm/gemini"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	// DefaultAntigravityEndpoint is the Cloud Code API base URL.
	DefaultAntigravityEndpoint = "https://cloudcode-pa.googleapis.com"

	// GenerateContentPath is the non-streaming API path.
	GenerateContentPath = "/v1internal:generateContent"

	// StreamGenerateContentPath is the streaming API path.
	StreamGenerateContentPath = "/v1internal:streamGenerateContent"

	// DefaultAntigravityClientID is the Google OAuth client ID for Antigravity.
	DefaultAntigravityClientID = "" // Set via config

	// DefaultAntigravityClientSecret is the Google OAuth client secret.
	DefaultAntigravityClientSecret = "" // Set via config

	// DefaultAuthURL is the Google OAuth authorization URL.
	DefaultAuthURL = "https://accounts.google.com/o/oauth2/auth"

	// DefaultTokenURL is the Google OAuth token URL.
	DefaultTokenURL = "https://oauth2.googleapis.com/token"

	// DefaultRedirectURL is the local callback URL for PKCE flow.
	DefaultRedirectURL = "http://127.0.0.1:51121/oauth-callback"
)

// AntigravityEnvelope is the wrapper envelope for Antigravity API requests.
type AntigravityEnvelope struct {
	Project   string                `json:"project"`
	Model     string                `json:"model"`
	Request   *gemini.GeminiRequest `json:"request"`
	UserAgent string                `json:"userAgent"`
	RequestID string                `json:"requestId"`
}

// AntigravityBackend implements llm.Backend for Google Antigravity (Cloud Code).
type AntigravityBackend struct {
	accessToken func() (string, error)
	model       string
	endpoint    string
	project     string
	httpClient  *http.Client
	acceptedToS bool
	temp        *float64
	maxTok      *int
}

// AntigravityConfig extends BackendConfig with Antigravity-specific settings.
type AntigravityConfig struct {
	llm.BackendConfig
	AcceptedToS bool   // Must be true to use Antigravity
	Project     string // Cloud Code project ID (auto-detected if empty)
}

// NewAntigravityBackend creates a new Antigravity backend.
// Returns an error if AcceptedToS is false.
func NewAntigravityBackend(cfg AntigravityConfig) (llm.Backend, error) {
	if !cfg.AcceptedToS {
		return nil, fmt.Errorf("antigravity: Google Antigravity uses an internal API that may violate Google's Terms of Service. " +
			"To accept this risk, set oauth.antigravity.acceptedToS = true in your config")
	}

	tokenSource := cfg.TokenSource
	if tokenSource == nil && cfg.APIKey != "" {
		staticToken := cfg.APIKey
		tokenSource = func() (string, error) { return staticToken, nil }
	}
	if tokenSource == nil {
		return nil, fmt.Errorf("antigravity: OAuth token is required (run 'blackcat configure' to set up Antigravity)")
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-2.5-pro"
	}

	endpoint := cfg.BaseURL
	if endpoint == "" {
		endpoint = DefaultAntigravityEndpoint
	}

	b := &AntigravityBackend{
		accessToken: tokenSource,
		model:       model,
		endpoint:    strings.TrimRight(endpoint, "/"),
		project:     cfg.Project,
		httpClient:  &http.Client{},
		acceptedToS: true,
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

// Chat sends a non-streaming request to the Antigravity API.
func (b *AntigravityBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	token, err := b.accessToken()
	if err != nil {
		return nil, fmt.Errorf("antigravity: get access token: %w", err)
	}

	// Encode messages using Gemini codec
	geminiReq := gemini.EncodeMessages(messages, tools)
	b.applyGenConfig(geminiReq)

	// Wrap in Antigravity envelope
	envelope := AntigravityEnvelope{
		Project:   b.project,
		Model:     b.model,
		Request:   geminiReq,
		UserAgent: "antigravity",
		RequestID: uuid.New().String(),
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("antigravity: marshal request: %w", err)
	}

	url := b.endpoint + GenerateContentPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("antigravity: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antigravity: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("antigravity: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("antigravity: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp gemini.GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("antigravity: decode response: %w", err)
	}

	return gemini.DecodeResponse(&geminiResp, b.model)
}

// Stream sends a streaming request to the Antigravity API.
func (b *AntigravityBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	token, err := b.accessToken()
	if err != nil {
		return nil, fmt.Errorf("antigravity: get access token: %w", err)
	}

	geminiReq := gemini.EncodeMessages(messages, tools)
	b.applyGenConfig(geminiReq)

	envelope := AntigravityEnvelope{
		Project:   b.project,
		Model:     b.model,
		Request:   geminiReq,
		UserAgent: "antigravity",
		RequestID: uuid.New().String(),
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("antigravity: marshal stream request: %w", err)
	}

	url := b.endpoint + StreamGenerateContentPath + "?alt=sse"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("antigravity: create stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("antigravity: stream request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("antigravity: stream HTTP %d: %s", resp.StatusCode, string(body))
	}

	chunks := make(chan types.Chunk)
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- types.Chunk{Done: true}
				return
			}

			var geminiResp gemini.GeminiResponse
			if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
				continue
			}

			llmResp, err := gemini.DecodeResponse(&geminiResp, b.model)
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
func (b *AntigravityBackend) Info() llm.BackendInfo {
	return llm.BackendInfo{
		Name:       "antigravity",
		Models:     []string{b.model},
		AuthMethod: "oauth-pkce",
	}
}

// applyGenConfig sets generation parameters on the inner Gemini request.
func (b *AntigravityBackend) applyGenConfig(req *gemini.GeminiRequest) {
	if b.temp != nil || b.maxTok != nil {
		if req.GenerationConfig == nil {
			req.GenerationConfig = &gemini.GeminiGenConfig{}
		}
		if b.temp != nil {
			req.GenerationConfig.Temperature = b.temp
		}
		if b.maxTok != nil {
			req.GenerationConfig.MaxOutputTokens = b.maxTok
		}
	}
}
