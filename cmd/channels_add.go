package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/config"
)

var channelsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a channel configuration non-interactively",
	Long: `Add or update a messaging channel token without interactive prompts.

Examples:
  blackcat channels add --channel telegram --token 123456:ABC-DEF
  blackcat channels add --channel discord --token MTk4NjIy...
  blackcat channels add --channel whatsapp --allow-from +628123456789,+628987654321`,
	RunE: runChannelsAdd,
}

func init() {
	channelsCmd.AddCommand(channelsAddCmd)
	channelsAddCmd.Flags().String("channel", "", "Channel to configure (whatsapp, telegram, discord)")
	channelsAddCmd.Flags().String("token", "", "Bot token for the channel")
	channelsAddCmd.Flags().String("allow-from", "", "Comma-separated phone whitelist for WhatsApp (E.164 format)")
	_ = channelsAddCmd.MarkFlagRequired("channel")
}

func runChannelsAdd(cmd *cobra.Command, args []string) error {
	channel, _ := cmd.Flags().GetString("channel")
	channel = strings.ToLower(strings.TrimSpace(channel))
	token, _ := cmd.Flags().GetString("token")
	token = strings.TrimSpace(token)
	allowFrom, _ := cmd.Flags().GetString("allow-from")

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".blackcat", "config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil || cfg == nil {
		cfg = config.Defaults()
	}

	switch channel {
	case "whatsapp":
		if token != "" {
			cfg.Channels.WhatsApp.Token = token
		}
		cfg.Channels.WhatsApp.Enabled = true
		if allowFrom != "" {
			phones := strings.Split(allowFrom, ",")
			for i := range phones {
				phones[i] = strings.TrimSpace(phones[i])
			}
			cfg.Channels.WhatsApp.AllowFrom = phones
		}
	case "telegram":
		if token == "" {
			return fmt.Errorf("--token is required for telegram")
		}
		cfg.Channels.Telegram.Token = token
		cfg.Channels.Telegram.Enabled = true
	case "discord":
		if token == "" {
			return fmt.Errorf("--token is required for discord")
		}
		cfg.Channels.Discord.Token = token
		cfg.Channels.Discord.Enabled = true
	default:
		return fmt.Errorf("unknown channel: %s (available: whatsapp, telegram, discord)", channel)
	}

	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("✓ %s configured successfully\n", channel)
	return nil
}
