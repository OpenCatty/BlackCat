//go:build !cgo

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func loginWhatsApp(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("WhatsApp login requires CGO: rebuild with CGO_ENABLED=1")
}
