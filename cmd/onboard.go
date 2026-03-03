package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/service"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for BlackCat",
	Long: `onboard walks you through the complete BlackCat setup:
  1. Configure an LLM provider
  2. Set up a messaging channel
  3. Install and start the daemon
  4. Run a health check

Run this after a fresh install to get BlackCat working in minutes.`,
	RunE: runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)
	onboardCmd.Flags().Bool("non-interactive", false, "Skip all prompts (for CI/scripted use)")
}

func runOnboard(cmd *cobra.Command, args []string) error {
	// Print banner
	fmt.Println()
	fmt.Println("  ▄▄▄▄    ██▓    ▄▄▄       ▄████▄   ██ ▄█▀ ▄████▄  ▄▄▄     ▄▄▄█████▓")
	fmt.Println("  ▓█████▄ ▓██▒   ▒████▄    ▒██▀ ▀█   ██▄█▒ ▒██▀ ▀█ ▒████▄   ▓  ██▒ ▓▒")
	fmt.Println("  ▒██▒ ▄██▒██░   ▒██  ▀█▄  ▒▓█    ▄ ▓███▄░ ▒▓█    ▄▒██  ▀█▄ ▒ ▓██░ ▒░")
	fmt.Println("  ▒██░█▀  ▒██░   ░██▄▄▄▄██ ▒▓▓▄ ▄██▒▓██ █▄ ▒▓▓▄ ▄██░██▄▄▄▄██░ ▓██▓ ░")
	fmt.Println("  ░▓█  ▀█▓░██████▒▓█   ▓██▒▒ ▓███▀ ░▒██▒ █▄▒ ▓███▀ ░▓█   ▓██▒ ▒██▒ ░")
	fmt.Println("  ░▒▓███▀▒░ ▒░▓  ░▒▒   ▓▒█░░ ░▒ ▒  ░▒ ▒▒ ▓▒░ ░▒ ▒  ░▒▒   ▓▒█░ ▒ ░░")
	fmt.Println("  ▒░▒   ░ ░ ░ ▒  ░ ▒   ▒▒ ░  ░  ▒   ░ ░▒ ▒░  ░  ▒    ▒   ▒▒ ░   ░")
	fmt.Println()
	fmt.Println("  Welcome to BlackCat — AI agent for your messaging channels")
	fmt.Println("  ─────────────────────────────────────────────────────────")
	fmt.Println()

	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	// Step 1: LLM Provider
	fmt.Println("Step 1/4: Configure LLM Provider")
	fmt.Println("─────────────────────────────────")
	if !nonInteractive {
		if err := configureInteractive(cmd); err != nil {
			fmt.Printf("  ⚠ Provider setup skipped: %v\n", err)
		}
	} else {
		fmt.Println("  (skipped — non-interactive mode)")
	}
	fmt.Println()

	// Step 2: Channel Setup
	fmt.Println("Step 2/4: Set Up a Messaging Channel")
	fmt.Println("─────────────────────────────────────")
	if !nonInteractive {
		if err := onboardChannel(cmd); err != nil {
			fmt.Printf("  ⚠ Channel setup skipped: %v\n", err)
		}
	} else {
		fmt.Println("  (skipped — non-interactive mode)")
	}
	fmt.Println()

	// Step 3: Daemon Install + Start
	fmt.Println("Step 3/4: Install and Start Daemon")
	fmt.Println("───────────────────────────────────")
	if err := onboardDaemon(cmd, nonInteractive); err != nil {
		fmt.Printf("  ⚠ Daemon setup: %v\n", err)
	}
	fmt.Println()

	// Step 4: Health Check
	fmt.Println("Step 4/4: Health Check")
	fmt.Println("───────────────────────")
	onboardHealthCheck()
	fmt.Println()

	// Done
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  BlackCat setup complete!")
	fmt.Println()
	fmt.Println("  Quick reference:")
	fmt.Println("    blackcat start       — start daemon")
	fmt.Println("    blackcat stop        — stop daemon")
	fmt.Println("    blackcat status      — check status")
	fmt.Println("    blackcat channels list  — list channels")
	fmt.Println("    blackcat doctor      — run diagnostics")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	return nil
}

func onboardChannel(cmd *cobra.Command) error {
	var channelChoice string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which channel do you want to set up?").
				Options(
					huh.NewOption("Telegram (recommended — easy token setup)", "telegram"),
					huh.NewOption("Discord", "discord"),
					huh.NewOption("WhatsApp (requires CGO build)", "whatsapp"),
					huh.NewOption("Skip for now", "skip"),
				).
				Value(&channelChoice),
		),
	).Run()
	if err != nil {
		return err
	}
	switch channelChoice {
	case "skip":
		fmt.Println("  Channel setup skipped. Run: blackcat channels login --channel <name>")
		return nil
	case "telegram":
		return loginTelegram(cmd)
	case "discord":
		return loginDiscord(cmd)
	case "whatsapp":
		return loginWhatsApp(cmd, nil)
	}
	return nil
}

func onboardDaemon(cmd *cobra.Command, nonInteractive bool) error {
	svc := service.New()

	if svc.IsInstalled() {
		fmt.Println("  Daemon already installed.")
	} else {
		home, _ := os.UserHomeDir()
		binaryPath := filepath.Join(home, ".blackcat", "bin", "blackcat")
		// Fall back to current executable if not at user-space path
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			binaryPath, _ = os.Executable()
		}
		configPath := filepath.Join(home, ".blackcat", "config.yaml")
		cfg := service.DefaultConfig()
		cfg.BinaryPath = binaryPath
		cfg.ConfigPath = configPath
		if err := svc.Install(cfg); err != nil {
			return fmt.Errorf("install service: %w", err)
		}
		fmt.Println("  ✓ Daemon installed")
	}

	status, _ := svc.Status()
	if status.Running {
		fmt.Println("  Daemon already running.")
		return nil
	}

	if !nonInteractive {
		var startNow bool
		_ = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Start the BlackCat daemon now?").
					Value(&startNow),
			),
		).Run()
		if !startNow {
			fmt.Println("  Start later with: blackcat start")
			return nil
		}
	}

	if err := svc.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	fmt.Println("  ✓ Daemon started")
	return nil
}

func onboardHealthCheck() {
	fmt.Print("  Checking daemon health... ")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/health")
	if err != nil {
		fmt.Println("⚠ Daemon not reachable (may still be starting up)")
		fmt.Println("  Run: blackcat status  to check later")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		fmt.Println("✓ Healthy")
	} else {
		fmt.Printf("⚠ Status %d\n", resp.StatusCode)
	}
}
