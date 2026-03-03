package rules

import (
	"strings"

	"github.com/startower-observability/blackcat/internal/hooks"
)

// RulesHookHandler injects matched rules into post-file-read output.
type RulesHookHandler struct {
	engine *Engine
}

// NewRulesHookHandler creates a hook handler backed by a rules engine.
func NewRulesHookHandler(engine *Engine) *RulesHookHandler {
	return &RulesHookHandler{engine: engine}
}

// HandlePostFileRead appends matched rules to the file-read response.
func (h *RulesHookHandler) HandlePostFileRead(ctx *hooks.HookContext) error {
	if h == nil || h.engine == nil || ctx == nil || ctx.LLMResponse == nil {
		return nil
	}

	matchedRules := h.engine.Match(ctx.FilePath)
	if len(matchedRules) == 0 {
		return nil
	}

	contents := make([]string, 0, len(matchedRules))
	for _, rule := range matchedRules {
		contents = append(contents, rule.Content)
	}

	ctx.LLMResponse.Content += "\n\n<!-- Rules -->\n" + strings.Join(contents, "\n\n") + "\n<!-- /Rules -->"
	return nil
}
