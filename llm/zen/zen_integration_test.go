package zen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/startower-observability/blackcat/llm"
	"github.com/startower-observability/blackcat/types"
)

// Integration tests for the full Zen backend flow:
// API key → chat → model list (all mocked via httptest)

func TestIntegrationZenFullFlow(t *testing.T) {
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Verify the Authorization header has the API key
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer zen-test-key") {
			t.Errorf("unexpected auth: %s", auth)
		}

		// Verify the path is for chat completions
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody["model"] != "opencode/claude-opus-4-6" {
			t.Errorf("unexpected model: %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-zen-integ",
			"object":"chat.completion",
			"created":1710000000,
			"model":"opencode/claude-opus-4-6",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Zen integration response"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello from zen integration test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "Zen integration response" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "opencode/claude-opus-4-6" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
}

func TestIntegrationZenModelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"model":"opencode/claude-sonnet-4-6",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	zenBackend := backend.(*ZenBackend)

	// Verify curated model list
	models := zenBackend.ListModels()
	if len(models) != len(DefaultModels) {
		t.Fatalf("expected %d models, got %d", len(DefaultModels), len(models))
	}

	for i, expected := range DefaultModels {
		if models[i] != expected {
			t.Errorf("model[%d]: expected %s, got %s", i, expected, models[i])
		}
	}

	// Verify default model is the first curated model
	if zenBackend.model != DefaultModels[0] {
		t.Fatalf("expected default model %s, got %s", DefaultModels[0], zenBackend.model)
	}
}

func TestIntegrationZenWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		tools, ok := reqBody["tools"].([]interface{})
		if !ok || len(tools) == 0 {
			t.Error("expected tools in request")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-tools",
			"object":"chat.completion",
			"model":"opencode/claude-opus-4-6",
			"choices":[{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"",
					"tool_calls":[{
						"id":"call_zen_1",
						"type":"function",
						"function":{"name":"search_code","arguments":"{\"query\":\"func main\"}"}
					}]
				},
				"finish_reason":"tool_calls"
			}],
			"usage":{"prompt_tokens":30,"completion_tokens":15,"total_tokens":45}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "search for main function"},
	}, []types.ToolDefinition{
		{
			Name:        "search_code",
			Description: "Search code in the workspace",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		},
	})
	if err != nil {
		t.Fatalf("chat with tools failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "search_code" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].ID != "call_zen_1" {
		t.Fatalf("unexpected tool call ID: %s", resp.ToolCalls[0].ID)
	}
}

func TestIntegrationZenMultiTurnConversation(t *testing.T) {
	callCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		messages := reqBody["messages"].([]interface{})
		if count == 1 && len(messages) != 2 {
			t.Errorf("first call: expected 2 messages, got %d", len(messages))
		}
		if count == 2 && len(messages) != 4 {
			t.Errorf("second call: expected 4 messages, got %d", len(messages))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-multi",
			"object":"chat.completion",
			"model":"opencode/claude-opus-4-6",
			"choices":[{"index":0,"message":{"role":"assistant","content":"turn response"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	// Turn 1
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You help with code."},
		{Role: "user", Content: "what does main.go do?"},
	}, nil)
	if err != nil {
		t.Fatalf("turn 1 failed: %v", err)
	}

	// Turn 2 (with assistant response from turn 1)
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You help with code."},
		{Role: "user", Content: "what does main.go do?"},
		{Role: "assistant", Content: "turn response"},
		{Role: "user", Content: "can you refactor it?"},
	}, nil)
	if err != nil {
		t.Fatalf("turn 2 failed: %v", err)
	}

	if count := atomic.LoadInt32(&callCount); count != 2 {
		t.Fatalf("expected 2 calls, got %d", count)
	}
}

func TestIntegrationZenAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
}

func TestIntegrationZenEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-empty",
			"object":"chat.completion",
			"model":"opencode/claude-opus-4-6",
			"choices":[],
			"usage":{"prompt_tokens":5,"completion_tokens":0,"total_tokens":5}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	// Empty choices should return empty content, not error
	if resp.Content != "" {
		t.Fatalf("expected empty content for empty choices, got: %s", resp.Content)
	}
	if resp.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}
}

func TestIntegrationZenTokenSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"model":"opencode/claude-opus-4-6",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer server.Close()

	tokenCalls := int32(0)
	backend, err := NewZenBackend(llm.BackendConfig{
		BaseURL: server.URL,
		Model:   "opencode/claude-opus-4-6",
		TokenSource: func() (string, error) {
			atomic.AddInt32(&tokenCalls, 1)
			return "dynamic-zen-key", nil
		},
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	// TokenSource is called once during NewZenBackend
	if count := atomic.LoadInt32(&tokenCalls); count != 1 {
		t.Fatalf("expected 1 token source call, got %d", count)
	}
}
