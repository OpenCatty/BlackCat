//go:build cgo

package whatsapp

import (
	"context"
	"fmt"
	"io"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"

	"github.com/startower-observability/blackcat/internal/qr"
)

// LoginInteractive initiates an interactive WhatsApp QR login from the terminal.
// It opens the whatsmeow SQLite store at storePath, and if no session exists,
// renders the QR code to w for the user to scan.
// If already logged in, it returns immediately with a message.
// noASCII=true renders a more compact QR using block characters instead of ASCII art.
func LoginInteractive(ctx context.Context, storePath string, w io.Writer, noASCII bool) error {
	container, err := sqlstore.New(ctx, "sqlite3", storePath, nil)
	if err != nil {
		return fmt.Errorf("whatsapp login: open store %q: %w", storePath, err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp login: get device: %w", err)
	}

	// Already logged in — nothing to do.
	if deviceStore.ID != nil {
		fmt.Fprintln(w, "WhatsApp: already logged in.")
		return nil
	}

	client := newClientFromDevice(deviceStore)
	defer client.Disconnect()

	qrChan, _ := client.GetQRChannel(ctx)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("whatsapp login: connect: %w", err)
	}

	fmt.Fprintln(w, "Scan the QR code below with your WhatsApp mobile app:")
	fmt.Fprintln(w)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("whatsapp login: timed out waiting for QR scan")
		case evt, ok := <-qrChan:
			if !ok {
				return fmt.Errorf("whatsapp login: QR channel closed unexpectedly")
			}
			switch evt.Event {
			case "code":
				if noASCII {
					qr.RenderToTerminalBasic(w, evt.Code)
				} else {
					qr.RenderToTerminal(w, evt.Code)
				}
			case "success":
				fmt.Fprintln(w)
				fmt.Fprintln(w, "✓ WhatsApp login successful!")
				return nil
			case "timeout":
				return fmt.Errorf("whatsapp login: QR code timed out — please try again")
			case "err":
				return fmt.Errorf("whatsapp login: error event received")
			}
		}
	}
}
