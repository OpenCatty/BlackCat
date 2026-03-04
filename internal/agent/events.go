package agent

import "time"

// EventKind identifies what kind of agent event occurred.
type EventKind string

const (
	EventThinking        EventKind = "thinking"         // LLM is being called
	EventToolCallStart   EventKind = "tool_call_start"  // Tool is about to execute
	EventToolCallResult  EventKind = "tool_call_result" // Tool finished executing
	EventPartialResponse EventKind = "partial_response" // Intermediate text output (future streaming)
	EventDone            EventKind = "done"             // Loop finished successfully
	EventError           EventKind = "error"            // Loop finished with error
	EventInterrupted     EventKind = "interrupted"      // Loop paused for HITL approval
	EventHandoff         EventKind = "handoff"          // Loop delegating to sub-agent
)

// AgentEvent is emitted by the agent loop during execution.
type AgentEvent struct {
	Kind      EventKind `json:"kind"`
	TurnNum   int       `json:"turn_num"`
	ToolName  string    `json:"tool_name,omitempty"` // set for tool_call_start / tool_call_result
	ToolArgs  string    `json:"tool_args,omitempty"` // set for tool_call_start
	Result    string    `json:"result,omitempty"`    // set for tool_call_result / partial_response
	Message   string    `json:"message,omitempty"`   // human-readable status e.g. "Calling exec_command..."
	Error     string    `json:"error,omitempty"`     // set for EventError
	Timestamp time.Time `json:"timestamp"`
}

// EventStream is a send-only channel for agent events.
// Callers that don't want events pass nil — the emit helper checks for nil.
type EventStream = chan<- AgentEvent

// emit sends an event on the stream without blocking the agent loop.
func (l *Loop) emit(stream chan<- AgentEvent, ev AgentEvent) {
	if stream == nil {
		return
	}
	ev.Timestamp = time.Now()
	select {
	case stream <- ev:
	default: // never block the agent loop
	}
}

// truncate cuts a string at n characters and appends "..." if it was longer.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
