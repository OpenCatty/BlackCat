package hooks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// HookHandler runs custom logic for a specific hook event.
type HookHandler func(ctx *HookContext) error

// HookFunc is kept as a backward-compatible alias.
type HookFunc = HookHandler

type prioritizedHandler struct {
	priority int
	handler  HookHandler
}

// HookRegistry stores hook handlers grouped by event.
type HookRegistry struct {
	mu    sync.RWMutex
	hooks map[HookEvent][]prioritizedHandler
}

// NewHookRegistry creates an empty hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{hooks: make(map[HookEvent][]prioritizedHandler)}
}

// Register adds a hook handler to an event.
func (r *HookRegistry) Register(event HookEvent, fn HookHandler) {
	r.RegisterWithPriority(event, 100, fn)
}

// RegisterWithPriority adds a hook handler to an event with explicit priority.
// Lower numbers run first.
func (r *HookRegistry) RegisterWithPriority(event HookEvent, priority int, fn HookHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[event] = append(r.hooks[event], prioritizedHandler{priority: priority, handler: fn})
	sort.Slice(r.hooks[event], func(i, j int) bool {
		return r.hooks[event][i].priority < r.hooks[event][j].priority
	})
}

// Fire executes all hooks registered for an event.
func (r *HookRegistry) Fire(ctx context.Context, event HookEvent, hctx *HookContext) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	r.mu.RLock()
	handlers := append([]prioritizedHandler(nil), r.hooks[event]...)
	r.mu.RUnlock()

	if hctx != nil {
		hctx.Event = event
	}

	if isPreEvent(event) {
		for _, handler := range handlers {
			if err := callHook(handler.handler, hctx); err != nil {
				return err
			}
		}
		return nil
	}

	var errs []error
	for _, handler := range handlers {
		if err := callHook(handler.handler, hctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func isPreEvent(event HookEvent) bool {
	switch event {
	case PreChat, PreToolExec, PreFileRead, PreFileWrite:
		return true
	default:
		return false
	}
}

func callHook(fn HookHandler, hctx *HookContext) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("hook panic recovered: %v", recovered)
		}
	}()

	return fn(hctx)
}
