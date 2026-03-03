package dashboard

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateSessionSecret(t *testing.T) {
	t.Parallel()

	secretA, err := generateSessionSecret()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(secretA) != 32 {
		t.Fatalf("expected 32-byte secret, got %d", len(secretA))
	}

	secretB, err := generateSessionSecret()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(secretB) != 32 {
		t.Fatalf("expected 32-byte secret, got %d", len(secretB))
	}

	if hmac.Equal(secretA, secretB) {
		t.Fatal("expected generated secrets to differ")
	}
}

func TestSignSession(t *testing.T) {
	t.Parallel()

	token := "top-secret-token"
	secret := []byte("session-signing-secret")
	signed := signSession(token, secret)

	parts := strings.Split(signed, ":")
	if len(parts) != 2 {
		t.Fatalf("expected two-part cookie value, got %q", signed)
	}

	hashPart, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("failed to decode hash part: %v", err)
	}

	sigPart, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode signature part: %v", err)
	}

	expectedHash := sha256.Sum256([]byte(token))
	if !hmac.Equal(hashPart, expectedHash[:]) {
		t.Fatal("token hash part does not match expected SHA256(token)")
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(expectedHash[:])
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(sigPart, expectedSig) {
		t.Fatal("signature part does not match expected HMAC-SHA256(token_hash)")
	}
}

func TestValidateSession(t *testing.T) {
	t.Parallel()

	token := "top-secret-token"
	secret := []byte("session-signing-secret")
	valid := signSession(token, secret)

	if !validateSession(valid, token, secret) {
		t.Fatal("expected valid signed cookie to pass validation")
	}

	if validateSession(valid, "wrong-token", secret) {
		t.Fatal("expected wrong token to fail validation")
	}

	tampered := valid + "tampered"
	if validateSession(tampered, token, secret) {
		t.Fatal("expected tampered cookie to fail validation")
	}
}

func TestSetSessionCookie(t *testing.T) {
	t.Parallel()

	secret := []byte("session-signing-secret")
	token := "top-secret-token"
	rr := httptest.NewRecorder()

	setSessionCookie(rr, token, secret, true)

	resp := rr.Result()
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", sessionCookieName, cookie.Name)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected path '/', got %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly=true")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected SameSiteStrictMode, got %v", cookie.SameSite)
	}
	if cookie.MaxAge != 86400 {
		t.Fatalf("expected MaxAge=86400, got %d", cookie.MaxAge)
	}
	if !cookie.Secure {
		t.Fatal("expected Secure=true")
	}
	if !validateSession(cookie.Value, token, secret) {
		t.Fatal("expected cookie value to be a valid session signature")
	}
}

func TestClearSessionCookie(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	clearSessionCookie(rr)

	resp := rr.Result()
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != sessionCookieName {
		t.Fatalf("expected cookie name %q, got %q", sessionCookieName, cookie.Name)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected path '/', got %q", cookie.Path)
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("expected MaxAge=-1, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Fatalf("expected empty cookie value, got %q", cookie.Value)
	}
}

func TestIsSecureRequest(t *testing.T) {
	t.Parallel()

	secureTLSReq := httptest.NewRequest(http.MethodGet, "https://example.com/dashboard/", nil)
	if !isSecureRequest(secureTLSReq) {
		t.Fatal("expected TLS request to be secure")
	}

	proxySecureReq := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/", nil)
	proxySecureReq.Header.Set("X-Forwarded-Proto", "https")
	if !isSecureRequest(proxySecureReq) {
		t.Fatal("expected X-Forwarded-Proto=https request to be secure")
	}

	proxyListReq := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/", nil)
	proxyListReq.Header.Set("X-Forwarded-Proto", "https,http")
	if !isSecureRequest(proxyListReq) {
		t.Fatal("expected first X-Forwarded-Proto=https to be secure")
	}

	insecureReq := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/", nil)
	if isSecureRequest(insecureReq) {
		t.Fatal("expected plain HTTP request to be insecure")
	}
}
