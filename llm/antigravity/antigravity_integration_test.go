package antigravity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/startower-observability/blackcat/llm"
	"github.com/startower-observability/blackcat/llm/gemini"
	"github.com/startower-observability/blackcat/types"
)

// Integration tests for the full Antigravity flow:
// PKCE → token → chat with Gemini codec in envelope (all mocked)

func TestIntegrationAntigravityFullFlow(t *testing.T) {
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Verify auth
		auth := r.Header.Get("Authorization")
		if auth != "Bearer ya29.integration_token" {
			t.Errorf("unexpected auth: %s", auth)
		}

		// Verify path
		if r.URL.Path != GenerateContentPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected content type: %s", ct)
		}

		// Verify envelope structure
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var envelope AntigravityEnvelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}

		// Verify envelope fields
		if envelope.Model != "gemini-2.5-pro" {
			t.Errorf("unexpected model: %s", envelope.Model)
		}
		if envelope.Project != "test-project-123" {
			t.Errorf("unexpected project: %s", envelope.Project)
		}
		if envelope.UserAgent != "antigravity" {
			t.Errorf("unexpected userAgent: %s", envelope.UserAgent)
		}
		if envelope.RequestID == "" {
			t.Error("empty requestId")
		}
		if envelope.Request == nil {
			t.Fatal("nil inner request")
		}

		// Verify inner Gemini request
		if len(envelope.Request.Contents) == 0 {
			t.Fatal("empty contents in inner request")
		}
		if envelope.Request.Contents[0].Role != "user" {
			t.Errorf("unexpected role: %s", envelope.Request.Contents[0].Role)
		}
		if len(envelope.Request.Contents[0].Parts) == 0 || envelope.Request.Contents[0].Parts[0].Text != "hello from integration test" {
			t.Error("unexpected content in inner request")
		}

		// Return Gemini-format response
		resp := gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{
					Content: gemini.GeminiContent{
						Role:  "model",
						Parts: []gemini.GeminiPart{{Text: "antigravity integration response"}},
					},
				},
			},
			UsageMetadata: &gemini.GeminiUsage{
				PromptTokenCount:     12,
				CandidatesTokenCount: 6,
				TotalTokenCount:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.integration_token",
			BaseURL: server.URL,
			Model:   "gemini-2.5-pro",
		},
		AcceptedToS: true,
		Project:     "test-project-123",
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello from integration test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "antigravity integration response" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gemini-2.5-pro" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 18 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
}

func TestIntegrationAntigravityWithSystemPromptAndTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope AntigravityEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode: %v", err)
		}

		// Verify system instruction is set
		if envelope.Request.SystemInstruction == nil {
			t.Error("expected system instruction")
		}

		// Verify tools are passed through
		if len(envelope.Request.Tools) == 0 {
			t.Error("expected tools in request")
		}

		// Return response with tool call
		resp := gemini.GeminiResponse{
			Candidates: []gemini.GeminiCandidate{
				{
					Content: gemini.GeminiContent{
						Role: "model",
						Parts: []gemini.GeminiPart{
							{
								FunctionCall: &gemini.GeminiFuncCall{
									Name: "execute_command",
									Args: json.RawMessage(`{"command":"ls -la"}`),
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

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test_token",
			BaseURL: server.URL,
			Model:   "gemini-2.5-pro",
		},
		AcceptedToS: true,
		Project:     "test-project",
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You are a server admin."},
		{Role: "user", Content: "list files"},
	}, []types.ToolDefinition{
		{
			Name:        "execute_command",
			Description: "Execute a shell command",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"}}}`),
		},
	})
	if err != nil {
		t.Fatalf("chat with tools failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "execute_command" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
}

func TestIntegrationAntigravityTokenSource(t *testing.T) {
	// Test that TokenSource function is called for each request
	tokenCallCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ya29.dynamic_") {
			t.Errorf("unexpected auth: %s", auth)
		}

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
			BaseURL: server.URL,
			Model:   "gemini-2.5-pro",
			TokenSource: func() (string, error) {
				count := atomic.AddInt32(&tokenCallCount, 1)
				return "ya29.dynamic_" + string(rune('0'+count)), nil
			},
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	// Make two calls — each should invoke TokenSource
	for i := 0; i < 2; i++ {
		_, err = backend.Chat(context.Background(), []types.LLMMessage{
			{Role: "user", Content: "test"},
		}, nil)
		if err != nil {
			t.Fatalf("chat %d failed: %v", i, err)
		}
	}

	if count := atomic.LoadInt32(&tokenCallCount); count != 2 {
		t.Fatalf("expected 2 token source calls, got %d", count)
	}
}

func TestIntegrationAntigravityHTTP403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"Permission denied: Cloud Code API is not enabled","status":"PERMISSION_DENIED"}}`))
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.bad_token",
			BaseURL: server.URL,
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 in error message, got: %v", err)
	}
}

func TestIntegrationAntigravityHTTP500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`internal server error`))
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test",
			BaseURL: server.URL,
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 in error message, got: %v", err)
	}
}

func TestIntegrationAntigravityStreamFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming path
		if !strings.HasSuffix(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Error("expected alt=sse query parameter")
		}

		// Verify envelope
		var envelope AntigravityEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode envelope: %v", err)
		}
		if envelope.Model != "gemini-2.5-pro" {
			t.Errorf("unexpected model: %s", envelope.Model)
		}

		// Write SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []gemini.GeminiResponse{
			{Candidates: []gemini.GeminiCandidate{{Content: gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "Hello "}}}}}},
			{Candidates: []gemini.GeminiCandidate{{Content: gemini.GeminiContent{Role: "model", Parts: []gemini.GeminiPart{{Text: "world!"}}}}}},
		}

		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	backend, err := NewAntigravityBackend(AntigravityConfig{
		BackendConfig: llm.BackendConfig{
			APIKey:  "ya29.test_stream",
			BaseURL: server.URL,
			Model:   "gemini-2.5-pro",
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	chunks, err := backend.Stream(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	var content strings.Builder
	for chunk := range chunks {
		if chunk.Done {
			break
		}
		content.WriteString(chunk.Content)
	}

	if content.String() != "Hello world!" {
		t.Fatalf("unexpected streamed content: %s", content.String())
	}
}

func TestIntegrationAntigravityGenerationConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope AntigravityEnvelope
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode: %v", err)
		}

		// Verify generation config is passed through
		if envelope.Request.GenerationConfig == nil {
			t.Fatal("expected generation config")
		}
		if envelope.Request.GenerationConfig.Temperature == nil {
			t.Fatal("expected temperature in generation config")
		}
		if *envelope.Request.GenerationConfig.Temperature != 0.7 {
			t.Fatalf("unexpected temperature: %f", *envelope.Request.GenerationConfig.Temperature)
		}
		if envelope.Request.GenerationConfig.MaxOutputTokens == nil {
			t.Fatal("expected maxOutputTokens in generation config")
		}
		if *envelope.Request.GenerationConfig.MaxOutputTokens != 2048 {
			t.Fatalf("unexpected maxOutputTokens: %d", *envelope.Request.GenerationConfig.MaxOutputTokens)
		}

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
			APIKey:      "ya29.test",
			BaseURL:     server.URL,
			Model:       "gemini-2.5-pro",
			Temperature: 0.7,
			MaxTokens:   2048,
		},
		AcceptedToS: true,
	})
	if err != nil {
		t.Fatalf("NewAntigravityBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}
