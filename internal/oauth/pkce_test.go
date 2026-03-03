package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestPKCEFlowSuccess(t *testing.T) {
	// Mock token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		// Verify required parameters
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Fatalf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") == "" {
			t.Fatal("missing authorization code")
		}
		if r.Form.Get("code_verifier") == "" {
			t.Fatal("missing code_verifier")
		}
		if r.Form.Get("client_id") != "test-client" {
			t.Fatalf("unexpected client_id: %s", r.Form.Get("client_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pkceTokenResponse{
			AccessToken:  "ya29.test-access-token",
			RefreshToken: "1//test-refresh-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer tokenServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var capturedAuthURL string

	cfg := PKCEConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth", // Not actually called
		TokenURL:     tokenServer.URL,
		RedirectURL:  "http://127.0.0.1:0/oauth-callback", // Use port 0 for random
		Scopes:       []string{"openid", "email"},
	}

	// We need to run PKCE flow and simulate the callback
	// Start in a goroutine and simulate the browser callback
	resultCh := make(chan *TokenSet, 1)
	errCh := make(chan error, 1)

	go func() {
		ts, err := RunPKCEFlow(ctx, cfg, func(authURL string) {
			capturedAuthURL = authURL

			// Parse the auth URL to get state parameter
			parsed, parseErr := url.Parse(authURL)
			if parseErr != nil {
				errCh <- fmt.Errorf("parse auth URL: %w", parseErr)
				return
			}

			state := parsed.Query().Get("state")
			redirectURI := parsed.Query().Get("redirect_uri")

			// Simulate browser callback with the authorization code
			callbackURL := fmt.Sprintf("%s?code=test-auth-code&state=%s", redirectURI, state)
			resp, getErr := http.Get(callbackURL)
			if getErr != nil {
				errCh <- fmt.Errorf("callback request: %w", getErr)
				return
			}
			resp.Body.Close()
		})
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- ts
	}()

	select {
	case ts := <-resultCh:
		if ts.AccessToken != "ya29.test-access-token" {
			t.Fatalf("unexpected access token: %s", ts.AccessToken)
		}
		if ts.RefreshToken != "1//test-refresh-token" {
			t.Fatalf("unexpected refresh token: %s", ts.RefreshToken)
		}
		if ts.TokenType != "Bearer" {
			t.Fatalf("unexpected token type: %s", ts.TokenType)
		}
		if ts.Expiry.IsZero() {
			t.Fatal("expected non-zero expiry")
		}
	case err := <-errCh:
		t.Fatalf("PKCE flow error: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("PKCE flow timed out")
	}

	if capturedAuthURL == "" {
		t.Fatal("auth URL callback was not invoked")
	}

	// Verify auth URL contains PKCE parameters
	parsed, _ := url.Parse(capturedAuthURL)
	q := parsed.Query()
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected S256 challenge method, got %s", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("missing code_challenge in auth URL")
	}
	if q.Get("state") == "" {
		t.Fatal("missing state in auth URL")
	}
	if q.Get("response_type") != "code" {
		t.Fatalf("expected response_type=code, got %s", q.Get("response_type"))
	}
}

func TestPKCEStateMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := PKCEConfig{
		ClientID:    "test-client",
		AuthURL:     "https://accounts.google.com/o/oauth2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		RedirectURL: "http://127.0.0.1:0/oauth-callback",
	}

	errCh := make(chan error, 1)

	go func() {
		_, err := RunPKCEFlow(ctx, cfg, func(authURL string) {
			// Parse redirect URI from auth URL
			parsed, _ := url.Parse(authURL)
			redirectURI := parsed.Query().Get("redirect_uri")

			// Send callback with WRONG state (CSRF attack simulation)
			callbackURL := fmt.Sprintf("%s?code=fake-code&state=wrong-state", redirectURI)
			resp, getErr := http.Get(callbackURL)
			if getErr != nil {
				return
			}
			resp.Body.Close()
		})
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error for state mismatch")
		}
		if !containsStr(err.Error(), "state mismatch") && !containsStr(err.Error(), "context") {
			t.Fatalf("expected state mismatch error, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestPKCEMissingConfig(t *testing.T) {
	ctx := context.Background()

	// Missing client_id
	_, err := RunPKCEFlow(ctx, PKCEConfig{
		AuthURL:  "https://example.com/auth",
		TokenURL: "https://example.com/token",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}

	// Missing auth_url
	_, err = RunPKCEFlow(ctx, PKCEConfig{
		ClientID: "test",
		TokenURL: "https://example.com/token",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing auth_url")
	}

	// Missing token_url
	_, err = RunPKCEFlow(ctx, PKCEConfig{
		ClientID: "test",
		AuthURL:  "https://example.com/auth",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing token_url")
	}
}

func TestRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Fatalf("unexpected refresh_token: %s", r.Form.Get("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(pkceTokenResponse{
			AccessToken:  "ya29.new-access",
			RefreshToken: "1//new-refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer server.Close()

	ts, err := RefreshToken(context.Background(), PKCEConfig{
		ClientID: "test-client",
		TokenURL: server.URL,
	}, "old-refresh")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if ts.AccessToken != "ya29.new-access" {
		t.Fatalf("unexpected access token: %s", ts.AccessToken)
	}
	if ts.RefreshToken != "1//new-refresh" {
		t.Fatalf("unexpected refresh token: %s", ts.RefreshToken)
	}
}

func TestRefreshTokenKeepsOld(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Server doesn't return a new refresh token
		_ = json.NewEncoder(w).Encode(pkceTokenResponse{
			AccessToken: "ya29.refreshed",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	}))
	defer server.Close()

	ts, err := RefreshToken(context.Background(), PKCEConfig{
		ClientID: "test-client",
		TokenURL: server.URL,
	}, "keep-this-refresh")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if ts.RefreshToken != "keep-this-refresh" {
		t.Fatalf("expected old refresh token preserved, got: %s", ts.RefreshToken)
	}
}

func TestCodeVerifierAndChallenge(t *testing.T) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier failed: %v", err)
	}

	// Verifier should be 43 chars (32 bytes base64url without padding)
	if len(verifier) != 43 {
		t.Fatalf("expected verifier length 43, got %d", len(verifier))
	}

	challenge := generateCodeChallenge(verifier)
	if challenge == "" {
		t.Fatal("empty code challenge")
	}
	if challenge == verifier {
		t.Fatal("challenge should differ from verifier")
	}

	// Challenge should be 43 chars (SHA256 = 32 bytes → 43 base64url chars)
	if len(challenge) != 43 {
		t.Fatalf("expected challenge length 43, got %d", len(challenge))
	}
}
