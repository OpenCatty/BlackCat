package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/tools"
	"github.com/startower-observability/blackcat/internal/types"
)

// supervisorMockLLM is a mock LLM client for supervisor tests.
type supervisorMockLLM struct {
	responses []*types.LLMResponse
	idx       int
}

func (m *supervisorMockLLM) Chat(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	if m.idx >= len(m.responses) {
		return &types.LLMResponse{Content: "done"}, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r, nil
}

func (m *supervisorMockLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk, 1)
	close(ch)
	return ch, nil
}

// supervisorMockTool is a mock tool for supervisor tests.
type supervisorMockTool struct {
	name   string
	result string
	params json.RawMessage
}

func (m *supervisorMockTool) Name() string                { return m.name }
func (m *supervisorMockTool) Description() string         { return "supervisor mock tool" }
func (m *supervisorMockTool) Parameters() json.RawMessage { return m.params }
func (m *supervisorMockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return m.result, nil
}

func TestClassifyMessage_Coding(t *testing.T) {
	tests := []struct {
		msg  string
		want TaskType
	}{
		{"fix the bug in my code", TaskTypeCoding},
		{"implement a new function", TaskTypeCoding},
		{"write some tests for the module", TaskTypeCoding},
		{"help me build this project", TaskTypeCoding},
		{"deploy the application", TaskTypeCoding},
	}
	for _, tc := range tests {
		got := ClassifyMessage(tc.msg)
		if got != tc.want {
			t.Errorf("ClassifyMessage(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestClassifyMessage_Research(t *testing.T) {
	tests := []struct {
		msg  string
		want TaskType
	}{
		{"search for recent Go news", TaskTypeResearch},
		{"what is the meaning of life", TaskTypeResearch},
		{"explain how goroutines work", TaskTypeResearch},
		{"summarize that article", TaskTypeResearch},
		{"browse the web for info", TaskTypeResearch},
	}
	for _, tc := range tests {
		got := ClassifyMessage(tc.msg)
		if got != tc.want {
			t.Errorf("ClassifyMessage(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestClassifyMessage_Admin(t *testing.T) {
	tests := []struct {
		msg  string
		want TaskType
	}{
		{"restart the service", TaskTypeAdmin},
		{"check the health endpoint", TaskTypeAdmin},
		{"update the config file", TaskTypeAdmin},
		{"check blackcat status", TaskTypeAdmin},
		{"stop the server now", TaskTypeAdmin},
	}
	for _, tc := range tests {
		got := ClassifyMessage(tc.msg)
		if got != tc.want {
			t.Errorf("ClassifyMessage(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestClassifyMessage_General(t *testing.T) {
	tests := []struct {
		msg  string
		want TaskType
	}{
		{"hello how are you", TaskTypeGeneral},
		{"tell me a joke", TaskTypeGeneral},
		{"good morning", TaskTypeGeneral},
	}
	for _, tc := range tests {
		got := ClassifyMessage(tc.msg)
		if got != tc.want {
			t.Errorf("ClassifyMessage(%q) = %q, want %q", tc.msg, got, tc.want)
		}
	}
}

func TestClassifyMessage_AdminPriorityOverCoding(t *testing.T) {
	// "restart" is admin, "build" is coding — admin should win.
	got := ClassifyMessage("restart the build server")
	if got != TaskTypeAdmin {
		t.Errorf("ClassifyMessage(admin+coding overlap) = %q, want %q", got, TaskTypeAdmin)
	}
}

func TestClassifyMessage_CodingPriorityOverResearch(t *testing.T) {
	// "test" is coding, "search" is research — coding should win.
	got := ClassifyMessage("search and test the code")
	if got != TaskTypeCoding {
		t.Errorf("ClassifyMessage(coding+research overlap) = %q, want %q", got, TaskTypeCoding)
	}
}

func TestSupervisor_NilReturnsError(t *testing.T) {
	var s *Supervisor
	_, err := s.RouteWithCfg(context.Background(), "hello", LoopConfig{})
	if err == nil {
		t.Fatal("expected error from nil supervisor, got nil")
	}
	if err.Error() != "supervisor not initialized" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestSupervisor_RouteWithCfg_CodingMessage(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&supervisorMockTool{
		name:   "echo",
		result: "ok",
		params: json.RawMessage(`{"type":"object"}`),
	})

	llm := &supervisorMockLLM{responses: []*types.LLMResponse{
		{Content: "I fixed the bug"},
	}}

	baseCfg := LoopConfig{
		LLM:       llm,
		Tools:     registry,
		Scrubber:  security.NewScrubber(),
		MaxTurns:  10,
		AgentName: "TestBot",
	}

	supervisor := NewSupervisor(baseCfg)

	execution, err := supervisor.RouteWithCfg(context.Background(), "fix the bug in my code", baseCfg)
	if err != nil {
		t.Fatalf("RouteWithCfg() error = %v", err)
	}
	if execution.NextStep != FinalOutput {
		t.Errorf("expected FinalOutput, got %v", execution.NextStep)
	}
	if execution.Response != "I fixed the bug" {
		t.Errorf("expected response 'I fixed the bug', got %q", execution.Response)
	}
}

func TestSupervisor_Route_UsesBaseConfig(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&supervisorMockTool{
		name:   "echo",
		result: "ok",
		params: json.RawMessage(`{"type":"object"}`),
	})

	llm := &supervisorMockLLM{responses: []*types.LLMResponse{
		{Content: "hello there"},
	}}

	baseCfg := LoopConfig{
		LLM:       llm,
		Tools:     registry,
		Scrubber:  security.NewScrubber(),
		MaxTurns:  10,
		AgentName: "TestBot",
	}

	supervisor := NewSupervisor(baseCfg)

	execution, err := supervisor.Route(context.Background(), "hello how are you")
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if execution.NextStep != FinalOutput {
		t.Errorf("expected FinalOutput, got %v", execution.NextStep)
	}
}

func TestNewSupervisor_DefaultConfigs(t *testing.T) {
	supervisor := NewSupervisor(LoopConfig{})

	// Verify all four task types have configs
	for _, tt := range []TaskType{TaskTypeCoding, TaskTypeResearch, TaskTypeAdmin, TaskTypeGeneral} {
		if _, ok := supervisor.subAgentConfigs[tt]; !ok {
			t.Errorf("missing sub-agent config for task type %q", tt)
		}
	}

	// Research should have restricted tools
	researchCfg := supervisor.subAgentConfigs[TaskTypeResearch]
	if researchCfg.AllowedTools == nil {
		t.Error("research sub-agent should have AllowedTools set")
	}
	if len(researchCfg.AllowedTools) != 5 {
		t.Errorf("research AllowedTools count = %d, want 5", len(researchCfg.AllowedTools))
	}

	// Coding should have nil AllowedTools (all tools)
	codingCfg := supervisor.subAgentConfigs[TaskTypeCoding]
	if codingCfg.AllowedTools != nil {
		t.Error("coding sub-agent should have nil AllowedTools (all tools)")
	}

	// General should have empty overlay
	generalCfg := supervisor.subAgentConfigs[TaskTypeGeneral]
	if generalCfg.SystemPromptOverlay != "" {
		t.Errorf("general overlay should be empty, got %q", generalCfg.SystemPromptOverlay)
	}
}
