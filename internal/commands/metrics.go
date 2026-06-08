package commands

import (
	"context"
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

var flagMetricFilter string

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Discover available metrics",
}

var metricsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available metric names from the cluster",
	Long: `Fetches all metric names from Thanos and prints them sorted.
Use --filter to match a subset with a regular expression.`,
	Example: `  ocpchart metrics list
  ocpchart metrics list --filter "cpu"
  ocpchart metrics list --filter "node_network.*bytes"`,
	RunE: runMetricsList,
}

func init() {
	metricsListCmd.Flags().StringVarP(&flagMetricFilter, "filter", "f", "", "regex to filter metric names")

	metricsCmd.AddCommand(metricsListCmd)
	rootCmd.AddCommand(metricsCmd)
}

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
