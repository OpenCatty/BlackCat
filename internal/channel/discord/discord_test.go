package discord

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/startower-observability/blackcat/internal/types"
)

func TestConvertMessage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	mc := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-123",
			ChannelID: "chan-456",
			Content:   "hello world",
			Timestamp: now,
			Author:    &discordgo.User{ID: "user-789"},
		},
	}

	msg := convertMessage(mc)

	if msg.ID != "msg-123" {
		t.Errorf("expected ID %q, got %q", "msg-123", msg.ID)
	}
	if msg.ChannelType != types.ChannelDiscord {
		t.Errorf("expected ChannelType %q, got %q", types.ChannelDiscord, msg.ChannelType)
	}
	if msg.ChannelID != "chan-456" {
		t.Errorf("expected ChannelID %q, got %q", "chan-456", msg.ChannelID)
	}
	if msg.UserID != "user-789" {
		t.Errorf("expected UserID %q, got %q", "user-789", msg.UserID)
	}
	if msg.Content != "hello world" {
		t.Errorf("expected Content %q, got %q", "hello world", msg.Content)
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, msg.Timestamp)
	}
	if msg.ReplyTo != "" {
		t.Errorf("expected empty ReplyTo, got %q", msg.ReplyTo)
	}
}

func TestConvertMessageWithReply(t *testing.T) {
	mc := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-reply-1",
			ChannelID: "chan-1",
			Content:   "replying to you",
			Timestamp: time.Now(),
			Author:    &discordgo.User{ID: "user-1"},
			MessageReference: &discordgo.MessageReference{
				MessageID: "original-msg-42",
				ChannelID: "chan-1",
			},
		},
	}

	msg := convertMessage(mc)

	if msg.ReplyTo != "original-msg-42" {
		t.Errorf("expected ReplyTo %q, got %q", "original-msg-42", msg.ReplyTo)
	}
}

func TestNewDiscordChannel(t *testing.T) {
	ch := NewDiscordChannel("test-token-abc")

	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	if ch.token != "test-token-abc" {
		t.Errorf("expected token %q, got %q", "test-token-abc", ch.token)
	}
	if ch.incoming == nil {
		t.Fatal("expected non-nil incoming channel")
	}
	if cap(ch.incoming) != incomingBufferSize {
		t.Errorf("expected incoming buffer size %d, got %d", incomingBufferSize, cap(ch.incoming))
	}
	if ch.started {
		t.Error("expected started to be false")
	}
}

func TestInfo(t *testing.T) {
	ch := NewDiscordChannel("test-token")

	info := ch.Info()

	if info.Type != types.ChannelDiscord {
		t.Errorf("expected Type %q, got %q", types.ChannelDiscord, info.Type)
	}
	if info.Name != "discord" {
		t.Errorf("expected Name %q, got %q", "discord", info.Name)
	}
	if info.Connected {
		t.Error("expected Connected to be false before Start")
	}
}
