package room

import (
	"errors"
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// DuelRematch is the original assignment and current display preferences of a
// shared-lobby Duel game. Its successor reserves humans and resets all roles.
type DuelRematch struct {
	Parent, Creator string
	Seats           [2]store.Seat
	FirstPlayer     engine.FirstPlayerRule
	Hints           view.HintSettings
}

// DuelSuccessors is the shared lobby's narrow rematch seam. ErrExists means
// another room took a reserved code; all other errors leave that code for retry.
type DuelSuccessors interface {
	Reserve(*Manager) (string, error)
	RematchDuel(*Manager, string, DuelRematch) error
}

// SetDuelSuccessors binds the lobby at composition time. Legacy Duel games do
// not use it; their one-tap rematches keep the original creation path.
func (m *Manager) SetDuelSuccessors(s DuelSuccessors) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.duelSuccessors = s
}
func (m *Manager) successors() DuelSuccessors {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.duelSuccessors
}

func (r *Room) rematchLobby(c *Client, mem member) (string, error) {
	maker := r.m.successors()
	if maker == nil {
		return "", ErrInvalidSpec
	}
	for range 3 {
		if r.rematch == "" {
			if c.AllowCreate != nil && !c.AllowCreate() {
				return "", ErrRateLimited
			}
			code, err := maker.Reserve(r.m)
			if err != nil {
				return "", err
			}
			r.rematch = code
		}
		// The parent owns the code durably before any successor can be created.
		if err := r.save(r.m.cfg.Store.Save); err != nil {
			return "", ErrStorage
		}
		request := DuelRematch{Parent: r.code, Creator: mem.user, Seats: r.file.Seats, FirstPlayer: r.file.Options.FirstPlayer, Hints: *r.file.Hints}
		if err := maker.RematchDuel(r.m, r.rematch, request); err != nil {
			if errors.Is(err, store.ErrExists) {
				r.rematch = ""
				continue
			}
			return "", fmt.Errorf("%w: %v", ErrStorage, err)
		}
		for cl, member := range r.clients {
			cl.Out.Push(r.rematchCreated(cl, member))
		}
		return r.rematch, nil
	}
	// A final collision must not be persisted as a successor on the next save.
	if err := r.save(r.m.cfg.Store.Save); err != nil {
		return "", ErrStorage
	}
	return "", ErrStorage
}
