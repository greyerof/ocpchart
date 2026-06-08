package chart

import (
	"fmt"
	"os"
	"strings"

	"github.com/greyerof/ocpchart/internal/config"
)

const (
	maxVisibleItems = 10
	boxMinWidth     = 40
)

type pickerState struct {
	allLabels []string
	input     string
	filtered  []int
	cursor    int
	scroll    int
}

func newPickerState(allLabels []string) *pickerState {
	p := &pickerState{allLabels: allLabels}
	p.refilter()

	return p
}

func (p *pickerState) refilter() {
	p.filtered = filterSeries(p.allLabels, p.input)
	p.cursor = 0
	p.scroll = 0
}

func (p *pickerState) cursorUp() {
	if p.cursor > 0 {
		p.cursor--

		if p.cursor < p.scroll {
			p.scroll = p.cursor
		}
	}
}

func (p *pickerState) cursorDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++

		if p.cursor >= p.scroll+maxVisibleItems {
			p.scroll = p.cursor - maxVisibleItems + 1
		}
	}
}

func (p *pickerState) addChar(ch byte) {
	p.input += string(ch)
	p.refilter()
}

func (p *pickerState) backspace() {
	if len(p.input) > 0 {
		p.input = p.input[:len(p.input)-1]
		p.refilter()
	}
}

func (p *pickerState) selectedIndex() int {
	if p.cursor >= 0 && p.cursor < len(p.filtered) {
		return p.filtered[p.cursor]
	}

	return -1
}

// filterSeries returns indices of labels that contain input as a
// case-insensitive substring. Empty input matches everything.
func filterSeries(allLabels []string, input string) []int {
	if input == "" {
		indices := make([]int, len(allLabels))
		for i := range indices {
			indices[i] = i
		}

		return indices
	}

	lower := strings.ToLower(input)
	var result []int

	for i, label := range allLabels {
		if strings.Contains(strings.ToLower(label), lower) {
			result = append(result, i)
		}
	}

	return result
}

// runPicker shows a modal series picker overlay and blocks until the user
// selects a series (Enter) or cancels (Escape). Returns the original series
// index and true on selection, or -1 and false on cancel.
// Must be called while the terminal is already in raw mode.
func runPicker(allLabels []string) (int, bool) {
	state := newPickerState(allLabels)
	renderPicker(state)

	// Show cursor inside the picker input field
	fmt.Print("\033[?25h")
	defer fmt.Print("\033[?25l")

	buf := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return -1, false
		}

		if n == 1 {
			switch {
			case buf[0] == 27: // Escape
				return -1, false
			case buf[0] == 13: // Enter
				idx := state.selectedIndex()
				if idx >= 0 {
					return idx, true
				}

				return -1, false
			case buf[0] == 127: // Backspace
				state.backspace()
			case buf[0] == 3: // Ctrl+C
				return -1, false
			case buf[0] >= 32 && buf[0] < 127: // Printable ASCII
				state.addChar(buf[0])
			}
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'A': // Up
				state.cursorUp()
			case 'B': // Down
				state.cursorDown()
			}
		}

		renderPicker(state)
	}
}

func renderPicker(state *pickerState) {
	termW, termH := config.TerminalSize()

	boxW := termW * 2 / 3
	if boxW < boxMinWidth {
		boxW = boxMinWidth
	}

	if boxW > termW-4 {
		boxW = termW - 4
	}

	innerW := boxW - 4 // 2 for border + 1 padding each side

	// Fixed box height based on maxVisibleItems so it never changes size
	boxH := maxVisibleItems + 7
	if boxH > termH-2 {
		boxH = termH - 2
	}

	startRow := (termH - boxH) / 2
	startCol := (termW - boxW) / 2

	if startRow < 1 {
		startRow = 1
	}

	if startCol < 1 {
		startCol = 1
	}

	var sb strings.Builder

	// Clear the entire overlay region first to avoid ghosting
	for r := startRow; r <= startRow+boxH; r++ {
		fmt.Fprintf(&sb, "\033[%d;%dH%-*s", r, startCol, boxW, "")
	}

	row := startRow

	// Top border with title
	title := " Go to series "
	topLine := buildTopBorder(boxW-2, title)
	fmt.Fprintf(&sb, "\033[%d;%dH\u250c%s\u2510", row, startCol, topLine)
	row++

	// Search input
	searchLine := fmt.Sprintf("Search: %s", state.input)
	fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, truncate(searchLine, innerW))
	row++

	// Blank separator
	fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, "")
	row++

	// Filtered items
	endIdx := state.scroll + maxVisibleItems
	if endIdx > len(state.filtered) {
		endIdx = len(state.filtered)
	}

	itemsShown := 0

	for i := state.scroll; i < endIdx; i++ {
		label := state.allLabels[state.filtered[i]]
		prefix := "  "

		if i == state.cursor {
			prefix = "> "
		}

		line := prefix + label
		fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, truncate(line, innerW))
		row++
		itemsShown++
	}

	// Fill remaining item slots with blank lines
	for itemsShown < maxVisibleItems && row < startRow+boxH-3 {
		fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, "")
		row++
		itemsShown++
	}

	// Blank separator
	fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, "")
	row++

	// Footer
	footer := fmt.Sprintf("%d of %d series | Enter select  Esc cancel", len(state.filtered), len(state.allLabels))
	fmt.Fprintf(&sb, "\033[%d;%dH\u2502 %-*s \u2502", row, startCol, innerW, truncate(footer, innerW))
	row++

	// Bottom border
	fmt.Fprintf(&sb, "\033[%d;%dH\u2514%s\u2518", row, startCol, strings.Repeat("\u2500", boxW-2))

	fmt.Print(sb.String())
}

func buildTopBorder(width int, title string) string {
	if len(title) >= width {
		return strings.Repeat("\u2500", width)
	}

	leftDashes := 3
	rightDashes := width - leftDashes - len(title)

	if rightDashes < 0 {
		rightDashes = 0
	}

	return strings.Repeat("\u2500", leftDashes) + title + strings.Repeat("\u2500", rightDashes)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
