//go:build cgo

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/startower-observability/blackcat/internal/channel/whatsapp"
)

func loginWhatsApp(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	storePath := filepath.Join(home, ".blackcat", "whatsapp.db")
	noASCII, _ := cmd.Flags().GetBool("no-ascii")
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()
	return whatsapp.LoginInteractive(ctx, storePath, os.Stdout, noASCII)
}
