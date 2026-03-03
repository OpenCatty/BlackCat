package qr

import (
	"bytes"
	"testing"
)

func TestRenderToTerminal(t *testing.T) {
	var buf bytes.Buffer
	RenderToTerminal(&buf, "https://example.com/test")
	if buf.Len() == 0 {
		t.Error("RenderToTerminal produced empty output")
	}
}

func TestRenderToTerminalBasic(t *testing.T) {
	var buf bytes.Buffer
	RenderToTerminalBasic(&buf, "https://example.com/test")
	if buf.Len() == 0 {
		t.Error("RenderToTerminalBasic produced empty output")
	}
}

func TestRenderToTerminal_DifferentContent(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	RenderToTerminal(&buf1, "content-a")
	RenderToTerminal(&buf2, "content-b")
	if buf1.String() == buf2.String() {
		t.Error("different content should produce different QR codes")
	}
}
