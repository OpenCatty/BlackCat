package config

import (
	"errors"
	"fmt"
)

// ValidateDeep performs deep validation of config values beyond basic presence checks.
// Collects all errors before returning and does not fail-fast.
func ValidateDeep(c *Config) error {
	if c == nil {
		return errors.New("config must not be nil")
	}

	var errs []error

	if c.LLM.Temperature < 0 || c.LLM.Temperature > 2.0 {
		errs = append(errs, fmt.Errorf("llm.temperature must be in [0.0, 2.0], got %f", c.LLM.Temperature))
	}
	if c.LLM.MaxTokens < 0 {
		errs = append(errs, fmt.Errorf("llm.maxTokens must be >= 0, got %d", c.LLM.MaxTokens))
	}
	if c.LLM.MaxContextTokens < 0 {
		errs = append(errs, fmt.Errorf("llm.maxContextTokens must be >= 0, got %d", c.LLM.MaxContextTokens))
	}

	if c.RateLimit.Enabled {
		if c.RateLimit.MaxRequests <= 0 {
			errs = append(errs, fmt.Errorf("rateLimit.maxRequests must be > 0 when enabled, got %d", c.RateLimit.MaxRequests))
		}
		if c.RateLimit.WindowSeconds <= 0 {
			errs = append(errs, fmt.Errorf("rateLimit.windowSeconds must be > 0 when enabled, got %d", c.RateLimit.WindowSeconds))
		}
	}

	for i, job := range c.Scheduler.Jobs {
		if job.Enabled {
			if job.Name == "" {
				errs = append(errs, fmt.Errorf("scheduler.jobs[%d].name must not be empty when enabled", i))
			}
			if job.Schedule == "" {
				errs = append(errs, fmt.Errorf("scheduler.jobs[%d].schedule must not be empty when enabled", i))
			}
		}
	}

	return errors.Join(errs...)
}
