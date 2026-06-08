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
	min, max := minMax([]float64{3, 1, 4, 1, 5, 9, 2, 6})
	if min != 1 {
		t.Fatalf("expected min 1, got %f", min)
	}

	if max != 9 {
		t.Fatalf("expected max 9, got %f", max)
	}
}

func TestMinMax_SingleValue(t *testing.T) {
	min, max := minMax([]float64{42})
	if min != 42 || max != 42 {
		t.Fatalf("expected 42/42, got %f/%f", min, max)
	}
}

func TestMinMax_NegativeValues(t *testing.T) {
	min, max := minMax([]float64{-5, -1, -10})
	if min != -10 {
		t.Fatalf("expected min -10, got %f", min)
	}

	if max != -1 {
		t.Fatalf("expected max -1, got %f", max)
	}
}

func TestFormatCaption_Empty(t *testing.T) {
	s := thanos.Series{}
	if got := formatCaption(s); got != "" {
		t.Fatalf("expected empty caption, got %q", got)
	}
}

func TestFormatCaption_ShortDuration(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := thanos.Series{
		Times:  []time.Time{now, now.Add(30 * time.Minute)},
		Values: []float64{1.0, 2.0},
	}
	got := formatCaption(s)
	if !strings.Contains(got, "10:00:00") || !strings.Contains(got, "10:30:00") {
		t.Fatalf("expected HH:MM:SS format for short durations, got %q", got)
	}
}

func TestFormatCaption_MediumDuration(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := thanos.Series{
		Times:  []time.Time{now, now.Add(2 * time.Hour)},
		Values: []float64{1.0, 2.0},
	}
	got := formatCaption(s)
	if !strings.Contains(got, "10:00") {
		t.Fatalf("expected HH:MM format for medium durations, got %q", got)
	}
}

func TestFormatCaption_LongDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)
	s := thanos.Series{
		Times:  []time.Time{start, end},
		Values: []float64{1.0, 2.0},
	}
	got := formatCaption(s)
	if !strings.Contains(got, "Jan 01") {
		t.Fatalf("expected Mon DD HH:MM format for long durations, got %q", got)
	}
}

func TestFormatCaption_IncludesMinMax(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := thanos.Series{
		Times:  []time.Time{now, now.Add(10 * time.Minute)},
		Values: []float64{100.0, 200.0},
	}
	got := formatCaption(s)
	if !strings.Contains(got, "min: 100.00") || !strings.Contains(got, "max: 200.00") {
		t.Fatalf("expected min/max in caption, got %q", got)
	}
}

func TestRenderStatic_ProducesOutput(t *testing.T) {
	now := time.Now()
	s := thanos.Series{
		Labels: map[string]string{"instance": "worker-0"},
		Times:  makeTimes(now, 60, 30*time.Second),
		Values: makeSineValues(60),
	}

	result := RenderStatic(s, 80, 15)
	if result == "" {
		t.Fatal("expected non-empty chart output")
	}

	lines := strings.Split(result, "\n")
	if len(lines) < 10 {
		t.Fatalf("expected at least 10 lines in chart, got %d", len(lines))
	}
}

func TestRenderStatic_RespectsWidthHeight(t *testing.T) {
	now := time.Now()
	s := thanos.Series{
		Labels: map[string]string{},
		Times:  makeTimes(now, 30, time.Minute),
		Values: makeSineValues(30),
	}

	narrow := RenderStatic(s, 40, 10)
	wide := RenderStatic(s, 120, 25)

	narrowLines := strings.Split(narrow, "\n")
	wideLines := strings.Split(wide, "\n")

	if len(wideLines) <= len(narrowLines) {
		t.Fatalf("taller chart should have more lines: narrow=%d, wide=%d", len(narrowLines), len(wideLines))
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
