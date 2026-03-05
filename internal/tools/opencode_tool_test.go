package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/internal/opencode"
)

// newMockOpenCodeServer creates an httptest.Server that simulates the OpenCode REST API.
// If delaySSE > 0, the SSE endpoint delays before sending the idle event.
// If skipCreateSession is true, POST /session returns 404 (for session reuse tests).
func newMockOpenCodeServer(delaySSE time.Duration, skipCreateSession bool) *httptest.Server {
	mux := http.NewServeMux()

	// POST /session — create a new session.
	mux.HandleFunc("POST /session", func(w http.ResponseWriter, r *http.Request) {
		if skipCreateSession {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(opencode.Session{
			ID:        "test-session-1",
			ProjectID: "proj-1",
			Directory: ".",
			Title:     "Test Session",
		})
	})

	// POST /session/{id}/prompt_async — send a prompt (returns 204 No Content).
	mux.HandleFunc("POST /session/{id}/prompt_async", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /global/event — SSE stream that emits session.status idle.
	mux.HandleFunc("GET /global/event", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		if delaySSE > 0 {
			select {
			case <-time.After(delaySSE):
			case <-r.Context().Done():
				return
			}
		}

		// Emit session.status idle event wrapped in GlobalEvent envelope.
		event := fmt.Sprintf(`{"directory":".","payload":{"type":"session.status","properties":{"sessionID":"test-session-1","status":{"type":"idle"}}}}`)
		fmt.Fprintf(w, "data:%s\n\n", event)
		flusher.Flush()
	})

	// GET /session/{id}/message — list messages (envelope format: {info, parts}).
	mux.HandleFunc("GET /session/{id}/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		helloText := "Hello! I fixed the bug."
		messages := []opencode.MessageWithParts{
			{
				Info: opencode.Message{
					ID:        "msg-user-1",
					SessionID: "test-session-1",
					Role:      "user",
				},
				Parts: []opencode.Part{
					{ID: "prt-1", Type: "text", Text: strPtr("Fix the bug")},
				},
			},
			{
				Info: opencode.Message{
					ID:        "msg-asst-1",
					SessionID: "test-session-1",
					Role:      "assistant",
					Agent:     "code",
				},
				Parts: []opencode.Part{
					{ID: "prt-2", Type: "text", Text: &helloText},
				},
			},
		}
		json.NewEncoder(w).Encode(messages)
	})

	return httptest.NewServer(mux)
}

func strPtr(s string) *string { return &s }

func TestOpenCodeToolExecute(t *testing.T) {
	srv := newMockOpenCodeServer(0, false)
	defer srv.Close()

	client := opencode.NewClient(srv.URL, opencode.WithHTTPClient(srv.Client()))
	tool := NewOpenCodeTool(client, false, 10*time.Second)

	args, _ := json.Marshal(map[string]string{
		"prompt": "Fix the bug",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify structured output.
	if !strings.Contains(result, "OpenCode Task Complete") {
		t.Errorf("result missing 'OpenCode Task Complete': %s", result)
	}
	if !strings.Contains(result, "test-session-1") {
		t.Errorf("result missing session ID: %s", result)
	}
	if !strings.Contains(result, "Messages: 2") {
		t.Errorf("result missing message count: %s", result)
	}
	if !strings.Contains(result, "Hello! I fixed the bug.") {
		t.Errorf("result missing assistant text content: %s", result)
	}
	t.Logf("result:\n%s", result)
}

func TestOpenCodeToolWithSessionID(t *testing.T) {
	// skipCreateSession=true so POST /session returns 404.
	// The tool should NOT create a new session because session_id is provided.
	srv := newMockOpenCodeServer(0, true)
	defer srv.Close()

	client := opencode.NewClient(srv.URL, opencode.WithHTTPClient(srv.Client()))
	tool := NewOpenCodeTool(client, false, 10*time.Second)

	args, _ := json.Marshal(map[string]string{
		"prompt":     "Refactor the module",
		"session_id": "test-session-1",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should succeed because it reuses the existing session (no POST /session call).
	if !strings.Contains(result, "OpenCode Task Complete") {
		t.Errorf("result missing 'OpenCode Task Complete': %s", result)
	}
	if !strings.Contains(result, "test-session-1") {
		t.Errorf("result missing session ID: %s", result)
	}
	t.Logf("result:\n%s", result)
}

func TestOpenCodeToolTimeout(t *testing.T) {
	// SSE endpoint delays 5 seconds, but tool timeout is 100ms.
	srv := newMockOpenCodeServer(5*time.Second, false)
	defer srv.Close()

	client := opencode.NewClient(srv.URL, opencode.WithHTTPClient(srv.Client()))
	tool := NewOpenCodeTool(client, false, 100*time.Millisecond)

	args, _ := json.Marshal(map[string]string{
		"prompt": "This should timeout",
	})

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return Go error (returns error string): %v", err)
	}

	if !strings.Contains(result, "error") {
		t.Errorf("expected error in result for timeout, got: %s", result)
	}
	t.Logf("timeout result: %s", result)
}

func TestOpenCodeToolParameters(t *testing.T) {
	client := opencode.NewClient("http://localhost:0")
	tool := NewOpenCodeTool(client, false, 0)

	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() returned nil")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("Parameters() returned invalid JSON: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("missing properties in schema")
	}
	for _, field := range []string{"prompt", "dir", "session_id", "model"} {
		if _, exists := props[field]; !exists {
			t.Errorf("missing property %q in schema", field)
		}
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("missing required in schema")
	}
	if len(required) != 1 || required[0] != "prompt" {
		t.Errorf("expected required=[prompt], got %v", required)
	}
}

func TestOpenCodeToolName(t *testing.T) {
	client := opencode.NewClient("http://localhost:0")
	tool := NewOpenCodeTool(client, false, 0)

	if got := tool.Name(); got != "opencode_task" {
		t.Errorf("Name() = %q, want %q", got, "opencode_task")
	}
	if got := tool.Description(); got == "" {
		t.Error("Description() is empty")
	}
}
