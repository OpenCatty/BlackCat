package daemon

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type LifecycleManager struct {
	subsystems []Subsystem
	mu         sync.Mutex
}

func NewLifecycleManager(subsystems ...Subsystem) *LifecycleManager {
	ordered := make([]Subsystem, 0, len(subsystems))
	for _, subsystem := range subsystems {
		if subsystem == nil {
			continue
		}
		ordered = append(ordered, subsystem)
	}

	return &LifecycleManager{subsystems: ordered}
}

func (lm *LifecycleManager) StartAll(ctx context.Context) error {
	lm.mu.Lock()
	subsystems := append([]Subsystem(nil), lm.subsystems...)
	lm.mu.Unlock()

	for _, subsystem := range subsystems {
		if err := subsystem.Start(ctx); err != nil {
			return fmt.Errorf("subsystem %q failed to start: %w", subsystem.Name(), err)
		}
	}

	return nil
}

func (lm *LifecycleManager) StopAll(ctx context.Context) error {
	lm.mu.Lock()
	subsystems := append([]Subsystem(nil), lm.subsystems...)
	lm.mu.Unlock()

	var errs []error
	for i := len(subsystems) - 1; i >= 0; i-- {
		subsystem := subsystems[i]
		if err := subsystem.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("subsystem %q failed to stop: %w", subsystem.Name(), err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
