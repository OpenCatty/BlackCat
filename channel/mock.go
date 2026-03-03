package channel

import (
	"context"
	"sync"

	"github.com/startower-observability/blackcat/types"
)

// MockChannel implements types.Channel for testing purposes.
type MockChannel struct {
	info     types.ChannelInfo
	incoming chan types.Message // messages TO the mock (from Inject)
	outgoing []types.Message    // messages SENT by the agent (from Send)
	mu       sync.Mutex
	started  bool
}

// NewMockChannel creates a MockChannel with the given channel type.
func NewMockChannel(channelType types.ChannelType) *MockChannel {
	return &MockChannel{
		info: types.ChannelInfo{
			Type:      channelType,
			Name:      "mock-" + string(channelType),
			Connected: false,
		},
		incoming: make(chan types.Message, 100),
	}
}

// Start marks the mock channel as started.
func (m *MockChannel) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	m.info.Connected = true
	return nil
}

// Stop marks the mock channel as stopped and closes the incoming channel.
func (m *MockChannel) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = false
	m.info.Connected = false
	close(m.incoming)
	return nil
}

// Send appends a message to the outgoing slice (thread-safe).
func (m *MockChannel) Send(_ context.Context, msg types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outgoing = append(m.outgoing, msg)
	return nil
}

// Receive returns the incoming channel for reading.
func (m *MockChannel) Receive() <-chan types.Message {
	return m.incoming
}

// Info returns the channel info with Connected reflecting the started state.
func (m *MockChannel) Info() types.ChannelInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.info
}

// Health checks the health of the mock channel.
func (m *MockChannel) Health() types.ChannelHealth {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return types.ChannelHealth{
			Name:    m.info.Name,
			Healthy: false,
			Details: "mock channel not started",
		}
	}

	return types.ChannelHealth{
		Name:    m.info.Name,
		Healthy: true,
		Details: "mock channel is healthy",
	}
}

// Inject pushes a message into the incoming channel, simulating a user message.
func (m *MockChannel) Inject(msg types.Message) {
	m.incoming <- msg
}

// Sent returns a copy of all outgoing messages (thread-safe).
func (m *MockChannel) Sent() []types.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]types.Message, len(m.outgoing))
	copy(out, m.outgoing)
	return out
}

// Reset clears all outgoing messages.
func (m *MockChannel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outgoing = nil
}
