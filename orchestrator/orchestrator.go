package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/startower-observability/blackcat/agent"
	"golang.org/x/sync/errgroup"
)

const (
	minConcurrent          = 1
	maxConcurrentHardCap   = 10
	defaultDispatchTimeout = 5 * time.Minute
)

// SpawnConfig defines a sub-agent task.
type SpawnConfig struct {
	Name           string
	Task           string
	Timeout        time.Duration
	Profile        string
	ProfileOverlay string // system prompt overlay applied from Profile
}

// Result holds the output of a sub-agent.
type Result struct {
	Name     string
	Output   string
	Error    error
	Duration time.Duration
	Profile  string
	Index    int
}

// Orchestrator manages parallel sub-agent dispatch.
type Orchestrator struct {
	maxConcurrent  int
	defaultTimeout time.Duration
	executor       func(ctx context.Context, cfg SpawnConfig) (string, error)
	Profiles       map[string]*agent.Profile
}

func New(maxConcurrent int, defaultTimeout time.Duration) *Orchestrator {
	o := &Orchestrator{
		maxConcurrent:  clamp(maxConcurrent, minConcurrent, maxConcurrentHardCap),
		defaultTimeout: sanitizeTimeout(defaultTimeout),
		executor:       defaultExecutor,
	}

	return o
}

func WithExecutor(exec func(ctx context.Context, cfg SpawnConfig) (string, error)) func(*Orchestrator) {
	return func(o *Orchestrator) {
		if o == nil || exec == nil {
			return
		}

		o.executor = exec
	}
}

func (o *Orchestrator) Dispatch(ctx context.Context, configs []SpawnConfig) ([]Result, error) {
	if len(configs) == 0 {
		return []Result{}, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	results := make([]Result, len(configs))

	g, groupCtx := errgroup.WithContext(ctx)
	g.SetLimit(clamp(o.maxConcurrent, minConcurrent, maxConcurrentHardCap))

	for i := range configs {
		idx := i
		cfg := configs[i]

		g.Go(func() error {
			start := time.Now()
			timeout := cfg.Timeout
			if timeout <= 0 {
				timeout = sanitizeTimeout(o.defaultTimeout)
			}

			timeoutCtx, cancel := context.WithTimeout(groupCtx, timeout)
			defer cancel()

			appliedProfile := ""
			profileOverlay := ""
			if cfg.Profile != "" && len(o.Profiles) > 0 {
				profile, ok := o.Profiles[cfg.Profile]
				if !ok {
					profile, ok = o.Profiles[strings.ToLower(cfg.Profile)]
				}
				if ok && profile != nil {
					profileOverlay = agent.ApplyProfile("", profile)
					appliedProfile = cfg.Profile
				}
			}

			cfgWithOverlay := cfg
			cfgWithOverlay.ProfileOverlay = profileOverlay
			output, execErr := o.executor(timeoutCtx, cfgWithOverlay)
			results[idx] = Result{
				Name:     cfg.Name,
				Output:   output,
				Error:    execErr,
				Duration: time.Since(start),
				Profile:  appliedProfile,
				Index:    idx,
			}

			return nil
		})
	}

	_ = g.Wait()

	return results, nil
}

func sanitizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultDispatchTimeout
	}

	return timeout
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}

// defaultExecutor is a placeholder; real sub-agent dispatch is wired in Task 17/26.
func defaultExecutor(_ context.Context, cfg SpawnConfig) (string, error) {
	return fmt.Sprintf("task accepted: %s", cfg.Task), nil
}
