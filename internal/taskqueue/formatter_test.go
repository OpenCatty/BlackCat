package taskqueue

import (
	"strings"
	"testing"
	"time"
)

// frozenNow is used by tests to fix time.
var frozenNow = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

func init() {
	nowFunc = func() time.Time { return frozenNow }
}

func TestFormatTaskStatus_Completed(t *testing.T) {
	completed := frozenNow.Add(-3 * time.Minute)
	task := &Task{
		ID:        42,
		TaskType:  "opencode_task",
		Status:    StatusCompleted,
		CreatedAt: frozenNow.Add(-5*time.Minute - 15*time.Second),
		CompletedAt: func() *time.Time {
			t := completed
			return &t
		}(),
	}

	got := FormatTaskStatus(task)

	assertContains(t, got, "📋 Task #42")
	assertContains(t, got, "Type: opencode_task")
	assertContains(t, got, "Status: ✅ completed")
	assertContains(t, got, "Started: 5 minutes ago")
	assertContains(t, got, "Duration: 2m 15s")
}

func TestFormatTaskStatus_Pending(t *testing.T) {
	task := &Task{
		ID:        1,
		TaskType:  "web",
		Status:    StatusPending,
		CreatedAt: frozenNow.Add(-30 * time.Second),
	}

	got := FormatTaskStatus(task)

	assertContains(t, got, "Status: ⏰ pending")
	assertContains(t, got, "Started: just now")
	assertNotContains(t, got, "Duration:")
}

func TestFormatTaskStatus_InProgress(t *testing.T) {
	task := &Task{
		ID:        7,
		TaskType:  "opencode_task",
		Status:    StatusInProgress,
		CreatedAt: frozenNow.Add(-1 * time.Minute),
	}

	got := FormatTaskStatus(task)
	assertContains(t, got, "Status: ⏳ in_progress")
	assertContains(t, got, "Started: 1 minute ago")
}

func TestFormatTaskStatus_Failed(t *testing.T) {
	task := &Task{
		ID:        9,
		TaskType:  "web",
		Status:    StatusFailed,
		CreatedAt: frozenNow.Add(-2 * time.Hour),
	}

	got := FormatTaskStatus(task)
	assertContains(t, got, "Status: ❌ failed")
	assertContains(t, got, "Started: 2 hours ago")
}

func TestFormatTaskList(t *testing.T) {
	completedAt := frozenNow.Add(-1 * time.Minute)
	tasks := []Task{
		{ID: 41, TaskType: "opencode_task", Status: StatusCompleted, CompletedAt: &completedAt},
		{ID: 42, TaskType: "opencode_task", Status: StatusInProgress},
		{ID: 43, TaskType: "web", Status: StatusFailed},
	}

	got := FormatTaskList(tasks)

	assertContains(t, got, "📊 Task Queue (3 tasks)")
	assertContains(t, got, "• #41 opencode_task — ✅ completed")
	assertContains(t, got, "• #42 opencode_task — ⏳ in_progress")
	assertContains(t, got, "• #43 web — ❌ failed")
}

func TestFormatTaskList_Empty(t *testing.T) {
	got := FormatTaskList(nil)
	assertContains(t, got, "📊 Task Queue (0 tasks)")
}

func TestFormatCompletion(t *testing.T) {
	completedAt := frozenNow
	task := &Task{
		ID:          42,
		TaskType:    "opencode_task",
		Status:      StatusCompleted,
		Result:      "All tests passed. 15 files changed.",
		CreatedAt:   frozenNow.Add(-2*time.Minute - 15*time.Second),
		CompletedAt: &completedAt,
	}

	got := FormatCompletion(task)

	assertContains(t, got, "✅ Task #42 completed")
	assertContains(t, got, "opencode_task finished successfully.")
	assertContains(t, got, "• Duration: 2m 15s")
	assertContains(t, got, "• Result preview: All tests passed. 15 files changed.")
}

func TestFormatCompletion_LongResult(t *testing.T) {
	completedAt := frozenNow
	task := &Task{
		ID:          10,
		TaskType:    "opencode_task",
		Status:      StatusCompleted,
		Result:      strings.Repeat("x", 250),
		CreatedAt:   frozenNow.Add(-10 * time.Second),
		CompletedAt: &completedAt,
	}

	got := FormatCompletion(task)

	assertContains(t, got, "• Result preview: "+strings.Repeat("x", 200)+"...")
	// Make sure full 250 chars are NOT present.
	if strings.Contains(got, strings.Repeat("x", 250)) {
		t.Error("result should be truncated to 200 chars")
	}
}

func TestFormatCompletion_EmptyResult(t *testing.T) {
	completedAt := frozenNow
	task := &Task{
		ID:          11,
		TaskType:    "web",
		Status:      StatusCompleted,
		CreatedAt:   frozenNow.Add(-5 * time.Second),
		CompletedAt: &completedAt,
	}

	got := FormatCompletion(task)
	assertNotContains(t, got, "Result preview:")
}

func TestFormatError(t *testing.T) {
	completedAt := frozenNow
	task := &Task{
		ID:          42,
		TaskType:    "opencode_task",
		Status:      StatusFailed,
		Error:       "timeout after 30s: context deadline exceeded",
		CreatedAt:   frozenNow.Add(-1*time.Minute - 30*time.Second),
		CompletedAt: &completedAt,
	}

	got := FormatError(task)

	assertContains(t, got, "❌ Task #42 failed")
	assertContains(t, got, "opencode_task encountered an error.")
	assertContains(t, got, "• Error: timeout after 30s: context deadline exceeded")
	assertContains(t, got, "• Duration: 1m 30s")
}

func TestFormatError_LongError(t *testing.T) {
	task := &Task{
		ID:        20,
		TaskType:  "web",
		Status:    StatusFailed,
		Error:     strings.Repeat("e", 350),
		CreatedAt: frozenNow,
	}

	got := FormatError(task)

	assertContains(t, got, "• Error: "+strings.Repeat("e", 300)+"...")
	if strings.Contains(got, strings.Repeat("e", 350)) {
		t.Error("error should be truncated to 300 chars")
	}
}

func TestFormatError_NoCompletedAt(t *testing.T) {
	task := &Task{
		ID:        30,
		TaskType:  "opencode_task",
		Status:    StatusFailed,
		Error:     "something broke",
		CreatedAt: frozenNow,
	}

	got := FormatError(task)
	assertNotContains(t, got, "Duration:")
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		offset time.Duration
		want   string
	}{
		{5 * time.Second, "just now"},
		{59 * time.Second, "just now"},
		{1*time.Minute + 30*time.Second, "1 minute ago"},
		{3 * time.Minute, "3 minutes ago"},
		{59 * time.Minute, "59 minutes ago"},
		{1*time.Hour + 30*time.Minute, "1 hour ago"},
		{5 * time.Hour, "5 hours ago"},
		{36 * time.Hour, "1 day ago"},
		{72 * time.Hour, "3 days ago"},
	}

	for _, tt := range tests {
		got := relativeTime(frozenNow.Add(-tt.offset))
		if got != tt.want {
			t.Errorf("relativeTime(-%v) = %q, want %q", tt.offset, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{1 * time.Minute, "1m 0s"},
		{2*time.Minute + 15*time.Second, "2m 15s"},
		{1 * time.Hour, "1h 0m"},
		{1*time.Hour + 30*time.Minute, "1h 30m"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is longer than ten", 10, "this is lo..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// TestNoMarkdown ensures no markdown formatting is used.
func TestNoMarkdown(t *testing.T) {
	completedAt := frozenNow
	task := &Task{
		ID:          42,
		TaskType:    "opencode_task",
		Status:      StatusCompleted,
		Result:      "done",
		CreatedAt:   frozenNow.Add(-1 * time.Minute),
		CompletedAt: &completedAt,
	}

	outputs := []string{
		FormatTaskStatus(task),
		FormatCompletion(task),
		FormatError(&Task{ID: 1, TaskType: "t", Status: StatusFailed, Error: "e", CreatedAt: frozenNow}),
		FormatTaskList([]Task{*task}),
	}

	for i, out := range outputs {
		if strings.Contains(out, "**") {
			t.Errorf("output[%d] contains markdown bold (**)", i)
		}
		if strings.Contains(out, "# ") {
			t.Errorf("output[%d] contains markdown heading (#)", i)
		}
	}
}

// assertContains fails the test if s does not contain substr.
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}

// assertNotContains fails the test if s contains substr.
func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, s)
	}
}
