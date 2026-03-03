package types

import "errors"

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrToolNotFound       = errors.New("tool not found")
	ErrChannelClosed      = errors.New("channel closed")
	ErrDenyListViolation  = errors.New("command blocked by deny list")
	ErrVaultLocked        = errors.New("vault is locked")
	ErrPathTraversal      = errors.New("path traversal detected")
	ErrMaxTurnsExceeded   = errors.New("maximum turns exceeded")
	ErrCompactionRequired = errors.New("context compaction required")
)
