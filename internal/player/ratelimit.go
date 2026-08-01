package player

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type rateLimiter interface {
	allow(key string, now time.Time, limit int, window time.Duration) error
	limited(key string, now time.Time, limit int, window time.Duration) bool
}

type memoryLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func (m *memoryLimiter) allow(key string, now time.Time, limit int, window time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	keep := trimHits(m.hits[key], now, window)
	if len(keep) >= limit {
		m.hits[key] = keep
		return ErrRateLimited
	}
	m.hits[key] = append(keep, now)
	return nil
}

func (m *memoryLimiter) limited(key string, now time.Time, limit int, window time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	keep := trimHits(m.hits[key], now, window)
	m.hits[key] = keep
	return len(keep) >= limit
}

type redisLimiter struct {
	rdb *redis.Client
}

func newRedisLimiter(rawURL string) (*redisLimiter, error) {
	opts, err := redis.ParseURL(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return &redisLimiter{rdb: rdb}, nil
}

func (r *redisLimiter) key(k string) string {
	return "gotype:ratelimit:" + k
}

func (r *redisLimiter) allow(key string, now time.Time, limit int, window time.Duration) error {
	n, err := r.count(key, now, window)
	if err != nil {
		return err
	}
	if n >= limit {
		return ErrRateLimited
	}
	return r.addHit(key, now, window)
}

func (r *redisLimiter) limited(key string, now time.Time, limit int, window time.Duration) bool {
	n, err := r.count(key, now, window)
	if err != nil {
		return false
	}
	return n >= limit
}

func (r *redisLimiter) addHit(key string, now time.Time, window time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	k := r.key(key)
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe := r.rdb.Pipeline()
	pipe.ZAdd(ctx, k, redis.Z{Score: float64(now.UnixNano()), Member: member})
	pipe.Expire(ctx, k, window+time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisLimiter) count(key string, now time.Time, window time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	k := r.key(key)
	cutoff := float64(now.Add(-window).UnixNano())
	pipe := r.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, k, "0", fmt.Sprintf("%f", cutoff))
	pipe.ZCard(ctx, k)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(cmds[1].(*redis.IntCmd).Val()), nil
}

func trimHits(xs []time.Time, now time.Time, window time.Duration) []time.Time {
	cut := now.Add(-window)
	var keep []time.Time
	for _, t := range xs {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	return keep
}

func newRateLimiterFromEnv() rateLimiter {
	if u := strings.TrimSpace(os.Getenv("REDIS_URL")); u != "" {
		if rl, err := newRedisLimiter(u); err == nil {
			return rl
		}
	}
	return &memoryLimiter{hits: map[string][]time.Time{}}
}
