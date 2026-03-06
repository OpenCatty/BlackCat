package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/startower-observability/blackcat/internal/opencode"
	"github.com/startower-observability/blackcat/internal/taskqueue"
)

const (
	openCodeTaskAsyncName        = "opencode_task_async"
	openCodeTaskAsyncDescription = "Enqueue a coding task to run in the background via OpenCode. Returns immediately with a task ID. Use check_opencode_status to monitor progress. Best for long-running tasks where you don't want to block."
)

var openCodeTaskAsyncParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"prompt": {
			"type": "string",
			"description": "The task to perform"
		},
		"dir": {
			"type": "string",
			"description": "Project directory (REQUIRED). Must be absolute path."
		},
		"recipient_id": {
			"type": "string",
			"description": "Optional WhatsApp number to notify on completion (e.g. +628xxx)"
		}
	},
	"required": ["prompt", "dir"]
}`)

// openCodeTaskPayload is the JSON structure stored in the task queue payload.
type openCodeTaskPayload struct {
	Prompt string `json:"prompt"`
	Dir    string `json:"dir"`
}

// OpenCodeTaskAsyncTool enqueues coding tasks for background execution via the task queue.
type OpenCodeTaskAsyncTool struct {
	client     *opencode.Client
	queue      *taskqueue.TaskQueue
	autoPermit bool
	timeout    time.Duration
}

// NewOpenCodeTaskAsyncTool creates an OpenCodeTaskAsyncTool and registers the
// "opencode_task" handler with the task queue.
func NewOpenCodeTaskAsyncTool(client *opencode.Client, queue *taskqueue.TaskQueue, autoPermit bool, timeout time.Duration) *OpenCodeTaskAsyncTool {
	if timeout <= 0 {
		timeout = defaultOpenCodeTimeout
	}
	t := &OpenCodeTaskAsyncTool{
		client:     client,
		queue:      queue,
		autoPermit: autoPermit,
		timeout:    timeout,
	}
	queue.RegisterHandler("opencode_task", t.executeOpenCodeTask)
	return t
}

func (t *OpenCodeTaskAsyncTool) Name() string                { return openCodeTaskAsyncName }
func (t *OpenCodeTaskAsyncTool) Description() string         { return openCodeTaskAsyncDescription }
func (t *OpenCodeTaskAsyncTool) Parameters() json.RawMessage { return openCodeTaskAsyncParameters }

// Execute enqueues the task and returns immediately with the task ID.
func (t *OpenCodeTaskAsyncTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Prompt      string `json:"prompt"`
		Dir         string `json:"dir"`
		RecipientID string `json:"recipient_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return fmt.Sprintf("error: invalid arguments: %s", err), nil
	}
	if params.Prompt == "" {
		return "error: prompt is required", nil
	}
	if params.Dir == "" {
		return "error: dir is required", nil
	}

	payload, err := json.Marshal(openCodeTaskPayload{
		Prompt: params.Prompt,
		Dir:    params.Dir,
	})
	if err != nil {
		return fmt.Sprintf("error: failed to marshal payload: %s", err), nil
	}

	id, err := t.queue.Enqueue(taskqueue.Task{
		TaskType:    "opencode_task",
		Payload:     string(payload),
		RecipientID: params.RecipientID,
	})
	if err != nil {
		return fmt.Sprintf("error: failed to enqueue task: %s", err), nil
	}

	return fmt.Sprintf("Task enqueued with ID: %d. Use check_opencode_status to monitor progress.", id), nil
}

// executeOpenCodeTask is the background handler invoked by the task queue worker.
func (t *OpenCodeTaskAsyncTool) executeOpenCodeTask(ctx context.Context, payload string) (string, error) {
	var p openCodeTaskPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", fmt.Errorf("unmarshal payload: %w", err)
	}

	sessionMgr := opencode.NewSessionManager(t.client)

	req := opencode.TaskRequest{
		Prompt:     p.Prompt,
		Dir:        p.Dir,
		AutoPermit: t.autoPermit,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	result, err := sessionMgr.Run(timeoutCtx, req)
	if err != nil {
		return "", fmt.Errorf("opencode task failed: %w", err)
	}

	// Extract the last assistant message.
	assistantContent := "(no assistant response)"
	for i := len(result.Messages) - 1; i >= 0; i-- {
		if result.Messages[i].Info.Role == "assistant" {
			assistantContent = extractAsyncMessageContent(result.Messages[i])
			break
		}
	}

	return fmt.Sprintf("OpenCode Task Complete\nSession: %s\nMessages: %d\n\nAssistant Response:\n%s",
		result.SessionID,
		len(result.Messages),
		assistantContent,
	), nil
}

// extractAsyncMessageContent returns a human-readable summary of a message.
func extractAsyncMessageContent(msg opencode.MessageWithParts) string {
	var texts []string
	for _, p := range msg.Parts {
		if p.Type == "text" && p.Text != nil && *p.Text != "" {
			texts = append(texts, *p.Text)
		}
	}
	if len(texts) > 0 {
		return strings.Join(texts, "\n")
	}
	agent := msg.Info.Agent
	if agent == "" {
		agent = "assistant"
	}
	return fmt.Sprintf("[%s] message %s (no text parts)", agent, msg.Info.ID)
}
