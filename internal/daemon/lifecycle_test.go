package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockLifecycleSubsystem struct {
	name       string
	startErr   error
	stopErr    error
	startOrder *[]string
	stopOrder  *[]string
	startCalls int
	stopCalls  int
}

func (m *mockLifecycleSubsystem) Name() string {
	return m.name
}

func (m *mockLifecycleSubsystem) Start(context.Context) error {
	m.startCalls++
	if m.startOrder != nil {
		*m.startOrder = append(*m.startOrder, m.name)
	}
	return m.startErr
}

func (m *mockLifecycleSubsystem) Stop(context.Context) error {
	m.stopCalls++
	if m.stopOrder != nil {
		*m.stopOrder = append(*m.stopOrder, m.name)
	}
	return m.stopErr
}

func (m *mockLifecycleSubsystem) Health() SubsystemHealth {
	return SubsystemHealth{Name: m.name, Status: "ok"}
}

func TestLifecycleStartOrder(t *testing.T) {
	ctx := context.Background()
	startOrder := []string{}

	lm := NewLifecycleManager(
		&mockLifecycleSubsystem{name: "s0", startOrder: &startOrder},
		&mockLifecycleSubsystem{name: "s1", startOrder: &startOrder},
		&mockLifecycleSubsystem{name: "s2", startOrder: &startOrder},
	)

	if err := lm.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	want := []string{"s0", "s1", "s2"}
	if len(startOrder) != len(want) {
		t.Fatalf("start order length = %d, want %d", len(startOrder), len(want))
	}
	for i := range want {
		if startOrder[i] != want[i] {
			t.Fatalf("start order[%d] = %q, want %q", i, startOrder[i], want[i])
		}
	}
}

func TestLifecycleStopReverse(t *testing.T) {
	ctx := context.Background()
	stopOrder := []string{}

	lm := NewLifecycleManager(
		&mockLifecycleSubsystem{name: "s0", stopOrder: &stopOrder},
		&mockLifecycleSubsystem{name: "s1", stopOrder: &stopOrder},
		&mockLifecycleSubsystem{name: "s2", stopOrder: &stopOrder},
	)

	if err := lm.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}
	if err := lm.StopAll(ctx); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}

	want := []string{"s2", "s1", "s0"}
	if len(stopOrder) != len(want) {
		t.Fatalf("stop order length = %d, want %d", len(stopOrder), len(want))
	}
	for i := range want {
		if stopOrder[i] != want[i] {
			t.Fatalf("stop order[%d] = %q, want %q", i, stopOrder[i], want[i])
		}
	}
}

func TestLifecycleStartFailure(t *testing.T) {
	ctx := context.Background()
	startErr := errors.New("boom")

	subsystems := []*mockLifecycleSubsystem{
		{name: "s0"},
		{name: "s1"},
		{name: "s2", startErr: startErr},
		{name: "s3"},
		{name: "s4"},
	}

	lm := NewLifecycleManager(
		subsystems[0],
		subsystems[1],
		subsystems[2],
		subsystems[3],
		subsystems[4],
	)

	err := lm.StartAll(ctx)
	if err == nil {
		t.Fatal("StartAll() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), `subsystem "s2" failed to start`) {
		t.Fatalf("StartAll() error = %q, missing subsystem context", err.Error())
	}
	if !strings.Contains(err.Error(), startErr.Error()) {
		t.Fatalf("StartAll() error = %q, missing wrapped start error", err.Error())
	}

	if subsystems[0].startCalls != 1 || subsystems[1].startCalls != 1 || subsystems[2].startCalls != 1 {
		t.Fatalf("expected subsystems 0-2 to be started once, got calls: [%d %d %d]",
			subsystems[0].startCalls,
			subsystems[1].startCalls,
			subsystems[2].startCalls,
		)
	}
	if subsystems[3].startCalls != 0 || subsystems[4].startCalls != 0 {
		t.Fatalf("expected subsystems 3-4 not started, got calls: [%d %d]",
			subsystems[3].startCalls,
			subsystems[4].startCalls,
		)
	}
}

func TestLifecycleStopCollectsErrors(t *testing.T) {
	ctx := context.Background()
	stopOrder := []string{}
	stopErrA := errors.New("stop-a")
	stopErrB := errors.New("stop-b")

	subsystems := []*mockLifecycleSubsystem{
		{name: "s0", stopErr: stopErrA, stopOrder: &stopOrder},
		{name: "s1", stopOrder: &stopOrder},
		{name: "s2", stopErr: stopErrB, stopOrder: &stopOrder},
	}

	lm := NewLifecycleManager(subsystems[0], subsystems[1], subsystems[2])

	if err := lm.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	err := lm.StopAll(ctx)
	if err == nil {
		t.Fatal("StopAll() error = nil, want non-nil")
	}

	wantStopOrder := []string{"s2", "s1", "s0"}
	if len(stopOrder) != len(wantStopOrder) {
		t.Fatalf("stop order length = %d, want %d", len(stopOrder), len(wantStopOrder))
	}
	for i := range wantStopOrder {
		if stopOrder[i] != wantStopOrder[i] {
			t.Fatalf("stop order[%d] = %q, want %q", i, stopOrder[i], wantStopOrder[i])
		}
	}

	for i, subsystem := range subsystems {
		if subsystem.stopCalls != 1 {
			t.Fatalf("subsystem %d stop calls = %d, want 1", i, subsystem.stopCalls)
		}
	}

	errText := err.Error()
	if !strings.Contains(errText, `subsystem "s0" failed to stop`) || !strings.Contains(errText, stopErrA.Error()) {
		t.Fatalf("StopAll() error = %q, missing first stop failure details", errText)
	}
	if !strings.Contains(errText, `subsystem "s2" failed to stop`) || !strings.Contains(errText, stopErrB.Error()) {
		t.Fatalf("StopAll() error = %q, missing second stop failure details", errText)
	}
}

func TestLifecycleEmpty(t *testing.T) {
	ctx := context.Background()
	lm := NewLifecycleManager()

	if err := lm.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}
	if err := lm.StopAll(ctx); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}
}
