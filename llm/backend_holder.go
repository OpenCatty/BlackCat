package llm

// BackendHolder wraps an llm.Backend for safe storage in sync/atomic.Value.
// atomic.Value panics if you try to swap between different concrete types,
// so we always store *BackendHolder (pointer to struct), never the bare Backend interface.
type BackendHolder struct {
	Backend Backend
}
