package agent

import "github.com/startower-observability/blackcat/internal/types"

type Execution struct {
	Messages        []types.LLMMessage
	ToolOutputs     map[string]string
	ToolMappings    map[string]types.Tool
	PendingApproval *PendingApproval
	TotalUsage      types.LLMUsage
	Compacted       bool
	NextStep        NextStep
	Done            bool
	TurnCount       int
	MaxTurns        int
	Response        string
	Error           error
	ToolRetryCount  map[string]int
}

func NewExecution(maxTurns int) *Execution {
	if maxTurns <= 0 {
		maxTurns = 50
	}

	return &Execution{
		Messages:     make([]types.LLMMessage, 0, 8),
		ToolOutputs:  make(map[string]string),
		ToolMappings:   make(map[string]types.Tool),
		ToolRetryCount: make(map[string]int),
		MaxTurns:       maxTurns,
	}
}

func (e *Execution) AddUserMessage(content string) {
	e.Messages = append(e.Messages, types.LLMMessage{Role: "user", Content: content})
}

func (e *Execution) AddAssistantMessage(content string, toolCalls []types.ToolCall) {
	e.Messages = append(e.Messages, types.LLMMessage{Role: "assistant", Content: content, ToolCalls: toolCalls})
}

func (e *Execution) AddToolResult(callID, name, result string) {
	e.ToolOutputs[callID] = result
	e.Messages = append(e.Messages, types.LLMMessage{
		Role:       "tool",
		Content:    result,
		ToolCallID: callID,
		Name:       name,
	})
}

func (e *Execution) AddSystemMessage(content string) {
	e.Messages = append(e.Messages, types.LLMMessage{Role: "system", Content: content})
}
