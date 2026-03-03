package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Integration tests for Device Flow + PKCE Flow with full HTTP server mocks.
// These test the complete flows end-to-end without requiring real OAuth providers.

// --- Device Flow Integration Tests ---

func TestIntegrationDeviceFlowFullFlow(t *testing.T) {
	pollCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/login/device/code":
			// Verify form data
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("client_id") != "test-client-id" {
				t.Errorf("unexpected client_id: %s", r.Form.Get("client_id"))
			}
			if r.Form.Get("scope") != "read:user" {
				t.Errorf("unexpected scope: %s", r.Form.Get("scope"))
			}

			_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
				DeviceCode:      "dev_code_12345",
				UserCode:        "ABCD-EFGH",
				VerificationURI: "https://example.com/verify",
				ExpiresIn:       900,
				Interval:        1,
			})

		case "/login/oauth/access_token":
			count := atomic.AddInt32(&pollCount, 1)

			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("device_code") != "dev_code_12345" {
				t.Errorf("unexpected device_code: %s", r.Form.Get("device_code"))
			}
			if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Errorf("unexpected grant_type: %s", r.Form.Get("grant_type"))
			}

			if count <= 2 {
				// First two polls: authorization pending
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			} else {
				// Third poll: success
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"access_token":  "gho_final_token_12345",
					"token_type":    "bearer",
					"scope":         "read:user",
					"refresh_token": "ghr_refresh_token",
				})
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client-id",
		DeviceCodeURL: server.URL + "/login/device/code",
		TokenURL:      server.URL + "/login/oauth/access_token",
		Scopes:        []string{"read:user"},
		PollInterval:  50 * time.Millisecond, // Fast polling for tests
	}

	// Step 1: Request device code
	dcr, err := RequestDeviceCode(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RequestDeviceCode failed: %v", err)
	}

	if dcr.DeviceCode != "dev_code_12345" {
		t.Fatalf("unexpected device code: %s", dcr.DeviceCode)
	}
	if dcr.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected user code: %s", dcr.UserCode)
	}
	if dcr.VerificationURI != "https://example.com/verify" {
		t.Fatalf("unexpected verification URI: %s", dcr.VerificationURI)
	}

	// Step 2: Poll for token (will succeed after 2 pending responses)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenSet, err := PollForToken(ctx, cfg, dcr.DeviceCode)
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}

	if tokenSet.AccessToken != "gho_final_token_12345" {
		t.Fatalf("unexpected access token: %s", tokenSet.AccessToken)
	}
	if tokenSet.RefreshToken != "ghr_refresh_token" {
		t.Fatalf("unexpected refresh token: %s", tokenSet.RefreshToken)
	}
	if tokenSet.TokenType != "bearer" {
		t.Fatalf("unexpected token type: %s", tokenSet.TokenType)
	}

	// Should have polled 3 times (2 pending + 1 success)
	if count := atomic.LoadInt32(&pollCount); count != 3 {
		t.Fatalf("expected 3 poll attempts, got %d", count)
	}
}

func TestIntegrationDeviceFlowSlowDown(t *testing.T) {
	pollCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		count := atomic.AddInt32(&pollCount, 1)

		switch {
		case count == 1:
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
		case count == 2:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "gho_after_slowdown",
				"token_type":   "bearer",
			})
		}
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL,
		PollInterval:  50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tokenSet, err := PollForToken(ctx, cfg, "dc_test")
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}

	if tokenSet.AccessToken != "gho_after_slowdown" {
		t.Fatalf("unexpected token: %s", tokenSet.AccessToken)
	}
}

func TestIntegrationDeviceFlowExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL,
		PollInterval:  50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := PollForToken(ctx, cfg, "dc_expired")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got: %v", err)
	}
}

func TestIntegrationDeviceFlowAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL,
		PollInterval:  50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := PollForToken(ctx, cfg, "dc_denied")
	if err == nil {
		t.Fatal("expected error for access denied")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denied error, got: %v", err)
	}
}

func TestIntegrationDeviceFlowContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Always pending — context should cancel before success
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL,
		PollInterval:  50 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := PollForToken(ctx, cfg, "dc_cancel")
	if err == nil {
		t.Fatal("expected error for context cancellation")
	}
}

func TestIntegrationDeviceFlowHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer server.Close()

	cfg := DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL + "/login/device/code",
		TokenURL:      server.URL,
	}

	_, err := RequestDeviceCode(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

// --- PKCE Flow Integration Tests ---

func TestIntegrationPKCEFlowFullFlow(t *testing.T) {
	// Mock OAuth token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		// Verify PKCE parameters
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "auth_code_from_browser" {
			t.Errorf("unexpected code: %s", r.Form.Get("code"))
		}
		if r.Form.Get("code_verifier") == "" {
			t.Error("missing code_verifier")
		}
		if r.Form.Get("client_id") != "test-pkce-client" {
			t.Errorf("unexpected client_id: %s", r.Form.Get("client_id"))
		}
		if r.Form.Get("redirect_uri") == "" {
			t.Error("missing redirect_uri")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "ya29.pkce_access_token",
			"refresh_token": "1//pkce_refresh_token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	cfg := PKCEConfig{
		ClientID:    "test-pkce-client",
		AuthURL:     "https://accounts.google.com/o/oauth2/auth", // Not actually hit
		TokenURL:    tokenServer.URL,
		RedirectURL: "http://127.0.0.1:0/oauth-callback", // Port 0 for dynamic allocation
		Scopes:      []string{"openid", "email"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var capturedAuthURL string

	// Run PKCE flow in a goroutine
	resultCh := make(chan *TokenSet, 1)
	errCh := make(chan error, 1)

	go func() {
		ts, err := RunPKCEFlow(ctx, cfg, func(authURL string) {
			capturedAuthURL = authURL

			// Simulate browser callback: extract the redirect URI and state, then POST back
			parsed, _ := url.Parse(authURL)
			state := parsed.Query().Get("state")
			redirectURI := parsed.Query().Get("redirect_uri")

			// Simulate browser completing auth and being redirected back
			callbackURL := fmt.Sprintf("%s?code=auth_code_from_browser&state=%s", redirectURI, state)
			resp, err := http.Get(callbackURL)
			if err != nil {
				errCh <- fmt.Errorf("callback GET: %w", err)
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
		if ts.AccessToken != "ya29.pkce_access_token" {
			t.Fatalf("unexpected access token: %s", ts.AccessToken)
		}
		if ts.RefreshToken != "1//pkce_refresh_token" {
			t.Fatalf("unexpected refresh token: %s", ts.RefreshToken)
		}
		if ts.TokenType != "Bearer" {
			t.Fatalf("unexpected token type: %s", ts.TokenType)
		}
	case err := <-errCh:
		t.Fatalf("PKCE flow failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("PKCE flow timed out")
	}

	// Verify auth URL was constructed correctly
	if capturedAuthURL == "" {
		t.Fatal("auth URL was not captured")
	}
	parsed, _ := url.Parse(capturedAuthURL)
	if parsed.Query().Get("client_id") != "test-pkce-client" {
		t.Errorf("unexpected client_id in auth URL: %s", parsed.Query().Get("client_id"))
	}
	if parsed.Query().Get("response_type") != "code" {
		t.Errorf("unexpected response_type: %s", parsed.Query().Get("response_type"))
	}
	if parsed.Query().Get("code_challenge_method") != "S256" {
		t.Errorf("unexpected code_challenge_method: %s", parsed.Query().Get("code_challenge_method"))
	}
	if parsed.Query().Get("code_challenge") == "" {
		t.Error("missing code_challenge in auth URL")
	}
	if parsed.Query().Get("state") == "" {
		t.Error("missing state in auth URL")
	}
	if parsed.Query().Get("access_type") != "offline" {
		t.Error("expected access_type=offline for refresh token")
	}
}

func TestIntegrationPKCEFlowStateMismatch(t *testing.T) {
	cfg := PKCEConfig{
		ClientID:    "test-client",
		AuthURL:     "https://accounts.google.com/o/oauth2/auth",
		TokenURL:    "https://unused.example.com/token",
		RedirectURL: "http://127.0.0.1:0/oauth-callback",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		_, err := RunPKCEFlow(ctx, cfg, func(authURL string) {
			parsed, _ := url.Parse(authURL)
			redirectURI := parsed.Query().Get("redirect_uri")

			// Send callback with WRONG state to trigger CSRF error
			callbackURL := fmt.Sprintf("%s?code=test_code&state=wrong_state", redirectURI)
			resp, _ := http.Get(callbackURL)
			if resp != nil {
				resp.Body.Close()
			}
		})
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error for state mismatch")
		}
		if !strings.Contains(err.Error(), "state mismatch") {
			t.Fatalf("expected state mismatch error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestIntegrationPKCERefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("unexpected grant_type: %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old_refresh_token" {
			t.Errorf("unexpected refresh_token: %s", r.Form.Get("refresh_token"))
		}
		if r.Form.Get("client_secret") != "test-secret" {
			t.Errorf("unexpected client_secret: %s", r.Form.Get("client_secret"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "ya29.new_access_token",
			"refresh_token": "new_refresh_token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	cfg := PKCEConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     server.URL,
	}

	ts, err := RefreshToken(context.Background(), cfg, "old_refresh_token")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if ts.AccessToken != "ya29.new_access_token" {
		t.Fatalf("unexpected access token: %s", ts.AccessToken)
	}
	if ts.RefreshToken != "new_refresh_token" {
		t.Fatalf("unexpected refresh token: %s", ts.RefreshToken)
	}
	if ts.Expiry.IsZero() {
		t.Fatal("expected non-zero expiry")
	}
	if ts.Expiry.Before(time.Now()) {
		t.Fatal("expected future expiry")
	}
}

func TestIntegrationPKCERefreshTokenKeepOld(t *testing.T) {
	// When the server doesn't return a new refresh token, the old one should be kept
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "ya29.new_access",
			"token_type":   "Bearer",
			"expires_in":   3600,
			// No refresh_token in response
		})
	}))
	defer server.Close()

	cfg := PKCEConfig{
		ClientID: "test-client",
		TokenURL: server.URL,
	}

	ts, err := RefreshToken(context.Background(), cfg, "old_refresh_token")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	// Should keep the old refresh token
	if ts.RefreshToken != "old_refresh_token" {
		t.Fatalf("expected old refresh token to be kept, got: %s", ts.RefreshToken)
	}
}

func TestIntegrationPKCERefreshTokenError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "invalid_grant",
			"error_description": "Token has been revoked",
		})
	}))
	defer server.Close()

	cfg := PKCEConfig{
		ClientID: "test-client",
		TokenURL: server.URL,
	}

	_, err := RefreshToken(context.Background(), cfg, "revoked_token")
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("expected invalid_grant error, got: %v", err)
	}
}

// --- PKCE Crypto Tests ---

func TestIntegrationPKCEVerifierAndChallenge(t *testing.T) {
	verifier1, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}
	verifier2, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier: %v", err)
	}

	// Verifiers should be unique
	if verifier1 == verifier2 {
		t.Fatal("expected unique verifiers")
	}

	// Verifier length should be valid (43 chars for 32 bytes base64url)
	if len(verifier1) < 43 || len(verifier1) > 128 {
		t.Fatalf("unexpected verifier length: %d", len(verifier1))
	}

	// Challenge should be deterministic for same verifier
	challenge1 := generateCodeChallenge(verifier1)
	challenge2 := generateCodeChallenge(verifier1)
	if challenge1 != challenge2 {
		t.Fatal("same verifier should produce same challenge")
	}

	// Different verifiers should produce different challenges
	challenge3 := generateCodeChallenge(verifier2)
	if challenge1 == challenge3 {
		t.Fatal("different verifiers should produce different challenges")
	}
}

func TestIntegrationPKCEStateGeneration(t *testing.T) {
	state1, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}

	if state1 == state2 {
		t.Fatal("states should be unique")
	}
	if state1 == "" || state2 == "" {
		t.Fatal("states should not be empty")
	}
}
