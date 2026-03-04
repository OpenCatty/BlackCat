package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultApprovalTimeout = 5 * time.Minute

type PendingApproval struct {
	ID        string
	UserID    string
	ToolName  string
	ToolArgs  string
	Reason    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type approvalStore interface {
	Save(pa *PendingApproval) error
	Load(id string) (*PendingApproval, error)
	DeleteByUserID(userID string) error
	CleanExpired() error
}

type InterruptManager struct {
	pending sync.Map
	store   approvalStore
}

func NewInterruptManager() *InterruptManager {
	return &InterruptManager{}
}

func NewInterruptManagerWithStore(store approvalStore) *InterruptManager {
	return &InterruptManager{store: store}
}

func (m *InterruptManager) CreateApproval(userID, toolName, toolArgs, reason string, timeout time.Duration) *PendingApproval {
	if timeout <= 0 {
		timeout = defaultApprovalTimeout
	}

	now := time.Now()
	pa := &PendingApproval{
		ID:        fmt.Sprintf("%s-%d", userID, now.UnixNano()),
		UserID:    userID,
		ToolName:  toolName,
		ToolArgs:  toolArgs,
		Reason:    reason,
		CreatedAt: now,
		ExpiresAt: now.Add(timeout),
	}

	m.pending.Store(userID, pa)
	if m.store != nil {
		_ = m.store.Save(pa)
	}

	return pa
}

func (m *InterruptManager) GetPending(userID string) *PendingApproval {
	v, ok := m.pending.Load(userID)
	if !ok {
		return nil
	}

	pa, ok := v.(*PendingApproval)
	if !ok || pa == nil {
		m.pending.Delete(userID)
		if m.store != nil {
			_ = m.store.DeleteByUserID(userID)
		}
		return nil
	}

	if time.Now().After(pa.ExpiresAt) {
		m.pending.Delete(userID)
		if m.store != nil {
			_ = m.store.DeleteByUserID(userID)
		}
		return nil
	}

	return pa
}

func (m *InterruptManager) HandleReply(userID, reply string) (approved bool, found bool) {
	pa := m.GetPending(userID)
	if pa == nil {
		return false, false
	}

	normalized := strings.ToLower(strings.TrimSpace(reply))

	switch normalized {
	case "ya", "yes", "approve", "ok", "oke":
		m.pending.Delete(userID)
		if m.store != nil {
			_ = m.store.DeleteByUserID(userID)
		}
		return true, true
	case "tidak", "no", "reject", "cancel", "batal":
		m.pending.Delete(userID)
		if m.store != nil {
			_ = m.store.DeleteByUserID(userID)
		}
		return false, true
	default:
		return false, false
	}
}

func (m *InterruptManager) CleanExpired() {
	now := time.Now()
	m.pending.Range(func(key, value any) bool {
		userID, ok := key.(string)
		if !ok {
			m.pending.Delete(key)
			return true
		}

		pa, ok := value.(*PendingApproval)
		if !ok || pa == nil || now.After(pa.ExpiresAt) {
			m.pending.Delete(userID)
			if m.store != nil {
				_ = m.store.DeleteByUserID(userID)
			}
		}

		return true
	})

	if m.store != nil {
		_ = m.store.CleanExpired()
	}
}
