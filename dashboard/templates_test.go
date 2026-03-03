package dashboard

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateRenderIndex(t *testing.T) {
	t.Setenv(devTemplateDirEnv, "")

	renderer, err := NewTemplateRenderer()
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	err = renderer.Render(recorder, "index", IndexView{SubsystemCount: 4, Uptime: "2h15m"})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(recorder.Body.String(), "System Overview") {
		t.Fatalf("rendered output missing System Overview: %q", recorder.Body.String())
	}
}

func TestTemplateRenderAgentCard(t *testing.T) {
	t.Setenv(devTemplateDirEnv, "")

	renderer, err := NewTemplateRenderer()
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	err = renderer.RenderPartial(recorder, "agent-card", AgentView{
		Name:        "alpha",
		State:       "running",
		CurrentTask: "sync",
		LastActive:  "just now",
	})
	if err != nil {
		t.Fatalf("RenderPartial failed: %v", err)
	}

	if !strings.Contains(recorder.Body.String(), "alpha") {
		t.Fatalf("rendered output missing agent name: %q", recorder.Body.String())
	}
}

func TestDevTemplateOverride(t *testing.T) {
	templateDir := filepath.Join(t.TempDir(), "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create temp templates directory: %v", err)
	}

	writeTemplate(t, filepath.Join(templateDir, "layout.html"), `<!DOCTYPE html>
<html>
<body>
  {{template "content" .}}
</body>
</html>`)

	writeTemplate(t, filepath.Join(templateDir, "index.html"), `{{define "content"}}
<h1>Custom Dashboard</h1>
{{end}}`)

	t.Setenv(devTemplateDirEnv, filepath.Dir(templateDir))

	renderer, err := NewTemplateRenderer()
	if err != nil {
		t.Fatalf("NewTemplateRenderer failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	err = renderer.Render(recorder, "index", IndexView{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(recorder.Body.String(), "Custom Dashboard") {
		t.Fatalf("dev override template was not rendered: %q", recorder.Body.String())
	}
}

func writeTemplate(t *testing.T, filePath string, content string) {
	t.Helper()

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write template %s: %v", filePath, err)
	}
}
