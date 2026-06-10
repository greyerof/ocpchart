package chart

import (
	"strings"
	"testing"
	"time"

	"github.com/greyerof/ocpchart/internal/thanos"
)

func TestLiveStatusLine_DoesNotContainLabels(t *testing.T) {
	state := &InteractiveState{
		AllSeries: []thanos.Series{{
			Labels: map[string]string{"instance": "worker-0", "job": "kubelet"},
			Times:  makeTimes(time.Now(), 50, 30*time.Second),
			Values: makeSineValues(50),
		}},
		Query:    "up",
		ViewEnd:  50,
	}

	got := liveStatusLine(state, 30*time.Second)

	if strings.Contains(got, "worker-0") {
		t.Fatalf("status line should not contain labels, got: %s", got)
	}

	if !strings.Contains(got, "Series 1/1") {
		t.Fatalf("status line should contain series count, got: %s", got)
	}

	if !strings.Contains(got, "Samples 1-50 of 50") {
		t.Fatalf("status line should contain sample range, got: %s", got)
	}

	if !strings.Contains(got, "Refresh 30s") {
		t.Fatalf("status line should contain refresh interval, got: %s", got)
	}
}

func TestLiveStatusLine_Format(t *testing.T) {
	state := &InteractiveState{
		AllSeries: []thanos.Series{
			{Labels: map[string]string{"a": "1"}, Times: makeTimes(time.Now(), 10, time.Second), Values: makeSineValues(10)},
			{Labels: map[string]string{"a": "2"}, Times: makeTimes(time.Now(), 10, time.Second), Values: makeSineValues(10)},
		},
		SeriesIndex: 1,
		Query:       "metric",
		ViewStart:   2,
		ViewEnd:     8,
	}

	got := liveStatusLine(state, 15*time.Second)

	if !strings.Contains(got, "Series 2/2") {
		t.Fatalf("expected Series 2/2, got: %s", got)
	}

	if !strings.Contains(got, "Samples 3-8 of 10") {
		t.Fatalf("expected Samples 3-8 of 10, got: %s", got)
	}
}
