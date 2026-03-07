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

	if c.Budget.Enabled {
		if c.Budget.DailyLimitUSD < 0 {
			errs = append(errs, fmt.Errorf("budget.daily_limit_usd must be >= 0"))
		}
		if c.Budget.MonthlyLimitUSD < 0 {
			errs = append(errs, fmt.Errorf("budget.monthly_limit_usd must be >= 0"))
		}
		if c.Budget.WarnThreshold < 0 || c.Budget.WarnThreshold > 1 {
			errs = append(errs, fmt.Errorf("budget.warn_threshold must be between 0 and 1"))
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
