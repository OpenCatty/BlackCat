package agent

import (
	"context"
	"testing"

	"github.com/startower-observability/blackcat/internal/security"
	"github.com/startower-observability/blackcat/internal/types"
)

// wave4MockLLM is a mock LLM client for Wave 4 integration tests.
type wave4MockLLM struct {
	responses []*types.LLMResponse
	idx       int
}

func (m *wave4MockLLM) Chat(ctx context.Context, msgs []types.LLMMessage, toolDefs []types.ToolDefinition) (*types.LLMResponse, error) {
	if m.idx >= len(m.responses) {
		return &types.LLMResponse{Content: "done"}, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r, nil
}

func (m *wave4MockLLM) Stream(ctx context.Context, msgs []types.LLMMessage, toolDefs []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk, 1)
	close(ch)
	return ch, nil
}

// TestWave4_Reflection_NilLLM_NoOp verifies that when the Reflector has a nil
// LLM, the agent loop runs without panic and produces final output normally.
func TestWave4_Reflection_NilLLM_NoOp(t *testing.T) {
	ctx := context.Background()

	llm := &wave4MockLLM{responses: []*types.LLMResponse{
		{Content: "reflection test done"},
	}}

	// Reflector with nil LLM — should no-op gracefully
	reflector := NewReflector(nil, nil)

	loop := NewLoop(LoopConfig{
		LLM:       llm,
		Scrubber:  security.NewScrubber(),
		MaxTurns:  10,
		Reflector: reflector,
	})

	execution, err := loop.Run(ctx, "hello with reflection")
	if err != nil {
		t.Fatalf("loop.Run() error = %v", err)
	}
	if execution.NextStep != FinalOutput {
		t.Fatalf("NextStep = %d, want FinalOutput (%d)", execution.NextStep, FinalOutput)
	}
	if execution.Response != "reflection test done" {
		t.Fatalf("Response = %q, want %q", execution.Response, "reflection test done")
	}
}

// TestWave4_Supervisor_ClassifiesCoding verifies that ClassifyMessage correctly
// classifies a coding-related message as TaskTypeCoding.
func TestWave4_Supervisor_ClassifiesCoding(t *testing.T) {
	got := ClassifyMessage("fix the bug in my code")
	if got != TaskTypeCoding {
		t.Fatalf("ClassifyMessage('fix the bug in my code') = %q, want %q", got, TaskTypeCoding)
	}
}

// TestWave4_Supervisor_ClassifiesAdmin verifies that ClassifyMessage correctly
// classifies an admin-related message as TaskTypeAdmin.
func TestWave4_Supervisor_ClassifiesAdmin(t *testing.T) {
	got := ClassifyMessage("restart the service")
	if got != TaskTypeAdmin {
		t.Fatalf("ClassifyMessage('restart the service') = %q, want %q", got, TaskTypeAdmin)
	}
}

// TestWave4_Planner_ShouldPlan_MultiStep verifies that Planner.ShouldPlan
// returns true for a multi-step message containing sequential keywords.
func TestWave4_Planner_ShouldPlan_MultiStep(t *testing.T) {
	planner := NewPlanner(nil) // nil LLM is fine — ShouldPlan doesn't call LLM
	got := planner.ShouldPlan("First do A. Then do B. Finally do C.")
	if !got {
		t.Fatal("ShouldPlan('First do A. Then do B. Finally do C.') = false, want true")
	}
}

// TestWave4_AdaptivePrefs_NilManager_NoOp verifies that NewPreferenceManager(nil)
// followed by LoadPreferences returns a default AdaptiveProfile with all fields
// set to "auto", without panicking.
func TestWave4_AdaptivePrefs_NilManager_NoOp(t *testing.T) {
	ctx := context.Background()
	pm := NewPreferenceManager(nil)

	profile := pm.LoadPreferences(ctx, "user-123")

	if profile.Language != "auto" {
		t.Fatalf("Language = %q, want %q", profile.Language, "auto")
	}
	if profile.Style != "auto" {
		t.Fatalf("Style = %q, want %q", profile.Style, "auto")
	}
	if profile.Verbosity != "auto" {
		t.Fatalf("Verbosity = %q, want %q", profile.Verbosity, "auto")
	}
	if profile.TechnicalDepth != "auto" {
		t.Fatalf("TechnicalDepth = %q, want %q", profile.TechnicalDepth, "auto")
	}
}
