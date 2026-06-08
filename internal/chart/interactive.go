package chart

import (
	"fmt"
	"os"
	"strings"

	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/guptarohit/asciigraph"
	"golang.org/x/term"
)

const statusBarRows = 1

// InteractiveState holds the mutable state for interactive chart navigation.
type InteractiveState struct {
	AllSeries   []thanos.Series
	SeriesIndex int
	ViewStart   int
	ViewEnd     int
	Query       string
}

// NewInteractiveState creates an initial state for the given series.
func NewInteractiveState(series []thanos.Series, query string) *InteractiveState {
	s := &InteractiveState{
		AllSeries: series,
		Query:     query,
	}

	s.resetViewport()

	return s
}

// resetViewport expands the viewport to the full range of the current series.
func (s *InteractiveState) resetViewport() {
	cur := s.currentSeries()
	s.ViewStart = 0
	s.ViewEnd = len(cur.Values)
}

// currentSeries returns the series currently selected for display.
func (s *InteractiveState) currentSeries() thanos.Series {
	return s.AllSeries[s.SeriesIndex]
}

// viewLength returns the number of points currently visible in the viewport.
func (s *InteractiveState) viewLength() int {
	return s.ViewEnd - s.ViewStart
}

// panRight shifts the viewport to newer samples.
func (s *InteractiveState) panRight() {
	cur := s.currentSeries()
	shift := max(1, s.viewLength()/10)

	if s.ViewEnd+shift > len(cur.Values) {
		shift = len(cur.Values) - s.ViewEnd
	}

	s.ViewStart += shift
	s.ViewEnd += shift
}

// panLeft shifts the viewport to older samples.
func (s *InteractiveState) panLeft() {
	shift := max(1, s.viewLength()/10)

	if s.ViewStart-shift < 0 {
		shift = s.ViewStart
	}

	s.ViewStart -= shift
	s.ViewEnd -= shift
}

// zoomIn narrows the viewport around its current center.
func (s *InteractiveState) zoomIn() {
	shrink := max(1, s.viewLength()/10)
	if s.viewLength()-2*shrink < 5 {
		return
	}

	s.ViewStart += shrink
	s.ViewEnd -= shrink
}

// zoomOut widens the viewport while staying inside data bounds.
func (s *InteractiveState) zoomOut() {
	cur := s.currentSeries()
	grow := max(1, s.viewLength()/10)

	s.ViewStart -= grow
	if s.ViewStart < 0 {
		s.ViewStart = 0
	}

	s.ViewEnd += grow
	if s.ViewEnd > len(cur.Values) {
		s.ViewEnd = len(cur.Values)
	}
}

// nextSeries selects the next series if one exists.
func (s *InteractiveState) nextSeries() {
	if s.SeriesIndex < len(s.AllSeries)-1 {
		s.SeriesIndex++
		s.resetViewport()
	}
}

// prevSeries selects the previous series if one exists.
func (s *InteractiveState) prevSeries() {
	if s.SeriesIndex > 0 {
		s.SeriesIndex--
		s.resetViewport()
	}
}

// seriesLabels returns rendered label strings for all available series.
func (s *InteractiveState) seriesLabels() []string {
	labels := make([]string, len(s.AllSeries))
	for i, series := range s.AllSeries {
		labels[i] = thanos.LabelSetString(series.Labels)
	}

	return labels
}

// RunInteractive enters a full-screen raw terminal loop for chart interaction.
func RunInteractive(series []thanos.Series, query string) error {
	if len(series) == 0 {
		return fmt.Errorf("no data to display")
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("entering raw mode: %w", err)
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		showTerminalCursor()
	}()

	hideTerminalCursor()

	state := NewInteractiveState(series, query)
	renderFrame(state)

	buf := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}

		if shouldQuit, handled := handleInteractiveSingleByteInput(state, n, buf); shouldQuit {
			return nil
		} else if !handled {
			handleInteractiveArrowInput(state, n, buf)
		}

		renderFrame(state)
	}
}

// handleInteractiveSingleByteInput handles one-byte key actions in interactive mode.
func handleInteractiveSingleByteInput(state *InteractiveState, n int, buf []byte) (quit bool, handled bool) {
	if n != 1 {
		return false, false
	}

	switch buf[0] {
	case 'q', 'Q', 3:
		return true, true
	case ' ':
		state.nextSeries()
	case 127:
		state.prevSeries()
	case 'g', 'G':
		idx, ok := runPicker(state.seriesLabels())
		if ok {
			state.SeriesIndex = idx
			state.resetViewport()
		}
	}

	return false, true
}

// handleInteractiveArrowInput handles ANSI arrow key sequences.
func handleInteractiveArrowInput(state *InteractiveState, n int, buf []byte) {
	// Arrow keys arrive as a 3-byte ANSI escape sequence: ESC [ <code>.
	if n != 3 || buf[0] != 27 || buf[1] != 91 {
		return
	}

	switch buf[2] {
	case 'C':
		state.panRight()
	case 'D':
		state.panLeft()
	case 'A':
		state.zoomIn()
	case 'B':
		state.zoomOut()
	}
}

// renderFrame renders one full-screen frame for interactive mode.
func renderFrame(s *InteractiveState) {
	termW, termH := config.TerminalSize()

	chartHeight := termH - statusBarRows

	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]

	caption := s.Query + "\n" + thanos.LabelSetString(cur.Labels)
	opts := buildChartOptions(viewVals, viewTimes, termW, chartHeight, caption)

	graph := toCRLF(asciigraph.Plot(viewVals, opts...))

	labels := thanos.LabelSetString(cur.Labels)
	status := fmt.Sprintf("  Series %d/%d %s | Samples %d-%d of %d | \u2190/\u2192 pan  \u2191/\u2193 zoom  Space/Bksp series  g goto  q quit",
		s.SeriesIndex+1, len(s.AllSeries), labels,
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)

	if len(status) > termW {
		status = status[:termW]
	}

	var sb strings.Builder
	clearScreenAndMoveHome(&sb)
	sb.WriteString(graph)
	sb.WriteString("\r\n")
	moveCursor(&sb, termH, 1)
	writePaddedLine(&sb, termW, status)

	fmt.Print(sb.String())
}
