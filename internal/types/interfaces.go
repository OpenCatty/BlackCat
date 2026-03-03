package types

import (
	"context"
	"encoding/json"
)

// Channel is the interface for messaging platform adapters.
type Channel interface {
	Start(ctx context.Context) error
	Stop() error
	Send(ctx context.Context, msg Message) error
	Receive() <-chan Message
	Info() ChannelInfo
	Health() ChannelHealth
}

// Tool is the interface for agent tools.
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// MemoryStore is the interface for persistent memory.
type MemoryStore interface {
	Read(ctx context.Context) ([]Entry, error)
	Write(ctx context.Context, entry Entry) error
	Search(ctx context.Context, query string) ([]Entry, error)
	Count() (int, error)
	Consolidate(ctx context.Context) error
}

// LLMClient is the interface for LLM providers.
type LLMClient interface {
	Chat(ctx context.Context, messages []LLMMessage, tools []ToolDefinition) (*LLMResponse, error)
	Stream(ctx context.Context, messages []LLMMessage, tools []ToolDefinition) (<-chan Chunk, error)
}

// Reconnectable is an optional interface for channels that support reconnection.
// Channels implement this to allow the heartbeat system to auto-reconnect
// when they become unhealthy. Use type assertion to check:
//
//	if r, ok := ch.(types.Reconnectable); ok { r.Reconnect(ctx) }
type Reconnectable interface {
	Reconnect(ctx context.Context) error
}
