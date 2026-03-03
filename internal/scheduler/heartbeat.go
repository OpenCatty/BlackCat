package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const defaultHeartbeatCapacity = 100

type HeartbeatResult struct {
	Timestamp      time.Time
	Subsystems     []SubsystemStatus
	OverallHealthy bool
}

type SubsystemStatus struct {
	Name    string
	Healthy bool
	Details string
}

type SubsystemHealthInfo struct {
	Name    string
	Healthy bool
	Details string
}

type SubsystemChecker interface {
	ListHealthy() []SubsystemHealthInfo
}

type HeartbeatStore struct {
	mu       sync.RWMutex
	capacity int
	results  []HeartbeatResult
	next     int
	count    int
}

func NewHeartbeatStore(capacity int) *HeartbeatStore {
	if capacity <= 0 {
		capacity = defaultHeartbeatCapacity
	}

	return &HeartbeatStore{
		capacity: capacity,
		results:  make([]HeartbeatResult, capacity),
	}
}

func (s *HeartbeatStore) Add(result HeartbeatResult) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.results[s.next] = copyHeartbeatResult(result)
	s.next = (s.next + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
	s.mu.Unlock()
}

func (s *HeartbeatStore) Latest(n int) []HeartbeatResult {
	if s == nil || n <= 0 {
		return []HeartbeatResult{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return []HeartbeatResult{}
	}

	if n > s.count {
		n = s.count
	}

	out := make([]HeartbeatResult, 0, n)
	for i := 0; i < n; i++ {
		idx := s.next - 1 - i
		if idx < 0 {
			idx += s.capacity
		}
		out = append(out, copyHeartbeatResult(s.results[idx]))
	}

	return out
}

type HeartbeatTask struct {
	checker           SubsystemChecker
	store             *HeartbeatStore
	reconnector       ChannelReconnector
	reconnectAttempts map[string]int
	nextRetry         map[string]time.Time
}

func NewHeartbeatTask(checker SubsystemChecker, store *HeartbeatStore) *HeartbeatTask {
	if store == nil {
		store = NewHeartbeatStore(defaultHeartbeatCapacity)
	}

	return &HeartbeatTask{
		checker:           checker,
		store:             store,
		reconnectAttempts: make(map[string]int),
		nextRetry:         make(map[string]time.Time),
	}
}

// SetReconnector sets the channel reconnector for auto-recovery.
func (t *HeartbeatTask) SetReconnector(r ChannelReconnector) {
	t.reconnector = r
}

// ChannelReconnector provides access to channels for reconnection.
type ChannelReconnector interface {
	UnhealthyReconnectable() []ReconnectableChannel
}

// ReconnectableChannel pairs a channel name with its reconnect function.
type ReconnectableChannel struct {
	Name      string
	Reconnect func(ctx context.Context) error
}

const maxReconnectAttempts = 3

func (t *HeartbeatTask) Run(ctx context.Context) error {
	if t == nil || t.checker == nil || t.store == nil {
		return nil
	}

	health := t.checker.ListHealthy()
	statuses := make([]SubsystemStatus, 0, len(health))
	overallHealthy := true
	unhealthyNames := make(map[string]bool)

	for _, subsystem := range health {
		status := SubsystemStatus{
			Name:    subsystem.Name,
			Healthy: subsystem.Healthy,
			Details: subsystem.Details,
		}
		statuses = append(statuses, status)

		if !subsystem.Healthy {
			overallHealthy = false
			unhealthyNames[subsystem.Name] = true
			slog.Error("heartbeat: unhealthy subsystem", "name", subsystem.Name, "details", subsystem.Details)
		}
	}

	t.store.Add(HeartbeatResult{
		Timestamp:      time.Now(),
		Subsystems:     statuses,
		OverallHealthy: overallHealthy,
	})

	// Reset counters for channels that recovered
	for name := range t.reconnectAttempts {
		if !unhealthyNames[name] {
			delete(t.reconnectAttempts, name)
			delete(t.nextRetry, name)
		}
	}

	// Attempt reconnection for unhealthy Reconnectable channels
	if t.reconnector != nil {
		now := time.Now()
		for _, rc := range t.reconnector.UnhealthyReconnectable() {
			attempts := t.reconnectAttempts[rc.Name]
			if attempts >= maxReconnectAttempts {
				continue // max attempts reached
			}
			if retry, ok := t.nextRetry[rc.Name]; ok && now.Before(retry) {
				continue // backoff not elapsed
			}

			slog.Info("heartbeat: attempting reconnect", "channel", rc.Name, "attempt", attempts+1)
			if err := rc.Reconnect(ctx); err != nil {
				t.reconnectAttempts[rc.Name] = attempts + 1
				// Exponential backoff: 30s, 60s, 120s
				backoff := 30 * time.Second * (1 << attempts)
				t.nextRetry[rc.Name] = now.Add(backoff)
				slog.Error("heartbeat: reconnect failed", "channel", rc.Name, "attempt", attempts+1, "next_retry", backoff, "err", err)
			} else {
				delete(t.reconnectAttempts, rc.Name)
				delete(t.nextRetry, rc.Name)
				slog.Info("heartbeat: reconnect succeeded", "channel", rc.Name)
			}
		}
	}

	return nil
}

func BuildHeartbeatTaskDef(task *HeartbeatTask) TaskDef {
	return TaskDef{
		Name:     "heartbeat",
		Schedule: "@every 30s",
		Handler:  task.Run,
		Enabled:  true,
	}
}

func copyHeartbeatResult(result HeartbeatResult) HeartbeatResult {
	copyResult := HeartbeatResult{
		Timestamp:      result.Timestamp,
		OverallHealthy: result.OverallHealthy,
	}

	if len(result.Subsystems) > 0 {
		copyResult.Subsystems = make([]SubsystemStatus, len(result.Subsystems))
		copy(copyResult.Subsystems, result.Subsystems)
	}

	return copyResult
}
