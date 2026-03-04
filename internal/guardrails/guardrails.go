package guardrails

// GuardrailResult is the result of a guardrail check.
type GuardrailResult struct {
	Allow    bool
	Reason   string
	Modified *string
}

// GuardrailsConfig configures all three guardrail stages.
type GuardrailsConfig struct {
	InputEnabled            bool
	ToolEnabled             bool
	OutputEnabled           bool
	CustomInputPatterns     []string
	RequireApprovalPatterns []string
}

// Pipeline runs all three guardrail stages.
type Pipeline struct {
	input  *InputGuardrail
	tool   *ToolGuardrail
	output *OutputGuardrail
}

// NewPipeline creates a guardrails pipeline from config.
func NewPipeline(cfg GuardrailsConfig) *Pipeline {
	return &Pipeline{
		input:  NewInputGuardrail(cfg.InputEnabled, cfg.CustomInputPatterns),
		tool:   NewToolGuardrail(cfg.ToolEnabled, cfg.RequireApprovalPatterns),
		output: NewOutputGuardrail(cfg.OutputEnabled),
	}
}

// CheckInput applies input guardrails to inbound text.
func (p *Pipeline) CheckInput(input string) GuardrailResult {
	if p == nil {
		return GuardrailResult{Allow: true}
	}
	return p.input.Check(input)
}

// CheckTool applies tool guardrails to tool requests.
func (p *Pipeline) CheckTool(toolName, toolArgs string) GuardrailResult {
	if p == nil {
		return GuardrailResult{Allow: true}
	}
	return p.tool.Check(toolName, toolArgs)
}

// CheckOutput applies output guardrails to model responses.
func (p *Pipeline) CheckOutput(output string) GuardrailResult {
	if p == nil {
		return GuardrailResult{Allow: true}
	}
	return p.output.Check(output)
}
