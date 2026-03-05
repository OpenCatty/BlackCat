// Package opencode — client.go provides an HTTP client for the OpenCode REST API.
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is an HTTP client for the OpenCode REST API.
type Client struct {
	base    string
	http    *http.Client
	password string
}

// NewClient creates a Client targeting the given base URL (e.g. "http://127.0.0.1:4096").
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		base: baseURL,
		http: &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithPassword sets Basic Auth credentials for the client.
func WithPassword(password string) ClientOption {
	return func(c *Client) { c.password = password }
}

// WithHTTPClient replaces the underlying http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.http = hc }
}

// Health calls GET /global/health.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var r HealthResponse
	return &r, c.get(ctx, "/global/health", &r)
}

// ListSessions calls GET /session.
func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var r []Session
	return r, c.get(ctx, "/session", &r)
}

// CreateSession calls POST /session.
func (c *Client) CreateSession(ctx context.Context, req SessionCreateRequest) (*Session, error) {
	var r Session
	return &r, c.post(ctx, "/session", req, &r)
}

// GetSession calls GET /session/:id.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var r Session
	return &r, c.get(ctx, "/session/"+sessionID, &r)
}

// DeleteSession calls DELETE /session/:id.
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	return c.do(ctx, http.MethodDelete, "/session/"+sessionID, nil, nil)
}

// ListMessages calls GET /session/:id/message.
// The API returns an array of {info: Message, parts: []Part} envelopes.
func (c *Client) ListMessages(ctx context.Context, sessionID string) ([]MessageWithParts, error) {
	var r []MessageWithParts
	return r, c.get(ctx, "/session/"+sessionID+"/message", &r)
}

// Prompt sends a prompt to a session asynchronously (POST /session/:id/prompt_async).
// The server returns 204 No Content; watch the SSE stream for results.
func (c *Client) Prompt(ctx context.Context, sessionID string, req PromptRequest) error {
	return c.post(ctx, "/session/"+sessionID+"/prompt_async", req, nil)
}

// RespondToPermission approves or denies a permission request.
func (c *Client) RespondToPermission(ctx context.Context, sessionID, permID, response string) error {
	req := PermissionResponseRequest{Response: response}
	return c.post(ctx, "/session/"+sessionID+"/permission/"+permID, req, nil)
}

// Interrupt sends an interrupt signal to a session (POST /session/:id/interrupt).
func (c *Client) Interrupt(ctx context.Context, sessionID string) error {
	return c.post(ctx, "/session/"+sessionID+"/interrupt", nil, nil)
}

// Abort sends an abort signal to a session (POST /session/:id/abort).
// The server responds with 204 No Content.
func (c *Client) Abort(ctx context.Context, sessionID string) error {
	return c.post(ctx, "/session/"+sessionID+"/abort", nil, nil)
}

// Shell executes a shell command in a session (POST /session/:id/shell).
func (c *Client) Shell(ctx context.Context, sessionID string, command string) error {
	body := struct {
		Command string `json:"command"`
	}{Command: command}
	return c.post(ctx, "/session/"+sessionID+"/shell", body, nil)
}

// SessionStatus retrieves status for all sessions (GET /session/status).
func (c *Client) SessionStatus(ctx context.Context) (map[string]SessionStatus, error) {
	var r map[string]SessionStatus
	return r, c.get(ctx, "/session/status", &r)
}

// ─── low-level helpers ───────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out interface{}) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.password != "" {
		req.SetBasicAuth("", c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opencode API %s %s: HTTP %d: %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
