package persist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRedisKey = "gotype:data"

type redisStorage struct {
	rdb *redis.Client
	key string
}

func (r *redisStorage) load() (db, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := r.rdb.Get(ctx, r.key).Bytes()
	if err == redis.Nil {
		return emptyDB(), nil
	}
	if err != nil {
		return db{}, err
	}
	var d db
	if err := json.Unmarshal(b, &d); err != nil {
		return db{}, err
	}
	normalizeDB(&d)
	return d, nil
}

func (r *redisStorage) save(d db) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, r.key, b, 0).Err()
}

// OpenRedis connects to Redis and loads the gotype document at key gotype:data.
func OpenRedis(rawURL string) (*Store, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("empty redis url")
	}
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	key := strings.TrimSpace(os.Getenv("GOTYPE_REDIS_KEY"))
	if key == "" {
		key = defaultRedisKey
	}
	return openWith(&redisStorage{rdb: rdb, key: key})
}

// OpenFromEnv prefers REDIS_URL, else GOTYPE_DATA_DIR / OS temp JSON file.
func OpenFromEnv(path string) (*Store, error) {
	if u := strings.TrimSpace(os.Getenv("REDIS_URL")); u != "" {
		return OpenRedis(u)
	}
	return Open(path)
}
