package hooks

import "github.com/startower-observability/blackcat/types"

// HookContext carries event-specific data through hook handlers.
type HookContext struct {
	Event     HookEvent
	UserID    string
	ChannelID string
	Message   string
	Metadata  map[string]any

	IncomingMessage *types.Message
	LLMResponse     *types.LLMResponse
	ToolName        string
	FilePath        string
}
