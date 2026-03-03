package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/types"
)

func TestSessionKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  SessionKey
		want string
	}{
		{
			name: "normal user",
			key:  SessionKey{ChannelType: "telegram", ChannelID: "chat123", UserID: "user456"},
			want: "telegram:chat123:user456",
		},
		{
			name: "anonymous user (empty UserID)",
			key:  SessionKey{ChannelType: "discord", ChannelID: "guild789"},
			want: "discord:guild789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.key.String()
			if got != tt.want {
				t.Errorf("SessionKey.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileStoreSaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	now := time.Now().Truncate(time.Millisecond)
	sess := &Session{
		Key:       SessionKey{ChannelType: "telegram", ChannelID: "chat1", UserID: "user1"},
		Messages:  []types.LLMMessage{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}},
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  map[string]string{"lang": "en"},
	}

	// Save
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get
	got, err := store.Get(sess.Key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Key != sess.Key {
		t.Errorf("Key = %v, want %v", got.Key, sess.Key)
	}
	if len(got.Messages) != 2 {
		t.Errorf("Messages len = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Content != "hello" {
		t.Errorf("Messages[0].Content = %q, want %q", got.Messages[0].Content, "hello")
	}
	if got.Metadata["lang"] != "en" {
		t.Errorf("Metadata[lang] = %q, want %q", got.Metadata["lang"], "en")
	}
}

func TestFileStoreGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	got, err := store.Get(SessionKey{ChannelType: "telegram", ChannelID: "nonexistent", UserID: "nobody"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("Get returned non-nil for missing session: %v", got)
	}
}

func TestFileStoreMaxHistory(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 3)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	msgs := make([]types.LLMMessage, 5)
	for i := range msgs {
		msgs[i] = types.LLMMessage{Role: "user", Content: "msg" + string(rune('0'+i))}
	}

	sess := &Session{
		Key:       SessionKey{ChannelType: "telegram", ChannelID: "chat1", UserID: "user1"},
		Messages:  msgs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(sess.Key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Messages) != 3 {
		t.Errorf("Messages len = %d, want 3 (maxHistory)", len(got.Messages))
	}
	// Should keep the last 3 messages (index 2, 3, 4)
	if got.Messages[0].Content != "msg2" {
		t.Errorf("Messages[0].Content = %q, want %q", got.Messages[0].Content, "msg2")
	}
}

func TestFileStoreList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	keys := []SessionKey{
		{ChannelType: "telegram", ChannelID: "chat1", UserID: "user1"},
		{ChannelType: "discord", ChannelID: "guild2", UserID: "user2"},
		{ChannelType: "whatsapp", ChannelID: "num3"}, // anonymous
	}

	now := time.Now()
	for _, key := range keys {
		sess := &Session{
			Key:       key,
			Messages:  []types.LLMMessage{{Role: "user", Content: "test"}},
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := store.Save(sess); err != nil {
			t.Fatalf("Save %v: %v", key, err)
		}
	}

	listed, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 3 {
		t.Errorf("List len = %d, want 3", len(listed))
	}

	// Verify all keys are present (order may vary).
	found := make(map[string]bool)
	for _, k := range listed {
		found[k.String()] = true
	}
	for _, k := range keys {
		if !found[k.String()] {
			t.Errorf("List missing key %s", k.String())
		}
	}
}

func TestFileStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := SessionKey{ChannelType: "telegram", ChannelID: "chat1", UserID: "user1"}
	sess := &Session{
		Key:       key,
		Messages:  []types.LLMMessage{{Role: "user", Content: "bye"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "telegram_chat1_user1.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("File should exist after save: %v", err)
	}

	// Delete.
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("File should not exist after delete, got err: %v", err)
	}

	// Get should return nil, nil.
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Errorf("Get after delete returned non-nil: %v", got)
	}
}

func TestFileStoreDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Deleting a nonexistent key should not return an error.
	err = store.Delete(SessionKey{ChannelType: "telegram", ChannelID: "nope", UserID: "nobody"})
	if err != nil {
		t.Errorf("Delete nonexistent should not error, got: %v", err)
	}
}

func TestFileStoreAnonymousSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := SessionKey{ChannelType: "whatsapp", ChannelID: "num123"}
	sess := &Session{
		Key:       key,
		Messages:  []types.LLMMessage{{Role: "user", Content: "anon"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file name for anonymous (no userID part).
	path := filepath.Join(dir, "whatsapp_num123.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Anonymous session file should be %s, stat error: %v", path, err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil for anonymous session")
	}
	if got.Key.UserID != "" {
		t.Errorf("UserID = %q, want empty", got.Key.UserID)
	}
}

func TestNewFileStoreCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "sessions")
	_, err := NewFileStore(dir, 50)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Dir should exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Expected directory, got file")
	}
}

func TestSanitize(t *testing.T) {
	// Keys with underscores should be sanitized to avoid filename parsing issues.
	key := SessionKey{ChannelType: "tele_gram", ChannelID: "chat_1", UserID: "user_1"}
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	sess := &Session{
		Key:       key,
		Messages:  []types.LLMMessage{{Role: "user", Content: "sanitize test"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File should have dashes replacing underscores.
	path := filepath.Join(dir, "tele-gram_chat-1_user-1.json")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Sanitized file should be at %s, stat error: %v", path, err)
	}
}

func TestExpiredSessionsReturnNil(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Create a session with ExpiresAt in the past
	key := SessionKey{ChannelType: "telegram", ChannelID: "chat1", UserID: "user1"}
	sess := &Session{
		Key:       key,
		Messages:  []types.LLMMessage{{Role: "user", Content: "expired test"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired 1 hour ago
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get should return nil for expired session
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("Get returned non-nil for expired session: %v", got)
	}
}

func TestStaleSessionsReturnNil(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	// Create a session JSON file directly with LastActivity 25h ago and no ExpiresAt
	key := SessionKey{ChannelType: "telegram", ChannelID: "chat2", UserID: "user2"}
	now := time.Now()
	sess := &Session{
		Key:          key,
		Messages:     []types.LLMMessage{{Role: "user", Content: "stale test"}},
		CreatedAt:    now.Add(-48 * time.Hour),
		UpdatedAt:    now.Add(-48 * time.Hour),
		LastActivity: now.Add(-25 * time.Hour), // inactive for 25h
		MessageCount: 1,
		// ExpiresAt is left zero to test fallback logic
	}

	// Write directly to JSON file to bypass Save() which would reset metadata
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	path := store.filePath(key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Get should return nil for stale session
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Errorf("Get returned non-nil for stale session: %v", got)
	}
}

func TestSaveUpdatesMetadata(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	key := SessionKey{ChannelType: "telegram", ChannelID: "chat3", UserID: "user3"}
	messages := []types.LLMMessage{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
	}
	sess := &Session{
		Key:       key,
		Messages:  messages,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.Save(sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}

	// Verify MessageCount was updated
	if got.MessageCount != 3 {
		t.Errorf("MessageCount = %d, want 3", got.MessageCount)
	}

	// Verify LastActivity was set to approximately now
	if got.LastActivity.IsZero() {
		t.Errorf("LastActivity should not be zero")
	}
	if time.Since(got.LastActivity) > 5*time.Second {
		t.Errorf("LastActivity should be recent, but diff is %v", time.Since(got.LastActivity))
	}

	// Verify ExpiresAt was set to approximately 24h from now
	if got.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt should not be zero")
	}
	expectedExpiry := time.Now().Add(24 * time.Hour)
	diff := got.ExpiresAt.Sub(expectedExpiry).Abs()
	if diff > 5*time.Second {
		t.Errorf("ExpiresAt diff from expected is %v, want <5s", diff)
	}
}

func TestCleanExpiredRemovesExpiredSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	now := time.Now()


	// Create 3 sessions: 1 expired, 1 stale, 1 active
	expiredKey := SessionKey{ChannelType: "telegram", ChannelID: "chat_exp", UserID: "user1"}
	expiredSess := &Session{
		Key:       expiredKey,
		Messages:  []types.LLMMessage{{Role: "user", Content: "expired"}},
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(-1 * time.Hour), // expired 1h ago
	}

	staleKey := SessionKey{ChannelType: "discord", ChannelID: "guild_stale", UserID: "user2"}
	staleSess := &Session{
		Key:          staleKey,
		Messages:     []types.LLMMessage{{Role: "user", Content: "stale"}},
		CreatedAt:    now.Add(-48 * time.Hour),
		UpdatedAt:    now.Add(-48 * time.Hour),
		LastActivity: now.Add(-25 * time.Hour), // inactive 25h
		MessageCount: 1,
	}

	activeKey := SessionKey{ChannelType: "whatsapp", ChannelID: "num_active", UserID: "user3"}
	activeSess := &Session{
		Key:       activeKey,
		Messages:  []types.LLMMessage{{Role: "user", Content: "active"}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save expired and active normally
	if err := store.Save(expiredSess); err != nil {
		t.Fatalf("Save expired: %v", err)
	}
	if err := store.Save(activeSess); err != nil {
		t.Fatalf("Save active: %v", err)
	}

	// Write stale session directly to bypass Save() resetting metadata
	data, err := json.MarshalIndent(staleSess, "", "  ")
	if err != nil {
		t.Fatalf("Marshal stale: %v", err)
	}
	path := store.filePath(staleKey)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile stale: %v", err)
	}

	// Run CleanExpired
	count, err := store.CleanExpired()
	if err != nil {
		t.Fatalf("CleanExpired: %v", err)
	}

	// Should have deleted 2 sessions (expired + stale)
	if count != 2 {
		t.Errorf("CleanExpired count = %d, want 2", count)
	}

	// Verify expired session is gone
	got, err := store.Get(expiredKey)
	if err != nil {
		t.Fatalf("Get expired: %v", err)
	}
	if got != nil {
		t.Errorf("Expired session should be gone but still exists")
	}

	// Verify stale session is gone
	got, err = store.Get(staleKey)
	if err != nil {
		t.Fatalf("Get stale: %v", err)
	}
	if got != nil {
		t.Errorf("Stale session should be gone but still exists")
	}

	// Verify active session still exists
	got, err = store.Get(activeKey)
	if err != nil {
		t.Fatalf("Get active: %v", err)
	}
	if got == nil {
		t.Errorf("Active session should exist but is gone")
	}
}

func TestCleanExpiredCountsCorrectly(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir, 100)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	now := time.Now()

	// Create 5 expired sessions
	for i := 0; i < 5; i++ {
		key := SessionKey{ChannelType: "telegram", ChannelID: "chat" + string(rune('0'+i)), UserID: "user" + string(rune('0'+i))}
		sess := &Session{
			Key:       key,
			Messages:  []types.LLMMessage{{Role: "user", Content: "test"}},
			CreatedAt: now,
			UpdatedAt: now,
			ExpiresAt: now.Add(-time.Hour),
		}
		if err := store.Save(sess); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	count, err := store.CleanExpired()
	if err != nil {
		t.Fatalf("CleanExpired: %v", err)
	}

	if count != 5 {
		t.Errorf("CleanExpired count = %d, want 5", count)
	}
}

