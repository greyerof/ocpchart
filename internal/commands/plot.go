package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/greyerof/ocpchart/internal/chart"
	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/spf13/cobra"
)

var (
	flagPlotSince   time.Duration
	flagPlotUntil   string
	flagPlotStep    time.Duration
	flagPlotWidth   int
	flagPlotHeight  int
	flagPlotOnce    bool
	flagPlotRefresh time.Duration
)

var plotCmd = &cobra.Command{
	Use:   "plot <promql>",
	Short: "Run a PromQL query and render an interactive ASCII chart",
	Long: `Executes a Prometheus range query against Thanos and displays the result
as an interactive ASCII line chart with pan/zoom and series navigation.

Use --once for a static one-shot chart, or --refresh for a live auto-refreshing
chart with a rolling time window.`,
	Example: `  # Interactive chart (default)
  ocpchart plot 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 1h

  # Static one-shot chart
  ocpchart plot 'node_memory_MemAvailable_bytes' --since 2h --once

  # Live-refresh chart
  ocpchart plot 'sum(up)' --since 30m --refresh 10s`,
	Args: cobra.ExactArgs(1),
	RunE: runPlot,
}

func init() {
	f := plotCmd.Flags()
	f.DurationVar(&flagPlotSince, "since", 0, "how far back to query (required, e.g. 1h, 30m)")
	f.StringVar(&flagPlotUntil, "until", "", "end time (Go duration relative to now, or RFC3339; default: now)")
	f.DurationVar(&flagPlotStep, "step", 0, "query step (default: auto-calculated)")
	f.IntVar(&flagPlotWidth, "width", 0, "chart width in columns (default: terminal width; only with --once)")
	f.IntVar(&flagPlotHeight, "height", 0, "chart height in rows (default: 20; only with --once)")
	f.BoolVar(&flagPlotOnce, "once", false, "fetch once and print a static chart (non-interactive)")
	f.DurationVar(&flagPlotRefresh, "refresh", 0, "auto-refresh interval for live mode (e.g. 30s, 1m)")

	_ = plotCmd.MarkFlagRequired("since")

	rootCmd.AddCommand(plotCmd)
}

func runPlot(cmd *cobra.Command, args []string) error {
	if flagPlotOnce && flagPlotRefresh > 0 {
		return fmt.Errorf("--once and --refresh are mutually exclusive")
	}

	if flagPlotUntil != "" && flagPlotRefresh > 0 {
		return fmt.Errorf("--until and --refresh are mutually exclusive")
	}

	promql := args[0]

	client, err := resolveClient()
	if err != nil {
		return err
	}

	if flagPlotRefresh > 0 {
		return runPlotLive(promql, client)
	}

	return runPlotQuery(promql, client)
}

func runPlotQuery(promql string, client *thanos.Client) error {
	start, end, step, err := buildPlotWindow()
	if err != nil {
		return err
	}

	if flagPlotOnce {
		fmt.Fprintf(os.Stderr, "Querying: %s\n", promql)
		fmt.Fprintf(os.Stderr, "Range: %s to %s (step %s)\n\n",
			start.Format("15:04:05"), end.Format("15:04:05"), step)
	}

	series, err := client.RangeQuery(context.Background(), promql, start, end, step)
	if err != nil {
		return err
	}

	if len(series) == 0 {
		return fmt.Errorf("query returned no data")
	}

	if flagPlotOnce {
		selected, err := selectSeries(series)
		if err != nil {
			return err
		}

		chart.PrintStatic(selected, flagPlotWidth, flagPlotHeight, promql)

		return nil
	}

	return chart.RunInteractive(series, promql)
}

func runPlotLive(promql string, client *thanos.Client) error {
	step := flagPlotStep
	if step == 0 {
		step = config.AutoStep(flagPlotSince, 0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return chart.RunLive(ctx, chart.LiveOptions{
		Client:  client,
		Query:   promql,
		Since:   flagPlotSince,
		Step:    step,
		Refresh: flagPlotRefresh,
	})
}

func buildPlotWindow() (time.Time, time.Time, time.Duration, error) {
	end := time.Now()
	if flagPlotUntil != "" {
		parsed, err := parseUntil(flagPlotUntil)
		if err != nil {
			return time.Time{}, time.Time{}, 0, fmt.Errorf("invalid --until: %w", err)
		}

		end = parsed
	}

	step := flagPlotStep
	if step == 0 {
		step = config.AutoStep(flagPlotSince, flagPlotWidth)
	}

	start := end.Add(-flagPlotSince)

	return start, end, step, nil
}

func selectSeries(series []thanos.Series) (thanos.Series, error) {
	if len(series) == 1 {
		return series[0], nil
	}

	fmt.Fprintf(os.Stderr, "Query returned %d time series:\n\n", len(series))

	for i, s := range series {
		fmt.Fprintf(os.Stderr, "  [%d] %s (%d points)\n", i+1, thanos.LabelSetString(s.Labels), len(s.Values))
	}

	fmt.Fprintf(os.Stderr, "\nSelect series [1-%d]: ", len(series))

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return thanos.Series{}, fmt.Errorf("reading selection: %w", err)
	}

	idx, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || idx < 1 || idx > len(series) {
		return thanos.Series{}, fmt.Errorf("invalid selection: %s", strings.TrimSpace(line))
	}

	return series[idx-1], nil
}

func parseUntil(s string) (time.Time, error) {
	d, err := time.ParseDuration(s)
	if err == nil {
		return time.Now().Add(-d), nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected Go duration (e.g. 5m) or RFC3339 timestamp, got %q", s)
}
