package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/greyerof/ocpchart/internal/chart"
	"github.com/greyerof/ocpchart/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagLiveSince   time.Duration
	flagLiveStep    time.Duration
	flagLiveRefresh time.Duration
)

var liveCmd = &cobra.Command{
	Use:   "live <promql>",
	Short: "Live-refresh ASCII chart of a PromQL query",
	Long: `Runs a PromQL range query in a loop, refreshing the ASCII chart at a
configurable interval. The chart is always interactive: use arrow keys to
pan/zoom and Space/Backspace to switch between time series.`,
	Example: `  ocpchart live 'rate(node_cpu_seconds_total{mode="idle"}[5m])' --since 30m
  ocpchart live 'sum(up)' --since 1h --refresh 10s
  ocpchart live 'node_memory_MemAvailable_bytes' --since 15m --step 15s`,
	Args: cobra.ExactArgs(1),
	RunE: runLive,
}

// init registers live command flags and adds the command to root.
func init() {
	f := liveCmd.Flags()
	f.DurationVar(&flagLiveSince, "since", 0, "rolling window size (required, e.g. 30m, 1h)")
	f.DurationVar(&flagLiveStep, "step", 0, "query step (default: auto-calculated)")
	f.DurationVar(&flagLiveRefresh, "refresh", config.DefaultRefresh, "refresh interval (e.g. 10s, 1m)")

	_ = liveCmd.MarkFlagRequired("since")

	rootCmd.AddCommand(liveCmd)
}

// runLive starts interactive live-refresh chart mode for a PromQL query.
func runLive(cmd *cobra.Command, args []string) error {
	promql := args[0]

	client, err := resolveClient()
	if err != nil {
		return err
	}

	step := flagLiveStep
	if step == 0 {
		step = config.AutoStep(flagLiveSince, 0)
	}

	fmt.Fprintf(os.Stderr, "Live mode: %s (window %s, step %s, refresh %s)\n",
		promql, flagLiveSince, step, flagLiveRefresh)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return chart.RunLive(ctx, chart.LiveOptions{
		Client:  client,
		Query:   promql,
		Since:   flagLiveSince,
		Step:    step,
		Refresh: flagLiveRefresh,
	})
}
