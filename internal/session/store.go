package session

// Store defines the interface for session persistence.
type Store interface {
	// Get retrieves a session by key. Returns (nil, nil) if not found.
	Get(key SessionKey) (*Session, error)

	// Save persists a session. Creates or overwrites as needed.
	Save(session *Session) error

	// List returns all stored session keys.
	List() ([]SessionKey, error)

	// Delete removes a session by key.
	Delete(key SessionKey) error

	// CleanExpired removes all expired sessions.
	// Returns the number of sessions deleted and any error.
	CleanExpired() (int, error)
}
