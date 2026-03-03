package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/startower-observability/blackcat/security"
)

// vaultKeyPrefix is the prefix for all OAuth token keys in the vault.
// Convention: "oauth.<provider>" (e.g., "oauth.copilot", "oauth.antigravity").
const vaultKeyPrefix = "oauth."

// ErrTokenNotFound is returned when a token for a provider is not in the vault.
var ErrTokenNotFound = errors.New("oauth token not found")

// VaultTokenStore implements TokenStore using the security.Vault backend.
// Tokens are JSON-encoded and stored as string values in the vault's map[string]string.
type VaultTokenStore struct {
	vault *security.Vault
}

// NewVaultTokenStore creates a new VaultTokenStore backed by the given vault.
func NewVaultTokenStore(vault *security.Vault) *VaultTokenStore {
	return &VaultTokenStore{vault: vault}
}

// Save stores a token set for the given provider in the vault.
func (s *VaultTokenStore) Save(provider string, tokens TokenSet) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("marshal token for %s: %w", provider, err)
	}
	return s.vault.Set(vaultKeyPrefix+provider, string(data))
}

// Load retrieves a token set for the given provider from the vault.
func (s *VaultTokenStore) Load(provider string) (*TokenSet, error) {
	raw, err := s.vault.Get(vaultKeyPrefix + provider)
	if err != nil {
		if errors.Is(err, security.ErrVaultKeyNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("load token for %s: %w", provider, err)
	}

	var tokens TokenSet
	if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
		return nil, fmt.Errorf("unmarshal token for %s: %w", provider, err)
	}
	return &tokens, nil
}

// Delete removes the token set for the given provider from the vault.
func (s *VaultTokenStore) Delete(provider string) error {
	return s.vault.Delete(vaultKeyPrefix + provider)
}

// List returns the provider names that have stored tokens in the vault.
func (s *VaultTokenStore) List() ([]string, error) {
	keys := s.vault.List()
	var providers []string
	for _, key := range keys {
		if strings.HasPrefix(key, vaultKeyPrefix) {
			providers = append(providers, strings.TrimPrefix(key, vaultKeyPrefix))
		}
	}
	return providers, nil
}
