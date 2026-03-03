package llm

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/internal/types"
)

// Client is the original LLM client that satisfies types.LLMClient.
// It delegates to OpenAIBackend internally for backward compatibility.
type Client struct {
	backend *OpenAIBackend
}

// NewClient creates a new Client (backward-compatible constructor).
// Internally delegates to OpenAIBackend.
func NewClient(apiKey, baseURL, model string, temperature float64, maxTokens int) *Client {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	return &Client{
		backend: &OpenAIBackend{
			oaiClient:   openai.NewClientWithConfig(config),
			model:       model,
			temperature: float32(temperature),
			maxTokens:   maxTokens,
		},
	}
}

// Chat delegates to the underlying OpenAIBackend.
func (c *Client) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	return c.backend.Chat(ctx, messages, tools)
}

// Stream delegates to the underlying OpenAIBackend.
func (c *Client) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	return c.backend.Stream(ctx, messages, tools)
}
