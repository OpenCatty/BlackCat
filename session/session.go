package session

import (
	"time"

	"github.com/startower-observability/blackcat/types"
)

// Session holds the conversation history and metadata for a single session.
type Session struct {
	Key          SessionKey         `json:"key"`
	Messages     []types.LLMMessage `json:"messages"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	LastActivity time.Time          `json:"lastActivity"`
	MessageCount int                `json:"messageCount"`
	ExpiresAt    time.Time          `json:"expiresAt,omitempty"`
	Metadata     map[string]string  `json:"metadata,omitempty"`
}
