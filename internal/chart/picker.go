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
	keyEscape       = 27
	keyEnter        = 13
	keyBackspace    = 127
	keyCtrlC        = 3
)

type pickerState struct {
	allLabels []string
	input     string
	filtered  []int
	cursor    int
	scroll    int
}

// newPickerState initializes picker state with all series visible.
func newPickerState(allLabels []string) *pickerState {
	p := &pickerState{allLabels: allLabels}
	p.refilter()

	return p
}

// refilter recomputes visible indices and resets cursor/scroll positions.
func (p *pickerState) refilter() {
	p.filtered = filterSeries(p.allLabels, p.input)
	p.cursor = 0
	p.scroll = 0
}

// cursorUp moves picker selection one item up.
func (p *pickerState) cursorUp() {
	if p.cursor > 0 {
		p.cursor--

		if p.cursor < p.scroll {
			p.scroll = p.cursor
		}
	}
}

// cursorDown moves picker selection one item down.
func (p *pickerState) cursorDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++

		if p.cursor >= p.scroll+maxVisibleItems {
			p.scroll = p.cursor - maxVisibleItems + 1
		}
	}
}

// addChar appends a typed character to the search query.
func (p *pickerState) addChar(ch byte) {
	p.input += string(ch)
	p.refilter()
}

// backspace removes the last typed character from the search query.
func (p *pickerState) backspace() {
	if len(p.input) > 0 {
		p.input = p.input[:len(p.input)-1]
		p.refilter()
	}
}

// selectedIndex returns the original series index under the cursor.
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
	showTerminalCursor()
	defer hideTerminalCursor()

	buf := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return -1, false
		}

		idx, done, selected := handlePickerSingleByteInput(state, n, buf)
		if done {
			return idx, selected
		}
		handlePickerArrowInput(state, n, buf)

		renderPicker(state)
	}
}

// handlePickerSingleByteInput handles Enter/Escape/backspace and printable keys.
func handlePickerSingleByteInput(state *pickerState, n int, buf []byte) (index int, done bool, selected bool) {
	if n != 1 {
		return -1, false, false
	}

	switch {
	case buf[0] == keyEscape || buf[0] == keyCtrlC:
		return -1, true, false
	case buf[0] == keyEnter:
		idx := state.selectedIndex()
		if idx >= 0 {
			return idx, true, true
		}
		return -1, true, false
	case buf[0] == keyBackspace:
		state.backspace()
	case buf[0] >= 32 && buf[0] < keyBackspace:
		state.addChar(buf[0])
	}

	return -1, false, false
}

// handlePickerArrowInput handles up/down arrow navigation for picker results.
func handlePickerArrowInput(state *pickerState, n int, buf []byte) {
	if n != 3 || buf[0] != keyEscape || buf[1] != 91 {
		return
	}

	switch buf[2] {
	case 'A':
		state.cursorUp()
	case 'B':
		state.cursorDown()
	}
}

// renderPicker draws the series picker overlay.
func renderPicker(state *pickerState) {
	termW, termH := config.TerminalSize()
	startRow, startCol, boxW, boxH := pickerBoxLayout(termW, termH)
	innerW := boxW - 4 // 2 for border + 1 padding each side

	var sb strings.Builder
	clearPickerOverlay(&sb, startRow, startCol, boxW, boxH)
	drawPickerContent(&sb, state, startRow, startCol, boxW, innerW)
	fmt.Print(sb.String())
}

// pickerBoxLayout calculates centered picker box geometry for the current terminal size.
func pickerBoxLayout(termW, termH int) (startRow, startCol, boxW, boxH int) {
	boxW = termW * 2 / 3
	if boxW < boxMinWidth {
		boxW = boxMinWidth
	}
	if boxW > termW-4 {
		boxW = termW - 4
	}

	boxH = maxVisibleItems + 7
	if boxH > termH-2 {
		boxH = termH - 2
	}

	startRow = max((termH-boxH)/2, 1)
	startCol = max((termW-boxW)/2, 1)
	return startRow, startCol, boxW, boxH
}

// clearPickerOverlay clears the rectangle where the picker overlay is rendered.
func clearPickerOverlay(sb *strings.Builder, startRow, startCol, boxW, boxH int) {
	for r := startRow; r <= startRow+boxH; r++ {
		writeAt(sb, r, startCol, "%-*s", boxW, "")
	}
}

// drawPickerContent renders picker borders, input, list area, and footer.
func drawPickerContent(sb *strings.Builder, state *pickerState, startRow, startCol, boxW, innerW int) {
	row := startRow
	writeAt(sb, row, startCol, "\u250c%s\u2510", buildTopBorder(boxW-2, " Go to series "))
	row++

	searchLine := fmt.Sprintf("Search: %s", state.input)
	writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, truncate(searchLine, innerW))
	row++
	writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, "")
	row++

	row = drawPickerItems(sb, state, row, startCol, innerW)
	writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, "")
	row++

	footer := fmt.Sprintf("%d of %d series | Enter select  Esc cancel", len(state.filtered), len(state.allLabels))
	writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, truncate(footer, innerW))
	row++
	writeAt(sb, row, startCol, "\u2514%s\u2518", strings.Repeat("\u2500", boxW-2))
}

// drawPickerItems renders the visible series rows and returns the next free row.
func drawPickerItems(sb *strings.Builder, state *pickerState, row, startCol, innerW int) int {
	endIdx := min(state.scroll+maxVisibleItems, len(state.filtered))
	itemsShown := 0

	for i := state.scroll; i < endIdx; i++ {
		label := state.allLabels[state.filtered[i]]
		prefix := "  "
		if i == state.cursor {
			prefix = "> "
		}
		writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, truncate(prefix+label, innerW))
		row++
		itemsShown++
	}

	for itemsShown < maxVisibleItems {
		writeAt(sb, row, startCol, "\u2502 %-*s \u2502", innerW, "")
		row++
		itemsShown++
	}

	return row
}

// buildTopBorder builds the titled top border for the picker box.
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

// truncate shortens strings to fit a fixed-width terminal field.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
