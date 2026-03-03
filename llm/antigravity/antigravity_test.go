package antigravity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/startower-observability/blackcat/llm"
	"github.com/startower-observability/blackcat/llm/gemini"
	"github.com/startower-observability/blackcat/types"
)

func TestAntigravityChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer ya29.test-token" {
			t.Fatalf("unexpected auth: %s", auth)
		}

		// Verify path
		if r.URL.Path != "/v1internal:generateContent" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Verify request body is an Antigravity envelope
		var envelope AntigravityEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		if envelope.Model != "gemini-2.5-pro" {
			t.Fatalf("unexpected model: %s", envelope.Model)
		}
		if envelope.UserAgent != "antigravity" {
			t.Fatalf("unexpected userAgent: %s", envelope.UserAgent)
		}
		if envelope.RequestID == "" {
			t.Fatal("missing requestId")
		}
		if envelope.Project != "test-project" {
			t.Fatalf("unexpected project: %s", envelope.Project)
		}
		if envelope.Request == nil {
			t.Fatal("missing inner request")
		}

		// Verify inner Gemini request has contents
		if len(envelope.Request.Contents) == 0 {
			t.Fatal("empty contents in inner request")
		}

		// Return Gemini-format response
		resp := gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{
					Content: gemini.GeminiContent{
						Role:  "model",
						Parts: []gemini.GeminiPart{{Text: "hello from antigravity"}},
					},
				},
			},
			UsageMetadata: &gemini.GeminiUsage{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test-token",
			BaseURL: server.URL,
			Model:   "gemini-2.5-pro",
		},
		AcceptedToS: true,
		Project:     "test-project",
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	// Verify Backend interface
	var _ llm.Backend = backend

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "hello from antigravity" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gemini-2.5-pro" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("unexpected usage: %d", resp.Usage.TotalTokens)
	}
}

func TestAntigravityToS(t *testing.T) {
	_, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey: "ya29.test-token",
			Model:  "gemini-2.5-pro",
		},
		AcceptedToS: false, // Should fail
	})
	if err == nil {
		t.Fatal("expected error when ToS not accepted")
	}
	if !containsStr(err.Error(), "Terms of Service") {
		t.Fatalf("expected ToS error, got: %v", err)
	}
}

func TestAntigravityMissingToken(t *testing.T) {
	_, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			Model: "gemini-2.5-pro",
		},
		AcceptedToS: true,
	})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestAntigravityInfo(t *testing.T) {
	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey: "ya29.test",
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	info := backend.(*AntigravityBackend).Info()
	if info.Name != "antigravity" {
		t.Fatalf("expected name 'antigravity', got %q", info.Name)
	}
	if info.AuthMethod != "oauth-pkce" {
		t.Fatalf("expected auth 'oauth-pkce', got %q", info.AuthMethod)
	}
}

func TestAntigravityDefaultModel(t *testing.T) {
	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey: "ya29.test",
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	ab := backend.(*AntigravityBackend)
	if ab.model != "gemini-2.5-pro" {
		t.Fatalf("expected default model 'gemini-2.5-pro', got %s", ab.model)
	}
}

func TestAntigravityEnvelopeConstruction(t *testing.T) {
	// Test that the envelope is constructed correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope AntigravityEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}

		// Verify all fields
		if envelope.Project != "my-project" {
			t.Fatalf("unexpected project: %s", envelope.Project)
		}
		if envelope.Model != "claude-sonnet-4-6" {
			t.Fatalf("unexpected model: %s", envelope.Model)
		}
		if envelope.UserAgent != "antigravity" {
			t.Fatalf("unexpected userAgent: %s", envelope.UserAgent)
		}
		if envelope.RequestID == "" {
			t.Fatal("empty requestId")
		}
		if envelope.Request == nil {
			t.Fatal("nil inner request")
		}

		// Verify inner request has system instruction
		if envelope.Request.SystemInstruction == nil {
			t.Fatal("expected system instruction")
		}

		// Verify inner request has contents
		if len(envelope.Request.Contents) != 1 {
			t.Fatalf("expected 1 content, got %d", len(envelope.Request.Contents))
		}

		// Return minimal Gemini response
		resp := gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{Content: gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "ok"}}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test",
			BaseURL: server.URL,
			Model:   "claude-sonnet-4-6",
		},
		AcceptedToS: true,
		Project:     "my-project",
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}

func TestAntigravityHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"forbidden"}}`))
	}))
	defer server.Close()

	backend, _ := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test",
			BaseURL: server.URL,
		},
		AcceptedToS: true,
	})

	_, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
}

// containsStr is a helper to check substring presence.
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
