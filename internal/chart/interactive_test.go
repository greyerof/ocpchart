package chart

import (
	"testing"
	"time"

	"github.com/greyerof/ocpchart/internal/thanos"
)

func newTestState(seriesCount, pointsPerSeries int) *InteractiveState {
	now := time.Now()
	allSeries := make([]thanos.Series, seriesCount)

	for i := range allSeries {
		allSeries[i] = thanos.Series{
			Labels: map[string]string{"series": string(rune('A' + i))},
			Times:  makeTimes(now, pointsPerSeries, 30*time.Second),
			Values: makeSineValues(pointsPerSeries),
		}
	}

	return NewInteractiveState(allSeries, "test_query")
}

func TestNewInteractiveState_InitialViewport(t *testing.T) {
	s := newTestState(2, 100)

	if s.SeriesIndex != 0 {
		t.Fatalf("expected series index 0, got %d", s.SeriesIndex)
	}

	if s.ViewStart != 0 {
		t.Fatalf("expected view start 0, got %d", s.ViewStart)
	}

	if s.ViewEnd != 100 {
		t.Fatalf("expected view end 100, got %d", s.ViewEnd)
	}
}

func TestInteractiveState_PanRight(t *testing.T) {
	s := newTestState(1, 100)
	// Zoom in first so there's room to pan
	s.ViewStart = 10
	s.ViewEnd = 60

	s.panRight()

	if s.ViewStart <= 10 {
		t.Fatal("expected ViewStart to increase after panRight")
	}
}

func TestInteractiveState_PanRight_AtEnd(t *testing.T) {
	s := newTestState(1, 100)
	// Already at end, should not go past
	for i := 0; i < 50; i++ {
		s.panRight()
	}

	if s.ViewEnd > 100 {
		t.Fatalf("ViewEnd should not exceed data length, got %d", s.ViewEnd)
	}
}

func TestInteractiveState_PanLeft(t *testing.T) {
	s := newTestState(1, 100)
	// Set a viewport in the middle so there's room to pan left
	s.ViewStart = 30
	s.ViewEnd = 80

	s.panLeft()
	if s.ViewStart >= 30 {
		t.Fatal("expected ViewStart to decrease after panLeft")
	}
}

func TestInteractiveState_PanLeft_AtStart(t *testing.T) {
	s := newTestState(1, 100)
	s.panLeft()

	if s.ViewStart != 0 {
		t.Fatalf("ViewStart should not go below 0, got %d", s.ViewStart)
	}
}

func TestInteractiveState_ZoomIn(t *testing.T) {
	s := newTestState(1, 100)
	prevLen := s.viewLength()

	s.zoomIn()

	if s.viewLength() >= prevLen {
		t.Fatalf("viewport should shrink on zoom in: was %d, now %d", prevLen, s.viewLength())
	}
}

func TestInteractiveState_ZoomIn_MinimumWindow(t *testing.T) {
	s := newTestState(1, 6) // very small dataset
	s.ViewStart = 0
	s.ViewEnd = 6

	s.zoomIn()
	// Should not zoom below 5 data points
	if s.viewLength() < 5 {
		t.Fatalf("viewport should not shrink below 5, got %d", s.viewLength())
	}
}

func TestInteractiveState_ZoomOut(t *testing.T) {
	s := newTestState(1, 100)
	s.zoomIn()
	s.zoomIn()
	prevLen := s.viewLength()

	s.zoomOut()

	if s.viewLength() <= prevLen {
		t.Fatalf("viewport should grow on zoom out: was %d, now %d", prevLen, s.viewLength())
	}
}

func TestInteractiveState_ZoomOut_ClampsToData(t *testing.T) {
	s := newTestState(1, 100)
	// Zoom out when already showing everything
	s.zoomOut()

	if s.ViewStart < 0 {
		t.Fatalf("ViewStart should not go below 0, got %d", s.ViewStart)
	}

	if s.ViewEnd > 100 {
		t.Fatalf("ViewEnd should not exceed data length, got %d", s.ViewEnd)
	}
}

func TestInteractiveState_NextSeries(t *testing.T) {
	s := newTestState(3, 50)

	s.nextSeries()
	if s.SeriesIndex != 1 {
		t.Fatalf("expected series 1, got %d", s.SeriesIndex)
	}

	s.nextSeries()
	if s.SeriesIndex != 2 {
		t.Fatalf("expected series 2, got %d", s.SeriesIndex)
	}

	// At last series, should not advance
	s.nextSeries()
	if s.SeriesIndex != 2 {
		t.Fatalf("expected series 2 (clamped), got %d", s.SeriesIndex)
	}
}

func TestInteractiveState_PrevSeries(t *testing.T) {
	s := newTestState(3, 50)
	s.SeriesIndex = 2
	s.resetViewport()

	s.prevSeries()
	if s.SeriesIndex != 1 {
		t.Fatalf("expected series 1, got %d", s.SeriesIndex)
	}

	s.prevSeries()
	if s.SeriesIndex != 0 {
		t.Fatalf("expected series 0, got %d", s.SeriesIndex)
	}

	// At first series, should not go below
	s.prevSeries()
	if s.SeriesIndex != 0 {
		t.Fatalf("expected series 0 (clamped), got %d", s.SeriesIndex)
	}
}

func TestInteractiveState_SeriesSwitchResetsViewport(t *testing.T) {
	s := newTestState(2, 100)
	s.zoomIn()
	s.zoomIn()
	s.panRight()

	s.nextSeries()

	if s.ViewStart != 0 {
		t.Fatalf("expected viewport reset on series switch, ViewStart=%d", s.ViewStart)
	}

	if s.ViewEnd != 100 {
		t.Fatalf("expected viewport reset on series switch, ViewEnd=%d", s.ViewEnd)
	}
}

