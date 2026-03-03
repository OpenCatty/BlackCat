package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

type mockLLM struct {
	calls     int
	responses []func() (*types.LLMResponse, error)
}

func (m *mockLLM) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.responses) {
		return m.responses[idx]()
	}
	return &types.LLMResponse{Content: "ok"}, nil
}

func (m *mockLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	return nil, nil
}

func TestRetryChat_TransientSuccess(t *testing.T) {
	m := &mockLLM{
		responses: []func() (*types.LLMResponse, error){
			func() (*types.LLMResponse, error) { return nil, errors.New("500 internal error") },
			func() (*types.LLMResponse, error) { return nil, errors.New("500 upstream error") },
			func() (*types.LLMResponse, error) { return &types.LLMResponse{Content: "success"}, nil },
		},
	}

	resp, err := RetryChat(context.Background(), m, nil, nil, 3)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp == nil || resp.Content != "success" {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if m.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", m.calls)
	}
}

func TestRetryChat_NonRetryable(t *testing.T) {
	m := &mockLLM{
		responses: []func() (*types.LLMResponse, error){
			func() (*types.LLMResponse, error) { return nil, errors.New("401 unauthorized") },
		},
	}

	_, err := RetryChat(context.Background(), m, nil, nil, 3)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, ErrAuthFailure) {
		t.Fatalf("expected ErrAuthFailure, got %v", err)
	}
	if m.calls != 1 {
		t.Fatalf("expected 1 call, got %d", m.calls)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "rate limit 429", err: errors.New("429 too many requests"), want: ErrRateLimit},
		{name: "auth 401", err: errors.New("401 unauthorized"), want: ErrAuthFailure},
		{name: "auth 403", err: errors.New("403 forbidden"), want: ErrAuthFailure},
		{name: "model 404", err: errors.New("404 model not found"), want: ErrModelNotFound},
		{name: "server 500", err: errors.New("500 internal server error"), want: ErrServerError},
		{name: "server 502", err: errors.New("502 bad gateway"), want: ErrServerError},
		{name: "server 503", err: errors.New("503 service unavailable"), want: ErrServerError},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: ErrTimeout},
		{name: "context length", err: errors.New("400 context length exceeded"), want: ErrContextLength},
		{name: "normal error", err: errors.New("some generic failure"), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyError(tt.err)
			if tt.want == nil {
				if got != tt.err {
					t.Fatalf("expected original error, got %v", got)
				}
				return
			}

			if !errors.Is(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
