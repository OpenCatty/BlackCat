// Package zen provides the Zen Coding Plan LLM backend for BlackCat.
// Zen is an OpenAI-compatible provider with a curated model list and
// hosted API endpoint.
package zen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	// DefaultZenBaseURL is the default Zen API endpoint.
	DefaultZenBaseURL = "https://api.opencode.ai/v1"
)

// DefaultModels is the curated list of models available through Zen Coding Plan.
var DefaultModels = []string{
	"opencode/claude-opus-4-6",
	"opencode/claude-sonnet-4-6",
	"opencode/gemini-3.1-pro",
}

// ZenBackend implements llm.Backend using the Zen Coding Plan API.
// It is OpenAI-compatible and uses go-openai for HTTP transport.
type ZenBackend struct {
	oaiClient   *openai.Client
	model       string
	temperature float32
	maxTokens   int
	models      []string // Curated model list
}

// NewZenBackend creates a new Zen backend from BackendConfig.
func NewZenBackend(cfg llm.BackendConfig) (llm.Backend, error) {
	apiKey := cfg.APIKey
	if apiKey == "" && cfg.TokenSource != nil {
		token, err := cfg.TokenSource()
		if err != nil {
			return nil, fmt.Errorf("zen: get API key: %w", err)
		}
		apiKey = token
	}
	if apiKey == "" {
		return nil, fmt.Errorf("zen: API key is required (set ZEN_API_KEY or config.zen.apiKey)")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultZenBaseURL
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModels[0] // Default to first curated model
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &ZenBackend{
		oaiClient:   openai.NewClientWithConfig(config),
		model:       model,
		temperature: float32(cfg.Temperature),
		maxTokens:   cfg.MaxTokens,
		models:      DefaultModels,
	}, nil
}

// Chat sends a non-streaming chat completion request via the Zen API.
func (b *ZenBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:       b.model,
		Messages:    convertToOpenAI(messages),
		Temperature: b.temperature,
		MaxTokens:   b.maxTokens,
		Tools:       convertTools(tools),
	}

	resp, err := b.oaiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("zen: chat: %w", err)
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

// Stream sends a streaming chat completion request via the Zen API.
func (b *ZenBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
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
		return nil, fmt.Errorf("zen: stream: %w", err)
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
func (b *ZenBackend) Info() llm.BackendInfo {
	return llm.BackendInfo{
		Name:       "zen",
		Models:     b.models,
		AuthMethod: "api-key",
	}
}

// ListModels returns the curated list of models available through Zen.
func (b *ZenBackend) ListModels() []string {
	result := make([]string, len(b.models))
	copy(result, b.models)
	return result
}

// --- Helper functions (local copies to avoid cross-package dependency) ---

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
