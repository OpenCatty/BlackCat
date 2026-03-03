//go:build integration

package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/internal/agent"
	"github.com/startower-observability/blackcat/internal/channel"
	"github.com/startower-observability/blackcat/internal/memory"
	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/tools"
	"github.com/startower-observability/blackcat/internal/types"
)

type mockLLM struct {
	responses []types.LLMResponse
	callIdx   int
	mu        sync.Mutex
}

func (m *mockLLM) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.responses) == 0 {
		return &types.LLMResponse{}, nil
	}

	if m.callIdx >= len(m.responses) {
		last := m.responses[len(m.responses)-1]
		return &last, nil
	}

	resp := m.responses[m.callIdx]
	m.callIdx++
	return &resp, nil
}

func (m *mockLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	return nil, errors.New("mockLLM Stream not implemented")
}

type errorLLM struct {
	err error
}

func (e *errorLLM) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	return nil, e.err
}

func (e *errorLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	return nil, errors.New("errorLLM Stream not implemented")
}

type mockTool struct {
	name      string
	result    string
	callCount int
	mu        sync.Mutex
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "integration mock tool"
}

func (m *mockTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.result, nil
}

func (m *mockTool) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestFullMessageFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockChannel := channel.NewMockChannel(types.ChannelTelegram)
	bus := channel.NewMessageBus(16)
	if err := bus.Register(mockChannel); err != nil {
		t.Fatalf("bus.Register() error = %v", err)
	}
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus.Start() error = %v", err)
	}
	defer func() { _ = bus.Stop() }()

	llm := &mockLLM{responses: []types.LLMResponse{{Content: "Hello from BlackCat!"}}}
	store := memory.NewFileStore(filepath.Join(t.TempDir(), "MEMORY.md"))
	registry := tools.NewRegistry()
	loop := agent.NewLoop(agent.LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		Memory:   store,
		MaxTurns: 10,
	})

	incoming := types.Message{
		ID:          "m-1",
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chat-1",
		UserID:      "u-1",
		Content:     "hello",
		Timestamp:   time.Now(),
	}
	mockChannel.Inject(incoming)

	var received types.Message
	select {
	case received = <-bus.Messages():
	case <-ctx.Done():
		t.Fatal("timed out waiting for bus message")
	}

	execution, err := loop.Run(ctx, received.Content)
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	reply := types.Message{
		ID:          "reply-m-1",
		ChannelType: received.ChannelType,
		ChannelID:   received.ChannelID,
		Content:     execution.Response,
		ReplyTo:     received.ID,
		Timestamp:   time.Now(),
	}
	if err := bus.Send(ctx, received.ChannelType, reply); err != nil {
		t.Fatalf("bus.Send() error = %v", err)
	}

	sent := mockChannel.Sent()
	if len(sent) != 1 {
		t.Fatalf("len(mockChannel.Sent()) = %d, want 1", len(sent))
	}
	if sent[0].Content != "Hello from BlackCat!" {
		t.Fatalf("sent response = %q, want %q", sent[0].Content, "Hello from BlackCat!")
	}
}

func TestMessageFlowWithToolCall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	llm := &mockLLM{responses: []types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"message":"hello"}`)}}},
		{Content: "Tool completed successfully."},
	}}

	registry := tools.NewRegistry()
	echoTool := &mockTool{name: "echo", result: "echoed: hello"}
	registry.Register(echoTool)

	loop := agent.NewLoop(agent.LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		Memory:   memory.NewFileStore(filepath.Join(t.TempDir(), "MEMORY.md")),
		MaxTurns: 10,
	})

	execution, err := loop.Run(ctx, "run echo")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	if echoTool.Calls() != 1 {
		t.Fatalf("echo tool calls = %d, want 1", echoTool.Calls())
	}
	if execution.Response != "Tool completed successfully." {
		t.Fatalf("execution.Response = %q, want %q", execution.Response, "Tool completed successfully.")
	}
}

func TestMemoryPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memoryPath := filepath.Join(t.TempDir(), "MEMORY.md")
	store := memory.NewFileStore(memoryPath)

	loop := agent.NewLoop(agent.LoopConfig{
		LLM:      &mockLLM{responses: []types.LLMResponse{{Content: "Persisted"}}},
		Tools:    tools.NewRegistry(),
		Scrubber: security.NewScrubber(),
		Memory:   store,
		MaxTurns: 10,
	})

	userMessage := "remember this integration message"
	if _, err := loop.Run(ctx, userMessage); err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}

	if err := store.Write(ctx, memory.Entry{
		Timestamp: time.Now().UTC(),
		Content:   userMessage,
		Tags:      []string{"integration"},
	}); err != nil {
		t.Fatalf("store.Write() error = %v", err)
	}

	entries, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("store.Read() error = %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected memory entries, got 0")
	}
	if entries[len(entries)-1].Content != userMessage {
		t.Fatalf("last memory entry = %q, want %q", entries[len(entries)-1].Content, userMessage)
	}

	if _, err := os.Stat(memoryPath); err != nil {
		t.Fatalf("memory file not found at %s: %v", memoryPath, err)
	}
}

func TestConcurrentMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockChannel := channel.NewMockChannel(types.ChannelTelegram)
	bus := channel.NewMessageBus(16)
	if err := bus.Register(mockChannel); err != nil {
		t.Fatalf("bus.Register() error = %v", err)
	}
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus.Start() error = %v", err)
	}
	defer func() { _ = bus.Stop() }()

	llm := &mockLLM{responses: []types.LLMResponse{
		{Content: "response-1"},
		{Content: "response-2"},
		{Content: "response-3"},
	}}
	loop := agent.NewLoop(agent.LoopConfig{
		LLM:      llm,
		Tools:    tools.NewRegistry(),
		Scrubber: security.NewScrubber(),
		Memory:   memory.NewFileStore(filepath.Join(t.TempDir(), "MEMORY.md")),
		MaxTurns: 10,
	})

	for i := 1; i <= 3; i++ {
		mockChannel.Inject(types.Message{
			ID:          fmt.Sprintf("m-%d", i),
			ChannelType: types.ChannelTelegram,
			ChannelID:   "chat-1",
			UserID:      "user-1",
			Content:     fmt.Sprintf("message-%d", i),
			Timestamp:   time.Now(),
		})
	}

	workers := 2
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		var msg types.Message
		select {
		case msg = <-bus.Messages():
		case <-ctx.Done():
			t.Fatal("timed out waiting for concurrent bus message")
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(m types.Message) {
			defer func() {
				<-sem
				wg.Done()
			}()

			execution, err := loop.Run(ctx, m.Content)
			if err != nil {
				t.Errorf("loop.Run() error = %v", err)
				return
			}

			reply := types.Message{
				ID:          "reply-" + m.ID,
				ChannelType: m.ChannelType,
				ChannelID:   m.ChannelID,
				Content:     execution.Response,
				ReplyTo:     m.ID,
				Timestamp:   time.Now(),
			}

			if err := bus.Send(ctx, m.ChannelType, reply); err != nil {
				t.Errorf("bus.Send() error = %v", err)
			}
		}(msg)
	}

	wg.Wait()

	sent := mockChannel.Sent()
	if len(sent) != 3 {
		t.Fatalf("len(mockChannel.Sent()) = %d, want 3", len(sent))
	}
}

func TestAgentErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockChannel := channel.NewMockChannel(types.ChannelTelegram)
	bus := channel.NewMessageBus(8)
	if err := bus.Register(mockChannel); err != nil {
		t.Fatalf("bus.Register() error = %v", err)
	}
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus.Start() error = %v", err)
	}
	defer func() { _ = bus.Stop() }()

	loop := agent.NewLoop(agent.LoopConfig{
		LLM:      &errorLLM{err: errors.New("llm unavailable")},
		Tools:    tools.NewRegistry(),
		Scrubber: security.NewScrubber(),
		Memory:   memory.NewFileStore(filepath.Join(t.TempDir(), "MEMORY.md")),
		MaxTurns: 10,
	})

	_, err := loop.Run(ctx, "trigger error")
	if err == nil {
		t.Fatal("expected loop.Run() error, got nil")
	}

	errorResponse := "Error: " + err.Error()
	if err := bus.Send(ctx, types.ChannelTelegram, types.Message{
		ID:          "reply-err-1",
		ChannelType: types.ChannelTelegram,
		ChannelID:   "chat-1",
		Content:     errorResponse,
		Timestamp:   time.Now(),
	}); err != nil {
		t.Fatalf("bus.Send() error = %v", err)
	}

	sent := mockChannel.Sent()
	if len(sent) != 1 {
		t.Fatalf("len(mockChannel.Sent()) = %d, want 1", len(sent))
	}
	if sent[0].Content != errorResponse {
		t.Fatalf("sent error response = %q, want %q", sent[0].Content, errorResponse)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mockChannel := channel.NewMockChannel(types.ChannelTelegram)
	bus := channel.NewMessageBus(4)
	if err := bus.Register(mockChannel); err != nil {
		t.Fatalf("bus.Register() error = %v", err)
	}
	if err := bus.Start(ctx); err != nil {
		t.Fatalf("bus.Start() error = %v", err)
	}

	cancel()

	done := make(chan error, 1)
	go func() {
		done <- bus.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("bus.Stop() error after cancellation = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("bus.Stop() timed out after context cancellation")
	}
}
