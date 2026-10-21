package room

import (
	"context"
	"errors"
	"io/fs"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// DuelKind registers the existing constructor, storage operations and actor.
func DuelKind() Kind {
	return Kind{ID: Duel, Create: func(m *Manager, c *Client, req wire.Create) (string, error) {
		f, err := m.newFile(c, req)
		if err != nil {
			return "", err
		}
		return m.createRoom(f)
	}, Check: checkDuelJoin, Load: loadDuel}
}

func checkDuelJoin(m *Manager, code string, c *Client, claim *string) error {
	hdr, err := m.cfg.Store.PeekHeader(code)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrRoomNotFound
		}
		m.log.Warn("reading a room's header failed", "room", code, "err", err)
		return ErrStorage
	}
	if hdr.Status == store.StatusArchived {
		return &archivedRoomError{reason: hdr.ArchiveReason}
	}
	user := c.UserID
	if claim != nil {
		u, ok := claimHolder(m.cfg.Claims, code, hdr.Seats, *claim)
		if !ok {
			return ErrBadClaim
		}
		user = u
	}
	if closedTo(hdr.Mode, hdr.Seats, user) {
		return ErrForbidden
	}
	return nil
}

func loadDuel(m *Manager, code string) (*Handle, error) {
	f, s, err := m.cfg.Store.Load(code)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !errors.Is(err, store.ErrRestoreWrite) {
			return nil, ErrRoomNotFound
		}
		m.log.Warn("loading a room failed", "room", code, "err", err)
		if errors.Is(err, store.ErrArchived) && f != nil {
			return nil, &archivedRoomError{reason: f.ArchiveReason}
		}
		return nil, ErrStorage
	}
	// The store checked keys and refreshed the hash. Duel still rebuilds its
	// own seats/events history for undo, exactly as before the seam.
	_, hist, err := replayKeys(f, f.Actions)
	if err != nil {
		m.log.Error("a stored room does not replay through the legal actions", "room", code, "err", err)
		return nil, ErrStorage
	}
	r := newRoom(m, f, s, hist)
	h := duelHandle(m, r)
	// Log after successful registration, as the old load path did.
	actions := len(f.Actions)
	h.afterStart = func() { r.log.Info("loaded room", "actions", actions) }
	return h, nil
}

func duelHandle(m *Manager, r *Room) *Handle {
	h := &Handle{owner: m, kind: Duel, code: r.code, duel: r,
		evictable: r.evictable, idleSince: r.idleSince, evicting: r.evicting}
	h.hooks = HandleConfig{
		Run: r.run, Cancel: r.cancel, Deliver: r.post,
		Attach: func(ctx context.Context, c *Client, claim *string) error {
			res, err := request(ctx, r, func(reply chan error) any { return msgAttach{c: c, claim: claim, reply: reply} })
			if err != nil && !errors.Is(err, errRoomGone) {
				// The caller stopped waiting after attach may have entered the inbox.
				// Queue its detach behind it, preserving the existing ghost prevention.
				_ = r.post(context.Background(), msgDetach{c: c})
			}
			if err != nil {
				return err
			}
			return res
		},
		Detach: func(c *Client) { _ = r.post(context.Background(), msgDetach{c: c}) },
		Evict: func(ctx context.Context) error {
			_, err := request(ctx, r, func(reply chan bool) any { return msgEvict{reply: reply} })
			return err
		},
	}
	// Duel's unchanged run method balances this registration itself. Other
	// handles use NewHandle's wrapper, which balances their Run callback.
	h.start = func() { m.wg.Add(1); go r.run() }
	return h
}
