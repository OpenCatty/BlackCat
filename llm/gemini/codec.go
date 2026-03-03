// Package gemini provides a wire format codec for the Gemini API,
// shared by both the official Google Gemini provider and the Antigravity provider.
// It converts between BlackCat's types.LLMMessage format and Gemini's
// contents[].parts[] JSON wire format.
package gemini

import (
	"encoding/json"
	"fmt"

	"github.com/startower-observability/blackcat/types"
)

// --- Request types (BlackCat → Gemini wire format) ---

// GeminiRequest is the top-level request envelope for the Gemini API.
type GeminiRequest struct {
	Contents          []GeminiContent  `json:"contents"`
	SystemInstruction *GeminiContent   `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenConfig `json:"generationConfig,omitempty"`
	Tools             []GeminiTool     `json:"tools,omitempty"`
}

// GeminiContent represents a single content block with a role and parts.
type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart is a single part within a content block.
type GeminiPart struct {
	Text         string          `json:"text,omitempty"`
	FunctionCall *GeminiFuncCall `json:"functionCall,omitempty"`
	FunctionResp *GeminiFuncResp `json:"functionResponse,omitempty"`
}

// GeminiFuncCall represents a function/tool call in Gemini format.
type GeminiFuncCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

// GeminiFuncResp represents a function/tool response in Gemini format.
type GeminiFuncResp struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

// GeminiGenConfig holds generation parameters.
type GeminiGenConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
}

// GeminiTool describes a tool/function declaration for the Gemini API.
type GeminiTool struct {
	FunctionDeclarations []GeminiFuncDecl `json:"functionDeclarations,omitempty"`
}

// GeminiFuncDecl is a single function declaration.
type GeminiFuncDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// --- Response types (Gemini wire format → BlackCat) ---

// GeminiResponse is the top-level response from the Gemini API.
type GeminiResponse struct {
	Candidates    []GeminiCandidate `json:"candidates"`
	UsageMetadata *GeminiUsage      `json:"usageMetadata,omitempty"`
}

// GeminiCandidate is a single candidate in the response.
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

// GeminiUsage tracks token usage in Gemini format.
type GeminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// --- Codec functions ---

// EncodeMessages converts BlackCat LLM messages to a Gemini request.
// System messages are extracted into SystemInstruction.
// Role mapping: "assistant" → "model", "user" → "user".
// Tool result messages are converted to functionResponse parts.
func EncodeMessages(messages []types.LLMMessage, tools []types.ToolDefinition) *GeminiRequest {
	req := &GeminiRequest{}

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			// System messages become SystemInstruction.
			req.SystemInstruction = &GeminiContent{
				Parts: []GeminiPart{{Text: msg.Content}},
			}

		case "assistant":
			content := GeminiContent{Role: "model"}
			// Text content.
			if msg.Content != "" {
				content.Parts = append(content.Parts, GeminiPart{Text: msg.Content})
			}
			// Tool calls from assistant.
			for _, call := range msg.ToolCalls {
				content.Parts = append(content.Parts, GeminiPart{
					FunctionCall: &GeminiFuncCall{
						Name: call.Name,
						Args: call.Arguments,
					},
				})
			}
			if len(content.Parts) > 0 {
				req.Contents = append(req.Contents, content)
			}

		case "tool":
			// Tool result messages become functionResponse parts.
			resp := json.RawMessage(fmt.Sprintf(`{"result":%q}`, msg.Content))
			content := GeminiContent{
				Role: "function",
				Parts: []GeminiPart{
					{
						FunctionResp: &GeminiFuncResp{
							Name:     msg.Name,
							Response: resp,
						},
					},
				},
			}
			req.Contents = append(req.Contents, content)

		default: // "user" and anything else
			req.Contents = append(req.Contents, GeminiContent{
				Role:  "user",
				Parts: []GeminiPart{{Text: msg.Content}},
			})
		}
	}

	// Convert tools to Gemini function declarations.
	if len(tools) > 0 {
		decls := make([]GeminiFuncDecl, 0, len(tools))
		for _, tool := range tools {
			decls = append(decls, GeminiFuncDecl{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			})
		}
		req.Tools = []GeminiTool{{FunctionDeclarations: decls}}
	}

	return req
}

// DecodeResponse converts a Gemini response to BlackCat's LLMResponse.
// Role mapping: "model" → "assistant".
func DecodeResponse(resp *GeminiResponse, model string) (*types.LLMResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("nil Gemini response")
	}

	result := &types.LLMResponse{Model: model}

	// Extract usage metadata.
	if resp.UsageMetadata != nil {
		result.Usage = types.LLMUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	// Extract first candidate content.
	if len(resp.Candidates) == 0 {
		return result, nil
	}

	candidate := resp.Candidates[0]
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			result.Content += part.Text
		}
		if part.FunctionCall != nil {
			result.ToolCalls = append(result.ToolCalls, types.ToolCall{
				ID:        fmt.Sprintf("call_%s", part.FunctionCall.Name),
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}

	return result, nil
}
