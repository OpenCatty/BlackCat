package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PKCEConfig holds parameters for the OAuth 2.0 PKCE authorization code flow.
type PKCEConfig struct {
	ClientID     string
	ClientSecret string // Optional: some providers require it even with PKCE
	AuthURL      string // e.g., "https://accounts.google.com/o/oauth2/auth"
	TokenURL     string // e.g., "https://oauth2.googleapis.com/token"
	RedirectURL  string // e.g., "http://127.0.0.1:51121/oauth-callback"
	Scopes       []string
}

// pkceTokenResponse is the internal token endpoint response format.
type pkceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// RunPKCEFlow orchestrates the full PKCE authorization code flow:
//  1. Generate code verifier and challenge
//  2. Start local HTTP callback server
//  3. Build authorization URL for the user to visit
//  4. Wait for callback with authorization code
//  5. Exchange code for tokens
//  6. Return TokenSet
//
// The authURLCallback is called with the URL the user should open in their browser.
// If nil, the URL is printed to stdout.
func RunPKCEFlow(ctx context.Context, cfg PKCEConfig, authURLCallback func(authURL string)) (*TokenSet, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("pkce flow: client_id is required")
	}
	if cfg.AuthURL == "" {
		return nil, fmt.Errorf("pkce flow: auth_url is required")
	}
	if cfg.TokenURL == "" {
		return nil, fmt.Errorf("pkce flow: token_url is required")
	}

	redirectURL := cfg.RedirectURL
	if redirectURL == "" {
		redirectURL = "http://127.0.0.1:51121/oauth-callback"
	}

	// Generate PKCE verifier and challenge
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("pkce flow: generate verifier: %w", err)
	}
	challenge := generateCodeChallenge(verifier)

	// Generate state for CSRF protection
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("pkce flow: generate state: %w", err)
	}

	// Parse redirect URL to get host:port for listener
	parsedURL, err := url.Parse(redirectURL)
	if err != nil {
		return nil, fmt.Errorf("pkce flow: parse redirect URL: %w", err)
	}

	// Start local callback server
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var server *http.Server

	mux := http.NewServeMux()
	mux.HandleFunc(parsedURL.Path, func(w http.ResponseWriter, r *http.Request) {
		// Verify state parameter (CSRF protection)
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("pkce flow: state mismatch (possible CSRF)")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}

		// Check for error in callback
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("pkce flow: OAuth error: %s — %s", errParam, errDesc)
			http.Error(w, "Authorization failed", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("pkce flow: no code in callback")
			http.Error(w, "No authorization code", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>Authorization successful!</h1><p>You can close this window.</p></body></html>"))
		codeCh <- code
	})

	listener, err := net.Listen("tcp", parsedURL.Host)
	if err != nil {
		return nil, fmt.Errorf("pkce flow: start callback server: %w", err)
	}

	// Resolve actual address (important when port 0 is used for testing)
	actualAddr := listener.Addr().String()
	redirectURL = fmt.Sprintf("http://%s%s", actualAddr, parsedURL.Path)

	server = &http.Server{Handler: mux}
	var serverWg sync.WaitGroup
	serverWg.Add(1)
	go func() {
		defer serverWg.Done()
		if sErr := server.Serve(listener); sErr != nil && sErr != http.ErrServerClosed {
			errCh <- fmt.Errorf("pkce flow: callback server error: %w", sErr)
		}
	}()

	defer func() {
		_ = server.Close()
		serverWg.Wait()
	}()

	// Build authorization URL
	authParams := url.Values{
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {redirectURL},
		"response_type":         {"code"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
		"access_type":           {"offline"}, // Request refresh token
	}
	if len(cfg.Scopes) > 0 {
		authParams.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	authURL := cfg.AuthURL + "?" + authParams.Encode()

	// Notify caller of auth URL
	if authURLCallback != nil {
		authURLCallback(authURL)
	} else {
		fmt.Printf("Open this URL in your browser to authorize:\n%s\n", authURL)
	}

	// Wait for callback or timeout/cancel
	var code string
	select {
	case code = <-codeCh:
		// Success — got authorization code
	case flowErr := <-errCh:
		return nil, flowErr
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Exchange code for tokens
	return exchangeCode(ctx, cfg, redirectURL, code, verifier)
}

// RefreshToken exchanges a refresh token for a new access token.
func RefreshToken(ctx context.Context, cfg PKCEConfig, refreshToken string) (*TokenSet, error) {
	if cfg.TokenURL == "" {
		return nil, fmt.Errorf("pkce refresh: token_url is required")
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("pkce refresh: refresh_token is required")
	}

	data := url.Values{
		"client_id":     {cfg.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if cfg.ClientSecret != "" {
		data.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("pkce refresh: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pkce refresh: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pkce refresh: read response: %w", err)
	}

	var tokenResp pkceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("pkce refresh: decode response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("pkce refresh: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("pkce refresh: empty access_token")
	}

	ts := &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
	}
	if ts.RefreshToken == "" {
		ts.RefreshToken = refreshToken // Keep old refresh token if not rotated
	}
	if tokenResp.ExpiresIn > 0 {
		ts.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return ts, nil
}

// exchangeCode exchanges an authorization code for tokens.
func exchangeCode(ctx context.Context, cfg PKCEConfig, redirectURL, code, verifier string) (*TokenSet, error) {
	data := url.Values{
		"client_id":     {cfg.ClientID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURL},
		"code_verifier": {verifier},
	}
	if cfg.ClientSecret != "" {
		data.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("pkce exchange: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pkce exchange: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pkce exchange: read response: %w", err)
	}

	var tokenResp pkceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("pkce exchange: decode response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("pkce exchange: %s — %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("pkce exchange: empty access_token")
	}

	ts := &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
	}
	if tokenResp.ExpiresIn > 0 {
		ts.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return ts, nil
}

// generateCodeVerifier generates a cryptographically random code verifier (43-128 chars).
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32) // 32 bytes → 43 chars base64url
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge computes the S256 code challenge from a verifier.
func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState generates a random state parameter for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
