package thanos

import (
	"testing"
)

func TestLabelSetString_Empty(t *testing.T) {
	result := LabelSetString(map[string]string{})
	if result != "{}" {
		t.Fatalf("expected {}, got %s", result)
	}
}

func TestLabelSetString_ExcludesName(t *testing.T) {
	result := LabelSetString(map[string]string{
		"__name__": "cpu_usage",
		"instance": "worker-0",
	})
	expected := `{instance="worker-0"}`
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestLabelSetString_MultipleLabels(t *testing.T) {
	result := LabelSetString(map[string]string{
		"instance": "worker-0",
		"job":      "node-exporter",
	})
	expected := `{instance="worker-0", job="node-exporter"}`
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}

func TestLabelSetString_OnlyName(t *testing.T) {
	result := LabelSetString(map[string]string{
		"__name__": "up",
	})
	if result != "{}" {
		t.Fatalf("expected {}, got %s", result)
	}
}

func TestLabelSetString_Sorted(t *testing.T) {
	result := LabelSetString(map[string]string{
		"z_label": "z",
		"a_label": "a",
		"m_label": "m",
	})
	expected := `{a_label="a", m_label="m", z_label="z"}`
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}
