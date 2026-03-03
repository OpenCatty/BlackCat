package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startower-observability/blackcat/internal/service"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show BlackCat daemon status",
	Long:  "Show the current state of the BlackCat daemon including service status, health, and uptime.",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := service.New()
		st, err := mgr.Status()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		// Probe health endpoint if daemon is running.
		var healthStatus string
		if st.Running {
			healthStatus = probeHealth()
		}

		if statusJSON {
			return printStatusJSON(st, healthStatus)
		}

		fmt.Println("BlackCat Daemon")

		if !st.Installed {
			fmt.Println("  Service:  not installed")
			fmt.Println("  Run 'blackcat onboard' to install the daemon.")
			return nil
		}

		if st.Running {
			fmt.Printf("  Service:  installed, running (PID %d)\n", st.PID)
			if st.Uptime > 0 {
				fmt.Printf("  Uptime:   %s\n", formatDuration(st.Uptime))
			}
			fmt.Printf("  Health:   %s\n", healthStatus)
		} else {
			fmt.Println("  Service:  installed, stopped")
			fmt.Println("  Run 'blackcat start' to start the daemon.")
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output in JSON format")
	rootCmd.AddCommand(statusCmd)
}

func probeHealth() string {
	addr := viper.GetString("addr")
	if addr == "" {
		addr = "http://127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(addr + "/health")
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return "healthy"
	}
	return fmt.Sprintf("unhealthy (HTTP %d)", resp.StatusCode)
}

type statusOutput struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
	Health    string `json:"health,omitempty"`
}

func printStatusJSON(st service.ServiceStatus, health string) error {
	out := statusOutput{
		Installed: st.Installed,
		Running:   st.Running,
		PID:       st.PID,
		Health:    health,
	}
	if st.Uptime > 0 {
		out.Uptime = st.Uptime.String()
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
}
