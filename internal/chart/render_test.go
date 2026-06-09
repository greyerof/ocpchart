package chart

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/greyerof/ocpchart/internal/thanos"
)

func TestHumanNumber_Tera(t *testing.T) {
	if got := humanNumber(1.5e12); got != "1.50T" {
		t.Fatalf("expected 1.50T, got %s", got)
	}
}

func TestHumanNumber_Giga(t *testing.T) {
	if got := humanNumber(2.3e9); got != "2.30G" {
		t.Fatalf("expected 2.30G, got %s", got)
	}
}

func TestHumanNumber_Mega(t *testing.T) {
	if got := humanNumber(4.56e6); got != "4.56M" {
		t.Fatalf("expected 4.56M, got %s", got)
	}
}

func TestHumanNumber_Kilo(t *testing.T) {
	if got := humanNumber(1234.0); got != "1.23K" {
		t.Fatalf("expected 1.23K, got %s", got)
	}
}

func TestHumanNumber_Normal(t *testing.T) {
	if got := humanNumber(42.5); got != "42.50" {
		t.Fatalf("expected 42.50, got %s", got)
	}
}

func TestHumanNumber_Small(t *testing.T) {
	got := humanNumber(0.001)
	if got != "0.001000" {
		t.Fatalf("expected 0.001000, got %s", got)
	}
}

func TestHumanNumber_Zero(t *testing.T) {
	if got := humanNumber(0); got != "0.00" {
		t.Fatalf("expected 0.00, got %s", got)
	}
}

func TestHumanNumber_Negative(t *testing.T) {
	if got := humanNumber(-5e6); got != "-5.00M" {
		t.Fatalf("expected -5.00M, got %s", got)
	}
}

func TestMinMax(t *testing.T) {
	mn, mx := minMax([]float64{3, 1, 4, 1, 5, 9, 2, 6})
	if mn != 1 {
		t.Fatalf("expected min 1, got %f", mn)
	}

	if mx != 9 {
		t.Fatalf("expected max 9, got %f", mx)
	}
}

func TestMinMax_SingleValue(t *testing.T) {
	mn, mx := minMax([]float64{42})
	if mn != 42 || mx != 42 {
		t.Fatalf("expected 42/42, got %f/%f", mn, mx)
	}
}

func TestMinMax_NegativeValues(t *testing.T) {
	mn, mx := minMax([]float64{-5, -1, -10})
	if mn != -10 {
		t.Fatalf("expected min -10, got %f", mn)
	}

	if mx != -1 {
		t.Fatalf("expected max -1, got %f", mx)
	}
}

func TestPickTimeFormat_ShortDuration(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	got := pickTimeFormat(now, now.Add(30*time.Minute))
	if got != "15:04:05" {
		t.Fatalf("expected 15:04:05, got %s", got)
	}
}

func TestPickTimeFormat_MediumDuration(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	got := pickTimeFormat(now, now.Add(2*time.Hour))
	if got != "15:04" {
		t.Fatalf("expected 15:04, got %s", got)
	}
}

func TestPickTimeFormat_LongDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)
	got := pickTimeFormat(start, end)
	if got != "Jan 02 15:04" {
		t.Fatalf("expected 'Jan 02 15:04', got %s", got)
	}
}

func TestChooseYLabelScale_NoScaling(t *testing.T) {
	scale := chooseYLabelScale(0, 100)
	if scale.divisor != 0 {
		t.Fatalf("expected no scaling for small values, got divisor %f", scale.divisor)
	}
}

func TestChooseYLabelScale_Kilo(t *testing.T) {
	scale := chooseYLabelScale(0, 50000)
	if scale.suffix != "K" {
		t.Fatalf("expected K suffix, got %q", scale.suffix)
	}
}

func TestChooseYLabelScale_Mega(t *testing.T) {
	scale := chooseYLabelScale(0, 5e6)
	if scale.suffix != "M" {
		t.Fatalf("expected M suffix, got %q", scale.suffix)
	}
}

func TestChooseYLabelScale_Giga(t *testing.T) {
	scale := chooseYLabelScale(0, 5e9)
	if scale.suffix != "G" {
		t.Fatalf("expected G suffix, got %q", scale.suffix)
	}
}

func TestChooseYLabelScale_Tera(t *testing.T) {
	scale := chooseYLabelScale(0, 5e12)
	if scale.suffix != "T" {
		t.Fatalf("expected T suffix, got %q", scale.suffix)
	}
}

func TestFormatYLabel_NoScale(t *testing.T) {
	got := formatYLabel(42.5, yLabelScale{0, ""})
	if got != "42.500" {
		t.Fatalf("expected 42.500, got %s", got)
	}
}

func TestFormatYLabel_WithScale(t *testing.T) {
	got := formatYLabel(1500, yLabelScale{1e3, "K"})
	if got != "1.500K" {
		t.Fatalf("expected 1.500K, got %s", got)
	}
}

func TestRenderStatic_ProducesOutput(t *testing.T) {
	now := time.Now()
	s := thanos.Series{
		Labels: map[string]string{"instance": "worker-0"},
		Times:  makeTimes(now, 60, 30*time.Second),
		Values: makeSineValues(60),
	}

	result := RenderStatic(s, 80, 25, "test_query")
	if result == "" {
		t.Fatal("expected non-empty chart output")
	}

	lines := strings.Split(result, "\n")
	if len(lines) < 10 {
		t.Fatalf("expected at least 10 lines in chart, got %d", len(lines))
	}
}

func TestRenderStatic_ContainsXAxisTimestamps(t *testing.T) {
	now := time.Now()
	s := thanos.Series{
		Labels: map[string]string{},
		Times:  makeTimes(now, 60, 30*time.Second),
		Values: makeSineValues(60),
	}

	result := RenderStatic(s, 80, 25, "test_query")
	// The X-axis should contain the hour from the start time formatted in local timezone
	hourStr := now.Format("15:")
	if !strings.Contains(result, hourStr) {
		t.Fatalf("expected X-axis timestamp containing %q, got:\n%s", hourStr, result)
	}
}

func TestRenderStatic_RespectsWidthHeight(t *testing.T) {
	now := time.Now()
	s := thanos.Series{
		Labels: map[string]string{},
		Times:  makeTimes(now, 30, time.Minute),
		Values: makeSineValues(30),
	}

	narrow := RenderStatic(s, 80, 15, "test_query")
	wide := RenderStatic(s, 120, 30, "test_query")

	narrowLines := strings.Split(narrow, "\n")
	wideLines := strings.Split(wide, "\n")

	if len(wideLines) <= len(narrowLines) {
		t.Fatalf("taller chart should have more lines: narrow=%d, wide=%d", len(narrowLines), len(wideLines))
	}
}

func TestCenterText_Centered(t *testing.T) {
	got := centerText("hello", 11)
	if got != "   hello" {
		t.Fatalf("expected %q, got %q", "   hello", got)
	}
}

func TestCenterText_ExactWidth(t *testing.T) {
	got := centerText("hello", 5)
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestCenterText_TooWide(t *testing.T) {
	got := centerText("hello world", 5)
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func makeTimes(start time.Time, count int, step time.Duration) []time.Time {
	times := make([]time.Time, count)
	for i := range times {
		times[i] = start.Add(time.Duration(i) * step)
	}

	return times
}

func makeSineValues(count int) []float64 {
	vals := make([]float64, count)
	for i := range vals {
		vals[i] = math.Sin(float64(i) * 0.2)
	}

	return vals
}
