package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteAndReadJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	src := payload{Name: "trailarr", Count: 3}
	if err := WriteJSONFile(path, src); err != nil {
		t.Fatalf("WriteJSONFile failed: %v", err)
	}

	var dest payload
	if err := ReadJSONFile(path, &dest); err != nil {
		t.Fatalf("ReadJSONFile failed: %v", err)
	}

	if !reflect.DeepEqual(dest, src) {
		t.Fatalf("roundtrip mismatch: got %v want %v", dest, src)
	}
}

func TestReadJSONFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nope.json")
	var dest interface{}
	if err := ReadJSONFile(path, &dest); err == nil {
		t.Fatalf("expected error reading non-existent file")
	}
}

func TestWriteJSONFileInvalidMarshal(t *testing.T) {
	// channels cannot be JSON marshaled; expect an error
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	src := map[string]interface{}{"ch": make(chan int)}
	err := WriteJSONFile(path, src)
	if err == nil {
		// cleanup file if it incorrectly succeeded
		_ = os.Remove(path)
		t.Fatalf("expected error when marshaling unsupported type, got nil")
	}
}
