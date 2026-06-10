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
		showTerminalCursor()
	}()

	hideTerminalCursor()

	state, lastRefresh, err := initLiveState(ctx, opts, oldState)
	if err != nil {
		return err
	}

	// The refresh goroutine and input loop both mutate/read `state` and `lastRefresh`.
	// This mutex keeps those operations serialized so we render consistent frames.
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startLiveRefreshLoop(ctx, opts, &mu, state, &lastRefresh)

	mu.Lock()
	renderLiveFrame(state, lastRefresh, opts.Refresh)
	mu.Unlock()

	return runLiveInputLoop(&mu, state, &lastRefresh, opts.Refresh)
}

// initLiveState fetches initial data and constructs the live interactive state.
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

// startLiveRefreshLoop starts the background ticker loop for data refresh.
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

// refreshLiveState fetches fresh data and redraws the frame when data is valid.
func refreshLiveState(ctx context.Context, opts LiveOptions, mu *sync.Mutex, state *InteractiveState, lastRefresh *time.Time) {
	newSeries, fetchErr := fetchLive(ctx, opts)
	// Keep the current frame if refresh fails; next ticks will retry automatically.
	if fetchErr != nil || len(newSeries) == 0 {
		return
	}

	mu.Lock()
	updateLiveSeries(state, newSeries)
	*lastRefresh = time.Now()
	renderLiveFrame(state, *lastRefresh, opts.Refresh)
	mu.Unlock()
}

// updateLiveSeries replaces the current series set while preserving valid index state.
func updateLiveSeries(state *InteractiveState, series []thanos.Series) {
	prevIdx := state.SeriesIndex
	state.AllSeries = series
	if prevIdx >= len(series) {
		state.SeriesIndex = 0
	}
	state.resetViewport()
}

// runLiveInputLoop handles keyboard interaction for live mode.
func runLiveInputLoop(mu *sync.Mutex, state *InteractiveState, lastRefresh *time.Time, refreshInterval time.Duration) error {
	buf := make([]byte, 3)

	for {
		n, readErr := os.Stdin.Read(buf)
		if readErr != nil {
			// Treat stdin closure/read failures as a graceful exit from live mode.
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

// fetchLive executes a range query for the current rolling live window.
func fetchLive(ctx context.Context, opts LiveOptions) ([]thanos.Series, error) {
	now := time.Now()
	start := now.Add(-opts.Since)

	return opts.Client.RangeQuery(ctx, opts.Query, start, now, opts.Step)
}

// renderLiveFrame renders one full-screen frame for live mode.
func renderLiveFrame(s *InteractiveState, lastRefresh time.Time, refreshInterval time.Duration) {
	termW, termH := config.TerminalSize()

	graph := liveGraph(s, termW, termH)
	status := centerText(liveStatusLine(s, lastRefresh, refreshInterval), termW)
	controls := "  " + controlsHint

	var sb strings.Builder
	clearScreenAndMoveHome(&sb)
	sb.WriteString(graph)
	sb.WriteString("\r\n")
	moveCursor(&sb, termH-1, 1)
	writePaddedLine(&sb, termW, status)
	moveCursor(&sb, termH, 1)
	writePaddedLine(&sb, termW, controls)

	fmt.Print(sb.String())
}

// liveGraph builds the chart body rendered in live mode (title + plot).
func liveGraph(s *InteractiveState, termW, termH int) string {
	cur := s.currentSeries()
	viewVals := cur.Values[s.ViewStart:s.ViewEnd]
	viewTimes := cur.Times[s.ViewStart:s.ViewEnd]
	caption := thanos.LabelSetString(cur.Labels)

	opts := buildChartOptions(viewVals, viewTimes, termW, termH-3, caption)
	title := centerText(s.Query, termW)
	return title + "\r\n" + toCRLF(asciigraph.Plot(viewVals, opts...))
}

// liveStatusLine formats the status line shown above live mode controls.
func liveStatusLine(s *InteractiveState, lastRefresh time.Time, refreshInterval time.Duration) string {
	cur := s.currentSeries()
	untilRefresh := time.Until(lastRefresh.Add(refreshInterval)).Truncate(time.Second)
	if untilRefresh < 0 {
		untilRefresh = 0
	}

	return fmt.Sprintf("LIVE (refresh %s, next in %s) | Series %d/%d | Samples %d-%d of %d",
		refreshInterval, untilRefresh,
		s.SeriesIndex+1, len(s.AllSeries),
		s.ViewStart+1, s.ViewEnd, len(cur.Values),
	)
}
