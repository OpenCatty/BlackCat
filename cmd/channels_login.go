package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/config"
)

var channelsLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to a messaging channel",
	Long: `Log in to a messaging channel interactively.

Examples:
  blackcat channels login --channel whatsapp
  blackcat channels login --channel telegram
  blackcat channels login --channel discord`,
	RunE: runChannelsLogin,
}

func init() {
	channelsCmd.AddCommand(channelsLoginCmd)
	channelsLoginCmd.Flags().String("channel", "", "Channel to log in to (whatsapp, telegram, discord)")
	channelsLoginCmd.Flags().Bool("no-ascii", false, "Disable ASCII QR code rendering (WhatsApp only)")
	_ = channelsLoginCmd.MarkFlagRequired("channel")
}

func runChannelsLogin(cmd *cobra.Command, args []string) error {
	channel, _ := cmd.Flags().GetString("channel")
	channel = strings.ToLower(strings.TrimSpace(channel))

	switch channel {
	case "whatsapp":
		return loginWhatsApp(cmd, args)
	case "telegram":
		return loginTelegram(cmd)
	case "discord":
		return loginDiscord(cmd)
	default:
		return fmt.Errorf("unknown channel: %s (available: whatsapp, telegram, discord)", channel)
	}
}

func loginTelegram(cmd *cobra.Command) error {
	var token string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Telegram Bot Token").
				Placeholder("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11").
				EchoMode(huh.EchoModePassword).
				Value(&token),
		),
	).Run()
	if err != nil {
		return fmt.Errorf("telegram login: %w", err)
	}

	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("telegram bot token cannot be empty")
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".blackcat", "config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil || cfg == nil {
		cfg = config.Defaults()
	}

	cfg.Channels.Telegram.Token = strings.TrimSpace(token)
	cfg.Channels.Telegram.Enabled = true

	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("✓ Telegram configured successfully")
	return nil
}

func loginDiscord(cmd *cobra.Command) error {
	var token string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Discord Bot Token").
				Placeholder("MTk4NjIyNDgzNDcxOTI1MjQ4.Cl2FMQ...").
				EchoMode(huh.EchoModePassword).
				Value(&token),
		),
	).Run()
	if err != nil {
		return fmt.Errorf("discord login: %w", err)
	}

	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("discord bot token cannot be empty")
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".blackcat", "config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil || cfg == nil {
		cfg = config.Defaults()
	}

	cfg.Channels.Discord.Token = strings.TrimSpace(token)
	cfg.Channels.Discord.Enabled = true

	if err := config.Save(configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("✓ Discord configured successfully")
	return nil
}
