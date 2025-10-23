package internal

import "testing"

func TestNormalizeRangeEmpty(t *testing.T) {
	s, e, empty := normalizeRange(0, 0, 1)
	if !empty {
		t.Fatalf("expected empty for n=0, got s=%d e=%d", s, e)
	}
}

func TestNormalizeRangeNegatives(t *testing.T) {
	s, e, empty := normalizeRange(5, -2, -1)
	if empty {
		t.Fatalf("did not expect empty for negative indices")
	}
	if s != 3 || e != 4 {
		t.Fatalf("unexpected normalized values: s=%d e=%d", s, e)
	}
}

func TestNormalizeRangeStartGreaterStop(t *testing.T) {
	_, _, empty := normalizeRange(5, 4, 2)
	if !empty {
		t.Fatalf("expected empty when start>stop")
	}
}

func TestRemoveMatchesBehavior(t *testing.T) {
	vals := [][]byte{[]byte("a"), []byte("b"), []byte("a"), []byte("c")}
	// count=0 => unchanged
	r := removeMatches(vals, 0, []byte("a"))
	if len(r) != 4 {
		t.Fatalf("expected unchanged when count=0, got %d", len(r))
	}
	// remove up to 1 occurrence
	r2 := removeMatches(vals, 1, []byte("a"))
	if len(r2) != 3 {
		t.Fatalf("expected length 3 after removing 1, got %d", len(r2))
	}
	// remove all occurrences
	r3 := removeMatches(vals, 10, []byte("a"))
	if len(r3) != 2 {
		t.Fatalf("expected length 2 after removing all a, got %d", len(r3))
	}
}
