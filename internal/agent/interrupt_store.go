//go:build cgo

package agent

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type InterruptStore struct {
	db *sql.DB
}

func NewInterruptStore(db *sql.DB) (*InterruptStore, error) {
	if db == nil {
		return nil, fmt.Errorf("interrupt store: nil db")
	}

	s := &InterruptStore{db: db}
	if err := s.createSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *InterruptStore) createSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS pending_approvals (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		tool_name TEXT NOT NULL,
		tool_args TEXT NOT NULL,
		reason TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_pending_approvals_expires_at ON pending_approvals(expires_at);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("interrupt store: create schema: %w", err)
	}

	return nil
}

func (s *InterruptStore) Save(pa *PendingApproval) error {
	if pa == nil {
		return fmt.Errorf("interrupt store: nil pending approval")
	}

	const query = `
	INSERT INTO pending_approvals (id, user_id, tool_name, tool_args, reason, created_at, expires_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(user_id) DO UPDATE SET
		id = excluded.id,
		tool_name = excluded.tool_name,
		tool_args = excluded.tool_args,
		reason = excluded.reason,
		created_at = excluded.created_at,
		expires_at = excluded.expires_at
	`

	if _, err := s.db.Exec(query,
		pa.ID,
		pa.UserID,
		pa.ToolName,
		pa.ToolArgs,
		pa.Reason,
		pa.CreatedAt.UTC(),
		pa.ExpiresAt.UTC(),
	); err != nil {
		return fmt.Errorf("interrupt store: save: %w", err)
	}

	return nil
}

func (s *InterruptStore) Load(id string) (*PendingApproval, error) {
	const query = `
	SELECT id, user_id, tool_name, tool_args, reason, created_at, expires_at
	FROM pending_approvals
	WHERE id = ?
	LIMIT 1
	`

	pa := &PendingApproval{}
	err := s.db.QueryRow(query, id).Scan(
		&pa.ID,
		&pa.UserID,
		&pa.ToolName,
		&pa.ToolArgs,
		&pa.Reason,
		&pa.CreatedAt,
		&pa.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("interrupt store: load: %w", err)
	}

	return pa, nil
}

func (s *InterruptStore) DeleteByUserID(userID string) error {
	const query = `DELETE FROM pending_approvals WHERE user_id = ?`

	if _, err := s.db.Exec(query, userID); err != nil {
		return fmt.Errorf("interrupt store: delete by user id: %w", err)
	}

	return nil
}

func (s *InterruptStore) CleanExpired() error {
	const query = `DELETE FROM pending_approvals WHERE expires_at <= ?`

	if _, err := s.db.Exec(query, time.Now().UTC()); err != nil {
		return fmt.Errorf("interrupt store: clean expired: %w", err)
	}

	return nil
}
