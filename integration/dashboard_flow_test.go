package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/daemon"
	"github.com/startower-observability/blackcat/internal/dashboard"
	"github.com/startower-observability/blackcat/internal/scheduler"
)

type dashboardMockSubsystemManager struct {
	health []daemon.SubsystemHealth
}

func (m dashboardMockSubsystemManager) Healthz() []daemon.SubsystemHealth {
	return append([]daemon.SubsystemHealth(nil), m.health...)
}

type dashboardMockTaskLister struct {
	tasks []string
}

func (m dashboardMockTaskLister) ListTasks() []string {
	return append([]string(nil), m.tasks...)
}

type dashboardMockHeartbeatStore struct {
	results []scheduler.HeartbeatResult
}

func (m dashboardMockHeartbeatStore) Latest(n int) []scheduler.HeartbeatResult {
	if n <= 0 {
		return []scheduler.HeartbeatResult{}
	}

	if n > len(m.results) {
		n = len(m.results)
	}

	return append([]scheduler.HeartbeatResult(nil), m.results[:n]...)
}

func TestDashboardFullAuth(t *testing.T) {
	t.Parallel()

	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, dashboard.DashboardDeps{})
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/api/status", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status %d, got %d", http.StatusUnauthorized, res.StatusCode)
	}
	if !strings.Contains(mustReadBody(t, res), `{"error":"unauthorized"}`) {
		t.Fatalf("expected unauthorized body, got %q", mustReadBody(t, res))
	}

	res = mustRequest(t, ts.URL+"/dashboard/api/status", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected authorized status %d, got %d", http.StatusOK, res.StatusCode)
	}
	if contentType := res.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json content-type, got %q", contentType)
	}
	if !strings.Contains(mustReadBody(t, res), `"healthy"`) {
		t.Fatalf("expected status payload to include healthy field, got %q", mustReadBody(t, res))
	}
}

func TestDashboardAgentListing(t *testing.T) {
	t.Parallel()

	deps := dashboard.DashboardDeps{
		SubsystemManager: dashboardMockSubsystemManager{health: []daemon.SubsystemHealth{
			{Name: "scheduler", Status: "running", Message: "ok"},
			{Name: "orchestrator", Status: "degraded", Message: "slow responses"},
		}},
	}

	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/api/agents", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var agents []map[string]string
	if err := json.NewDecoder(res.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode agents response: %v", err)
	}

	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0]["name"] != "scheduler" || agents[1]["name"] != "orchestrator" {
		t.Fatalf("unexpected agent names: %#v", agents)
	}
}

func TestDashboardTaskHistory(t *testing.T) {
	t.Parallel()

	deps := dashboard.DashboardDeps{
		TaskLister: dashboardMockTaskLister{tasks: []string{"heartbeat", "cleanup", "notify"}},
	}

	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/api/tasks?page=1&limit=2", "secret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var payload struct {
		Tasks []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"tasks"`
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode tasks response: %v", err)
	}

	if payload.Total != 3 || payload.Page != 1 || payload.Limit != 2 {
		t.Fatalf("unexpected pagination payload: %#v", payload)
	}
	if len(payload.Tasks) != 2 || payload.Tasks[0].Name != "heartbeat" || payload.Tasks[1].Name != "cleanup" {
		t.Fatalf("unexpected tasks payload: %#v", payload.Tasks)
	}
}

func TestDashboardSSELifecycle(t *testing.T) {
	t.Parallel()

	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, dashboard.DashboardDeps{})
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	res := mustRequest(t, ts.URL+"/dashboard/events", "secret")
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
	if contentType := res.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream content-type, got %q", contentType)
	}

	// Read first chunk to verify keepalive, then close the connection
	buf := make([]byte, 1024)
	n, err := res.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("failed to read response body: %v", err)
	}
	body := string(buf[:n])
	if !strings.Contains(body, ": keepalive") {
		t.Fatalf("expected keepalive body, got %q", body)
	}
}
func TestDashboardDisabled(t *testing.T) {
	t.Parallel()

	server := dashboard.NewServer(config.DashboardConfig{Enabled: false, Token: "secret"}, dashboard.DashboardDeps{})
	if server != nil {
		t.Fatal("expected nil server when dashboard is disabled")
	}
}

func TestDashboardTemplateDevOverride(t *testing.T) {
	templateRoot := filepath.Join(t.TempDir(), "templates")
	mustWriteFile(t, filepath.Join(templateRoot, "layout.html"), `<!doctype html><html><body>{{template "agent-card" .}}</body></html>`)
	mustWriteFile(t, filepath.Join(templateRoot, "agents.html"), `{{define "content"}}<p>unused</p>{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "index.html"), `{{define "content"}}<h1>index</h1>{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "tasks.html"), `{{define "content"}}<p>tasks</p>{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "partials", "agent-card.html"), `{{define "agent-card"}}DEV-OVERRIDE-MARKER{{.Name}}{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "partials", "task-row.html"), `{{define "task-row"}}<tr><td>{{.Name}}</td></tr>{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "schedule.html"), `{{define "content"}}{{template "schedule-content" .}}{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "partials", "schedule-content.html"), `{{define "schedule-content"}}<div>SCHEDULE-DEV-OVERRIDE</div>{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "partials", "schedule-calendar.html"), `{{define "schedule-calendar"}}{{end}}`)
	mustWriteFile(t, filepath.Join(templateRoot, "partials", "schedule-event.html"), `{{define "schedule-event"}}{{end}}`)

	t.Setenv("BLACKCAT_DEV_TEMPLATE_DIR", filepath.Dir(templateRoot))

	deps := dashboard.DashboardDeps{
		SubsystemManager: dashboardMockSubsystemManager{health: []daemon.SubsystemHealth{
			{Name: "agent-x", Status: "running", Message: "ok"},
		}},
	}

	server := dashboard.NewServer(config.DashboardConfig{Enabled: true, Token: "secret"}, deps)
	ts := newDashboardFlowTestServer(t, server)
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/dashboard/api/agents", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Accept", "text/html")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
	body := mustReadBody(t, res)
	if contentType := res.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected application/json content-type, got %q", contentType)
	}
	if !strings.Contains(body, "\"agent-x\"") {
		t.Fatalf("expected agent payload in body, got %q", body)
	}
}

func newDashboardFlowTestServer(t *testing.T, server *dashboard.Server) *httptest.Server {
	t.Helper()
	if server == nil {
		t.Fatal("dashboard server is nil")
	}

	handler := extractDashboardFlowRouterHandler(t, server)
	return httptest.NewServer(handler)
}

func extractDashboardFlowRouterHandler(t *testing.T, server *dashboard.Server) http.Handler {
	t.Helper()

	value := reflect.ValueOf(server)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		t.Fatalf("invalid server value: %T", server)
	}

	elem := value.Elem()
	routerField := elem.FieldByName("router")
	if !routerField.IsValid() || routerField.IsNil() {
		t.Fatal("dashboard server router is not initialized")
	}

	unsafeRouter := reflect.NewAt(routerField.Type(), unsafe.Pointer(routerField.UnsafeAddr())).Elem().Interface()
	handler, ok := unsafeRouter.(http.Handler)
	if !ok {
		t.Fatalf("router does not implement http.Handler: %T", unsafeRouter)
	}

	return handler
}

func mustRequest(t *testing.T, url string, token string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	return res
}

func mustReadBody(t *testing.T, res *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return string(body)
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create template directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write template %s: %v", path, err)
	}
}
