// Package oauth provides OAuth token types and vault-backed storage
// for BlackCat's OAuth-authenticated LLM providers.
package oauth

import "time"

// TokenSet holds the credentials obtained from an OAuth flow.
// It is JSON-encoded and stored as a string in the vault.
type TokenSet struct {
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken,omitempty"`
	TokenType    string            `json:"tokenType,omitempty"` // e.g., "Bearer"
	Expiry       time.Time         `json:"expiry,omitempty"`
	Email        string            `json:"email,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"` // Provider-specific fields
}

// IsExpired reports whether the token has expired.
// A zero Expiry means the token never expires.
func (t *TokenSet) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().After(t.Expiry)
}

// TokenStore is the interface for persisting and retrieving OAuth tokens.
type TokenStore interface {
	// Save stores a token set for the given provider.
	Save(provider string, tokens TokenSet) error

	// Load retrieves a token set for the given provider.
	// Returns an error if the provider token is not found.
	Load(provider string) (*TokenSet, error)

	// Delete removes the token set for the given provider.
	Delete(provider string) error

	// List returns the provider names that have stored tokens.
	List() ([]string, error)
}
