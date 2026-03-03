package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/types"
)

func TestInitDeepDepthCap(t *testing.T) {
	root := t.TempDir()

	levels := make([]string, 0, 5)
	current := root
	for i := 1; i <= 5; i++ {
		current = filepath.Join(current, fmt.Sprintf("level%d", i))
		if err := os.MkdirAll(current, 0o755); err != nil {
			t.Fatalf("mkdir level %d: %v", i, err)
		}
		writeTestFile(t, filepath.Join(current, fmt.Sprintf("file%d.go", i)), "package test")
		levels = append(levels, current)
	}

	mock := &mockInitDeepLLM{response: "generated depth content"}
	withMockInitDeepDeps(t, mock)

	if err := executeInitDeepForTest(root); err != nil {
		t.Fatalf("execute init-deep: %v", err)
	}

	for i, dir := range levels {
		agentsPath := filepath.Join(dir, "AGENTS.md")
		exists := fileExists(agentsPath)
		if i < 3 && !exists {
			t.Fatalf("expected AGENTS.md at level %d (%s)", i+1, agentsPath)
		}
		if i >= 3 && exists {
			t.Fatalf("did not expect AGENTS.md at level %d (%s)", i+1, agentsPath)
		}
	}

	if mock.calls != 3 {
		t.Fatalf("expected 3 LLM calls for levels 1-3, got %d", mock.calls)
	}
}

func TestInitDeepDryRun(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, filepath.Join(root, "pkg", "main.go"), "package pkg")
	writeTestFile(t, filepath.Join(root, "docs", "readme.md"), "# docs")

	mock := &mockInitDeepLLM{response: "generated dry run content"}
	withMockInitDeepDeps(t, mock)

	if err := executeInitDeepForTest("--dry-run", root); err != nil {
		t.Fatalf("execute init-deep --dry-run: %v", err)
	}

	agentsFiles := findAgentsFiles(t, root)
	if len(agentsFiles) != 0 {
		t.Fatalf("expected no AGENTS.md files in dry-run, found %v", agentsFiles)
	}
}

func TestInitDeepSkipExisting(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg")
	writeTestFile(t, filepath.Join(dir, "main.go"), "package pkg")

	agentsPath := filepath.Join(dir, "AGENTS.md")
	original := "existing agents content\n"
	writeTestFile(t, agentsPath, original)

	mockNoForce := &mockInitDeepLLM{response: "new content without force"}
	withMockInitDeepDeps(t, mockNoForce)

	if err := executeInitDeepForTest(root); err != nil {
		t.Fatalf("execute init-deep without force: %v", err)
	}

	if mockNoForce.calls != 0 {
		t.Fatalf("expected no LLM calls when AGENTS.md exists without force, got %d", mockNoForce.calls)
	}

	contentAfterNoForce := readTestFile(t, agentsPath)
	if contentAfterNoForce != original {
		t.Fatalf("expected AGENTS.md to remain unchanged without force, got %q", contentAfterNoForce)
	}

	mockForce := &mockInitDeepLLM{response: "new forced content"}
	withMockInitDeepDeps(t, mockForce)

	if err := executeInitDeepForTest("--force", root); err != nil {
		t.Fatalf("execute init-deep --force: %v", err)
	}

	if mockForce.calls != 1 {
		t.Fatalf("expected 1 LLM call with force overwrite, got %d", mockForce.calls)
	}

	contentAfterForce := readTestFile(t, agentsPath)
	if strings.TrimSpace(contentAfterForce) != "new forced content" {
		t.Fatalf("expected AGENTS.md to be overwritten with forced content, got %q", contentAfterForce)
	}
}

func TestInitDeepSkipHidden(t *testing.T) {
	root := t.TempDir()

	hiddenDir := filepath.Join(root, ".hidden")
	visibleDir := filepath.Join(root, "visible")
	writeTestFile(t, filepath.Join(hiddenDir, "hidden.go"), "package hidden")
	writeTestFile(t, filepath.Join(visibleDir, "visible.go"), "package visible")

	mock := &mockInitDeepLLM{response: "visible content"}
	withMockInitDeepDeps(t, mock)

	if err := executeInitDeepForTest(root); err != nil {
		t.Fatalf("execute init-deep: %v", err)
	}

	if fileExists(filepath.Join(hiddenDir, "AGENTS.md")) {
		t.Fatalf("did not expect AGENTS.md in hidden directory %s", hiddenDir)
	}
	if !fileExists(filepath.Join(visibleDir, "AGENTS.md")) {
		t.Fatalf("expected AGENTS.md in visible directory %s", visibleDir)
	}
	if mock.calls != 1 {
		t.Fatalf("expected exactly 1 LLM call for visible directory, got %d", mock.calls)
	}
}

func TestInitDeepEmptyDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "empty", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir empty tree: %v", err)
	}

	mock := &mockInitDeepLLM{response: "should not be used"}
	withMockInitDeepDeps(t, mock)

	if err := executeInitDeepForTest(root); err != nil {
		t.Fatalf("execute init-deep on empty tree: %v", err)
	}

	if mock.calls != 0 {
		t.Fatalf("expected no LLM calls for empty directories, got %d", mock.calls)
	}

	agentsFiles := findAgentsFiles(t, root)
	if len(agentsFiles) != 0 {
		t.Fatalf("expected no AGENTS.md files for empty directories, found %v", agentsFiles)
	}
}

type mockInitDeepLLM struct {
	response string
	calls    int
	prompts  []string
}

func (m *mockInitDeepLLM) Chat(_ context.Context, messages []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	m.calls++
	if len(messages) > 0 {
		m.prompts = append(m.prompts, messages[len(messages)-1].Content)
	}
	return &types.LLMResponse{Content: m.response, Model: "mock"}, nil
}

func (m *mockInitDeepLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk)
	close(ch)
	return ch, nil
}

func withMockInitDeepDeps(t *testing.T, mock types.LLMClient) {
	t.Helper()

	originalClientFactory := initDeepNewLLMClient
	originalSleep := initDeepSleep

	initDeepNewLLMClient = func() (types.LLMClient, error) {
		return mock, nil
	}
	initDeepSleep = func(time.Duration) {}

	t.Cleanup(func() {
		initDeepNewLLMClient = originalClientFactory
		initDeepSleep = originalSleep
	})
}

func executeInitDeepForTest(args ...string) error {
	cmd := &cobra.Command{
		Use:   "init-deep [directory]",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInitDeep,
		Short: "test wrapper",
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().Int("max-depth", initDeepDefaultMaxDepth, "")

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	return cmd.Execute()
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func findAgentsFiles(t *testing.T, root string) []string {
	t.Helper()

	found := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "AGENTS.md" {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	return found
}
