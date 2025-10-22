package internal

import (
	"context"
	"fmt"
	"sync"
)

// Keep older function names but make them use BoltDB under the hood to avoid large changes across the codebase.

type fakeRedisClient struct {
	b *BoltClient
}

var (
	fakeClient *fakeRedisClient
	fakeOnce   sync.Once
)

// GetRedisClient returns an adapter that satisfies the minimal methods callers expect.
func GetRedisClient() *fakeRedisClient {
	fakeOnce.Do(func() {
		b, err := GetBoltClient()
		if err != nil {
			panic(fmt.Sprintf("failed to open bolt db: %v", err))
		}
		fakeClient = &fakeRedisClient{b: b}
	})
	return fakeClient
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
