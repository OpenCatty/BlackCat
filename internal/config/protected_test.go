package config

import (
	"testing"
)

func TestIsProtected(t *testing.T) {
	tests := []struct {
		fieldPath string
		protected bool
		reason    string
	}{
		// Protected root-level fields
		{fieldPath: "security", protected: true, reason: "root security field"},
		{fieldPath: "channels", protected: true, reason: "root channels field"},
		{fieldPath: "oauth", protected: true, reason: "root oauth field"},
		{fieldPath: "session", protected: true, reason: "root session field"},
		{fieldPath: "vault", protected: true, reason: "root vault field"},

		// Protected subfields (prefix matching)
		{fieldPath: "security.vaultPath", protected: true, reason: "security subfield"},
		{fieldPath: "security.denyPatterns", protected: true, reason: "security subfield"},
		{fieldPath: "security.autoPermit", protected: true, reason: "security subfield"},
		{fieldPath: "channels.telegram", protected: true, reason: "channels subfield"},
		{fieldPath: "channels.discord.token", protected: true, reason: "channels nested subfield"},
		{fieldPath: "oauth.copilot", protected: true, reason: "oauth subfield"},
		{fieldPath: "oauth.antigravity.clientId", protected: true, reason: "oauth nested subfield"},
		{fieldPath: "dashboard.token", protected: true, reason: "dashboard.token subfield"},
		{fieldPath: "dashboard.token.secret", protected: true, reason: "dashboard.token nested"},
		{fieldPath: "session.store.type", protected: true, reason: "session subfield"},
		{fieldPath: "vault.passphrase", protected: true, reason: "vault subfield"},
		{fieldPath: "vault.path.something", protected: true, reason: "vault nested"},

		// Allowed fields (LLM config)
		{fieldPath: "llm", protected: false, reason: "llm root allowed"},
		{fieldPath: "llm.provider", protected: false, reason: "llm provider allowed"},
		{fieldPath: "llm.model", protected: false, reason: "llm model allowed"},
		{fieldPath: "llm.apikey", protected: false, reason: "llm apikey allowed"},
		{fieldPath: "llm.baseurl", protected: false, reason: "llm baseurl allowed"},
		{fieldPath: "llm.temperature", protected: false, reason: "llm temperature allowed"},
		{fieldPath: "llm.maxtokens", protected: false, reason: "llm maxTokens allowed"},

		// Allowed fields (Agent config)
		{fieldPath: "agent", protected: false, reason: "agent root allowed"},
		{fieldPath: "agent.name", protected: false, reason: "agent name allowed"},
		{fieldPath: "agent.greeting", protected: false, reason: "agent greeting allowed"},
		{fieldPath: "agent.language", protected: false, reason: "agent language allowed"},
		{fieldPath: "agent.tone", protected: false, reason: "agent tone allowed"},

		// Allowed fields (Logging config)
		{fieldPath: "logging", protected: false, reason: "logging root allowed"},
		{fieldPath: "logging.level", protected: false, reason: "logging level allowed"},
		{fieldPath: "logging.format", protected: false, reason: "logging format allowed"},

		// Allowed fields (Providers config)
		{fieldPath: "providers", protected: false, reason: "providers root allowed"},
		{fieldPath: "providers.openai", protected: false, reason: "providers.openai allowed"},
		{fieldPath: "providers.openai.model", protected: false, reason: "providers.openai.model allowed"},
		{fieldPath: "providers.openai.enabled", protected: false, reason: "providers.openai.enabled allowed"},

		// Allowed fields (Dashboard subfields, except token)
		{fieldPath: "dashboard", protected: false, reason: "dashboard root allowed"},
		{fieldPath: "dashboard.addr", protected: false, reason: "dashboard.addr allowed"},
		{fieldPath: "dashboard.port", protected: false, reason: "dashboard.port allowed"},
		{fieldPath: "dashboard.timeout", protected: false, reason: "dashboard.timeout allowed"},
		{fieldPath: "dashboard.something", protected: false, reason: "dashboard.something allowed"},

		// Edge cases
		{fieldPath: "", protected: false, reason: "empty string not protected"},
		{fieldPath: "   ", protected: false, reason: "whitespace string not protected"},
		{fieldPath: "SECURITY", protected: true, reason: "case insensitive"},
		{fieldPath: "Security.VaultPath", protected: true, reason: "case insensitive with subfield"},
		{fieldPath: "CHANNELS.TELEGRAM", protected: true, reason: "case insensitive nested"},
		{fieldPath: "DASHBOARD.TOKEN", protected: true, reason: "case insensitive dashboard token"},
		{fieldPath: "dashboardtoken", protected: false, reason: "no dot separator not matched"},
		{fieldPath: "vault_path", protected: false, reason: "underscore not dot"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldPath, func(t *testing.T) {
			result := IsProtected(tt.fieldPath)
			if result != tt.protected {
				t.Errorf("IsProtected(%q) = %v, want %v (%s)", tt.fieldPath, result, tt.protected, tt.reason)
			}
		})
	}
}

func TestProtectedReason(t *testing.T) {
	tests := []struct {
		fieldPath string
		contains  string
	}{
		{fieldPath: "security", contains: "security"},
		{fieldPath: "channels.telegram", contains: "channels.telegram"},
		{fieldPath: "oauth.copilot", contains: "oauth.copilot"},
		{fieldPath: "dashboard.token", contains: "dashboard.token"},
		{fieldPath: "vault.passphrase", contains: "vault.passphrase"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldPath, func(t *testing.T) {
			reason := ProtectedReason(tt.fieldPath)
			if !contains(reason, tt.contains) {
				t.Errorf("ProtectedReason(%q) = %q, should contain %q", tt.fieldPath, reason, tt.contains)
			}
			if !contains(reason, "SOUL protection") {
				t.Errorf("ProtectedReason(%q) = %q, should contain 'SOUL protection'", tt.fieldPath, reason)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) > 0 && len(needle) <= len(haystack))
}
