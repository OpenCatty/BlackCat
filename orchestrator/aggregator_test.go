package orchestrator

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAggregateAllSuccess(t *testing.T) {
	results := []Result{
		{Name: "alpha", Output: "output alpha", Duration: 10 * time.Millisecond},
		{Name: "beta", Output: "output beta", Duration: 20 * time.Millisecond},
		{Name: "gamma", Output: "output gamma", Duration: 30 * time.Millisecond},
	}

	report := (Aggregator{}).Summarize(results)

	if report.TotalTasks != 3 {
		t.Fatalf("TotalTasks = %d, want 3", report.TotalTasks)
	}
	if report.Succeeded != 3 {
		t.Fatalf("Succeeded = %d, want 3", report.Succeeded)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
	if report.Duration != 60*time.Millisecond {
		t.Fatalf("Duration = %v, want %v", report.Duration, 60*time.Millisecond)
	}

	for _, output := range []string{"output alpha", "output beta", "output gamma"} {
		if !strings.Contains(report.CombinedOutput, output) {
			t.Fatalf("CombinedOutput missing %q", output)
		}
	}
}

func TestAggregateMixed(t *testing.T) {
	results := []Result{
		{Name: "success-1", Output: "ok one", Duration: 5 * time.Millisecond},
		{Name: "success-2", Output: "ok two", Duration: 5 * time.Millisecond},
		{Name: "success-3", Output: "ok three", Duration: 5 * time.Millisecond},
		{Name: "failed-1", Error: errors.New("boom one"), Duration: 5 * time.Millisecond},
		{Name: "failed-2", Error: errors.New("boom two"), Duration: 5 * time.Millisecond},
	}

	report := (Aggregator{}).Summarize(results)

	if report.TotalTasks != 5 {
		t.Fatalf("TotalTasks = %d, want 5", report.TotalTasks)
	}
	if report.Succeeded != 3 {
		t.Fatalf("Succeeded = %d, want 3", report.Succeeded)
	}
	if report.Failed != 2 {
		t.Fatalf("Failed = %d, want 2", report.Failed)
	}
}

func TestAggregateAllFailure(t *testing.T) {
	results := []Result{
		{Name: "a", Error: errors.New("error a"), Duration: 2 * time.Millisecond},
		{Name: "b", Error: errors.New("error b"), Duration: 2 * time.Millisecond},
		{Name: "c", Error: errors.New("error c"), Duration: 2 * time.Millisecond},
	}

	report := (Aggregator{}).Summarize(results)

	if report.Succeeded != 0 {
		t.Fatalf("Succeeded = %d, want 0", report.Succeeded)
	}
	if report.Failed != len(results) {
		t.Fatalf("Failed = %d, want %d", report.Failed, len(results))
	}

	for _, marker := range []string{
		"[FAILED: error a]",
		"[FAILED: error b]",
		"[FAILED: error c]",
	} {
		if !strings.Contains(report.CombinedOutput, marker) {
			t.Fatalf("CombinedOutput missing failure marker %q", marker)
		}
	}
}

func TestAggregateEmpty(t *testing.T) {
	report := (Aggregator{}).Summarize(nil)

	if report == nil {
		t.Fatal("Summarize() returned nil report")
	}
	if report.TotalTasks != 0 {
		t.Fatalf("TotalTasks = %d, want 0", report.TotalTasks)
	}
	if report.Succeeded != 0 {
		t.Fatalf("Succeeded = %d, want 0", report.Succeeded)
	}
	if report.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", report.Failed)
	}
	if report.Duration != 0 {
		t.Fatalf("Duration = %v, want 0", report.Duration)
	}
	if report.CombinedOutput != "" {
		t.Fatalf("CombinedOutput = %q, want empty string", report.CombinedOutput)
	}
}

func TestCombinedOutput(t *testing.T) {
	results := []Result{
		{Name: "agent-1", Output: "out 1"},
		{Name: "agent-2", Output: "out 2"},
	}

	report := (Aggregator{}).Summarize(results)

	for _, separator := range []string{
		"--- Agent: agent-1 ---",
		"--- Agent: agent-2 ---",
	} {
		if !strings.Contains(report.CombinedOutput, separator) {
			t.Fatalf("CombinedOutput missing separator %q", separator)
		}
	}
}

func TestFailureMarker(t *testing.T) {
	report := (Aggregator{}).Summarize([]Result{
		{Name: "bad-agent", Error: errors.New("error message")},
	})

	if !strings.Contains(report.CombinedOutput, "[FAILED: error message]") {
		t.Fatalf("CombinedOutput = %q, want failure marker", report.CombinedOutput)
	}
}
