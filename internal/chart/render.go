package chart

import (
	"fmt"
	"math"
	"time"

	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/guptarohit/asciigraph"
)

const (
	defaultXAxisTickCount = 8
	minXAxisTickCount     = 2
	yAxisDefaultOffset    = 3
	minYAxisLabelWidth    = 4
	// extraRows: X-axis separator (1) + X-axis labels (1) + caption (2) + blank (1)
	extraRows             = 5
	yLabelFormatThreshold = 10000
	yLabelFormatPrecision = 3
)

type yLabelScale struct {
	divisor float64
	suffix  string
}

var yLabelScales = []yLabelScale{
	{1e12, "T"},
	{1e9, "G"},
	{1e6, "M"},
	{1e3, "K"},
}

// RenderStatic renders a single series as an ASCII chart and returns the string.
func RenderStatic(s thanos.Series, widthOverride, heightOverride int, query string) string {
	width := widthOverride
	if width <= 0 {
		width = config.TerminalWidth()
	}

	height := heightOverride
	if height <= 0 {
		height = config.DefaultHeight
	}

	caption := query + "\n" + thanos.LabelSetString(s.Labels)
	opts := buildChartOptions(s.Values, s.Times, width, height, caption)

	return asciigraph.Plot(s.Values, opts...)
}

// PrintStatic prints a single series chart to stdout with a header.
func PrintStatic(s thanos.Series, widthOverride, heightOverride int, query string) {
	labels := thanos.LabelSetString(s.Labels)
	fmt.Printf("Series: %s\n", labels)
	fmt.Printf("Points: %d | Range: %s to %s\n\n",
		len(s.Values),
		s.Times[0].Format("2006-01-02 15:04:05"),
		s.Times[len(s.Times)-1].Format("2006-01-02 15:04:05"),
	)
	fmt.Println(RenderStatic(s, widthOverride, heightOverride, query))
	fmt.Println()
}

// buildChartOptions computes plot dimensions and returns asciigraph options
// with proper X-axis timestamps and Y-axis formatting.
func buildChartOptions(values []float64, times []time.Time, totalWidth, totalHeight int, caption string) []asciigraph.Option {
	first := times[0]
	last := times[len(times)-1]

	firstUnix := float64(first.Unix())
	lastUnix := float64(last.Unix())

	tickCount := min(len(times), defaultXAxisTickCount)
	tickCount = max(tickCount, minXAxisTickCount)

	timeFmt := pickTimeFormat(first, last)

	minVal, maxVal := minMax(values)
	scale := chooseYLabelScale(minVal, maxVal)
	yLabelWidth := yAxisLabelWidth(values, scale)

	labelOverhang := (len(timeFmt) + 1) / 2
	plotWidth := totalWidth - yLabelWidth - yAxisDefaultOffset - labelOverhang
	if plotWidth < 10 {
		plotWidth = 10
	}

	plotHeight := totalHeight - extraRows
	if plotHeight < 5 {
		plotHeight = 5
	}

	return []asciigraph.Option{
		asciigraph.Height(plotHeight),
		asciigraph.Width(plotWidth),
		asciigraph.Caption(caption),
		asciigraph.XAxisRange(firstUnix, lastUnix),
		asciigraph.XAxisTickCount(tickCount),
		asciigraph.XAxisValueFormatter(func(v float64) string {
			return time.Unix(int64(v), 0).Format(timeFmt)
		}),
		asciigraph.YAxisValueFormatter(func(v float64) string {
			return formatYLabel(v, scale)
		}),
	}
}

func chooseYLabelScale(minVal, maxVal float64) yLabelScale {
	peak := max(math.Abs(minVal), math.Abs(maxVal))
	if peak < yLabelFormatThreshold {
		return yLabelScale{0, ""}
	}

	for _, scale := range yLabelScales {
		if peak >= scale.divisor {
			return scale
		}
	}

	return yLabelScale{0, ""}
}

func formatYLabel(v float64, scale yLabelScale) string {
	if scale.divisor == 0 {
		return fmt.Sprintf("%.*f", yLabelFormatPrecision, v)
	}

	return fmt.Sprintf("%.*f%s", yLabelFormatPrecision, v/scale.divisor, scale.suffix)
}

func yAxisLabelWidth(values []float64, scale yLabelScale) int {
	minVal, maxVal := minMax(values)
	w := max(len(formatYLabel(minVal, scale)), len(formatYLabel(maxVal, scale)))

	return max(w, minYAxisLabelWidth)
}

func pickTimeFormat(first, last time.Time) string {
	span := last.Sub(first)

	switch {
	case span < time.Hour:
		return "15:04:05"
	case span < 24*time.Hour:
		return "15:04"
	default:
		return "Jan 02 15:04"
	}
}

func minMax(vals []float64) (float64, float64) {
	mn, mx := vals[0], vals[0]

	for _, v := range vals[1:] {
		if v < mn {
			mn = v
		}

		if v > mx {
			mx = v
		}
	}

	return mn, mx
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
