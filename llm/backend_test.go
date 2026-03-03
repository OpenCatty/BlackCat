package llm

import (
	"context"
	"testing"

	"github.com/startower-observability/blackcat/types"
)

// mockBackend is a test double that satisfies the Backend interface.
type mockBackend struct {
	chatResp  *types.LLMResponse
	chatErr   error
	streamCh  chan types.Chunk
	streamErr error
}

func (m *mockBackend) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	return m.chatResp, m.chatErr
}

func (m *mockBackend) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.streamCh, nil
}

func (m *mockBackend) Info() BackendInfo {
	return BackendInfo{
		Name:       "mock",
		Models:     []string{"mock-model"},
		AuthMethod: "api-key",
	}
}

// TestBackendInterface verifies that a mock struct can satisfy the Backend interface.
func TestBackendInterface(t *testing.T) {
	mock := &mockBackend{
		chatResp: &types.LLMResponse{
			Content: "hello",
			Model:   "mock-model",
		},
		streamCh: make(chan types.Chunk),
	}

	// Verify Backend interface satisfaction at compile time.
	var _ Backend = mock

	// Verify InfoProvider interface satisfaction at compile time.
	var _ InfoProvider = mock

	// Test Chat method.
	resp, err := mock.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("expected content 'hello', got %q", resp.Content)
	}
	if resp.Model != "mock-model" {
		t.Errorf("expected model 'mock-model', got %q", resp.Model)
	}

	// Test Stream method.
	ch, err := mock.Stream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Test Info method.
	info := mock.Info()
	if info.Name != "mock" {
		t.Errorf("expected name 'mock', got %q", info.Name)
	}
	if info.AuthMethod != "api-key" {
		t.Errorf("expected auth method 'api-key', got %q", info.AuthMethod)
	}
}

// TestBackendFactory verifies the BackendFactory type works.
func TestBackendFactory(t *testing.T) {
	factory := BackendFactory(func(cfg BackendConfig) (Backend, error) {
		return &mockBackend{
			chatResp: &types.LLMResponse{
				Content: "from-factory",
				Model:   cfg.Model,
			},
		}, nil
	})

	backend, err := factory(BackendConfig{
		Model:       "test-model",
		Temperature: 0.7,
		MaxTokens:   4096,
	})
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}

	resp, err := backend.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("chat error: %v", err)
	}
	if resp.Content != "from-factory" {
		t.Errorf("expected 'from-factory', got %q", resp.Content)
	}
	if resp.Model != "test-model" {
		t.Errorf("expected model 'test-model', got %q", resp.Model)
	}
}

// TestBackendConfigTokenSource verifies TokenSource field is callable.
func TestBackendConfigTokenSource(t *testing.T) {
	cfg := BackendConfig{
		TokenSource: func() (string, error) {
			return "test-token-123", nil
		},
	}

	token, err := cfg.TokenSource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-token-123" {
		t.Errorf("expected 'test-token-123', got %q", token)
	}
}
