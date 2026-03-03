package scheduler

import (
	"context"
	"testing"
	"time"
)

type mockSubsystemChecker struct {
	subsystems []SubsystemHealthInfo
}

func (m mockSubsystemChecker) ListHealthy() []SubsystemHealthInfo {
	out := make([]SubsystemHealthInfo, len(m.subsystems))
	copy(out, m.subsystems)
	return out
}

func TestHeartbeatRun(t *testing.T) {
	store := NewHeartbeatStore(100)
	task := NewHeartbeatTask(mockSubsystemChecker{subsystems: []SubsystemHealthInfo{
		{Name: "scheduler", Healthy: true, Details: "ok"},
		{Name: "health", Healthy: true, Details: "ok"},
	}}, store)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	latest := store.Latest(1)
	if len(latest) != 1 {
		t.Fatalf("Latest(1) len = %d, want 1", len(latest))
	}

	result := latest[0]
	if !result.OverallHealthy {
		t.Fatalf("OverallHealthy = false, want true")
	}
	if result.Timestamp.IsZero() {
		t.Fatal("Timestamp is zero, want set")
	}
	if len(result.Subsystems) != 2 {
		t.Fatalf("len(Subsystems) = %d, want 2", len(result.Subsystems))
	}
}

func TestHeartbeatUnhealthy(t *testing.T) {
	store := NewHeartbeatStore(100)
	task := NewHeartbeatTask(mockSubsystemChecker{subsystems: []SubsystemHealthInfo{
		{Name: "scheduler", Healthy: true, Details: "ok"},
		{Name: "discord", Healthy: false, Details: "connection timeout"},
	}}, store)

	if err := task.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	latest := store.Latest(1)
	if len(latest) != 1 {
		t.Fatalf("Latest(1) len = %d, want 1", len(latest))
	}

	if latest[0].OverallHealthy {
		t.Fatal("OverallHealthy = true, want false")
	}
}

func TestHeartbeatRingBuffer(t *testing.T) {
	store := NewHeartbeatStore(100)
	base := time.Unix(1700000000, 0)

	for i := 0; i < 150; i++ {
		store.Add(HeartbeatResult{
			Timestamp:      base.Add(time.Duration(i) * time.Second),
			OverallHealthy: true,
		})
	}

	latest := store.Latest(200)
	if len(latest) != 100 {
		t.Fatalf("Latest(200) len = %d, want 100", len(latest))
	}
}

func TestHeartbeatLatestOrder(t *testing.T) {
	store := NewHeartbeatStore(10)
	base := time.Unix(1700000000, 0)

	store.Add(HeartbeatResult{Timestamp: base.Add(1 * time.Second)})
	store.Add(HeartbeatResult{Timestamp: base.Add(2 * time.Second)})
	store.Add(HeartbeatResult{Timestamp: base.Add(3 * time.Second)})

	latest := store.Latest(3)
	if len(latest) != 3 {
		t.Fatalf("Latest(3) len = %d, want 3", len(latest))
	}

	if !latest[0].Timestamp.Equal(base.Add(3 * time.Second)) {
		t.Fatalf("latest[0].Timestamp = %v, want %v", latest[0].Timestamp, base.Add(3*time.Second))
	}
	if !latest[1].Timestamp.Equal(base.Add(2 * time.Second)) {
		t.Fatalf("latest[1].Timestamp = %v, want %v", latest[1].Timestamp, base.Add(2*time.Second))
	}
	if !latest[2].Timestamp.Equal(base.Add(1 * time.Second)) {
		t.Fatalf("latest[2].Timestamp = %v, want %v", latest[2].Timestamp, base.Add(1*time.Second))
	}
}
