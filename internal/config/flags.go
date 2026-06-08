package config

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	MinStep        = 15 * time.Second
	DefaultWidth   = 80
	DefaultHeight  = 25
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

// TerminalWidth returns the current terminal width, or DefaultWidth as a fallback.
func TerminalWidth() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return DefaultWidth
	}

	w, _, err := term.GetSize(fd)
	if err != nil || w <= 0 {
		return DefaultWidth
	}

	return w
}

// TerminalHeight returns the current terminal height, or DefaultHeight as a fallback.
func TerminalHeight() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return DefaultHeight
	}

	_, h, err := term.GetSize(fd)
	if err != nil || h <= 0 {
		return DefaultHeight
	}

	return max(h, DefaultHeight)
}

// TerminalSize returns both width and height in one call, with proper
// fallbacks. Used by interactive/live modes that need both dimensions.
func TerminalSize() (int, int) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return DefaultWidth, DefaultHeight
	}

	w, h, err := term.GetSize(fd)
	if err != nil || w <= 0 || h <= 0 {
		return DefaultWidth, DefaultHeight
	}

	return w, max(h, DefaultHeight)
}
