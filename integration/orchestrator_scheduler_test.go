package integration_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/internal/orchestrator"
	"github.com/startower-observability/blackcat/internal/scheduler"
)

func TestOrchestratorSchedulerTriggersOrchestrator(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "scheduler-state.json")
	sched := scheduler.New(stateFile)

	o := orchestrator.New(5, 30*time.Second)
	orchestrator.WithExecutor(func(ctx context.Context, cfg orchestrator.SpawnConfig) (string, error) {
		return cfg.Name + ":ok", nil
	})(o)

	err := sched.Register(scheduler.TaskDef{
		Name:     "orchestrator-dispatch",
		Schedule: "@every 1s",
		Enabled:  true,
		Handler: func(ctx context.Context) error {
			configs := []orchestrator.SpawnConfig{
				{Name: "agent-1", Task: "do thing", Timeout: 2 * time.Second},
				{Name: "agent-2", Task: "do thing", Timeout: 2 * time.Second},
			}
			results, _ := o.Dispatch(ctx, configs)
			if len(results) != 2 {
				return errors.New("unexpected dispatch result count")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	sched.Start()
	time.Sleep(2200 * time.Millisecond)
	sched.Stop(context.Background())

	state := schedulerStateByName(t, sched.ListTasks(), "orchestrator-dispatch")
	if state.RunCount < 1 {
		t.Fatalf("expected scheduler task to run at least once, got run_count=%d", state.RunCount)
	}
}

func TestOrchestratorParallelDispatch(t *testing.T) {
	o := orchestrator.New(5, 30*time.Second)
	orchestrator.WithExecutor(func(ctx context.Context, cfg orchestrator.SpawnConfig) (string, error) {
		time.Sleep(100 * time.Millisecond)
		return cfg.Name + ":done", nil
	})(o)

	configs := make([]orchestrator.SpawnConfig, 0, 5)
	for i := 0; i < 5; i++ {
		configs = append(configs, orchestrator.SpawnConfig{Name: "agent", Task: "parallel", Timeout: 2 * time.Second})
	}

	start := time.Now()
	results, _ := o.Dispatch(context.Background(), configs)
	elapsed := time.Since(start)

	if elapsed >= 450*time.Millisecond {
		t.Fatalf("expected concurrent runtime under 450ms, got %v", elapsed)
	}
	if len(results) != len(configs) {
		t.Fatalf("expected %d results, got %d", len(configs), len(results))
	}
	for i, result := range results {
		if result.Index != i {
			t.Fatalf("result index mismatch at %d: got %d", i, result.Index)
		}
	}
}

func TestOrchestratorSubAgentTimeout(t *testing.T) {
	o := orchestrator.New(2, 30*time.Second)
	orchestrator.WithExecutor(func(ctx context.Context, cfg orchestrator.SpawnConfig) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})(o)

	start := time.Now()
	results, _ := o.Dispatch(context.Background(), []orchestrator.SpawnConfig{{
		Name:    "slow-agent",
		Task:    "block",
		Timeout: 200 * time.Millisecond,
	}})
	elapsed := time.Since(start)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !errors.Is(results[0].Error, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", results[0].Error)
	}
	if elapsed > time.Second {
		t.Fatalf("dispatch took too long, possible goroutine leak: %v", elapsed)
	}
}

func TestOrchestratorSubAgentHardCap(t *testing.T) {
	o := orchestrator.New(10, 30*time.Second)

	var current int32
	var peak int32
	orchestrator.WithExecutor(func(ctx context.Context, cfg orchestrator.SpawnConfig) (string, error) {
		now := atomic.AddInt32(&current, 1)
		for {
			seen := atomic.LoadInt32(&peak)
			if now <= seen {
				break
			}
			if atomic.CompareAndSwapInt32(&peak, seen, now) {
				break
			}
		}
		defer atomic.AddInt32(&current, -1)

		time.Sleep(50 * time.Millisecond)
		return cfg.Name, nil
	})(o)

	configs := make([]orchestrator.SpawnConfig, 0, 15)
	for i := 0; i < 15; i++ {
		configs = append(configs, orchestrator.SpawnConfig{Name: "agent", Task: "cap", Timeout: 2 * time.Second})
	}

	results, err := o.Dispatch(context.Background(), configs)
	if err != nil {
		t.Fatalf("Dispatch returned unexpected error: %v", err)
	}
	if len(results) != 15 {
		t.Fatalf("expected 15 results, got %d", len(results))
	}
	for i, result := range results {
		if result.Index != i {
			t.Fatalf("result index mismatch at %d: got %d", i, result.Index)
		}
	}
	if atomic.LoadInt32(&peak) > 10 {
		t.Fatalf("expected max peak concurrency <= 10, got %d", atomic.LoadInt32(&peak))
	}
}

func TestOrchestratorSchedulerSkipIfRunning(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "skip-running-state.json")
	sched := scheduler.New(stateFile)

	err := sched.Register(scheduler.TaskDef{
		Name:     "long-job",
		Schedule: "@every 1s",
		Enabled:  true,
		Handler: func(ctx context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	sched.Start()
	time.Sleep(5 * time.Second)
	sched.Stop(context.Background())

	state := schedulerStateByName(t, sched.ListTasks(), "long-job")
	if state.RunCount < 1 || state.RunCount > 2 {
		t.Fatalf("expected run_count in [1,2] with skip-if-running, got %d", state.RunCount)
	}
}

func TestOrchestratorHeartbeatIntegration(t *testing.T) {
	checker := staticChecker{statuses: []scheduler.SubsystemHealthInfo{
		{Name: "database", Healthy: true, Details: "ok"},
		{Name: "queue", Healthy: true, Details: "ok"},
		{Name: "redis", Healthy: false, Details: "timeout"},
	}}

	store := scheduler.NewHeartbeatStore(10)
	task := scheduler.NewHeartbeatTask(checker, store)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	latest := store.Latest(1)
	if len(latest) != 1 {
		t.Fatalf("expected 1 heartbeat result, got %d", len(latest))
	}
	if len(latest[0].Subsystems) != 3 {
		t.Fatalf("expected 3 subsystem results, got %d", len(latest[0].Subsystems))
	}
	if latest[0].OverallHealthy {
		t.Fatalf("expected overall healthy=false when one subsystem is unhealthy")
	}

	healthyCount := 0
	unhealthyCount := 0
	for _, subsystem := range latest[0].Subsystems {
		if subsystem.Healthy {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	if healthyCount != 2 || unhealthyCount != 1 {
		t.Fatalf("expected 2 healthy and 1 unhealthy subsystems, got healthy=%d unhealthy=%d", healthyCount, unhealthyCount)
	}
}

type staticChecker struct {
	statuses []scheduler.SubsystemHealthInfo
}

func (s staticChecker) ListHealthy() []scheduler.SubsystemHealthInfo {
	return append([]scheduler.SubsystemHealthInfo(nil), s.statuses...)
}

func schedulerStateByName(t *testing.T, states []scheduler.TaskState, name string) scheduler.TaskState {
	t.Helper()
	for _, state := range states {
		if state.Name == name {
			return state
		}
	}
	t.Fatalf("task state not found: %s", name)
	return scheduler.TaskState{}
}
