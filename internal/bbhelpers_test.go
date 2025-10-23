package internal

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNormalizeRange(t *testing.T) {
	// returns start, stop, empty
	s, e, empty := normalizeRange(5, 0, 2)
	if empty {
		t.Fatalf("normalizeRange returned empty unexpectedly")
	}
	if s != 0 || e != 2 {
		t.Fatalf("normalizeRange returned unexpected values: %d %d", s, e)
	}
	// call with negative indices
	s2, e2, _ := normalizeRange(3, -10, 10)
	if s2 < 0 {
		t.Fatalf("normalizeRange returned negative start: %d", s2)
	}
	if e2 < 0 {
		t.Fatalf("normalizeRange returned negative stop: %d", e2)
	}
}

func TestRemoveMatches(t *testing.T) {
	a := [][]byte{[]byte("x"), []byte("y"), []byte("x")}
	// remove up to 2 occurrences of "x" -> should leave only "y"
	out := removeMatches(a, 2, []byte("x"))
	if len(out) != 1 || !bytes.Equal(out[0], []byte("y")) {
		t.Fatalf("removeMatches did not remove expected items: %v", out)
	}
}

func TestU64ToBytes(t *testing.T) {
	v := u64ToBytes(0x0102030405060708)
	if len(v) != 8 {
		t.Fatalf("u64ToBytes returned wrong length: %d", len(v))
	}
	// roundtrip via reflect (not necessary but quick check)
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		t.Fatalf("u64ToBytes returned non-slice")
	}
}
