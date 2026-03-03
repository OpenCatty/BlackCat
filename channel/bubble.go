package channel

import (
	"strings"
)

// SplitBubbles splits text into human-like message bubbles for chat platforms.
//
// Algorithm:
// 1. Split text by double-newline (\n\n) into paragraphs
// 2. For each paragraph:
//   - If it contains code block delimiters (```), keep it whole
//   - If length > 400 chars, split by sentence endings (". ", "! ", "? ")
//   - Otherwise keep as-is
//
// 3. Cap output at maxBubbles (if more, join remaining into last bubble)
// 4. Trim whitespace and filter empty bubbles
// 5. Always return at least 1 bubble
//
// Parameters:
//   - text: The text to split
//   - maxBubbles: Maximum number of bubbles to return (defaults to 5 if <= 0)
//
// Returns: Slice of message bubbles
func SplitBubbles(text string, maxBubbles int, maxBubbleSize int) []string {
	// Default maxBubbles
	if maxBubbles <= 0 {
		maxBubbles = 5
	}

	// Trim input
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""} // Never return empty slice
	}

	// Split by double-newline into paragraphs
	paragraphs := strings.Split(text, "\n\n")

	var rawBubbles []string

	// Process each paragraph
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Check if paragraph contains code blocks
		if containsCodeBlock(para) {
			// Keep code blocks whole
			rawBubbles = append(rawBubbles, para)
		} else if len(para) > 400 {
			// Split long paragraphs by sentences
			sentences := splitBySentences(para)
			rawBubbles = append(rawBubbles, sentences...)
		} else {
			// Keep short paragraphs as-is
			rawBubbles = append(rawBubbles, para)
		}
	}

	// Filter empty bubbles
	var filtered []string
	for _, b := range rawBubbles {
		if strings.TrimSpace(b) != "" {
			filtered = append(filtered, strings.TrimSpace(b))
		}
	}

	// If no bubbles after filtering, return the original text as single bubble
	if len(filtered) == 0 {
		return []string{strings.TrimSpace(text)}
	}

	bubbles := filtered

	// Cap at maxBubbles
	if len(bubbles) > maxBubbles {
		capped := bubbles[:maxBubbles-1]
		// Join remaining bubbles into the last slot
		remaining := strings.Join(bubbles[maxBubbles-1:], "\n\n")
		bubbles = append(capped, remaining)
	}

	// Hard cap: truncate any bubble exceeding maxBubbleSize
	if maxBubbleSize > 0 {
		for i, b := range bubbles {
			runes := []rune(b)
			if len(runes) > maxBubbleSize {
				const suffix = "\n…[truncated]"
				if maxBubbleSize > len([]rune(suffix)) {
					bubbles[i] = string(runes[:maxBubbleSize-len([]rune(suffix))]) + suffix
				} else {
					bubbles[i] = string(runes[:maxBubbleSize])
				}
			}
		}
	}

	return bubbles
}

// containsCodeBlock checks if text contains code block delimiters (```).
// A paragraph with an odd number of ``` is inside a code block boundary.
func containsCodeBlock(text string) bool {
	count := strings.Count(text, "```")
	return count > 0 && count%2 == 1
}

// splitBySentences splits text by sentence-ending punctuation followed by space.
// Attempts to keep sentences together while respecting a ~400 char threshold per bubble.
func splitBySentences(text string) []string {
	// Define sentence endings
	endings := []string{". ", "! ", "? "}

	// Split by any sentence ending
	var parts []string
	current := text
	for {
		found := false
		minIdx := len(current)
		var foundEnding string

		for _, ending := range endings {
			idx := strings.Index(current, ending)
			if idx != -1 && idx < minIdx {
				minIdx = idx
				foundEnding = ending
				found = true
			}
		}

		if !found {
			// No more sentence endings
			if strings.TrimSpace(current) != "" {
				parts = append(parts, strings.TrimSpace(current))
			}
			break
		}

		// Extract up to and including the sentence ending
		sentence := current[:minIdx+len(foundEnding)]
		parts = append(parts, strings.TrimSpace(sentence))
		current = current[minIdx+len(foundEnding):]
	}

	// Group sentences into bubbles, respecting ~400 char limit
	if len(parts) == 0 {
		return []string{text}
	}

	var bubbles []string
	var currentBubble strings.Builder

	for _, part := range parts {
		// If adding this part would exceed 400 chars, start a new bubble
		potentialLen := currentBubble.Len() + 1 + len(part) // +1 for space
		if currentBubble.Len() > 0 && potentialLen > 400 {
			bubbles = append(bubbles, strings.TrimSpace(currentBubble.String()))
			currentBubble.Reset()
		}

		if currentBubble.Len() > 0 {
			currentBubble.WriteString(" ")
		}
		currentBubble.WriteString(part)
	}

	if currentBubble.Len() > 0 {
		bubbles = append(bubbles, strings.TrimSpace(currentBubble.String()))
	}

	if len(bubbles) == 0 {
		return []string{text}
	}

	return bubbles
}
