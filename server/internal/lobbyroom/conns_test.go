package lobbyroom

import (
	"errors"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// tabs joins n more connections of u, named after it.
func (f *fixture) tabs(code, u string, n int) {
	f.t.Helper()
	for i := range n {
		f.join(f.conn(u, u+"-tab-"+strconv.Itoa(i), u), code)
	}
}

// TestLobbyNinthConnectionIsRefused: a user holds at most 8 connections to
// a lobby, with a slot or without; the ninth is refused with room-full.
// Before, one user could hold all 32.
func TestLobbyNinthConnectionIsRefused(t *testing.T) {
	f := newFixture(t, nil)
	code := f.create(f.conn(user(0), "c0", "P0"), 2, wire.LobbyOptions{})
	f.join(f.conn(user(1), "c1", "P1"), code)
	f.join(f.conn(user(2), "c2", "P2"), code)
	for _, u := range []string{user(0), user(2)} {
		f.tabs(code, u, tableroom.DefaultMaxUserClients-1)
		if err := f.m.Join(f.ctx(), f.conn(u, u+"-ninth", u).c, code); !errors.Is(err, room.ErrRoomFull) {
			t.Fatalf("%s's ninth connection: %v", u, err)
		}
	}
}

// TestLobbySlotHolderIsNotLockedOut: watchers fill the lobby's connections,
// and a player holding a slot still gets a new connection in, while a
// watcher does not. Before, a full lobby refused the player too.
func TestLobbySlotHolderIsNotLockedOut(t *testing.T) {
	f := newFixture(t, nil)
	code := f.create(f.conn(user(0), "c0", "P0"), 2, wire.LobbyOptions{})
	f.join(f.conn(user(1), "c1", "P1"), code)
	left := f.lobbies.cfg.MaxClients - 2
	for u := 2; left > 0; u++ {
		n := min(left, tableroom.DefaultMaxUserClients)
		f.tabs(code, user(u), n)
		left -= n
	}
	if err := f.m.Join(f.ctx(), f.conn("late-watcher", "late", "Late").c, code); !errors.Is(err, room.ErrRoomFull) {
		t.Fatalf("a watcher in a full lobby: %v", err)
	}
	f.join(f.conn(user(1), "c1-tab", "P1"), code)
}
