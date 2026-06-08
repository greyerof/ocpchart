package chart

import (
	"testing"
)

var testLabels = []string{
	`{instance="worker-0", cpu="0"}`,
	`{instance="worker-0", cpu="1"}`,
	`{instance="worker-1", cpu="0"}`,
	`{instance="worker-1", cpu="1"}`,
	`{instance="master-0", mode="idle"}`,
	`{instance="master-0", mode="system"}`,
}

func TestFilterSeries_EmptyInput(t *testing.T) {
	result := filterSeries(testLabels, "")
	if len(result) != len(testLabels) {
		t.Fatalf("expected %d results, got %d", len(testLabels), len(result))
	}

	for i, idx := range result {
		if idx != i {
			t.Fatalf("expected index %d at position %d, got %d", i, i, idx)
		}
	}
}

func TestFilterSeries_SubstringMatch(t *testing.T) {
	result := filterSeries(testLabels, "worker-0")
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0] != 0 || result[1] != 1 {
		t.Fatalf("expected indices [0, 1], got %v", result)
	}
}

func TestFilterSeries_CaseInsensitive(t *testing.T) {
	result := filterSeries(testLabels, "MASTER")
	if len(result) != 2 {
		t.Fatalf("expected 2 results for 'MASTER', got %d", len(result))
	}

	if result[0] != 4 || result[1] != 5 {
		t.Fatalf("expected indices [4, 5], got %v", result)
	}
}

func TestFilterSeries_NoMatch(t *testing.T) {
	result := filterSeries(testLabels, "nonexistent")
	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestFilterSeries_PartialLabel(t *testing.T) {
	result := filterSeries(testLabels, "cpu")
	if len(result) != 4 {
		t.Fatalf("expected 4 results for 'cpu', got %d", len(result))
	}
}

func TestPickerState_CursorDown(t *testing.T) {
	state := newPickerState(testLabels)

	state.cursorDown()
	if state.cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", state.cursor)
	}

	state.cursorDown()
	if state.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", state.cursor)
	}
}

func TestPickerState_CursorUp(t *testing.T) {
	state := newPickerState(testLabels)
	state.cursor = 3

	state.cursorUp()
	if state.cursor != 2 {
		t.Fatalf("expected cursor 2, got %d", state.cursor)
	}
}

func TestPickerState_CursorClampsAtBounds(t *testing.T) {
	state := newPickerState(testLabels)

	state.cursorUp()
	if state.cursor != 0 {
		t.Fatalf("cursor should not go below 0, got %d", state.cursor)
	}

	for i := 0; i < 20; i++ {
		state.cursorDown()
	}

	if state.cursor != len(testLabels)-1 {
		t.Fatalf("cursor should clamp at %d, got %d", len(testLabels)-1, state.cursor)
	}
}

func TestPickerState_AddCharFilters(t *testing.T) {
	state := newPickerState(testLabels)

	state.addChar('m')
	state.addChar('a')
	state.addChar('s')
	state.addChar('t')

	if len(state.filtered) != 2 {
		t.Fatalf("expected 2 filtered results after typing 'mast', got %d", len(state.filtered))
	}

	if state.cursor != 0 {
		t.Fatalf("cursor should reset to 0 after refilter, got %d", state.cursor)
	}
}

func TestPickerState_BackspaceWidens(t *testing.T) {
	state := newPickerState(testLabels)
	state.addChar('i')
	state.addChar('d')
	state.addChar('l')
	state.addChar('e')
	narrow := len(state.filtered) // "idle" matches 1 entry

	state.backspace() // "idl" -> still 1
	state.backspace() // "id" -> still 1
	state.backspace() // "i" -> matches all 6 (all contain "i")
	wide := len(state.filtered)

	if wide <= narrow {
		t.Fatalf("backspace should widen results: narrow=%d, wide=%d", narrow, wide)
	}
}

func TestPickerState_SelectedIndex(t *testing.T) {
	state := newPickerState(testLabels)
	state.cursorDown()
	state.cursorDown()

	idx := state.selectedIndex()
	if idx != 2 {
		t.Fatalf("expected selected index 2, got %d", idx)
	}
}

func TestPickerState_SelectedIndexWithFilter(t *testing.T) {
	state := newPickerState(testLabels)
	state.addChar('m')
	state.addChar('a')
	state.addChar('s')

	idx := state.selectedIndex()
	if idx != 4 {
		t.Fatalf("expected original index 4 (first master), got %d", idx)
	}
}

func TestTruncate_Short(t *testing.T) {
	got := truncate("hello", 10)
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}

func TestTruncate_Long(t *testing.T) {
	got := truncate("hello world this is long", 10)
	if got != "hello w..." {
		t.Fatalf("expected 'hello w...', got %q", got)
	}
}

func TestTruncate_Exact(t *testing.T) {
	got := truncate("hello", 5)
	if got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
}
