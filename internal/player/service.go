package player

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

var (
	ErrRateLimited = errors.New("rate limited")
	ErrBadClaim    = errors.New("invalid name or claim code")
)

// Service handles register / claim / reclaim against a Store.
type Service struct {
	Store *persist.Store

	mu    sync.Mutex
	hits  map[string][]time.Time // key → timestamps
	limit int
	window time.Duration
}

// NewService wraps a persist store with basic rate limits.
func NewService(store *persist.Store) *Service {
	return &Service{
		Store:  store,
		hits:   map[string][]time.Time{},
		limit:  10,
		window: time.Minute,
	}
}

// RegisterResult is returned after creating a Player.
type RegisterResult struct {
	Player    persist.Player
	ClaimCode string // plaintext once; never stored
	Display   string // XXXX-XXXX-XXXX
}

// Register creates a Player and returns a one-time Claim Code.
func (s *Service) Register(name, ip, sessionID string, now time.Time) (RegisterResult, error) {
	if err := s.allow("reg:"+ip, now); err != nil {
		return RegisterResult{}, err
	}
	n, err := NormalizeName(name)
	if err != nil {
		return RegisterResult{}, err
	}
	if err := s.allow("name:"+NameKey(n), now); err != nil {
		return RegisterResult{}, err
	}
	code, err := GenerateClaimCode()
	if err != nil {
		return RegisterResult{}, err
	}
	hash, err := HashClaimCode(code)
	if err != nil {
		return RegisterResult{}, err
	}
	id, err := newID()
	if err != nil {
		return RegisterResult{}, err
	}
	p := persist.Player{
		ID:              id,
		Name:            n,
		NameKey:         NameKey(n),
		ClaimHash:       hash,
		ActiveSessionID: sessionID,
		CreatedAt:       now.UTC(),
	}
	if err := s.Store.CreatePlayer(p); err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{
		Player:    p,
		ClaimCode: code,
		Display:   FormatClaimCode(code),
	}, nil
}

// Claim re-enters an existing Player with name + Claim Code.
// Per-name rate limit applies only to failed verifications so attackers cannot
// lock out the owner by spamming wrong codes before the correct one is tried.
// Successful claims do not count against the name limit. IP flood limit still applies first.
func (s *Service) Claim(name, claimCode, ip, sessionID string, now time.Time) (persist.Player, error) {
	if err := s.allow("claim:"+ip, now); err != nil {
		return persist.Player{}, err
	}
	n, err := NormalizeName(name)
	if err != nil {
		return persist.Player{}, ErrBadClaim
	}
	nameKey := "name:" + NameKey(n)
	p, err := s.Store.GetPlayerByNameKey(NameKey(n))
	if err != nil {
		if err := s.allow(nameKey, now); err != nil {
			return persist.Player{}, err
		}
		return persist.Player{}, ErrBadClaim
	}
	if !VerifyClaimCode(claimCode, p.ClaimHash) {
		if err := s.allow(nameKey, now); err != nil {
			return persist.Player{}, err
		}
		return persist.Player{}, ErrBadClaim
	}
	// Correct code still respects an active failed-attempt lockout for this name.
	if s.limited(nameKey, now) {
		return persist.Player{}, ErrRateLimited
	}
	if err := s.Store.SetActiveSession(p.ID, sessionID); err != nil {
		return persist.Player{}, err
	}
	p.ActiveSessionID = sessionID
	return p, nil
}

func (s *Service) allow(key string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := s.trimLocked(key, now)
	if len(keep) >= s.limit {
		s.hits[key] = keep
		return ErrRateLimited
	}
	s.hits[key] = append(keep, now)
	return nil
}

func (s *Service) limited(key string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := s.trimLocked(key, now)
	s.hits[key] = keep
	return len(keep) >= s.limit
}

func (s *Service) trimLocked(key string, now time.Time) []time.Time {
	cut := now.Add(-s.window)
	var keep []time.Time
	for _, t := range s.hits[key] {
		if t.After(cut) {
			keep = append(keep, t)
		}
	}
	return keep
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("p_%x", b[:]), nil
}
