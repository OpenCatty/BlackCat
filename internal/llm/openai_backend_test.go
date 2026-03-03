package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

func TestOpenAIBackendChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-backend",
			"object":"chat.completion",
			"created":1710000000,
			"model":"gpt-4o-mini",
			"choices":[{"index":0,"message":{"role":"assistant","content":"backend hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`))
	}))
	defer server.Close()

	backend, err := NewOpenAIBackend(BackendConfig{
		APIKey:      "test-key",
		BaseURL:     server.URL + "/v1",
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
		MaxTokens:   128,
	})
	if err != nil {
		t.Fatalf("NewOpenAIBackend failed: %v", err)
	}

	// Verify Backend interface satisfaction at compile time
	var _ Backend = backend

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{{Role: "user", Content: "hello"}}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "backend hello" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage total: %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIBackendInfo(t *testing.T) {
	backend, err := NewOpenAIBackend(BackendConfig{
		APIKey: "test-key",
	Model:  "gpt-5.2",
	})
	if err != nil {
		t.Fatalf("NewOpenAIBackend failed: %v", err)
	}

	info := backend.(*OpenAIBackend).Info()
	if info.Name != "openai" {
		t.Fatalf("expected name 'openai', got %q", info.Name)
	}
	if info.AuthMethod != "api-key" {
		t.Fatalf("expected auth 'api-key', got %q", info.AuthMethod)
	}
	if len(info.Models) != 1 || info.Models[0] != "gpt-5.2" {
		t.Fatalf("unexpected models: %v", info.Models)
	}
}

func TestOpenAIBackendFactory(t *testing.T) {
	// Verify NewOpenAIBackend matches the BackendFactory signature
	var factory BackendFactory = func(cfg BackendConfig) (Backend, error) {
		return NewOpenAIBackend(cfg)
	}

	backend, err := factory(BackendConfig{
		APIKey: "factory-test",
		Model:  "test-model",
	})
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestOpenAIBackendTokenSource(t *testing.T) {
	backend, err := NewOpenAIBackend(BackendConfig{
		TokenSource: func() (string, error) { return "dynamic-token", nil },
		Model:       "test-model",
	})
	if err != nil {
		t.Fatalf("NewOpenAIBackend with TokenSource failed: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}
