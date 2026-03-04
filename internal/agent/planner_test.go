package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/startower-observability/blackcat/internal/types"
)

// plannerMockLLM is a mock LLM client for planner tests.
type plannerMockLLM struct {
	responses []*types.LLMResponse
	idx       int
	lastMsgs  []types.LLMMessage
}

func (m *plannerMockLLM) Chat(_ context.Context, msgs []types.LLMMessage, _ []types.ToolDefinition) (*types.LLMResponse, error) {
	m.lastMsgs = msgs
	if m.idx >= len(m.responses) {
		return &types.LLMResponse{Content: "done"}, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r, nil
}

func (m *plannerMockLLM) Stream(_ context.Context, _ []types.LLMMessage, _ []types.ToolDefinition) (<-chan types.Chunk, error) {
	ch := make(chan types.Chunk, 1)
	close(ch)
	return ch, nil
}

// --- IsComplexTask tests ---

func TestIsComplexTask_StepByStep(t *testing.T) {
	if !IsComplexTask("Can you do this step by step?") {
		t.Error("expected 'step by step' to be complex")
	}
}

func TestIsComplexTask_PlanKeyword(t *testing.T) {
	if !IsComplexTask("Please plan the deployment") {
		t.Error("expected 'plan' keyword to be complex")
	}
}

func TestIsComplexTask_SequenceKeywords(t *testing.T) {
	if !IsComplexTask("First create the file, then write the tests, after that deploy") {
		t.Error("expected 'first...then...after that' to be complex")
	}
}

func TestIsComplexTask_MultipleSentences(t *testing.T) {
	if !IsComplexTask("Create the database schema. Add the API endpoints. Write integration tests. Deploy to staging.") {
		t.Error("expected 3+ sentences to be complex")
	}
}

func TestIsComplexTask_SimpleMessage(t *testing.T) {
	if IsComplexTask("hello") {
		t.Error("expected simple greeting to NOT be complex")
	}
}

func TestIsComplexTask_SimpleQuestion(t *testing.T) {
	if IsComplexTask("What is Go?") {
		t.Error("expected simple question to NOT be complex")
	}
}

// --- Plan struct tests ---

func TestPlan_IsComplete_NilPlan(t *testing.T) {
	var p *Plan
	if !p.IsComplete() {
		t.Error("nil plan should be complete")
	}
}

func TestPlan_IsComplete_EmptySteps(t *testing.T) {
	p := &Plan{Steps: []PlanStep{}}
	if !p.IsComplete() {
		t.Error("empty steps should be complete")
	}
}

func TestPlan_IsComplete_AllDone(t *testing.T) {
	p := &Plan{Steps: []PlanStep{
		{Status: StepCompleted},
		{Status: StepSkipped},
		{Status: StepFailed},
	}}
	if !p.IsComplete() {
		t.Error("all terminal statuses should be complete")
	}
}

func TestPlan_IsComplete_HasPending(t *testing.T) {
	p := &Plan{Steps: []PlanStep{
		{Status: StepCompleted},
		{Status: StepPending},
	}}
	if p.IsComplete() {
		t.Error("pending step means not complete")
	}
}

func TestPlan_Summary(t *testing.T) {
	p := &Plan{
		Goal: "Deploy the app",
		Steps: []PlanStep{
			{Index: 0, Description: "Build project", Status: StepCompleted},
			{Index: 1, Description: "Run tests", ToolName: "exec", Status: StepInProgress},
			{Index: 2, Description: "Deploy", Status: StepPending},
		},
	}
	summary := p.Summary()
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if !contains(summary, "Deploy the app") {
		t.Error("summary missing goal")
	}
	if !contains(summary, "[+]") {
		t.Error("summary missing completed icon")
	}
	if !contains(summary, "[>]") {
		t.Error("summary missing in-progress icon")
	}
	if !contains(summary, "[ ]") {
		t.Error("summary missing pending icon")
	}
	if !contains(summary, "(tool: exec)") {
		t.Error("summary missing tool annotation")
	}
}

func TestPlan_Summary_Nil(t *testing.T) {
	var p *Plan
	if p.Summary() != "" {
		t.Error("nil plan summary should be empty")
	}
}

// --- parsePlanJSON tests ---

func TestParsePlanJSON_Valid(t *testing.T) {
	raw := `{"goal":"test","steps":[{"index":0,"description":"step one","status":"pending"}]}`
	plan, err := parsePlanJSON(raw)
	if err != nil {
		t.Fatalf("parsePlanJSON error: %v", err)
	}
	if plan.Goal != "test" {
		t.Errorf("goal = %q, want 'test'", plan.Goal)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("steps count = %d, want 1", len(plan.Steps))
	}
	if plan.Steps[0].Description != "step one" {
		t.Errorf("step description = %q, want 'step one'", plan.Steps[0].Description)
	}
}

func TestParsePlanJSON_WithCodeFence(t *testing.T) {
	raw := "```json\n{\"goal\":\"test\",\"steps\":[{\"description\":\"do it\",\"status\":\"pending\"}]}\n```"
	plan, err := parsePlanJSON(raw)
	if err != nil {
		t.Fatalf("parsePlanJSON with fence error: %v", err)
	}
	if plan.Goal != "test" {
		t.Errorf("goal = %q, want 'test'", plan.Goal)
	}
}

func TestParsePlanJSON_NoSteps(t *testing.T) {
	raw := `{"goal":"test","steps":[]}`
	_, err := parsePlanJSON(raw)
	if err == nil {
		t.Error("expected error for empty steps")
	}
}

func TestParsePlanJSON_InvalidJSON(t *testing.T) {
	_, err := parsePlanJSON("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePlanJSON_NormalizesIndices(t *testing.T) {
	raw := `{"goal":"g","steps":[{"index":99,"description":"a","status":""},{"index":5,"description":"b"}]}`
	plan, err := parsePlanJSON(raw)
	if err != nil {
		t.Fatalf("parsePlanJSON error: %v", err)
	}
	if plan.Steps[0].Index != 0 {
		t.Errorf("step 0 index = %d, want 0", plan.Steps[0].Index)
	}
	if plan.Steps[1].Index != 1 {
		t.Errorf("step 1 index = %d, want 1", plan.Steps[1].Index)
	}
	// Empty status should be normalized to pending
	if plan.Steps[0].Status != StepPending {
		t.Errorf("step 0 status = %q, want %q", plan.Steps[0].Status, StepPending)
	}
	if plan.Steps[1].Status != StepPending {
		t.Errorf("step 1 status = %q, want %q", plan.Steps[1].Status, StepPending)
	}
}

// --- GeneratePlan tests ---

func TestGeneratePlan_NilPlanner(t *testing.T) {
	var p *Planner
	plan, err := p.GeneratePlan(context.Background(), "do stuff", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != nil {
		t.Error("nil planner should return nil plan")
	}
}

func TestGeneratePlan_NilLLM(t *testing.T) {
	p := NewPlanner(nil)
	plan, err := p.GeneratePlan(context.Background(), "do stuff", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != nil {
		t.Error("nil LLM should return nil plan")
	}
}

func TestGeneratePlan_Success(t *testing.T) {
	planJSON := `{"goal":"test goal","steps":[{"description":"step one","status":"pending"},{"description":"step two","status":"pending"}]}`
	llm := &plannerMockLLM{responses: []*types.LLMResponse{
		{Content: planJSON},
	}}

	p := NewPlanner(llm)
	plan, err := p.GeneratePlan(context.Background(), "do something complex step by step", nil)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.Goal != "test goal" {
		t.Errorf("goal = %q, want 'test goal'", plan.Goal)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("steps = %d, want 2", len(plan.Steps))
	}
	if plan.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestGeneratePlan_InvalidJSON_GracefulDegradation(t *testing.T) {
	llm := &plannerMockLLM{responses: []*types.LLMResponse{
		{Content: "I can't generate a plan for that, sorry"},
	}}

	p := NewPlanner(llm)
	plan, err := p.GeneratePlan(context.Background(), "do stuff", nil)
	if err != nil {
		t.Fatalf("expected nil error for graceful degradation, got: %v", err)
	}
	if plan != nil {
		t.Error("expected nil plan for graceful degradation")
	}
}

func TestGeneratePlan_WithTools(t *testing.T) {
	planJSON := `{"goal":"use tools","steps":[{"description":"run it","tool_name":"exec","tool_args":"ls","status":"pending"}]}`
	llm := &plannerMockLLM{responses: []*types.LLMResponse{
		{Content: planJSON},
	}}

	tools := []types.ToolDefinition{
		{Name: "exec", Description: "Execute commands"},
		{Name: "web_search", Description: "Search the web"},
	}

	p := NewPlanner(llm)
	plan, err := p.GeneratePlan(context.Background(), "do complex thing", tools)
	if err != nil {
		t.Fatalf("GeneratePlan error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	// Verify tool info was passed to LLM
	if len(llm.lastMsgs) < 1 {
		t.Fatal("expected at least 1 message to LLM")
	}
	sysMsg := llm.lastMsgs[0].Content
	if !contains(sysMsg, "exec") {
		t.Error("system message should contain tool name 'exec'")
	}
}

// --- Replan tests ---

func TestReplan_NilPlanner(t *testing.T) {
	var p *Planner
	plan, err := p.Replan(context.Background(), &Plan{}, PlanStep{}, "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != nil {
		t.Error("nil planner should return nil plan")
	}
}

func TestReplan_MaxAttemptsReached(t *testing.T) {
	llm := &plannerMockLLM{responses: []*types.LLMResponse{
		{Content: `{"goal":"x","steps":[{"description":"y","status":"pending"}]}`},
	}}
	p := NewPlanner(llm)
	original := &Plan{
		Goal:        "original",
		ReplanCount: MaxReplanAttempts, // already at max
		Steps:       []PlanStep{{Description: "a", Status: StepFailed}},
	}
	plan, err := p.Replan(context.Background(), original, PlanStep{Description: "a", Error: "fail"}, "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan != nil {
		t.Error("should return nil when max replans reached")
	}
}

func TestReplan_Success(t *testing.T) {
	replanJSON := `{"goal":"revised goal","steps":[{"description":"new step","status":"pending"}]}`
	llm := &plannerMockLLM{responses: []*types.LLMResponse{
		{Content: replanJSON},
	}}
	p := NewPlanner(llm)
	original := &Plan{
		Goal:        "original goal",
		ReplanCount: 0,
		Steps: []PlanStep{
			{Index: 0, Description: "done step", Status: StepCompleted},
			{Index: 1, Description: "failed step", Status: StepFailed, Error: "something broke"},
		},
	}
	failed := original.Steps[1]

	newPlan, err := p.Replan(context.Background(), original, failed, "do the thing")
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
	if newPlan == nil {
		t.Fatal("expected non-nil replan")
	}
	if newPlan.ReplanCount != 1 {
		t.Errorf("replan count = %d, want 1", newPlan.ReplanCount)
	}
	if newPlan.Goal != "revised goal" {
		t.Errorf("goal = %q, want 'revised goal'", newPlan.Goal)
	}
}

// --- helper ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	// Use json package just to avoid importing strings in test
	_ = json.Marshal // keep json import used
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- ShouldPlan tests (Wave 4 T4) ---

func TestShouldPlan_ShortMessage_False(t *testing.T) {
	p := NewPlanner(nil)
	if p.ShouldPlan("fix bug") {
		t.Fatal("expected ShouldPlan to return false for short message")
	}
}

func TestShouldPlan_MultiSentence_True(t *testing.T) {
	p := NewPlanner(nil)
	msg := "First do A. Then do B. Finally do C."
	if !p.ShouldPlan(msg) {
		t.Fatal("expected ShouldPlan to return true for 3+ sentence message")
	}
}

func TestShouldPlan_Keyword_True(t *testing.T) {
	p := NewPlanner(nil)
	if !p.ShouldPlan("step by step guide") {
		t.Fatal("expected ShouldPlan to return true for keyword match")
	}
}

func TestShouldPlan_NilPlanner_False(t *testing.T) {
	var p *Planner
	if p.ShouldPlan("step by step multi plan") {
		t.Fatal("expected ShouldPlan to return false on nil planner")
	}
}

// --- FormatPlanForPrompt tests (Wave 4 T4) ---

func TestFormatPlanForPrompt_NilPlan_Empty(t *testing.T) {
	result := FormatPlanForPrompt(nil)
	if result != "" {
		t.Fatalf("expected empty string for nil plan, got %q", result)
	}
}

func TestFormatPlanForPrompt_WithSteps(t *testing.T) {
	plan := &Plan{
		Goal: "deploy the app",
		Steps: []PlanStep{
			{Index: 0, Description: "build binary", Status: StepPending},
			{Index: 1, Description: "run tests", ToolName: "exec", Status: StepPending},
		},
	}
	result := FormatPlanForPrompt(plan)
	if !strings.Contains(result, "### Execution Plan") {
		t.Fatal("missing plan header")
	}
	if !strings.Contains(result, "Goal: deploy the app") {
		t.Fatal("missing goal")
	}
	if !strings.Contains(result, "1. build binary") {
		t.Fatal("missing step 1")
	}
	if !strings.Contains(result, "2. run tests") {
		t.Fatal("missing step 2")
	}
	if !strings.Contains(result, "(tool: exec)") {
		t.Fatal("missing tool annotation")
	}
}

func TestFormatPlanForPrompt_EmptySteps_Empty(t *testing.T) {
	plan := &Plan{Goal: "something", Steps: []PlanStep{}}
	result := FormatPlanForPrompt(plan)
	if result != "" {
		t.Fatalf("expected empty string for plan with no steps, got %q", result)
	}
}