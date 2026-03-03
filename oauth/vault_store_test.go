package oauth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/security"
)

func newTestVault(t *testing.T) *security.Vault {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-vault.json")
	v, err := security.NewVault(path, "test-passphrase")
	if err != nil {
		t.Fatalf("failed to create test vault: %v", err)
	}
	return v
}

func TestVaultTokenStore(t *testing.T) {
	vault := newTestVault(t)
	store := NewVaultTokenStore(vault)

	// Save a token set.
	original := TokenSet{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		Email:        "user@example.com",
		Extra:        map[string]string{"scope": "read:user"},
	}

	if err := store.Save("copilot", original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load it back.
	loaded, err := store.Load("copilot")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify all fields.
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if loaded.TokenType != original.TokenType {
		t.Errorf("TokenType: got %q, want %q", loaded.TokenType, original.TokenType)
	}
	if !loaded.Expiry.Equal(original.Expiry) {
		t.Errorf("Expiry: got %v, want %v", loaded.Expiry, original.Expiry)
	}
	if loaded.Email != original.Email {
		t.Errorf("Email: got %q, want %q", loaded.Email, original.Email)
	}
	if loaded.Extra["scope"] != "read:user" {
		t.Errorf("Extra[scope]: got %q, want %q", loaded.Extra["scope"], "read:user")
	}
}

func TestVaultTokenStoreNotFound(t *testing.T) {
	vault := newTestVault(t)
	store := NewVaultTokenStore(vault)

	_, err := store.Load("nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestVaultTokenStoreDelete(t *testing.T) {
	vault := newTestVault(t)
	store := NewVaultTokenStore(vault)

	// Save and then delete.
	token := TokenSet{AccessToken: "to-delete"}
	if err := store.Save("copilot", token); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := store.Delete("copilot"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Load should return not found.
	_, err := store.Load("copilot")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound after delete, got %v", err)
	}
}

func TestVaultTokenStoreList(t *testing.T) {
	vault := newTestVault(t)
	store := NewVaultTokenStore(vault)

	// Empty list initially.
	providers, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}

	// Save two tokens.
	if err := store.Save("copilot", TokenSet{AccessToken: "a"}); err != nil {
		t.Fatalf("Save copilot failed: %v", err)
	}
	if err := store.Save("antigravity", TokenSet{AccessToken: "b"}); err != nil {
		t.Fatalf("Save antigravity failed: %v", err)
	}

	// List should return both.
	providers, err = store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	// Check both are present (order may vary).
	found := map[string]bool{}
	for _, p := range providers {
		found[p] = true
	}
	if !found["copilot"] {
		t.Error("expected 'copilot' in list")
	}
	if !found["antigravity"] {
		t.Error("expected 'antigravity' in list")
	}
}

func TestVaultTokenStoreMultipleProviders(t *testing.T) {
	vault := newTestVault(t)
	store := NewVaultTokenStore(vault)

	// Save tokens for two different providers.
	if err := store.Save("copilot", TokenSet{AccessToken: "copilot-token"}); err != nil {
		t.Fatalf("Save copilot failed: %v", err)
	}
	if err := store.Save("antigravity", TokenSet{AccessToken: "ag-token"}); err != nil {
		t.Fatalf("Save antigravity failed: %v", err)
	}

	// Load each independently.
	c, err := store.Load("copilot")
	if err != nil {
		t.Fatalf("Load copilot failed: %v", err)
	}
	if c.AccessToken != "copilot-token" {
		t.Errorf("copilot AccessToken: got %q, want %q", c.AccessToken, "copilot-token")
	}

	a, err := store.Load("antigravity")
	if err != nil {
		t.Fatalf("Load antigravity failed: %v", err)
	}
	if a.AccessToken != "ag-token" {
		t.Errorf("antigravity AccessToken: got %q, want %q", a.AccessToken, "ag-token")
	}
}

func TestTokenSetIsExpired(t *testing.T) {
	// Zero expiry = never expires.
	tok := &TokenSet{AccessToken: "a"}
	if tok.IsExpired() {
		t.Error("zero expiry should not be expired")
	}

	// Past expiry = expired.
	tok.Expiry = time.Now().Add(-1 * time.Hour)
	if !tok.IsExpired() {
		t.Error("past expiry should be expired")
	}

	// Future expiry = not expired.
	tok.Expiry = time.Now().Add(1 * time.Hour)
	if tok.IsExpired() {
		t.Error("future expiry should not be expired")
	}
}

// TestVaultTokenStoreIgnoresNonOAuthKeys verifies that non-oauth vault keys are not listed.
func TestVaultTokenStoreIgnoresNonOAuthKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.json")
	vault, err := security.NewVault(path, "test")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	// Set a non-oauth key directly in vault.
	if err := vault.Set("some.other.key", "value"); err != nil {
		t.Fatalf("set non-oauth key: %v", err)
	}

	store := NewVaultTokenStore(vault)

	// Save an oauth token.
	if err := store.Save("copilot", TokenSet{AccessToken: "x"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// List should only return oauth providers.
	providers, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(providers) != 1 || providers[0] != "copilot" {
		t.Errorf("expected [copilot], got %v", providers)
	}

	// Clean up (suppress unused import warning).
	_ = os.Remove(path)
}
