package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/observability"

	_ "github.com/mattn/go-sqlite3"
)

var usageUser string

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show token usage and estimated cost",
	Long:  "Display token usage statistics and estimated costs per model, grouped by user.",
	RunE:  runUsage,
}

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.Flags().StringVar(&usageUser, "user", "", "Filter usage for a specific user ID")
}

func runUsage(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	dbPath := filepath.Join(homeDir, ".blackcat", "memory.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "No usage data found. The daemon has not recorded any token usage yet.")
		return nil
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&mode=ro")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	tracker, err := observability.NewCostTracker(db)
	if err != nil {
		return fmt.Errorf("init cost tracker: %w", err)
	}

	ctx := cmd.Context()

	if usageUser != "" {
		summary, err := tracker.UserSummary(ctx, usageUser)
		if err != nil {
			return fmt.Errorf("query user summary: %w", err)
		}
		if len(summary) == 0 {
			fmt.Printf("No usage data found for user %q.\n", usageUser)
			return nil
		}

		fmt.Printf("Token usage for user: %s\n\n", usageUser)
		fmt.Printf("%-25s %-15s %15s %15s %12s %8s\n",
			"Model", "Provider", "Input Tokens", "Output Tokens", "Cost (USD)", "Calls")
		fmt.Printf("%-25s %-15s %15s %15s %12s %8s\n",
			"-------------------------", "---------------", "---------------", "---------------", "------------", "--------")

		var totalCost float64
		for _, m := range summary {
			fmt.Printf("%-25s %-15s %15d %15d %12.4f %8d\n",
				m.Model, m.Provider, m.TotalInputTokens, m.TotalOutputTokens, m.EstimatedCostUSD, m.CallCount)
			totalCost += m.EstimatedCostUSD
		}
		fmt.Printf("\nTotal estimated cost: $%.4f\n", totalCost)
	} else {
		all, err := tracker.AllSummary(ctx)
		if err != nil {
			return fmt.Errorf("query all summary: %w", err)
		}
		if len(all) == 0 {
			fmt.Println("No usage data recorded yet.")
			return nil
		}

		fmt.Printf("%-15s %-25s %-15s %15s %15s %12s %8s\n",
			"User", "Model", "Provider", "Input Tokens", "Output Tokens", "Cost (USD)", "Calls")
		fmt.Printf("%-15s %-25s %-15s %15s %15s %12s %8s\n",
			"---------------", "-------------------------", "---------------", "---------------", "---------------", "------------", "--------")

		var totalCost float64
		for _, m := range all {
			fmt.Printf("%-15s %-25s %-15s %15d %15d %12.4f %8d\n",
				m.UserID, m.Model, m.Provider, m.TotalInputTokens, m.TotalOutputTokens, m.EstimatedCostUSD, m.CallCount)
			totalCost += m.EstimatedCostUSD
		}
		fmt.Printf("\nTotal estimated cost: $%.4f\n", totalCost)
	}

	return nil
}
