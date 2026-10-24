package lobbyroom

import (
	"errors"
	"fmt"
	"slices"
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

// bound reads the lobby's connections and checks its bounds: each user at
// most MaxUserClients, watching connections (those of users with no
// slot, taken or reserved) at most MaxClients plus seven less the
// players, so at most 88 in all.
func (f *fixture) bound(code, what string) {
	f.t.Helper()
	a := f.lobbies.lookup(code)
	if a == nil {
		f.t.Fatalf("%s: the lobby is not running", what)
	}
	var total, watching, players int
	per := map[string]int{}
	if err := a.call(f.ctx(), func() {
		for _, s := range a.rec.Slots {
			if s.UserID != "" {
				players++
			}
		}
		for _, c := range a.conns {
			u := a.userOf(c)
			per[u]++
			if !a.holdsSlot(u) {
				watching++
			}
		}
		total = len(a.conns)
	}); err != nil {
		f.t.Fatal(err)
	}
	checkBound(f.t, what, per, total, watching, players)
}

func checkBound(t *testing.T, what string, per map[string]int, total, watching, players int) {
	t.Helper()
	for u, n := range per {
		if n > tableroom.DefaultMaxUserClients {
			t.Fatalf("%s: %s holds %d connections", what, u, n)
		}
	}
	if watching > defaultMaxClients+MaxSeats-players || total > defaultMaxClients+MaxSeats+MaxSeats*(tableroom.DefaultMaxUserClients-1) {
		t.Fatalf("%s: %d connections, %d watching, %d players", what, total, watching, players)
	}
}

// closedFull reports whether k received error room-full.
func (k *conn) closedFull() bool {
	k.drain()
	return slices.ContainsFunc(k.frames, func(m any) bool {
		e, ok := m.(wire.Error)
		return ok && e.Code == wire.ErrorCodeRoomFull
	})
}

// TestLobbyWatchersStayBounded walks a seven-slot lobby through slot
// edits, each user who takes a slot opening eight connections: removals,
// a release and a smaller seat count, then Start. It checks the bounds
// after every step, in the lobby and then at the table: a user who loses
// a slot while the watchers are full keeps only their oldest connection,
// and the others get error room-full. Before, they all stayed.
func TestLobbyWatchersStayBounded(t *testing.T) {
	f := newFixture(t, nil)
	f.burst = 1000
	f.restart()
	var all []*conn
	join := func(u, id, code string) *conn {
		k := f.conn(u, id, u)
		f.join(k, code)
		all = append(all, k)
		return k
	}
	tabs := func(u, code string) []*conn {
		var out []*conn
		for j := 1; j < tableroom.DefaultMaxUserClients; j++ {
			out = append(out, join(u, fmt.Sprintf("%s-tab-%d", u, j), code))
		}
		return out
	}
	admin := f.conn(user(0), "c0", "P0")
	all = append(all, admin)
	code := f.create(admin, 7, wire.LobbyOptions{Seating: ptr("lobby")})
	ks := []*conn{admin}
	for i := 1; i <= 6; i++ {
		ks = append(ks, join(user(i), "c"+strconv.Itoa(i), code))
	}
	for i := range 12 {
		join(fmt.Sprintf("puppet-%d", i), fmt.Sprintf("puppet-%d", i), code)
	}
	f.bound(code, "twelve watchers")
	seq := 0
	opened := map[string][]*conn{}
	remove := func(u string) {
		t.Helper()
		seq++
		e := edit(code, seq, "remove-human")
		e.UserID = ptr(u)
		f.send(admin, e)
		admin.acked(seq)
		f.bound(code, "removing "+u)
	}
	opened[user(6)] = tabs(user(6), code)
	f.bound(code, "user-6's tabs")
	remove(user(6))
	for i := range 12 {
		u := fmt.Sprintf("puppet-%d", i)
		seq++
		se := edit(code, seq, "seat")
		se.Seat, se.UserID = ptr(6), ptr(u)
		f.send(admin, se)
		admin.acked(seq)
		f.bound(code, u+" seated")
		opened[u] = tabs(u, code)
		f.bound(code, u+"'s tabs")
		remove(u)
	}
	opened[user(5)] = tabs(user(5), code)
	f.send(ks[5], edit(code, 1, "release"))
	ks[5].acked(1)
	f.bound(code, "user-5's release")
	opened[user(4)] = tabs(user(4), code)
	seq++
	sc := edit(code, seq, "seats")
	sc.SeatCount = ptr(4)
	f.send(admin, sc)
	admin.acked(seq)
	f.bound(code, "four seats")
	closed := 0
	for _, tabs := range opened {
		for _, k := range tabs {
			if k.closedFull() {
				closed++
			}
		}
	}
	if closed == 0 {
		t.Fatal("no connection was closed: the walk never filled the watchers")
	}

	// Start: every connection still the lobby's gets the table, and the
	// table keeps the same bounds.
	seq++
	f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: seq, FillBots: true})
	st := admin.table("the table", func(wire.TableState) bool { return true })
	seated := map[string]bool{}
	players := 0
	for _, s := range st.Seats {
		if s.UserID != nil {
			seated[*s.UserID] = true
			players++
		}
	}
	per := map[string]int{}
	total, watching := 0, 0
	for _, k := range all {
		if k.closedFull() {
			continue
		}
		if k != admin {
			k.table("the table for "+k.c.ClientID, func(wire.TableState) bool { return true })
		}
		total++
		per[k.c.UserID]++
		if !seated[k.c.UserID] {
			watching++
		}
	}
	checkBound(t, "the table after Start", per, total, watching, players)
}

// TestRematchLobbyStaysBounded: in a rematch lobby, watchers fill the
// connections while a player is away; the player comes back with eight
// connections, which their reserved slot admits, and is then removed:
// they keep their oldest connection, and the bounds hold.
func TestRematchLobbyStaysBounded(t *testing.T) {
	f := newFixture(t, nil)
	code, k0, _ := f.finished()
	if _, err := f.m.Rematch(f.ctx(), k0.c, code); err != nil {
		t.Fatal(err)
	}
	next := k0.until("the created frame", func(v any) bool { _, ok := v.(wire.Created); return ok }).(wire.Created).Code
	admin := f.conn(user(0), "r0", "P0")
	f.join(admin, next)
	admin.lobby("the rematch lobby", any2)
	f.bound(next, "the requester back")
	for i := 0; ; i++ {
		k := f.conn(fmt.Sprintf("w-%d", i), fmt.Sprintf("w-%d", i), "W")
		if err := f.m.Join(f.ctx(), k.c, next); err != nil {
			if !errors.Is(err, room.ErrRoomFull) {
				t.Fatal(err)
			}
			break
		}
	}
	f.bound(next, "full of watchers")
	back := f.conn(user(1), "r1", "P1")
	f.join(back, next)
	tabs := []*conn{}
	for j := 1; j < tableroom.DefaultMaxUserClients; j++ {
		k := f.conn(user(1), fmt.Sprintf("r1-tab-%d", j), "P1")
		f.join(k, next)
		tabs = append(tabs, k)
	}
	f.bound(next, "user-1 back with eight connections")
	e := edit(next, 1, "remove-human")
	e.UserID = ptr(user(1))
	f.send(admin, e)
	admin.acked(1)
	f.bound(next, "user-1 removed")
	if back.closedFull() {
		t.Fatal("user-1 lost their oldest connection")
	}
	for _, k := range tabs {
		if !k.closedFull() {
			t.Fatalf("%s stayed past the watchers' allowance", k.c.ClientID)
		}
	}
}
