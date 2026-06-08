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

	state, lastRefresh, err := initLiveState(ctx, opts, oldState)
	if err != nil {
		return err
	}

	var mu sync.Mutex
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startLiveRefreshLoop(ctx, opts, &mu, state, &lastRefresh)

	mu.Lock()
	renderLiveFrame(state, lastRefresh, opts.Refresh)
	mu.Unlock()

	return runLiveInputLoop(&mu, state, &lastRefresh, opts.Refresh)
}

func initLiveState(ctx context.Context, opts LiveOptions, oldState *term.State) (*InteractiveState, time.Time, error) {
	series, err := fetchLive(ctx, opts)
	if err != nil {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		return nil, time.Time{}, fmt.Errorf("initial query failed: %w", err)
	}

	if len(series) == 0 {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
		return nil, time.Time{}, fmt.Errorf("query returned no data")
	}

	return NewInteractiveState(series, opts.Query), time.Now(), nil
}

func startLiveRefreshLoop(ctx context.Context, opts LiveOptions, mu *sync.Mutex, state *InteractiveState, lastRefresh *time.Time) {
	go func() {
		ticker := time.NewTicker(opts.Refresh)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshLiveState(ctx, opts, mu, state, lastRefresh)
			}
		}
	}()
}

func refreshLiveState(ctx context.Context, opts LiveOptions, mu *sync.Mutex, state *InteractiveState, lastRefresh *time.Time) {
	newSeries, fetchErr := fetchLive(ctx, opts)
	if fetchErr != nil || len(newSeries) == 0 {
		return
	}

	mu.Lock()
	updateLiveSeries(state, newSeries)
	*lastRefresh = time.Now()
	renderLiveFrame(state, *lastRefresh, opts.Refresh)
	mu.Unlock()
}

func updateLiveSeries(state *InteractiveState, series []thanos.Series) {
	prevIdx := state.SeriesIndex
	state.AllSeries = series
	if prevIdx >= len(series) {
		state.SeriesIndex = 0
	}
	state.resetViewport()
}

func runLiveInputLoop(mu *sync.Mutex, state *InteractiveState, lastRefresh *time.Time, refreshInterval time.Duration) error {
	buf := make([]byte, 3)

	for {
		n, readErr := os.Stdin.Read(buf)
		if readErr != nil {
			return nil
		}

		mu.Lock()

		if shouldQuit, handled := handleInteractiveSingleByteInput(state, n, buf); shouldQuit {
			mu.Unlock()
			return nil
		} else if !handled {
			handleInteractiveArrowInput(state, n, buf)
		}

		renderLiveFrame(state, *lastRefresh, refreshInterval)
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

	graph := liveGraph(s, termW, termH)
	status := liveStatusLine(s, lastRefresh, refreshInterval)
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

func liveGraph(s *InteractiveState, termW, termH int) string {
	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]
	caption := s.Query + "\n" + thanos.LabelSetString(cur.Labels)

	opts := buildChartOptions(viewVals, viewTimes, termW, termH-2, caption)
	return strings.ReplaceAll(asciigraph.Plot(viewVals, opts...), "\n", "\r\n")
}

func liveStatusLine(s *InteractiveState, lastRefresh time.Time, refreshInterval time.Duration) string {
	cur := s.currentSeries()
	labels := thanos.LabelSetString(cur.Labels)
	untilRefresh := time.Until(lastRefresh.Add(refreshInterval)).Truncate(time.Second)
	if untilRefresh < 0 {
		untilRefresh = 0
	}

	return fmt.Sprintf("  LIVE (refresh %s, next in %s) | Series %d/%d %s | Samples %d-%d of %d",
		refreshInterval, untilRefresh,
		s.SeriesIndex+1, len(s.AllSeries), labels,
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)
}
