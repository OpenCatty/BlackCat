package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/dashboard"
	"github.com/startower-observability/blackcat/internal/daemon"
	"github.com/startower-observability/blackcat/internal/scheduler"
	"github.com/startower-observability/blackcat/internal/session"
	"github.com/startower-observability/blackcat/internal/types"
)

type mockSubsystemManager struct {
	health []daemon.SubsystemHealth
}

func (m mockSubsystemManager) Healthz() []daemon.SubsystemHealth {
	return m.health
}

type mockTaskLister struct {
	tasks []string
}

func (m mockTaskLister) ListTasks() []string {
	return m.tasks
}

type mockHeartbeatStore struct {
	results []scheduler.HeartbeatResult
}

func (m mockHeartbeatStore) Latest(n int) []scheduler.HeartbeatResult {
	if n <= 0 {
		return []scheduler.HeartbeatResult{}
	}
	if n > len(m.results) {
		n = len(m.results)
	}
	return slices.Clone(m.results[:n])
}

func TestSessionPersistRestart(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	key := session.SessionKey{ChannelType: "telegram", ChannelID: "123", UserID: "user1"}

	store, err := session.NewFileStore(storeDir, 50)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	first := newSession(key, []string{"msg-1", "msg-2", "msg-3", "msg-4", "msg-5"})
	if err := store.Save(first); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	restartedStore, err := session.NewFileStore(storeDir, 50)
	if err != nil {
		t.Fatalf("NewFileStore restart failed: %v", err)
	}

	loaded, err := restartedStore.Get(key)
	if err != nil {
		t.Fatalf("Get after restart failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected session after restart, got nil")
	}
	if len(loaded.Messages) != 5 {
		t.Fatalf("expected 5 messages after restart, got %d", len(loaded.Messages))
	}
	for i, msg := range loaded.Messages {
		expected := "msg-" + strconv.Itoa(i+1)
		if msg.Content != expected {
			t.Fatalf("expected message %q at index %d, got %q", expected, i, msg.Content)
		}
	}
}

func TestDashboardSessionAPI(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	store, err := session.NewFileStore(storeDir, 50)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	sessions := []session.SessionKey{
		{ChannelType: "telegram", ChannelID: "ch-1", UserID: "user-1"},
		{ChannelType: "telegram", ChannelID: "ch-1", UserID: "user-2"},
		{ChannelType: "discord", ChannelID: "guild-9", UserID: "user-1"},
	}
	for i, key := range sessions {
		if err := store.Save(newSession(key, []string{"hello-" + strconv.Itoa(i+1)})); err != nil {
			t.Fatalf("Save failed for key %s: %v", key.String(), err)
		}
	}

	keys, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 stored sessions, got %d", len(keys))
	}

	server := newDashboardHTTPTestServer(t, "test-token")
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/dashboard/api/agents", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard/api/agents failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestDashboardTokenAuth(t *testing.T) {
	t.Parallel()

	server := newDashboardHTTPTestServer(t, "test-token")
	defer server.Close()

	respNoAuth, err := http.Get(server.URL + "/dashboard/api/status")
	if err != nil {
		t.Fatalf("GET without auth failed: %v", err)
	}
	defer respNoAuth.Body.Close()

	if respNoAuth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d without auth, got %d", http.StatusUnauthorized, respNoAuth.StatusCode)
	}
	bodyNoAuth, err := io.ReadAll(respNoAuth.Body)
	if err != nil {
		t.Fatalf("ReadAll unauthorized body failed: %v", err)
	}
	if strings.TrimSpace(string(bodyNoAuth)) != `{"error":"unauthorized"}` {
		t.Fatalf("unexpected unauthorized body: %q", strings.TrimSpace(string(bodyNoAuth)))
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/dashboard/api/status", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	respAuth, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET with auth failed: %v", err)
	}
	defer respAuth.Body.Close()

	if respAuth.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d with auth, got %d", http.StatusOK, respAuth.StatusCode)
	}

	var statusPayload map[string]any
	if err := json.NewDecoder(respAuth.Body).Decode(&statusPayload); err != nil {
		t.Fatalf("decode authorized JSON failed: %v", err)
	}
	if _, ok := statusPayload["healthy"]; !ok {
		t.Fatalf("expected healthy field in status response, got: %#v", statusPayload)
	}
}

func TestDashboardSSEEvents(t *testing.T) {
	t.Parallel()

	server := newDashboardHTTPTestServer(t, "test-token")
	defer server.Close()

	// Create a request with a timeout context to read only the first chunk
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/dashboard/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard/events failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", ct)
	}

	// Read first chunk to verify keepalive
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read SSE body failed: %v", err)
	}
	body := string(buf[:n])
	if !strings.Contains(body, ": keepalive") {
		t.Fatalf("expected keepalive payload, got %q", body)
	}
}

func TestSessionPerUserPerChannel(t *testing.T) {
	t.Parallel()

	store, err := session.NewFileStore(t.TempDir(), 50)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	keyA := session.SessionKey{ChannelType: "telegram", ChannelID: "ch1", UserID: "user1"}
	keyB := session.SessionKey{ChannelType: "telegram", ChannelID: "ch1", UserID: "user2"}
	keyC := session.SessionKey{ChannelType: "discord", ChannelID: "ch1", UserID: "user1"}

	if err := store.Save(newSession(keyA, []string{"telegram-user1"})); err != nil {
		t.Fatalf("Save keyA failed: %v", err)
	}
	if err := store.Save(newSession(keyB, []string{"telegram-user2"})); err != nil {
		t.Fatalf("Save keyB failed: %v", err)
	}
	if err := store.Save(newSession(keyC, []string{"discord-user1"})); err != nil {
		t.Fatalf("Save keyC failed: %v", err)
	}

	keys, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(keys))
	}

	loadedA, err := store.Get(keyA)
	if err != nil {
		t.Fatalf("Get keyA failed: %v", err)
	}
	if loadedA == nil {
		t.Fatal("expected keyA session, got nil")
	}
	if len(loadedA.Messages) != 1 || loadedA.Messages[0].Content != "telegram-user1" {
		t.Fatalf("unexpected keyA messages: %#v", loadedA.Messages)
	}

	loadedB, err := store.Get(keyB)
	if err != nil {
		t.Fatalf("Get keyB failed: %v", err)
	}
	if loadedB == nil {
		t.Fatal("expected keyB session, got nil")
	}
	if len(loadedB.Messages) != 1 || loadedB.Messages[0].Content != "telegram-user2" {
		t.Fatalf("unexpected keyB messages: %#v", loadedB.Messages)
	}

	loadedC, err := store.Get(keyC)
	if err != nil {
		t.Fatalf("Get keyC failed: %v", err)
	}
	if loadedC == nil {
		t.Fatal("expected keyC session, got nil")
	}
	if len(loadedC.Messages) != 1 || loadedC.Messages[0].Content != "discord-user1" {
		t.Fatalf("unexpected keyC messages: %#v", loadedC.Messages)
	}
}

func TestSessionAnonymousUser(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	store, err := session.NewFileStore(storeDir, 50)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	key := session.SessionKey{ChannelType: "telegram", ChannelID: "123", UserID: ""}
	if err := store.Save(newSession(key, []string{"anonymous-message"})); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected anonymous session, got nil")
	}
	if loaded.Key.UserID != "" {
		t.Fatalf("expected empty UserID, got %q", loaded.Key.UserID)
	}

	path := filepath.Join(storeDir, "telegram_123.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected anonymous session filename %s to exist: %v", path, err)
	}

	invalid := filepath.Join(storeDir, "telegram_123_.json")
	if _, err := os.Stat(invalid); err == nil {
		t.Fatalf("did not expect anonymous filename with trailing underscore to exist: %s", invalid)
	}
}

func newSession(key session.SessionKey, messages []string) *session.Session {
	now := time.Now()
	llmMessages := make([]types.LLMMessage, 0, len(messages))
	for _, content := range messages {
		llmMessages = append(llmMessages, types.LLMMessage{Role: "user", Content: content})
	}

	return &session.Session{
		Key:       key,
		Messages:  llmMessages,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newDashboardHTTPTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()

	deps := dashboard.DashboardDeps{
		SubsystemManager: mockSubsystemManager{health: []daemon.SubsystemHealth{{
			Name:    "session-store",
			Status:  "running",
			Message: "healthy",
		}}},
		TaskLister:     mockTaskLister{tasks: []string{"session-sync", "dashboard-refresh"}},
		HeartbeatStore: mockHeartbeatStore{results: []scheduler.HeartbeatResult{}},
	}

	s := dashboard.NewServer(config.DashboardConfig{
		Enabled: true,
		Addr:    ":0",
		Token:   token,
	}, deps)
	if s == nil {
		t.Fatal("expected dashboard server instance, got nil")
	}

	return httptest.NewServer(extractDashboardHandler(t, s))
}

func extractDashboardHandler(t *testing.T, s *dashboard.Server) http.Handler {
	t.Helper()

	serverValue := reflect.ValueOf(s).Elem()
	httpServerField := serverValue.FieldByName("httpServer")
	if !httpServerField.IsValid() {
		t.Fatal("dashboard server has no httpServer field")
	}

	httpServerPtr := reflect.NewAt(httpServerField.Type(), unsafe.Pointer(httpServerField.UnsafeAddr())).Elem().Interface().(*http.Server)
	if httpServerPtr == nil {
		t.Fatal("dashboard http server is nil")
	}
	if httpServerPtr.Handler == nil {
		t.Fatal("dashboard http server handler is nil")
	}

	return httpServerPtr.Handler
}
