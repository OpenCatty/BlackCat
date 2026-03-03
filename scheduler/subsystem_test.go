package scheduler

import (
	"context"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/config"
)

func TestSubsystemName(t *testing.T) {
	subsystem := NewSchedulerSubsystem(config.SchedulerConfig{})

	if got, want := subsystem.Name(), "scheduler"; got != want {
		t.Fatalf("Name() = %q, want %q", got, want)
	}
}

func TestSubsystemLifecycle(t *testing.T) {
	subsystem := NewSchedulerSubsystem(config.SchedulerConfig{
		Enabled: true,
		Jobs: []config.ScheduledJob{
			{
				Name:     "lifecycle-job",
				Schedule: "@every 1h",
				Enabled:  true,
			},
		},
	})

	if err := subsystem.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := subsystem.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestSubsystemHealth(t *testing.T) {
	subsystem := NewSchedulerSubsystem(config.SchedulerConfig{
		Enabled: true,
		Jobs: []config.ScheduledJob{
			{
				Name:     "health-job",
				Schedule: "@every 1h",
				Enabled:  true,
			},
		},
	})

	if err := subsystem.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = subsystem.Stop(context.Background())
	})

	health := subsystem.Health()

	if got, want := health.Name, "scheduler"; got != want {
		t.Fatalf("Health().Name = %q, want %q", got, want)
	}
	if got, want := health.Status, "running"; got != want {
		t.Fatalf("Health().Status = %q, want %q", got, want)
	}
	// Now there's heartbeat (auto-registered) + 1 configured job = 2 tasks
	if !strings.Contains(health.Message, "tasks=2") {
		t.Fatalf("Health().Message = %q, want to contain %q", health.Message, "tasks=2")
	}
}

func TestSubsystemHeartbeatRegistered(t *testing.T) {
	subsystem := NewSchedulerSubsystem(config.SchedulerConfig{Enabled: true})

	if err := subsystem.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = subsystem.Stop(context.Background()) })

	// heartbeat task auto-registered even with no configured jobs
	if subsystem.HeartbeatStore() == nil {
		t.Fatal("HeartbeatStore() returned nil")
	}
	health := subsystem.Health()
	if !strings.Contains(health.Message, "tasks=1") {
		t.Fatalf("Health().Message = %q, want heartbeat registered (tasks=1)", health.Message)
	}
}
