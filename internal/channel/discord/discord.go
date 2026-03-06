package discord

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/startower-observability/blackcat/internal/channel"
	"github.com/startower-observability/blackcat/internal/types"
)

const (
	// maxMessageLen is Discord's maximum message length.
	maxMessageLen = 2000
	// incomingBufferSize is the buffer size for the incoming message channel.
	incomingBufferSize = 256
)

// DiscordChannel implements types.Channel for the Discord messaging platform.
type DiscordChannel struct {
	session  *discordgo.Session
	token    string
	incoming chan types.Message
	mu       sync.Mutex
	started  bool
	cancel   context.CancelFunc
}

// compile-time interface check
var _ types.Channel = (*DiscordChannel)(nil)

// NewDiscordChannel creates a new Discord channel adapter with the given bot token.
func NewDiscordChannel(token string) *DiscordChannel {
	return &DiscordChannel{
		token:    token,
		incoming: make(chan types.Message, incomingBufferSize),
	}
}

// Start opens a WebSocket connection to Discord and begins listening for messages.
func (d *DiscordChannel) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.started {
		return fmt.Errorf("discord channel already started")
	}

	session, err := discordgo.New("Bot " + d.token)
	if err != nil {
		return fmt.Errorf("failed to create discord session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Skip the bot's own messages.
		if s.State != nil && s.State.User != nil && m.Author.ID == s.State.User.ID {
			return
		}

		msg := convertMessage(m)

		select {
		case d.incoming <- msg:
		default:
			slog.Warn("discord incoming buffer full, dropping message", "id", m.ID)
		}
	})

	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to open discord websocket: %w", err)
	}

	d.session = session
	d.started = true

	ctx, d.cancel = context.WithCancel(ctx)

	go func() {
		<-ctx.Done()
		slog.Info("discord context cancelled, closing session")
		_ = session.Close()
	}()

	slog.Info("discord channel started")
	return nil
}

// Stop closes the Discord session and incoming channel.
func (d *DiscordChannel) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.started {
		return nil
	}

	if d.cancel != nil {
		d.cancel()
	}

	var err error
	if d.session != nil {
		err = d.session.Close()
		d.session = nil
	}

	close(d.incoming)
	d.started = false

	slog.Info("discord channel stopped")
	return err
}

// Send sends a message to a Discord channel. Messages are split into up to 5
// human-like bubbles using channel.SplitBubbles. A typing indicator is sent once
// before the first bubble, and random delays (300-800ms) are added between bubbles.
func (d *DiscordChannel) Send(ctx context.Context, msg types.Message) error {
	d.mu.Lock()
	session := d.session
	started := d.started
	d.mu.Unlock()

	if !started || session == nil {
		return fmt.Errorf("discord channel not started")
	}

	bubbles := channel.SplitBubbles(msg.Content, 5, 0)

	// Send typing indicator once before the first bubble.
	_ = session.ChannelTyping(msg.ChannelID)

	totalDelay := time.Duration(0)
	const maxTotalDelay = 10 * time.Second

	for i, bubble := range bubbles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Add a random delay between bubbles (not before the first).
		if i > 0 && totalDelay < maxTotalDelay {
			delay := randomDelay(300, 800)
			if totalDelay+delay > maxTotalDelay {
				delay = maxTotalDelay - totalDelay
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			totalDelay += delay
		}

		if msg.ReplyTo != "" && i == 0 {
			ref := &discordgo.MessageReference{
				MessageID: msg.ReplyTo,
				ChannelID: msg.ChannelID,
			}
			_, err := session.ChannelMessageSendReply(msg.ChannelID, bubble, ref)
			if err != nil {
				return fmt.Errorf("failed to send discord reply: %w", err)
			}
		} else {
			_, err := session.ChannelMessageSend(msg.ChannelID, bubble)
			if err != nil {
				return fmt.Errorf("failed to send discord message: %w", err)
			}
		}
	}

	return nil
}

// Receive returns a read-only channel of incoming messages.
func (d *DiscordChannel) Receive() <-chan types.Message {
	return d.incoming
}

// Health checks the health of the Discord channel by testing API connectivity.
func (d *DiscordChannel) Health() types.ChannelHealth {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.started || d.session == nil {
		return types.ChannelHealth{
			Name:    "discord",
			Healthy: false,
			Details: "channel not started",
		}
	}

	// Test API connectivity by getting user info
	_, err := d.session.User("@me")
	if err != nil {
		return types.ChannelHealth{
			Name:    "discord",
			Healthy: false,
			Details: fmt.Sprintf("API error: %v", err),
		}
	}

	return types.ChannelHealth{
		Name:    "discord",
		Healthy: true,
		Details: "connected and responsive",
	}
}

// convertMessage converts a discordgo MessageCreate event to a types.Message.
func (d *DiscordChannel) Info() types.ChannelInfo {
	d.mu.Lock()
	defer d.mu.Unlock()

	return types.ChannelInfo{
		Type:      types.ChannelDiscord,
		Name:      "discord",
		Connected: d.started,
	}
}

// convertMessage converts a discordgo MessageCreate event to a types.Message.
func convertMessage(m *discordgo.MessageCreate) types.Message {
	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	msg := types.Message{
		ID:          m.ID,
		ChannelType: types.ChannelDiscord,
		ChannelID:   m.ChannelID,
		UserID:      m.Author.ID,
		Content:     m.Content,
		Timestamp:   ts,
	}

	if m.MessageReference != nil && m.MessageReference.MessageID != "" {
		msg.ReplyTo = m.MessageReference.MessageID
	}

	// Detect audio attachments
	for _, att := range m.Attachments {
		isAudio := (att.ContentType != "" && strings.HasPrefix(att.ContentType, "audio/")) ||
			strings.HasSuffix(att.Filename, ".ogg") ||
			strings.HasSuffix(att.Filename, ".mp3") ||
			strings.HasSuffix(att.Filename, ".wav") ||
			strings.HasSuffix(att.Filename, ".m4a")
		if isAudio {
			msg.MediaType = "audio"
			msg.MediaURL = att.URL
			msg.MediaSize = int64(att.Size)
			break
		}
	}

	return msg
}

// randomDelay returns a random duration between minMs and maxMs milliseconds.
func randomDelay(minMs, maxMs int) time.Duration {
	return time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
}
