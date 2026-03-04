package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// adaptiveMockCoreStore is a mock CoreStoreIface for adaptive tests.
// Uses a unique name to avoid collision with any other mock types.
type adaptiveMockCoreStore struct {
	data     map[string]string // key = "userID:key"
	setErr   error
	getErr   error
	setCalls []adaptiveSetCall
}

type adaptiveSetCall struct {
	UserID string
	Key    string
	Value  string
}

func newAdaptiveMockCoreStore() *adaptiveMockCoreStore {
	return &adaptiveMockCoreStore{data: make(map[string]string)}
}

func (m *adaptiveMockCoreStore) Get(_ context.Context, userID, key string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.data[userID+":"+key], nil
}

func (m *adaptiveMockCoreStore) Set(_ context.Context, userID, key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[userID+":"+key] = value
	m.setCalls = append(m.setCalls, adaptiveSetCall{UserID: userID, Key: key, Value: value})
	return nil
}

// --- Tests ---

func TestLoadPreferences_Defaults(t *testing.T) {
	// nil store should return all "auto" defaults.
	pm := NewPreferenceManager(nil)
	profile := pm.LoadPreferences(context.Background(), "user1")

	if profile.Language != "auto" {
		t.Errorf("Language = %q, want %q", profile.Language, "auto")
	}
	if profile.Style != "auto" {
		t.Errorf("Style = %q, want %q", profile.Style, "auto")
	}
	if profile.Verbosity != "auto" {
		t.Errorf("Verbosity = %q, want %q", profile.Verbosity, "auto")
	}
	if profile.TechnicalDepth != "auto" {
		t.Errorf("TechnicalDepth = %q, want %q", profile.TechnicalDepth, "auto")
	}
}

func TestLoadPreferences_NilManager(t *testing.T) {
	// A nil *PreferenceManager should not panic.
	var pm *PreferenceManager
	profile := pm.LoadPreferences(context.Background(), "user1")

	if profile.Language != "auto" {
		t.Errorf("Language = %q, want %q", profile.Language, "auto")
	}
}

func TestLoadPreferences_WithValues(t *testing.T) {
	mock := newAdaptiveMockCoreStore()
	mock.data["u1:pref:language"] = "Spanish"
	mock.data["u1:pref:style"] = "formal"
	mock.data["u1:pref:technical_depth"] = "expert"

	pm := NewPreferenceManager(mock)
	profile := pm.LoadPreferences(context.Background(), "u1")

	if profile.Language != "Spanish" {
		t.Errorf("Language = %q, want %q", profile.Language, "Spanish")
	}
	if profile.Style != "formal" {
		t.Errorf("Style = %q, want %q", profile.Style, "formal")
	}
	if profile.Verbosity != "auto" {
		t.Errorf("Verbosity = %q, want %q (default)", profile.Verbosity, "auto")
	}
	if profile.TechnicalDepth != "expert" {
		t.Errorf("TechnicalDepth = %q, want %q", profile.TechnicalDepth, "expert")
	}
}

func TestUpdatePreference_ValidKey(t *testing.T) {
	mock := newAdaptiveMockCoreStore()
	pm := NewPreferenceManager(mock)

	err := pm.UpdatePreference(context.Background(), "u1", "pref:language", "French")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.setCalls) != 1 {
		t.Fatalf("expected 1 Set call, got %d", len(mock.setCalls))
	}
	if mock.setCalls[0].Value != "French" {
		t.Errorf("Set value = %q, want %q", mock.setCalls[0].Value, "French")
	}
}

func TestUpdatePreference_InvalidKey(t *testing.T) {
	mock := newAdaptiveMockCoreStore()
	pm := NewPreferenceManager(mock)

	err := pm.UpdatePreference(context.Background(), "u1", "pref:nonexistent", "value")
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown preference key") {
		t.Errorf("error = %q, want it to mention 'unknown preference key'", err.Error())
	}
}

func TestUpdatePreference_NilManager(t *testing.T) {
	var pm *PreferenceManager
	err := pm.UpdatePreference(context.Background(), "u1", "pref:language", "English")
	if err != nil {
		t.Fatalf("nil manager should return nil error, got: %v", err)
	}
}

func TestUpdatePreference_StoreError(t *testing.T) {
	mock := newAdaptiveMockCoreStore()
	mock.setErr = fmt.Errorf("db write failed")
	pm := NewPreferenceManager(mock)

	err := pm.UpdatePreference(context.Background(), "u1", "pref:style", "casual")
	if err == nil {
		t.Fatal("expected error from store, got nil")
	}
}

func TestFormatForPrompt_AllAuto_ReturnsEmpty(t *testing.T) {
	profile := AdaptiveProfile{
		Language:       "auto",
		Style:          "auto",
		Verbosity:      "auto",
		TechnicalDepth: "auto",
	}
	got := FormatForPrompt(profile)
	if got != "" {
		t.Errorf("expected empty string for all-auto profile, got %q", got)
	}
}

func TestFormatForPrompt_AllEmpty_ReturnsEmpty(t *testing.T) {
	profile := AdaptiveProfile{}
	got := FormatForPrompt(profile)
	if got != "" {
		t.Errorf("expected empty string for zero-value profile, got %q", got)
	}
}

func TestFormatForPrompt_WithValues(t *testing.T) {
	profile := AdaptiveProfile{
		Language:       "Spanish",
		Style:          "auto",
		Verbosity:      "detailed",
		TechnicalDepth: "expert",
	}
	got := FormatForPrompt(profile)

	if !strings.Contains(got, "### User Preferences") {
		t.Errorf("missing header, got:\n%s", got)
	}
	if !strings.Contains(got, "- Language: Spanish") {
		t.Errorf("missing Language line, got:\n%s", got)
	}
	if strings.Contains(got, "Style") {
		t.Errorf("auto Style should be omitted, got:\n%s", got)
	}
	if !strings.Contains(got, "- Verbosity: detailed") {
		t.Errorf("missing Verbosity line, got:\n%s", got)
	}
	if !strings.Contains(got, "- Technical depth: expert") {
		t.Errorf("missing Technical depth line, got:\n%s", got)
	}
}

func TestFormatForPrompt_CaseInsensitiveAuto(t *testing.T) {
	profile := AdaptiveProfile{
		Language:       "AUTO",
		Style:          "Auto",
		Verbosity:      "aUtO",
		TechnicalDepth: "basic",
	}
	got := FormatForPrompt(profile)

	if strings.Contains(got, "Language") {
		t.Errorf("AUTO Language should be omitted, got:\n%s", got)
	}
	if !strings.Contains(got, "- Technical depth: basic") {
		t.Errorf("missing Technical depth line, got:\n%s", got)
	}
}
