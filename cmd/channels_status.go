package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var channelsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live channel connection status from the running daemon",
	Long: `Query the running BlackCat daemon health endpoint and display
per-channel connection status.

Examples:
  blackcat channels status`,
	RunE: runChannelsStatus,
}

func init() {
	channelsCmd.AddCommand(channelsStatusCmd)
}

func runChannelsStatus(cmd *cobra.Command, args []string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/health")
	if err != nil {
		fmt.Println("Daemon not running. Start with: blackcat start")
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read health response: %w", err)
	}

	var health struct {
		Channels map[string]struct {
			Status  string `json:"status"`
			Details string `json:"details"`
		} `json:"channels"`
	}

	if err := json.Unmarshal(body, &health); err != nil {
		// If we can't parse the structured response, show raw
		fmt.Printf("Daemon is running (HTTP %d)\n", resp.StatusCode)
		fmt.Println(string(body))
		return nil
	}

	fmt.Printf("%-12s %-12s %-30s\n", "CHANNEL", "STATUS", "DETAILS")
	fmt.Printf("%-12s %-12s %-30s\n", "-------", "------", "-------")

	for name, ch := range health.Channels {
		fmt.Printf("%-12s %-12s %-30s\n", name, ch.Status, ch.Details)
	}

	return nil
}
