package zen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/types"
)

func TestZenBackendChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer zen-test-key" {
			t.Fatalf("unexpected auth header: %s", auth)
		}

		// Verify path
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Verify model in request body
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if reqBody["model"] != "opencode/claude-opus-4-6" {
			t.Fatalf("unexpected model: %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-zen",
			"object":"chat.completion",
			"created":1710000000,
			"model":"opencode/claude-opus-4-6",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello from zen"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`))
	}))
	defer server.Close()

	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey:  "zen-test-key",
		BaseURL: server.URL + "/v1",
		Model:   "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	// Verify Backend interface
	var _ llm.Backend = backend

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "hello from zen" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "opencode/claude-opus-4-6" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %d", resp.Usage.TotalTokens)
	}
}

func TestZenBackendMissingAPIKey(t *testing.T) {
	_, err := NewZenBackend(llm.BackendConfig{
		Model: "opencode/claude-opus-4-6",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestZenBackendDefaultModel(t *testing.T) {
	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	zen := backend.(*ZenBackend)
	if zen.model != DefaultModels[0] {
		t.Fatalf("expected default model %s, got %s", DefaultModels[0], zen.model)
	}
}

func TestZenBackendInfo(t *testing.T) {
	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	info := backend.(*ZenBackend).Info()
	if info.Name != "zen" {
		t.Fatalf("expected name 'zen', got %q", info.Name)
	}
	if info.AuthMethod != "api-key" {
		t.Fatalf("expected auth 'api-key', got %q", info.AuthMethod)
	}
	if len(info.Models) != len(DefaultModels) {
		t.Fatalf("expected %d models, got %d", len(DefaultModels), len(info.Models))
	}
}

func TestZenModelList(t *testing.T) {
	backend, err := NewZenBackend(llm.BackendConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewZenBackend failed: %v", err)
	}

	models := backend.(*ZenBackend).ListModels()
	if len(models) != len(DefaultModels) {
		t.Fatalf("expected %d models, got %d", len(DefaultModels), len(models))
	}

	// Verify all defaults present with opencode/ prefix
	for i, m := range models {
		if m != DefaultModels[i] {
			t.Fatalf("model[%d] = %s, expected %s", i, m, DefaultModels[i])
		}
	}

	// Verify mutation safety — modifying returned slice shouldn't affect backend
	models[0] = "modified"
	original := backend.(*ZenBackend).ListModels()
	if original[0] == "modified" {
		t.Fatal("ListModels should return a copy, not the internal slice")
	}
}

func TestZenBackendTokenSource(t *testing.T) {
	backend, err := NewZenBackend(llm.BackendConfig{
		TokenSource: func() (string, error) { return "dynamic-zen-key", nil },
		Model:       "opencode/claude-opus-4-6",
	})
	if err != nil {
		t.Fatalf("NewZenBackend with TokenSource failed: %v", err)
	}
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}
