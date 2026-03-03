package workspace

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSingleLevel(t *testing.T) {
	root := t.TempDir()

	// Place AGENTS.md at root only
	writeFile(t, filepath.Join(root, "AGENTS.md"), "root-level agent instructions")

	entries, err := LoadHierarchicalAgents(root, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "root-level agent instructions" {
		t.Errorf("unexpected content: %q", entries[0].Content)
	}
	if entries[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", entries[0].Depth)
	}
}

func TestHierarchicalMerge(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "subproject")
	mkdirAll(t, child)

	writeFile(t, filepath.Join(root, "AGENTS.md"), "root instructions")
	writeFile(t, filepath.Join(child, "AGENTS.md"), "child instructions")

	entries, err := LoadHierarchicalAgents(root, child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Root comes first (shallowest)
	if entries[0].Content != "root instructions" {
		t.Errorf("first entry should be root, got %q", entries[0].Content)
	}
	if entries[0].Depth != 0 {
		t.Errorf("first entry depth should be 0, got %d", entries[0].Depth)
	}
	if entries[1].Content != "child instructions" {
		t.Errorf("second entry should be child, got %q", entries[1].Content)
	}
	if entries[1].Depth != 1 {
		t.Errorf("second entry depth should be 1, got %d", entries[1].Depth)
	}

	// Verify merge
	merged := MergeHierarchyEntries(entries)
	expected := "root instructions\n\n---\n\n" + "child instructions"
	if merged != expected {
		t.Errorf("merged mismatch:\ngot:  %q\nwant: %q", merged, expected)
	}
}

func TestMaxDepthCap(t *testing.T) {
	root := t.TempDir()

	// Create 5-level deep path: root/a/b/c/d/e
	levels := []string{"a", "b", "c", "d", "e"}
	current := root
	for _, lvl := range levels {
		current = filepath.Join(current, lvl)
		mkdirAll(t, current)
		writeFile(t, filepath.Join(current, "AGENTS.md"), "level-"+lvl)
	}
	writeFile(t, filepath.Join(root, "AGENTS.md"), "level-root")

	deepTarget := current // root/a/b/c/d/e

	entries, err := LoadHierarchicalAgents(root, deepTarget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should cap at 3: root, root/a, root/a/b
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (capped), got %d", len(entries))
	}

	expectedContents := []string{"level-root", "level-a", "level-b"}
	for i, exp := range expectedContents {
		if entries[i].Content != exp {
			t.Errorf("entry[%d]: expected %q, got %q", i, exp, entries[i].Content)
		}
	}
}

func TestSymlinkLoop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}

	root := t.TempDir()
	child := filepath.Join(root, "child")
	mkdirAll(t, child)

	writeFile(t, filepath.Join(root, "AGENTS.md"), "root-content")

	// Create a circular symlink: child/loop → root
	loopLink := filepath.Join(child, "loop")
	if err := os.Symlink(root, loopLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Target via the symlink loop: root/child/loop/child
	target := filepath.Join(loopLink, "child")

	entries, err := LoadHierarchicalAgents(root, target)
	if err != nil {
		t.Fatalf("unexpected error (should handle gracefully): %v", err)
	}

	// Should have at most root's AGENTS.md; no infinite loop
	if len(entries) > maxHierarchyDepth {
		t.Errorf("exceeded max depth, got %d entries", len(entries))
	}
}

func TestMissingAgentsMD(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "sub")
	grandchild := filepath.Join(child, "deep")
	mkdirAll(t, grandchild)

	// Only place AGENTS.md at grandchild — root and child have none
	writeFile(t, filepath.Join(grandchild, "AGENTS.md"), "deep-only")

	entries, err := LoadHierarchicalAgents(root, grandchild)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (skipping dirs without AGENTS.md), got %d", len(entries))
	}
	if entries[0].Content != "deep-only" {
		t.Errorf("unexpected content: %q", entries[0].Content)
	}
	if entries[0].Depth != 2 {
		t.Errorf("expected depth 2, got %d", entries[0].Depth)
	}
}

func TestRootNotAncestorError(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	_, err := LoadHierarchicalAgents(dir1, dir2)
	if err == nil {
		t.Fatal("expected error when rootDir is not ancestor of targetDir")
	}
}

func TestMergeHierarchyEntriesEmpty(t *testing.T) {
	result := MergeHierarchyEntries(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestMergeHierarchyEntriesSingle(t *testing.T) {
	entries := []HierarchyEntry{{Content: "only-one"}}
	result := MergeHierarchyEntries(entries)
	if result != "only-one" {
		t.Errorf("expected %q, got %q", "only-one", result)
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// mkdirAll is a test helper that creates directories.
func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
}
