// Package tools provides the tool registry and built-in tools for the agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/startower-observability/blackcat/internal/memory"
	"github.com/startower-observability/blackcat/internal/types"
)

// Registry holds registered tools and dispatches execution requests.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]types.Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]types.Tool),
	}
}

// Register adds a tool to the registry, keyed by its Name().
func (r *Registry) Register(tool types.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get returns a registered tool by name, or types.ErrToolNotFound.
func (r *Registry) Get(name string) (types.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, types.ErrToolNotFound
	}
	return t, nil
}

// List returns tool definitions for all registered tools (for LLM consumption).
func (r *Registry) List() []types.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]types.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, types.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

// ValidationError is returned when tool arguments fail schema validation.
type ValidationError struct {
	ToolName string
	Details  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("tool %q argument validation failed: %s", e.ToolName, e.Details)
}

// Execute finds a tool by name, validates args against the tool's schema, and runs it.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	t, err := r.Get(name)
	if err != nil {
		return "", err
	}

	// Validate args against the tool's JSON Schema (required-field check only).
	params := t.Parameters()
	if len(params) > 0 && string(params) != "null" && string(params) != "{}" {
		var argsMap map[string]any
		if err := json.Unmarshal(args, &argsMap); err != nil {
			return "", &ValidationError{ToolName: name, Details: fmt.Sprintf("args is not valid JSON: %v", err)}
		}

		var schema struct {
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(params, &schema); err == nil {
			for _, field := range schema.Required {
				if _, ok := argsMap[field]; !ok {
					return "", &ValidationError{ToolName: name, Details: fmt.Sprintf("missing required field: %s", field)}
				}
			}
		}
	}

	return t.Execute(ctx, args)
}

// RegisterMemoryTools registers the core memory and archival memory tools
// into the given registry. The userID is baked into each tool handler at
// construction time for security — it is NOT passed via tool parameters.
func RegisterMemoryTools(r *Registry, core *memory.CoreStore, archival *memory.SQLiteStore, embed *memory.EmbeddingClient, userID string) {
	if core != nil {
		h := NewCoreMemoryToolHandler(core, userID)
		h.RegisterTools(r)
	}
	if archival != nil {
		h := NewArchivalMemoryToolHandler(archival, embed, userID)
		h.RegisterTools(r)
	}
}

// Filter returns a new Registry containing only the tools with names in allowedTools.
// If allowedTools is nil or empty, returns a copy of the entire registry.
// The original registry is not mutated.
func (r *Registry) Filter(allowedTools []string) *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	newRegistry := NewRegistry()
	
	// If no filter criteria, return a full copy
	if len(allowedTools) == 0 {
		for name, tool := range r.tools {
			newRegistry.tools[name] = tool
		}
		return newRegistry
	}
	
	// Copy only allowed tools
	allowedSet := make(map[string]bool, len(allowedTools))
	for _, name := range allowedTools {
		allowedSet[name] = true
	}
	
	for name, tool := range r.tools {
		if allowedSet[name] {
			newRegistry.tools[name] = tool
		}
	}
	
	return newRegistry
}
