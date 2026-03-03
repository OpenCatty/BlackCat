package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/config"
)

var channelsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a channel configuration with confirmation",
	Long: `Remove a messaging channel configuration after interactive confirmation.
This clears the token, disables the channel, and removes any session files.

Examples:
  blackcat channels remove --channel whatsapp
  blackcat channels remove --channel telegram
  blackcat channels remove --channel discord`,
	RunE: runChannelsRemove,
}

func init() {
	channelsCmd.AddCommand(channelsRemoveCmd)
	channelsRemoveCmd.Flags().String("channel", "", "Channel to remove (whatsapp, telegram, discord)")
	_ = channelsRemoveCmd.MarkFlagRequired("channel")
}

func runChannelsRemove(cmd *cobra.Command, args []string) error {
	channel, _ := cmd.Flags().GetString("channel")
	channel = strings.ToLower(strings.TrimSpace(channel))

	if channel != "whatsapp" && channel != "telegram" && channel != "discord" {
		return fmt.Errorf("unknown channel: %s (available: whatsapp, telegram, discord)", channel)
	}

	var confirmed bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Remove %s configuration?", channel)).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).Run()
	if err != nil {
		return fmt.Errorf("confirmation: %w", err)
	}

	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".blackcat", "config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil || cfg == nil {
		cfg = config.Defaults()
	}

	switch channel {
	case "whatsapp":
		cfg.Channels.WhatsApp.Token = ""
		cfg.Channels.WhatsApp.Enabled = false
		cfg.Channels.WhatsApp.AllowFrom = nil
		// Also remove the session database
		dbPath := filepath.Join(home, ".blackcat", "whatsapp.db")
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove WhatsApp session: %w", err)
		}
	case "telegram":
		cfg.Channels.Telegram.Token = ""
		cfg.Channels.Telegram.Enabled = false
	case "discord":
		cfg.Channels.Discord.Token = ""
		cfg.Channels.Discord.Enabled = false
	}

	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("✓ %s configuration removed\n", channel)
	return nil
}
