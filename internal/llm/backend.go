package llm

import (
	"context"

	"github.com/startower-observability/blackcat/internal/types"
)

// Backend is the interface that all LLM provider implementations must satisfy.
// It mirrors types.LLMClient but lives in the llm package for provider use.
type Backend interface {
	Chat(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (*types.LLMResponse, error)
	Stream(ctx context.Context, messages []types.LLMMessage, tools []types.ToolDefinition) (<-chan types.Chunk, error)
}

// BackendInfo provides introspection metadata about a backend provider.
// It complements ProviderSpec (which holds config defaults) with runtime info.
type BackendInfo struct {
	Name       string   // Human-readable provider name (e.g., "copilot", "antigravity")
	Models     []string // Models supported by this backend
	AuthMethod string   // Auth method: "api-key", "oauth-device", "oauth-pkce"
}

// BackendConfig holds generic configuration that all backends use for construction.
type BackendConfig struct {
	APIKey      string                 // API key (for key-based auth providers)
	BaseURL     string                 // Provider API base URL
	Model       string                 // Model name to use
	Temperature float64                // Sampling temperature (0.0 to 2.0)
	MaxTokens   int                    // Max tokens per response
	TokenSource func() (string, error) // Dynamic token source (for OAuth providers)
}

// BackendFactory is a constructor function that creates a Backend from config.
type BackendFactory func(cfg BackendConfig) (Backend, error)

// InfoProvider is an optional interface backends can implement to expose metadata.
type InfoProvider interface {
	Info() BackendInfo
}
