//go:build cgo

package observability

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCostTracker_Record(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}

	ctx := context.Background()
	if err := ct.Record(ctx, "user1", "sess1", "gpt-4o", "openai", 100, 50); err != nil {
		t.Fatalf("Record: %v", err)
	}

	summary, err := ct.UserSummary(ctx, "user1")
	if err != nil {
		t.Fatalf("UserSummary: %v", err)
	}
	if len(summary) != 1 {
		t.Fatalf("expected 1 model summary, got %d", len(summary))
	}
	if summary[0].Model != "gpt-4o" {
		t.Errorf("model = %q, want %q", summary[0].Model, "gpt-4o")
	}
	if summary[0].TotalInputTokens != 100 {
		t.Errorf("input tokens = %d, want 100", summary[0].TotalInputTokens)
	}
	if summary[0].TotalOutputTokens != 50 {
		t.Errorf("output tokens = %d, want 50", summary[0].TotalOutputTokens)
	}
	if summary[0].CallCount != 1 {
		t.Errorf("call count = %d, want 1", summary[0].CallCount)
	}
}

func TestCostTracker_Aggregate(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := ct.Record(ctx, "user1", "sess1", "gpt-4o", "openai", 1000, 500); err != nil {
			t.Fatalf("Record %d: %v", i, err)
		}
	}

	summary, err := ct.UserSummary(ctx, "user1")
	if err != nil {
		t.Fatalf("UserSummary: %v", err)
	}
	if len(summary) != 1 {
		t.Fatalf("expected 1 model summary, got %d", len(summary))
	}
	if summary[0].TotalInputTokens != 3000 {
		t.Errorf("input tokens = %d, want 3000", summary[0].TotalInputTokens)
	}
	if summary[0].TotalOutputTokens != 1500 {
		t.Errorf("output tokens = %d, want 1500", summary[0].TotalOutputTokens)
	}
	if summary[0].CallCount != 3 {
		t.Errorf("call count = %d, want 3", summary[0].CallCount)
	}
}

func TestCostTracker_UserIsolation(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}

	ctx := context.Background()
	if err := ct.Record(ctx, "alice", "s1", "gpt-4o", "openai", 500, 200); err != nil {
		t.Fatalf("Record alice: %v", err)
	}
	if err := ct.Record(ctx, "bob", "s2", "gpt-4o-mini", "openai", 1000, 400); err != nil {
		t.Fatalf("Record bob: %v", err)
	}

	aliceSummary, err := ct.UserSummary(ctx, "alice")
	if err != nil {
		t.Fatalf("UserSummary alice: %v", err)
	}
	if len(aliceSummary) != 1 {
		t.Fatalf("alice: expected 1 summary, got %d", len(aliceSummary))
	}
	if aliceSummary[0].Model != "gpt-4o" {
		t.Errorf("alice model = %q, want %q", aliceSummary[0].Model, "gpt-4o")
	}

	bobSummary, err := ct.UserSummary(ctx, "bob")
	if err != nil {
		t.Fatalf("UserSummary bob: %v", err)
	}
	if len(bobSummary) != 1 {
		t.Fatalf("bob: expected 1 summary, got %d", len(bobSummary))
	}
	if bobSummary[0].Model != "gpt-4o-mini" {
		t.Errorf("bob model = %q, want %q", bobSummary[0].Model, "gpt-4o-mini")
	}
}

func TestCostTracker_AllSummary(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}

	ctx := context.Background()
	if err := ct.Record(ctx, "alice", "s1", "gpt-4o", "openai", 100, 50); err != nil {
		t.Fatalf("Record alice: %v", err)
	}
	if err := ct.Record(ctx, "bob", "s2", "gpt-4o-mini", "openai", 200, 100); err != nil {
		t.Fatalf("Record bob: %v", err)
	}

	all, err := ct.AllSummary(ctx)
	if err != nil {
		t.Fatalf("AllSummary: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}

	// Results are ordered by user_id, model
	foundAlice, foundBob := false, false
	for _, entry := range all {
		switch entry.UserID {
		case "alice":
			foundAlice = true
			if entry.TotalInputTokens != 100 {
				t.Errorf("alice input = %d, want 100", entry.TotalInputTokens)
			}
		case "bob":
			foundBob = true
			if entry.TotalInputTokens != 200 {
				t.Errorf("bob input = %d, want 200", entry.TotalInputTokens)
			}
		}
	}
	if !foundAlice {
		t.Error("alice not found in AllSummary")
	}
	if !foundBob {
		t.Error("bob not found in AllSummary")
	}
}

func TestCostTracker_EstimatedCost(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}

	ctx := context.Background()
	// Record 1M input tokens and 1M output tokens for gpt-4o-mini
	if err := ct.Record(ctx, "user1", "s1", "gpt-4o-mini", "openai", 1_000_000, 1_000_000); err != nil {
		t.Fatalf("Record: %v", err)
	}

	summary, err := ct.UserSummary(ctx, "user1")
	if err != nil {
		t.Fatalf("UserSummary: %v", err)
	}
	if len(summary) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summary))
	}

	// gpt-4o-mini: $0.15/1M input + $0.60/1M output = $0.75
	cost := summary[0].EstimatedCostUSD
	if cost < 0.01 {
		t.Errorf("estimated cost = %f, expected non-zero positive value", cost)
	}
	// Expected: 0.15 + 0.60 = 0.75
	expectedCost := 0.75
	if cost < expectedCost-0.01 || cost > expectedCost+0.01 {
		t.Errorf("estimated cost = %f, want ~%f", cost, expectedCost)
	}
}
