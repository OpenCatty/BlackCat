package gemini

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

// Integration tests for the full Gemini backend flow:
// API key → chat → Gemini codec decode (all mocked)

func TestIntegrationGeminiFullFlow(t *testing.T) {
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Verify API key header
		apiKey := r.Header.Get("x-goog-api-key")
		if apiKey != "test-gemini-api-key" {
			t.Errorf("unexpected API key: %s", apiKey)
		}

		// Verify path includes model name
		if !strings.Contains(r.URL.Path, "/models/gemini-2.5-flash:generateContent") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify request is valid Gemini format
		var geminiReq GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&geminiReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(geminiReq.Contents) == 0 {
			t.Fatal("empty contents")
		}
		if geminiReq.Contents[0].Role != "user" {
			t.Errorf("unexpected role: %s", geminiReq.Contents[0].Role)
		}

		// Return Gemini-format response
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role:  "model",
						Parts: []GeminiPart{{Text: "Gemini integration response"}},
					},
				},
			},
			UsageMetadata: &GeminiUsage{
				PromptTokenCount:     8,
				CandidatesTokenCount: 4,
				TotalTokenCount:      12,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "test-gemini-api-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello gemini"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "Gemini integration response" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Fatalf("unexpected model: %s", resp.Model)
	}
	if resp.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected total tokens: %d", resp.Usage.TotalTokens)
	}

	if count := atomic.LoadInt32(&requestCount); count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
}

func TestIntegrationGeminiSystemInstruction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var geminiReq GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&geminiReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		// Verify system instruction is set
		if geminiReq.SystemInstruction == nil {
			t.Fatal("expected system instruction")
		}
		if len(geminiReq.SystemInstruction.Parts) == 0 {
			t.Fatal("expected parts in system instruction")
		}
		if geminiReq.SystemInstruction.Parts[0].Text != "You are a coding expert." {
			t.Errorf("unexpected system instruction: %s", geminiReq.SystemInstruction.Parts[0].Text)
		}

		// Verify user message is separate from system
		if len(geminiReq.Contents) != 1 {
			t.Fatalf("expected 1 content (user msg), got %d", len(geminiReq.Contents))
		}

		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "system acknowledged"}}}},
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

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "system", Content: "You are a coding expert."},
		{Role: "user", Content: "write hello world"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if resp.Content != "system acknowledged" {
		t.Fatalf("unexpected content: %s", resp.Content)
	}
}

func TestIntegrationGeminiWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var geminiReq GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&geminiReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		// Verify tools are present
		if len(geminiReq.Tools) == 0 {
			t.Fatal("expected tools in request")
		}

		// Return function call response
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					Content: GeminiContent{
						Role: "model",
						Parts: []GeminiPart{
							{
								FunctionCall: &GeminiFuncCall{
									Name: "read_file",
									Args: json.RawMessage(`{"path":"/etc/hosts"}`),
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

	resp, err := backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "read hosts file"},
	}, []types.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`),
		},
	})
	if err != nil {
		t.Fatalf("chat with tools failed: %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
}

func TestIntegrationGeminiStreamFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify streaming path
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Error("expected alt=sse")
		}

		// Verify API key
		if r.Header.Get("x-goog-api-key") != "test-key" {
			t.Errorf("unexpected API key")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []GeminiResponse{
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "streaming "}}}}}},
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "gemini "}}}}}},
			{Candidates: []GeminiCandidate{{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "response"}}}}}},
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

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	ch, err := backend.Stream(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	var content strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		content.WriteString(chunk.Content)
	}

	if content.String() != "streaming gemini response" {
		t.Fatalf("unexpected streamed content: %q", content.String())
	}
}

func TestIntegrationGeminiHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":401,"message":"API key not valid","status":"UNAUTHENTICATED"}}`))
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "hello"},
	}, nil)
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 in error: %v", err)
	}
}

func TestIntegrationGeminiGenerationConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var geminiReq GeminiRequest
		if err := json.NewDecoder(r.Body).Decode(&geminiReq); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if geminiReq.GenerationConfig == nil {
			t.Fatal("expected generation config")
		}
		if geminiReq.GenerationConfig.Temperature == nil || *geminiReq.GenerationConfig.Temperature != 0.5 {
			t.Fatalf("unexpected temperature: %v", geminiReq.GenerationConfig.Temperature)
		}
		if geminiReq.GenerationConfig.MaxOutputTokens == nil || *geminiReq.GenerationConfig.MaxOutputTokens != 4096 {
			t.Fatalf("unexpected maxOutputTokens: %v", geminiReq.GenerationConfig.MaxOutputTokens)
		}

		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "ok"}}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		APIKey:      "test-key",
		BaseURL:     server.URL,
	Model:       "gemini-2.5-flash",
		Temperature: 0.5,
		MaxTokens:   4096,
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
}

func TestIntegrationGeminiTokenSource(t *testing.T) {
	tokenCalls := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Role: "model", Parts: []GeminiPart{{Text: "ok"}}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	backend, err := NewGeminiBackend(llm.BackendConfig{
		BaseURL: server.URL,
	Model:   "gemini-2.5-flash",
		TokenSource: func() (string, error) {
			atomic.AddInt32(&tokenCalls, 1)
			return "dynamic-key", nil
		},
	})
	if err != nil {
		t.Fatalf("NewGeminiBackend failed: %v", err)
	}

	_, err = backend.Chat(context.Background(), []types.LLMMessage{
		{Role: "user", Content: "test"},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	// TokenSource is called once during NewGeminiBackend
	if count := atomic.LoadInt32(&tokenCalls); count != 1 {
		t.Fatalf("expected 1 token source call, got %d", count)
	}
}
