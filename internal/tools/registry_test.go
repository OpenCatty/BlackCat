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

func TestFilter_SubsetKeepsOnlyAllowedTools(t *testing.T) {
	r := NewRegistry()
	r.Register(&filterMockTool{name: "tool_a"})
	r.Register(&filterMockTool{name: "tool_b"})
	r.Register(&filterMockTool{name: "tool_c"})
	
	filtered := r.Filter([]string{"tool_a", "tool_c"})
	
	_, err1 := filtered.Get("tool_a")
	if err1 != nil {
		t.Errorf("expected tool_a to be in filtered registry, got error: %v", err1)
	}
	
	_, err2 := filtered.Get("tool_b")
	if err2 == nil {
		t.Error("expected tool_b to NOT be in filtered registry")
	}
	
	_, err3 := filtered.Get("tool_c")
	if err3 != nil {
		t.Errorf("expected tool_c to be in filtered registry, got error: %v", err3)
	}
	
	// Verify original registry is unchanged
	_, origErr := r.Get("tool_b")
	if origErr != nil {
		t.Error("original registry should still contain tool_b")
	}
}
	
func TestFilter_NilAllowedToolsReturnsFullCopy(t *testing.T) {
	r := NewRegistry()
	r.Register(&filterMockTool{name: "tool_a"})
	r.Register(&filterMockTool{name: "tool_b"})
	
	filtered := r.Filter(nil)
	
	_, err1 := filtered.Get("tool_a")
	if err1 != nil {
		t.Errorf("expected tool_a in full copy, got error: %v", err1)
	}
	
	_, err2 := filtered.Get("tool_b")
	if err2 != nil {
		t.Errorf("expected tool_b in full copy, got error: %v", err2)
	}
}
	
func TestFilter_EmptyAllowedToolsReturnsFullCopy(t *testing.T) {
	r := NewRegistry()
	r.Register(&filterMockTool{name: "tool_x"})
	r.Register(&filterMockTool{name: "tool_y"})
	
	filtered := r.Filter([]string{})
	
	_, err1 := filtered.Get("tool_x")
	if err1 != nil {
		t.Errorf("expected tool_x in full copy, got error: %v", err1)
	}
	
	_, err2 := filtered.Get("tool_y")
	if err2 != nil {
		t.Errorf("expected tool_y in full copy, got error: %v", err2)
	}
}
	
func TestFilter_DoesNotMutateOriginal(t *testing.T) {
	r := NewRegistry()
	r.Register(&filterMockTool{name: "tool_a"})
	r.Register(&filterMockTool{name: "tool_b"})
	
	filtered := r.Filter([]string{"tool_a"})
	
	// Original should still have both tools
	_, errOrig1 := r.Get("tool_a")
	_, errOrig2 := r.Get("tool_b")
	if errOrig1 != nil || errOrig2 != nil {
		t.Error("original registry was mutated")
	}
	
	// Filtered should only have tool_a
	_, errFilt := filtered.Get("tool_a")
	if errFilt != nil {
		t.Error("filtered registry missing tool_a")
	}
}
	
// filterMockTool is a mock Tool implementation for Filter tests
type filterMockTool struct {
	name string
}
	
func (m *filterMockTool) Name() string                { return m.name }
func (m *filterMockTool) Description() string         { return "filter test tool" }
func (m *filterMockTool) Parameters() json.RawMessage { return json.RawMessage(`{}`) }
func (m *filterMockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "ok", nil
}
