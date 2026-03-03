package config

import (
	"strings"
	"testing"
)

func TestValidateDeep_TemperatureOutOfRange(t *testing.T) {
	cfg := &Config{}
	cfg.LLM.Temperature = 3.0
	err := ValidateDeep(cfg)
	if err == nil {
		t.Fatal("expected error for temperature out of range")
	}
	if !strings.Contains(err.Error(), "temperature") {
		t.Fatalf("expected error about temperature, got: %s", err.Error())
	}
}

func TestValidateDeep_MultipleErrors(t *testing.T) {
	cfg := &Config{}
	cfg.LLM.Temperature = 3.0
	cfg.LLM.MaxTokens = -1
	err := ValidateDeep(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "temperature") || !strings.Contains(msg, "maxTokens") {
		t.Fatalf("expected both errors, got: %s", msg)
	}
}

func TestValidateDeep_Valid(t *testing.T) {
	cfg := &Config{}
	cfg.LLM.Temperature = 1.0
	err := ValidateDeep(cfg)
	if err != nil {
		t.Fatalf("expected nil error for valid config, got: %s", err.Error())
	}
}
