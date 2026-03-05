// Package opencode — sse.go provides an SSE event stream consumer.
package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EventHandler is a callback invoked for each SSE event received.
// Return a non-nil error to stop the stream.
type EventHandler func(event RawEvent) error

// SubscribeEvents opens the GET /global/event SSE stream and calls handler
// for each event whose sessionID matches the given filter (empty = all events).
// It blocks until ctx is cancelled, an error occurs, or handler returns an error.
func (c *Client) SubscribeEvents(ctx context.Context, sessionID string, handler EventHandler) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/global/event", nil)
	if err != nil {
		return fmt.Errorf("build SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.password != "" {
		req.SetBasicAuth("", c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE: unexpected status %d", resp.StatusCode)
	}

	return parseSSEStream(resp, sessionID, handler)
}

// parseSSEStream reads the SSE stream and dispatches events.
func parseSSEStream(resp *http.Response, sessionID string, handler EventHandler) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1 MB max to handle large SSE events
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		case line == "":
			// End of event — process accumulated data lines.
			if len(dataLines) > 0 {
				raw := strings.Join(dataLines, "\n")
				dataLines = dataLines[:0]

				// OpenCode wraps events in GlobalEvent; unwrap to get the payload.
				var ge GlobalEvent
				if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &ge); err != nil {
					// Ignore unparseable events (heartbeats, etc.).
					continue
				}

				// Apply session filter.
				if sessionID != "" && !eventBelongsToSession(ge.Payload, sessionID) {
					continue
				}

				if err := handler(ge.Payload); err != nil {
					return err
				}
			}
		}
	}
	return scanner.Err()
}

// eventBelongsToSession returns true if the event carries the given sessionID.
// Because OpenCode has no server-side session filter (issue #7451), we do it client-side.
func eventBelongsToSession(ev RawEvent, sessionID string) bool {
	// Fast path: parse only the sessionID field.
	var probe struct {
		SessionID string `json:"sessionID"`
		Info      struct {
			ID string `json:"id"`
		} `json:"info"`
	}
	_ = json.Unmarshal(ev.Properties, &probe)
	if probe.SessionID == sessionID {
		return true
	}
	// session.created / session.updated / session.deleted nest it in info.id.
	if probe.Info.ID == sessionID {
		return true
	}
	// Global events (server.connected, file.watcher.*) are broadcast to all.
	switch ev.Type {
	case EventTypeServerConnected,
		EventTypeServerInstanceDisposed,
		EventTypeInstallationUpdated,
		EventTypeInstallationUpdateAvailable,
		EventTypeVcsBranchUpdated,
		EventTypeFileWatcherUpdated:
		return true
	}
	return false
}

// ErrSessionIdle is returned from an EventHandler to signal clean completion.
var ErrSessionIdle = fmt.Errorf("session idle")

// WaitForIdle subscribes to events and blocks until the given session reports
// SessionStatus.idle, then returns the final list of messages.
func (c *Client) WaitForIdle(ctx context.Context, sessionID string) error {
	return c.SubscribeEvents(ctx, sessionID, func(ev RawEvent) error {
		switch ev.Type {
		case EventTypeSessionStatus:
			var p EventPropsSessionStatus
			if err := json.Unmarshal(ev.Properties, &p); err != nil {
				return nil
			}
			if p.Status.IsIdle() {
				return ErrSessionIdle
			}
		case EventTypeSessionError:
			var p EventPropsSessionError
			_ = json.Unmarshal(ev.Properties, &p)
			return fmt.Errorf("session error: %s", string(p.Error))
		case EventTypePermissionUpdated:
			// A permission gate arrived — the caller should handle it.
			// By default we return an error so the caller can decide.
			return fmt.Errorf("permission required: see PermissionUpdated event")
		}
		return nil
	})
}

// ReconnectConfig controls the SSE reconnection behaviour.
type ReconnectConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultReconnectConfig returns sensible defaults: 10 retries, 1 s initial backoff, 30 s max.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		MaxRetries:     10,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
	}
}

// SubscribeEventsWithReconnect opens the SSE stream and automatically reconnects
// on disconnection using exponential backoff. It honours Last-Event-ID and
// Retry-After (HTTP 429) semantics.
func (c *Client) SubscribeEventsWithReconnect(
	ctx context.Context,
	sessionID string,
	config ReconnectConfig,
	handler EventHandler,
) error {
	var (
		lastEventID string
		mu          sync.Mutex
		backoff     = config.InitialBackoff
	)

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			// Exponential backoff: double, capped at MaxBackoff.
			backoff *= 2
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/global/event", nil)
		if err != nil {
			return fmt.Errorf("build SSE request: %w", err)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		if c.password != "" {
			req.SetBasicAuth("", c.password)
		}

		mu.Lock()
		if lastEventID != "" {
			req.Header.Set("Last-Event-ID", lastEventID)
		}
		mu.Unlock()

		resp, err := c.http.Do(req)
		if err != nil {
			// Connection error — retry.
			continue
		}

		// Handle HTTP 429 Too Many Requests.
		if resp.StatusCode == http.StatusTooManyRequests {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, parseErr := strconv.Atoi(ra); parseErr == nil {
					backoff = time.Duration(secs) * time.Second
				}
			}
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("SSE: unexpected status %d", resp.StatusCode)
		}

		// Reset backoff on successful connection.
		backoff = config.InitialBackoff

		streamErr := parseSSEStreamWithID(resp, sessionID, handler, &lastEventID, &mu)
		resp.Body.Close()

		// If the handler returned an error (including ErrSessionIdle), propagate it.
		if streamErr != nil {
			return streamErr
		}

		// Stream ended cleanly (server closed) — reconnect.
	}

	return fmt.Errorf("SSE reconnect: exhausted %d retries", config.MaxRetries)
}

// parseSSEStreamWithID is like parseSSEStream but also tracks the SSE "id:" field
// so that Last-Event-ID can be sent on reconnection.
func parseSSEStreamWithID(
	resp *http.Response,
	sessionID string,
	handler EventHandler,
	lastEventID *string,
	mu *sync.Mutex,
) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1 MB max to handle large SSE events
	var (
		dataLines []string
		currentID string
	)

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		case strings.HasPrefix(line, "id:"):
			currentID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case line == "":
			// End of event — process accumulated data lines.
			if len(dataLines) > 0 {
				// Track the last event ID.
				if currentID != "" {
					mu.Lock()
					*lastEventID = currentID
					mu.Unlock()
				}

				raw := strings.Join(dataLines, "\n")
				dataLines = dataLines[:0]
				currentID = ""

				var ge GlobalEvent
				if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &ge); err != nil {
					continue
				}

				if sessionID != "" && !eventBelongsToSession(ge.Payload, sessionID) {
					continue
				}

				if err := handler(ge.Payload); err != nil {
					return err
				}
			}
		}
	}
	return scanner.Err()
}
