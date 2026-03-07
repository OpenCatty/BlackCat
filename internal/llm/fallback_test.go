package llm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

// fallbackMock implements Backend for fallback tests with a call counter.
type fallbackMock struct {
	chatResp  *types.LLMResponse
	chatErr   error
	streamErr error
	chatCalls int
}

func (m *fallbackMock) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	m.chatCalls++
	return m.chatResp, m.chatErr
}

func (m *fallbackMock) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

func TestFallback_PrimarySucceeds(t *testing.T) {
	primary := &fallbackMock{chatResp: &types.LLMResponse{Content: "primary"}}
	fallback := &fallbackMock{chatResp: &types.LLMResponse{Content: "fallback"}}

	fb, err := NewFallbackBackend([]Backend{primary, fallback}, []string{"primary", "fallback"})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := fb.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Content != "primary" {
		t.Fatalf("expected primary response, got %q", resp.Content)
	}
	if fallback.chatCalls != 0 {
		t.Fatalf("fallback should not have been called, got %d calls", fallback.chatCalls)
	}
}

func TestFallback_PrimaryFailsFallbackSucceeds(t *testing.T) {
	// Use ErrAuthFailure — non-retryable, so RetryChat returns immediately
	primary := &fallbackMock{chatErr: ErrAuthFailure}
	fallback := &fallbackMock{chatResp: &types.LLMResponse{Content: "fallback-ok"}}

	fb, err := NewFallbackBackend([]Backend{primary, fallback}, []string{"primary", "fallback"})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := fb.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Content != "fallback-ok" {
		t.Fatalf("expected fallback response, got %q", resp.Content)
	}
	if primary.chatCalls != 1 {
		t.Fatalf("primary should be called once, got %d", primary.chatCalls)
	}
	if fallback.chatCalls != 1 {
		t.Fatalf("fallback should be called once, got %d", fallback.chatCalls)
	}
}

func TestFallback_AuthFailureImmediateFallback(t *testing.T) {
	// ErrAuthFailure is non-retryable; RetryChat returns it immediately,
	// then FallbackBackend moves to the next backend.
	primary := &fallbackMock{chatErr: ErrAuthFailure}
	fallback := &fallbackMock{chatResp: &types.LLMResponse{Content: "auth-fallback"}}

	fb, err := NewFallbackBackend([]Backend{primary, fallback}, []string{"primary", "fallback"})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := fb.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Content != "auth-fallback" {
		t.Fatalf("expected auth-fallback response, got %q", resp.Content)
	}
	// Verify primary was only called once (no retries for auth failure)
	if primary.chatCalls != 1 {
		t.Fatalf("primary should be called exactly once (no retries for auth), got %d", primary.chatCalls)
	}
}

func TestFallback_ContextLengthNoFallback(t *testing.T) {
	primary := &fallbackMock{chatErr: ErrContextLength}
	fallback := &fallbackMock{chatResp: &types.LLMResponse{Content: "should-not-reach"}}

	fb, err := NewFallbackBackend([]Backend{primary, fallback}, []string{"primary", "fallback"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fb.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrContextLength) {
		t.Fatalf("expected ErrContextLength, got %v", err)
	}
	// Fallback must NOT have been called
	if fallback.chatCalls != 0 {
		t.Fatalf("fallback should not be called for ErrContextLength, got %d calls", fallback.chatCalls)
	}
	// Primary called exactly once (ErrContextLength is non-retryable)
	if primary.chatCalls != 1 {
		t.Fatalf("primary should be called exactly once, got %d", primary.chatCalls)
	}
}

func TestFallback_AllProvidersFail(t *testing.T) {
	primary := &fallbackMock{chatErr: ErrAuthFailure}
	fallback := &fallbackMock{chatErr: ErrAuthFailure}

	fb, err := NewFallbackBackend([]Backend{primary, fallback}, []string{"primary", "fallback"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fb.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "all providers failed") {
		t.Fatalf("expected 'all providers failed' in error, got %v", err)
	}
}

func TestFallback_ContextCancelled(t *testing.T) {
	primary := &fallbackMock{chatResp: &types.LLMResponse{Content: "should-not-reach"}}

	fb, err := NewFallbackBackend([]Backend{primary}, []string{"primary"})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Chat

	_, err = fb.Chat(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if primary.chatCalls != 0 {
		t.Fatalf("primary should not be called when context is cancelled, got %d", primary.chatCalls)
	}
}

func TestFallback_SingleBackend(t *testing.T) {
	primary := &fallbackMock{chatResp: &types.LLMResponse{Content: "single"}}

	fb, err := NewFallbackBackend([]Backend{primary}, []string{"primary"})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := fb.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Content != "single" {
		t.Fatalf("expected single response, got %q", resp.Content)
	}
	if primary.chatCalls != 1 {
		t.Fatalf("expected 1 call, got %d", primary.chatCalls)
	}
}
