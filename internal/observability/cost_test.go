//go:build cgo

package observability

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/startower-observability/blackcat/internal/config"
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

func TestCheckBudget(t *testing.T) {
	db := newTestDB(t)
	ct, err := NewCostTracker(db)
	if err != nil {
		t.Fatalf("NewCostTracker: %v", err)
	}
	ctx := context.Background()

	// Helper to insert usage using default pricing (input $2/M, output $8/M)
	insertUsage := func(userID string, inputTokens, outputTokens int) {
		t.Helper()
		if err := ct.Record(ctx, userID, "sess1", "gpt-4o", "openai", inputTokens, outputTokens); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	// Helper to calculate output tokens needed for a dollar amount at default pricing
	// defaultOutputPricePerM = $8.00, so outputTokens = dollars / 8.00 * 1_000_000 = dollars * 125_000
	outputTokensForDollars := func(dollars float64) int {
		return int(dollars * 125_000)
	}

	t.Run("Disabled", func(t *testing.T) {
		cfg := config.BudgetConfig{Enabled: false, DailyLimitUSD: 1.00}
		result, err := ct.CheckBudget(ctx, "user_disabled", cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetOK {
			t.Errorf("Status = %d, want BudgetOK (%d)", result.Status, BudgetOK)
		}
	})

	t.Run("UnderLimit", func(t *testing.T) {
		userID := "user_under"
		// Insert $8 spend (1M output tokens at $8/M = $8)
		insertUsage(userID, 0, outputTokensForDollars(8))

		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   100,
			MonthlyLimitUSD: 1000,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetOK {
			t.Errorf("Status = %d, want BudgetOK (%d)", result.Status, BudgetOK)
		}
	})

	t.Run("Warning", func(t *testing.T) {
		userID := "user_warn"
		// Insert $82 spend to be above 80% of $100 daily limit but below $100
		// 82 / 8.00 * 1M = 10.25M output tokens
		insertUsage(userID, 0, outputTokensForDollars(82))

		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   100,
			MonthlyLimitUSD: 1000,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetWarning {
			t.Errorf("Status = %d, want BudgetWarning (%d)", result.Status, BudgetWarning)
		}
		if result.DailySpend < 80 || result.DailySpend > 85 {
			t.Errorf("DailySpend = %f, want ~82", result.DailySpend)
		}
	})

	t.Run("Exceeded", func(t *testing.T) {
		userID := "user_exceeded"
		// Insert $110 spend to exceed $100 daily limit
		insertUsage(userID, 0, outputTokensForDollars(110))

		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   100,
			MonthlyLimitUSD: 1000,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetExceeded {
			t.Errorf("Status = %d, want BudgetExceeded (%d)", result.Status, BudgetExceeded)
		}
	})

	t.Run("DailyAndMonthly", func(t *testing.T) {
		userID := "user_both"
		// Insert $50 spend
		insertUsage(userID, 0, outputTokensForDollars(50))

		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   100,
			MonthlyLimitUSD: 500,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetOK {
			t.Errorf("Status = %d, want BudgetOK (%d)", result.Status, BudgetOK)
		}
		// Verify both daily and monthly spend are populated
		if result.DailySpend == 0 {
			t.Errorf("DailySpend should be populated, got %f", result.DailySpend)
		}
		if result.MonthlySpend == 0 {
			t.Errorf("MonthlySpend should be populated, got %f", result.MonthlySpend)
		}
		if result.DailyLimit != 100 {
			t.Errorf("DailyLimit = %f, want 100", result.DailyLimit)
		}
		if result.MonthlyLimit != 500 {
			t.Errorf("MonthlyLimit = %f, want 500", result.MonthlyLimit)
		}
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		userID := "user_zero"
		// Insert large usage
		insertUsage(userID, 0, outputTokensForDollars(500))

		// Zero limits = unlimited
		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   0,
			MonthlyLimitUSD: 0,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetOK {
			t.Errorf("Status = %d, want BudgetOK (%d) with unlimited limits", result.Status, BudgetOK)
		}
	})

	t.Run("DailyEmpty", func(t *testing.T) {
		userID := "user_empty"
		// No usage records at all
		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   100,
			MonthlyLimitUSD: 1000,
			WarnThreshold:   0.8,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetOK {
			t.Errorf("Status = %d, want BudgetOK (%d) for empty usage", result.Status, BudgetOK)
		}
		if result.DailySpend != 0 {
			t.Errorf("DailySpend = %f, want 0 for empty usage", result.DailySpend)
		}
	})

	t.Run("MonthlyBoundary", func(t *testing.T) {
		userID := "user_monthly_warn"
		// Insert usage at exactly 90% of $100 monthly limit = $90
		insertUsage(userID, 0, outputTokensForDollars(90))

		cfg := config.BudgetConfig{
			Enabled:         true,
			DailyLimitUSD:   0, // unlimited daily
			MonthlyLimitUSD: 100,
			WarnThreshold:   0.9,
		}
		result, err := ct.CheckBudget(ctx, userID, cfg)
		if err != nil {
			t.Fatalf("CheckBudget: %v", err)
		}
		if result.Status != BudgetWarning {
			t.Errorf("Status = %d, want BudgetWarning (%d) at 90%% threshold", result.Status, BudgetWarning)
		}
		if result.MonthlySpend < 85 || result.MonthlySpend > 95 {
			t.Errorf("MonthlySpend = %f, want ~90", result.MonthlySpend)
		}
	})
}
