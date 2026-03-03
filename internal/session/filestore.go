package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileStore implements Store using JSON files on disk.
// Each session is stored as a separate JSON file. Thread-safe via sync.RWMutex.
type FileStore struct {
	dir        string
	maxHistory int
	mu         sync.RWMutex
}

// NewFileStore creates a new FileStore that persists sessions as JSON files
// in the given directory. maxHistory limits the number of messages kept per session.
// The directory is created if it does not exist.
func NewFileStore(dir string, maxHistory int) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("session: create dir %s: %w", dir, err)
	}
	return &FileStore{
		dir:        dir,
		maxHistory: maxHistory,
	}, nil
}

// Get retrieves a session by key. Returns (nil, nil) if the file does not exist.
func (fs *FileStore) Get(key SessionKey) (*Session, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.filePath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("session: read %s: %w", path, err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("session: unmarshal %s: %w", path, err)
	}

	// Check if session is expired
	if !sess.ExpiresAt.IsZero() && time.Now().After(sess.ExpiresAt) {
		return nil, nil // expired
	}

	// Fallback: if ExpiresAt is zero but LastActivity is set and older than 24h, treat as stale
	if sess.ExpiresAt.IsZero() && !sess.LastActivity.IsZero() && time.Now().Sub(sess.LastActivity) > 24*time.Hour {
		return nil, nil // stale
	}

	return &sess, nil
}

// Save persists a session to disk as a JSON file.
// Messages are trimmed to the last maxHistory entries before writing.
func (fs *FileStore) Save(session *Session) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Update metadata
	session.LastActivity = time.Now()
	session.MessageCount = len(session.Messages)
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(24 * time.Hour)
	}

	// Trim messages to maxHistory.
	if fs.maxHistory > 0 && len(session.Messages) > fs.maxHistory {
		session.Messages = session.Messages[len(session.Messages)-fs.maxHistory:]
		session.MessageCount = fs.maxHistory // recalculate after trim
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}

	path := fs.filePath(session.Key)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", path, err)
	}
	return nil
}

// List returns all session keys found in the directory by parsing filenames.
func (fs *FileStore) List() ([]SessionKey, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("session: list dir %s: %w", fs.dir, err)
	}

	var keys []SessionKey
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		key, ok := parseFileName(e.Name())
		if ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// CleanExpired removes all expired sessions. Returns the count of deleted sessions.
func (fs *FileStore) CleanExpired() (int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return 0, fmt.Errorf("session: list dir %s: %w", fs.dir, err)
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		key, ok := parseFileName(e.Name())
		if !ok {
			continue
		}

		path := fs.filePath(key)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}

		// Check if expired
		isExpired := false
		if !sess.ExpiresAt.IsZero() && time.Now().After(sess.ExpiresAt) {
			isExpired = true
		} else if sess.ExpiresAt.IsZero() && !sess.LastActivity.IsZero() && time.Now().Sub(sess.LastActivity) > 24*time.Hour {
			isExpired = true
		}

		if isExpired {
			if err := os.Remove(path); err == nil {
				count++
			}
		}
	}

	return count, nil
}


// Delete removes the session file for the given key.
func (fs *FileStore) Delete(key SessionKey) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.filePath(key)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("session: delete %s: %w", path, err)
	}
	return nil
}

// filePath returns the JSON file path for a session key.
// Format: {channelType}_{channelID}_{userID}.json
// Anonymous (empty UserID): {channelType}_{channelID}.json
func (fs *FileStore) filePath(key SessionKey) string {
	name := sanitize(key.ChannelType) + "_" + sanitize(key.ChannelID)
	if key.UserID != "" {
		name += "_" + sanitize(key.UserID)
	}
	return filepath.Join(fs.dir, name+".json")
}

// parseFileName extracts a SessionKey from a filename like "type_id_user.json".
func parseFileName(name string) (SessionKey, bool) {
	name = strings.TrimSuffix(name, ".json")
	parts := strings.SplitN(name, "_", 3)
	switch len(parts) {
	case 2:
		return SessionKey{ChannelType: parts[0], ChannelID: parts[1]}, true
	case 3:
		return SessionKey{ChannelType: parts[0], ChannelID: parts[1], UserID: parts[2]}, true
	default:
		return SessionKey{}, false
	}
}

// sanitize replaces path separators and underscores in key components to avoid
// filename collisions. Underscores in values are replaced with dashes.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, string(os.PathSeparator), "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}
