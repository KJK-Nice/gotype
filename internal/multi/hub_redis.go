package multi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var errHubConflict = errors.New("hub: redis conflict")

type hubRedisStore struct {
	rdb      *redis.Client
	roomsKey string
	roomKey  func(code string) string
}

func newHubRedisStore(rawURL string) (*hubRedisStore, error) {
	opts, err := redis.ParseURL(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("hub redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("hub redis ping: %w", err)
	}
	prefix := strings.TrimSpace(os.Getenv("GOTYPE_REDIS_KEY"))
	if prefix == "" {
		prefix = "gotype"
	}
	return &hubRedisStore{
		rdb:      rdb,
		roomsKey: prefix + ":hub:rooms",
		roomKey:  func(code string) string { return prefix + ":hub:room:" + code },
	}, nil
}

func (rs *hubRedisStore) loadInto(h *Hub) (map[string]struct{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	codes, err := rs.rdb.SMembers(ctx, rs.roomsKey).Result()
	if err != nil {
		return nil, err
	}
	rooms := make(map[string]*Room, len(codes))
	prev := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		prev[code] = struct{}{}
		room, err := rs.loadRoom(ctx, code)
		if err != nil {
			return nil, err
		}
		if room != nil {
			rooms[code] = room
		}
	}
	h.rooms = rooms
	h.byPlayer = buildByPlayer(rooms)
	return prev, nil
}

func (rs *hubRedisStore) loadRoom(ctx context.Context, code string) (*Room, error) {
	b, err := rs.rdb.Get(ctx, rs.roomKey(code)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var room Room
	if err := json.Unmarshal(b, &room); err != nil {
		return nil, err
	}
	if room.Players == nil {
		room.Players = map[string]*Player{}
	}
	return &room, nil
}

func buildByPlayer(rooms map[string]*Room) map[string]string {
	out := make(map[string]string)
	for code, room := range rooms {
		for pid := range room.Players {
			out[pid] = code
		}
	}
	return out
}

func (rs *hubRedisStore) saveFrom(h *Hub, prev map[string]struct{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	nowCodes := make(map[string]struct{}, len(h.rooms))
	for code, room := range h.rooms {
		nowCodes[code] = struct{}{}
		if err := rs.putRoom(ctx, code, room); err != nil {
			return err
		}
	}
	for code := range prev {
		if _, ok := nowCodes[code]; ok {
			continue
		}
		if err := rs.deleteRoom(ctx, code); err != nil {
			return err
		}
	}
	return nil
}

func (rs *hubRedisStore) putRoom(ctx context.Context, code string, room *Room) error {
	const maxRetries = 8
	for range maxRetries {
		err := rs.rdb.Watch(ctx, func(tx *redis.Tx) error {
			payload, err := json.Marshal(room)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, rs.roomKey(code), payload, 0)
				pipe.SAdd(ctx, rs.roomsKey, code)
				return nil
			})
			return err
		}, rs.roomKey(code))
		if err == nil {
			return nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}
	return errHubConflict
}

func (rs *hubRedisStore) deleteRoom(ctx context.Context, code string) error {
	pipe := rs.rdb.Pipeline()
	pipe.Del(ctx, rs.roomKey(code))
	pipe.SRem(ctx, rs.roomsKey, code)
	_, err := pipe.Exec(ctx)
	return err
}

// NewHubFromEnv returns a Hub backed by Redis when REDIS_URL is set.
func NewHubFromEnv() (*Hub, error) {
	h := NewHub()
	if u := strings.TrimSpace(os.Getenv("REDIS_URL")); u != "" {
		rs, err := newHubRedisStore(u)
		if err != nil {
			return nil, err
		}
		h.redis = rs
	}
	return h, nil
}
