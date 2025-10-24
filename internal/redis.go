package internal

import (
	"context"
	"fmt"
	"sync"
)

// Keep older function names but make them use BoltDB under the hood to avoid large changes across the codebase.

var (
	storeClient *Store
	storeMu     sync.Mutex
)

// Centralized error values used in this file. Keeping them as package-level
// variables avoids repeating the same string literal in multiple places and
// makes it easier to change the message in one spot.
var (
	ErrNoStoreClient   = fmt.Errorf("no store client")
	ErrIndexOutOfRange = fmt.Errorf("index out of range")
)

// Store is a concrete adapter that delegates store operations to either the
// on-disk BoltClient or an in-memory memBolt. We prefer the real Bolt client
// when available but fall back to memBolt for tests or when the DB cannot be
// opened.
type Store struct {
	bolt *BoltClient
	mem  *memBolt
}

// The Store type exposes the same methods previously used by callers; callers
// will receive a *Store from GetStoreClient() and call these methods directly.

// memBolt is a lightweight in-memory implementation used when the
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
		return ErrIndexOutOfRange
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

// GetStoreClient returns a boltLike implementation. It prefers a real BoltDB-backed
// client when available, otherwise falls back to an in-memory implementation for tests.
func GetStoreClient() *Store {
	// Fast path: return existing client if present; attempt upgrade when memBolt is used.
	storeMu.Lock()
	sc := storeClient
	storeMu.Unlock()
	if sc != nil {
		tryUpgradeToBolt()
		return storeClient
	}

	// Try Bolt-backed client first
	if b, err := GetBoltClient(); err == nil && b != nil {
		storeMu.Lock()
		if storeClient == nil {
			storeClient = &Store{bolt: b}
		}
		sc = storeClient
		storeMu.Unlock()
		return sc
	}

	// Try openBoltDB directly
	if b2, err2 := openBoltDB(); err2 == nil && b2 != nil {
		storeMu.Lock()
		if storeClient == nil {
			storeClient = &Store{bolt: b2}
		}
		sc = storeClient
		storeMu.Unlock()
		return sc
	}

	// Fall back to in-memory implementation
	storeMu.Lock()
	if storeClient == nil {
		storeClient = &Store{mem: newMemBolt()}
	}
	sc = storeClient
	storeMu.Unlock()
	return sc
}

// tryUpgradeToBolt attempts to replace a memBolt-backed fakeClient with a real Bolt-backed
// client. It is best-effort and returns silently if it cannot open Bolt.
func tryUpgradeToBolt() {
	storeMu.Lock()
	curr := storeClient
	storeMu.Unlock()
	if curr == nil {
		return
	}
	// If already using bolt, nothing to do
	if curr.bolt != nil {
		return
	}
	// Attempt to open bolt and attach to existing Store (best-effort)
	if b, err := GetBoltClient(); err == nil && b != nil {
		storeMu.Lock()
		if storeClient != nil {
			storeClient.bolt = b
		}
		storeMu.Unlock()
		return
	}
	if b2, err2 := openBoltDB(); err2 == nil && b2 != nil {
		storeMu.Lock()
		if storeClient != nil {
			storeClient.bolt = b2
		}
		storeMu.Unlock()
	}
}

// Store method implementations delegate to the underlying BoltClient or memBolt.
func (s *Store) Ping(ctx context.Context) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.Ping(ctx)
	}
	if s.mem != nil {
		return s.mem.Ping(ctx)
	}
	return ErrNoStoreClient
}

func (s *Store) RPush(ctx context.Context, key string, value []byte) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.RPush(ctx, key, value)
	}
	return s.mem.RPush(ctx, key, value)
}

func (s *Store) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	if s == nil {
		return nil, ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.LRange(ctx, key, start, stop)
	}
	return s.mem.LRange(ctx, key, start, stop)
}

func (s *Store) LTrim(ctx context.Context, key string, start, stop int64) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.LTrim(ctx, key, start, stop)
	}
	return s.mem.LTrim(ctx, key, start, stop)
}

func (s *Store) LSet(ctx context.Context, key string, index int64, value []byte) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.LSet(ctx, key, index, value)
	}
	return s.mem.LSet(ctx, key, index, value)
}

func (s *Store) LRem(ctx context.Context, key string, count int, value []byte) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.LRem(ctx, key, count, value)
	}
	return s.mem.LRem(ctx, key, count, value)
}

func (s *Store) Del(ctx context.Context, key string) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.Del(ctx, key)
	}
	return s.mem.Del(ctx, key)
}

func (s *Store) Set(ctx context.Context, key string, value []byte) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.Set(ctx, key, value)
	}
	return s.mem.Set(ctx, key, value)
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	if s == nil {
		return "", ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.Get(ctx, key)
	}
	return s.mem.Get(ctx, key)
}

func (s *Store) HSet(ctx context.Context, key, field string, value []byte) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.HSet(ctx, key, field, value)
	}
	return s.mem.HSet(ctx, key, field, value)
}

func (s *Store) HGet(ctx context.Context, key, field string) (string, error) {
	if s == nil {
		return "", ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.HGet(ctx, key, field)
	}
	return s.mem.HGet(ctx, key, field)
}

func (s *Store) HVals(ctx context.Context, key string) ([]string, error) {
	if s == nil {
		return nil, ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.HVals(ctx, key)
	}
	return s.mem.HVals(ctx, key)
}

func (s *Store) HDel(ctx context.Context, key, field string) error {
	if s == nil {
		return ErrNoStoreClient
	}
	if s.bolt != nil {
		return s.bolt.HDel(ctx, key, field)
	}
	return s.mem.HDel(ctx, key, field)
}

// PingStore checks if backend is reachable
func PingStore(ctx context.Context) error {
	c := GetStoreClient()
	if c == nil {
		return ErrNoStoreClient
	}
	return c.Ping(ctx)
}

// ---- adapter methods ----
// No adapter methods beyond the Store methods are provided — callers should
// call `GetStoreClient()` and use the Store methods directly
// (e.g. LRange(ctx, key, start, stop) ([]string, error)).
