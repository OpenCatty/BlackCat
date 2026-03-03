package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startower-observability/blackcat/opencode"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of the OpenCode server",
	RunE:  runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
	healthCmd.Flags().Bool("json", false, "Output result as JSON")
}

func runHealth(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	addr := viper.GetString("addr")
	password := viper.GetString("password")

	var opts []opencode.ClientOption
	if password != "" {
		opts = append(opts, opencode.WithPassword(password))
	}
	c := opencode.NewClient(addr, opts...)

	h, err := c.Health(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unhealthy: %v\n", err)
		os.Exit(1)
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(h)
	}
	fmt.Printf("healthy: %v\nversion: %s\n", h.Healthy, h.Version)
	return nil
}
