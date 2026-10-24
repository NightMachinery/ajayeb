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

// DuelSuccessors is the shared lobby's narrow rematch seam. ReserveDuel
// gives a fresh code, which the game stores before RematchDuel makes the
// lobby under it. ErrExists from RematchDuel means another room took the
// reserved code; all other errors leave that code for retry.
type DuelSuccessors interface {
	ReserveDuel(*Manager) (string, error)
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

// rematchLobby is doRematch for a shared game: its rematch is a two-seat
// lobby that the shared lobby makes (DuelSuccessors), as a table's is. The
// game first stores a reserved code (reserved), then has the lobby made
// under it, then keeps that code as its successor (rematch). From then on
// every request answers that code without asking the lobby again, since
// the lobby may have started its own game by then. Only a reservation that
// another room took before the lobby was made is given up for a new one.
//
// The code goes out (created frames, the answer) only once the file holds
// it as the successor, as a table commits its Successor first: a code
// handed out and then lost to a crash would let the lobby start under it
// while the reloaded game, finding no lobby there, made a second one. A
// failed write keeps the successor in memory and answers storage; the next
// request writes again before answering.
func (r *Room) rematchLobby(c *Client, mem member) (string, error) {
	maker := r.m.successors()
	if maker == nil {
		return "", ErrInvalidSpec
	}
	for range 3 {
		if r.rematch != "" {
			break
		}
		if r.reserved == "" {
			if c.AllowCreate != nil && !c.AllowCreate() {
				return "", ErrRateLimited
			}
			code, err := maker.ReserveDuel(r.m)
			if err != nil {
				return "", err
			}
			r.reserved = code
		}
		// The parent owns the code durably before any successor can be created.
		if err := r.save(r.m.cfg.Store.Save); err != nil {
			return "", ErrStorage
		}
		request := DuelRematch{Parent: r.code, Creator: mem.user, Seats: r.file.Seats, FirstPlayer: r.file.Options.FirstPlayer, Hints: *r.file.Hints}
		if err := maker.RematchDuel(r.m, r.reserved, request); err != nil {
			if errors.Is(err, store.ErrExists) {
				r.reserved = ""
				continue
			}
			return "", fmt.Errorf("%w: %v", ErrStorage, err)
		}
		r.rematch, r.reserved = r.reserved, ""
		r.dirty = true // the successor is not in the file yet
		r.log.Info("rematch", "code", r.rematch)
	}
	if r.rematch == "" {
		// A final collision must not be persisted as a successor on the next save.
		if err := r.save(r.m.cfg.Store.Save); err != nil {
			return "", ErrStorage
		}
		return "", ErrStorage
	}
	if r.dirty {
		if err := r.save(r.m.cfg.Store.Save); err != nil {
			return "", ErrStorage
		}
	}
	for cl, member := range r.clients {
		cl.Out.Push(r.rematchCreated(cl, member))
	}
	return r.rematch, nil
}
