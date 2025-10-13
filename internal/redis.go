package internal

import (
	"context"
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisOnce   sync.Once
)

// GetRedisClient returns a singleton Redis client
func GetRedisClient() *redis.Client {
	redisOnce.Do(func() {
		addr := os.Getenv("REDIS_ADDR")
		if addr == "" {
			addr = "localhost:6379"
		}
		password := os.Getenv("REDIS_PASSWORD")
		db := 0
		redisClient = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		})
	})
	return redisClient
}

// PingRedis checks if Redis is reachable
func PingRedis(ctx context.Context) error {
	client := GetRedisClient()
	return client.Ping(ctx).Err()
}
