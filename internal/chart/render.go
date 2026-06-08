package chart

import (
	"fmt"
	"math"
	"time"

	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/guptarohit/asciigraph"
)

// RenderStatic renders a single series as an ASCII chart and returns the string.
func RenderStatic(s thanos.Series, widthOverride, heightOverride int) string {
	width := widthOverride
	if width <= 0 {
		width = config.TerminalWidth()
	}

	height := heightOverride
	if height <= 0 {
		height = config.DefaultHeight
	}

	opts := buildOptions(s, width, height)

	return asciigraph.Plot(s.Values, opts...)
}

// PrintStatic prints a single series chart to stdout with a header.
func PrintStatic(s thanos.Series, widthOverride, heightOverride int) {
	labels := thanos.LabelSetString(s.Labels)
	fmt.Printf("Series: %s\n", labels)
	fmt.Printf("Points: %d | Range: %s to %s\n\n",
		len(s.Values),
		s.Times[0].Format("2006-01-02 15:04:05"),
		s.Times[len(s.Times)-1].Format("2006-01-02 15:04:05"),
	)
	fmt.Println(RenderStatic(s, widthOverride, heightOverride))
	fmt.Println()
}

func buildOptions(s thanos.Series, width, height int) []asciigraph.Option {
	opts := []asciigraph.Option{
		asciigraph.Height(height),
		asciigraph.Width(width - 15), // leave room for Y-axis labels
	}

	caption := formatCaption(s)
	if caption != "" {
		opts = append(opts, asciigraph.Caption(caption))
	}

	return opts
}

func formatCaption(s thanos.Series) string {
	if len(s.Times) == 0 {
		return ""
	}

	start := s.Times[0]
	end := s.Times[len(s.Times)-1]
	dur := end.Sub(start)

	var timeFormat string
	switch {
	case dur < time.Hour:
		timeFormat = "15:04:05"
	case dur < 24*time.Hour:
		timeFormat = "15:04"
	default:
		timeFormat = "Jan 02 15:04"
	}

	minVal, maxVal := minMax(s.Values)

	return fmt.Sprintf("%s .. %s | min: %s  max: %s",
		start.Format(timeFormat),
		end.Format(timeFormat),
		humanNumber(minVal),
		humanNumber(maxVal),
	)
}

func minMax(vals []float64) (float64, float64) {
	min, max := vals[0], vals[0]

	for _, v := range vals[1:] {
		if v < min {
			min = v
		}

		if v > max {
			max = v
		}
	}

	return min, max
}

func humanNumber(v float64) string {
	abs := math.Abs(v)

	switch {
	case abs >= 1e12:
		return fmt.Sprintf("%.2fT", v/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("%.2fG", v/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%.2fM", v/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("%.2fK", v/1e3)
	case abs < 0.01 && abs > 0:
		return fmt.Sprintf("%.6f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}
