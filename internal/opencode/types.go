// Package opencode provides Go types for the OpenCode REST API and SSE events.
// Source: https://github.com/sst/opencode/blob/dev/packages/sdk/js/src/gen/types.gen.ts
package opencode

import "encoding/json"

// Session represents an OpenCode coding session.
type Session struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"projectID"`
	Directory string          `json:"directory"`
	ParentID  *string         `json:"parentID,omitempty"`
	Title     string          `json:"title"`
	Version   string          `json:"version"`
	Time      SessionTime     `json:"time"`
	Summary   *SessionSummary `json:"summary,omitempty"`
	Share     *SessionShare   `json:"share,omitempty"`
	Revert    *SessionRevert  `json:"revert,omitempty"`
}

type SessionTime struct {
	Created    int64  `json:"created"`
	Updated    int64  `json:"updated"`
	Compacting *int64 `json:"compacting,omitempty"`
}

type SessionSummary struct {
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Files     int        `json:"files"`
	Diffs     []FileDiff `json:"diffs,omitempty"`
}

type SessionShare struct {
	URL string `json:"url"`
}

type SessionRevert struct {
	MessageID string  `json:"messageID"`
	PartID    *string `json:"partID,omitempty"`
	Snapshot  *string `json:"snapshot,omitempty"`
	Diff      *string `json:"diff,omitempty"`
}

// FileDiff represents a single file change.
type FileDiff struct {
	File      string `json:"file"`
	Before    string `json:"before"`
	After     string `json:"after"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// Message is the union of UserMessage and AssistantMessage.
// Discriminate on Role ("user" | "assistant").
type Message struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionID"`
	Role      string          `json:"role"`
	Time      json.RawMessage `json:"time"`
	Agent     string          `json:"agent,omitempty"`
	Model     json.RawMessage `json:"model,omitempty"`
	System    *string         `json:"system,omitempty"`
	Tools     map[string]bool `json:"tools,omitempty"`
	ParentID   *string         `json:"parentID,omitempty"`
	ModelID    *string         `json:"modelID,omitempty"`
	ProviderID *string         `json:"providerID,omitempty"`
	Mode       *string         `json:"mode,omitempty"`
	Path       json.RawMessage `json:"path,omitempty"`
	Cost       *float64        `json:"cost,omitempty"`
	Tokens     json.RawMessage `json:"tokens,omitempty"`
	Finish     *string         `json:"finish,omitempty"`
	Error      *MessageError   `json:"error,omitempty"`
	Summary    *bool           `json:"summary,omitempty"`
}

type MessageError struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// Part is the discriminated union of all message part types.
type Part struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionID"`
	MessageID string          `json:"messageID"`
	Type      string          `json:"type"`
	Text      *string         `json:"text,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Mime      *string         `json:"mime,omitempty"`
	Filename  *string         `json:"filename,omitempty"`
	URL       *string         `json:"url,omitempty"`
	Source    json.RawMessage `json:"source,omitempty"`
	CallID    *string         `json:"callID,omitempty"`
	Tool      *string         `json:"tool,omitempty"`
	State     *ToolState      `json:"state,omitempty"`
	Reason    *string         `json:"reason,omitempty"`
	Cost      *float64        `json:"cost,omitempty"`
	Tokens    json.RawMessage `json:"tokens,omitempty"`
	Name      *string         `json:"name,omitempty"`
	Prompt    *string         `json:"prompt,omitempty"`
	Description *string       `json:"description,omitempty"`
	Agent     *string         `json:"agent,omitempty"`
	Attempt   *int            `json:"attempt,omitempty"`
	Error     json.RawMessage `json:"error,omitempty"`
	Auto      *bool           `json:"auto,omitempty"`
	Snapshot  *string         `json:"snapshot,omitempty"`
	Hash      *string         `json:"hash,omitempty"`
	Files     []string        `json:"files,omitempty"`
	Time      json.RawMessage `json:"time,omitempty"`
}

// ToolState is the current state of a tool call.
type ToolState struct {
	Status      string          `json:"status"`
	Input       json.RawMessage `json:"input"`
	Raw         *string         `json:"raw,omitempty"`
	Title       *string         `json:"title,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Time        json.RawMessage `json:"time,omitempty"`
	Output      *string         `json:"output,omitempty"`
	Attachments []Part          `json:"attachments,omitempty"`
	Error       *string         `json:"error,omitempty"`
}

// SessionStatus is the discriminated session lifecycle state.
type SessionStatus struct {
	Type    string  `json:"type"`
	Attempt *int    `json:"attempt,omitempty"`
	Message *string `json:"message,omitempty"`
	Next    *int64  `json:"next,omitempty"`
}

// IsIdle reports whether the session has finished processing.
func (s SessionStatus) IsIdle() bool { return s.Type == "idle" }

// Permission represents a pending permission gate.
type Permission struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Pattern   json.RawMessage `json:"pattern,omitempty"`
	SessionID string          `json:"sessionID"`
	MessageID string          `json:"messageID"`
	CallID    *string         `json:"callID,omitempty"`
	Title     string          `json:"title"`
	Metadata  json.RawMessage `json:"metadata"`
	Time      PermissionTime  `json:"time"`
}

type PermissionTime struct {
	Created int64 `json:"created"`
}

// Todo mirrors OpenCode todo item.
type Todo struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// GlobalEvent is the SSE envelope from GET /global/event.
type GlobalEvent struct {
	Directory string   `json:"directory"`
	Payload   RawEvent `json:"payload"`
}

// RawEvent holds an SSE event before type-switching.
type RawEvent struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

const (
	EventTypeServerConnected             = "server.connected"
	EventTypeServerInstanceDisposed      = "server.instance.disposed"
	EventTypeInstallationUpdated         = "installation.updated"
	EventTypeInstallationUpdateAvailable = "installation.update-available"
	EventTypeLspClientDiagnostics        = "lsp.client.diagnostics"
	EventTypeLspUpdated                  = "lsp.updated"
	EventTypeMessageUpdated              = "message.updated"
	EventTypeMessageRemoved              = "message.removed"
	EventTypeMessagePartUpdated          = "message.part.updated"
	EventTypeMessagePartRemoved          = "message.part.removed"
	EventTypePermissionUpdated           = "permission.updated"
	EventTypePermissionReplied           = "permission.replied"
	EventTypeSessionStatus               = "session.status"
	EventTypeSessionIdle                 = "session.idle"
	EventTypeSessionCompacted            = "session.compacted"
	EventTypeSessionCreated              = "session.created"
	EventTypeSessionUpdated              = "session.updated"
	EventTypeSessionDeleted              = "session.deleted"
	EventTypeSessionDiff                 = "session.diff"
	EventTypeSessionError                = "session.error"
	EventTypeFileEdited                  = "file.edited"
	EventTypeFileWatcherUpdated          = "file.watcher.updated"
	EventTypeTodoUpdated                 = "todo.updated"
	EventTypeCommandExecuted             = "command.executed"
	EventTypeVcsBranchUpdated            = "vcs.branch.updated"
	EventTypeTuiPromptAppend             = "tui.prompt.append"
	EventTypeTuiCommandExecute           = "tui.command.execute"
	EventTypeTuiToastShow                = "tui.toast.show"
	EventTypePtyCreated                  = "pty.created"
	EventTypePtyUpdated                  = "pty.updated"
	EventTypePtyExited                   = "pty.exited"
	EventTypePtyDeleted                  = "pty.deleted"
)

type EventPropsServerConnected struct{}
type EventPropsServerInstanceDisposed struct {
	Directory string `json:"directory"`
}
type EventPropsSessionStatus struct {
	SessionID string        `json:"sessionID"`
	Status    SessionStatus `json:"status"`
}
type EventPropsSessionIdle struct{ SessionID string `json:"sessionID"` }
type EventPropsSessionCompacted struct{ SessionID string `json:"sessionID"` }
type EventPropsSessionCreated struct{ Info Session `json:"info"` }
type EventPropsSessionUpdated struct{ Info Session `json:"info"` }
type EventPropsSessionDeleted struct{ Info Session `json:"info"` }
type EventPropsSessionDiff struct {
	SessionID string     `json:"sessionID"`
	Diff      []FileDiff `json:"diff"`
}
type EventPropsSessionError struct {
	SessionID *string         `json:"sessionID,omitempty"`
	Error     json.RawMessage `json:"error,omitempty"`
}
type EventPropsMessageUpdated struct{ Info Message `json:"info"` }
type EventPropsMessageRemoved struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
}
type EventPropsMessagePartUpdated struct {
	Part  Part    `json:"part"`
	Delta *string `json:"delta,omitempty"`
}
type EventPropsMessagePartRemoved struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
}
type EventPropsPermissionUpdated = Permission
type EventPropsPermissionReplied struct {
	SessionID    string `json:"sessionID"`
	PermissionID string `json:"permissionID"`
	Response     string `json:"response"`
}
type EventPropsTodoUpdated struct {
	SessionID string `json:"sessionID"`
	Todos     []Todo `json:"todos"`
}
type EventPropsFileEdited struct{ File string `json:"file"` }
type EventPropsCommandExecuted struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionID"`
	Arguments string `json:"arguments"`
	MessageID string `json:"messageID"`
}

// HealthResponse is returned by GET /global/health.
type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version"`
}

// SessionCreateRequest is the body for POST /session.
type SessionCreateRequest struct {
	Directory *string `json:"directory,omitempty"`
	ParentID  *string `json:"parentID,omitempty"`
}

// PromptRequest is the body for POST /session/:id/prompt_async.
type PromptRequest struct {
	Parts      []PromptPart `json:"parts"`
	ModelID    *string      `json:"modelID,omitempty"`
	ProviderID *string      `json:"providerID,omitempty"`
}

// PromptPart is a single input part of a prompt.
type PromptPart struct {
	ID          *string         `json:"id,omitempty"`
	Type        string          `json:"type"`
	Text        *string         `json:"text,omitempty"`
	Synthetic   *bool           `json:"synthetic,omitempty"`
	Ignored     *bool           `json:"ignored,omitempty"`
	Time        json.RawMessage `json:"time,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Mime        *string         `json:"mime,omitempty"`
	Filename    *string         `json:"filename,omitempty"`
	URL         *string         `json:"url,omitempty"`
	Source      json.RawMessage `json:"source,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Prompt      *string         `json:"prompt,omitempty"`
	Description *string         `json:"description,omitempty"`
	Agent       *string         `json:"agent,omitempty"`
}

// TextPromptPart constructs a plain-text prompt part.
func TextPromptPart(text string) PromptPart {
	return PromptPart{Type: "text", Text: &text}
}

// PermissionResponseRequest is the body for POST /session/:id/permission/:permID.
type PermissionResponseRequest struct {
	Response string `json:"response"`
}

// JSONLEvent is a single event from "opencode run --format json".
type JSONLEvent struct {
	Type          string          `json:"type"`
	StepType      *string         `json:"step_type,omitempty"`
	ID            *string         `json:"id,omitempty"`
	Name          *string         `json:"name,omitempty"`
	Input         json.RawMessage `json:"input,omitempty"`
	Status        *string         `json:"status,omitempty"`
	Output        *string         `json:"output,omitempty"`
	Text          *string         `json:"text,omitempty"`
	FinishReason  *string         `json:"finish_reason,omitempty"`
	Usage         json.RawMessage `json:"usage,omitempty"`
	ProviderUsage json.RawMessage `json:"provider_usage,omitempty"`
	Message       *string         `json:"message,omitempty"`
}
