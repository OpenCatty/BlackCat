package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/llm"
	"github.com/startower-observability/blackcat/types"
)

func TestCopilotBackendChat(t *testing.T) {
	// Mock Copilot token exchange endpoint
	tokenExchangeCount := int32(0)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenExchangeCount, 1)

		// Verify OAuth token in header
		auth := r.Header.Get("Authorization")
		if auth != "token gho_test_oauth" {
			t.Fatalf("unexpected auth header on token exchange: %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     "tid=test;exp=" + fmt.Sprintf("%d", time.Now().Add(30*time.Minute).Unix()) + ";sku=free;st=dotcom",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		})
	}))
	defer tokenServer.Close()

	// Mock Copilot chat endpoint
	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required Copilot headers
		if r.Header.Get("User-Agent") != headerUserAgent {
			t.Fatalf("missing/wrong User-Agent: %s", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Editor-Version") != headerEditorVersion {
			t.Fatalf("missing/wrong Editor-Version: %s", r.Header.Get("Editor-Version"))
		}
		if r.Header.Get("Copilot-Integration-Id") != headerIntegrationID {
			t.Fatalf("missing/wrong Copilot-Integration-Id: %s", r.Header.Get("Copilot-Integration-Id"))
		}
		if r.Header.Get("Openai-Intent") != headerOpenAIIntent {
			t.Fatalf("missing/wrong Openai-Intent: %s", r.Header.Get("Openai-Intent"))
		}

		// Verify Authorization uses Bearer token (Copilot API token, not OAuth)
		auth := r.Header.Get("Authorization")
		if auth == "" || auth == "Bearer gho_test_oauth" {
			t.Fatalf("expected Copilot API token in Bearer auth, got: %s", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-copilot",
			"object":"chat.completion",
			"created":1710000000,
			"model":"gpt-4.1",
			"choices":[{"index":0,"message":{"role":"assistant","content":"hello from copilot"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
		}`))
	}))
	defer chatServer.Close()

	// Create backend with mock endpoints
	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey:  "gho_test_oauth",
		BaseURL: chatServer.URL + "/chat/completions",
	Model:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	// Override token endpoint to mock server
	cb := backend.(*CopilotBackend)
	cb.tokenEndpoint = tokenServer.URL

	// Verify Backend interface
	var _ llm.Backend = backend

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "hello from copilot" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gpt-4.1" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %d", resp.Usage.TotalTokens)
	}

	// Token should have been exchanged exactly once
	if count := atomic.LoadInt32(&tokenExchangeCount); count != 1 {
		t.Fatalf("expected 1 token exchange, got %d", count)
	}
}

func TestCopilotTokenRefresh(t *testing.T) {
	tokenExchangeCount := int32(0)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&tokenExchangeCount, 1)

		w.Header().Set("Content-Type", "application/json")
		// First token: already expired
		// Second token: valid for 30 min
		var expiresAt int64
		if count == 1 {
			expiresAt = time.Now().Add(-1 * time.Minute).Unix() // Already expired
		} else {
			expiresAt = time.Now().Add(30 * time.Minute).Unix()
		}

		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     fmt.Sprintf("token-%d", count),
			ExpiresAt: expiresAt,
		})
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-test",
			"object":"chat.completion",
			"model":"gpt-4.1",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer chatServer.Close()

	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey:  "gho_test",
		BaseURL: chatServer.URL + "/chat/completions",
	Model:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	cb := backend.(*CopilotBackend)
	cb.tokenEndpoint = tokenServer.URL

	// First call: gets token-1 (expired), should use it but it's returned anyway
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("first chat failed: %v", err)
	}

	// Second call: token-1 is expired, should refresh to token-2
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test again"},
	}, nil)
	if err != nil {
		t.Fatalf("second chat failed: %v", err)
	}

	// Should have exchanged token twice (first call + refresh on second call)
	if count := atomic.LoadInt32(&tokenExchangeCount); count != 2 {
		t.Fatalf("expected 2 token exchanges, got %d", count)
	}
}

func TestCopilotBackendMissingToken(t *testing.T) {
	_, err := NewCopilotBackend(llm.BackendConfig{
	Model: "gpt-4.1",
	})
	if err == nil {
		t.Fatal("expected error for missing OAuth token")
	}
}

func TestCopilotBackendInfo(t *testing.T) {
	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey: "gho_test",
	Model:  "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	info := backend.(*CopilotBackend).Info()
	if info.Name != "copilot" {
		t.Fatalf("expected name 'copilot', got %q", info.Name)
	}
	if info.AuthMethod != "oauth-device" {
		t.Fatalf("expected auth 'oauth-device', got %q", info.AuthMethod)
	}
	if len(info.Models) != 1 || info.Models[0] != "gpt-4.1" {
		t.Fatalf("unexpected models: %v", info.Models)
	}
}

func TestCopilotDefaultModel(t *testing.T) {
	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey: "gho_test",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	cb := backend.(*CopilotBackend)
	if cb.model != "gpt-4.1" {
		t.Fatalf("expected default model gpt-4.1, got %s", cb.model)
	}
}

func TestCopilotTokenExchangeError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("chat should not be called when token exchange fails")
	}))
	defer chatServer.Close()

	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey:  "bad_token",
		BaseURL: chatServer.URL + "/chat/completions",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	cb := backend.(*CopilotBackend)
	cb.tokenEndpoint = tokenServer.URL

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for bad OAuth token")
	}
}
