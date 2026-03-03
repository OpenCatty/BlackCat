// Package telegram implements a Telegram channel adapter using the Bot API.
package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/startower-observability/blackcat/channel"
	"github.com/startower-observability/blackcat/types"
)

const (
	// maxMessageLen is Telegram's maximum message length.
	maxMessageLen = 4096
	// incomingBufferSize is the buffer size for the incoming message channel.
	incomingBufferSize = 256
)

// TelegramChannel implements types.Channel for the Telegram messaging platform.
type TelegramChannel struct {
	bot      *tgbotapi.BotAPI
	token    string
	incoming chan types.Message
	mu       sync.Mutex
	started  bool
	cancel   context.CancelFunc
}

// compile-time interface check
var _ types.Channel = (*TelegramChannel)(nil)

// NewTelegramChannel creates a new Telegram channel adapter with the given bot token.
func NewTelegramChannel(token string) *TelegramChannel {
	return &TelegramChannel{
		token:    token,
		incoming: make(chan types.Message, incomingBufferSize),
	}
}

// Start connects to the Telegram Bot API and begins polling for updates.
func (t *TelegramChannel) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.started {
		return fmt.Errorf("telegram channel already started")
	}

	bot, err := tgbotapi.NewBotAPI(t.token)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	t.bot = bot
	t.started = true

	ctx, t.cancel = context.WithCancel(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("telegram context cancelled, stopping updates")
				bot.StopReceivingUpdates()
				return
			case update := <-updates:
				msg, ok := convertUpdate(update)
				if !ok {
					continue
				}
				select {
				case t.incoming <- msg:
				default:
					slog.Warn("telegram incoming buffer full, dropping message",
						"id", msg.ID)
				}
			}
		}
	}()

	slog.Info("telegram channel started", "bot", bot.Self.UserName)
	return nil
}

// Stop closes the Telegram bot and incoming channel.
func (t *TelegramChannel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	close(t.incoming)
	t.started = false

	slog.Info("telegram channel stopped")
	return nil
}

// Send sends a message to a Telegram chat. Messages are split into up to 5
// human-like bubbles using channel.SplitBubbles.
func (t *TelegramChannel) Send(ctx context.Context, msg types.Message) error {
	t.mu.Lock()
	bot := t.bot
	started := t.started
	t.mu.Unlock()

	if !started || bot == nil {
		return fmt.Errorf("telegram channel not started")
	}

	chatID, err := strconv.ParseInt(msg.ChannelID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat ID %q: %w", msg.ChannelID, err)
	}

	bubbles := channel.SplitBubbles(msg.Content, 5, 0)

	totalDelay := time.Duration(0)
	const maxTotalDelay = 10 * time.Second

	for i, bubble := range bubbles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

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

		tgMsg := tgbotapi.NewMessage(chatID, bubble)
		if msg.ReplyTo != "" && i == 0 {
			replyID, parseErr := strconv.Atoi(msg.ReplyTo)
			if parseErr == nil {
				tgMsg.ReplyToMessageID = replyID
			}
		}

		if _, sendErr := bot.Send(tgMsg); sendErr != nil {
			return fmt.Errorf("failed to send telegram message: %w", sendErr)
		}
	}

	return nil
}

// Receive returns a read-only channel of incoming messages.
func (t *TelegramChannel) Receive() <-chan types.Message {
	return t.incoming
}

// Info returns metadata about this channel.
func (t *TelegramChannel) Info() types.ChannelInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	return types.ChannelInfo{
		Type:      types.ChannelTelegram,
		Name:      "telegram",
		Connected: t.started,
	}
}

// Health checks the health of the Telegram channel by testing API connectivity.
func (t *TelegramChannel) Health() types.ChannelHealth {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started || t.bot == nil {
		return types.ChannelHealth{
			Name:    "telegram",
			Healthy: false,
			Details: "channel not started",
		}
	}

	_, err := t.bot.GetMe()
	if err != nil {
		return types.ChannelHealth{
			Name:    "telegram",
			Healthy: false,
			Details: fmt.Sprintf("API error: %v", err),
		}
	}

	return types.ChannelHealth{
		Name:    "telegram",
		Healthy: true,
		Details: "connected and responsive",
	}
}

// convertUpdate converts a tgbotapi.Update to a types.Message.
// Returns false if the update contains no text content.
func convertUpdate(update tgbotapi.Update) (types.Message, bool) {
	if update.Message == nil || update.Message.Text == "" {
		return types.Message{}, false
	}

	m := update.Message
	msg := types.Message{
		ID:          strconv.Itoa(m.MessageID),
		ChannelType: types.ChannelTelegram,
		ChannelID:   strconv.FormatInt(m.Chat.ID, 10),
		UserID:      strconv.FormatInt(m.From.ID, 10),
		Content:     m.Text,
		Timestamp:   time.Unix(int64(m.Date), 0),
		Metadata: map[string]string{
			"chat_type": m.Chat.Type,
		},
	}

	if m.ReplyToMessage != nil && m.ReplyToMessage.MessageID != 0 {
		msg.ReplyTo = strconv.Itoa(m.ReplyToMessage.MessageID)
	}

	return msg, true
}

// randomDelay returns a random duration between minMs and maxMs milliseconds.
func randomDelay(minMs, maxMs int) time.Duration {
	return time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
}
