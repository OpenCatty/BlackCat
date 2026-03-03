package dashboard

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

const sessionCookieName = "blackcat_session"

func generateSessionSecret() ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func signSession(token string, secret []byte) string {
	tokenHash := sha256.Sum256([]byte(token))

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(tokenHash[:])
	signature := mac.Sum(nil)

	encodedHash := base64.StdEncoding.EncodeToString(tokenHash[:])
	encodedSignature := base64.StdEncoding.EncodeToString(signature)

	return encodedHash + ":" + encodedSignature
}

func validateSession(cookieValue string, token string, secret []byte) bool {
	if cookieValue == "" || token == "" || len(secret) == 0 {
		return false
	}

	parts := strings.Split(cookieValue, ":")
	if len(parts) != 2 {
		return false
	}

	decodedHash, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	decodedSignature, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	expectedHash := sha256.Sum256([]byte(token))
	if !hmac.Equal(decodedHash, expectedHash[:]) {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(expectedHash[:])
	expectedSignature := mac.Sum(nil)

	return hmac.Equal(decodedSignature, expectedSignature)
}

func setSessionCookie(w http.ResponseWriter, token string, secret []byte, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signSession(token, secret),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
		Secure:   secure,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func isSecureRequest(r *http.Request) bool {
	if r == nil {
		return false
	}

	if r.TLS != nil {
		return true
	}

	forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if forwardedProto == "" {
		return false
	}

	firstProto := strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
	return strings.EqualFold(firstProto, "https")
}
