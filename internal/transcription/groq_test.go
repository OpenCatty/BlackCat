package transcription

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewGroqClient_Defaults(t *testing.T) {
	c := NewGroqClient("test-key")
	if c.apiKey != "test-key" {
		t.Errorf("expected apiKey 'test-key', got %q", c.apiKey)
	}
	if c.model != defaultModel {
		t.Errorf("expected model %q, got %q", defaultModel, c.model)
	}
	if c.maxFileSizeB != defaultMaxFileSizeMB*1024*1024 {
		t.Errorf("expected maxFileSizeB %d, got %d", defaultMaxFileSizeMB*1024*1024, c.maxFileSizeB)
	}
	if c.httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}
}

func TestNewGroqClient_WithOptions(t *testing.T) {
	custom := &http.Client{}
	c := NewGroqClient("key", WithModel("whisper-large-v3"), WithMaxFileSizeMB(10), WithHTTPClient(custom))
	if c.model != "whisper-large-v3" {
		t.Errorf("expected model 'whisper-large-v3', got %q", c.model)
	}
	if c.maxFileSizeB != 10*1024*1024 {
		t.Errorf("expected maxFileSizeB %d, got %d", 10*1024*1024, c.maxFileSizeB)
	}
	if c.httpClient != custom {
		t.Error("expected custom httpClient to be set")
	}
}

func TestTranscribeFile_TooLarge(t *testing.T) {
	c := NewGroqClient("test-key", WithMaxFileSizeMB(1))
	bigAudio := make([]byte, 2*1024*1024) // 2 MB > 1 MB limit
	_, err := c.TranscribeFile(context.Background(), "audio.ogg", bigAudio)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !errors.Is(err, ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got: %v", err)
	}
}

func TestTranscribeFile_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header is present
		auth := r.Header.Get("Authorization")
		if auth != "Bearer bad-key" {
			t.Errorf("expected 'Bearer bad-key', got %q", auth)
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer srv.Close()

	c := NewGroqClient("bad-key", WithHTTPClient(redirectClient(srv.URL)))
	_, err := c.TranscribeFile(context.Background(), "audio.ogg", []byte("fake"))
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if got := err.Error(); got != "groq API error: Invalid API key" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestTranscribeFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			t.Error("expected Content-Type header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer srv.Close()

	c := NewGroqClient("test-key", WithHTTPClient(redirectClient(srv.URL)))
	text, err := c.TranscribeFile(context.Background(), "audio.ogg", []byte("fake audio"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got %q", text)
	}
}

// redirectClient returns an HTTP client that redirects all requests to the test server.
func redirectClient(baseURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{baseURL: baseURL},
	}
}

type redirectTransport struct {
	baseURL string
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	parsed, err := url.Parse(rt.baseURL)
	if err != nil {
		return nil, err
	}
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = parsed.Scheme
	req2.URL.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(req2)
}
