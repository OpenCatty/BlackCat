package channel

import (
	"context"
	"testing"
	"time"

	"github.com/startower-observability/blackcat/types"
)

func TestNewMessageBus(t *testing.T) {
	bus := NewMessageBus(10)
	if bus == nil {
		t.Fatal("expected non-nil bus")
	}
	if bus.Messages() == nil {
		t.Fatal("expected non-nil Messages channel")
	}
}

func TestRegisterChannel(t *testing.T) {
	bus := NewMessageBus(10)
	mock := NewMockChannel(types.ChannelTelegram)

	if err := bus.Register(mock); err != nil {
		t.Fatalf("unexpected error registering channel: %v", err)
	}

	channels := bus.Channels()
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if channels[0].Type != types.ChannelTelegram {
		t.Fatalf("expected channel type %q, got %q", types.ChannelTelegram, channels[0].Type)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	bus := NewMessageBus(10)
	mock1 := NewMockChannel(types.ChannelTelegram)
	mock2 := NewMockChannel(types.ChannelTelegram)

	if err := bus.Register(mock1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bus.Register(mock2); err == nil {
		t.Fatal("expected error when registering duplicate channel type")
	}
}

func TestMessageBusFanIn(t *testing.T) {
	bus := NewMessageBus(10)
	mockTG := NewMockChannel(types.ChannelTelegram)
	mockDC := NewMockChannel(types.ChannelDiscord)

	if err := bus.Register(mockTG); err != nil {
		t.Fatalf("register telegram: %v", err)
	}
	if err := bus.Register(mockDC); err != nil {
		t.Fatalf("register discord: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bus.Start(ctx); err != nil {
		t.Fatalf("start bus: %v", err)
	}

	// Inject messages into both channels.
	mockTG.Inject(types.Message{ID: "tg-1", ChannelType: types.ChannelTelegram, Content: "hello from telegram"})
	mockDC.Inject(types.Message{ID: "dc-1", ChannelType: types.ChannelDiscord, Content: "hello from discord"})

	received := make(map[string]bool)
	timeout := time.After(2 * time.Second)

	for i := 0; i < 2; i++ {
		select {
		case msg := <-bus.Messages():
			received[msg.ID] = true
		case <-timeout:
			t.Fatalf("timed out waiting for messages, received %d/2", len(received))
		}
	}

	if !received["tg-1"] {
		t.Error("did not receive telegram message")
	}
	if !received["dc-1"] {
		t.Error("did not receive discord message")
	}

	cancel()
}

func TestMessageBusSend(t *testing.T) {
	bus := NewMessageBus(10)
	mock := NewMockChannel(types.ChannelTelegram)

	if err := bus.Register(mock); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bus.Start(ctx); err != nil {
		t.Fatalf("start bus: %v", err)
	}

	msg := types.Message{ID: "out-1", ChannelType: types.ChannelTelegram, Content: "reply"}
	if err := bus.Send(ctx, types.ChannelTelegram, msg); err != nil {
		t.Fatalf("send: %v", err)
	}

	sent := mock.Sent()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
	if sent[0].ID != "out-1" {
		t.Fatalf("expected message ID %q, got %q", "out-1", sent[0].ID)
	}

	cancel()
}

func TestMessageBusSendUnknown(t *testing.T) {
	bus := NewMessageBus(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msg := types.Message{ID: "out-1", Content: "test"}
	err := bus.Send(ctx, types.ChannelDiscord, msg)
	if err == nil {
		t.Fatal("expected error when sending to unregistered channel")
	}
}

func TestMockChannelInjectAndReceive(t *testing.T) {
	mock := NewMockChannel(types.ChannelTelegram)

	injected := types.Message{ID: "inject-1", ChannelType: types.ChannelTelegram, Content: "test message"}
	mock.Inject(injected)

	select {
	case msg := <-mock.Receive():
		if msg.ID != "inject-1" {
			t.Fatalf("expected ID %q, got %q", "inject-1", msg.ID)
		}
		if msg.Content != "test message" {
			t.Fatalf("expected content %q, got %q", "test message", msg.Content)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for injected message")
	}
}

func TestMockChannelSent(t *testing.T) {
	mock := NewMockChannel(types.ChannelTelegram)

	ctx := context.Background()

	msg1 := types.Message{ID: "s-1", Content: "first"}
	msg2 := types.Message{ID: "s-2", Content: "second"}

	if err := mock.Send(ctx, msg1); err != nil {
		t.Fatalf("send msg1: %v", err)
	}
	if err := mock.Send(ctx, msg2); err != nil {
		t.Fatalf("send msg2: %v", err)
	}

	sent := mock.Sent()
	if len(sent) != 2 {
		t.Fatalf("expected 2 sent messages, got %d", len(sent))
	}
	if sent[0].ID != "s-1" || sent[1].ID != "s-2" {
		t.Fatalf("unexpected sent messages: %v", sent)
	}

	// Verify Reset clears messages.
	mock.Reset()
	if len(mock.Sent()) != 0 {
		t.Fatal("expected 0 sent messages after reset")
	}
}
