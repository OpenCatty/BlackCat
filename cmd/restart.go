package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/service"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the BlackCat daemon",
	Long:  "Restart the BlackCat daemon service. Equivalent to stop + start.",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.New()
		if !mgr.IsInstalled() {
			return fmt.Errorf("daemon is not installed. Run 'blackcat onboard' first")
		}
		if err := mgr.Restart(); err != nil {
			return fmt.Errorf("failed to restart daemon: %w", err)
		}
		fmt.Println("BlackCat daemon restarted.")
		st, _ := mgr.Status()
		if st.Running {
			fmt.Printf("  PID: %d\n", st.PID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
