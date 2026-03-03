package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/startower-observability/blackcat/internal/config"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	configUpdateToolName        = "config_update"
	configUpdateToolDescription = "Update a non-protected YAML configuration field at runtime"
)

var configUpdateToolParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"field": {
			"type": "string",
			"description": "YAML field path in dot notation, e.g. 'llm.model' or 'agent.name'"
		},
		"value": {
			"type": "string",
			"description": "New value to set for the field"
		}
	},
	"required": ["field", "value"]
}`)

// ConfigUpdateTool updates configurable YAML fields in-place.
type ConfigUpdateTool struct {
	configPath string // absolute path to blackcat.yaml
}

var _ types.Tool = (*ConfigUpdateTool)(nil)

func NewConfigUpdateTool(configPath string) *ConfigUpdateTool {
	return &ConfigUpdateTool{configPath: configPath}
}

func (t *ConfigUpdateTool) Name() string                { return configUpdateToolName }
func (t *ConfigUpdateTool) Description() string         { return configUpdateToolDescription }
func (t *ConfigUpdateTool) Parameters() json.RawMessage { return configUpdateToolParameters }

func (t *ConfigUpdateTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("config_update: invalid arguments: %w", err)
	}

	field := strings.TrimSpace(params.Field)
	if field == "" {
		return "", fmt.Errorf("config_update: field is required")
	}
	if strings.TrimSpace(params.Value) == "" {
		return "", fmt.Errorf("config_update: value is required")
	}

	if config.IsProtected(field) {
		return "", fmt.Errorf("config_update: %s", config.ProtectedReason(field))
	}

	content, err := os.ReadFile(t.configPath)
	if err != nil {
		return "", fmt.Errorf("config_update: read config: %w", err)
	}

	astFile, err := parser.ParseBytes(content, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("config_update: parse yaml: %w", err)
	}

	path, err := yaml.PathString("$." + field)
	if err != nil {
		return "", fmt.Errorf("config_update: invalid field path: %w", err)
	}

	replacementValue := coerceStringValue(params.Value)
	replacementYAML, err := yaml.Marshal(replacementValue)
	if err != nil {
		return "", fmt.Errorf("config_update: marshal value: %w", err)
	}

	if err := path.ReplaceWithReader(astFile, bytes.NewReader(replacementYAML)); err != nil {
		return "", fmt.Errorf("config_update: update field %q: %w", field, err)
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(t.configPath); statErr == nil {
		mode = info.Mode()
	}

	if err := os.WriteFile(t.configPath, []byte(astFile.String()), mode); err != nil {
		return "", fmt.Errorf("config_update: write config: %w", err)
	}

	return fmt.Sprintf("Config updated: %s = %s", field, params.Value), nil
}

func coerceStringValue(raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)

	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}

	return raw
}
