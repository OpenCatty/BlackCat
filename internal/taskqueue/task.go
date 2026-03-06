// Package taskqueue provides a background async task queue with SQLite
// persistence and a configurable goroutine worker pool.
package taskqueue

import (
	"context"
	"time"
)

// Task status constants.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Task represents a unit of work persisted in SQLite.
type Task struct {
	ID          int64      `json:"id"`
	TaskType    string     `json:"task_type"`    // e.g. "opencode_task"
	Status      string     `json:"status"`       // pending, in_progress, completed, failed
	Payload     string     `json:"payload"`      // JSON params passed to handler
	Result      string     `json:"result"`       // JSON result from handler
	Error       string     `json:"error"`        // error message if failed
	RecipientID string     `json:"recipient_id"` // WA number to notify on completion
	RetryCount  int        `json:"retry_count"`  // number of retries attempted so far
	MaxRetries  int        `json:"max_retries"`  // max retry attempts (default 3)
	TimeoutSecs int        `json:"timeout_secs"` // per-execution timeout in seconds (default 1800)
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TaskHandler is a function that processes a task payload and returns
// a result string or an error.
type TaskHandler func(ctx context.Context, payload string) (string, error)

// Notification is emitted when a task reaches a terminal state.
type Notification struct {
	TaskID      int64  `json:"task_id"`
	TaskType    string `json:"task_type"`
	Status      string `json:"status"`
	RecipientID string `json:"recipient_id"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
}
