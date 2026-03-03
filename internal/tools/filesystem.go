package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/startower-observability/blackcat/internal/types"
)

const (
	fsToolName        = "filesystem"
	fsToolDescription = "Read, write, and list files within the workspace"
)

var fsToolParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"action": {
			"type": "string",
			"enum": ["read", "write", "list", "search"],
			"description": "File operation"
		},
		"path": {
			"type": "string",
			"description": "File or directory path"
		},
		"content": {
			"type": "string",
			"description": "Content to write (for write action)"
		},
		"pattern": {
			"type": "string",
			"description": "Glob pattern (for search action)"
		}
	},
	"required": ["action", "path"]
}`)

// FilesystemTool provides sandboxed file operations within a workspace root.
type FilesystemTool struct {
	workspaceRoot string
}

// NewFilesystemTool creates a FilesystemTool rooted at the given directory.
func NewFilesystemTool(workspaceRoot string) *FilesystemTool {
	abs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		abs = workspaceRoot
	}
	return &FilesystemTool{workspaceRoot: abs}
}

func (t *FilesystemTool) Name() string                { return fsToolName }
func (t *FilesystemTool) Description() string         { return fsToolDescription }
func (t *FilesystemTool) Parameters() json.RawMessage { return fsToolParameters }

// safePath resolves a path and ensures it stays within the workspace root.
func (t *FilesystemTool) safePath(path string) (string, error) {
	// Resolve path relative to workspace root.
	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(t.workspaceRoot, path))
	}

	// Use filepath.Rel to check containment.
	rel, err := filepath.Rel(t.workspaceRoot, abs)
	if err != nil {
		return "", types.ErrPathTraversal
	}
	if strings.HasPrefix(rel, "..") {
		return "", types.ErrPathTraversal
	}
	return abs, nil
}

// Execute runs a filesystem operation.
func (t *FilesystemTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action  string `json:"action"`
		Path    string `json:"path"`
		Content string `json:"content"`
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("filesystem: invalid arguments: %w", err)
	}

	switch params.Action {
	case "read":
		return t.read(params.Path)
	case "write":
		return t.write(params.Path, params.Content)
	case "list":
		return t.list(params.Path)
	case "search":
		return t.search(params.Pattern)
	default:
		return "", fmt.Errorf("filesystem: unknown action %q", params.Action)
	}
}

func (t *FilesystemTool) read(path string) (string, error) {
	safe, err := t.safePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(safe)
	if err != nil {
		return "", fmt.Errorf("filesystem: read: %w", err)
	}
	return string(data), nil
}

func (t *FilesystemTool) write(path, content string) (string, error) {
	safe, err := t.safePath(path)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(safe)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("filesystem: mkdir: %w", err)
	}
	if err := os.WriteFile(safe, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("filesystem: write: %w", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path), nil
}

func (t *FilesystemTool) list(path string) (string, error) {
	safe, err := t.safePath(path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(safe)
	if err != nil {
		return "", fmt.Errorf("filesystem: list: %w", err)
	}
	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			b.WriteString(e.Name() + "/\n")
		} else {
			b.WriteString(e.Name() + "\n")
		}
	}
	return b.String(), nil
}

func (t *FilesystemTool) search(pattern string) (string, error) {
	// Resolve pattern relative to workspace root.
	fullPattern := filepath.Join(t.workspaceRoot, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return "", fmt.Errorf("filesystem: search: %w", err)
	}
	// Return paths relative to workspace root.
	var b strings.Builder
	for _, m := range matches {
		rel, _ := filepath.Rel(t.workspaceRoot, m)
		b.WriteString(rel + "\n")
	}
	return b.String(), nil
}
