package types

import (
	"encoding/json"
	"time"
)

// ChannelType identifies the messaging platform.
type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelDiscord  ChannelType = "discord"
	ChannelWhatsApp ChannelType = "whatsapp"
)

// Message represents a message from any channel.
type Message struct {
	ID          string            `json:"id"`
	ChannelType ChannelType       `json:"channelType"`
	ChannelID   string            `json:"channelID"`
	UserID      string            `json:"userID"`
	Content     string            `json:"content"`
	Timestamp   time.Time         `json:"timestamp"`
	ReplyTo     string            `json:"replyTo,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	// Media fields — populated by channel adapters when a voice/audio message is received
	MediaType string `json:"media_type,omitempty"` // "voice", "audio", "video_note"
	MediaURL  string `json:"media_url,omitempty"`  // direct download URL
	MediaSize int64  `json:"media_size,omitempty"` // file size in bytes
}

// ToolCall represents an LLM-requested tool invocation.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Result    string          `json:"result,omitempty"`
}

// ToolDefinition describes a tool for the LLM.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Provider represents an LLM provider configuration.
type Provider struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	APIKey  string `json:"-"` // never serialize
	BaseURL string `json:"baseURL,omitempty"`
}

// SessionState is the lifecycle state of an agent session.
type SessionState string

const (
	SessionIdle       SessionState = "idle"
	SessionRunning    SessionState = "running"
	SessionCompacting SessionState = "compacting"
	SessionCompleted  SessionState = "completed"
	SessionFailed     SessionState = "failed"
)

// ChannelInfo describes a connected channel.
type ChannelInfo struct {
	Type      ChannelType `json:"type"`
	Name      string      `json:"name"`
	Connected bool        `json:"connected"`
}

// ChannelHealth describes the health status of a channel.
type ChannelHealth struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Details string `json:"details"`
}

// Entry is a single memory entry.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
}

// LLMResponse is a response from an LLM.
type LLMResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	Model     string     `json:"model"`
	Usage     LLMUsage   `json:"usage"`
}

// LLMUsage tracks token consumption.
type LLMUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// LLMMessage is a message in an LLM conversation.
type LLMMessage struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string     `json:"toolCallID,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// Chunk is a streaming chunk from the LLM.
type Chunk struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	Done      bool       `json:"done"`
}
