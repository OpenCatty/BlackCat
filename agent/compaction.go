package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/startower-observability/blackcat/memory"
	"github.com/startower-observability/blackcat/types"
)

const (
	defaultCompactionThreshold   = 0.835
	defaultCompactionMaxTokens   = 128000
	defaultCompactionMinMessages = 10
)

type Compactor struct {
	llm         types.LLMClient
	memory      *memory.FileStore
	threshold   float64
	maxTokens   int
	minMessages int
}

type CompactorConfig struct {
	LLM         types.LLMClient
	Memory      *memory.FileStore
	Threshold   float64
	MaxTokens   int
	MinMessages int
}

func NewCompactor(cfg CompactorConfig) *Compactor {
	threshold := cfg.Threshold
	if threshold == 0 {
		threshold = defaultCompactionThreshold
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultCompactionMaxTokens
	}

	minMessages := cfg.MinMessages
	if minMessages == 0 {
		minMessages = defaultCompactionMinMessages
	}

	return &Compactor{
		llm:         cfg.LLM,
		memory:      cfg.Memory,
		threshold:   threshold,
		maxTokens:   maxTokens,
		minMessages: minMessages,
	}
}

func (c *Compactor) ShouldCompact(messages []types.LLMMessage) bool {
	if len(messages) <= c.minMessages {
		return false
	}
	tokens := estimateTokens(messages)
	return float64(tokens) > c.threshold*float64(c.maxTokens)
}

func (c *Compactor) Compact(ctx context.Context, messages []types.LLMMessage) ([]types.LLMMessage, error) {
	if len(messages) == 0 {
		return messages, nil
	}
	if c.llm == nil {
		return nil, fmt.Errorf("compactor: llm client is nil")
	}

	hasSystem := messages[0].Role == "system"
	summaryStart := 0
	if hasSystem {
		summaryStart = 1
	}

	if len(messages)-summaryStart <= c.minMessages {
		return messages, nil
	}

	tailStart := len(messages) - c.minMessages
	if tailStart < summaryStart {
		tailStart = summaryStart
	}

	toSummarize := messages[summaryStart:tailStart]
	if len(toSummarize) == 0 {
		return messages, nil
	}

	prompt := "Summarize the following conversation concisely, preserving key decisions, code changes, and important context:\n\n" + formatConversation(toSummarize)
	resp, err := c.llm.Chat(ctx, []types.LLMMessage{{Role: "user", Content: prompt}}, nil)
	if err != nil {
		return nil, fmt.Errorf("compactor: summarize conversation: %w", err)
	}

	summaryMessage := types.LLMMessage{
		Role:    "assistant",
		Content: "[Compaction Summary]\n<!-- compaction-boundary -->\n" + strings.TrimSpace(resp.Content),
	}

	compacted := make([]types.LLMMessage, 0, 2+c.minMessages)
	if hasSystem {
		compacted = append(compacted, messages[0])
	}
	compacted = append(compacted, summaryMessage)
	compacted = append(compacted, messages[tailStart:]...)

	return compacted, nil
}

func (c *Compactor) FlushToMemory(ctx context.Context, messages []types.LLMMessage) error {
	if c.memory == nil || len(messages) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	now := time.Now().UTC()
	for _, msg := range messages {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}

		for _, line := range splitFacts(msg.Content) {
			fact := strings.TrimSpace(line)
			if fact == "" {
				continue
			}
			key := msg.Role + "|" + fact
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			entry := memory.Entry{
				Timestamp: now,
				Content:   fmt.Sprintf("%s: %s", msg.Role, fact),
				Tags:      []string{"compaction", "auto"},
			}
			if err := c.memory.Write(ctx, entry); err != nil {
				return fmt.Errorf("compactor: write memory entry: %w", err)
			}
		}
	}

	return nil
}

func estimateTokens(messages []types.LLMMessage) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content) / 4
		for _, call := range msg.ToolCalls {
			total += len(call.Arguments) / 4
		}
	}
	return total
}

func formatConversation(messages []types.LLMMessage) string {
	var b strings.Builder
	for i, msg := range messages {
		b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, msg.Role, strings.TrimSpace(msg.Content)))
		for _, call := range msg.ToolCalls {
			args := strings.TrimSpace(string(call.Arguments))
			if args == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("   tool:%s args:%s\n", call.Name, args))
		}
	}
	return b.String()
}

func splitFacts(content string) []string {
	lines := strings.Split(content, "\n")
	facts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-* ")
		if line == "" {
			continue
		}
		facts = append(facts, line)
	}
	return facts
}
