package channel

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/startower-observability/blackcat/types"
)

// MessageBus fans-in messages from all registered channels into a single
// consumer channel and routes outgoing messages to the correct channel.
type MessageBus struct {
	incoming chan types.Message
	channels map[types.ChannelType]types.Channel
	mu       sync.RWMutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

// NewMessageBus creates a MessageBus with a buffered incoming channel.
func NewMessageBus(bufferSize int) *MessageBus {
	return &MessageBus{
		incoming: make(chan types.Message, bufferSize),
		channels: make(map[types.ChannelType]types.Channel),
	}
}

// Register adds a channel to the bus. Returns an error if a channel with the
// same ChannelType is already registered.
func (b *MessageBus) Register(ch types.Channel) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	info := ch.Info()
	if _, exists := b.channels[info.Type]; exists {
		return fmt.Errorf("channel type %q already registered", info.Type)
	}
	b.channels[info.Type] = ch
	slog.Info("channel registered", "type", info.Type, "name", info.Name)
	return nil
}

// Start starts all registered channels and launches fan-in goroutines that
// forward messages from each channel's Receive() into the single incoming
// channel. When ctx is cancelled, all channels are stopped.
func (b *MessageBus) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	ctx, b.cancel = context.WithCancel(ctx)

	for ct, ch := range b.channels {
		if err := ch.Start(ctx); err != nil {
			return fmt.Errorf("failed to start channel %q: %w", ct, err)
		}
		slog.Info("channel started", "type", ct)

		// Fan-in goroutine for this channel.
		b.wg.Add(1)
		go func(recv <-chan types.Message) {
			defer b.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-recv:
					if !ok {
						return
					}
					select {
					case b.incoming <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}(ch.Receive())
	}

	slog.Info("message bus started", "channels", len(b.channels))
	return nil
}

// Stop cancels the bus context, waits for fan-in goroutines to exit, stops all
// channels, and closes the incoming channel.
func (b *MessageBus) Stop() error {
	if b.cancel != nil {
		b.cancel()
	}
	b.wg.Wait()

	b.mu.RLock()
	defer b.mu.RUnlock()
	var firstErr error
	for ct, ch := range b.channels {
		if err := ch.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to stop channel %q: %w", ct, err)
		}
	}
	close(b.incoming)
	slog.Info("message bus stopped")
	return firstErr
}

// Messages returns the read-only incoming channel for the consumer.
func (b *MessageBus) Messages() <-chan types.Message {
	return b.incoming
}

// Send routes an outgoing message to the channel identified by channelType.
// Returns an error if no channel of that type is registered.
func (b *MessageBus) Send(ctx context.Context, channelType types.ChannelType, msg types.Message) error {
	b.mu.RLock()
	ch, ok := b.channels[channelType]
	b.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no channel registered for type %q", channelType)
	}
	return ch.Send(ctx, msg)
}

// Channels returns info from all registered channels.
func (b *MessageBus) Channels() []types.ChannelInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	infos := make([]types.ChannelInfo, 0, len(b.channels))
	for _, ch := range b.channels {
		infos = append(infos, ch.Info())
	}
	return infos
}

// GetChannel returns the channel for the given type, or nil if not found.
func (b *MessageBus) GetChannel(channelType types.ChannelType) types.Channel {
	b.mu.RLock()
	ch, ok := b.channels[channelType]
	b.mu.RUnlock()
	if !ok {
		return nil
	}
	return ch
}
