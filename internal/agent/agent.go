// Package agent implements the BlackCat agent loop.
//
// The agent wraps an opencode.SessionManager and provides a higher-level
// interface for dispatching coding tasks, streaming progress events, and
// collecting results.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/startower-observability/blackcat/internal/opencode"
)

// Config holds all settings needed to create an Agent.
type Config struct {
	// OpenCodeAddr is the base URL of a running opencode server (e.g. "http://127.0.0.1:4096").
	OpenCodeAddr string
	// Password is the optional Basic Auth password for the opencode server.
	Password string
	// AutoPermit automatically approves all permission requests during task runs.
	// WARNING: This allows the agent to edit files and run commands without confirmation.
	AutoPermit bool
	// Verbose enables streaming SSE event output to the provided writer.
	Verbose bool
	// Output is where progress events are written when Verbose is true.
	// Defaults to io.Discard if nil.
	Output io.Writer
}

// Agent orchestrates coding tasks against an opencode server.
type Agent struct {
	cfg     Config
	client  *opencode.Client
	session *opencode.SessionManager
	out     io.Writer
	loop    *Loop
}

// New creates an Agent from the given Config.
func New(cfg Config) *Agent {
	var opts []opencode.ClientOption
	if cfg.Password != "" {
		opts = append(opts, opencode.WithPassword(cfg.Password))
	}
	out := cfg.Output
	if out == nil {
		out = io.Discard
	}
	c := opencode.NewClient(cfg.OpenCodeAddr, opts...)
	return &Agent{
		cfg:     cfg,
		client:  c,
		session: opencode.NewSessionManager(c),
		out:     out,
	}
}

// NewWithLoop creates an Agent and initializes the think-act-observe loop.
func NewWithLoop(cfg Config, loopCfg LoopConfig) *Agent {
	a := New(cfg)
	a.loop = NewLoop(loopCfg)
	return a
}

// Client returns the underlying opencode REST client.
func (a *Agent) Client() *opencode.Client { return a.client }

// TaskRequest is the input to Agent.Run.
type TaskRequest = opencode.TaskRequest

// TaskResult is the output from Agent.Run.
type TaskResult = opencode.TaskResult

// Run executes a coding task and blocks until the session is idle.
// Progress events are written to the configured output when Verbose is set.
func (a *Agent) Run(ctx context.Context, req TaskRequest) (*TaskResult, error) {
	if a.cfg.Verbose {
		fmt.Fprintf(a.out, "[blackcat] starting task: %s\n", req.Prompt)
	}
	result, err := a.session.Run(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agent run: %w", err)
	}
	if a.cfg.Verbose {
		fmt.Fprintf(a.out, "[blackcat] task complete (session %s, %d messages)\n",
			result.SessionID, len(result.Messages))
	}
	return result, nil
}

// RunLoop executes the local think-act-observe agent loop.
func (a *Agent) RunLoop(ctx context.Context, userMessage string) (string, error) {
	if a.loop == nil {
		return "", fmt.Errorf("agent loop not configured")
	}

	if a.cfg.Verbose {
		fmt.Fprintf(a.out, "[blackcat] loop start\n")
	}

	execution, err := a.loop.Run(ctx, userMessage)
	if err != nil {
		return "", err
	}

	if a.cfg.Verbose {
		fmt.Fprintf(a.out, "[blackcat] loop complete (%d turns)\n", execution.TurnCount)
	}

	return execution.Response, nil
}

// Health checks whether the opencode server is reachable and healthy.
func (a *Agent) Health(ctx context.Context) error {
	h, err := a.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("opencode health check failed: %w", err)
	}
	if !h.Healthy {
		return fmt.Errorf("opencode server reports unhealthy")
	}
	return nil
}

// EventSummary is a human-readable summary of a single SSE event.
type EventSummary struct {
	Type      string
	SessionID string
	Detail    string
}

// SummariseEvent extracts a human-readable summary from a raw SSE event.
func SummariseEvent(ev opencode.RawEvent) EventSummary {
	s := EventSummary{Type: ev.Type}
	switch ev.Type {
	case opencode.EventTypeSessionStatus:
		var p opencode.EventPropsSessionStatus
		_ = json.Unmarshal(ev.Properties, &p)
		s.SessionID = p.SessionID
		s.Detail = p.Status.Type
	case opencode.EventTypeMessagePartUpdated:
		var p opencode.EventPropsMessagePartUpdated
		_ = json.Unmarshal(ev.Properties, &p)
		s.SessionID = p.Part.SessionID
		if p.Delta != nil {
			s.Detail = *p.Delta
		}
	case opencode.EventTypePermissionUpdated:
		var p opencode.EventPropsPermissionUpdated
		_ = json.Unmarshal(ev.Properties, &p)
		s.SessionID = p.SessionID
		s.Detail = fmt.Sprintf("permission requested: %s (id=%s)", p.Title, p.ID)
	case opencode.EventTypeSessionError:
		var p opencode.EventPropsSessionError
		_ = json.Unmarshal(ev.Properties, &p)
		if p.SessionID != nil {
			s.SessionID = *p.SessionID
		}
		s.Detail = string(p.Error)
	default:
		s.Detail = string(ev.Properties)
	}
	return s
}
