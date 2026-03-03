package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/startower-observability/blackcat/internal/types"
)

func TestProviderDetect(t *testing.T) {
	registry := NewRegistry()
	provider, key := registry.Detect(func(name string) string {
		if name == "OPENAI_API_KEY" {
			return "sk-test-openai"
		}
		return ""
	})

	if provider == nil {
		t.Fatal("expected provider to be detected")
	}
	if provider.Name != "openai" {
		t.Fatalf("expected openai, got %s", provider.Name)
	}
	if key != "sk-test-openai" {
		t.Fatalf("expected key sk-test-openai, got %s", key)
	}
}

func TestProviderDetectAnthropic(t *testing.T) {
	registry := NewRegistry()
	provider, key := registry.Detect(func(name string) string {
		if name == "ANTHROPIC_API_KEY" {
			return "sk-ant-test"
		}
		return ""
	})

	if provider == nil {
		t.Fatal("expected provider to be detected")
	}
	if provider.Name != "anthropic" {
		t.Fatalf("expected anthropic, got %s", provider.Name)
	}
	if key != "sk-ant-test" {
		t.Fatalf("expected key sk-ant-test, got %s", key)
	}
}

func TestProviderDetectNone(t *testing.T) {
	registry := NewRegistry()
	provider, key := registry.Detect(func(string) string { return "" })

	if provider != nil {
		t.Fatalf("expected nil provider, got %s", provider.Name)
	}
	if key != "" {
		t.Fatalf("expected empty key, got %s", key)
	}
}

func TestProviderGet(t *testing.T) {
	registry := NewRegistry()
	provider, ok := registry.Get("openai")

	if !ok {
		t.Fatal("expected provider to be found")
	}
	if provider == nil || provider.Name != "openai" {
		t.Fatalf("expected openai provider, got %#v", provider)
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []types.LLMMessage{
		{Role: "user", Content: "hello"},
		{
			Role:    "assistant",
			Content: "using tool",
			ToolCalls: []types.ToolCall{
				{ID: "call_1", Name: "weather", Arguments: json.RawMessage(`{"city":"London"}`)},
			},
		},
		{Role: "tool", ToolCallID: "call_1", Content: "{\"temp\":22}"},
	}

	converted := convertToOpenAI(messages)
	if len(converted) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(converted))
	}

	if converted[0].Role != openai.ChatMessageRoleUser || converted[0].Content != "hello" {
		t.Fatalf("unexpected user message: %#v", converted[0])
	}

	if converted[1].Role != openai.ChatMessageRoleAssistant {
		t.Fatalf("unexpected assistant role: %s", converted[1].Role)
	}
	if len(converted[1].ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(converted[1].ToolCalls))
	}
	if converted[1].ToolCalls[0].Function.Name != "weather" {
		t.Fatalf("expected weather tool name, got %s", converted[1].ToolCalls[0].Function.Name)
	}

	if converted[2].Role != openai.ChatMessageRoleTool {
		t.Fatalf("unexpected tool role: %s", converted[2].Role)
	}
	if converted[2].ToolCallID != "call_1" {
		t.Fatalf("expected tool call id call_1, got %s", converted[2].ToolCallID)
	}
}

func TestConvertTools(t *testing.T) {
	defs := []types.ToolDefinition{
		{
			Name:        "weather",
			Description: "Get weather",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
		},
	}

	tools := convertTools(defs)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Type != openai.ToolTypeFunction {
		t.Fatalf("expected function tool type, got %s", tools[0].Type)
	}
	if tools[0].Function == nil {
		t.Fatal("expected function definition")
	}

	params, ok := tools[0].Function.Parameters.(map[string]any)
	if !ok {
		t.Fatalf("expected parameters map, got %T", tools[0].Function.Parameters)
	}
	if params["type"] != "object" {
		t.Fatalf("expected schema type object, got %v", params["type"])
	}
}

func TestConvertToolCalls(t *testing.T) {
	calls := []openai.ToolCall{
		{
			ID:   "call_1",
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      "weather",
				Arguments: `{"city":"Berlin"}`,
			},
		},
	}

	converted := convertToolCalls(calls)
	if len(converted) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(converted))
	}
	if converted[0].Name != "weather" {
		t.Fatalf("expected weather tool name, got %s", converted[0].Name)
	}
	if string(converted[0].Arguments) != `{"city":"Berlin"}` {
		t.Fatalf("unexpected arguments: %s", string(converted[0].Arguments))
	}
}

func TestClientChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"created":1710000000,
			"model":"gpt-4o-mini",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello from mock"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/v1", "gpt-4o-mini", 0.1, 128)
	resp, err := client.Chat(context.Background(), []types.LLMMessage{{Role: "user", Content: "hello"}}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "hello from mock" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage total: %d", resp.Usage.TotalTokens)
	}
}

func TestClientChatWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test-tool",
			"object":"chat.completion",
			"created":1710000001,
			"model":"gpt-4o-mini",
			"choices":[{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"",
					"tool_calls":[{"id":"call_abc","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Tokyo\"}"}}]
				},
				"finish_reason":"tool_calls"
			}],
			"usage":{"prompt_tokens":12,"completion_tokens":7,"total_tokens":19}
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL+"/v1", "gpt-4o-mini", 0.1, 128)
	resp, err := client.Chat(context.Background(), []types.LLMMessage{{Role: "user", Content: "weather?"}}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ID != "call_abc" {
		t.Fatalf("unexpected tool call id: %s", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "weather" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
	if string(resp.ToolCalls[0].Arguments) != `{"city":"Tokyo"}` {
		t.Fatalf("unexpected tool args: %s", string(resp.ToolCalls[0].Arguments))
	}
}
