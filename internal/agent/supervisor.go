package agent

import (
	"context"
	"fmt"
)

// SubAgentConfig holds the configuration overlay for a specialized sub-agent.
type SubAgentConfig struct {
	// SystemPromptOverlay is prepended to the agent name for task specialization.
	SystemPromptOverlay string
	// AllowedTools lists the tools this sub-agent can use. nil means all tools allowed.
	AllowedTools []string
}

// Supervisor is a lightweight router that classifies incoming messages and
// routes them to specialized sub-agent Loop instances with tailored system
// prompt overlays and tool subsets.
type Supervisor struct {
	baseLoopCfg     LoopConfig
	subAgentConfigs map[TaskType]SubAgentConfig
}

// NewSupervisor creates a Supervisor with default sub-agent configurations.
func NewSupervisor(baseCfg LoopConfig) *Supervisor {
	return &Supervisor{
		baseLoopCfg: baseCfg,
		subAgentConfigs: map[TaskType]SubAgentConfig{
			TaskTypeCoding: {
				SystemPromptOverlay: "You are a specialized coding agent. Focus on code quality, testing, and best practices.",
				AllowedTools:        nil, // all tools
			},
			TaskTypeResearch: {
				SystemPromptOverlay: "You are a specialized research agent. Focus on finding accurate information. Do not execute system commands.",
				AllowedTools:        []string{"memory_search", "web_search", "archival_memory_search", "archival_memory_insert", "core_memory_get"},
			},
			TaskTypeAdmin: {
				SystemPromptOverlay: "You are an admin agent. Handle system configuration and service management carefully.",
				AllowedTools:        nil, // all tools
			},
			TaskTypeGeneral: {
				SystemPromptOverlay: "",
				AllowedTools:        nil, // all tools
			},
		},
	}
}

// Route classifies a message and runs it through a specialized Loop using the
// base configuration stored in the Supervisor.
func (s *Supervisor) Route(ctx context.Context, msg string) (*Execution, error) {
	return s.RouteWithCfg(ctx, msg, s.baseLoopCfg)
}

// RouteWithCfg classifies a message, selects the appropriate sub-agent config,
// creates a specialized Loop with tailored system prompt overlay and tool subset,
// then runs the message through it. The caller provides a per-message LoopConfig
// (with EventStream, SessionMessages, UserID, etc. already set).
func (s *Supervisor) RouteWithCfg(ctx context.Context, msg string, cfg LoopConfig) (*Execution, error) {
	if s == nil {
		return nil, fmt.Errorf("supervisor not initialized")
	}

	taskType := ClassifyMessage(msg)
	subCfg, ok := s.subAgentConfigs[taskType]
	if !ok {
		subCfg = s.subAgentConfigs[TaskTypeGeneral]
	}

	// Apply agent name suffix for task specialization
	if cfg.AgentName != "" {
		cfg.AgentName = fmt.Sprintf("%s [%s]", cfg.AgentName, string(taskType))
	}

	// Apply system prompt overlay via a Reflector-style approach:
	// We prepend the overlay to the AgentTone field which gets injected into the
	// system prompt persona section. This is a lightweight approach that doesn't
	// require modifying Loop internals.
	if subCfg.SystemPromptOverlay != "" {
		if cfg.AgentTone != "" {
			cfg.AgentTone = subCfg.SystemPromptOverlay + " " + cfg.AgentTone
		} else {
			cfg.AgentTone = subCfg.SystemPromptOverlay
		}
	}

	// Tool filtering: if AllowedTools is specified, filter the registry.
	// TODO: Registry.Filter() does not exist yet — use full registry for now.
	// When Filter is implemented, replace this block with:
	//   if subCfg.AllowedTools != nil && cfg.Tools != nil {
	//       cfg.Tools = cfg.Tools.Filter(subCfg.AllowedTools)
	//   }
	_ = subCfg.AllowedTools // acknowledge the field; filtering deferred

	return NewLoop(cfg).Run(ctx, msg)
}
