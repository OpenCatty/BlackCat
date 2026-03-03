package workspace

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed AGENTS.md SOUL.md MEMORY.md skills/*.md
var templateFS embed.FS

// InitWorkspace initializes a workspace directory with bootstrap templates.
// It walks through the embedded FS and copies each file to targetDir,
// skipping any files that already exist (no overwrite).
func InitWorkspace(targetDir string) error {
	return fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == "." {
			return nil
		}

		// Build the target file path
		targetPath := filepath.Join(targetDir, path)

		if d.IsDir() {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		} else {
			// Check if file already exists (skip if it does)
			if _, err := os.Stat(targetPath); err == nil {
				// File exists, skip it
				return nil
			} else if !os.IsNotExist(err) {
				// Error other than not found
				return fmt.Errorf("failed to check file %s: %w", targetPath, err)
			}

			// File doesn't exist, create parent directories
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories for %s: %w", targetPath, err)
			}

			// Open the source file from embedded FS
			src, err := templateFS.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open embedded file %s: %w", path, err)
			}
			defer src.Close()

			// Create the target file
			dst, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			defer dst.Close()

			// Copy the contents
			if _, err := io.Copy(dst, src); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", path, err)
			}
		}

		return nil
	})
}

// ListTemplates returns a list of template file names from the embedded FS.
func ListTemplates() []string {
	var templates []string

	fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." || d.IsDir() {
			return nil
		}
		templates = append(templates, path)
		return nil
	})

	return templates
}
