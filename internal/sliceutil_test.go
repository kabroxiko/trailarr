package internal

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFilterMapMapAndFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	// keep even numbers and map to string
	out := FilterMap(input, func(i int) bool { return i%2 == 0 }, func(i int) string { return fmt.Sprintf("n=%d", i) })
	// expected: for 2 and 4 -> "n=2", "n=4"
	expected := []string{"n=2", "n=4"}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("unexpected result: got %v want %v", out, expected)
	}
}

func TestFilterEmptyAndNonEmpty(t *testing.T) {
	input := []string{"a", "", "c"}
	out := Filter(input, func(s string) bool { return s != "" })
	expected := []string{"a", "c"}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("Filter failed: got %v want %v", out, expected)
	}

	// empty input
	empty := []int{}
	got := Filter(empty, func(i int) bool { return true })
	if len(got) != 0 {
		t.Fatalf("Filter on empty input should return empty slice, got %v", got)
	}
}

func TestMapGeneric(t *testing.T) {
	input := []int{1, 2, 3}
	out := Map(input, func(i int) int { return i * i })
	expected := []int{1, 4, 9}
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("Map failed: got %v want %v", out, expected)
	}
}
