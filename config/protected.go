package config

import (
	"fmt"
	"strings"
)

// ProtectedFields lists YAML field path prefixes that the agent cannot modify.
// These are critical security, integration, and authentication fields.
var ProtectedFields = []string{
	"security",        // Security settings, deny patterns, auto-permit
	"channels",        // Channel configs (telegram, discord, whatsapp tokens)
	"oauth",           // OAuth provider settings (Copilot, Antigravity)
	"dashboard.token", // Dashboard auth token (only this subfield, not whole dashboard)
	"session",         // Session store settings
	"vault",           // Vault passphrase/path
}

// IsProtected returns true if the given YAML field path (dot-notation) is protected.
// It checks for exact matches or prefix matches (e.g., "security.vaultPath" is blocked
// because it starts with "security").
func IsProtected(fieldPath string) bool {
	normalized := strings.ToLower(strings.TrimSpace(fieldPath))
	if normalized == "" {
		return false
	}

	for _, prefix := range ProtectedFields {
		// Exact match or prefix match with dot separator
		if normalized == prefix || strings.HasPrefix(normalized, prefix+".") {
			return true
		}
	}
	return false
}

// ProtectedReason returns a human-readable message explaining why a field is protected.
func ProtectedReason(fieldPath string) string {
	return fmt.Sprintf("field %q is protected and cannot be modified by the agent (SOUL protection)", fieldPath)
}
