package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrRateLimit     = errors.New("llm: rate limit exceeded")
	ErrServerError   = errors.New("llm: server error")
	ErrAuthFailure   = errors.New("llm: authentication failed")
	ErrModelNotFound = errors.New("llm: model not found")
	ErrContextLength = errors.New("llm: context length exceeded")
	ErrTimeout       = errors.New("llm: request timeout")
)

// ClassifyError maps raw API errors to typed sentinel errors.
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}

	msg := strings.ToLower(err.Error())

	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return fmt.Errorf("%w: %v", ErrRateLimit, err)
	}

	if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") {
		return fmt.Errorf("%w: %v", ErrAuthFailure, err)
	}

	if strings.Contains(msg, "404") || strings.Contains(msg, "model not found") || strings.Contains(msg, "does not exist") {
		return fmt.Errorf("%w: %v", ErrModelNotFound, err)
	}

	if strings.Contains(msg, "context length") || (strings.Contains(msg, "token") && strings.Contains(msg, "400")) {
		return fmt.Errorf("%w: %v", ErrContextLength, err)
	}

	if strings.Contains(msg, "500") || strings.Contains(msg, "502") || strings.Contains(msg, "503") || strings.Contains(msg, "504") || strings.Contains(msg, "server error") || strings.Contains(msg, "internal error") {
		return fmt.Errorf("%w: %v", ErrServerError, err)
	}

	return err
}
