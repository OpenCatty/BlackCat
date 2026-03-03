package rules

import (
	"testing"

	"github.com/startower-observability/blackcat/internal/hooks"
	"github.com/startower-observability/blackcat/internal/types"
)

func TestHookInjection(t *testing.T) {
	engine := NewEngine()
	engine.rules = []Rule{
		{
			Name:    "Go Rule",
			Globs:   []string{"src/*.go"},
			Content: "Use wrapped errors with context.",
		},
	}

	handler := NewRulesHookHandler(engine)
	ctx := &hooks.HookContext{
		FilePath: "src/main.go",
		LLMResponse: &types.LLMResponse{
			Content: "Original file content",
		},
	}

	if err := handler.HandlePostFileRead(ctx); err != nil {
		t.Fatalf("HandlePostFileRead() error = %v", err)
	}

	want := "Original file content\n\n<!-- Rules -->\nUse wrapped errors with context.\n<!-- /Rules -->"
	if ctx.LLMResponse.Content != want {
		t.Fatalf("LLMResponse.Content = %q, want %q", ctx.LLMResponse.Content, want)
	}
}

func TestHookNoMatch(t *testing.T) {
	engine := NewEngine()
	engine.rules = []Rule{
		{
			Name:    "Go Rule",
			Globs:   []string{"src/*.go"},
			Content: "Use wrapped errors with context.",
		},
	}

	handler := NewRulesHookHandler(engine)
	ctx := &hooks.HookContext{
		FilePath: "docs/readme.md",
		LLMResponse: &types.LLMResponse{
			Content: "Original file content",
		},
	}

	if err := handler.HandlePostFileRead(ctx); err != nil {
		t.Fatalf("HandlePostFileRead() error = %v", err)
	}

	if ctx.LLMResponse.Content != "Original file content" {
		t.Fatalf("LLMResponse.Content = %q, want unchanged content", ctx.LLMResponse.Content)
	}
}

func TestHookMultipleRules(t *testing.T) {
	engine := NewEngine()
	engine.rules = []Rule{
		{
			Name:    "Rule One",
			Globs:   []string{"src/*.go"},
			Content: "First matching rule.",
		},
		{
			Name:    "Rule Two",
			Globs:   []string{"**/*.go"},
			Content: "Second matching rule.",
		},
	}

	handler := NewRulesHookHandler(engine)
	ctx := &hooks.HookContext{
		FilePath: "src/main.go",
		LLMResponse: &types.LLMResponse{
			Content: "Original file content",
		},
	}

	if err := handler.HandlePostFileRead(ctx); err != nil {
		t.Fatalf("HandlePostFileRead() error = %v", err)
	}

	want := "Original file content\n\n<!-- Rules -->\nFirst matching rule.\n\nSecond matching rule.\n<!-- /Rules -->"
	if ctx.LLMResponse.Content != want {
		t.Fatalf("LLMResponse.Content = %q, want %q", ctx.LLMResponse.Content, want)
	}
}

func TestHookNoRules(t *testing.T) {
	engine := NewEngine()

	handler := NewRulesHookHandler(engine)
	ctx := &hooks.HookContext{
		FilePath: "src/main.go",
		LLMResponse: &types.LLMResponse{
			Content: "Original file content",
		},
	}

	if err := handler.HandlePostFileRead(ctx); err != nil {
		t.Fatalf("HandlePostFileRead() error = %v", err)
	}

	if ctx.LLMResponse.Content != "Original file content" {
		t.Fatalf("LLMResponse.Content = %q, want unchanged content", ctx.LLMResponse.Content)
	}
}
