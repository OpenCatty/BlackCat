package channel

import (
	"strings"
	"testing"
)

func TestSplitBubbles(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		maxBubbles int
		want       []string
	}{
		{
			name:       "short text returns single bubble",
			text:       "Hello world",
			maxBubbles: 5,
			want:       []string{"Hello world"},
		},
		{
			name:       "empty string returns single empty bubble",
			text:       "",
			maxBubbles: 5,
			want:       []string{""},
		},
		{
			name:       "whitespace only returns single empty bubble",
			text:       "   \n\n   ",
			maxBubbles: 5,
			want:       []string{""},
		},
		{
			name:       "two paragraphs returns two bubbles",
			text:       "Paragraph 1\n\nParagraph 2",
			maxBubbles: 5,
			want:       []string{"Paragraph 1", "Paragraph 2"},
		},
		{
			name:       "three paragraphs returns three bubbles",
			text:       "Para 1\n\nPara 2\n\nPara 3",
			maxBubbles: 5,
			want:       []string{"Para 1", "Para 2", "Para 3"},
		},
		{
			name:       "default maxBubbles when zero (defaults to 5)",
			text:       "Para 1\n\nPara 2\n\nPara 3\n\nPara 4\n\nPara 5\n\nPara 6",
			maxBubbles: 0,
			want:       []string{"Para 1", "Para 2", "Para 3", "Para 4", "Para 5\n\nPara 6"},
		},
		{
			name:       "default maxBubbles when negative (defaults to 5)",
			text:       "Para 1\n\nPara 2\n\nPara 3\n\nPara 4\n\nPara 5\n\nPara 6",
			maxBubbles: -1,
			want:       []string{"Para 1", "Para 2", "Para 3", "Para 4", "Para 5\n\nPara 6"},
		},
		{
			name:       "long paragraph does not split if under 400 chars",
			text:       "This is a long sentence. This is another long sentence. This is a third long sentence.",
			maxBubbles: 5,
			want:       []string{"This is a long sentence. This is another long sentence. This is a third long sentence."},
		},
		{
			name:       "very long paragraph (>400) splits by sentences",
			text:       generateLongParagraph(500), // ~500 chars of sentences
			maxBubbles: 5,
			want:       nil, // Checked below with len check
		},
		{
			name:       "max bubbles cap - 6 paras with maxBubbles 3",
			text:       "Para 1\n\nPara 2\n\nPara 3\n\nPara 4\n\nPara 5\n\nPara 6",
			maxBubbles: 3,
			want:       []string{"Para 1", "Para 2", "Para 3\n\nPara 4\n\nPara 5\n\nPara 6"},
		},
		{
			name:       "code block not split",
			text:       "Here is code:\n\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\nEnd.",
			maxBubbles: 5,
			want:       []string{"Here is code:", "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```", "End."},
		},
		{
			name:       "code block in middle not split - odd backticks",
			text:       "Text before\n\nSome code with ``` in it that's incomplete",
			maxBubbles: 5,
			want:       []string{"Text before", "Some code with ``` in it that's incomplete"},
		},
		{
			name:       "multiple short paragraphs stay together if under 400",
			text:       "Short para.\n\nAnother short one.\n\nThird short para.",
			maxBubbles: 5,
			want:       []string{"Short para.", "Another short one.", "Third short para."},
		},
		{
			name:       "leading and trailing whitespace trimmed",
			text:       "  \n\n  Hello world  \n\n  ",
			maxBubbles: 5,
			want:       []string{"Hello world"},
		},
		{
			name:       "single word",
			text:       "Hello",
			maxBubbles: 5,
			want:       []string{"Hello"},
		},
		{
			name:       "single sentence with punctuation",
			text:       "Hello world.",
			maxBubbles: 5,
			want:       []string{"Hello world."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitBubbles(tt.text, tt.maxBubbles, 0)

			// For generated long text, just verify it splits into multiple bubbles
			if tt.name == "very long paragraph (>400) splits by sentences" {
				if len(got) < 2 {
					t.Errorf("Expected at least 2 bubbles for long text, got %d: %v", len(got), got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("SplitBubbles(%q, %d) returned %d bubbles, want %d\nGot: %v\nWant: %v", tt.text, tt.maxBubbles, len(got), len(tt.want), got, tt.want)
				return
			}

			for i, bubble := range got {
				if bubble != tt.want[i] {
					t.Errorf("SplitBubbles(%q, %d) bubble %d:\ngot:  %q\nwant: %q", tt.text, tt.maxBubbles, i, bubble, tt.want[i])
				}
			}
		})
	}
}

func TestSplitBubbles_NeverReturnsEmptySlice(t *testing.T) {
	tests := []string{
		"",
		"  ",
		"\n\n",
		"  \n\n  ",
	}

	for _, text := range tests {
		got := SplitBubbles(text, 5, 0)
		if len(got) == 0 {
			t.Errorf("SplitBubbles(%q, 5) returned empty slice, should have at least 1 element", text)
		}
	}
}

func TestSplitBubbles_MaxBubblesRespected(t *testing.T) {
	tests := []struct {
		text       string
		maxBubbles int
		maxLength  int
	}{
		{"A\n\nB\n\nC\n\nD\n\nE\n\nF", 3, 3},
		{"A\n\nB\n\nC\n\nD\n\nE\n\nF", 2, 2},
		{"A\n\nB\n\nC", 10, 3},
	}

	for _, tt := range tests {
		got := SplitBubbles(tt.text, tt.maxBubbles, 0)
		if len(got) > tt.maxLength {
			t.Errorf("SplitBubbles(%q, %d) returned %d bubbles, max should be %d", tt.text, tt.maxBubbles, len(got), tt.maxLength)
		}
	}
}

func TestContainsCodeBlock(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"normal text", false},
		{"text with ` backtick", false},
		{"text with ``` code", true},
		{"```\ncode\n```", false}, // Balanced, so even count = false
		{"start ``` middle", true},
		{"```go\ncode\n``` more text ```", true}, // Odd count = true
	}

	for _, tt := range tests {
		got := containsCodeBlock(tt.text)
		if got != tt.want {
			t.Errorf("containsCodeBlock(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestSplitBySentences(t *testing.T) {
	// Note: splitBySentences is called on text > 400 chars
	// Generate longer text for proper testing
	long1 := generateLongParagraph(500) // Will be split
	long2 := generateLongParagraph(600)

	tests := []struct {
		text       string
		minBubbles int
	}{
		{long1, 2}, // Should split into at least 2 bubbles
		{long2, 2}, // Should split into at least 2 bubbles
	}

	for i, tt := range tests {
		got := splitBySentences(tt.text)
		if len(got) < tt.minBubbles {
			t.Errorf("Test %d: splitBySentences(%d-char text) returned %d bubbles, expected at least %d", i, len(tt.text), len(got), tt.minBubbles)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int // Minimum expected bubbles
	}{
		{"Single word", "Hello", 1},
		{"Single sentence", "Hello world.", 1},
		{"Sentence without punctuation", "Hello world", 1},
		{"Only punctuation", "...", 1},
		{"Mixed punctuation", "What!?", 1},
	}

	for _, tt := range tests {
		got := SplitBubbles(tt.text, 5, 0)
		if len(got) < tt.want {
			t.Errorf("SplitBubbles(%q, 5) returned %d bubbles, expected at least %d", tt.text, len(got), tt.want)
		}
	}
}

func TestSplitBubbles_HardCap(t *testing.T) {
	text := strings.Repeat("a", 10050)
	bubbles := SplitBubbles(text, 10, 4096)

	if len(bubbles) == 0 {
		t.Fatal("SplitBubbles returned no bubbles")
	}

	for i, bubble := range bubbles {
		if len([]rune(bubble)) > 4096 {
			t.Fatalf("bubble %d length = %d, want <= 4096", i, len([]rune(bubble)))
		}
	}
}

func TestSplitBubbles_NoCap(t *testing.T) {
	text := strings.Repeat("a", 10050)
	bubbles := SplitBubbles(text, 10, 0)

	if len(bubbles) == 0 {
		t.Fatal("SplitBubbles returned no bubbles")
	}

	foundOver4096 := false
	for _, bubble := range bubbles {
		if len([]rune(bubble)) > 4096 {
			foundOver4096 = true
			break
		}
	}

	if !foundOver4096 {
		t.Fatal("expected at least one bubble to exceed 4096 runes when no cap is set")
	}
}

// Helper: Generate text longer than 400 chars with multiple sentences
func generateLongParagraph(minLength int) string {
	sentences := []string{
		"This is the first sentence in our test. ",
		"This is the second sentence. ",
		"This is the third sentence. ",
		"This is the fourth sentence. ",
		"This is the fifth sentence. ",
		"This is the sixth sentence. ",
		"This is the seventh sentence. ",
		"This is the eighth sentence. ",
		"This is the ninth sentence. ",
		"This is the tenth sentence. ",
	}

	var result string
	for result = ""; len(result) < minLength; {
		for _, s := range sentences {
			result += s
			if len(result) >= minLength {
				break
			}
		}
	}
	return result
}
