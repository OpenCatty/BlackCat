package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/startower-observability/blackcat/internal/opencode"
)

const checkOpenCodeStatusName = "check_opencode_status"
const checkOpenCodeStatusDescription = `Check the real-time status of OpenCode sessions. Use this BEFORE claiming a task is "still running" or "completed". Returns live session data directly from the OpenCode daemon.`

var checkOpenCodeStatusParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"session_id": {
			"type": "string",
			"description": "Optional. If provided, returns detailed info for this specific session. If omitted, returns summary of all sessions."
		}
	}
}`)

// OpenCodeStatusTool checks the real-time status of OpenCode sessions.
type OpenCodeStatusTool struct {
	client *opencode.Client
}

// NewOpenCodeStatusTool creates an OpenCodeStatusTool with the given OpenCode client.
func NewOpenCodeStatusTool(client *opencode.Client) *OpenCodeStatusTool {
	return &OpenCodeStatusTool{client: client}
}

func (t *OpenCodeStatusTool) Name() string                { return checkOpenCodeStatusName }
func (t *OpenCodeStatusTool) Description() string         { return checkOpenCodeStatusDescription }
func (t *OpenCodeStatusTool) Parameters() json.RawMessage { return checkOpenCodeStatusParameters }

// Execute checks OpenCode session status. With no args, returns a summary of all sessions.
// With session_id, returns detailed info for that session.
func (t *OpenCodeStatusTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	// Health check first.
	if _, err := t.client.Health(ctx); err != nil {
		return fmt.Sprintf("OpenCode daemon is not running: %v", err), nil
	}

	// Parse optional session_id.
	var params struct {
		SessionID string `json:"session_id"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &params)
	}

	if params.SessionID != "" {
		return t.detailedSession(ctx, params.SessionID)
	}
	return t.allSessionsSummary(ctx)
}

func (t *OpenCodeStatusTool) allSessionsSummary(ctx context.Context) (string, error) {
	sessions, err := t.client.ListSessions(ctx)
	if err != nil {
		return "", fmt.Errorf("check_opencode_status: list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return "No active OpenCode sessions found.", nil
	}

	// Sort by updated time descending (most recent first).
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Time.Updated > sessions[j].Time.Updated
	})

	// Fetch session statuses for idle/busy info.
	statuses, _ := t.client.SessionStatus(ctx)

	var b strings.Builder
	fmt.Fprintf(&b, "OpenCode Sessions: %d total\n\n", len(sessions))

	for i, s := range sessions {
		if i >= 10 {
			fmt.Fprintf(&b, "... and %d more sessions\n", len(sessions)-10)
			break
		}

		title := s.Title
		if title == "" {
			title = "(untitled)"
		}

		statusStr := "unknown"
		if statuses != nil {
			if st, ok := statuses[s.ID]; ok {
				statusStr = st.Type
			}
		}

		fmt.Fprintf(&b, "• %s\n", title)
		fmt.Fprintf(&b, "  ID: %s\n", s.ID)
		fmt.Fprintf(&b, "  Dir: %s\n", s.Directory)
		fmt.Fprintf(&b, "  Status: %s\n", statusStr)
		fmt.Fprintf(&b, "  Updated: %s\n", relativeTime(s.Time.Updated))

		if s.Summary != nil {
			fmt.Fprintf(&b, "  Changes: +%d -%d in %d files\n",
				s.Summary.Additions, s.Summary.Deletions, s.Summary.Files)
		}
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String()), nil
}

func (t *OpenCodeStatusTool) detailedSession(ctx context.Context, sessionID string) (string, error) {
	session, err := t.client.GetSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("check_opencode_status: get session %s: %w", sessionID, err)
	}

	messages, err := t.client.ListMessages(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("check_opencode_status: list messages for %s: %w", sessionID, err)
	}

	// Fetch status for this session.
	statusStr := "unknown"
	if statuses, err := t.client.SessionStatus(ctx); err == nil {
		if st, ok := statuses[sessionID]; ok {
			statusStr = st.Type
		}
	}

	title := session.Title
	if title == "" {
		title = "(untitled)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Session: %s\n\n", title)
	fmt.Fprintf(&b, "• ID: %s\n", session.ID)
	fmt.Fprintf(&b, "• Dir: %s\n", session.Directory)
	fmt.Fprintf(&b, "• Status: %s\n", statusStr)
	fmt.Fprintf(&b, "• Created: %s\n", relativeTime(session.Time.Created))
	fmt.Fprintf(&b, "• Updated: %s\n", relativeTime(session.Time.Updated))
	fmt.Fprintf(&b, "• Messages: %d\n", len(messages))

	if session.Summary != nil {
		fmt.Fprintf(&b, "• Changes: +%d -%d in %d files\n",
			session.Summary.Additions, session.Summary.Deletions, session.Summary.Files)
	}

	// Show the last message info.
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		b.WriteString("\nLast message:\n")
		fmt.Fprintf(&b, "• Role: %s\n", last.Info.Role)

		if last.Info.Finish != nil {
			fmt.Fprintf(&b, "• Finish: %s\n", *last.Info.Finish)
		}
		if last.Info.Error != nil {
			fmt.Fprintf(&b, "• Error: %s\n", last.Info.Error.Name)
		}

		// Extract text content preview from parts.
		preview := extractTextPreview(last.Parts, 200)
		if preview != "" {
			fmt.Fprintf(&b, "• Content: %s\n", preview)
		}
	}

	return strings.TrimSpace(b.String()), nil
}

// extractTextPreview extracts a text preview from message parts, truncated to maxLen.
func extractTextPreview(parts []opencode.Part, maxLen int) string {
	for _, p := range parts {
		if p.Type == "text" && p.Text != nil && *p.Text != "" {
			text := *p.Text
			// Collapse whitespace for preview.
			text = strings.Join(strings.Fields(text), " ")
			if len(text) > maxLen {
				return text[:maxLen] + "..."
			}
			return text
		}
	}
	return ""
}

// relativeTime converts a Unix timestamp (seconds) to a human-readable relative time.
func relativeTime(unixSec int64) string {
	if unixSec == 0 {
		return "unknown"
	}

	// OpenCode timestamps may be in milliseconds — detect and normalize.
	ts := unixSec
	if ts > 1e12 {
		ts = ts / 1000
	}

	t := time.Unix(ts, 0)
	d := time.Since(t)

	switch {
	case d < 0:
		return "just now"
	case d < 30*time.Second:
		return "just now"
	case d < 90*time.Second:
		return "1 minute ago"
	case d < 60*time.Minute:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 90*time.Minute:
		return "1 hour ago"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}
