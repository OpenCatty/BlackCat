package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxHierarchyDepth is the maximum number of directory levels to scan
// for AGENTS.md files when walking from root toward the target directory.
const maxHierarchyDepth = 3

// HierarchyEntry represents an AGENTS.md file at a specific directory level.
type HierarchyEntry struct {
	Dir     string
	Content string
	Depth   int
}

// LoadHierarchicalAgents walks from targetDir toward rootDir collecting AGENTS.md files.
// Returns entries in root→leaf order (shallowest to deepest).
// Max depth: 3 levels. Returns error if rootDir is not an ancestor of targetDir.
func LoadHierarchicalAgents(rootDir, targetDir string) ([]HierarchyEntry, error) {
	// Resolve symlinks and clean both paths
	absRoot, err := resolveAndClean(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve rootDir: %w", err)
	}

	absTarget, err := resolveAndClean(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolve targetDir: %w", err)
	}

	// Use forward slashes for reliable string comparison (E9: Windows paths)
	normRoot := filepath.ToSlash(absRoot)
	normTarget := filepath.ToSlash(absTarget)

	// Verify rootDir is an ancestor of (or equal to) targetDir
	if !strings.HasPrefix(normTarget, normRoot) {
		return nil, fmt.Errorf("rootDir %q is not an ancestor of targetDir %q", rootDir, targetDir)
	}

	// Compute the relative path segments from root to target
	rel := strings.TrimPrefix(normTarget, normRoot)
	rel = strings.TrimPrefix(rel, "/")

	var segments []string
	if rel != "" {
		segments = strings.Split(rel, "/")
	}

	// Build the list of directories to check: root + each segment toward target
	dirs := make([]string, 0, len(segments)+1)
	dirs = append(dirs, absRoot)
	current := absRoot
	for _, seg := range segments {
		current = filepath.Join(current, seg)
		dirs = append(dirs, current)
	}

	// Cap at maxHierarchyDepth levels
	if len(dirs) > maxHierarchyDepth {
		dirs = dirs[:maxHierarchyDepth]
	}

	// Walk directories, collecting AGENTS.md entries with symlink protection
	visited := make(map[string]bool)
	var entries []HierarchyEntry

	for depth, dir := range dirs {
		// Resolve symlinks for each directory to detect loops
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			// Directory might not exist; skip silently
			continue
		}

		// Normalize for visited check
		normReal := filepath.ToSlash(realDir)
		if visited[normReal] {
			// Symlink loop detected; skip
			continue
		}
		visited[normReal] = true

		// Check for AGENTS.md in this directory
		agentsPath := filepath.Join(dir, "AGENTS.md")
		data, err := os.ReadFile(agentsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read AGENTS.md in %s: %w", dir, err)
		}

		content := strings.TrimSpace(string(data))
		if content != "" {
			entries = append(entries, HierarchyEntry{
				Dir:     dir,
				Content: content,
				Depth:   depth,
			})
		}
	}

	return entries, nil
}

// MergeHierarchyEntries combines multiple hierarchy entries into a single string
// using a separator between each entry.
func MergeHierarchyEntries(entries []HierarchyEntry) string {
	if len(entries) == 0 {
		return ""
	}
	if len(entries) == 1 {
		return entries[0].Content
	}

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = e.Content
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// resolveAndClean resolves symlinks and returns a cleaned absolute path.
func resolveAndClean(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}
