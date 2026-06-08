package chart

import (
	"fmt"
	"strings"
)

const (
	ansiShowCursor        = "\033[?25h"
	ansiHideCursor        = "\033[?25l"
	ansiClearScreenAndTop = "\033[2J\033[H"
	ansiMoveCursorFormat  = "\033[%d;%dH"
)

// showTerminalCursor makes the terminal cursor visible.
func showTerminalCursor() {
	fmt.Print(ansiShowCursor)
}

// hideTerminalCursor hides the terminal cursor during full-screen rendering.
func hideTerminalCursor() {
	fmt.Print(ansiHideCursor)
}

// clearScreenAndMoveHome clears the terminal and moves the cursor to row 1, col 1.
func clearScreenAndMoveHome(sb *strings.Builder) {
	sb.WriteString(ansiClearScreenAndTop)
}

// moveCursor writes an ANSI command to place the cursor at a terminal row/column.
func moveCursor(sb *strings.Builder, row, col int) {
	fmt.Fprintf(sb, ansiMoveCursorFormat, row, col)
}

// writeAt moves the cursor and writes formatted content at that position.
func writeAt(sb *strings.Builder, row, col int, format string, args ...any) {
	moveCursor(sb, row, col)
	fmt.Fprintf(sb, format, args...)
}

// toCRLF normalizes line endings for raw-mode terminal drawing.
func toCRLF(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// writePaddedLine writes text padded to the terminal width.
func writePaddedLine(sb *strings.Builder, width int, text string) {
	fmt.Fprintf(sb, "%-*s", width, text)
}
