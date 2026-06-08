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

func (s *InteractiveState) resetViewport() {
	cur := s.currentSeries()
	s.ViewStart = 0
	s.ViewEnd = len(cur.Values)
}

func (s *InteractiveState) currentSeries() thanos.Series {
	return s.AllSeries[s.SeriesIndex]
}

func (s *InteractiveState) viewLength() int {
	return s.ViewEnd - s.ViewStart
}

func (s *InteractiveState) panRight() {
	cur := s.currentSeries()
	shift := max(1, s.viewLength()/10)

	if s.ViewEnd+shift > len(cur.Values) {
		shift = len(cur.Values) - s.ViewEnd
	}

	s.ViewStart += shift
	s.ViewEnd += shift
}

func (s *InteractiveState) panLeft() {
	shift := max(1, s.viewLength()/10)

	if s.ViewStart-shift < 0 {
		shift = s.ViewStart
	}

	s.ViewStart -= shift
	s.ViewEnd -= shift
}

func (s *InteractiveState) zoomIn() {
	shrink := max(1, s.viewLength()/10)
	if s.viewLength()-2*shrink < 5 {
		return
	}

	s.ViewStart += shrink
	s.ViewEnd -= shrink
}

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

func (s *InteractiveState) nextSeries() {
	if s.SeriesIndex < len(s.AllSeries)-1 {
		s.SeriesIndex++
		s.resetViewport()
	}
}

func (s *InteractiveState) prevSeries() {
	if s.SeriesIndex > 0 {
		s.SeriesIndex--
		s.resetViewport()
	}
}

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
		fmt.Print("\033[?25h")
	}()

	fmt.Print("\033[?25l")

	state := NewInteractiveState(series, query)
	renderFrame(state)

	buf := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return nil
		}

		if n == 1 {
			switch buf[0] {
			case 'q', 'Q':
				return nil
			case 3: // Ctrl+C
				return nil
			case ' ':
				state.nextSeries()
			case 127: // Backspace
				state.prevSeries()
			case 'g', 'G':
				idx, ok := runPicker(state.seriesLabels())
				if ok {
					state.SeriesIndex = idx
					state.resetViewport()
				}
			}
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case 'C': // Right
				state.panRight()
			case 'D': // Left
				state.panLeft()
			case 'A': // Up = zoom in
				state.zoomIn()
			case 'B': // Down = zoom out
				state.zoomOut()
			}
		}

		renderFrame(state)
	}
}

func renderFrame(s *InteractiveState) {
	termW, termH := config.TerminalSize()

	chartHeight := termH - statusBarRows

	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]

	caption := s.Query + "\n" + thanos.LabelSetString(cur.Labels)
	opts := buildChartOptions(viewVals, viewTimes, termW, chartHeight, caption)

	graph := strings.ReplaceAll(
		asciigraph.Plot(viewVals, opts...),
		"\n", "\r\n",
	)

	labels := thanos.LabelSetString(cur.Labels)
	status := fmt.Sprintf("  Series %d/%d %s | Samples %d-%d of %d | \u2190/\u2192 pan  \u2191/\u2193 zoom  Space/Bksp series  g goto  q quit",
		s.SeriesIndex+1, len(s.AllSeries), labels,
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)

	if len(status) > termW {
		status = status[:termW]
	}

	var sb strings.Builder
	sb.WriteString("\033[2J\033[H")
	sb.WriteString(graph)
	sb.WriteString("\r\n")
	fmt.Fprintf(&sb, "\033[%d;1H", termH)
	fmt.Fprintf(&sb, "%-*s", termW, status)

	fmt.Print(sb.String())
}
