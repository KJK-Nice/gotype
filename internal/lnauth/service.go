package lnauth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
)

// Service coordinates LNURL-auth challenges and Player session binding.
type Service struct {
	publicURL string
	store     challengeStore
	players   *player.Service
}

// NewService wires challenge storage and the player service.
func NewService(players *player.Service) *Service {
	return &Service{
		publicURL: strings.TrimRight(strings.TrimSpace(os.Getenv("GOTYPE_PUBLIC_URL")), "/"),
		store:     newChallengeStoreFromEnv(),
		players:   players,
	}
}

// Enabled reports whether wallet login can be offered (public callback URL set).
func (s *Service) Enabled() bool {
	return s.publicURL != ""
}

// PublicURL returns the configured public base URL.
func (s *Service) PublicURL() string {
	return s.publicURL
}

// StartResult is returned when the TUI begins a wallet auth flow.
type StartResult struct {
	K1    string
	LNURL string
}

// Start creates a challenge and LNURL QR payload.
func (s *Service) Start(sessionID string, action Action, playerID string, now time.Time) (StartResult, error) {
	if !s.Enabled() {
		return StartResult{}, fmt.Errorf("wallet login unavailable")
	}
	if sessionID == "" {
		return StartResult{}, fmt.Errorf("session id required")
	}
	switch action {
	case ActionRegister, ActionLogin:
	case ActionLink:
		if playerID == "" {
			return StartResult{}, fmt.Errorf("player id required for link")
		}
	default:
		return StartResult{}, fmt.Errorf("unknown action %q", action)
	}
	k1, err := GenerateK1()
	if err != nil {
		return StartResult{}, err
	}
	callback, err := AuthURL(s.publicURL, k1, action, sessionID)
	if err != nil {
		return StartResult{}, err
	}
	lnurl, err := EncodeLNURL(callback)
	if err != nil {
		return StartResult{}, err
	}
	c := Challenge{
		K1:        k1,
		SessionID: sessionID,
		Action:    action,
		PlayerID:  playerID,
		State:     StatePending,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.put(c); err != nil {
		return StartResult{}, err
	}
	return StartResult{K1: k1, LNURL: lnurl}, nil
}

// HandleCallback processes the wallet GET with sig and key (LUD-04).
func (s *Service) HandleCallback(k1, sig, key, ip string, now time.Time) error {
	if k1 == "" || sig == "" || key == "" {
		return ErrBadChallenge
	}
	linkingKey, err := NormalizeLinkingKey(key)
	if err != nil {
		return err
	}
	ok, err := Verify(k1, sig, key)
	if err != nil || !ok {
		return ErrBadChallenge
	}
	_, err = s.store.update(k1, func(c *Challenge) error {
		if c.State != StatePending {
			return ErrUsed
		}
		c.LinkingKey = linkingKey
		switch c.Action {
		case ActionLink:
			if err := s.players.LinkWallet(c.PlayerID, linkingKey); err != nil {
				c.State = StateError
				c.Err = err.Error()
				return nil
			}
			if err := s.players.Store.SetActiveSession(c.PlayerID, c.SessionID); err != nil {
				c.State = StateError
				c.Err = err.Error()
				return nil
			}
			p, err := s.players.Store.GetPlayer(c.PlayerID)
			if err != nil {
				c.State = StateError
				c.Err = err.Error()
				return nil
			}
			c.PlayerID = p.ID
			c.Name = p.Name
			c.State = StateOK
		case ActionRegister, ActionLogin:
			p, err := s.players.Store.GetPlayerByLinkingKey(linkingKey)
			if err == nil {
				if _, err := s.players.LoginWithLinkingKey(linkingKey, ip, c.SessionID, now); err != nil {
					c.State = StateError
					c.Err = err.Error()
					return nil
				}
				c.PlayerID = p.ID
				c.Name = p.Name
				c.State = StateOK
				return nil
			}
			if err != persist.ErrNotFound {
				c.State = StateError
				c.Err = err.Error()
				return nil
			}
			c.State = StateVerified
		default:
			return fmt.Errorf("unknown action")
		}
		return nil
	})
	return err
}

// Status returns challenge progress for TUI polling.
func (s *Service) Status(k1 string) (Status, error) {
	c, ok, err := s.store.get(k1)
	if err != nil {
		return Status{}, err
	}
	if !ok {
		return Status{}, ErrNotFound
	}
	return challengeStatus(c), nil
}

// CompleteRegister finishes a new-wallet flow after the user picks a display name.
func (s *Service) CompleteRegister(k1, name, ip string, now time.Time) (persist.Player, error) {
	c, err := s.store.update(k1, func(c *Challenge) error {
		if c.State != StateVerified {
			return ErrBadChallenge
		}
		if c.LinkingKey == "" {
			return ErrBadChallenge
		}
		return nil
	})
	if err != nil {
		return persist.Player{}, err
	}
	p, err := s.players.RegisterWithLinkingKey(name, c.LinkingKey, ip, c.SessionID, now)
	if err != nil {
		_, _ = s.store.update(k1, func(ch *Challenge) error {
			ch.State = StateError
			ch.Err = err.Error()
			return nil
		})
		return persist.Player{}, err
	}
	_, _ = s.store.update(k1, func(ch *Challenge) error {
		ch.State = StateOK
		ch.PlayerID = p.ID
		ch.Name = p.Name
		return nil
	})
	return p, nil
}

// LoginAfterVerify logs in a returning wallet when challenge is already verified/ok.
func (s *Service) LoginAfterVerify(k1, ip string, now time.Time) (persist.Player, error) {
	c, ok, err := s.store.get(k1)
	if err != nil {
		return persist.Player{}, err
	}
	if !ok {
		return persist.Player{}, ErrNotFound
	}
	if c.State == StateOK && c.PlayerID != "" {
		return s.players.Store.GetPlayer(c.PlayerID)
	}
	if c.LinkingKey == "" {
		return persist.Player{}, ErrBadChallenge
	}
	return s.players.LoginWithLinkingKey(c.LinkingKey, ip, c.SessionID, now)
}
