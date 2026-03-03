package llm

// EstimateTokens provides a rough token count using the ~4 chars/token heuristic.
// Uses rune count (not byte count) for correct handling of multi-byte UTF-8 (Indonesian, emoji).
func EstimateTokens(s string) int {
	return (len([]rune(s)) + 3) / 4
}
