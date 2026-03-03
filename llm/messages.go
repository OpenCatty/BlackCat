package llm

import (
	"encoding/json"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/types"
)

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
