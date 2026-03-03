package llm

import (
	"fmt"
	"strings"
	"sync"
)

// ProviderSpec describes provider-specific configuration defaults.
type ProviderSpec struct {
	Name           string
	BaseURL        string
	APIKeyEnvVar   string
	Models         []string
	DetectByPrefix string
}

var defaultProviders = []ProviderSpec{
	{
		Name:           "openai",
		BaseURL:        "https://api.openai.com/v1",
		APIKeyEnvVar:   "OPENAI_API_KEY",
		DetectByPrefix: "sk-",
		Models:         []string{"gpt-5.2", "gpt-5.1", "gpt-5-mini", "gpt-4.1", "gpt-4.1-mini", "o3", "o4-mini", "gpt-4o", "gpt-4o-mini"},
	},
	{
		Name:           "anthropic",
		BaseURL:        "https://api.anthropic.com/v1",
		APIKeyEnvVar:   "ANTHROPIC_API_KEY",
		DetectByPrefix: "sk-ant-",
		Models:         []string{"claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"},
	},
	{
		Name:           "google",
		BaseURL:        "https://generativelanguage.googleapis.com/v1beta",
		APIKeyEnvVar:   "GOOGLE_API_KEY",
		DetectByPrefix: "AI",
		Models:         []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-3.1-pro-preview", "gemini-3-flash-preview"},
	},
	{
		Name:           "openrouter",
		BaseURL:        "https://openrouter.ai/api/v1",
		APIKeyEnvVar:   "OPENROUTER_API_KEY",
		DetectByPrefix: "sk-or-",
		Models:         []string{},
	},
	{
		Name:           "ollama",
		BaseURL:        "http://localhost:11434/v1",
		APIKeyEnvVar:   "",
		DetectByPrefix: "",
		Models:         []string{},
	},
}

type ProviderRegistry struct {
	providers []ProviderSpec
}

func NewRegistry(extra ...ProviderSpec) *ProviderRegistry {
	providers := make([]ProviderSpec, 0, len(defaultProviders)+len(extra))
	providers = append(providers, defaultProviders...)
	providers = append(providers, extra...)

	return &ProviderRegistry{providers: providers}
}

func (r *ProviderRegistry) Detect(getEnv func(string) string) (*ProviderSpec, string) {
	if getEnv == nil {
		return nil, ""
	}

	for i := range r.providers {
		provider := &r.providers[i]
		if provider.APIKeyEnvVar == "" {
			continue
		}

		apiKey := strings.TrimSpace(getEnv(provider.APIKeyEnvVar))
		if apiKey == "" {
			continue
		}

		if provider.DetectByPrefix != "" && !strings.HasPrefix(apiKey, provider.DetectByPrefix) {
			continue
		}

		return provider, apiKey
	}

	return nil, ""
}

func (r *ProviderRegistry) Get(name string) (*ProviderSpec, bool) {
	for i := range r.providers {
		if strings.EqualFold(r.providers[i].Name, name) {
			return &r.providers[i], true
		}
	}

	return nil, false
}

// --- Backend Registry (Phase 2) ---

// backendRegistry stores registered backend factories keyed by provider name.
var backendRegistry = struct {
	mu        sync.RWMutex
	factories map[string]BackendFactory
}{
	factories: make(map[string]BackendFactory),
}

// RegisterBackend registers a BackendFactory under the given provider name.
// It is safe for concurrent use.
func RegisterBackend(name string, factory BackendFactory) {
	backendRegistry.mu.Lock()
	defer backendRegistry.mu.Unlock()
	backendRegistry.factories[strings.ToLower(name)] = factory
}

// CreateBackend looks up a registered BackendFactory by name and creates a Backend.
// Returns an error if no factory is registered for the given name.
func CreateBackend(name string, cfg BackendConfig) (Backend, error) {
	backendRegistry.mu.RLock()
	factory, ok := backendRegistry.factories[strings.ToLower(name)]
	backendRegistry.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no backend registered for provider %q", name)
	}

	return factory(cfg)
}

// RegisteredBackends returns the names of all registered backend providers.
func RegisteredBackends() []string {
	backendRegistry.mu.RLock()
	defer backendRegistry.mu.RUnlock()

	names := make([]string, 0, len(backendRegistry.factories))
	for name := range backendRegistry.factories {
		names = append(names, name)
	}
	return names
}

// ResetBackendRegistry clears all registered backends. Used for testing.
func ResetBackendRegistry() {
	backendRegistry.mu.Lock()
	defer backendRegistry.mu.Unlock()
	backendRegistry.factories = make(map[string]BackendFactory)
}
