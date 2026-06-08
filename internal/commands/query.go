package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/greyerof/ocpchart/internal/chart"
	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/spf13/cobra"
)

var (
	flagQuerySince       time.Duration
	flagQueryUntil       string
	flagQueryStep        time.Duration
	flagQueryWidth       int
	flagQueryHeight      int
	flagQueryInteractive bool
)

var queryCmd = &cobra.Command{
	Use:   "query <promql>",
	Short: "Run a PromQL range query and render an ASCII chart",
	Long: `Executes a Prometheus range query against Thanos and displays the result
as an ASCII line chart. If the query returns multiple time series, you will
be prompted to select one (or navigate them in interactive mode).`,
	Example: `  ocpchart query 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 1h
  ocpchart query 'sum(up)' --since 30m --step 30s
  ocpchart query 'node_memory_MemAvailable_bytes' --since 2h -i`,
	Args: cobra.ExactArgs(1),
	RunE: runQuery,
}

func init() {
	f := queryCmd.Flags()
	f.DurationVar(&flagQuerySince, "since", 0, "how far back to query (required, e.g. 1h, 30m)")
	f.StringVar(&flagQueryUntil, "until", "", "end time (Go duration relative to now, or RFC3339; default: now)")
	f.DurationVar(&flagQueryStep, "step", 0, "query step (default: auto-calculated)")
	f.IntVar(&flagQueryWidth, "width", 0, "chart width in columns (default: terminal width)")
	f.IntVar(&flagQueryHeight, "height", 0, "chart height in rows (default: 20)")
	f.BoolVarP(&flagQueryInteractive, "interactive", "i", false, "interactive mode with pan/zoom/series navigation")

	_ = queryCmd.MarkFlagRequired("since")

	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	promql := args[0]

	client, err := resolveClient()
	if err != nil {
		return err
	}

	end := time.Now()
	if flagQueryUntil != "" {
		end, err = parseUntil(flagQueryUntil)
		if err != nil {
			return fmt.Errorf("invalid --until: %w", err)
		}
	}

	start := end.Add(-flagQuerySince)

	step := flagQueryStep
	if step == 0 {
		step = config.AutoStep(flagQuerySince, flagQueryWidth)
	}

	fmt.Fprintf(os.Stderr, "Querying: %s\n", promql)
	fmt.Fprintf(os.Stderr, "Range: %s to %s (step %s)\n\n",
		start.Format("15:04:05"), end.Format("15:04:05"), step)

	series, err := client.RangeQuery(context.Background(), promql, start, end, step)
	if err != nil {
		return err
	}

	if len(series) == 0 {
		return fmt.Errorf("query returned no data")
	}

	if flagQueryInteractive {
		return chart.RunInteractive(series, promql)
	}

	selected, err := selectSeries(series)
	if err != nil {
		return err
	}

	chart.PrintStatic(selected, flagQueryWidth, flagQueryHeight)

	return nil
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
	// Try as Go duration relative to now
	d, err := time.ParseDuration(s)
	if err == nil {
		return time.Now().Add(-d), nil
	}

	// Try as RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("expected Go duration (e.g. 5m) or RFC3339 timestamp, got %q", s)
}
