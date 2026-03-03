package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/startower-observability/blackcat/internal/llm"
	"github.com/startower-observability/blackcat/internal/types"
)

func TestGeminiBackendChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		apiKey := r.Header.Get("x-goog-api-key")
		if apiKey != "test-gemini-key" {
			t.Fatalf("unexpected API key: %s", apiKey)
		}

		// Verify Content-Type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("unexpected content type: %s", ct)
		}

		// Verify path contains model
		expectedPath := "/models/gemini-2.5-flash:generateContent"
		if r.URL.Path != expectedPath {
			t.Fatalf("unexpected path: %s (expected %s)", r.URL.Path, expectedPath)
		}

		// Return Gemini response
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role:  "model",
						Parts: []GeminiPart{{Text: "hello from gemini"}},
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

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "test-gemini-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	// Verify Backend interface
	var _ llm.Backend = backend

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "hello from gemini" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %d", resp.Usage.TotalTokens)
	}
}

func TestGeminiBackendChatWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body contains tools
		var reqBody GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if len(reqBody.Tools) == 0 {
			t.Fatal("expected tools in request")
		}
		if len(reqBody.Tools[0].FunctionDeclarations) != 1 {
			t.Fatalf("expected 1 function decl, got %d", len(reqBody.Tools[0].FunctionDeclarations))
		}
		if reqBody.Tools[0].FunctionDeclarations[0].Name != "weather" {
			t.Fatalf("unexpected tool name: %s", reqBody.Tools[0].FunctionDeclarations[0].Name)
		}

		// Return tool call response
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role: "model",
						Parts: []GeminiPart{
							{
								FunctionCall: &GeminiFuncCall{
									Name: "weather",
									Args: json.RawMessage(`{"city":"Tokyo"}`),
								},
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	tools := []types.ToolDefinition{
		{
			Name:        "weather",
			Description: "Get weather",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
		},
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "weather in Tokyo?"},
	}, tools)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "weather" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
}

func TestGeminiBackendMissingAPIKey(t *testing.T) {
	_, err := NewGeminiBackend(llm.BackendConfig{
	Model: "gemini-2.5-flash",
	})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestGeminiBackendInfo(t *testing.T) {
	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey: "test-key",
		Model:  "gemini-2.0-pro",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	info := backend.(*GeminiBackend).Info()
	if info.Name != "gemini" {
		t.Fatalf("expected name 'gemini', got %q", info.Name)
	}
	if info.AuthMethod != "api-key" {
		t.Fatalf("expected auth 'api-key', got %q", info.AuthMethod)
	}
	if len(info.Models) != 1 || info.Models[0] != "gemini-2.0-pro" {
		t.Fatalf("unexpected models: %v", info.Models)
	}
}

func TestGeminiBackendDefaultModel(t *testing.T) {
	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	info := backend.(*GeminiBackend).Info()
	if info.Models[0] != "gemini-2.5-flash" {
		t.Fatalf("expected default model 'gemini-2.5-flash', got %s", info.Models[0])
	}
}

func TestGeminiBackendHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid API key"}}`))
	}))
	defer server.Close()

	backend, _ := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "bad-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})

	_, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
}
