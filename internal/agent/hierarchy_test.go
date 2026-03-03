package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/tools"
)

func TestBuildSystemPromptHierarchical(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	subdir := filepath.Join(workspace, "subdir")

	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "AGENTS.md"), []byte("root instructions"), 0o644); err != nil {
		t.Fatalf("write root AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "AGENTS.md"), []byte("subdir instructions"), 0o644); err != nil {
		t.Fatalf("write subdir AGENTS.md: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("chdir subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	loop := NewLoop(LoopConfig{
		Tools:        tools.NewRegistry(),
		Scrubber:     security.NewScrubber(),
		WorkspaceDir: workspace,
		MaxTurns:     5,
	})

	systemPrompt, err := loop.buildSystemPrompt(ctx)
	if err != nil {
		t.Fatalf("buildSystemPrompt() error = %v", err)
	}

	expectedMerged := "root instructions\n\n---\n\nsubdir instructions"
	if !strings.Contains(systemPrompt, expectedMerged) {
		t.Fatalf("system prompt missing merged AGENTS.md content: %q", systemPrompt)
	}
}

func TestBuildSystemPromptSingle(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()

	if err := os.WriteFile(filepath.Join(workspace, "AGENTS.md"), []byte("single instructions"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	loop := NewLoop(LoopConfig{
		Tools:        tools.NewRegistry(),
		Scrubber:     security.NewScrubber(),
		WorkspaceDir: workspace,
		MaxTurns:     5,
	})

	systemPrompt, err := loop.buildSystemPrompt(ctx)
	if err != nil {
		t.Fatalf("buildSystemPrompt() error = %v", err)
	}

	if !strings.Contains(systemPrompt, "single instructions") {
		t.Fatalf("system prompt missing single AGENTS.md content: %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, "\n\n---\n\n") {
		t.Fatalf("system prompt should not contain hierarchy separator for single AGENTS.md: %q", systemPrompt)
	}
}
