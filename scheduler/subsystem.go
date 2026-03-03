package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/startower-observability/blackcat/config"
	"github.com/startower-observability/blackcat/internal/daemon"
)

type SchedulerSubsystem struct {
	scheduler      *Scheduler
	cfg            config.SchedulerConfig
	heartbeatStore *HeartbeatStore
	checker        SubsystemChecker
	executor       JobExecutor
	reconnector    ChannelReconnector

	mu      sync.RWMutex
	status  string
	message string
	started bool
}

func NewSchedulerSubsystem(cfg config.SchedulerConfig) *SchedulerSubsystem {
	return &SchedulerSubsystem{
		scheduler:      New(defaultStateFile()),
		cfg:            cfg,
		heartbeatStore: NewHeartbeatStore(defaultHeartbeatCapacity),
		status:         "stopped",
		message:        "not started",
	}
}

// WithChecker sets the subsystem checker used by the heartbeat task.
func (s *SchedulerSubsystem) WithChecker(checker SubsystemChecker) {
	s.mu.Lock()
	s.checker = checker
	s.mu.Unlock()
}

// WithExecutor sets the job executor for scheduled commands.
func (s *SchedulerSubsystem) WithExecutor(executor JobExecutor) {
	s.mu.Lock()
	s.executor = executor
	s.mu.Unlock()
}

// WithReconnector sets the channel reconnector for heartbeat auto-recovery.
func (s *SchedulerSubsystem) WithReconnector(r ChannelReconnector) {
	s.mu.Lock()
	s.reconnector = r
	s.mu.Unlock()
}

// HeartbeatStore returns the heartbeat result store for dashboard consumption.
func (s *SchedulerSubsystem) HeartbeatStore() *HeartbeatStore {
	return s.heartbeatStore
}

// ListTasks returns a snapshot of all task states for dashboard display.
func (s *SchedulerSubsystem) ListTasks() []TaskState {
	return s.scheduler.ListTasks()
}

// SetOnTaskComplete registers a callback invoked after each task completes.
// It is called with the task name and any error (nil if successful).
func (s *SchedulerSubsystem) SetOnTaskComplete(fn func(name string, err error)) {
	s.scheduler.SetOnTaskComplete(fn)
}

func (s *SchedulerSubsystem) Name() string {
	return "scheduler"
}

func (s *SchedulerSubsystem) Start(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	if !s.cfg.Enabled {
		s.status = "stopped"
		s.message = "disabled"
		s.mu.Unlock()
		return nil
	}
	if s.status == "running" {
		s.mu.Unlock()
		return nil
	}
	s.status = "starting"
	s.message = "registering tasks"
	needsRegistration := !s.started
	s.mu.Unlock()

	if needsRegistration {
		// Register heartbeat task
		s.mu.RLock()
		checker := s.checker
		store := s.heartbeatStore
		reconnector := s.reconnector
		s.mu.RUnlock()
		heartbeatTask := NewHeartbeatTask(checker, store)
		if reconnector != nil {
			heartbeatTask.SetReconnector(reconnector)
		}
		if err := s.scheduler.Register(BuildHeartbeatTaskDef(heartbeatTask)); err != nil {
			err = fmt.Errorf("scheduler: register heartbeat task: %w", err)
			s.setDegraded(err)
			return err
		}

		for _, job := range s.cfg.Jobs {
			if !job.Enabled {
				continue
			}
			if job.Name == "" {
				err := fmt.Errorf("scheduler: job name is required")
				s.setDegraded(err)
				return err
			}
			if job.Schedule == "" {
				err := fmt.Errorf("scheduler: schedule is required for job %q", job.Name)
				s.setDegraded(err)
				return err
			}

			executor := s.executor
			if executor == nil {
				executor = &ShellExecutor{}
			}
			jobCopy := job // capture for closure
			if err := s.scheduler.Register(TaskDef{
				Name:     job.Name,
				Schedule: job.Schedule,
				Handler: func(ctx context.Context) error {
					return executor.Execute(ctx, jobCopy)
				},
				Enabled:  job.Enabled,
			}); err != nil {
				err = fmt.Errorf("scheduler: register job %q: %w", job.Name, err)
				s.setDegraded(err)
				return err
			}
		}

		s.mu.Lock()
		s.started = true
		s.mu.Unlock()
	}

	s.scheduler.Start()

	taskCount := len(s.scheduler.ListTasks())
	s.mu.Lock()
	s.status = "running"
	s.message = fmt.Sprintf("running; tasks=%d", taskCount)
	s.mu.Unlock()

	return nil
}

func (s *SchedulerSubsystem) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.status != "running" {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	s.scheduler.Stop(ctx)

	s.mu.Lock()
	s.status = "stopped"
	s.message = "stopped"
	s.mu.Unlock()

	return nil
}

func (s *SchedulerSubsystem) Health() daemon.SubsystemHealth {
	s.mu.RLock()
	status := s.status
	message := s.message
	s.mu.RUnlock()

	taskCount := len(s.scheduler.ListTasks())
	if message == "" {
		message = fmt.Sprintf("tasks=%d", taskCount)
	} else {
		message = fmt.Sprintf("%s; tasks=%d", message, taskCount)
	}

	return daemon.SubsystemHealth{
		Name:    s.Name(),
		Status:  status,
		Message: message,
	}
}

func (s *SchedulerSubsystem) setDegraded(err error) {
	s.mu.Lock()
	s.status = "degraded"
	s.message = err.Error()
	s.mu.Unlock()
}

func defaultStateFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "blackcat", "scheduler-state.json")
	}

	return filepath.Join(home, ".blackcat", "scheduler-state.json")
}
