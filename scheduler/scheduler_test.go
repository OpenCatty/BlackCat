package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskRegistration(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	s := New(stateFile)

	err := s.Register(TaskDef{
		Name:     "test-task",
		Schedule: "@every 1h",
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	tasks := s.ListTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "test-task" {
		t.Errorf("expected name %q, got %q", "test-task", tasks[0].Name)
	}

	// Disabled tasks should not register.
	err = s.Register(TaskDef{
		Name:     "disabled-task",
		Schedule: "@every 1h",
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  false,
	})
	if err != nil {
		t.Fatalf("Register disabled task returned error: %v", err)
	}
	tasks = s.ListTasks()
	if len(tasks) != 1 {
		t.Fatalf("disabled task should not appear; got %d tasks", len(tasks))
	}

	// Duplicate name should error.
	err = s.Register(TaskDef{
		Name:     "test-task",
		Schedule: "@every 1h",
		Handler:  func(ctx context.Context) error { return nil },
		Enabled:  true,
	})
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestStatePersistence(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")

	// First scheduler: register and run a task.
	s1 := New(stateFile)
	ran := make(chan struct{}, 1)
	err := s1.Register(TaskDef{
		Name:     "persist-task",
		Schedule: "@every 50ms",
		Handler: func(ctx context.Context) error {
			select {
			case ran <- struct{}{}:
			default:
			}
			return nil
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	s1.Start()

	// Wait for at least one execution.
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("task did not run within timeout")
	}

	s1.Stop(context.Background())

	// Verify state file exists.
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// Second scheduler: load state from file.
	s2 := New(stateFile)
	st := s2.GetTask("persist-task")
	if st == nil {
		t.Fatal("task state not loaded from file")
	}
	if st.RunCount < 1 {
		t.Errorf("expected RunCount >= 1, got %d", st.RunCount)
	}
	if st.LastStatus != "ok" {
		t.Errorf("expected LastStatus %q, got %q", "ok", st.LastStatus)
	}
	if st.LastRun.IsZero() {
		t.Error("LastRun should not be zero")
	}
}

func TestPanicRecovery(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	s := New(stateFile)

	panicRan := make(chan struct{}, 1)
	err := s.Register(TaskDef{
		Name:     "panic-task",
		Schedule: "@every 50ms",
		Handler: func(ctx context.Context) error {
			select {
			case panicRan <- struct{}{}:
			default:
			}
			panic("boom")
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	s.Start()

	// Wait for panic task to execute.
	select {
	case <-panicRan:
	case <-time.After(2 * time.Second):
		t.Fatal("panic task did not run within timeout")
	}

	// Give the handler wrapper time to update state.
	time.Sleep(100 * time.Millisecond)

	st := s.GetTask("panic-task")
	if st == nil {
		t.Fatal("task state not found")
	}
	if st.LastStatus != "failed" {
		t.Errorf("expected LastStatus %q after panic, got %q", "failed", st.LastStatus)
	}
	if st.LastError == "" {
		t.Error("expected LastError to contain panic message")
	}

	s.Stop(context.Background())
}

func TestSkipIfRunning(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	s := New(stateFile)

	var runCount atomic.Int32
	started := make(chan struct{}, 1)
	hold := make(chan struct{})

	err := s.Register(TaskDef{
		Name:     "slow-task",
		Schedule: "@every 50ms",
		Handler: func(ctx context.Context) error {
			count := runCount.Add(1)
			if count == 1 {
				// Signal that first execution started.
				select {
				case started <- struct{}{}:
				default:
				}
				// Block to simulate slow task.
				<-hold
			}
			return nil
		},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	s.Start()

	// Wait for first execution to start.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("slow task did not start within timeout")
	}

	// Wait enough time for several scheduled ticks to be skipped.
	time.Sleep(200 * time.Millisecond)

	// Release the blocked first execution.
	close(hold)

	// Give time for any queued executions.
	time.Sleep(100 * time.Millisecond)

	s.Stop(context.Background())

	// With SkipIfStillRunning, the concurrent ticks while the first was
	// blocked should have been skipped. We expect very few runs.
	total := runCount.Load()
	if total > 3 {
		t.Errorf("expected SkipIfStillRunning to limit runs; got %d", total)
	}
}
