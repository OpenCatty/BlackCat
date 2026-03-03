package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// Integration tests for the configure command:
// Flag parsing, help output, and command structure validation.
// These tests verify the cobra command structure without requiring
// interactive input or real OAuth flows.

func TestIntegrationConfigureHelpOutput(t *testing.T) {
	// Reset the root command output for capture
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Run configure --help
	rootCmd.SetArgs([]string{"configure", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("configure --help failed: %v", err)
	}

	output := buf.String()

	// Verify help text contains key information
	if !strings.Contains(output, "configure") {
		t.Error("help output missing 'configure'")
	}
	if !strings.Contains(output, "--provider") {
		t.Error("help output missing '--provider' flag")
	}
	if !strings.Contains(output, "--api-key") {
		t.Error("help output missing '--api-key' flag")
	}
	if !strings.Contains(output, "--model") {
		t.Error("help output missing '--model' flag")
	}
	if !strings.Contains(output, "openai") {
		t.Error("help output missing 'openai' provider")
	}
	if !strings.Contains(output, "copilot") {
		t.Error("help output missing 'copilot' provider")
	}
}

func TestIntegrationConfigureFlagParsing(t *testing.T) {
	// Verify the configure command has the expected flags
	cmd := configureCmd

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag == nil {
		t.Fatal("missing --provider flag")
	}
	if providerFlag.DefValue != "" {
		t.Errorf("expected empty default for --provider, got %q", providerFlag.DefValue)
	}

	apiKeyFlag := cmd.Flags().Lookup("api-key")
	if apiKeyFlag == nil {
		t.Fatal("missing --api-key flag")
	}
	if apiKeyFlag.DefValue != "" {
		t.Errorf("expected empty default for --api-key, got %q", apiKeyFlag.DefValue)
	}

	modelFlag := cmd.Flags().Lookup("model")
	if modelFlag == nil {
		t.Fatal("missing --model flag")
	}
	if modelFlag.DefValue != "" {
		t.Errorf("expected empty default for --model, got %q", modelFlag.DefValue)
	}
}

func TestIntegrationConfigureCommandRegistered(t *testing.T) {
	// Verify configure command is registered as a subcommand of root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "configure" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("configure command not registered as subcommand of root")
	}
}

func TestIntegrationConfigureProviderList(t *testing.T) {
	// Verify allProviders contains expected providers
	expectedProviders := map[string]bool{
		"openai":      false,
		"anthropic":   false,
		"copilot":     false,
		"antigravity": false,
		"gemini":      false,
		"zen":         false,
		"openrouter":  false,
		"ollama":      false,
	}

	for _, p := range allProviders {
		if _, ok := expectedProviders[p.Name]; ok {
			expectedProviders[p.Name] = true
		}
	}

	for name, found := range expectedProviders {
		if !found {
			t.Errorf("expected provider %q not found in allProviders", name)
		}
	}
}

func TestIntegrationConfigureProviderAuthMethods(t *testing.T) {
	// Verify auth methods are correct for each provider
	authMethods := map[string]string{
		"openai":      "api-key",
		"anthropic":   "api-key",
		"copilot":     "oauth-device",
		"antigravity": "oauth-pkce",
		"gemini":      "api-key",
		"zen":         "api-key",
		"openrouter":  "api-key",
		"ollama":      "none",
	}

	for _, p := range allProviders {
		expected, ok := authMethods[p.Name]
		if !ok {
			continue
		}
		if p.AuthMethod != expected {
			t.Errorf("provider %s: expected auth method %q, got %q", p.Name, expected, p.AuthMethod)
		}
	}
}

func TestIntegrationConfigureProviderDefaultModels(t *testing.T) {
	// Verify key providers have default models set
	for _, p := range allProviders {
		switch p.Name {
		case "openai":
			if p.DefaultModel == "" {
				t.Errorf("provider %s should have a default model", p.Name)
			}
		case "copilot":
			if p.DefaultModel == "" {
				t.Errorf("provider %s should have a default model", p.Name)
			}
		case "gemini":
			if p.DefaultModel == "" {
				t.Errorf("provider %s should have a default model", p.Name)
			}
		case "zen":
			if p.DefaultModel == "" {
				t.Errorf("provider %s should have a default model", p.Name)
			}
		}
	}
}

func TestIntegrationRootCommandHasSubcommands(t *testing.T) {
	// Verify root command has configure and other expected subcommands
	expectedCmds := []string{"configure", "daemon", "init", "vault"}
	for _, name := range expectedCmds {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name || strings.HasPrefix(cmd.Use, name+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", name)
		}
	}
}
