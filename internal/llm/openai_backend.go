package llm

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/internal/types"
)

// OpenAIBackend implements the Backend interface using the go-openai library.
// It is the extracted core from the original Client, supporting both static
// API key auth and dynamic token sources (for providers like Copilot/Zen).
type OpenAIBackend struct {
	oaiClient   *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

// NewOpenAIBackend creates a new OpenAI-compatible Backend from BackendConfig.
// This is the BackendFactory for OpenAI-compatible providers.
func NewOpenAIBackend(cfg BackendConfig) (Backend, error) {
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.TokenSource != nil {
		token, err := cfg.TokenSource()
		if err != nil {
			return nil, err
		}
		apiKey = token
	}

	config := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	return &OpenAIBackend{
		oaiClient:   openai.NewClientWithConfig(config),
		model:       cfg.Model,
		temperature: float32(cfg.Temperature),
		maxTokens:   cfg.MaxTokens,
	}, nil
}

// Chat sends a non-streaming chat completion request.
func (b *OpenAIBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:       b.model,
		Messages:    convertToOpenAI(messages),
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		Tools:       convertTools(tools),
	}

	resp, err := b.oaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
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

// Stream sends a streaming chat completion request and returns a channel of chunks.
func (b *OpenAIBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	req := openai.ChatCompletionRequest{
		Model:       b.model,
		Messages:    convertToOpenAI(messages),
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		Tools:       convertTools(tools),
		Stream:      true,
	}

	stream, err := b.oaiClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}

	chunks := make(chan types.Chunk)
	go func() {
		defer close(chunks)
		defer stream.Close()

		toolCalls := map[int]*types.ToolCall{}

		for {
			resp, recvErr := stream.Recv()
			if recvErr == io.EOF {
				finalCalls := backendFlattenToolCalls(toolCalls)
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
					idx := backendReadToolCallIndex(call)
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
func (b *OpenAIBackend) Info() BackendInfo {
	return BackendInfo{
		Name:       "openai",
		Models:     []string{b.model},
		AuthMethod: "api-key",
	}
}

func backendFlattenToolCalls(source map[int]*types.ToolCall) []types.ToolCall {
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

func backendReadToolCallIndex(call openai.ToolCall) int {
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
