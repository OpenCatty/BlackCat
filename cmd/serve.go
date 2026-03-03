package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/opencode"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start and supervise an opencode server process",
	Long: `serve launches 'opencode serve' as a managed child process,
waits for it to become healthy, then keeps it alive until interrupted.

Examples:
  blackcat serve
  blackcat serve --port 4096 --dir ./myproject`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().Int("port", 4096, "Port for the opencode server")
	serveCmd.Flags().String("dir", "", "Working directory for opencode")
	serveCmd.Flags().String("binary", "opencode", "Path to the opencode binary")
	serveCmd.Flags().String("password", "", "OPENCODE_SERVER_PASSWORD to set")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	port, _ := cmd.Flags().GetInt("port")
	dir, _ := cmd.Flags().GetString("dir")
	binary, _ := cmd.Flags().GetString("binary")
	password, _ := cmd.Flags().GetString("password")

	sup := opencode.NewSupervisor(opencode.SupervisorConfig{
		Binary:   binary,
		Port:     port,
		Dir:      dir,
		Password: password,
	})

	fmt.Fprintf(os.Stderr, "[blackcat] starting opencode serve on port %d...\n", port)
	if err := sup.Start(ctx); err != nil {
		return fmt.Errorf("failed to start opencode: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[blackcat] opencode server healthy at %s\n", sup.BaseURL())

	// Block until signal.
	<-ctx.Done()
	fmt.Fprintln(os.Stderr, "[blackcat] shutting down opencode server...")
	return sup.Stop()
}
