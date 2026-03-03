package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigUpdateToolNameAndSchema(t *testing.T) {
	tool := NewConfigUpdateTool("/tmp/blackcat.yaml")

	if got := tool.Name(); got != "config_update" {
		t.Fatalf("Name() = %q, want %q", got, "config_update")
	}
	if !strings.Contains(strings.ToLower(tool.Description()), "update") {
		t.Fatalf("Description() should explain update behavior, got %q", tool.Description())
	}
	if len(tool.Parameters()) == 0 {
		t.Fatal("Parameters() returned empty schema")
	}
}

func TestConfigUpdateToolExecuteSuccessPreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "blackcat.yaml")

	input := `# top comment
llm:
  # model comment
  model: gpt-4o
  temperature: 0.5
agent:
  name: sirius
`
	if err := os.WriteFile(configPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write input config: %v", err)
	}

	tool := NewConfigUpdateTool(configPath)
	result, err := tool.Execute(context.Background(), mustJSON(map[string]string{
		"field": "llm.model",
		"value": "gpt-5",
	}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "Config updated: llm.model = gpt-5" {
		t.Fatalf("unexpected result: %q", result)
	}

	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}
	text := string(updated)

	if !strings.Contains(text, "model: gpt-5") {
		t.Fatalf("expected updated model value, got:\n%s", text)
	}
	if !strings.Contains(text, "# top comment") || !strings.Contains(text, "# model comment") {
		t.Fatalf("expected comments to be preserved, got:\n%s", text)
	}
	if !strings.Contains(text, "temperature: 0.5") || !strings.Contains(text, "agent:") {
		t.Fatalf("expected unrelated content to be preserved, got:\n%s", text)
	}
}

func TestConfigUpdateToolProtectedField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "blackcat.yaml")

	input := "security:\n  autoPermit: false\n"
	if err := os.WriteFile(configPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write input config: %v", err)
	}

	tool := NewConfigUpdateTool(configPath)
	_, err := tool.Execute(context.Background(), mustJSON(map[string]string{
		"field": "security.autoPermit",
		"value": "true",
	}))
	if err == nil {
		t.Fatal("expected error for protected field, got nil")
	}
	if !strings.Contains(err.Error(), "protected") {
		t.Fatalf("expected protected-field error, got: %v", err)
	}

	updated, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config after protected update attempt: %v", readErr)
	}
	if string(updated) != input {
		t.Fatalf("protected update should not modify file, got:\n%s", string(updated))
	}
}

func TestConfigUpdateToolValidationAndTypeCoercion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "blackcat.yaml")

	input := "llm:\n  stream: false\n"
	if err := os.WriteFile(configPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write input config: %v", err)
	}

	tool := NewConfigUpdateTool(configPath)

	_, err := tool.Execute(context.Background(), mustJSON(map[string]string{
		"field": "",
		"value": "true",
	}))
	if err == nil || !strings.Contains(err.Error(), "field is required") {
		t.Fatalf("expected field validation error, got: %v", err)
	}

	_, err = tool.Execute(context.Background(), mustJSON(map[string]string{
		"field": "llm.stream",
		"value": "   ",
	}))
	if err == nil || !strings.Contains(err.Error(), "value is required") {
		t.Fatalf("expected value validation error, got: %v", err)
	}

	_, err = tool.Execute(context.Background(), mustJSON(map[string]string{
		"field": "llm.stream",
		"value": "true",
	}))
	if err != nil {
		t.Fatalf("unexpected error updating bool field: %v", err)
	}

	updated, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read updated config: %v", readErr)
	}
	if !strings.Contains(string(updated), "stream: true") {
		t.Fatalf("expected boolean coercion in YAML output, got:\n%s", string(updated))
	}
}
