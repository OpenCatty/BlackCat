package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startower-observability/blackcat/agent"
	"github.com/startower-observability/blackcat/opencode"
)

var runCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Submit a coding task to OpenCode and wait for the result",
	Long: `run sends a prompt to the OpenCode server as a coding task and
blocks until the session reaches idle state, then prints a summary.

Examples:
  blackcat run "Add unit tests to the calculator package"
  blackcat run --dir ./myproject "Refactor main.go to use slog"
  blackcat run --session abc123 "Continue with the previous refactor"`,
	Args: cobra.ExactArgs(1),
	RunE: runTask,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().String("dir", "", "Working directory for the OpenCode session")
	runCmd.Flags().String("session", "", "Reuse an existing session by ID")
	runCmd.Flags().String("model", "", "Model ID override (e.g. claude-3-5-sonnet)")
	runCmd.Flags().String("provider", "", "Provider ID override (e.g. anthropic)")
	runCmd.Flags().Bool("auto-permit", false, "Auto-approve all permission requests (DANGEROUS)")
	runCmd.Flags().Bool("verbose", true, "Stream progress events to stderr")
}

func runTask(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := viper.GetString("addr")
	password := viper.GetString("password")
	dir, _ := cmd.Flags().GetString("dir")
	sessionID, _ := cmd.Flags().GetString("session")
	modelID, _ := cmd.Flags().GetString("model")
	providerID, _ := cmd.Flags().GetString("provider")
	autoPermit, _ := cmd.Flags().GetBool("auto-permit")
	verbose, _ := cmd.Flags().GetBool("verbose")

	ag := agent.New(agent.Config{
		OpenCodeAddr: addr,
		Password:     password,
		AutoPermit:   autoPermit,
		Verbose:      verbose,
		Output:       os.Stderr,
	})

	if err := ag.Health(ctx); err != nil {
		return fmt.Errorf("opencode server not reachable at %s: %w\n\nHint: start with 'opencode serve' or use 'blackcat serve'", addr, err)
	}

	req := opencode.TaskRequest{
		Prompt:     args[0],
		SessionID:  sessionID,
		Dir:        dir,
		ModelID:    modelID,
		ProviderID: providerID,
		AutoPermit: autoPermit,
	}

	result, err := ag.Run(ctx, req)
	if err != nil {
		return fmt.Errorf("task failed: %w", err)
	}

	fmt.Fprintf(os.Stdout, "session: %s\nmessages: %d\n", result.SessionID, len(result.Messages))
	return nil
}
