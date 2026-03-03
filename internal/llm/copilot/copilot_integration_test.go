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

	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/types"
)

// Integration tests for the full Copilot flow:
// device code → token exchange → chat (all mocked via httptest)

func TestIntegrationCopilotFullFlow(t *testing.T) {
	// Phase 1: Mock device code + token poll server (simulates GitHub OAuth)
	deviceFlowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login/device/code":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code":      "dc_test_12345",
				"user_code":        "WDJB-MJHT",
				"verification_uri": "https://github.com/login/device",
				"expires_in":       900,
				"interval":         1,
			})
		case "/login/oauth/access_token":
			// Immediately grant the token (no pending loop in this integration test)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "gho_integration_test_token",
				"token_type":   "bearer",
				"scope":        "read:user",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer deviceFlowServer.Close()

	// Phase 2: Mock Copilot token exchange endpoint
	tokenExchangeCount := int32(0)
	copilotTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenExchangeCount, 1)

		auth := r.Header.Get("Authorization")
		if auth != "token gho_integration_test_token" {
			t.Errorf("unexpected auth on token exchange: %s", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     "tid=integ;exp=" + fmt.Sprintf("%d", time.Now().Add(30*time.Minute).Unix()),
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		})
	}))
	defer copilotTokenServer.Close()

	// Phase 3: Mock Copilot chat endpoint
	chatRequestCount := int32(0)
	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chatRequestCount, 1)

		// Verify all required Copilot headers
		if r.Header.Get("User-Agent") != headerUserAgent {
			t.Errorf("missing/wrong User-Agent: %s", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Editor-Version") != headerEditorVersion {
			t.Errorf("missing/wrong Editor-Version: %s", r.Header.Get("Editor-Version"))
		}
		if r.Header.Get("Copilot-Integration-Id") != headerIntegrationID {
			t.Errorf("missing/wrong Copilot-Integration-Id: %s", r.Header.Get("Copilot-Integration-Id"))
		}

		// Verify the request body is valid JSON with model field
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if reqBody["model"] != "gpt-4.1" {
			t.Errorf("unexpected model in request: %v", reqBody["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-integ",
			"object":"chat.completion",
			"created":1710000000,
			"model":"gpt-4.1",
			"choices":[{"index":0,"message":{"role":"assistant","content":"integration test response"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":15,"completion_tokens":8,"total_tokens":23}
		}`))
	}))
	defer chatServer.Close()

	// Create backend using the OAuth token obtained from "device flow"
	backend, err := NewCopilotBackend(llm.BackendConfig{
		APIKey:  "gho_integration_test_token",
		BaseURL: chatServer.URL + "/chat/completions",
	Model:   "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("NewCopilotBackend failed: %v", err)
	}

	cb := backend.(*CopilotBackend)
	cb.tokenEndpoint = copilotTokenServer.URL

	// Execute chat — this triggers token exchange → chat API call
	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You are a coding assistant."},
		{Role: "user", Content: "Write hello world"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	// Verify response
	if resp.Content != "integration test response" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gpt-4.1" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 23 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}

	// Verify token exchange happened exactly once
	if count := atomic.LoadInt32(&tokenExchangeCount); count != 1 {
		t.Fatalf("expected 1 token exchange, got %d", count)
	}
	if count := atomic.LoadInt32(&chatRequestCount); count != 1 {
		t.Fatalf("expected 1 chat request, got %d", count)
	}
}

func TestIntegrationCopilotTokenRefreshOnExpiry(t *testing.T) {
	tokenExchangeCount := int32(0)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&tokenExchangeCount, 1)

		w.Header().Set("Content-Type", "application/json")
		var expiresAt int64
		if count == 1 {
			// First token: already expired (should trigger refresh on next call)
			expiresAt = time.Now().Add(-5 * time.Minute).Unix()
		} else {
			// Second token: valid for 30 min
			expiresAt = time.Now().Add(30 * time.Minute).Unix()
		}

		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     fmt.Sprintf("api-token-%d", count),
			ExpiresAt: expiresAt,
		})
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-refresh",
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

	// First call: gets token-1 (expired)
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "first"},
	}, nil)
	if err != nil {
		t.Fatalf("first chat failed: %v", err)
	}

	// Second call: token-1 is expired, triggers refresh
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "second"},
	}, nil)
	if err != nil {
		t.Fatalf("second chat failed: %v", err)
	}

	// Third call: token-2 is still valid, no refresh needed
	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "third"},
	}, nil)
	if err != nil {
		t.Fatalf("third chat failed: %v", err)
	}

	// Should have exchanged token exactly 2 times (first + refresh)
	if count := atomic.LoadInt32(&tokenExchangeCount); count != 2 {
		t.Fatalf("expected 2 token exchanges, got %d", count)
	}
}

func TestIntegrationCopilotAPIError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     "valid-token",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		})
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal server error","type":"server_error"}}`))
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

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for API 500 response")
	}
}

func TestIntegrationCopilotWithToolCalls(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     "valid-token",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		})
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that tools were included in the request
		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		tools, ok := reqBody["tools"].([]interface{})
		if !ok || len(tools) == 0 {
			t.Error("expected tools in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-tools",
			"object":"chat.completion",
			"model":"gpt-4.1",
			"choices":[{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"",
					"tool_calls":[{
						"id":"call_1",
						"type":"function",
						"function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}
					}]
				},
				"finish_reason":"tool_calls"
			}],
			"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}
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

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "read main.go"},
	}, []types.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file from the filesystem",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		},
	})
	if err != nil {
		t.Fatalf("chat with tools failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("unexpected tool call name: %s", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("unexpected tool call ID: %s", resp.ToolCalls[0].ID)
	}
}

func TestIntegrationCopilotContextCancellation(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow token exchange
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(copilotTokenResponse{
			Token:     "token",
			ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		})
	}))
	defer tokenServer.Close()

	chatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("chat should not be called when context is cancelled")
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

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = backend.Chat(ctx, []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error due to context timeout")
	}
}
