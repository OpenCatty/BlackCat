//go:build cgo

package taskqueue

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// mockSender captures notification calls for test assertions.
type mockSender struct {
	mu         sync.Mutex
	messages   []string
	recipients []string
}

func (m *mockSender) Send(_ context.Context, recipientID, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
	m.recipients = append(m.recipients, recipientID)
	return nil
}

func (m *mockSender) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func TestNotification_CompletionSent(t *testing.T) {
	q := newTestQueue(t)
	sender := &mockSender{}
	q.SetNotificationSender(sender)

	q.RegisterHandler("notify_task", func(_ context.Context, payload string) (string, error) {
		return "done: " + payload, nil
	})

	id, err := q.Enqueue(Task{
		TaskType:    "notify_task",
		Payload:     "test-payload",
		RecipientID: "user-abc",
	})
	if err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	q.Start(context.Background())

	// Wait for task to complete.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := q.GetTask(id)
		if err != nil {
			t.Fatalf("GetTask() error: %v", err)
		}
		if task.Status == StatusCompleted || task.Status == StatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Allow goroutine to deliver the notification.
	time.Sleep(200 * time.Millisecond)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(sender.messages))
	}
	if sender.recipients[0] != "user-abc" {
		t.Errorf("recipient: got %q, want %q", sender.recipients[0], "user-abc")
	}
	if len(sender.messages[0]) == 0 {
		t.Error("notification message should not be empty")
	}
}

func TestNotification_FailureSent(t *testing.T) {
	q := newTestQueue(t)
	sender := &mockSender{}
	q.SetNotificationSender(sender)

	q.RegisterHandler("fail_notify", func(_ context.Context, _ string) (string, error) {
		return "", os.ErrPermission
	})

	id, err := q.Enqueue(Task{
		TaskType:    "fail_notify",
		Payload:     "data",
		RecipientID: "user-xyz",
	})
	if err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	q.Start(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := q.GetTask(id)
		if err != nil {
			t.Fatalf("GetTask() error: %v", err)
		}
		if task.Status == StatusCompleted || task.Status == StatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	sender.mu.Lock()
	defer sender.mu.Unlock()

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(sender.messages))
	}
	if sender.recipients[0] != "user-xyz" {
		t.Errorf("recipient: got %q, want %q", sender.recipients[0], "user-xyz")
	}
	if len(sender.messages[0]) == 0 {
		t.Error("notification message should not be empty")
	}
}

func TestNotification_NoRecipient_NoNotification(t *testing.T) {
	q := newTestQueue(t)
	sender := &mockSender{}
	q.SetNotificationSender(sender)

	q.RegisterHandler("silent_task", func(_ context.Context, payload string) (string, error) {
		return "ok", nil
	})

	// No RecipientID set.
	id, err := q.Enqueue(Task{
		TaskType: "silent_task",
		Payload:  "data",
	})
	if err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	q.Start(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := q.GetTask(id)
		if err != nil {
			t.Fatalf("GetTask() error: %v", err)
		}
		if task.Status == StatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	if sender.count() != 0 {
		t.Errorf("expected 0 notifications for task without RecipientID, got %d", sender.count())
	}
}

func TestNotification_NoSender_NoNotification(t *testing.T) {
	q := newTestQueue(t)
	// No sender registered — should not panic.

	q.RegisterHandler("no_sender_task", func(_ context.Context, payload string) (string, error) {
		return "ok", nil
	})

	id, err := q.Enqueue(Task{
		TaskType:    "no_sender_task",
		Payload:     "data",
		RecipientID: "user-123",
	})
	if err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	q.Start(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := q.GetTask(id)
		if err != nil {
			t.Fatalf("GetTask() error: %v", err)
		}
		if task.Status == StatusCompleted {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// If we get here without panic, the test passes.
}
