package taskqueue

import (
	"fmt"
	"strings"
	"time"
)

// statusEmoji maps task status strings to their WhatsApp-friendly emoji.
var statusEmoji = map[string]string{
	StatusPending:    "⏰",
	StatusInProgress: "⏳",
	StatusCompleted:  "✅",
	StatusFailed:     "❌",
}

// FormatTaskStatus returns a WhatsApp-friendly status summary for a single task.
func FormatTaskStatus(task *Task) string {
	emoji := statusEmoji[task.Status]
	if emoji == "" {
		emoji = "❓"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "📋 Task #%d\n", task.ID)
	fmt.Fprintf(&b, "Type: %s\n", task.TaskType)
	fmt.Fprintf(&b, "Status: %s %s\n", emoji, task.Status)
	fmt.Fprintf(&b, "Started: %s\n", relativeTime(task.CreatedAt))

	if task.CompletedAt != nil {
		dur := task.CompletedAt.Sub(task.CreatedAt)
		fmt.Fprintf(&b, "Duration: %s", formatDuration(dur))
	}

	return b.String()
}

// FormatTaskList returns a WhatsApp-friendly summary of multiple tasks.
func FormatTaskList(tasks []Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📊 Task Queue (%d tasks)\n", len(tasks))

	for i := range tasks {
		t := &tasks[i]
		emoji := statusEmoji[t.Status]
		if emoji == "" {
			emoji = "❓"
		}
		fmt.Fprintf(&b, "\n• #%d %s — %s %s", t.ID, t.TaskType, emoji, t.Status)
	}

	return b.String()
}

// FormatCompletion returns a WhatsApp-friendly completion notification.
func FormatCompletion(task *Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✅ Task #%d completed\n", task.ID)
	fmt.Fprintf(&b, "\n%s finished successfully.\n", task.TaskType)

	if task.CompletedAt != nil {
		dur := task.CompletedAt.Sub(task.CreatedAt)
		fmt.Fprintf(&b, "\n• Duration: %s", formatDuration(dur))
	}

	preview := truncate(task.Result, 200)
	if preview != "" {
		fmt.Fprintf(&b, "\n• Result preview: %s", preview)
	}

	return b.String()
}

// FormatError returns a WhatsApp-friendly error notification.
func FormatError(task *Task) string {
	var b strings.Builder
	fmt.Fprintf(&b, "❌ Task #%d failed\n", task.ID)
	fmt.Fprintf(&b, "\n%s encountered an error.\n", task.TaskType)

	errMsg := truncate(task.Error, 300)
	if errMsg != "" {
		fmt.Fprintf(&b, "\n• Error: %s", errMsg)
	}

	if task.CompletedAt != nil {
		dur := task.CompletedAt.Sub(task.CreatedAt)
		fmt.Fprintf(&b, "\n• Duration: %s", formatDuration(dur))
	}

	return b.String()
}

// nowFunc is overridable in tests to freeze time.
var nowFunc = time.Now

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	d := nowFunc().Sub(t)
	if d < 0 {
		d = -d
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < 2*time.Minute:
		return "1 minute ago"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 2*time.Hour:
		return "1 hour ago"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "1 day ago"
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}

// formatDuration returns a compact duration string like "2m 15s" or "1h 3m".
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh 0m", h)
	case m > 0 && s > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	case m > 0:
		return fmt.Sprintf("%dm 0s", m)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
