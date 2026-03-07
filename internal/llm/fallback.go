package llm

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/startower-observability/blackcat/internal/types"
)

// FallbackBackend wraps multiple Backend instances and tries each in order.
// If the primary backend fails, it falls through to the next one.
// ErrContextLength is never retried — it returns immediately.
type FallbackBackend struct {
	backends []Backend
	names    []string
}

// NewFallbackBackend creates a FallbackBackend from the given backends and names.
// Returns an error if backends is empty or if len(backends) != len(names).
func NewFallbackBackend(backends []Backend, names []string) (*FallbackBackend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("fallback: at least one backend is required")
	}
	if len(backends) != len(names) {
		return nil, fmt.Errorf("fallback: backends and names must have the same length")
	}
	return &FallbackBackend{
		backends: backends,
		names:    names,
	}, nil
}

// Chat tries each backend in order using RetryChat. If a backend returns
// ErrContextLength, it is returned immediately without trying further backends.
// Other errors cause a fallback to the next backend.
func (fb *FallbackBackend) Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error) {
	var errs []error

	for i, backend := range fb.backends {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := RetryChat(ctx, backend, messages, tools, 3)
		if err == nil {
			return resp, nil
		}

		// ErrContextLength must never trigger fallback
		if errors.Is(err, ErrContextLength) {
			return nil, err
		}

		log.Printf("fallback: backend %q failed: %v", fb.names[i], err)
		errs = append(errs, fmt.Errorf("%s: %w", fb.names[i], err))
	}

	return nil, fmt.Errorf("all providers failed: %w", errors.Join(errs...))
}

// Stream tries each backend in order by calling Stream directly (no retry wrapper).
// If a backend returns ErrContextLength, it is returned immediately without
// trying further backends. Other errors cause a fallback to the next backend.
func (fb *FallbackBackend) Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error) {
	var errs []error

	for i, backend := range fb.backends {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ch, err := backend.Stream(ctx, messages, tools)
		if err == nil {
			return ch, nil
		}

		// ErrContextLength must never trigger fallback
		if errors.Is(err, ErrContextLength) {
			return nil, err
		}

		log.Printf("fallback: backend %q stream failed: %v", fb.names[i], err)
		errs = append(errs, fmt.Errorf("%s: %w", fb.names[i], err))
	}

	return nil, fmt.Errorf("all providers failed: %w", errors.Join(errs...))
}
