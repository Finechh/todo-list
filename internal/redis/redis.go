package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"todo_list/internal/metrics"
)

var ErrCacheMiss = errors.New("cache miss")

type CacheStore interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Close() error
}

type Cache struct {
	rdb *redis.Client
}

func NewCache(ctx context.Context, addr string) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := rdb.Ping(cctx).Err(); err != nil {
		return nil, err
	}
	return &Cache{rdb: rdb}, nil
}

func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	start := time.Now()
	defer metrics.ObserveBackendDuration("redis", "set", time.Since(start))
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		metrics.IncBackendError("redis", "set")
		return err
	}
	metrics.IncBackendSuccess("redis", "set")
	return nil
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer metrics.ObserveBackendDuration("redis", "get", time.Since(start))
	result, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			metrics.CacheMiss.WithLabelValues("redis", "cache_miss")
		}
		metrics.IncBackendError("redis", "get")
		return "", err
	}
	metrics.IncBackendSuccess("redis", "get")
	return result, nil
}

func (c *Cache) Del(ctx context.Context, key string) error {
	start := time.Now()
	defer metrics.ObserveBackendDuration("redis", "del", time.Since(start))
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		metrics.IncBackendError("redis", "del")
		return err
	}
	metrics.IncBackendSuccess("redis", "del")
	return nil
}

func (c *Cache) Close() error {
	return c.rdb.Close()

}
