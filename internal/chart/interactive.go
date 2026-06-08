package chart

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/guptarohit/asciigraph"
	"golang.org/x/term"
)

// InteractiveState holds the mutable state for interactive chart navigation.
type InteractiveState struct {
	AllSeries    []thanos.Series
	SeriesIndex  int
	ViewStart    int
	ViewEnd      int
	Query        string
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
		fmt.Print("\033[?25h") // show cursor
	}()

	fmt.Print("\033[?25l") // hide cursor

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
	termW := config.TerminalWidth()
	termH := config.TerminalHeight()

	chartHeight := termH - 6 // room for header + status bar
	if chartHeight < 5 {
		chartHeight = 5
	}

	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]

	opts := []asciigraph.Option{
		asciigraph.Height(chartHeight),
		asciigraph.Width(termW - 15),
	}

	if len(viewTimes) > 0 {
		caption := formatViewCaption(viewTimes, viewVals)
		opts = append(opts, asciigraph.Caption(caption))
	}

	chart := asciigraph.Plot(viewVals, opts...)

	// Build status bar
	labels := thanos.LabelSetString(cur.Labels)
	status := fmt.Sprintf("Series %d/%d %s | Samples %d-%d of %d | ←/→ pan  ↑/↓ zoom  Space/Bksp series  q quit",
		s.SeriesIndex+1, len(s.AllSeries), labels,
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)

	if len(status) > termW {
		status = status[:termW]
	}

	// Clear screen and draw
	var sb strings.Builder
	sb.WriteString("\033[2J\033[H") // clear + home
	fmt.Fprintf(&sb, "Query: %s\n", s.Query)
	sb.WriteString(chart)
	sb.WriteString("\n\n")
	sb.WriteString(status)

	fmt.Print(sb.String())
}

func formatViewCaption(times []time.Time, values []float64) string {
	if len(times) == 0 {
		return ""
	}

	start := times[0]
	end := times[len(times)-1]

	minVal, maxVal := minMax(values)

	return fmt.Sprintf("%s .. %s | min: %s  max: %s",
		start.Format("15:04:05"),
		end.Format("15:04:05"),
		humanNumber(minVal),
		humanNumber(maxVal),
	)
}
