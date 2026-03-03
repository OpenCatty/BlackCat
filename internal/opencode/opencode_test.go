package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// helper: write a single SSE event to an http.ResponseWriter.
func writeSSE(w http.ResponseWriter, id, eventType string, props interface{}) {
	payload := RawEvent{Type: eventType}
	if props != nil {
		b, _ := json.Marshal(props)
		payload.Properties = b
	} else {
		payload.Properties = json.RawMessage(`{}`)
	}
	ge := GlobalEvent{Directory: "/tmp", Payload: payload}
	data, _ := json.Marshal(ge)
	if id != "" {
		fmt.Fprintf(w, "id:%s\n", id)
	}
	fmt.Fprintf(w, "data:%s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ---------- TestSubscribeEventsWithReconnect ----------

func TestSubscribeEventsWithReconnect(t *testing.T) {
	var connectCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global/event" {
			http.NotFound(w, r)
			return
		}

		attempt := atomic.AddInt32(&connectCount, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if attempt == 1 {
			// First connection: send 2 events then drop.
			writeSSE(w, "evt-1", EventTypeTodoUpdated, EventPropsTodoUpdated{SessionID: "sess-1"})
			writeSSE(w, "evt-2", EventTypeTodoUpdated, EventPropsTodoUpdated{SessionID: "sess-1"})
			return // close connection
		}

		// Second connection: verify Last-Event-ID header.
		lastID := r.Header.Get("Last-Event-ID")
		if lastID != "evt-2" {
			t.Errorf("expected Last-Event-ID=evt-2, got %q", lastID)
		}

		// Send one more event then idle to terminate cleanly.
		writeSSE(w, "evt-3", EventTypeTodoUpdated, EventPropsTodoUpdated{SessionID: "sess-1"})
		writeSSE(w, "evt-4", EventTypeSessionStatus, EventPropsSessionStatus{
			SessionID: "sess-1",
			Status:    SessionStatus{Type: "idle"},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))

	cfg := ReconnectConfig{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond, // fast for tests
		MaxBackoff:     100 * time.Millisecond,
	}

	var received []string
	var mu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.SubscribeEventsWithReconnect(ctx, "sess-1", cfg, func(ev RawEvent) error {
		mu.Lock()
		received = append(received, ev.Type)
		mu.Unlock()

		if ev.Type == EventTypeSessionStatus {
			return ErrSessionIdle
		}
		return nil
	})

	if err != ErrSessionIdle {
		t.Fatalf("expected ErrSessionIdle, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// We expect: 2 events from first connection + 2 events from second = 4 total.
	if len(received) < 3 {
		t.Errorf("expected at least 3 events, got %d: %v", len(received), received)
	}

	if atomic.LoadInt32(&connectCount) < 2 {
		t.Errorf("expected at least 2 connections, got %d", connectCount)
	}
}

// ---------- TestAbort ----------

func TestAbort(t *testing.T) {
	var called atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/abort") {
			called.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))
	err := client.Abort(context.Background(), "sess-abc")
	if err != nil {
		t.Fatalf("Abort failed: %v", err)
	}
	if called.Load() != 1 {
		t.Errorf("expected abort endpoint called once, got %d", called.Load())
	}
}

// ---------- TestShell ----------

func TestShell(t *testing.T) {
	var capturedBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/shell") {
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))
	err := client.Shell(context.Background(), "sess-xyz", "ls -la")
	if err != nil {
		t.Fatalf("Shell failed: %v", err)
	}

	var parsed struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(capturedBody), &parsed); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if parsed.Command != "ls -la" {
		t.Errorf("expected command 'ls -la', got %q", parsed.Command)
	}
}

// ---------- TestRunAsync ----------

func TestRunAsync(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			// CreateSession
			json.NewEncoder(w).Encode(Session{ID: "async-sess"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/prompt_async"):
			// Prompt
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"messageID": "msg-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/global/event":
			// SSE stream
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			writeSSE(w, "", EventTypeTodoUpdated, EventPropsTodoUpdated{SessionID: "async-sess"})
			writeSSE(w, "", EventTypeTodoUpdated, EventPropsTodoUpdated{SessionID: "async-sess"})
			writeSSE(w, "", EventTypeSessionStatus, EventPropsSessionStatus{
				SessionID: "async-sess",
				Status:    SessionStatus{Type: "idle"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))
	mgr := NewSessionManager(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := mgr.RunAsync(ctx, TaskRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("RunAsync failed: %v", err)
	}

	var events []string
	for ev := range ch {
		events = append(events, ev.Type)
	}

	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d: %v", len(events), events)
	}

	// Channel should be closed now.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}
}

// ---------- TestReconnectHTTP429 ----------

func TestReconnectHTTP429(t *testing.T) {
	var connectCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global/event" {
			http.NotFound(w, r)
			return
		}

		attempt := atomic.AddInt32(&connectCount, 1)

		if attempt == 1 {
			// Return 429 with Retry-After.
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		// Second attempt: succeed and send idle.
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		writeSSE(w, "", EventTypeSessionStatus, EventPropsSessionStatus{
			SessionID: "sess-429",
			Status:    SessionStatus{Type: "idle"},
		})
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))

	cfg := ReconnectConfig{
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.SubscribeEventsWithReconnect(ctx, "sess-429", cfg, func(ev RawEvent) error {
		if ev.Type == EventTypeSessionStatus {
			return ErrSessionIdle
		}
		return nil
	})

	elapsed := time.Since(start)

	if err != ErrSessionIdle {
		t.Fatalf("expected ErrSessionIdle, got %v", err)
	}

	// The Retry-After header says 1 second, so we should have waited at least ~1s.
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected at least ~1s wait due to Retry-After, but only waited %v", elapsed)
	}

	if atomic.LoadInt32(&connectCount) < 2 {
		t.Errorf("expected at least 2 connections, got %d", connectCount)
	}
}

// ---------- TestSessionStatus ----------

func TestSessionStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/session/status" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]SessionStatus{
				"sess-1": {Type: "idle"},
				"sess-2": {Type: "busy"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := NewClient(ts.URL, WithHTTPClient(ts.Client()))
	statuses, err := client.SessionStatus(context.Background())
	if err != nil {
		t.Fatalf("SessionStatus failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses["sess-1"].Type != "idle" {
		t.Errorf("expected sess-1 idle, got %q", statuses["sess-1"].Type)
	}
}
