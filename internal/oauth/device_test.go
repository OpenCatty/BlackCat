package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content type: %s", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("client_id") != "test-client" {
			t.Fatalf("unexpected client_id: %s", r.Form.Get("client_id"))
		}
		if r.Form.Get("scope") != "read:user" {
			t.Fatalf("unexpected scope: %s", r.Form.Get("scope"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "DEVCODE123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	resp, err := RequestDeviceCode(context.Background(), DeviceFlowConfig{
		ClientID:      "test-client",
		DeviceCodeURL: server.URL,
		Scopes:        []string{"read:user"},
	})
	if err != nil {
		t.Fatalf("RequestDeviceCode failed: %v", err)
	}
	if resp.DeviceCode != "DEVCODE123" {
		t.Fatalf("unexpected device code: %s", resp.DeviceCode)
	}
	if resp.UserCode != "ABCD-1234" {
		t.Fatalf("unexpected user code: %s", resp.UserCode)
	}
	if resp.VerificationURI != "https://github.com/login/device" {
		t.Fatalf("unexpected verification URI: %s", resp.VerificationURI)
	}
}

func TestRequestDeviceCodeMissingClientID(t *testing.T) {
	_, err := RequestDeviceCode(context.Background(), DeviceFlowConfig{
		DeviceCodeURL: "http://example.com",
	})
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}
}

func TestPollForTokenSuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")

		if attempts < 3 {
			// First 2 attempts: authorization_pending
			_ = json.NewEncoder(w).Encode(deviceTokenResponse{
				Error: "authorization_pending",
			})
			return
		}

		// Third attempt: success
		_ = json.NewEncoder(w).Encode(deviceTokenResponse{
			AccessToken:  "gho_abc123",
			TokenType:    "bearer",
			Scope:        "read:user",
			RefreshToken: "",
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := PollForToken(ctx, DeviceFlowConfig{
		ClientID:     "test-client",
		TokenURL:     server.URL,
		PollInterval: 50 * time.Millisecond, // Fast polling for test
	}, "DEVCODE123")
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}
	if token.AccessToken != "gho_abc123" {
		t.Fatalf("unexpected access token: %s", token.AccessToken)
	}
	if token.TokenType != "bearer" {
		t.Fatalf("unexpected token type: %s", token.TokenType)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 poll attempts, got %d", attempts)
	}
}

func TestPollForTokenSlowDown(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")

		if attempts == 1 {
			_ = json.NewEncoder(w).Encode(deviceTokenResponse{
				Error: "slow_down",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(deviceTokenResponse{
			AccessToken: "gho_slow",
			TokenType:   "bearer",
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := PollForToken(ctx, DeviceFlowConfig{
		ClientID:     "test-client",
		TokenURL:     server.URL,
		PollInterval: 50 * time.Millisecond,
	}, "DEVCODE123")
	if err != nil {
		t.Fatalf("PollForToken failed: %v", err)
	}
	if token.AccessToken != "gho_slow" {
		t.Fatalf("unexpected token: %s", token.AccessToken)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestPollForTokenExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deviceTokenResponse{
			Error: "expired_token",
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := PollForToken(ctx, DeviceFlowConfig{
		ClientID:     "test-client",
		TokenURL:     server.URL,
		PollInterval: 50 * time.Millisecond,
	}, "DEVCODE123")
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !containsStr(err.Error(), "expired") {
		t.Fatalf("expected expired error, got: %v", err)
	}
}

func TestPollForTokenDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deviceTokenResponse{
			Error: "access_denied",
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := PollForToken(ctx, DeviceFlowConfig{
		ClientID:     "test-client",
		TokenURL:     server.URL,
		PollInterval: 50 * time.Millisecond,
	}, "DEVCODE123")
	if err == nil {
		t.Fatal("expected error for denied access")
	}
	if !containsStr(err.Error(), "denied") {
		t.Fatalf("expected denied error, got: %v", err)
	}
}

func TestPollForTokenMissingURL(t *testing.T) {
	_, err := PollForToken(context.Background(), DeviceFlowConfig{
		ClientID: "test-client",
	}, "DEVCODE123")
	if err == nil {
		t.Fatal("expected error for missing token URL")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
