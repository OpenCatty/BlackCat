package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/service"
)

var (
	uninstallAll bool
	uninstallYes bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the BlackCat daemon service",
	Long:  "Remove the BlackCat daemon service registration. Use --all to also delete all BlackCat data (~/.blackcat/).",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.New()
		if !mgr.IsInstalled() {
			fmt.Println("BlackCat daemon is not installed.")
			return nil
		}

		// Confirmation prompt.
		if !uninstallYes {
			var prompt string
			if uninstallAll {
				prompt = "This will PERMANENTLY DELETE all BlackCat data including config, vault, and WhatsApp sessions. Continue? [y/N] "
			} else {
				prompt = "This will remove the BlackCat daemon service. Your config and data will be preserved. Continue? [y/N] "
			}
			fmt.Print(prompt)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Stop if running.
		st, _ := mgr.Status()
		if st.Running {
			fmt.Println("Stopping daemon...")
			_ = mgr.Stop()
		}

		// Uninstall service.
		if err := mgr.Uninstall(); err != nil {
			return fmt.Errorf("failed to uninstall: %w", err)
		}
		fmt.Println("BlackCat daemon service removed.")

		// If --all, remove data directory.
		if uninstallAll {
			home, _ := os.UserHomeDir()
			dataDir := filepath.Join(home, ".blackcat")
			if err := os.RemoveAll(dataDir); err != nil {
				return fmt.Errorf("failed to remove data directory %s: %w", dataDir, err)
			}
			fmt.Printf("Removed %s\n", dataDir)
		}

		fmt.Println("BlackCat has been uninstalled.")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "also delete all BlackCat data (~/.blackcat/)")
	uninstallCmd.Flags().BoolVar(&uninstallYes, "yes", false, "skip confirmation prompt")
	rootCmd.AddCommand(uninstallCmd)
}
