// Package opencode — session.go provides the SessionManager for running tasks.
package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// TaskRequest describes a coding task to run in a new or existing OpenCode session.
type TaskRequest struct {
	// Prompt is the human-readable task description.
	Prompt string
	// SessionID optionally re-uses an existing session.
	// If empty, a new session is created.
	SessionID string
	// Dir is the working directory for the session.
	Dir string
	// ModelID optionally overrides the model for this turn (e.g. "claude-3-5-sonnet").
	ModelID string
	// ProviderID optionally overrides the provider (e.g. "anthropic").
	ProviderID string
	// AutoPermit automatically approves all permission requests.
	// WARNING: This allows the agent to edit files and run commands without confirmation.
	AutoPermit bool
}

// TaskResult holds the outcome of a completed task.
type TaskResult struct {
	SessionID string
	Messages  []Message
}

// SessionManager orchestrates the full lifecycle of an OpenCode task:
// create session → send prompt → stream events → collect result.
type SessionManager struct {
	client *Client
}

// NewSessionManager creates a SessionManager backed by the given client.
func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{client: client}
}

// Run executes a task and returns after the session reaches idle state.
func (m *SessionManager) Run(ctx context.Context, req TaskRequest) (*TaskResult, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		sess, err := m.client.CreateSession(ctx, SessionCreateRequest{
			Directory: strPtr(req.Dir),
		})
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		sessionID = sess.ID
	}

	pr := PromptRequest{
		Parts: []PromptPart{TextPromptPart(req.Prompt)},
	}
	if req.ModelID != "" {
		pr.ModelID = &req.ModelID
	}
	if req.ProviderID != "" {
		pr.ProviderID = &req.ProviderID
	}

	if _, err := m.client.Prompt(ctx, sessionID, pr); err != nil {
		return nil, fmt.Errorf("send prompt: %w", err)
	}

	// Subscribe to events and wait for idle.
	err := m.client.SubscribeEvents(ctx, sessionID, func(ev RawEvent) error {
		switch ev.Type {
		case EventTypePermissionUpdated:
			if req.AutoPermit {
				var p Permission
				_ = unmarshalProps(ev.Properties, &p)
				_ = m.client.RespondToPermission(ctx, sessionID, p.ID, "allow")
			}
		case EventTypeSessionStatus:
			var p EventPropsSessionStatus
			_ = unmarshalProps(ev.Properties, &p)
			if p.Status.IsIdle() {
				return ErrSessionIdle
			}
		case EventTypeSessionError:
			var p EventPropsSessionError
			_ = unmarshalProps(ev.Properties, &p)
			return fmt.Errorf("session error event: %s", string(p.Error))
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrSessionIdle) {
		return nil, fmt.Errorf("waiting for session idle: %w", err)
	}

	messages, err := m.client.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	return &TaskResult{
		SessionID: sessionID,
		Messages:  messages,
	}, nil
}

// unmarshalProps is a convenience wrapper for json.Unmarshal on event properties.
func unmarshalProps(raw []byte, v interface{}) error {
	return jsonUnmarshal(raw, v)
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// RunAsync creates a session (if needed), sends the prompt, and returns a channel
// that receives every RawEvent for the session. The channel is closed when the
// session goes idle, an error occurs, or ctx is cancelled.
func (m *SessionManager) RunAsync(ctx context.Context, req TaskRequest) (<-chan RawEvent, error) {
	sessionID := req.SessionID
	if sessionID == "" {
		sess, err := m.client.CreateSession(ctx, SessionCreateRequest{
			Directory: strPtr(req.Dir),
		})
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		sessionID = sess.ID
	}

	pr := PromptRequest{
		Parts: []PromptPart{TextPromptPart(req.Prompt)},
	}
	if req.ModelID != "" {
		pr.ModelID = &req.ModelID
	}
	if req.ProviderID != "" {
		pr.ProviderID = &req.ProviderID
	}

	if _, err := m.client.Prompt(ctx, sessionID, pr); err != nil {
		return nil, fmt.Errorf("send prompt: %w", err)
	}

	ch := make(chan RawEvent, 64)

	go func() {
		defer close(ch)
		_ = m.client.SubscribeEvents(ctx, sessionID, func(ev RawEvent) error {
			// Auto-permit if requested.
			if req.AutoPermit && ev.Type == EventTypePermissionUpdated {
				var p Permission
				_ = json.Unmarshal(ev.Properties, &p)
				_ = m.client.RespondToPermission(ctx, sessionID, p.ID, "allow")
			}

			select {
			case ch <- ev:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Stop when session goes idle or errors.
			switch ev.Type {
			case EventTypeSessionStatus:
				var p EventPropsSessionStatus
				_ = json.Unmarshal(ev.Properties, &p)
				if p.Status.IsIdle() {
					return ErrSessionIdle
				}
			case EventTypeSessionError:
				var p EventPropsSessionError
				_ = json.Unmarshal(ev.Properties, &p)
				return fmt.Errorf("session error: %s", string(p.Error))
			}
			return nil
		})
	}()

	return ch, nil
}
