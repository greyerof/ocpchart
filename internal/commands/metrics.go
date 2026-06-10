package commands

import (
	"context"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

var flagMetricFilter string

// newMetricsCmd builds the "metrics" subcommand group and its children.
func newMetricsCmd() *cobra.Command {
	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Discover available metrics",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available metric names from the cluster",
		Long: `Fetches all metric names from Thanos and prints them sorted.
Use --filter to match a subset with a regular expression.`,
		Example: `  ocpchart metrics list
  ocpchart metrics list --filter "cpu"
  ocpchart metrics list --filter "node_network.*bytes"`,
		RunE: runMetricsList,
	}

	listCmd.Flags().StringVarP(&flagMetricFilter, "filter", "f", "", "regex to filter metric names")

	metricsCmd.AddCommand(listCmd)

	return metricsCmd
}

// runMetricsList fetches all Prometheus metric names from the cluster and prints
// them to stdout, optionally filtering by a user-supplied regular expression.
func runMetricsList(cmd *cobra.Command, args []string) error {
	client, err := resolveClient()
	if err != nil {
		return err
	}

	names, err := client.MetricNames(context.Background())
	if err != nil {
		return err
	}

	var re *regexp.Regexp
	if flagMetricFilter != "" {
		re, err = regexp.Compile(flagMetricFilter)
		if err != nil {
			return fmt.Errorf("invalid filter regex: %w", err)
		}
	}

	count := 0
	for _, name := range names {
		if re != nil && !re.MatchString(name) {
			continue
		}

		fmt.Println(name)
		count++
	}

	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "\n%d metric(s) found\n", count)

	return nil
}
