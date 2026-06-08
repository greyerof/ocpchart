package main

import (
	"os"

	"github.com/greyerof/ocpchart/internal/commands"
)

// main executes the CLI and exits non-zero on command failure.
func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
