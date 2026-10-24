package room

import (
	"fmt"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// DuelStart is the lobby's fully assigned two-seat setup. The lobby owns its
// serialization and passes only validated participants and frozen hint admins.
// Seats retain lobby order; FirstPlayer selects the starter, not the seating.
type DuelStart struct {
	Seats       [2]store.Seat
	FirstPlayer *int
	Creator     string
	HintAdmins  []string
	Hints       *view.HintSettings
}

// StartMember is a lobby connection and its original device claim. The Duel
// actor revalidates claims against the carried seat epochs before it admits it.
type StartMember struct {
	Client *Client
	Claim  string
}

// StartedDuel has not started running yet: Swap starts Handle only after the
// conditional file replacement. Later detaches belong to the new actor.
type StartedDuel struct {
	Handle *Handle
	Detach func(*Client)
	Abort  func()
}

// StartDuel builds the same engine, history and schema-1 room as ordinary
// Duel creation, but under the lobby's existing code, with no unclaimed seat.
func (m *Manager) StartDuel(code, from string, spec DuelStart, members []StartMember) (StartedDuel, error) {
	writer, ok := m.cfg.Store.(interface {
		ReplaceDuel(string, *store.File) error
	})
	if !ok {
		return StartedDuel{}, ErrStorage
	}
	humans := 0
	seen := map[string]bool{}
	for _, s := range spec.Seats {
		switch s.Kind {
		case store.SeatKindHuman:
			if s.UserID == "" || seen[s.UserID] || s.Epoch < 0 {
				return StartedDuel{}, ErrInvalidSpec
			}
			seen[s.UserID] = true
			humans++
		case store.SeatKindBot:
			if s.UserID != "" || s.Bot != store.BotEasy && s.Bot != store.BotNormal && s.Bot != store.BotHard {
				return StartedDuel{}, ErrInvalidSpec
			}
		default:
			return StartedDuel{}, ErrInvalidSpec
		}
	}
	if humans == 0 || len(spec.HintAdmins) == 0 || len(spec.HintAdmins) > 33 || spec.Creator == "" {
		return StartedDuel{}, ErrInvalidSpec
	}
	admins := slices.Clone(spec.HintAdmins)
	slices.Sort(admins)
	for i, user := range admins {
		if user == "" || len(user) > 64 || i > 0 && admins[i-1] == user {
			return StartedDuel{}, ErrInvalidSpec
		}
	}
	opts := engine.Options{}
	if spec.FirstPlayer != nil {
		switch *spec.FirstPlayer {
		case 0:
			opts.FirstPlayer = engine.FirstPlayerSeat0
		case 1:
			opts.FirstPlayer = engine.FirstPlayerSeat1
		default:
			return StartedDuel{}, ErrInvalidSpec
		}
	}
	f := store.New(code, store.ModeOnline, opts, m.cfg.Seed(), m.cfg.Clock.Now())
	f.Seats = spec.Seats
	f.Admin = spec.Creator
	f.HintAdmins = admins
	if spec.Hints != nil {
		h := *spec.Hints
		f.Hints = &h
	}
	state, hist, err := replayKeys(f, f.Actions)
	if err != nil {
		return StartedDuel{}, fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}
	r := newRoom(m, f, state, hist)
	for _, mb := range members {
		if mb.Client == nil {
			r.cancel()
			return StartedDuel{}, ErrInvalidSpec
		}
		var claim *string
		if mb.Claim != "" {
			claim = &mb.Claim
		}
		mem, _, err := r.admit(mb.Client, claim)
		if err != nil {
			r.cancel()
			return StartedDuel{}, err
		}
		r.clients[mb.Client] = mem
	}
	r.handoff = true
	if err := r.save(func(f *store.File) error { return writer.ReplaceDuel(from, f) }); err != nil {
		r.dirty = false
		r.cancel()
		return StartedDuel{}, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	h := duelHandle(m, r)
	return StartedDuel{Handle: h, Detach: h.hooks.Detach, Abort: r.cancel}, nil
}
