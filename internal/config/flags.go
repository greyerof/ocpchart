package config

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	MinStep        = 15 * time.Second
	DefaultHeight  = 20
	DefaultRefresh = 30 * time.Second
)

// ValidateAuthFlags checks that auth flags form a valid combination:
// either --kubeconfig alone, or --token + --thanos-url together.
func ValidateAuthFlags(kubeconfig, thanosURL, token string) error {
	hasKubeconfig := kubeconfig != ""
	hasManual := thanosURL != "" || token != ""

	if hasKubeconfig && hasManual {
		return fmt.Errorf("--kubeconfig cannot be combined with --token or --thanos-url")
	}

	if !hasKubeconfig && !hasManual {
		if os.Getenv("KUBECONFIG") != "" {
			return nil
		}

		return fmt.Errorf("provide --kubeconfig, or both --token and --thanos-url")
	}

	if hasManual && (thanosURL == "" || token == "") {
		return fmt.Errorf("--token and --thanos-url must be used together")
	}

	return nil
}

// AutoStep calculates a step duration that produces datapoints fitting the terminal width.
func AutoStep(since time.Duration, widthOverride int) time.Duration {
	width := widthOverride
	if width <= 0 {
		width = TerminalWidth()
	}

	points := width - 10
	if points < 20 {
		points = 20
	}

	step := since / time.Duration(points)
	if step < MinStep {
		step = MinStep
	}

	return step
}

// TerminalWidth returns the current terminal width, or 80 as a fallback.
func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}

	return w
}

// TerminalHeight returns the current terminal height, or 24 as a fallback.
func TerminalHeight() int {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return 24
	}

	return h
}
