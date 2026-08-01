package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/redis/go-redis/v9"
)

var errNoSave = errors.New("persist: no save")

type transactionalStorage interface {
	storage
	stateless() bool
	update(func(*db) error) error
}

func (r *redisStorage) stateless() bool { return true }

func (f *fileStorage) stateless() bool { return false }

func (r *redisStorage) update(fn func(*db) error) error {
	const maxRetries = 12
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for range maxRetries {
		err := r.rdb.Watch(ctx, func(tx *redis.Tx) error {
			b, err := tx.Get(ctx, r.key).Bytes()
			var d db
			switch {
			case err == redis.Nil:
				d = emptyDB()
			case err != nil:
				return err
			default:
				if err := json.Unmarshal(b, &d); err != nil {
					return err
				}
				normalizeDB(&d)
			}
			if err := fn(&d); err != nil {
				if errors.Is(err, errNoSave) {
					return nil
				}
				return err
			}
			payload, err := json.Marshal(d)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, r.key, payload, 0)
				return nil
			})
			return err
		}, r.key)
		if err == nil {
			return nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}
	return fmt.Errorf("redis: transaction retries exhausted")
}

func (s *Store) isStateless() bool {
	ts, ok := s.st.(transactionalStorage)
	return ok && ts.stateless()
}

func (s *Store) mutate(fn func(*db) error) error {
	if ts, ok := s.st.(transactionalStorage); ok && ts.stateless() {
		return ts.update(fn)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.data); err != nil {
		if errors.Is(err, errNoSave) {
			return nil
		}
		return err
	}
	return s.saveLocked()
}

func queryStore[T any](s *Store, fn func(*db) (T, error)) (T, error) {
	var zero T
	if s.isStateless() {
		d, err := s.st.load()
		if err != nil {
			return zero, err
		}
		return fn(&d)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(&s.data)
}

func ensureSeason(d *db, now time.Time) error {
	now = now.UTC()
	for _, se := range d.Seasons {
		if !now.Before(se.StartsAt) && now.Before(se.EndsAt) {
			return nil
		}
	}
	id := d.NextSeason
	if id < 1 {
		id = 1
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	se := Season{
		ID:       id,
		StartsAt: start,
		EndsAt:   start.AddDate(0, 0, catalog.SeasonLengthDays),
		TrackRef: "v1",
	}
	d.Seasons[id] = se
	d.NextSeason = id + 1
	return nil
}
