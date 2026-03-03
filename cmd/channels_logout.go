package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/config"
)

var channelsLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from a messaging channel",
	Long: `Clear authentication credentials for a messaging channel.

Examples:
  blackcat channels logout --channel whatsapp
  blackcat channels logout --channel telegram
  blackcat channels logout --channel discord`,
	RunE: runChannelsLogout,
}

func init() {
	channelsCmd.AddCommand(channelsLogoutCmd)
	channelsLogoutCmd.Flags().String("channel", "", "Channel to log out from (whatsapp, telegram, discord)")
	_ = channelsLogoutCmd.MarkFlagRequired("channel")
}

func runChannelsLogout(cmd *cobra.Command, args []string) error {
	channel, _ := cmd.Flags().GetString("channel")
	channel = strings.ToLower(strings.TrimSpace(channel))

	home, _ := os.UserHomeDir()

	switch channel {
	case "whatsapp":
		dbPath := filepath.Join(home, ".blackcat", "whatsapp.db")
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove WhatsApp session: %w", err)
		}
		fmt.Println("WhatsApp session cleared")

	case "telegram":
		configPath := filepath.Join(home, ".blackcat", "config.yaml")
		cfg, err := config.Load(configPath)
		if err != nil || cfg == nil {
			cfg = config.Defaults()
		}
		cfg.Channels.Telegram.Token = ""
		cfg.Channels.Telegram.Enabled = false
		if err := config.Save(configPath, cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("Telegram token cleared")

	case "discord":
		configPath := filepath.Join(home, ".blackcat", "config.yaml")
		cfg, err := config.Load(configPath)
		if err != nil || cfg == nil {
			cfg = config.Defaults()
		}
		cfg.Channels.Discord.Token = ""
		cfg.Channels.Discord.Enabled = false
		if err := config.Save(configPath, cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("Discord token cleared")

	default:
		return fmt.Errorf("unknown channel: %s (available: whatsapp, telegram, discord)", channel)
	}

	return nil
}
