package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// validationMockTool is a Tool implementation for validation tests.
type validationMockTool struct {
	name   string
	params json.RawMessage
	result string
	err    error
}

func (m *validationMockTool) Name() string                { return m.name }
func (m *validationMockTool) Description() string         { return "mock tool" }
func (m *validationMockTool) Parameters() json.RawMessage { return m.params }
func (m *validationMockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return m.result, m.err
}

func TestExecute_MissingRequiredField(t *testing.T) {
	r := NewRegistry()
	r.Register(&validationMockTool{
		name: "greet",
		params: json.RawMessage(`{
			"type": "object",
			"properties": {"name": {"type": "string"}},
			"required": ["name"]
		}`),
		result: "hello",
	})

	// Call with empty object — missing "name"
	_, err := r.Execute(context.Background(), "greet", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.ToolName != "greet" {
		t.Errorf("expected ToolName=greet, got %q", valErr.ToolName)
	}
	if want := "missing required field: name"; valErr.Details != want {
		t.Errorf("expected Details=%q, got %q", want, valErr.Details)
	}
}

func TestExecute_RequiredFieldPresent(t *testing.T) {
	r := NewRegistry()
	r.Register(&validationMockTool{
		name: "greet",
		params: json.RawMessage(`{
			"type": "object",
			"properties": {"name": {"type": "string"}},
			"required": ["name"]
		}`),
		result: "hello alice",
	})

	result, err := r.Execute(context.Background(), "greet", json.RawMessage(`{"name": "alice"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello alice" {
		t.Errorf("expected result=%q, got %q", "hello alice", result)
	}
}

func TestExecute_NilParametersPassesThrough(t *testing.T) {
	r := NewRegistry()
	r.Register(&validationMockTool{
		name:   "ping",
		params: nil,
		result: "pong",
	})

	result, err := r.Execute(context.Background(), "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "pong" {
		t.Errorf("expected result=%q, got %q", "pong", result)
	}
}

func TestExecute_InvalidJSON(t *testing.T) {
	r := NewRegistry()
	r.Register(&validationMockTool{
		name: "greet",
		params: json.RawMessage(`{
			"type": "object",
			"properties": {"name": {"type": "string"}},
			"required": ["name"]
		}`),
		result: "hello",
	})

	_, err := r.Execute(context.Background(), "greet", json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected ValidationError for invalid JSON, got nil")
	}
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.ToolName != "greet" {
		t.Errorf("expected ToolName=greet, got %q", valErr.ToolName)
	}
}

func TestExecute_EmptySchemaPassesThrough(t *testing.T) {
	r := NewRegistry()
	r.Register(&validationMockTool{
		name:   "noop",
		params: json.RawMessage(`{}`),
		result: "ok",
	})

	result, err := r.Execute(context.Background(), "noop", json.RawMessage(`{"anything": true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected result=%q, got %q", "ok", result)
	}
}

func TestExecute_ToolNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "nonexistent", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for nonexistent tool, got nil")
	}
}
