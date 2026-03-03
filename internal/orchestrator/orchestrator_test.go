package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatchOrdering(t *testing.T) {
	o := New(5, time.Second)
	WithExecutor(func(ctx context.Context, cfg SpawnConfig) (string, error) {
		delay := time.Duration(60-len(cfg.Name)*10) * time.Millisecond
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
			return fmt.Sprintf("done:%s", cfg.Name), nil
		}
	})(o)

	configs := []SpawnConfig{
		{Name: "a", Task: "1"},
		{Name: "bb", Task: "2"},
		{Name: "ccc", Task: "3"},
		{Name: "dddd", Task: "4"},
		{Name: "eeeee", Task: "5"},
	}

	results, err := o.Dispatch(context.Background(), configs)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if len(results) != len(configs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(configs))
	}

	for i := range configs {
		if results[i].Name != configs[i].Name {
			t.Fatalf("results[%d].Name = %q, want %q", i, results[i].Name, configs[i].Name)
		}
		if results[i].Index != i {
			t.Fatalf("results[%d].Index = %d, want %d", i, results[i].Index, i)
		}
		if results[i].Error != nil {
			t.Fatalf("results[%d].Error = %v, want nil", i, results[i].Error)
		}
	}
}

func TestHardCap(t *testing.T) {
	o := New(15, time.Second)
	if o.maxConcurrent != 10 {
		t.Fatalf("maxConcurrent = %d, want 10", o.maxConcurrent)
	}

	var running int32
	var peak int32

	WithExecutor(func(ctx context.Context, _ SpawnConfig) (string, error) {
		current := atomic.AddInt32(&running, 1)
		defer atomic.AddInt32(&running, -1)

		for {
			seen := atomic.LoadInt32(&peak)
			if current <= seen {
				break
			}
			if atomic.CompareAndSwapInt32(&peak, seen, current) {
				break
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(30 * time.Millisecond):
			return "ok", nil
		}
	})(o)

	configs := make([]SpawnConfig, 30)
	for i := range configs {
		configs[i] = SpawnConfig{Name: fmt.Sprintf("task-%d", i), Task: "cap"}
	}

	results, err := o.Dispatch(context.Background(), configs)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if len(results) != len(configs) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(configs))
	}

	if peak > 10 {
		t.Fatalf("observed concurrency peak = %d, exceeded hard cap 10", peak)
	}
}

func TestTimeoutEnforcement(t *testing.T) {
	o := New(3, 500*time.Millisecond)
	WithExecutor(func(ctx context.Context, _ SpawnConfig) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return "late", nil
		}
	})(o)

	results, err := o.Dispatch(context.Background(), []SpawnConfig{{Name: "slow", Task: "timeout", Timeout: 20 * time.Millisecond}})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if !errors.Is(results[0].Error, context.DeadlineExceeded) {
		t.Fatalf("results[0].Error = %v, want deadline exceeded", results[0].Error)
	}
}

func TestContextCancellation(t *testing.T) {
	o := New(4, time.Second)
	WithExecutor(func(ctx context.Context, _ SpawnConfig) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})(o)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	configs := []SpawnConfig{
		{Name: "a", Task: "cancel", Timeout: time.Second},
		{Name: "b", Task: "cancel", Timeout: time.Second},
		{Name: "c", Task: "cancel", Timeout: time.Second},
		{Name: "d", Task: "cancel", Timeout: time.Second},
		{Name: "e", Task: "cancel", Timeout: time.Second},
	}

	results, err := o.Dispatch(ctx, configs)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	for i, result := range results {
		if !errors.Is(result.Error, context.Canceled) {
			t.Fatalf("results[%d].Error = %v, want context canceled", i, result.Error)
		}
	}
}

func TestEmptyConfigs(t *testing.T) {
	o := New(2, time.Second)
	results, err := o.Dispatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
}
