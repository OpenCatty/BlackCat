package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileLoad(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "builder.md")
	content := "# Builder\n\nYou enforce build and release discipline.\n\ntags: ci, release\nAlways run checks before shipping.\n"

	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	loader := &ProfileLoader{}
	profiles, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("len(profiles) = %d, want 1", len(profiles))
	}

	profile, ok := profiles["builder"]
	if !ok {
		t.Fatalf("expected key %q not found", "builder")
	}

	if profile.Name != "Builder" {
		t.Fatalf("Name = %q, want %q", profile.Name, "Builder")
	}

	if profile.FilePath != profilePath {
		t.Fatalf("FilePath = %q, want %q", profile.FilePath, profilePath)
	}

	if len(profile.Tags) != 2 || profile.Tags[0] != "ci" || profile.Tags[1] != "release" {
		t.Fatalf("Tags = %#v, want [ci release]", profile.Tags)
	}

	wantOverlay := "You enforce build and release discipline.\n\nAlways run checks before shipping."
	if profile.SystemPromptOverlay != wantOverlay {
		t.Fatalf("SystemPromptOverlay = %q, want %q", profile.SystemPromptOverlay, wantOverlay)
	}
}

func TestProfileApplyOverlay(t *testing.T) {
	base := "Base system prompt"
	profile := &Profile{SystemPromptOverlay: "Profile overlay"}

	got := ApplyProfile(base, profile)
	want := "Profile overlay\n\n---\n\nBase system prompt"

	if got != want {
		t.Fatalf("ApplyProfile() = %q, want %q", got, want)
	}

	if !strings.HasPrefix(got, "Profile overlay") {
		t.Fatalf("ApplyProfile() did not prepend overlay: %q", got)
	}
}

func TestProfileMissingDir(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "missing-profiles")

	loader := &ProfileLoader{}
	profiles, err := loader.Load(missingDir)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if len(profiles) != 0 {
		t.Fatalf("len(profiles) = %d, want 0", len(profiles))
	}
}

func TestProfileEmptyDir(t *testing.T) {
	dir := t.TempDir()
	nonMarkdown := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(nonMarkdown, []byte("ignore me"), 0o644); err != nil {
		t.Fatalf("write non-markdown file: %v", err)
	}

	loader := &ProfileLoader{}
	profiles, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(profiles) != 0 {
		t.Fatalf("len(profiles) = %d, want 0", len(profiles))
	}
}
