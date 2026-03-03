package gemini

import (
	"encoding/json"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

func TestEncodeMessages(t *testing.T) {
	messages := []types.LLMMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello, world!"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	req := EncodeMessages(messages, nil)

	// System message should be in SystemInstruction.
	if req.SystemInstruction == nil {
		t.Fatal("expected SystemInstruction to be set")
	}
	if len(req.SystemInstruction.Parts) != 1 || req.SystemInstruction.Parts[0].Text != "You are a helpful assistant." {
		t.Errorf("unexpected SystemInstruction: %+v", req.SystemInstruction)
	}

	// Contents should have 3 entries (user, model, user) — system is extracted.
	if len(req.Contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(req.Contents))
	}

	// First content: user.
	if req.Contents[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", req.Contents[0].Role)
	}
	if req.Contents[0].Parts[0].Text != "Hello, world!" {
		t.Errorf("unexpected content: %q", req.Contents[0].Parts[0].Text)
	}

	// Second content: model (mapped from assistant).
	if req.Contents[1].Role != "model" {
		t.Errorf("expected role 'model', got %q", req.Contents[1].Role)
	}
	if req.Contents[1].Parts[0].Text != "Hi there!" {
		t.Errorf("unexpected content: %q", req.Contents[1].Parts[0].Text)
	}

	// Third content: user.
	if req.Contents[2].Role != "user" {
		t.Errorf("expected role 'user', got %q", req.Contents[2].Role)
	}
}

func TestRoleMapping(t *testing.T) {
	// Encode: assistant → model.
	messages := []types.LLMMessage{
		{Role: "assistant", Content: "I am the assistant."},
	}
	req := EncodeMessages(messages, nil)
	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(req.Contents))
	}
	if req.Contents[0].Role != "model" {
		t.Errorf("encode: expected 'model', got %q", req.Contents[0].Role)
	}

	// Decode: model → assistant (implicit — response content comes from model role).
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role:  "model",
					Parts: []GeminiPart{{Text: "Hello from model"}},
				},
			},
		},
	}
	result, err := DecodeResponse(resp, "gemini-2.5-pro")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Content != "Hello from model" {
		t.Errorf("expected 'Hello from model', got %q", result.Content)
	}
}

func TestDecodeResponse(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Here is the answer."},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &GeminiUsage{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
			TotalTokenCount:      15,
		},
	}

	result, err := DecodeResponse(resp, "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if result.Content != "Here is the answer." {
		t.Errorf("content: got %q, want %q", result.Content, "Here is the answer.")
	}
	if result.Model != "gemini-2.5-flash" {
		t.Errorf("model: got %q, want %q", result.Model, "gemini-2.5-flash")
	}
	if result.Usage.PromptTokens != 10 {
		t.Errorf("prompt tokens: got %d, want 10", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 5 {
		t.Errorf("completion tokens: got %d, want 5", result.Usage.CompletionTokens)
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("total tokens: got %d, want 15", result.Usage.TotalTokens)
	}
}

func TestDecodeResponseEmpty(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{},
	}
	result, err := DecodeResponse(resp, "gemini-2.5-pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "" {
		t.Errorf("expected empty content, got %q", result.Content)
	}
}

func TestDecodeResponseNil(t *testing.T) {
	_, err := DecodeResponse(nil, "gemini-2.5-pro")
	if err == nil {
		t.Fatal("expected error for nil response")
	}
}

func TestEncodeMessagesWithToolCalls(t *testing.T) {
	args := json.RawMessage(`{"query":"test"}`)
	messages := []types.LLMMessage{
		{Role: "user", Content: "Search for test"},
		{
			Role: "assistant",
			ToolCalls: []types.ToolCall{
				{ID: "call_1", Name: "search", Arguments: args},
			},
		},
		{
			Role:       "tool",
			Name:       "search",
			Content:    "Found 3 results",
			ToolCallID: "call_1",
		},
	}

	req := EncodeMessages(messages, nil)

	if len(req.Contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(req.Contents))
	}

	// Assistant message should have function call.
	modelMsg := req.Contents[1]
	if modelMsg.Role != "model" {
		t.Errorf("expected 'model', got %q", modelMsg.Role)
	}
	if len(modelMsg.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(modelMsg.Parts))
	}
	if modelMsg.Parts[0].FunctionCall == nil {
		t.Fatal("expected function call in part")
	}
	if modelMsg.Parts[0].FunctionCall.Name != "search" {
		t.Errorf("function name: got %q, want 'search'", modelMsg.Parts[0].FunctionCall.Name)
	}

	// Tool result should be function response.
	toolMsg := req.Contents[2]
	if toolMsg.Role != "function" {
		t.Errorf("expected role 'function', got %q", toolMsg.Role)
	}
	if toolMsg.Parts[0].FunctionResp == nil {
		t.Fatal("expected function response in part")
	}
	if toolMsg.Parts[0].FunctionResp.Name != "search" {
		t.Errorf("function resp name: got %q, want 'search'", toolMsg.Parts[0].FunctionResp.Name)
	}
}

func TestDecodeResponseWithFunctionCall(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{
							FunctionCall: &GeminiFuncCall{
								Name: "search",
								Args: json.RawMessage(`{"query":"hello"}`),
							},
						},
					},
				},
			},
		},
	}

	result, err := DecodeResponse(resp, "gemini-2.5-pro")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Name != "search" {
		t.Errorf("tool call name: got %q, want 'search'", result.ToolCalls[0].Name)
	}
	if string(result.ToolCalls[0].Arguments) != `{"query":"hello"}` {
		t.Errorf("tool call args: got %s, want %s", result.ToolCalls[0].Arguments, `{"query":"hello"}`)
	}
}

func TestEncodeMessagesWithTools(t *testing.T) {
	tools := []types.ToolDefinition{
		{
			Name:        "search",
			Description: "Search the web",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
		},
	}

	req := EncodeMessages(nil, tools)

	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool group, got %d", len(req.Tools))
	}
	if len(req.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("expected 1 function declaration, got %d", len(req.Tools[0].FunctionDeclarations))
	}
	decl := req.Tools[0].FunctionDeclarations[0]
	if decl.Name != "search" {
		t.Errorf("name: got %q, want 'search'", decl.Name)
	}
	if decl.Description != "Search the web" {
		t.Errorf("description: got %q, want 'Search the web'", decl.Description)
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	// Encode messages.
	messages := []types.LLMMessage{
		{Role: "system", Content: "Be brief."},
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
	}

	req := EncodeMessages(messages, nil)

	// Verify structure is valid JSON.
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal back.
	var decoded GeminiRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.SystemInstruction == nil {
		t.Fatal("expected system instruction after roundtrip")
	}
	if len(decoded.Contents) != 2 {
		t.Errorf("expected 2 contents after roundtrip, got %d", len(decoded.Contents))
	}
}
