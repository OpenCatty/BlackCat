package agent

import "strings"

// TaskType represents the classification of an incoming message by task category.
type TaskType string

const (
	TaskTypeCoding   TaskType = "coding"
	TaskTypeResearch TaskType = "research"
	TaskTypeAdmin    TaskType = "admin"
	TaskTypeGeneral  TaskType = "general"
)

// classificationKeywords maps each TaskType to its trigger keywords.
var classificationKeywords = map[TaskType][]string{
	TaskTypeCoding: {
		"code", "implement", "write", "function", "bug", "fix", "test", "build",
		"compile", "git", "deploy", "opencode", "typescript", "golang", "python",
		"javascript",
	},
	TaskTypeResearch: {
		"search", "find", "look up", "what is", "explain", "research", "summarize",
		"web", "browse", "read",
	},
	TaskTypeAdmin: {
		"restart", "stop", "config", "setting", "service", "systemctl", "status",
		"health", "blackcat", "server",
	},
}

// ClassifyMessage classifies a message into a TaskType using keyword heuristics.
// Priority when multiple match: Admin > Coding > Research > General.
func ClassifyMessage(msg string) TaskType {
	lower := strings.ToLower(msg)

	matched := make(map[TaskType]bool)
	for taskType, keywords := range classificationKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				matched[taskType] = true
				break
			}
		}
	}

	// Priority: Admin > Coding > Research > General
	if matched[TaskTypeAdmin] {
		return TaskTypeAdmin
	}
	if matched[TaskTypeCoding] {
		return TaskTypeCoding
	}
	if matched[TaskTypeResearch] {
		return TaskTypeResearch
	}
	return TaskTypeGeneral
}
