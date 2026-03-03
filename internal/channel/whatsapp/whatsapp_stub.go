//go:build !cgo

// Package whatsapp implements a WhatsApp channel adapter using the unofficial
// whatsmeow library (go.mau.fi/whatsmeow). This is NOT an official WhatsApp
// Business API integration — it uses the WhatsApp Web multi-device protocol
// directly. Use at your own risk; WhatsApp may ban accounts using unofficial APIs.
//
// This is the non-CGO stub. whatsmeow requires CGO for its SQLite session store.
// Build with CGO_ENABLED=1 and a C compiler to use the full implementation.
package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/startower-observability/blackcat/internal/types"
)

// maxMessageLen is the soft limit per message chunk for readability.
const maxMessageLen = 4096

// WhatsAppChannel implements types.Channel using whatsmeow.
// This is a stub for builds without CGO — Start() returns an error.
type WhatsAppChannel struct {
	storePath string
	allowFrom map[string]bool // normalised E.164 → true; nil = allow all
	incoming  chan types.Message
	mu        sync.Mutex
	started   bool
	cancel    context.CancelFunc
}

// NewWhatsAppChannel creates a new WhatsApp channel adapter.
func NewWhatsAppChannel(storePath string, allowFrom []string) *WhatsAppChannel {
	ch := &WhatsAppChannel{
		storePath: storePath,
		incoming:  make(chan types.Message, 256),
	}
	if len(allowFrom) > 0 {
		ch.allowFrom = make(map[string]bool, len(allowFrom))
		for _, phone := range allowFrom {
			if phone == "*" {
				ch.allowFrom = nil
				break
			}
			ch.allowFrom[normalizeE164(phone)] = true
		}
	}
	return ch
}

// Start returns an error because CGO is required for whatsmeow's SQLite store.
func (w *WhatsAppChannel) Start(_ context.Context) error {
	return fmt.Errorf("whatsapp: CGO is required but not enabled; rebuild with CGO_ENABLED=1")
}

// Stop is a no-op in the non-CGO stub.
func (w *WhatsAppChannel) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	if w.cancel != nil {
		w.cancel()
	}

	close(w.incoming)
	w.started = false
	return nil
}

// Send returns an error because CGO is required.
func (w *WhatsAppChannel) Send(_ context.Context, _ types.Message) error {
	return fmt.Errorf("whatsapp: CGO is required but not enabled; rebuild with CGO_ENABLED=1")
}

// Receive returns the read-only channel of incoming messages.
func (w *WhatsAppChannel) Receive() <-chan types.Message {
	return w.incoming
}

// Info returns metadata about this channel.
func (w *WhatsAppChannel) Info() types.ChannelInfo {
	w.mu.Lock()
	defer w.mu.Unlock()
	return types.ChannelInfo{
		Type:      types.ChannelWhatsApp,
		Name:      "whatsapp",
		Connected: w.started,
	}
}

// Health checks the health of the WhatsApp channel.
// In non-CGO builds, the channel is always unhealthy since it cannot start.
func (w *WhatsAppChannel) Health() types.ChannelHealth {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return types.ChannelHealth{
			Name:    "whatsapp",
			Healthy: false,
			Details: "channel not started (CGO required)",
		}
	}

	return types.ChannelHealth{
		Name:    "whatsapp",
		Healthy: false,
		Details: "CGO required for full functionality",
	}
}

// splitMessage splits text into chunks of at most maxLen characters.
// It prefers splitting at newline boundaries for readability.
func splitMessage(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = maxMessageLen
	}
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		// Try to split at the last newline within maxLen.
		chunk := text[:maxLen]
		splitAt := strings.LastIndex(chunk, "\n")
		if splitAt <= 0 {
			// No newline found — hard split at maxLen.
			splitAt = maxLen
		} else {
			splitAt++ // include the newline in this chunk
		}

		chunks = append(chunks, text[:splitAt])
		text = text[splitAt:]
	}
	return chunks
}

// isAllowed checks whether a message sender passes the phone whitelist.
// resolvedPhone is the E.164 phone number resolved from LID (may be empty).
func (w *WhatsAppChannel) isAllowed(chatJID, senderJID, resolvedPhone string) bool {
	if w.allowFrom == nil {
		return true
	}
	if resolvedPhone != "" && w.allowFrom[resolvedPhone] {
		return true
	}
	if phone := phoneFromJID(chatJID); phone != "" && w.allowFrom[phone] {
		return true
	}
	if phone := phoneFromJID(senderJID); phone != "" && w.allowFrom[phone] {
		return true
	}
	return false
}

// phoneFromJID extracts a normalised E.164 phone number from a WhatsApp JID.
// Handles device suffix: "6282394921432:5@s.whatsapp.net" → "+6282394921432"
func phoneFromJID(jid string) string {
	user, _, _ := strings.Cut(jid, "@")
	if user == "" {
		return ""
	}
	if idx := strings.IndexByte(user, ':'); idx > 0 {
		user = user[:idx]
	}
	return "+" + user
}

// normalizeE164 normalises a phone string to E.164 format.
func normalizeE164(phone string) string {
	var buf strings.Builder
	for i, r := range phone {
		if r == '+' && i == 0 {
			buf.WriteRune(r)
		} else if r >= '0' && r <= '9' {
			buf.WriteRune(r)
		}
	}
	s := buf.String()
	if !strings.HasPrefix(s, "+") {
		s = "+" + s
	}
	return s
}
