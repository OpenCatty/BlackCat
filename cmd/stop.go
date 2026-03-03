package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/service"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the BlackCat daemon",
	Long:  "Stop the running BlackCat daemon service.",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.New()
		if !mgr.IsInstalled() {
			return fmt.Errorf("daemon is not installed")
		}
		st, _ := mgr.Status()
		if !st.Running {
			fmt.Println("BlackCat daemon is not running.")
			return nil
		}
		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
		fmt.Println("BlackCat daemon stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
