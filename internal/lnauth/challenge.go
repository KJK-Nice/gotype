package lnauth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const challengeTTL = 5 * time.Minute

// ChallengeState tracks wallet auth progress for one k1.
type ChallengeState string

const (
	StatePending  ChallengeState = "pending"
	StateVerified ChallengeState = "verified" // wallet ok, needs display name
	StateOK       ChallengeState = "ok"
	StateError    ChallengeState = "error"
)

// Challenge is stored while a wallet auth flow is in flight.
type Challenge struct {
	K1         string         `json:"k1"`
	SessionID  string         `json:"session_id"`
	Action     Action         `json:"action"`
	PlayerID   string         `json:"player_id,omitempty"` // link flow
	State      ChallengeState `json:"state"`
	LinkingKey string         `json:"linking_key,omitempty"`
	Name       string         `json:"name,omitempty"`
	Err        string         `json:"err,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Status is returned to the TUI poller.
type Status struct {
	State      ChallengeState `json:"state"`
	LinkingKey string         `json:"linking_key,omitempty"`
	PlayerID   string         `json:"player_id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Err        string         `json:"err,omitempty"`
}

type challengeStore interface {
	put(c Challenge) error
	get(k1 string) (Challenge, bool, error)
	update(k1 string, fn func(*Challenge) error) (Challenge, error)
}

type memoryChallengeStore struct {
	mu sync.Mutex
	m  map[string]Challenge
}

func newMemoryChallengeStore() *memoryChallengeStore {
	return &memoryChallengeStore{m: map[string]Challenge{}}
}

func (s *memoryChallengeStore) put(c Challenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[c.K1] = c
	return nil
}

func (s *memoryChallengeStore) get(k1 string) (Challenge, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.m[k1]
	if !ok {
		return Challenge{}, false, nil
	}
	if time.Since(c.CreatedAt) > challengeTTL {
		delete(s.m, k1)
		return Challenge{}, false, nil
	}
	return c, true, nil
}

func (s *memoryChallengeStore) update(k1 string, fn func(*Challenge) error) (Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.m[k1]
	if !ok {
		return Challenge{}, ErrNotFound
	}
	if time.Since(c.CreatedAt) > challengeTTL {
		delete(s.m, k1)
		return Challenge{}, ErrNotFound
	}
	if err := fn(&c); err != nil {
		return Challenge{}, err
	}
	s.m[k1] = c
	return c, nil
}

type redisChallengeStore struct {
	rdb *redis.Client
}

func newRedisChallengeStore(rawURL string) (*redisChallengeStore, error) {
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
	return &redisChallengeStore{rdb: rdb}, nil
}

func (s *redisChallengeStore) key(k1 string) string {
	return "gotype:lnauth:k1:" + k1
}

func (s *redisChallengeStore) put(c Challenge) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, s.key(c.K1), b, challengeTTL).Err()
}

func (s *redisChallengeStore) get(k1 string) (Challenge, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b, err := s.rdb.Get(ctx, s.key(k1)).Bytes()
	if err == redis.Nil {
		return Challenge{}, false, nil
	}
	if err != nil {
		return Challenge{}, false, err
	}
	var c Challenge
	if err := json.Unmarshal(b, &c); err != nil {
		return Challenge{}, false, err
	}
	return c, true, nil
}

func (s *redisChallengeStore) update(k1 string, fn func(*Challenge) error) (Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	k := s.key(k1)
	b, err := s.rdb.Get(ctx, k).Bytes()
	if err == redis.Nil {
		return Challenge{}, ErrNotFound
	}
	if err != nil {
		return Challenge{}, err
	}
	var c Challenge
	if err := json.Unmarshal(b, &c); err != nil {
		return Challenge{}, err
	}
	if err := fn(&c); err != nil {
		return Challenge{}, err
	}
	b, err = json.Marshal(c)
	if err != nil {
		return Challenge{}, err
	}
	if err := s.rdb.Set(ctx, k, b, challengeTTL).Err(); err != nil {
		return Challenge{}, err
	}
	return c, nil
}

func newChallengeStoreFromEnv() challengeStore {
	if u := strings.TrimSpace(os.Getenv("REDIS_URL")); u != "" {
		if s, err := newRedisChallengeStore(u); err == nil {
			return s
		}
	}
	return newMemoryChallengeStore()
}

var (
	ErrNotFound     = errors.New("challenge not found")
	ErrBadChallenge = errors.New("invalid challenge")
	ErrUsed         = errors.New("challenge already used")
)

func challengeStatus(c Challenge) Status {
	return Status{
		State:      c.State,
		LinkingKey: c.LinkingKey,
		PlayerID:   c.PlayerID,
		Name:       c.Name,
		Err:        c.Err,
	}
}
