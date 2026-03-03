package daemon

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type SubsystemRegistry struct {
	mu         sync.RWMutex
	subsystems []Subsystem
}

func NewSubsystemRegistry() *SubsystemRegistry {
	return &SubsystemRegistry{}
}

func (r *SubsystemRegistry) Register(s Subsystem) {
	if s == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subsystems = append(r.subsystems, s)
}

func (r *SubsystemRegistry) StartAll(ctx context.Context) error {
	r.mu.RLock()
	subsystems := append([]Subsystem(nil), r.subsystems...)
	r.mu.RUnlock()

	started := make([]Subsystem, 0, len(subsystems))
	for _, subsystem := range subsystems {
		if err := subsystem.Start(ctx); err != nil {
			stopErr := stopAllReverse(ctx, started)
			if stopErr != nil {
				return fmt.Errorf("start subsystem %s: %w", subsystem.Name(), errors.Join(err, stopErr))
			}
			return fmt.Errorf("start subsystem %s: %w", subsystem.Name(), err)
		}
		started = append(started, subsystem)
	}

	return nil
}

func (r *SubsystemRegistry) StopAll(ctx context.Context) error {
	r.mu.RLock()
	subsystems := append([]Subsystem(nil), r.subsystems...)
	r.mu.RUnlock()

	return stopAllReverse(ctx, subsystems)
}

func (r *SubsystemRegistry) Healthz() []SubsystemHealth {
	r.mu.RLock()
	subsystems := append([]Subsystem(nil), r.subsystems...)
	r.mu.RUnlock()

	health := make([]SubsystemHealth, 0, len(subsystems))
	for _, subsystem := range subsystems {
		h := subsystem.Health()
		if h.Name == "" {
			h.Name = subsystem.Name()
		}
		health = append(health, h)
	}

	return health
}

func stopAllReverse(ctx context.Context, subsystems []Subsystem) error {
	var errs []error
	for i := len(subsystems) - 1; i >= 0; i-- {
		if err := subsystems[i].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop subsystem %s: %w", subsystems[i].Name(), err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
