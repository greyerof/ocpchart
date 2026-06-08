package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ocpchart %s (commit: %s)\n", version, commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
