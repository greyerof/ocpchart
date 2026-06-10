package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd builds the "version" subcommand, which prints the build version
// and commit hash injected at compile time via ldflags.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ocpchart %s (commit: %s)\n", version, commit)
		},
	}
}
