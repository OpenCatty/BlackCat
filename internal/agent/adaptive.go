package agent

import (
	"context"
	"fmt"
	"strings"
)

// CoreStoreIface is a minimal interface for reading/writing core memory entries.
// *memory.CoreStore already satisfies this interface.
type CoreStoreIface interface {
	Get(ctx context.Context, userID, key string) (string, error)
	Set(ctx context.Context, userID, key, value string) error
}

// AdaptiveProfile holds per-user adaptive behaviour preferences.
type AdaptiveProfile struct {
	Language       string // e.g. "English", "Spanish", or "auto"
	Style          string // "casual", "formal", "technical", or "auto"
	Verbosity      string // "brief", "normal", "detailed", or "auto"
	TechnicalDepth string // "basic", "intermediate", "expert", or "auto"
}

// Preference key constants stored in core memory.
const (
	prefKeyLanguage       = "pref:language"
	prefKeyStyle          = "pref:style"
	prefKeyVerbosity      = "pref:verbosity"
	prefKeyTechnicalDepth = "pref:technical_depth"
)

// validPrefKeys enumerates all recognised preference keys.
var validPrefKeys = map[string]bool{
	prefKeyLanguage:       true,
	prefKeyStyle:          true,
	prefKeyVerbosity:      true,
	prefKeyTechnicalDepth: true,
}

// PreferenceManager loads and updates per-user adaptive preferences
// backed by a CoreStore.
type PreferenceManager struct {
	coreStore CoreStoreIface
}

// NewPreferenceManager creates a PreferenceManager.
// A nil store is allowed — methods will return safe defaults.
func NewPreferenceManager(store CoreStoreIface) *PreferenceManager {
	return &PreferenceManager{coreStore: store}
}

// LoadPreferences reads the four preference keys from core store and returns
// an AdaptiveProfile. Missing keys default to "auto".
func (pm *PreferenceManager) LoadPreferences(ctx context.Context, userID string) AdaptiveProfile {
	profile := AdaptiveProfile{
		Language:       "auto",
		Style:          "auto",
		Verbosity:      "auto",
		TechnicalDepth: "auto",
	}

	if pm == nil || pm.coreStore == nil {
		return profile
	}

	if v, err := pm.coreStore.Get(ctx, userID, prefKeyLanguage); err == nil && v != "" {
		profile.Language = v
	}
	if v, err := pm.coreStore.Get(ctx, userID, prefKeyStyle); err == nil && v != "" {
		profile.Style = v
	}
	if v, err := pm.coreStore.Get(ctx, userID, prefKeyVerbosity); err == nil && v != "" {
		profile.Verbosity = v
	}
	if v, err := pm.coreStore.Get(ctx, userID, prefKeyTechnicalDepth); err == nil && v != "" {
		profile.TechnicalDepth = v
	}

	return profile
}

// UpdatePreference validates that key is a recognised preference key, then
// persists the value to the core store. Returns an error for unknown keys
// or store failures. Nil-safe — returns nil if the manager or store is nil.
func (pm *PreferenceManager) UpdatePreference(ctx context.Context, userID, key, value string) error {
	if pm == nil || pm.coreStore == nil {
		return nil
	}

	if !validPrefKeys[key] {
		return fmt.Errorf("adaptive: unknown preference key %q", key)
	}

	return pm.coreStore.Set(ctx, userID, key, value)
}

// FormatForPrompt renders an adaptive instruction block for the system prompt.
// If every field is empty or "auto", it returns "" (nothing to inject).
func FormatForPrompt(profile AdaptiveProfile) string {
	type entry struct {
		label string
		value string
	}

	entries := []entry{
		{"Language", profile.Language},
		{"Style", profile.Style},
		{"Verbosity", profile.Verbosity},
		{"Technical depth", profile.TechnicalDepth},
	}

	var lines []string
	for _, e := range entries {
		if e.value == "" || strings.EqualFold(e.value, "auto") {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", e.label, e.value))
	}

	if len(lines) == 0 {
		return ""
	}

	return "### User Preferences\n" + strings.Join(lines, "\n")
}
