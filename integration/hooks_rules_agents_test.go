package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/hooks"
	"github.com/startower-observability/blackcat/internal/rules"
	"github.com/startower-observability/blackcat/types"
	"github.com/startower-observability/blackcat/workspace"
)

func TestHooksRulesIntegration(t *testing.T) {
	root := t.TempDir()
	rulesDir := filepath.Join(root, "rules")

	writeFile(t, filepath.Join(rulesDir, "go-style.md"), `---
name: go-style
globs:
  - "internal/**/*.go"
---
Prefer short, clear function names.
`)

	engine := rules.NewEngine()
	if err := engine.LoadRules(rulesDir); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	handler := rules.NewRulesHookHandler(engine)
	registry := hooks.NewHookRegistry()
	registry.Register(hooks.PostFileRead, handler.HandlePostFileRead)

	hctx := &hooks.HookContext{
		FilePath:    "internal/hooks/registry.go",
		LLMResponse: &types.LLMResponse{Content: "existing response"},
	}

	if err := registry.Fire(context.Background(), hooks.PostFileRead, hctx); err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if !strings.Contains(hctx.LLMResponse.Content, "<!-- Rules -->") {
		t.Fatalf("expected rules block in response, got: %q", hctx.LLMResponse.Content)
	}
	if !strings.Contains(hctx.LLMResponse.Content, "Prefer short, clear function names.") {
		t.Fatalf("expected rule content appended, got: %q", hctx.LLMResponse.Content)
	}
}

func TestHierarchicalAgentsWithHooks(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "apps")

	writeFile(t, filepath.Join(root, "AGENTS.md"), "Root agent policy")
	writeFile(t, filepath.Join(subDir, "AGENTS.md"), "Subdir agent policy")

	entries, err := workspace.LoadHierarchicalAgents(root, subDir)
	if err != nil {
		t.Fatalf("LoadHierarchicalAgents failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 hierarchy entries, got %d", len(entries))
	}

	merged := workspace.MergeHierarchyEntries(entries)
	if !strings.Contains(merged, "Root agent policy") || !strings.Contains(merged, "Subdir agent policy") {
		t.Fatalf("merged hierarchy missing expected contents: %q", merged)
	}
	if !strings.Contains(merged, "\n\n---\n\n") {
		t.Fatalf("expected merged hierarchy separator, got: %q", merged)
	}
}

func TestRulesAndAgentsCombined(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "foo")
	rulesDir := filepath.Join(root, "rules")

	writeFile(t, filepath.Join(root, "AGENTS.md"), "Top-level AGENTS instructions")
	writeFile(t, filepath.Join(subDir, "AGENTS.md"), "Service-specific AGENTS instructions")
	writeFile(t, filepath.Join(rulesDir, "go-style.md"), `---
name: go-style
globs:
  - "**/*.go"
---
Always return wrapped errors.
`)

	entries, err := workspace.LoadHierarchicalAgents(root, subDir)
	if err != nil {
		t.Fatalf("LoadHierarchicalAgents failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 hierarchy entries, got %d", len(entries))
	}

	engine := rules.NewEngine()
	if err := engine.LoadRules(rulesDir); err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	handler := rules.NewRulesHookHandler(engine)
	registry := hooks.NewHookRegistry()
	registry.Register(hooks.PostFileRead, handler.HandlePostFileRead)

	hctx := &hooks.HookContext{
		FilePath:    "foo/bar.go",
		LLMResponse: &types.LLMResponse{Content: "file summary"},
	}
	if err := registry.Fire(context.Background(), hooks.PostFileRead, hctx); err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if !strings.Contains(hctx.LLMResponse.Content, "Always return wrapped errors.") {
		t.Fatalf("expected matched rule content in LLM response, got: %q", hctx.LLMResponse.Content)
	}

	merged := workspace.MergeHierarchyEntries(entries)
	if !strings.Contains(merged, "Top-level AGENTS instructions") || !strings.Contains(merged, "Service-specific AGENTS instructions") {
		t.Fatalf("merged hierarchy missing AGENTS content: %q", merged)
	}
}

func TestHookPanicRecovery(t *testing.T) {
	registry := hooks.NewHookRegistry()
	called := 0

	registry.Register(hooks.PostChat, func(ctx *hooks.HookContext) error {
		panic("simulated hook panic")
	})
	registry.Register(hooks.PostChat, func(ctx *hooks.HookContext) error {
		called++
		return nil
	})

	err := registry.Fire(context.Background(), hooks.PostChat, &hooks.HookContext{})
	if err == nil {
		t.Fatalf("expected non-nil error from recovered panic")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("expected panic text in error, got: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected second post-event hook to run once, got %d", called)
	}
}

func TestAgentsCircularSymlink(t *testing.T) {
	root := t.TempDir()
	subA := filepath.Join(root, "subA")
	loop := filepath.Join(subA, "loop")

	writeFile(t, filepath.Join(subA, "AGENTS.md"), "subA instructions")

	if err := os.Symlink(subA, loop); err != nil {
		t.Skipf("symlink creation requires elevated privileges: %v", err)
	}

	entries, err := workspace.LoadHierarchicalAgents(root, loop)
	if err == nil {
		if len(entries) == 0 {
			t.Fatalf("expected at least one hierarchy entry when no error is returned")
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed for %s: %v", path, err)
	}
}
