package chart

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/guptarohit/asciigraph"
	"golang.org/x/term"
)

// LiveOptions configures the live-refresh chart loop.
type LiveOptions struct {
	Client  *thanos.Client
	Query   string
	Since   time.Duration
	Step    time.Duration
	Refresh time.Duration
}

// RunLive enters a full-screen interactive chart that auto-refreshes.
func RunLive(ctx context.Context, opts LiveOptions) error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("entering raw mode: %w", err)
	}
	defer func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		fmt.Print("\033[?25h")
	}()

	fmt.Print("\033[?25l")

	series, err := fetchLive(ctx, opts)
	if err != nil {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		return fmt.Errorf("initial query failed: %w", err)
	}

	if len(series) == 0 {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		return fmt.Errorf("query returned no data")
	}

	state := NewInteractiveState(series, opts.Query)
	lastRefresh := time.Now()

	var mu sync.Mutex

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(opts.Refresh)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				newSeries, fetchErr := fetchLive(ctx, opts)
				if fetchErr != nil || len(newSeries) == 0 {
					continue
				}

				mu.Lock()
				prevIdx := state.SeriesIndex
				state.AllSeries = newSeries

				if prevIdx >= len(newSeries) {
					state.SeriesIndex = 0
				}

				state.resetViewport()
				lastRefresh = time.Now()
				renderLiveFrame(state, lastRefresh, opts.Refresh)
				mu.Unlock()
			}
		}
	}()

	mu.Lock()
	renderLiveFrame(state, lastRefresh, opts.Refresh)
	mu.Unlock()

	buf := make([]byte, 3)

	for {
		n, readErr := os.Stdin.Read(buf)
		if readErr != nil {
			return nil
		}

		mu.Lock()

		if n == 1 {
			switch buf[0] {
			case 'q', 'Q', 3:
				mu.Unlock()
				return nil
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
		}

		if n == 3 && buf[0] == 27 && buf[1] == 91 {
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

		renderLiveFrame(state, lastRefresh, opts.Refresh)
		mu.Unlock()
	}
}

func fetchLive(ctx context.Context, opts LiveOptions) ([]thanos.Series, error) {
	now := time.Now()
	start := now.Add(-opts.Since)

	return opts.Client.RangeQuery(ctx, opts.Query, start, now, opts.Step)
}

func renderLiveFrame(s *InteractiveState, lastRefresh time.Time, refreshInterval time.Duration) {
	termW, termH := config.TerminalSize()

	// Reserve 2 rows: status + controls
	chartHeight := termH - 2

	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]

	caption := thanos.LabelSetString(cur.Labels)
	opts := buildChartOptions(viewVals, viewTimes, termW, chartHeight, caption)

	graph := strings.ReplaceAll(
		asciigraph.Plot(viewVals, opts...),
		"\n", "\r\n",
	)

	labels := thanos.LabelSetString(cur.Labels)
	nextRefresh := lastRefresh.Add(refreshInterval)
	untilRefresh := time.Until(nextRefresh).Truncate(time.Second)

	if untilRefresh < 0 {
		untilRefresh = 0
	}

	status := fmt.Sprintf("  LIVE (refresh %s, next in %s) | Series %d/%d %s | Samples %d-%d of %d",
		refreshInterval, untilRefresh,
		s.SeriesIndex+1, len(s.AllSeries), labels,
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)

	controls := "  \u2190/\u2192 pan  \u2191/\u2193 zoom  Space/Bksp series  g goto  q quit"

	if len(status) > termW {
		status = status[:termW]
	}

	var sb strings.Builder
	sb.WriteString("\033[2J\033[H")
	sb.WriteString(graph)
	sb.WriteString("\r\n")
	fmt.Fprintf(&sb, "\033[%d;1H", termH-1)
	fmt.Fprintf(&sb, "%-*s", termW, status)
	fmt.Fprintf(&sb, "\033[%d;1H", termH)
	fmt.Fprintf(&sb, "%-*s", termW, controls)

	fmt.Print(sb.String())
}
