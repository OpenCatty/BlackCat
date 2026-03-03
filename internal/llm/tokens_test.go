package llm

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"hello", "hello", 2},           // 5 runes -> (5+3)/4 = 2
		{"indonesian", "ini bahasa", 3}, // 10 runes -> (10+3)/4 = 3
		{"emoji", "hello 🌍", 2},         // 7 runes -> (7+3)/4 = 2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
