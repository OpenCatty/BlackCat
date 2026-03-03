//go:build cgo

package whatsapp

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
)

// newClientFromDevice creates a whatsmeow client from a device store.
// Separated to avoid duplication between Start() and LoginInteractive().
func newClientFromDevice(deviceStore *store.Device) *whatsmeow.Client {
	return whatsmeow.NewClient(deviceStore, nil)
}
