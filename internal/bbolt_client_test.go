package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const (
	errRPush = "RPush failed: %v"
)

func setupBoltClient(t *testing.T) *BoltClient {
	t.Helper()
	tmp := t.TempDir()
	// set TrailarrRoot so openBoltDB uses the temp dir
	oldRoot := TrailarrRoot
	TrailarrRoot = tmp
	t.Cleanup(func() {
		TrailarrRoot = oldRoot
	})
	// ensure db file directory exists
	_ = os.MkdirAll(tmp, 0o755)
	c, err := openBoltDB()
	if err != nil {
		t.Fatalf("openBoltDB failed: %v", err)
	}
	// if a db file path is used, ensure cleanup
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
			_ = os.Remove(filepath.Join(tmp, "trailarr.db"))
		}
	})
	return c
}

func TestBoltSetGetAndDel(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// Set and Get
	if err := c.Set(ctx, "k1", []byte("v1")); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	v, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if v != "v1" {
		t.Fatalf("unexpected Get value: %s", v)
	}
	// Del
	if err := c.Del(ctx, "k1"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if _, err := c.Get(ctx, "k1"); err == nil {
		t.Fatalf("expected error after delete")
	}
}

func TestBoltHashOperations(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	if err := c.HSet(ctx, "h1", "f1", []byte("fv1")); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	got, err := c.HGet(ctx, "h1", "f1")
	if err != nil {
		t.Fatalf("HGet failed: %v", err)
	}
	if got != "fv1" {
		t.Fatalf("unexpected HGet value: %s", got)
	}
	vals, err := c.HVals(ctx, "h1")
	if err != nil {
		t.Fatalf("HVals failed: %v", err)
	}
	if len(vals) != 1 || vals[0] != "fv1" {
		t.Fatalf("unexpected HVals: %v", vals)
	}
	if err := c.HDel(ctx, "h1", "f1"); err != nil {
		t.Fatalf("HDel failed: %v", err)
	}
	vals2, err := c.HVals(ctx, "h1")
	if err != nil {
		t.Fatalf("HVals after delete failed: %v", err)
	}
	if len(vals2) != 0 {
		t.Fatalf("expected empty HVals after delete: %v", vals2)
	}
}

func TestBoltListRPushAndLRange(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()

	if err := c.RPush(ctx, "lst", []byte("a")); err != nil {
		t.Fatalf(errRPush, err)
	}
	if err := c.RPush(ctx, "lst", []byte("b")); err != nil {
		t.Fatalf(errRPush, err)
	}
	if err := c.RPush(ctx, "lst", []byte("c")); err != nil {
		t.Fatalf(errRPush, err)
	}
	vals, err := c.LRange(ctx, "lst", 0, -1)
	if err != nil {
		t.Fatalf("LRange failed: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("unexpected LRange length: %v", vals)
	}
}

func TestBoltListLSetUpdatesElement(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// prepare list
	_ = c.RPush(ctx, "lst", []byte("a"))
	_ = c.RPush(ctx, "lst", []byte("b"))
	_ = c.RPush(ctx, "lst", []byte("c"))

	if err := c.LSet(ctx, "lst", 1, []byte("x")); err != nil {
		t.Fatalf("LSet failed: %v", err)
	}
	vals2, err := c.LRange(ctx, "lst", 0, -1)
	if err != nil {
		t.Fatalf("LRange after LSet failed: %v", err)
	}
	if vals2[1] != "x" {
		t.Fatalf("LSet did not update element: %v", vals2)
	}
}

func TestBoltListLRemAndLRange(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// prepare list with occurrence to remove
	_ = c.RPush(ctx, "lst", []byte("a"))
	_ = c.RPush(ctx, "lst", []byte("x"))
	_ = c.RPush(ctx, "lst", []byte("c"))

	if err := c.LRem(ctx, "lst", 1, []byte("x")); err != nil {
		t.Fatalf("LRem failed: %v", err)
	}
	vals3, err := c.LRange(ctx, "lst", 0, -1)
	if err != nil {
		t.Fatalf("LRange after LRem failed: %v", err)
	}
	if len(vals3) != 2 {
		t.Fatalf("unexpected length after LRem: %v", vals3)
	}
}

func TestBoltListLTrimToSingle(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// prepare list
	_ = c.RPush(ctx, "lst", []byte("a"))
	_ = c.RPush(ctx, "lst", []byte("b"))
	_ = c.RPush(ctx, "lst", []byte("c"))

	if err := c.LTrim(ctx, "lst", 0, 0); err != nil {
		t.Fatalf("LTrim failed: %v", err)
	}
	vals4, err := c.LRange(ctx, "lst", 0, -1)
	if err != nil {
		t.Fatalf("LRange after LTrim failed: %v", err)
	}
	if len(vals4) != 1 {
		t.Fatalf("unexpected length after LTrim: %v", vals4)
	}
}

func TestBoltListDelListResultsEmpty(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// prepare list
	_ = c.RPush(ctx, "lst", []byte("a"))

	if err := c.Del(ctx, "lst"); err != nil {
		t.Fatalf("Del list failed: %v", err)
	}
	vals5, err := c.LRange(ctx, "lst", 0, -1)
	if err != nil {
		t.Fatalf("LRange after Del failed: %v", err)
	}
	if len(vals5) != 0 {
		t.Fatalf("expected empty list after Del: %v", vals5)
	}
}

func TestBoltListEdgeCases(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// prepare list
	_ = c.RPush(ctx, "lst", []byte("a"))
	_ = c.RPush(ctx, "lst", []byte("b"))
	_ = c.RPush(ctx, "lst", []byte("c"))

	// negative indices: last two
	vals, err := c.LRange(ctx, "lst", -2, -1)
	if err != nil {
		t.Fatalf("LRange negative indices failed: %v", err)
	}
	if len(vals) != 2 || vals[0] != "b" || vals[1] != "c" {
		t.Fatalf("unexpected LRange negative result: %v", vals)
	}

	// LSet out of range should return error
	if err := c.LSet(ctx, "lst", 10, []byte("z")); err == nil {
		t.Fatalf("expected error from LSet out of range")
	}

	// LRem with count=0 should not remove anything (per helper semantics)
	if err := c.LRem(ctx, "lst", 0, []byte("a")); err != nil {
		t.Fatalf("LRem with count=0 failed: %v", err)
	}
	vals2, _ := c.LRange(ctx, "lst", 0, -1)
	if len(vals2) != 3 {
		t.Fatalf("LRem with count=0 unexpectedly removed items: %v", vals2)
	}

	// LTrim with start > stop should clear the list
	if err := c.LTrim(ctx, "lst", 2, 1); err != nil {
		t.Fatalf("LTrim start>stop failed: %v", err)
	}
	vals3, _ := c.LRange(ctx, "lst", 0, -1)
	if len(vals3) != 0 {
		t.Fatalf("expected empty list after LTrim start>stop: %v", vals3)
	}
}

func TestHashMissingField(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()
	// ensure HGet on missing field returns ErrNotFound
	_, err := c.HGet(ctx, "h_missing", "nope")
	if err == nil {
		t.Fatalf("expected ErrNotFound for missing hash field")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestBoltMoreEdges(t *testing.T) {
	c := setupBoltClient(t)
	ctx := context.Background()

	// Ping is a no-op but should not error
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Directly test listBucketValues helper via creating a bucket and reading values
	// RPush a couple values and then open DB view to call helper
	_ = c.RPush(ctx, "lbv", []byte("1"))
	_ = c.RPush(ctx, "lbv", []byte("2"))
	// ensure values are present
	vals, err := c.LRange(ctx, "lbv", 0, -1)
	if err != nil || len(vals) != 2 {
		t.Fatalf("setup for listBucketValues failed: %v %v", err, vals)
	}

	// LSet negative index should return error
	if err := c.LSet(ctx, "lbv", -10, []byte("z")); err == nil {
		t.Fatalf("expected error from LSet negative index")
	}

	// LRem with large count should remove all matches
	_ = c.RPush(ctx, "remtest", []byte("x"))
	_ = c.RPush(ctx, "remtest", []byte("y"))
	_ = c.RPush(ctx, "remtest", []byte("x"))
	if err := c.LRem(ctx, "remtest", 1000, []byte("x")); err != nil {
		t.Fatalf("LRem remove-all failed: %v", err)
	}
	rvals, _ := c.LRange(ctx, "remtest", 0, -1)
	if len(rvals) != 1 || rvals[0] != "y" {
		t.Fatalf("unexpected remtest after remove-all: %v", rvals)
	}

	// LTrim on a non-existent list should not error
	if err := c.LTrim(ctx, "noexist", 0, 1); err != nil {
		t.Fatalf("LTrim on missing bucket failed: %v", err)
	}

	// Del should remove both hash and kv entries
	_ = c.Set(ctx, "kdel", []byte("vdel"))
	_ = c.HSet(ctx, "hdel", "f", []byte("vf"))
	if err := c.Del(ctx, "hdel"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if _, err := c.HGet(ctx, "hdel", "f"); err == nil {
		t.Fatalf("expected error after Del on hash")
	}
	// also delete kv
	if err := c.Del(ctx, "kdel"); err != nil {
		t.Fatalf("Del kv failed: %v", err)
	}
	if _, err := c.Get(ctx, "kdel"); err == nil {
		t.Fatalf("expected error after Del on kv")
	}
}
