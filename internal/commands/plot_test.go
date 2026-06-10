package commands

import (
	"testing"
	"time"
)

func TestParseUntil_Duration(t *testing.T) {
	before := time.Now()
	got, err := parseUntil("5m")
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be roughly now - 5m
	expected := before.Add(-5 * time.Minute)
	if got.Before(expected.Add(-time.Second)) || got.After(after.Add(-5*time.Minute).Add(time.Second)) {
		t.Fatalf("time %v not within expected range around now-5m", got)
	}
}

func TestParseUntil_RFC3339(t *testing.T) {
	input := "2026-06-08T14:30:00Z"
	got, err := parseUntil(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := time.Date(2026, 6, 8, 14, 30, 0, 0, time.UTC)
	if !got.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestParseUntil_Invalid(t *testing.T) {
	_, err := parseUntil("not-a-time")
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParseUntil_ZeroDuration(t *testing.T) {
	before := time.Now()
	got, err := parseUntil("0s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 0s duration means "now"
	if got.Before(before.Add(-time.Second)) {
		t.Fatalf("expected time near now, got %v", got)
	}
}
