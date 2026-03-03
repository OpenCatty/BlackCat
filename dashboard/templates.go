package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const devTemplateDirEnv = "BLACKCAT_DEV_TEMPLATE_DIR"

//go:embed templates
var templateFS embed.FS

type AgentView struct {
	Name        string
	State       string
	CurrentTask string
	LastActive  string
}

type TaskView struct {
	ID       string
	Name     string
	Status   string
	Duration string
	LastRun  string
}

type IndexView struct {
	SubsystemCount int
	Uptime         string
}

type LoginView struct {
	Error string
	Next  string
}

type AgentsView struct {
	Agents []AgentView
}

type TasksView struct {
	Tasks    []TaskView
	NextPage int
}

type EventView struct {
	Name        string
	Status      string // "ok", "failed", "running", "scheduled"
	TimeStr     string // formatted time string e.g. "14:30"
	IsProjected bool
	IsHighFreq  bool
}

type DayView struct {
	DayNum         int    // 1-31
	DateStr        string // "2006-01-02"
	IsCurrentMonth bool
	IsToday        bool
	Events         []EventView
	HeartbeatOK    *bool // nil = no data, true = healthy, false = unhealthy
}

type WeekView struct {
	Days [7]DayView
}

type ScheduleView struct {
	Year      int
	Month     int    // 1-12
	MonthName string // "January"
	Weeks     []WeekView
	PrevYear  int
	PrevMonth int
	NextYear  int
	NextMonth int
}

type TemplateRenderer struct {
	templates  *template.Template
	fsys       fs.FS
	root       string
	layoutName string
	devMode    bool
}

func NewTemplateRenderer() (*TemplateRenderer, error) {
	templateSource, root, devMode, err := resolveTemplateFS()
	if err != nil {
		return nil, err
	}

	baseTemplates, layoutName, pageTemplates, err := parseBaseTemplates(templateSource, root)
	if err != nil {
		return nil, err
	}

	for _, pageTemplate := range pageTemplates {
		instance, err := baseTemplates.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone template set for %q: %w", pageTemplate, err)
		}

		if _, err := instance.ParseFS(templateSource, pageTemplate); err != nil {
			return nil, fmt.Errorf("parse page template %q: %w", pageTemplate, err)
		}
	}

	return &TemplateRenderer{
		templates:  baseTemplates,
		fsys:       templateSource,
		root:       root,
		layoutName: layoutName,
		devMode:    devMode,
	}, nil
}

func (r *TemplateRenderer) Render(w http.ResponseWriter, name string, data interface{}) error {
	pageTemplate, err := r.resolvePageTemplatePath(name)
	if err != nil {
		return err
	}

	baseTemplates, layoutName, err := r.currentBaseTemplates()
	if err != nil {
		return err
	}

	instance, err := baseTemplates.Clone()
	if err != nil {
		return fmt.Errorf("clone template set: %w", err)
	}

	if _, err := instance.ParseFS(r.fsys, pageTemplate); err != nil {
		return fmt.Errorf("parse page template %q: %w", pageTemplate, err)
	}

	if err := instance.ExecuteTemplate(w, layoutName, data); err != nil {
		return fmt.Errorf("execute template %q: %w", name, err)
	}

	return nil
}

func (r *TemplateRenderer) RenderPartial(w http.ResponseWriter, name string, data interface{}) error {
	partialName, err := normalizePartialTemplateName(name)
	if err != nil {
		return err
	}

	baseTemplates, _, err := r.currentBaseTemplates()
	if err != nil {
		return err
	}

	if err := baseTemplates.ExecuteTemplate(w, partialName, data); err != nil {
		return fmt.Errorf("execute partial template %q: %w", partialName, err)
	}

	return nil
}

func (r *TemplateRenderer) currentBaseTemplates() (*template.Template, string, error) {
	if !r.devMode {
		return r.templates, r.layoutName, nil
	}

	baseTemplates, layoutName, _, err := parseBaseTemplates(r.fsys, r.root)
	if err != nil {
		return nil, "", err
	}

	return baseTemplates, layoutName, nil
}

func (r *TemplateRenderer) resolvePageTemplatePath(name string) (string, error) {
	normalizedName := normalizeTemplatePath(name)
	if normalizedName == "" {
		return "", fmt.Errorf("template name is required")
	}

	if strings.HasPrefix(normalizedName, "templates/") {
		normalizedName = strings.TrimPrefix(normalizedName, "templates/")
	}

	if strings.HasPrefix(normalizedName, "partials/") {
		return "", fmt.Errorf("template %q is a partial, not a page", name)
	}

	if !strings.HasSuffix(normalizedName, ".html") {
		normalizedName += ".html"
	}

	if strings.HasPrefix(normalizedName, "../") || normalizedName == ".." {
		return "", fmt.Errorf("invalid template path %q", name)
	}

	if r.root == "." {
		return normalizedName, nil
	}

	return path.Join(r.root, normalizedName), nil
}

func resolveTemplateFS() (fs.FS, string, bool, error) {
	devTemplateDir := strings.TrimSpace(os.Getenv(devTemplateDirEnv))
	if devTemplateDir == "" {
		return templateFS, "templates", false, nil
	}

	info, err := os.Stat(devTemplateDir)
	if err != nil {
		return nil, "", true, fmt.Errorf("stat %s=%q: %w", devTemplateDirEnv, devTemplateDir, err)
	}
	if !info.IsDir() {
		return nil, "", true, fmt.Errorf("%s=%q is not a directory", devTemplateDirEnv, devTemplateDir)
	}

	root := "."
	if dirExists(filepath.Join(devTemplateDir, "templates")) {
		root = "templates"
	}

	return os.DirFS(devTemplateDir), root, true, nil
}

func dirExists(directory string) bool {
	info, err := os.Stat(directory)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func parseBaseTemplates(templateSource fs.FS, root string) (*template.Template, string, []string, error) {
	files, err := listTemplateFiles(templateSource, root)
	if err != nil {
		return nil, "", nil, fmt.Errorf("list templates from %q: %w", root, err)
	}

	if len(files) == 0 {
		return nil, "", nil, fmt.Errorf("no template files found in %q", root)
	}

	layoutTemplate, partialTemplates, pageTemplates := classifyTemplateFiles(files)
	if layoutTemplate == "" {
		return nil, "", nil, fmt.Errorf("layout template not found")
	}

	baseFiles := []string{layoutTemplate}
	baseFiles = append(baseFiles, partialTemplates...)

	funcMap := template.FuncMap{
		"weekdayOffset": func(t time.Time) int { return int(t.Weekday()) + 1 },
		"formatDate":    func(t time.Time, layout string) string { return t.Format(layout) },
		"isZero":        func(t time.Time) bool { return t.IsZero() },
		"add":           func(a, b int) int { return a + b },
		"sub":           func(a, b int) int { return a - b },
		"monthInt":      func(m time.Month) int { return int(m) },
	}

	baseTemplates, err := template.New(path.Base(layoutTemplate)).Funcs(funcMap).ParseFS(templateSource, baseFiles...)
	if err != nil {
		return nil, "", nil, fmt.Errorf("parse base templates: %w", err)
	}

	return baseTemplates, path.Base(layoutTemplate), pageTemplates, nil
}

func listTemplateFiles(templateSource fs.FS, root string) ([]string, error) {
	var files []string

	err := fs.WalkDir(templateSource, root, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		if path.Ext(file) != ".html" {
			return nil
		}

		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func classifyTemplateFiles(files []string) (string, []string, []string) {
	var layoutTemplate string
	var partialTemplates []string
	var pageTemplates []string

	for _, file := range files {
		switch {
		case path.Base(file) == "layout.html":
			layoutTemplate = file
		case strings.HasPrefix(file, "partials/") || strings.Contains(file, "/partials/"):
			partialTemplates = append(partialTemplates, file)
		default:
			pageTemplates = append(pageTemplates, file)
		}
	}

	sort.Strings(partialTemplates)
	sort.Strings(pageTemplates)

	return layoutTemplate, partialTemplates, pageTemplates
}

func normalizeTemplatePath(name string) string {
	normalized := strings.TrimSpace(name)
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = path.Clean(normalized)
	normalized = strings.TrimPrefix(normalized, "./")

	if normalized == "." {
		return ""
	}

	return normalized
}

func normalizePartialTemplateName(name string) (string, error) {
	normalizedName := normalizeTemplatePath(name)
	if normalizedName == "" {
		return "", fmt.Errorf("partial template name is required")
	}

	normalizedName = strings.TrimPrefix(normalizedName, "templates/partials/")
	normalizedName = strings.TrimPrefix(normalizedName, "partials/")
	normalizedName = path.Base(normalizedName)
	normalizedName = strings.TrimSuffix(normalizedName, path.Ext(normalizedName))

	if normalizedName == "" || normalizedName == "." {
		return "", fmt.Errorf("invalid partial template name %q", name)
	}

	return normalizedName, nil
}
