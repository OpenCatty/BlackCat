package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// TaskDef defines a scheduled task. Tasks are config-defined only (G10);
// dynamic creation from external input is not permitted.
type TaskDef struct {
	Name     string
	Schedule string // cron expression or "@every 30s"
	Handler  func(ctx context.Context) error
	Enabled  bool
}

// TaskState records execution state for a registered task.
type TaskState struct {
	Name       string    `json:"name"`
	LastRun    time.Time `json:"last_run"`
	LastStatus string    `json:"last_status"` // "ok", "failed", "skipped", "running"
	NextRun    time.Time `json:"next_run"`
	RunCount   int       `json:"run_count"`
	LastError  string    `json:"last_error"`
}

// Scheduler wraps robfig/cron/v3 with state persistence, SkipIfStillRunning
// semantics, and panic recovery per task.
type Scheduler struct {
	cron           *cron.Cron
	stateFile      string
	mu             sync.RWMutex
	states         map[string]*TaskState
	entryIDs       map[string]cron.EntryID
	onTaskComplete func(name string, err error)
}

// New creates a Scheduler that persists task state to stateFile.
// If stateFile already exists, previous state is restored.
func New(stateFile string) *Scheduler {
	s := &Scheduler{
		cron: cron.New(
			cron.WithSeconds(),
			cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
		),
		stateFile: stateFile,
		states:    make(map[string]*TaskState),
		entryIDs:  make(map[string]cron.EntryID),
	}
	s.loadState()
	return s
}

// Register adds a task definition to the scheduler. The task is only added
// if Enabled is true. Returns an error if the cron expression is invalid
// or the task name is already registered.
func (s *Scheduler) Register(def TaskDef) error {
	if !def.Enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.entryIDs[def.Name]; exists {
		return fmt.Errorf("scheduler: task %q already registered", def.Name)
	}

	// Ensure state entry exists (may have been loaded from file).
	if _, ok := s.states[def.Name]; !ok {
		s.states[def.Name] = &TaskState{Name: def.Name}
	}

	handler := s.wrapHandler(def.Name, def.Handler)
	id, err := s.cron.AddFunc(def.Schedule, handler)
	if err != nil {
		return fmt.Errorf("scheduler: invalid schedule for %q: %w", def.Name, err)
	}

	s.entryIDs[def.Name] = id

	// Set initial NextRun from cron entry.
	entry := s.cron.Entry(id)
	if !entry.Next.IsZero() {
		s.states[def.Name].NextRun = entry.Next
	}

	return nil
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop halts the scheduler and waits for running jobs to finish.
func (s *Scheduler) Stop(_ context.Context) {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// ListTasks returns a snapshot of all task states.
func (s *Scheduler) ListTasks() []TaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]TaskState, 0, len(s.states))
	for _, st := range s.states {
		out = append(out, *st)
	}
	return out
}

// GetTask returns the state for the named task, or nil if not found.
func (s *Scheduler) GetTask(name string) *TaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.states[name]
	if !ok {
		return nil
	}
	cp := *st
	return &cp
}

// wrapHandler returns a cron-compatible func that executes the task handler
// with panic recovery and state bookkeeping.
func (s *Scheduler) wrapHandler(name string, handler func(ctx context.Context) error) func() {
	return func() {
		s.mu.Lock()
		st := s.states[name]
		st.LastStatus = "running"
		s.mu.Unlock()

		var runErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					runErr = fmt.Errorf("panic: %v", r)
				}
			}()
			runErr = handler(context.Background())
		}()

		now := time.Now()
		s.mu.Lock()
		st.LastRun = now
		st.RunCount++
		if runErr != nil {
			st.LastStatus = "failed"
			st.LastError = runErr.Error()
		} else {
			st.LastStatus = "ok"
			st.LastError = ""
		}

		// Update NextRun from cron entry.
		if id, ok := s.entryIDs[name]; ok {
			entry := s.cron.Entry(id)
			if !entry.Next.IsZero() {
				st.NextRun = entry.Next
			}
		}
		s.mu.Unlock()

		s.persistState()

		if s.onTaskComplete != nil {
			s.onTaskComplete(name, runErr)
		}
	}
}

// SetOnTaskComplete registers a callback invoked after each task completes.
// It is called with the task name and any error (nil if successful).
func (s *Scheduler) SetOnTaskComplete(fn func(name string, err error)) {
	s.mu.Lock()
	s.onTaskComplete = fn
	s.mu.Unlock()
}

// loadState reads task state from the state file, if it exists.
func (s *Scheduler) loadState() {
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		return // file doesn't exist or unreadable — start fresh
	}

	var states []*TaskState
	if err := json.Unmarshal(data, &states); err != nil {
		return // corrupt file — start fresh
	}

	for _, st := range states {
		s.states[st.Name] = st
	}
}

// persistState writes all task states to the state file atomically
// (write-to-temp + rename), following the memory/store.go pattern.
func (s *Scheduler) persistState() {
	s.mu.RLock()
	states := make([]*TaskState, 0, len(s.states))
	for _, st := range s.states {
		cp := *st
		states = append(states, &cp)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return // should not happen with simple structs
	}

	dir := filepath.Dir(s.stateFile)
	base := filepath.Base(s.stateFile)
	tmpPath := filepath.Join(dir, "."+base+".tmp")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return
	}

	if err := os.Rename(tmpPath, s.stateFile); err != nil {
		os.Remove(tmpPath)
	}
}
