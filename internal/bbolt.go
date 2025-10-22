package internal

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	bolt "go.etcd.io/bbolt"
)

// BoltDB-backed simple compatibility layer for the small subset of Redis ops
// used in Trailarr. This is intentionally minimal and synchronous.

type BoltClient struct {
	db *bolt.DB
}

var boltClient *BoltClient
var boltOnce sync.Once

func openBoltDB() (*BoltClient, error) {
	dbPath := filepath.Join(TrailarrRoot, "trailarr.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltClient{db: db}, nil
}

// GetBoltClient returns a singleton BoltClient
func GetBoltClient() (*BoltClient, error) {
	var err error
	boltOnce.Do(func() {
		boltClient, err = openBoltDB()
	})
	return boltClient, err
}

var ErrNotFound = errors.New("not found")

// Ping is a no-op for BoltDB
func (c *BoltClient) Ping(ctx context.Context) error {
	return nil
}

// ------------ string key/value (simple KV) ----------------
func (c *BoltClient) Set(ctx context.Context, key string, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("kv"))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), value)
	})
}

func (c *BoltClient) Get(ctx context.Context, key string) (string, error) {
	var out []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("kv"))
		if b == nil {
			return ErrNotFound
		}
		v := b.Get([]byte(key))
		if v == nil {
			return ErrNotFound
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ------------ hash (HSET/HGET/HVALS/HDEL) ----------------
func hashBucketName(key string) []byte { return []byte("hash:" + key) }

func (c *BoltClient) HSet(ctx context.Context, key, field string, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(hashBucketName(key))
		if err != nil {
			return err
		}
		return b.Put([]byte(field), value)
	})
}

func (c *BoltClient) HGet(ctx context.Context, key, field string) (string, error) {
	var out []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(hashBucketName(key))
		if b == nil {
			return ErrNotFound
		}
		v := b.Get([]byte(field))
		if v == nil {
			return ErrNotFound
		}
		out = append([]byte(nil), v...)
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *BoltClient) HVals(ctx context.Context, key string) ([]string, error) {
	var vals []string
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(hashBucketName(key))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			vals = append(vals, string(v))
			return nil
		})
	})
	return vals, err
}

func (c *BoltClient) HDel(ctx context.Context, key, field string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(hashBucketName(key))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(field))
	})
}

// ------------ list (LRANGE, RPUSH, LTRIM, LSET, LREM, DEL) ----------------
func listBucketName(key string) []byte { return []byte("list:" + key) }

func u64ToBytes(i uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], i)
	return b[:]
}

func (c *BoltClient) RPush(ctx context.Context, key string, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(listBucketName(key))
		if err != nil {
			return err
		}
		seq, _ := b.NextSequence()
		return b.Put(u64ToBytes(seq), value)
	})
}

func (c *BoltClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	var out []string
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(listBucketName(key))
		if b == nil {
			return nil
		}
		// Collect all items in order
		return b.ForEach(func(k, v []byte) error {
			out = append(out, string(v))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	// Apply start/stop like Redis semantics (support -1)
	n := int64(len(out))
	if n == 0 {
		return []string{}, nil
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop || start >= n {
		return []string{}, nil
	}
	return out[start : stop+1], nil
}

func listBucketValues(b *bolt.Bucket) [][]byte {
	var vals [][]byte
	if b == nil {
		return vals
	}
	_ = b.ForEach(func(k, v []byte) error {
		vals = append(vals, append([]byte(nil), v...))
		return nil
	})
	return vals
}

func normalizeRange(n, start, stop int64) (int64, int64, bool) {
	if n == 0 {
		return 0, 0, true
	}
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop {
		return 0, 0, true
	}
	return start, stop, false
}

func (c *BoltClient) LTrim(ctx context.Context, key string, start, stop int64) error {
	// Simplified implementation using helpers to reduce cognitive complexity.
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(listBucketName(key))
		if b == nil {
			return nil
		}
		vals := listBucketValues(b)
		s, e, empty := normalizeRange(int64(len(vals)), start, stop)
		if empty {
			// clear bucket
			_ = tx.DeleteBucket(listBucketName(key))
			return nil
		}
		keep := vals[s : e+1]
		// delete and recreate
		_ = tx.DeleteBucket(listBucketName(key))
		nb, err := tx.CreateBucketIfNotExists(listBucketName(key))
		if err != nil {
			return err
		}
		for i, v := range keep {
			if err := nb.Put(u64ToBytes(uint64(i+1)), v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *BoltClient) LSet(ctx context.Context, key string, index int64, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(listBucketName(key))
		if b == nil {
			return fmt.Errorf("index out of range")
		}
		// rebuild all into slice, set index, rewrite
		var vals [][]byte
		_ = b.ForEach(func(k, v []byte) error {
			vals = append(vals, append([]byte(nil), v...))
			return nil
		})
		if index < 0 || index >= int64(len(vals)) {
			return fmt.Errorf("index out of range")
		}
		vals[index] = append([]byte(nil), value...)
		_ = tx.DeleteBucket(listBucketName(key))
		nb, err := tx.CreateBucketIfNotExists(listBucketName(key))
		if err != nil {
			return err
		}
		for i, v := range vals {
			if err := nb.Put(u64ToBytes(uint64(i+1)), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// helper to remove up to count occurrences of value from vals (preserves original behavior for count <= 0)
func removeMatches(vals [][]byte, count int, value []byte) [][]byte {
	if len(vals) == 0 || count <= 0 {
		return vals
	}
	target := string(value)
	newVals := make([][]byte, 0, len(vals))
	removed := 0
	for _, v := range vals {
		if removed < count && string(v) == target {
			removed++
			continue
		}
		newVals = append(newVals, v)
	}
	return newVals
}

func (c *BoltClient) LRem(ctx context.Context, key string, count int, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(listBucketName(key))
		if b == nil {
			return nil
		}
		var vals [][]byte
		_ = b.ForEach(func(k, v []byte) error {
			vals = append(vals, append([]byte(nil), v...))
			return nil
		})
		newVals := removeMatches(vals, count, value)
		_ = tx.DeleteBucket(listBucketName(key))
		if len(newVals) == 0 {
			return nil
		}
		nb, err := tx.CreateBucketIfNotExists(listBucketName(key))
		if err != nil {
			return err
		}
		for i, v := range newVals {
			if err := nb.Put(u64ToBytes(uint64(i+1)), v); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *BoltClient) Del(ctx context.Context, key string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		_ = tx.DeleteBucket(listBucketName(key))
		_ = tx.DeleteBucket(hashBucketName(key))
		b := tx.Bucket([]byte("kv"))
		if b != nil {
			_ = b.Delete([]byte(key))
		}
		return nil
	})
}
