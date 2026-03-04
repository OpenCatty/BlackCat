package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	guardrailsPkg "github.com/startower-observability/blackcat/internal/guardrails"
	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/tools"
	"github.com/startower-observability/blackcat/internal/types"
)

// wave2MockLLM is a mock LLM client for Wave 2 integration tests.
type wave2MockLLM struct {
	responses []*types.LLMResponse
	idx       int
}

func (m *wave2MockLLM) Chat(ctx context.Context, msgs []types.LLMMessage, toolDefs []types.ToolDefinition) (*types.LLMResponse, error) {
	if m.idx >= len(m.responses) {
		return &types.LLMResponse{Content: "done"}, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r, nil
}

func (m *wave2MockLLM) Stream(ctx context.Context, msgs []types.LLMMessage, toolDefs []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk, 1)
	close(ch)
	return ch, nil
}

// wave2MockTool is a mock tool for Wave 2 integration tests.
type wave2MockTool struct {
	name      string
	result    string
	params    json.RawMessage
	callCount int
}

func (m *wave2MockTool) Name() string                { return m.name }
func (m *wave2MockTool) Description() string         { return "wave2 mock tool" }
func (m *wave2MockTool) Parameters() json.RawMessage { return m.params }
func (m *wave2MockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	m.callCount++
	return m.result, nil
}

// TestWave2_LoopNoGuardrails_RunsNormally verifies that an agent loop with
// nil guardrails runs to completion and returns FinalOutput.
func TestWave2_LoopNoGuardrails_RunsNormally(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	registry.Register(&wave2MockTool{
		name:   "echo_tool",
		result: "echoed",
		params: json.RawMessage(`{"type":"object"}`),
	})

	llm := &wave2MockLLM{responses: []*types.LLMResponse{
		{Content: "all good, no tools needed"},
	}}

	loop := NewLoop(LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		MaxTurns: 10,
		// Guardrails intentionally nil
	})

	execution, err := loop.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}
	if execution.NextStep != FinalOutput {
		t.Fatalf("NextStep = %d, want FinalOutput (%d)", execution.NextStep, FinalOutput)
	}
	if execution.Response == "" {
		t.Fatal("Response is empty, want non-empty")
	}
	if execution.Response != "all good, no tools needed" {
		t.Fatalf("Response = %q, want %q", execution.Response, "all good, no tools needed")
	}
}

// TestWave2_LoopGuardrailsBlocksTool verifies that when guardrails block a
// tool call, the loop returns Interrupted with PendingApproval set.
func TestWave2_LoopGuardrailsBlocksTool(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	registry.Register(&wave2MockTool{
		name:   "rm_tool",
		result: "deleted",
		params: json.RawMessage(`{"type":"object"}`),
	})

	llm := &wave2MockLLM{responses: []*types.LLMResponse{
		{ToolCalls: []types.ToolCall{{ID: "call-1", Name: "rm_tool", Arguments: json.RawMessage(`{"path":"/tmp"}`)}}},
	}}

	pipeline := guardrailsPkg.NewPipeline(guardrailsPkg.GuardrailsConfig{
		ToolEnabled:             true,
		RequireApprovalPatterns: []string{"rm_tool"},
	})

	loop := NewLoop(LoopConfig{
		LLM:        llm,
		Tools:      registry,
		Scrubber:   security.NewScrubber(),
		MaxTurns:   10,
		Guardrails: pipeline,
		UserID:     "test-user",
	})

	execution, err := loop.Run(ctx, "delete something")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}
	if execution.NextStep != Interrupted {
		t.Fatalf("NextStep = %d, want Interrupted (%d)", execution.NextStep, Interrupted)
	}
	if execution.PendingApproval == nil {
		t.Fatal("PendingApproval is nil, want non-nil")
	}
	if execution.PendingApproval.ToolName != "rm_tool" {
		t.Fatalf("PendingApproval.ToolName = %q, want %q", execution.PendingApproval.ToolName, "rm_tool")
	}
	if execution.PendingApproval.UserID != "test-user" {
		t.Fatalf("PendingApproval.UserID = %q, want %q", execution.PendingApproval.UserID, "test-user")
	}
}

// TestWave2_ValidationErrorRetry verifies that when a tool call fails schema
// validation, the loop retries and eventually returns Error with "exceeded max retries".
func TestWave2_ValidationErrorRetry(t *testing.T) {
	ctx := context.Background()
	registry := tools.NewRegistry()
	registry.Register(&wave2MockTool{
		name:   "strict_tool",
		result: "ok",
		params: json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}},"required":["x"]}`),
	})

	// LLM keeps sending bad args (missing required field "x") on every call.
	badCall := &types.LLMResponse{
		ToolCalls: []types.ToolCall{{ID: "call-1", Name: "strict_tool", Arguments: json.RawMessage(`{}`)}},
	}
	llm := &wave2MockLLM{responses: []*types.LLMResponse{
		badCall, // First attempt — validation error, retry allowed
		badCall, // Second attempt — validation error, exceeds max retries
	}}

	loop := NewLoop(LoopConfig{
		LLM:      llm,
		Tools:    registry,
		Scrubber: security.NewScrubber(),
		MaxTurns: 10,
	})

	execution, err := loop.Run(ctx, "use strict tool")
	if err == nil {
		t.Fatal("loop.Run() error = nil, want non-nil")
	}
	if execution.NextStep != Error {
		t.Fatalf("NextStep = %d, want Error (%d)", execution.NextStep, Error)
	}
	if execution.Error == nil {
		t.Fatal("execution.Error is nil, want non-nil")
	}
	if !strings.Contains(execution.Error.Error(), "exceeded max retries") {
		t.Fatalf("execution.Error = %q, want substring %q", execution.Error.Error(), "exceeded max retries")
	}
}

// TestWave2_InterruptManagerApproveReject verifies the InterruptManager's
// CreateApproval and HandleReply flow for approve, reject, and not-found cases.
func TestWave2_InterruptManagerApproveReject(t *testing.T) {
	mgr := NewInterruptManager()

	// Test 1: Approve flow
	mgr.CreateApproval("user1", "rm_tool", `{"path":"/tmp"}`, "dangerous", 5*time.Minute)

	approved, found := mgr.HandleReply("user1", "yes")
	if !found {
		t.Fatal("HandleReply('yes') found = false, want true")
	}
	if !approved {
		t.Fatal("HandleReply('yes') approved = false, want true")
	}

	// After approval, pending should be cleared
	if pa := mgr.GetPending("user1"); pa != nil {
		t.Fatalf("GetPending after approve = %+v, want nil", pa)
	}

	// Test 2: Reject flow
	mgr.CreateApproval("user1", "drop_db", `{}`, "dangerous operation", 5*time.Minute)

	approved, found = mgr.HandleReply("user1", "no")
	if !found {
		t.Fatal("HandleReply('no') found = false, want true")
	}
	if approved {
		t.Fatal("HandleReply('no') approved = true, want false")
	}

	// After rejection, pending should be cleared
	if pa := mgr.GetPending("user1"); pa != nil {
		t.Fatalf("GetPending after reject = %+v, want nil", pa)
	}

	// Test 3: No pending approval — HandleReply returns not found
	approved, found = mgr.HandleReply("user1", "random")
	if found {
		t.Fatal("HandleReply with no pending: found = true, want false")
	}
	if approved {
		t.Fatal("HandleReply with no pending: approved = true, want false")
	}
}
