package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: create a temp dir with rule files
func setupTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to write test file %s: %v", name, err)
		}
	}
	return dir
}

func TestLoadRules(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"go-errors.md": `---
name: "Go Error Handling"
globs:
  - "**/*.go"
---
Always return errors with context using fmt.Errorf("doing X: %w", err).
`,
		"js-lint.md": `---
name: "JS Lint Rules"
globs:
  - "src/*.js"
  - "lib/*.js"
---
Use const by default, let only when reassignment is needed.
`,
	})

	engine := NewEngine()
	err := engine.LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules() error: %v", err)
	}

	rules := engine.Rules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	// Find rules by name (order depends on os.ReadDir)
	ruleMap := make(map[string]Rule)
	for _, r := range rules {
		ruleMap[r.Name] = r
	}

	goRule, ok := ruleMap["Go Error Handling"]
	if !ok {
		t.Fatal("expected 'Go Error Handling' rule")
	}
	if len(goRule.Globs) != 1 {
		t.Fatalf("expected 1 glob, got %d", len(goRule.Globs))
	}
	if goRule.Globs[0] != "**/*.go" {
		t.Errorf("expected glob '**/*.go', got %q", goRule.Globs[0])
	}
	if goRule.Content != `Always return errors with context using fmt.Errorf("doing X: %w", err).` {
		t.Errorf("unexpected content: %q", goRule.Content)
	}

	jsRule, ok := ruleMap["JS Lint Rules"]
	if !ok {
		t.Fatal("expected 'JS Lint Rules' rule")
	}
	if len(jsRule.Globs) != 2 {
		t.Fatalf("expected 2 globs, got %d", len(jsRule.Globs))
	}
}

func TestLoadRulesNonexistentDir(t *testing.T) {
	engine := NewEngine()
	err := engine.LoadRules("/nonexistent/path/to/rules")
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got: %v", err)
	}
	if len(engine.Rules()) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(engine.Rules()))
	}
}

func TestLoadRulesNameFromFilename(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"my-rule.md": `---
globs:
  - "*.txt"
---
Body content here.
`,
	})

	engine := NewEngine()
	err := engine.LoadRules(dir)
	if err != nil {
		t.Fatalf("LoadRules() error: %v", err)
	}

	rules := engine.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "my-rule" {
		t.Errorf("expected name 'my-rule' from filename, got %q", rules[0].Name)
	}
}

func TestMatchGlob(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"go-rule.md": `---
name: "Go Rule"
globs:
  - "src/*.go"
---
Rule body.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	// Should match
	matches := engine.Match("src/main.go")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for src/main.go, got %d", len(matches))
	}
	if matches[0].Name != "Go Rule" {
		t.Errorf("expected 'Go Rule', got %q", matches[0].Name)
	}

	// Should NOT match (deeper path)
	matches = engine.Match("src/sub/deep.go")
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for src/sub/deep.go, got %d", len(matches))
	}
}

func TestMatchDoublestarGlob(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"go-all.md": `---
name: "All Go"
globs:
  - "**/*.go"
---
Applies to all Go files.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path    string
		matches bool
	}{
		{"main.go", true},
		{"src/main.go", true},
		{"src/sub/deep.go", true},
		{"main.js", false},
		{"src/main.js", false},
	}

	for _, tc := range tests {
		matches := engine.Match(tc.path)
		got := len(matches) > 0
		if got != tc.matches {
			t.Errorf("Match(%q): got matched=%v, want %v", tc.path, got, tc.matches)
		}
	}
}

func TestMatchMultiGlob(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"multi.md": `---
name: "Multi Glob Rule"
globs:
  - "src/*.go"
  - "lib/*.go"
---
Multi-glob body.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	// Should match first glob
	matches := engine.Match("src/main.go")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for src/main.go, got %d", len(matches))
	}

	// Should match second glob
	matches = engine.Match("lib/util.go")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for lib/util.go, got %d", len(matches))
	}

	// Should NOT match
	matches = engine.Match("test/main.go")
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for test/main.go, got %d", len(matches))
	}
}

func TestMatchNoMatch(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"go-only.md": `---
name: "Go Only"
globs:
  - "*.go"
---
Go only rule.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	matches := engine.Match("readme.md")
	if len(matches) != 0 {
		t.Fatalf("expected empty slice, got %d matches", len(matches))
	}

	matches = engine.Match("src/main.js")
	if len(matches) != 0 {
		t.Fatalf("expected empty slice, got %d matches", len(matches))
	}
}

func TestWindowsPathNormalization(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"go-rule.md": `---
name: "Go Rule"
globs:
  - "src/*.go"
---
Rule body.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	// Windows-style backslash path should match forward-slash glob
	matches := engine.Match("src\\main.go")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for src\\main.go (Windows path), got %d", len(matches))
	}
}

func TestNoFrontmatterSkipped(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"no-fm.md": `# Just a markdown file
No frontmatter here.
`,
		"valid.md": `---
name: "Valid Rule"
globs:
  - "*.go"
---
Valid rule body.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	rules := engine.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (no-frontmatter skipped), got %d", len(rules))
	}
	if rules[0].Name != "Valid Rule" {
		t.Errorf("expected 'Valid Rule', got %q", rules[0].Name)
	}
}

func TestNoGlobsSkipped(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"no-globs.md": `---
name: "No Globs Rule"
---
This rule has frontmatter but no globs.
`,
		"valid.md": `---
name: "Valid Rule"
globs:
  - "*.go"
---
Valid rule body.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	rules := engine.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (no-globs skipped), got %d", len(rules))
	}
	if rules[0].Name != "Valid Rule" {
		t.Errorf("expected 'Valid Rule', got %q", rules[0].Name)
	}
}

func TestMultipleRulesMatch(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"rule1.md": `---
name: "Rule One"
globs:
  - "**/*.go"
---
First rule.
`,
		"rule2.md": `---
name: "Rule Two"
globs:
  - "src/*.go"
---
Second rule.
`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	// Both rules should match src/main.go
	matches := engine.Match("src/main.go")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for src/main.go, got %d", len(matches))
	}

	// Only rule1 should match deep paths
	matches = engine.Match("src/sub/deep.go")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for src/sub/deep.go, got %d", len(matches))
	}
	if matches[0].Name != "Rule One" {
		t.Errorf("expected 'Rule One', got %q", matches[0].Name)
	}
}

func TestNonMdFilesIgnored(t *testing.T) {
	dir := setupTestDir(t, map[string]string{
		"rule.md": `---
name: "Valid Rule"
globs:
  - "*.go"
---
Valid.
`,
		"notes.txt":   `This is not a rule file.`,
		"config.yaml": `key: value`,
	})

	engine := NewEngine()
	if err := engine.LoadRules(dir); err != nil {
		t.Fatal(err)
	}

	rules := engine.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (non-.md ignored), got %d", len(rules))
	}
}
