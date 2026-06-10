package main

import (
	"os"

	"github.com/greyerof/ocpchart/internal/commands"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
