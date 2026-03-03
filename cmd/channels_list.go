package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/config"
)

var channelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all channel configurations",
	Long: `List all messaging channels with their enabled and configured status.

Examples:
  blackcat channels list
  blackcat channels list --json`,
	RunE: runChannelsList,
}

func init() {
	channelsCmd.AddCommand(channelsListCmd)
	channelsListCmd.Flags().Bool("json", false, "Output in JSON format")
}

type channelInfo struct {
	Channel    string `json:"channel"`
	Enabled    bool   `json:"enabled"`
	Configured bool   `json:"configured"`
}

func runChannelsList(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".blackcat", "config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil || cfg == nil {
		cfg = config.Defaults()
	}

	// Check if WhatsApp DB file exists
	waDBPath := filepath.Join(home, ".blackcat", "whatsapp.db")
	_, waDBErr := os.Stat(waDBPath)
	waDBExists := waDBErr == nil

	channels := []channelInfo{
		{
			Channel:    "whatsapp",
			Enabled:    cfg.Channels.WhatsApp.Enabled,
			Configured: cfg.Channels.WhatsApp.Token != "" || waDBExists,
		},
		{
			Channel:    "telegram",
			Enabled:    cfg.Channels.Telegram.Enabled,
			Configured: cfg.Channels.Telegram.Token != "",
		},
		{
			Channel:    "discord",
			Enabled:    cfg.Channels.Discord.Enabled,
			Configured: cfg.Channels.Discord.Token != "",
		},
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut {
		data, err := json.Marshal(channels)
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Print table
	fmt.Printf("%-12s %-10s %-12s\n", "CHANNEL", "ENABLED", "CONFIGURED")
	fmt.Printf("%-12s %-10s %-12s\n", "-------", "-------", "----------")
	for _, ch := range channels {
		enabled := "false"
		if ch.Enabled {
			enabled = "true"
		}
		configured := "false"
		if ch.Configured {
			configured = "true"
		}
		fmt.Printf("%-12s %-10s %-12s\n", ch.Channel, enabled, configured)
	}

	return nil
}
