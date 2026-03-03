// Package copilot provides the GitHub Copilot LLM backend for BlackCat.
// It implements the two-token architecture: OAuth token (long-lived) + Copilot
// API token (short-lived, ~30min) with automatic refresh.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	// DefaultCopilotChatEndpoint is the Copilot chat completions API base URL.
	DefaultCopilotChatEndpoint = "https://api.githubcopilot.com/chat/completions"

	// CopilotTokenEndpoint is the endpoint to exchange OAuth token for Copilot API token.
	CopilotTokenEndpoint = "https://api.github.com/copilot_internal/v2/token"

	// GitHubDeviceCodeURL is the GitHub device code OAuth endpoint.
	GitHubDeviceCodeURL = "https://github.com/login/device/code"

	// GitHubTokenURL is the GitHub OAuth token endpoint.
	GitHubTokenURL = "https://github.com/login/oauth/access_token"

	// DefaultCopilotClientID is the VS Code client ID used for device flow.
	DefaultCopilotClientID = "01ab8ac9400c4e429b23"

	// tokenRefreshBuffer is how early (before expiry) to refresh the Copilot API token.
	tokenRefreshBuffer = 60 * time.Second

	// Required headers for Copilot API requests.
	headerUserAgent     = "GitHubCopilotChat/0.37.5"
	headerEditorVersion = "vscode/1.109.2"
	headerIntegrationID = "vscode-chat"
	headerPluginVersion = "copilot-chat/0.37.5"
	headerOpenAIIntent  = "conversation-panel"
)

// copilotTokenResponse is the response from the Copilot token exchange endpoint.
type copilotTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"` // Unix timestamp
}

// CopilotBackend implements llm.Backend for GitHub Copilot.
// It manages the two-token architecture with automatic Copilot API token refresh.
type CopilotBackend struct {
	mu sync.Mutex

	// OAuth token (long-lived, from device flow)
	oauthToken string

	// Copilot API token (short-lived, ~30min)
	apiToken       string
	apiTokenExpiry time.Time

	// Configuration
	model         string
	temperature   float32
	maxTokens     int
	chatEndpoint  string
	tokenEndpoint string

	// HTTP client for token exchange
	httpClient *http.Client
}

// NewCopilotBackend creates a new Copilot backend from BackendConfig.
// The APIKey field should contain the GitHub OAuth token obtained via device flow.
// Alternatively, TokenSource can provide the OAuth token dynamically.
func NewCopilotBackend(cfg llm.BackendConfig) (llm.Backend, error) {
	oauthToken := cfg.APIKey
	if oauthToken == "" && cfg.TokenSource != nil {
		token, err := cfg.TokenSource()
		if err != nil {
			return nil, fmt.Errorf("copilot: get OAuth token: %w", err)
		}
		oauthToken = token
	}
	if oauthToken == "" {
		return nil, fmt.Errorf("copilot: OAuth token is required (run 'blackcat configure' to set up Copilot)")
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4.1"
	}

	chatEndpoint := cfg.BaseURL
	if chatEndpoint == "" {
		chatEndpoint = DefaultCopilotChatEndpoint
	}

	return &CopilotBackend{
		oauthToken:    oauthToken,
		model:         model,
		temperature:   float32(cfg.Temperature),
		maxTokens:     cfg.MaxTokens,
		chatEndpoint:  chatEndpoint,
		tokenEndpoint: CopilotTokenEndpoint,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Chat sends a non-streaming chat completion request to the Copilot API.
func (b *CopilotBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	apiToken, err := b.getAPIToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("copilot: get API token: %w", err)
	}

	// Create go-openai client with Copilot endpoint and custom transport
	config := openai.DefaultConfig(apiToken)
	config.BaseURL = strings.TrimSuffix(b.chatEndpoint, "/chat/completions")
	config.HTTPClient = &http.Client{
		Transport: &copilotTransport{apiToken: apiToken},
		Timeout:   30 * time.Second,
	}

	client := openai.NewClientWithConfig(config)

	req := openai.ChatCompletionRequest{
		Model:       b.model,
		Messages:    convertToOpenAI(messages),
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		Tools:       convertTools(tools),
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("copilot: chat: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &types.LLMResponse{Model: resp.Model, Usage: convertUsage(resp.Usage)}, nil
	}

	message := resp.Choices[0].Message
	return &types.LLMResponse{
		Content:   message.Content,
		ToolCalls: convertToolCalls(message.ToolCalls),
		Model:     resp.Model,
		Usage:     convertUsage(resp.Usage),
	}, nil
}

// Stream sends a streaming chat completion request to the Copilot API.
func (b *CopilotBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	apiToken, err := b.getAPIToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("copilot: get API token: %w", err)
	}

	config := openai.DefaultConfig(apiToken)
	config.BaseURL = strings.TrimSuffix(b.chatEndpoint, "/chat/completions")
	config.HTTPClient = &http.Client{
		Transport: &copilotTransport{apiToken: apiToken},
		Timeout:   0, // No timeout for streaming
	}

	client := openai.NewClientWithConfig(config)

	req := openai.ChatCompletionRequest{
		Model:       b.model,
		Messages:    convertToOpenAI(messages),
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		Tools:       convertTools(tools),
		Stream:      true,
	}

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("copilot: stream: %w", err)
	}

	chunks := make(chan types.Chunk)
	go func() {
		defer close(chunks)
		defer stream.Close()

		toolCalls := map[int]*types.ToolCall{}

		for {
			resp, recvErr := stream.Recv()
			if recvErr == io.EOF {
				finalCalls := flattenToolCalls(toolCalls)
				if len(finalCalls) > 0 {
					chunks <- types.Chunk{ToolCalls: finalCalls}
				}
				chunks <- types.Chunk{Done: true}
				return
			}
			if recvErr != nil {
				chunks <- types.Chunk{Done: true}
				return
			}

			for _, choice := range resp.Choices {
				delta := choice.Delta
				if delta.Content != "" {
					chunks <- types.Chunk{Content: delta.Content}
				}

				for _, call := range delta.ToolCalls {
					idx := readToolCallIndex(call)
					current, ok := toolCalls[idx]
					if !ok {
						current = &types.ToolCall{ID: call.ID, Name: call.Function.Name}
						toolCalls[idx] = current
					}

					if current.ID == "" {
						current.ID = call.ID
					}
					if call.Function.Name != "" {
						current.Name = call.Function.Name
					}
					if call.Function.Arguments != "" {
						current.Arguments = append(current.Arguments, json.RawMessage(call.Function.Arguments)...)
					}

					chunks <- types.Chunk{ToolCalls: []types.ToolCall{*current}}
				}
			}
		}
	}()

	return chunks, nil
}

// Info returns metadata about this backend.
func (b *CopilotBackend) Info() llm.BackendInfo {
	return llm.BackendInfo{
		Name:       "copilot",
		Models:     []string{b.model},
		AuthMethod: "oauth-device",
	}
}

// getAPIToken returns a valid Copilot API token, refreshing if needed.
func (b *CopilotBackend) getAPIToken(ctx context.Context) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if current token is still valid
	if b.apiToken != "" && time.Now().Before(b.apiTokenExpiry.Add(-tokenRefreshBuffer)) {
		return b.apiToken, nil
	}

	// Exchange OAuth token for Copilot API token
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.tokenEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Authorization", "token "+b.oauthToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", headerUserAgent)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp copilotTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Token == "" {
		return "", fmt.Errorf("empty token in response")
	}

	b.apiToken = tokenResp.Token
	if tokenResp.ExpiresAt > 0 {
		b.apiTokenExpiry = time.Unix(tokenResp.ExpiresAt, 0)
	} else {
		// Default to 30 minutes if no expiry provided
		b.apiTokenExpiry = time.Now().Add(30 * time.Minute)
	}

	return b.apiToken, nil
}

// copilotTransport is an http.RoundTripper that adds required Copilot headers.
type copilotTransport struct {
	apiToken string
}

func (t *copilotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiToken)
	req.Header.Set("User-Agent", headerUserAgent)
	req.Header.Set("Editor-Version", headerEditorVersion)
	req.Header.Set("Copilot-Integration-Id", headerIntegrationID)
	req.Header.Set("Editor-Plugin-Version", headerPluginVersion)
	req.Header.Set("Openai-Intent", headerOpenAIIntent)
	return http.DefaultTransport.RoundTrip(req)
}

// --- Helper functions (local copies for this package) ---

func convertToOpenAI(messages []types.LLMMessage) []openai.ChatCompletionMessage {
	converted := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role:    toOpenAIRole(msg.Role),
			Content: msg.Content,
			Name:    msg.Name,
		}

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				openaiMsg.ToolCalls = append(openaiMsg.ToolCalls, openai.ToolCall{
					ID:   call.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      call.Name,
						Arguments: string(call.Arguments),
					},
				})
			}
		}

		if msg.Role == "tool" {
			openaiMsg.ToolCallID = msg.ToolCallID
			openaiMsg.Content = msg.Content
		}

		converted = append(converted, openaiMsg)
	}
	return converted
}

func convertTools(tools []types.ToolDefinition) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}
	converted := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		var params any
		if len(tool.Parameters) > 0 {
			var decoded map[string]any
			if err := json.Unmarshal(tool.Parameters, &decoded); err == nil {
				params = decoded
			}
		}
		converted = append(converted, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		})
	}
	return converted
}

func convertToolCalls(calls []openai.ToolCall) []types.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	converted := make([]types.ToolCall, 0, len(calls))
	for _, call := range calls {
		converted = append(converted, types.ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: json.RawMessage(call.Function.Arguments),
		})
	}
	return converted
}

func convertUsage(usage openai.Usage) types.LLMUsage {
	return types.LLMUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func toOpenAIRole(role string) string {
	switch role {
	case "system":
		return openai.ChatMessageRoleSystem
	case "assistant":
		return openai.ChatMessageRoleAssistant
	case "tool":
		return openai.ChatMessageRoleTool
	case "user":
		fallthrough
	default:
		return openai.ChatMessageRoleUser
	}
}

func flattenToolCalls(source map[int]*types.ToolCall) []types.ToolCall {
	if len(source) == 0 {
		return nil
	}
	result := make([]types.ToolCall, 0, len(source))
	for _, call := range source {
		if call == nil {
			continue
		}
		result = append(result, *call)
	}
	return result
}

func readToolCallIndex(call openai.ToolCall) int {
	if call.Index != nil {
		return *call.Index
	}
	if call.ID != "" {
		if parsed, err := strconv.Atoi(call.ID); err == nil {
			return parsed
		}
	}
	return 0
}
