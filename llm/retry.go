package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/startower-observability/blackcat/types"
)

// RetryChat calls client.Chat with retry logic for transient errors.
// maxAttempts is the total number of attempts (e.g., 3 = 1 initial + 2 retries).
func RetryChat(ctx context.Context, client types.LLMClient, messages []types.LLMMessage, tools []types.ToolDefinition, maxAttempts int) (*types.LLMResponse, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	var lastErr error
	base := time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := client.Chat(ctx, messages, tools)
		if err == nil {
			return resp, nil
		}

		classified := ClassifyError(err)
		lastErr = classified

		// Never retry non-transient errors
		if !isRetryable(classified) {
			return nil, classified
		}

		// Don't retry on last attempt
		if attempt == maxAttempts-1 {
			break
		}

		// Exponential backoff with jitter
		wait := base * time.Duration(1<<uint(attempt))
		jitter := time.Duration(rand.Int63n(int64(base)))
		wait += jitter
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}

		slog.Warn("llm call failed, retrying",
			"attempt", attempt+1,
			"max", maxAttempts,
			"err", err,
			"backoff", wait,
		)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("%w: %v", ErrTimeout, ctx.Err())
		case <-timer.C:
		}
	}

	return nil, lastErr
}

func isRetryable(err error) bool {
	return errors.Is(err, ErrRateLimit) || errors.Is(err, ErrServerError) || errors.Is(err, ErrTimeout)
}
