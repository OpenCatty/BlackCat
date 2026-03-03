package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startower-observability/blackcat/internal/opencode"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List all OpenCode sessions",
	RunE:  runSessions,
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.Flags().Bool("json", false, "Output as JSON")
}

func runSessions(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	addr := viper.GetString("addr")
	password := viper.GetString("password")

	var opts []opencode.ClientOption
	if password != "" {
		opts = append(opts, opencode.WithPassword(password))
	}
	c := opencode.NewClient(addr, opts...)

	sessions, err := c.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}
	if len(sessions) == 0 {
		fmt.Println("no sessions")
		return nil
	}
	for _, s := range sessions {
		fmt.Printf("%s\t%s\t%s\n", s.ID, s.Title, s.Directory)
	}
	return nil
}
