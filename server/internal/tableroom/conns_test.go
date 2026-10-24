package tableroom

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestNinthConnectionIsRefused: a user holds at most 8 connections to a
// table, seated or watching; the ninth is refused with room-full, and
// another user still gets in. Before, a seated user could hold 32.
func TestNinthConnectionIsRefused(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 404))
	for _, who := range []struct {
		user  string
		first tabletest.ClientID
	}{{userOf(1), tabletest.SeatClient(1)}, {"watcher", tabletest.SpectatorClient}} {
		for i := 1; i < DefaultMaxUserClients; i++ {
			h.joinAs(tabletest.ClientID(fmt.Sprintf("%s-tab-%d", who.user, i)), who.user)
		}
		err := h.p.join(h.ctx, tabletest.ClientID(who.user+"-tab-9"), who.user, who.user)
		if !errors.Is(err, room.ErrRoomFull) {
			t.Fatalf("%s's ninth connection: %v", who.user, err)
		}
	}
	h.joinAs("other", "another-watcher")
}

// bound reads the table's connections and checks its bounds: each user
// at most MaxUserClients, watching connections (those of users with no
// seat) at most MaxClients plus seven less the players, so at most 88 in
// all.
func (h *th) bound(what string) {
	h.t.Helper()
	var total, watching, players int
	per := map[string]int{}
	h.on(func(a *actor) {
		for _, sr := range a.st.rec.Seats {
			if sr.Kind == SeatHuman {
				players++
			}
		}
		for _, c := range a.conns {
			u := a.userOf(c)
			per[u]++
			if a.seatOf(u) < 0 {
				watching++
			}
		}
		total = len(a.conns)
	})
	for u, n := range per {
		if n > DefaultMaxUserClients {
			h.t.Fatalf("%s: %s holds %d connections", what, u, n)
		}
	}
	if watching > DefaultMaxClients+MaxSeats-players || total > DefaultMaxClients+MaxSeats+MaxSeats*(DefaultMaxUserClients-1) {
		h.t.Fatalf("%s: %d connections, %d watching, %d players", what, total, watching, players)
	}
}

// attached reports whether the connection id is still the table's.
func (h *th) attached(id tabletest.ClientID) (in bool) {
	h.on(func(a *actor) { in = slices.Contains(a.conns, h.p.clients[id]) })
	return in
}

// roomFull reports whether id received error room-full.
func (h *th) roomFull(id tabletest.ClientID) bool {
	h.p.drain()
	return slices.ContainsFunc(h.p.other[id], func(m wire.ServerMessage) bool {
		e, ok := m.(wire.Error)
		return ok && e.Code == wire.ErrorCodeRoomFull
	})
}

// TestWatchersStayBounded walks a seven-seat table through reseatings,
// promotions, seat switches to bots and a resignation, each user who takes
// a seat opening eight connections, and checks the bounds after every
// step: a user who loses a seat while the watchers are full keeps only
// their oldest connection, and the others get error room-full. Before,
// they all stayed, and twelve reseatings left 104 connections (found by
// the slice's review).
func TestWatchersStayBounded(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots, RequestBurst: 1000}, humans(7, 451))
	admin := tabletest.SeatClient(0)
	tabs := func(u string) []tabletest.ClientID {
		ids := []tabletest.ClientID{}
		for j := 1; j < DefaultMaxUserClients; j++ {
			id := tabletest.ClientID(fmt.Sprintf("%s-tab-%d", u, j))
			h.joinAs(id, u)
			ids = append(ids, id)
		}
		return ids
	}
	h.bound("the start")
	for i := range 12 {
		h.joinAs(tabletest.ClientID(fmt.Sprintf("puppet-%d", i)), fmt.Sprintf("puppet-%d", i))
	}
	h.bound("twelve watchers")
	seq := 0
	opened := map[string][]tabletest.ClientID{}
	for i := range 12 {
		u := fmt.Sprintf("puppet-%d", i)
		seq++
		h.want(admin, toHuman(seq, 1, u), outAccepted, u+" into seat 1")
		h.bound(u + " seated")
		opened[u] = tabs(u)
		h.bound(u + "'s tabs")
	}
	for i := range 3 {
		seq++
		h.want(admin, roleOp(seq, "promote", fmt.Sprintf("puppet-%d", i)), outAccepted, "a promotion")
		h.bound("a promotion")
	}
	for s := 2; s <= 5; s++ {
		opened[userOf(s)] = tabs(userOf(s))
		h.bound(userOf(s) + "'s tabs")
	}
	for s := 2; s <= 4; s++ {
		seq++
		h.want(admin, toBot(seq, s, BotEasy), outAccepted, fmt.Sprintf("seat %d to a bot", s))
		h.bound(fmt.Sprintf("seat %d to a bot", s))
	}
	if out := h.do(tabletest.SeatClient(5), wire.TableResign{T: "table-resign", Code: h.p.code, Seq: 1, Confirm: true}); out != outAccepted {
		t.Fatalf("seat 5's resignation: %s", out)
	}
	h.bound("seat 5's resignation")
	// Everyone who lost a seat kept their oldest connection, and every
	// connection that went got error room-full.
	oldest := map[string]tabletest.ClientID{"puppet-0": "puppet-0", "puppet-10": "puppet-10",
		userOf(2): tabletest.SeatClient(2), userOf(4): tabletest.SeatClient(4), userOf(5): tabletest.SeatClient(5)}
	closed := 0
	for u, first := range oldest {
		if !h.attached(first) {
			t.Fatalf("%s lost their oldest connection", u)
		}
		for _, id := range opened[u] {
			if !h.attached(id) {
				closed++
				if !h.roomFull(id) {
					t.Fatalf("%s went without error room-full", id)
				}
			}
		}
	}
	if closed == 0 {
		t.Fatal("no connection was closed: the walk never filled the watchers")
	}
}
