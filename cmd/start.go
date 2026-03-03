package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/service"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the BlackCat daemon",
	Long:  "Start the BlackCat daemon as a background service. The daemon must be installed first (use 'blackcat onboard' or install manually).",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.New()
		if !mgr.IsInstalled() {
			return fmt.Errorf("daemon is not installed. Run 'blackcat onboard' first")
		}
		if err := mgr.Start(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}
		fmt.Println("BlackCat daemon started.")
		st, _ := mgr.Status()
		if st.Running {
			fmt.Printf("  PID: %d\n", st.PID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
