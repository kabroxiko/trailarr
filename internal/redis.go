package internal

import (
	"context"
	"fmt"
	"sync"
)

// Keep older function names but make them use BoltDB under the hood to avoid large changes across the codebase.

type fakeRedisClient struct {
	b boltLike
}

var (
	fakeClient *fakeRedisClient
	fakeMu     sync.Mutex
)

// GetRedisClient returns an adapter that satisfies the minimal methods callers expect.
// boltLike defines the subset of BoltClient methods used by the Redis adapter.
// Using an interface allows providing a noop fallback when the real DB can't be opened
// (useful in CI or tests that haven't set TrailarrRoot yet).
type boltLike interface {
	Ping(ctx context.Context) error
	RPush(ctx context.Context, key string, value []byte) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	LTrim(ctx context.Context, key string, start, stop int64) error
	LSet(ctx context.Context, key string, index int64, value []byte) error
	LRem(ctx context.Context, key string, count int, value []byte) error
	Del(ctx context.Context, key string) error
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) (string, error)
	HSet(ctx context.Context, key, field string, value []byte) error
	HGet(ctx context.Context, key, field string) (string, error)
	HVals(ctx context.Context, key string) ([]string, error)
	HDel(ctx context.Context, key, field string) error
}

// memBolt is a lightweight in-memory implementation of boltLike used when the
// on-disk BoltDB cannot be opened. It provides basic persistence semantics for
// sets, hashes and lists sufficient for tests.
type memBolt struct {
	mu     sync.RWMutex
	kv     map[string][]byte
	hashes map[string]map[string][]byte
	lists  map[string][][]byte
}

func newMemBolt() *memBolt {
	return &memBolt{
		kv:     make(map[string][]byte),
		hashes: make(map[string]map[string][]byte),
		lists:  make(map[string][][]byte),
	}
}

func (m *memBolt) Ping(ctx context.Context) error { return nil }

func (m *memBolt) Set(ctx context.Context, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kv[key] = append([]byte(nil), value...)
	return nil
}

func (m *memBolt) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.kv[key]
	if !ok {
		return "", ErrNotFound
	}
	return string(append([]byte(nil), v...)), nil
}

func (m *memBolt) Del(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.kv, key)
	delete(m.hashes, key)
	delete(m.lists, key)
	return nil
}

// Hash operations
func (m *memBolt) HSet(ctx context.Context, key, field string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hashes[key]; !ok {
		m.hashes[key] = make(map[string][]byte)
	}
	m.hashes[key][field] = append([]byte(nil), value...)
	return nil
}

func (m *memBolt) HGet(ctx context.Context, key, field string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hashes[key]
	if !ok {
		return "", ErrNotFound
	}
	v, ok2 := h[field]
	if !ok2 {
		return "", ErrNotFound
	}
	return string(append([]byte(nil), v...)), nil
}

func (m *memBolt) HVals(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.hashes[key]
	if !ok {
		return []string{}, nil
	}
	out := make([]string, 0, len(h))
	for _, v := range h {
		out = append(out, string(append([]byte(nil), v...)))
	}
	return out, nil
}

func (m *memBolt) HDel(ctx context.Context, key, field string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hashes[key]; ok {
		delete(h, field)
		if len(h) == 0 {
			delete(m.hashes, key)
		}
	}
	return nil
}

// List operations
func (m *memBolt) RPush(ctx context.Context, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists[key] = append(m.lists[key], append([]byte(nil), value...))
	return nil
}

func (m *memBolt) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vals := m.lists[key]
	n := int64(len(vals))
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
	out := make([]string, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		out = append(out, string(append([]byte(nil), vals[i]...)))
	}
	return out, nil
}

func (m *memBolt) LTrim(ctx context.Context, key string, start, stop int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vals := m.lists[key]
	n := int64(len(vals))
	if n == 0 {
		delete(m.lists, key)
		return nil
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
		delete(m.lists, key)
		return nil
	}
	newVals := make([][]byte, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		newVals = append(newVals, append([]byte(nil), vals[i]...))
	}
	m.lists[key] = newVals
	return nil
}

func (m *memBolt) LSet(ctx context.Context, key string, index int64, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vals := m.lists[key]
	if index < 0 || index >= int64(len(vals)) {
		return fmt.Errorf("index out of range")
	}
	vals[index] = append([]byte(nil), value...)
	m.lists[key] = vals
	return nil
}

func (m *memBolt) LRem(ctx context.Context, key string, count int, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	vals := m.lists[key]
	if len(vals) == 0 {
		return nil
	}
	target := string(value)
	newVals := make([][]byte, 0, len(vals))
	removed := 0
	for _, v := range vals {
		if removed < count && string(v) == target {
			removed++
			continue
		}
		newVals = append(newVals, append([]byte(nil), v...))
	}
	if len(newVals) == 0 {
		delete(m.lists, key)
		return nil
	}
	m.lists[key] = newVals
	return nil
}

func GetRedisClient() *fakeRedisClient {
	// Fast path: if we've already created a client, try to return it. If it's backed
	// by noopBolt, attempt to re-open the real BoltDB in case tests changed TrailarrRoot.
	fakeMu.Lock()
	fc := fakeClient
	fakeMu.Unlock()
	if fc != nil {
		// if currently memBolt, attempt to upgrade to real bolt
		tryUpgradeToBolt()
		return fakeClient
	}

	// Not created yet; try to create a real Bolt-backed client first.
	if b, err := GetBoltClient(); err == nil && b != nil {
		fakeMu.Lock()
		if fakeClient == nil {
			fakeClient = &fakeRedisClient{b: b}
		}
		fc = fakeClient
		fakeMu.Unlock()
		return fc
	}
	// If GetBoltClient failed (boltOnce may have run earlier), try openBoltDB
	// directly so tests that set TrailarrRoot get a working DB.
	if b2, err2 := openBoltDB(); err2 == nil && b2 != nil {
		fakeMu.Lock()
		if fakeClient == nil {
			fakeClient = &fakeRedisClient{b: b2}
		}
		fc = fakeClient
		fakeMu.Unlock()
		return fc
	}

	// Fall back to noop but store it so subsequent calls are fast; later calls will
	// attempt to upgrade to real Bolt if it becomes available.
	fakeMu.Lock()
	if fakeClient == nil {
		fakeClient = &fakeRedisClient{b: newMemBolt()}
	}
	fc = fakeClient
	fakeMu.Unlock()
	return fc
}

// tryUpgradeToBolt attempts to replace a memBolt-backed fakeClient with a real Bolt-backed
// client. It is best-effort and returns silently if it cannot open Bolt.
func tryUpgradeToBolt() {
	fakeMu.Lock()
	curr := fakeClient
	fakeMu.Unlock()
	if curr == nil {
		return
	}
	if _, ok := curr.b.(*memBolt); !ok {
		return
	}
	if b, err := GetBoltClient(); err == nil && b != nil {
		fakeMu.Lock()
		fakeClient = &fakeRedisClient{b: b}
		fakeMu.Unlock()
		return
	}
	if b2, err2 := openBoltDB(); err2 == nil && b2 != nil {
		fakeMu.Lock()
		fakeClient = &fakeRedisClient{b: b2}
		fakeMu.Unlock()
	}
}

// PingRedis checks if backend is reachable
func PingRedis(ctx context.Context) error {
	c := GetRedisClient()
	return c.Ping(ctx)
}

// ---- result adapter types ----
type simpleErrResult struct{ err error }

func (r simpleErrResult) Err() error { return r.err }

type strSliceResult struct {
	vals []string
	err  error
}

func (r strSliceResult) Result() ([]string, error) { return r.vals, r.err }

type strResult struct {
	val string
	err error
}

func (r strResult) Result() (string, error) { return r.val, r.err }

// ---- adapter methods ----
func (c *fakeRedisClient) Ping(ctx context.Context) error {
	return c.b.Ping(ctx)
}

func (c *fakeRedisClient) RPush(ctx context.Context, key string, value []byte) simpleErrResult {
	return simpleErrResult{err: c.b.RPush(ctx, key, value)}
}

func (c *fakeRedisClient) LRange(ctx context.Context, key string, start, stop int64) strSliceResult {
	vals, err := c.b.LRange(ctx, key, start, stop)
	return strSliceResult{vals: vals, err: err}
}

func (c *fakeRedisClient) LTrim(ctx context.Context, key string, start, stop int64) simpleErrResult {
	return simpleErrResult{err: c.b.LTrim(ctx, key, start, stop)}
}

func (c *fakeRedisClient) LSet(ctx context.Context, key string, index int64, value []byte) simpleErrResult {
	return simpleErrResult{err: c.b.LSet(ctx, key, index, value)}
}

func (c *fakeRedisClient) LRem(ctx context.Context, key string, count int, value []byte) simpleErrResult {
	return simpleErrResult{err: c.b.LRem(ctx, key, count, value)}
}

func (c *fakeRedisClient) Del(ctx context.Context, key string) simpleErrResult {
	return simpleErrResult{err: c.b.Del(ctx, key)}
}

func (c *fakeRedisClient) Set(ctx context.Context, key string, value []byte, _ int) simpleErrResult {
	return simpleErrResult{err: c.b.Set(ctx, key, value)}
}

func (c *fakeRedisClient) Get(ctx context.Context, key string) strResult {
	v, err := c.b.Get(ctx, key)
	if err != nil {
		return strResult{val: "", err: err}
	}
	return strResult{val: v, err: nil}
}

func (c *fakeRedisClient) HSet(ctx context.Context, key, field string, value []byte) simpleErrResult {
	return simpleErrResult{err: c.b.HSet(ctx, key, field, value)}
}

func (c *fakeRedisClient) HGet(ctx context.Context, key, field string) strResult {
	v, err := c.b.HGet(ctx, key, field)
	if err != nil {
		return strResult{val: "", err: err}
	}
	return strResult{val: v, err: nil}
}

func (c *fakeRedisClient) HVals(ctx context.Context, key string) strSliceResult {
	vals, err := c.b.HVals(ctx, key)
	return strSliceResult{vals: vals, err: err}
}

func (c *fakeRedisClient) HDel(ctx context.Context, key, field string) simpleErrResult {
	return simpleErrResult{err: c.b.HDel(ctx, key, field)}
}
