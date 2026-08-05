package player

import (
	"time"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

// RegisterWithLinkingKey creates a Player bound to a wallet linking key.
func (s *Service) RegisterWithLinkingKey(name, linkingKey, ip, sessionID string, now time.Time) (persist.Player, error) {
	if err := s.allow("reg:"+ip, now); err != nil {
		return persist.Player{}, err
	}
	n, err := NormalizeName(name)
	if err != nil {
		return persist.Player{}, err
	}
	if err := s.allow("name:"+NameKey(n), now); err != nil {
		return persist.Player{}, err
	}
	lk, err := NormalizeLinkingKey(linkingKey)
	if err != nil {
		return persist.Player{}, err
	}
	if _, err := s.Store.GetPlayerByLinkingKey(lk); err == nil {
		return persist.Player{}, persist.ErrLinkingKeyTaken
	} else if err != persist.ErrNotFound {
		return persist.Player{}, err
	}
	id, err := newID()
	if err != nil {
		return persist.Player{}, err
	}
	p := persist.Player{
		ID:              id,
		Name:            n,
		NameKey:         NameKey(n),
		LinkingKey:      lk,
		ActiveSessionID: sessionID,
		CreatedAt:       now.UTC(),
	}
	if err := s.Store.CreatePlayer(p); err != nil {
		return persist.Player{}, err
	}
	return p, nil
}

// LoginWithLinkingKey reclaims a Player via wallet linking key.
func (s *Service) LoginWithLinkingKey(linkingKey, ip, sessionID string, now time.Time) (persist.Player, error) {
	if err := s.allow("claim:"+ip, now); err != nil {
		return persist.Player{}, err
	}
	lk, err := NormalizeLinkingKey(linkingKey)
	if err != nil {
		return persist.Player{}, ErrBadClaim
	}
	p, err := s.Store.GetPlayerByLinkingKey(lk)
	if err != nil {
		return persist.Player{}, ErrBadClaim
	}
	if err := s.Store.SetActiveSession(p.ID, sessionID); err != nil {
		return persist.Player{}, err
	}
	p.ActiveSessionID = sessionID
	return p, nil
}

// LinkWallet attaches a linking key to an existing Player.
func (s *Service) LinkWallet(playerID, linkingKey string) error {
	lk, err := NormalizeLinkingKey(linkingKey)
	if err != nil {
		return err
	}
	return s.Store.SetLinkingKey(playerID, lk)
}
