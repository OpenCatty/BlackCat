// Package qr provides terminal QR code rendering for BlackCat.
// It wraps the qrterminal library to display scannable QR codes as ASCII art.
package qr

import (
	"io"

	"github.com/mdp/qrterminal/v3"
)

// RenderToTerminal prints a QR code as ASCII art to the given writer.
// The content string is encoded into the QR code (e.g. a WhatsApp pairing URL).
func RenderToTerminal(w io.Writer, content string) {
	qrterminal.GenerateWithConfig(content, qrterminal.Config{
		Level:          qrterminal.L,
		Writer:         w,
		HalfBlocks:     true,
		BlackChar:      qrterminal.BLACK_BLACK,
		WhiteChar:      qrterminal.WHITE_WHITE,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
	})
}

// RenderToTerminalBasic prints a QR code using simple block characters.
// Use this as a fallback when the terminal doesn't support half-block characters.
func RenderToTerminalBasic(w io.Writer, content string) {
	qrterminal.GenerateWithConfig(content, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    w,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
	})
}
