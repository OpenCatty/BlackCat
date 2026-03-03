//go:build cgo

package whatsapp

import "testing"

func TestFormatForWhatsApp(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "bold", in: "**bold**", want: "*bold*"},
		{name: "italic", in: "*italic*", want: "_italic_"},
		{name: "strikethrough", in: "~~strike~~", want: "~strike~"},
		{name: "heading", in: "# Header", want: "*Header*"},
		{name: "code block unchanged", in: "```code```", want: "```code```"},
		{name: "mixed", in: "**bold** and *italic*", want: "*bold* and _italic_"},
		{name: "idempotent already wa bold", in: "already *bold*", want: "already *bold*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatForWhatsApp(tt.in)
			if got != tt.want {
				t.Fatalf("FormatForWhatsApp(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
