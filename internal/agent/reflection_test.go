package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

// --- Mock types unique to reflection tests ---

// reflectionMockLLM implements types.LLMClient for reflection tests.
type reflectionMockLLM struct {
	response string
	err      error
	called   bool
}

func (m *reflectionMockLLM) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return &types.LLMResponse{Content: m.response}, nil
}

func (m *reflectionMockLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

// reflectionMockStore implements ArchivalStoreIface for reflection tests.
type reflectionMockStore struct {
	insertedContent string
	insertedTags    []string
	insertedUserID  string
	err             error
	called          bool
}

func (m *reflectionMockStore) InsertArchival(_ context.Context, userID, content string, tags []string, _ []float32) error {
	m.called = true
	m.insertedUserID = userID
	m.insertedContent = content
	m.insertedTags = tags
	return m.err
}

// --- Tests ---

func TestReflect_NilLLM_ReturnsNil(t *testing.T) {
	store := &reflectionMockStore{}
	r := NewReflector(nil, store)

	lesson, err := r.Reflect(context.Background(), "user1", "tool_x", `{"a":1}`, "Tool error: boom")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if lesson != "" {
		t.Fatalf("expected empty lesson, got %q", lesson)
	}
	if store.called {
		t.Fatal("store should not have been called when llm is nil")
	}
}

func TestReflect_NilStore_ReturnsNil(t *testing.T) {
	llm := &reflectionMockLLM{response: "some lesson"}
	r := NewReflector(llm, nil)

	lesson, err := r.Reflect(context.Background(), "user1", "tool_x", `{"a":1}`, "Tool error: boom")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if lesson != "" {
		t.Fatalf("expected empty lesson, got %q", lesson)
	}
	if llm.called {
		t.Fatal("llm should not have been called when store is nil")
	}
}

func TestReflect_Success_StoresLesson(t *testing.T) {
	expectedLesson := "Always validate input arguments before calling external APIs."
	llm := &reflectionMockLLM{response: expectedLesson}
	store := &reflectionMockStore{}
	r := NewReflector(llm, store)

	lesson, err := r.Reflect(context.Background(), "user42", "http_fetch", `{"url":"bad"}`, "Tool error: invalid URL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lesson != expectedLesson {
		t.Fatalf("lesson = %q, want %q", lesson, expectedLesson)
	}
	if !llm.called {
		t.Fatal("llm.Chat should have been called")
	}
	if !store.called {
		t.Fatal("store.InsertArchival should have been called")
	}
	if store.insertedUserID != "user42" {
		t.Fatalf("userID = %q, want %q", store.insertedUserID, "user42")
	}
	if store.insertedContent != expectedLesson {
		t.Fatalf("content = %q, want %q", store.insertedContent, expectedLesson)
	}
	if len(store.insertedTags) != 2 || store.insertedTags[0] != "reflection" || store.insertedTags[1] != "http_fetch" {
		t.Fatalf("tags = %v, want [reflection, http_fetch]", store.insertedTags)
	}
}

func TestReflect_LLMError_ReturnsError(t *testing.T) {
	llm := &reflectionMockLLM{err: errors.New("llm down")}
	store := &reflectionMockStore{}
	r := NewReflector(llm, store)

	lesson, err := r.Reflect(context.Background(), "user1", "tool_x", `{}`, "error")
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
	if lesson != "" {
		t.Fatalf("expected empty lesson on LLM error, got %q", lesson)
	}
	if store.called {
		t.Fatal("store should not be called when LLM fails")
	}
}

func TestReflect_TruncatesLongLesson(t *testing.T) {
	longLesson := strings.Repeat("a", 600)
	llm := &reflectionMockLLM{response: longLesson}
	store := &reflectionMockStore{}
	r := NewReflector(llm, store)

	lesson, err := r.Reflect(context.Background(), "user1", "tool_x", `{}`, "error result")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lesson) != maxLessonLength {
		t.Fatalf("lesson length = %d, want %d (truncated)", len(lesson), maxLessonLength)
	}
	if store.insertedContent != lesson {
		t.Fatal("stored content should match truncated lesson")
	}
}

func TestReflectionCount_IncrementsThroughLoop(t *testing.T) {
	exec := NewExecution(10)
	if exec.ReflectionCount != 0 {
		t.Fatalf("ReflectionCount = %d, want 0", exec.ReflectionCount)
	}

	exec.ReflectionCount++
	if exec.ReflectionCount != 1 {
		t.Fatalf("ReflectionCount = %d, want 1", exec.ReflectionCount)
	}

	exec.ReflectionCount++
	if exec.ReflectionCount != 2 {
		t.Fatalf("ReflectionCount = %d, want 2", exec.ReflectionCount)
	}

	// Verify max check works
	if exec.ReflectionCount < MaxReflections {
		t.Fatalf("ReflectionCount %d should be >= MaxReflections %d", exec.ReflectionCount, MaxReflections)
	}
}

func TestReflect_StoreError_ReturnsLessonAndError(t *testing.T) {
	llm := &reflectionMockLLM{response: "some lesson"}
	store := &reflectionMockStore{err: errors.New("db write failed")}
	r := NewReflector(llm, store)

	lesson, err := r.Reflect(context.Background(), "user1", "tool_x", `{}`, "Tool error: boom")
	if err == nil {
		t.Fatal("expected error from store failure")
	}
	if lesson != "some lesson" {
		t.Fatalf("lesson = %q, want %q (should return lesson even on store error)", lesson, "some lesson")
	}
}
