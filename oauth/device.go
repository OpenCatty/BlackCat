// Package oauth provides OAuth flow engines for BlackCat LLM providers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DeviceFlowConfig holds parameters for the RFC 8628 device authorization flow.
type DeviceFlowConfig struct {
	ClientID      string
	DeviceCodeURL string        // e.g., "https://github.com/login/device/code"
	TokenURL      string        // e.g., "https://github.com/login/oauth/access_token"
	Scopes        []string      // e.g., ["read:user"]
	PollInterval  time.Duration // Default 5s; increased on slow_down
}

// DeviceCodeResponse holds the device code endpoint response per RFC 8628.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"` // Seconds until device code expires
	Interval        int    `json:"interval"`   // Poll interval in seconds
}

// deviceTokenResponse is the internal representation of the token endpoint response.
type deviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

// RequestDeviceCode initiates the device authorization flow per RFC 8628.
// It POSTs to the device code endpoint and returns the device/user codes for display.
func RequestDeviceCode(ctx context.Context, cfg DeviceFlowConfig) (*DeviceCodeResponse, error) {
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("device flow: client_id is required")
	}
	if cfg.DeviceCodeURL == "" {
		return nil, fmt.Errorf("device flow: device_code_url is required")
	}

	data := url.Values{
		"client_id": {cfg.ClientID},
	}
	if len(cfg.Scopes) > 0 {
		data.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.DeviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("device flow: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device flow: request device code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("device flow: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device flow: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var dcr DeviceCodeResponse
	if err := json.Unmarshal(body, &dcr); err != nil {
		return nil, fmt.Errorf("device flow: decode response: %w", err)
	}

	if dcr.DeviceCode == "" {
		return nil, fmt.Errorf("device flow: empty device_code in response")
	}

	return &dcr, nil
}

// PollForToken polls the token endpoint until the user completes authorization.
// It implements the polling logic from RFC 8628 section 3.5:
//   - "authorization_pending": continue polling
//   - "slow_down": increase interval by 5 seconds
//   - "expired_token": return error
//   - "access_denied": return error
func PollForToken(ctx context.Context, cfg DeviceFlowConfig, deviceCode string) (*TokenSet, error) {
	if cfg.TokenURL == "" {
		return nil, fmt.Errorf("device flow: token_url is required")
	}
	if deviceCode == "" {
		return nil, fmt.Errorf("device flow: device_code is required")
	}

	interval := cfg.PollInterval
	if interval == 0 {
		interval = 5 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		tokenSet, pollErr := pollOnce(ctx, cfg, deviceCode)
		if pollErr != nil {
			// Check for retryable vs terminal errors
			switch {
			case isPollError(pollErr, "authorization_pending"):
				continue
			case isPollError(pollErr, "slow_down"):
				interval += 5 * time.Second
				continue
			case isPollError(pollErr, "expired_token"):
				return nil, fmt.Errorf("device flow: device code expired — restart the flow")
			case isPollError(pollErr, "access_denied"):
				return nil, fmt.Errorf("device flow: user denied authorization")
			default:
				return nil, pollErr
			}
		}

		return tokenSet, nil
	}
}

// pollError is a typed error for OAuth error responses during polling.
type pollError struct {
	code string
}

func (e *pollError) Error() string {
	return fmt.Sprintf("device flow poll: %s", e.code)
}

func isPollError(err error, code string) bool {
	if pe, ok := err.(*pollError); ok {
		return pe.code == code
	}
	return false
}

// pollOnce makes a single token poll request.
func pollOnce(ctx context.Context, cfg DeviceFlowConfig, deviceCode string) (*TokenSet, error) {
	data := url.Values{
		"client_id":   {cfg.ClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("device flow: create poll request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device flow: poll request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("device flow: read poll response: %w", err)
	}

	var tokenResp deviceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("device flow: decode poll response: %w", err)
	}

	// Check for OAuth error codes
	if tokenResp.Error != "" {
		return nil, &pollError{code: tokenResp.Error}
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("device flow: empty access_token in response")
	}

	return &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
	}, nil
}
