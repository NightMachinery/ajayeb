package room_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Device links (package ident's claims) and the numbers that tell equal
// display names apart.

// joinClaim joins code with a claim and returns the first state frame.
func (c *tclient) joinClaim(code, claim string) wire.State {
	c.t.Helper()
	if err := c.h.m.JoinClaim(context.Background(), c.Client, code, &claim); err != nil {
		c.t.Fatalf("JoinClaim(%s): %v", code, err)
	}
	c.code = code
	return c.waitState(func(s wire.State) bool { return s.Code == code })
}

// joinClaimErr joins code with a claim expecting it to fail, and returns
// the error frame.
func (c *tclient) joinClaimErr(code, claim string) (wire.Error, error) {
	c.t.Helper()
	err := c.h.m.JoinClaim(context.Background(), c.Client, code, &claim)
	if err == nil {
		c.t.Fatalf("JoinClaim(%s): want an error, got none", code)
	}
	for {
		switch m := c.next().(type) {
		case wire.Error:
			return m, err
		case wire.State:
		default:
			c.t.Fatalf("JoinClaim(%s) failed with %v but sent %#v", code, err, m)
		}
	}
}

// link asks for a seat's link (or resets it) and returns the linked frame.
func (c *tclient) link(seat int, reset bool) wire.Linked {
	c.t.Helper()
	c.sendLink(seat, reset)
	for {
		switch m := c.next().(type) {
		case wire.Linked:
			return m
		case wire.State:
		default:
			c.t.Fatalf("client %s: want a linked frame, got %#v", c.ClientID, m)
		}
	}
}

func (c *tclient) sendLink(seat int, reset bool) {
	c.t.Helper()
	var err error
	if reset {
		err = c.h.m.Relink(context.Background(), c.Client, wire.NewRelink(c.code, seat))
	} else {
		err = c.h.m.Link(context.Background(), c.Client, wire.NewLink(c.code, seat))
	}
	if err != nil {
		c.t.Fatalf("link: %v", err)
	}
}

// onlinePair makes an online room: Ann (user-a, the admin) on seat 0 and
// Bob (user-b) on seat 1.
func onlinePair(h *harness, nameA, nameB string) (string, *tclient, *tclient) {
	a := h.client("user-a", "tab-a", nameA)
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", nameB)
	b.join(code)
	h.snapshot(code)
	a.current()
	return code, a, b
}

func TestLinkClaimIsRoomScoped(t *testing.T) {
	h := newHarness(t)
	code, a, _ := onlinePair(h, "Ann", "Bob")
	l := a.link(0, false)
	if l.Code != code || l.Seat != 0 || l.Claim != h.secret.Claim(code, "user-a", 0) {
		t.Fatalf("linked %+v", l)
	}
	// Asked again, the same link: nothing is stored, nothing changes.
	if again := a.link(0, false); again.Claim != l.Claim {
		t.Fatalf("a second request gave another claim")
	}
	if snap := h.snapshot(code); snap.Seats[0].Epoch != 0 || len(snap.RoomLog) != 0 {
		t.Fatalf("a link request changed the room: %+v %+v", snap.Seats, snap.RoomLog)
	}
}

// TestClaimJoinOnline checks that a connection with a valid claim acts in
// the room as the seat's user (and may move for it), keeps its own identity
// in every other room, and that both devices work at once.
func TestClaimJoinOnline(t *testing.T) {
	h := newHarness(t)
	code, a, b := onlinePair(h, "Ann", "Bob")
	claim := a.link(0, false).Claim

	c := h.client("user-c", "tab-c", "Phone")
	st := c.joinClaim(code, claim)
	if len(st.You.Seats) != 1 || st.You.Seats[0] != 0 || !st.You.Admin || !st.Room.Seats[0].You || st.Room.Seats[1].You {
		t.Fatalf("the claimed connection: you %+v, seats %+v", st.You, st.Room.Seats)
	}
	if st.Room.Seats[0].Name != "Ann" {
		t.Fatalf("a claim renamed the seat to %q", st.Room.Seats[0].Name)
	}
	snap := h.snapshot(code)
	if snap.Seats[0].UserID != "user-a" || snap.Seats[1].UserID != "user-b" || snap.Spectators != 0 {
		t.Fatalf("seats after the claim: %+v, spectators %d", snap.Seats, snap.Spectators)
	}

	// The claimed device moves for seat 0 when it is seat 0's turn, and
	// the original device sees it.
	mover := c
	if *st.View.ToMove == 1 {
		mover = b
		b.current()
	}
	mover.act(mover.last.View.Choices[0].Key, 0, nil)
	h.snapshot(code)
	if got := a.current(); got.Version != 1 {
		t.Fatalf("the original device: version %d", got.Version)
	}
	if got := c.current(); got.Version != 1 {
		t.Fatalf("the claimed device: version %d", got.Version)
	}

	// Elsewhere the connection is itself: in a fresh online room it takes a
	// seat as user-c.
	d := h.client("user-d", "tab-d", "Dee")
	other, _ := d.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	c.join(other)
	if snap := h.snapshot(other); snap.Seats[1].UserID != "user-c" {
		t.Fatalf("in another room the claimed connection is %q", snap.Seats[1].UserID)
	}
}

// TestClaimJoinHotseatAndAI checks claims of the rooms only their owner may
// enter: the connection gets both seats of a hot-seat room, the human seat
// of a bot room, and without the claim it is refused.
func TestClaimJoinHotseatAndAI(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	hot, _ := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", "Omid"}))
	claim := a.link(0, false).Claim

	c := h.client("user-c", "tab-c", "Phone")
	if e, err := c.joinErr(hot); !errors.Is(err, room.ErrForbidden) || e.Code != wire.ErrorCodeForbidden {
		t.Fatalf("a hot-seat room without the claim: %v %+v", err, e)
	}
	st := c.joinClaim(hot, claim)
	if len(st.You.Seats) != 2 || !st.You.Admin {
		t.Fatalf("the hot-seat claim: you %+v", st.You)
	}
	seat := int(*st.View.ToMove)
	c.act(st.View.Choices[0].Key, 0, &seat)
	h.snapshot(hot)
	if got := c.current(); got.Version != 1 {
		t.Fatalf("the claimed hot-seat device could not move: version %d", got.Version)
	}

	// A bot room, the human on seat 1. The claim is evicted first, so the
	// header check (the room not in memory) is exercised too.
	bot := wire.BotLevelEasy
	a.drain()
	ai, _ := a.create(wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(1), nil))
	aiClaim := a.link(1, false).Claim
	a.join(hot) // leaves the bot room
	h.snapshot(ai)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("the bot room's eviction", func() bool { return !h.m.Loaded(ai) })

	e := h.client("user-e", "tab-e", "Tab")
	if _, err := e.joinErr(ai); !errors.Is(err, room.ErrForbidden) {
		t.Fatalf("a bot room without the claim: %v", err)
	}
	if h.m.Loaded(ai) {
		t.Fatalf("a refused join loaded the room")
	}
	st = e.joinClaim(ai, aiClaim)
	if len(st.You.Seats) != 1 || st.You.Seats[0] != 1 {
		t.Fatalf("the bot room claim: you %+v", st.You)
	}
}

// TestBadClaimRefused checks that a claim of the wrong shape, of another
// room or of nobody is refused with bad-claim, in memory and from the
// header, and that the connection keeps nothing.
func TestBadClaimRefused(t *testing.T) {
	h := newHarness(t)
	code, a, b := onlinePair(h, "Ann", "Bob")
	d := h.client("user-d", "tab-d", "Dee")
	d.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	otherClaim := d.link(0, false).Claim
	a.current()

	bad := map[string]string{
		"garbage":      "not a claim",
		"right shape":  "AAAAAAAAAAAAAAAAAAAAAA",
		"other room":   otherClaim,
		"other secret": "x" + h.secret.Claim(code, "user-a", 0)[1:],
		"next epoch":   h.secret.Claim(code, "user-a", 1),
	}
	for name, claim := range bad {
		c := h.client("user-c", "tab-"+name, "C")
		e, err := c.joinClaimErr(code, claim)
		if !errors.Is(err, room.ErrBadClaim) || e.Code != wire.ErrorCodeBadClaim {
			t.Fatalf("%s: %v %+v", name, err, e)
		}
		if c.Joined() != "" {
			t.Fatalf("%s: the client counts as joined", name)
		}
	}
	if snap := h.snapshot(code); snap.Clients != 2 {
		t.Fatalf("bad claims attached connections: %d", snap.Clients)
	}

	// From the header, with the room evicted: refused without loading it.
	h.m.Disconnect(a.Client)
	h.m.Disconnect(b.Client)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })
	c := h.client("user-c", "tab-evicted", "C")
	if e, err := c.joinClaimErr(code, otherClaim); !errors.Is(err, room.ErrBadClaim) || e.Code != wire.ErrorCodeBadClaim {
		t.Fatalf("from the header: %v %+v", err, e)
	}
	if h.m.Loaded(code) {
		t.Fatalf("a bad claim loaded the room")
	}
}

// TestLinkPermissions checks who may have a seat's link: the seat's own
// player and the game's admin; not the other player, nor a spectator; and
// no seat without a player.
func TestLinkPermissions(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	// The open seat has no link.
	a.sendLink(1, false)
	a.expectReject(0, wire.RejectCodeBadRequest)

	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	h.snapshot(code)

	if l := b.link(1, false); l.Claim != h.secret.Claim(code, "user-b", 0) {
		t.Fatalf("the player's own link: %+v", l)
	}
	if l := a.link(1, false); l.Claim != h.secret.Claim(code, "user-b", 0) {
		t.Fatalf("the admin's copy of the other link: %+v", l)
	}
	b.sendLink(0, false)
	b.expectReject(0, wire.RejectCodeNotAdmin)
	b.sendLink(0, true)
	b.expectReject(0, wire.RejectCodeNotAdmin)

	sp := h.client("user-s", "tab-s", "Spy")
	sp.join(code)
	for _, seat := range []int{0, 1} {
		sp.sendLink(seat, false)
		sp.expectReject(0, wire.RejectCodeNotAdmin)
	}
	for _, m := range sp.drain() {
		if _, ok := m.(wire.Linked); ok {
			t.Fatalf("the spectator received a link")
		}
	}

	// Not joined.
	x := h.client("user-x", "tab-x", "X")
	x.code = code
	if err := h.m.Link(context.Background(), x.Client, wire.NewLink(code, 0)); !errors.Is(err, room.ErrNotJoined) {
		t.Fatalf("a link from a client that did not join: %v", err)
	}
	x.expectReject(0, wire.RejectCodeBadRequest)

	// A bot seat has no link.
	bot := wire.BotLevelEasy
	a.drain()
	a.create(wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil))
	a.sendLink(1, false)
	a.expectReject(0, wire.RejectCodeBadRequest)

	// Linked frames never go to anyone but the asker.
	if snap := h.snapshot(code); snap.Seats[0].Epoch != 0 || snap.Seats[1].Epoch != 0 {
		t.Fatalf("refused requests changed the epochs: %+v", snap.Seats)
	}
}

// TestLinksOff checks a Manager without Claims: links are refused and so
// is every claim.
func TestLinksOff(t *testing.T) {
	h := newHarness(t, func(cfg *room.Config) { cfg.Claims = nil })
	code, a, _ := onlinePair(h, "Ann", "Bob")
	a.sendLink(0, false)
	a.expectReject(0, wire.RejectCodeBadRequest)
	c := h.client("user-c", "tab-c", "C")
	if _, err := c.joinClaimErr(code, h.secret.Claim(code, "user-a", 0)); !errors.Is(err, room.ErrBadClaim) {
		t.Fatalf("a claim with links off: %v", err)
	}
}

// TestRelinkRevokes checks a link reset: the old claim stops working (a
// device using it is let go with bad-claim), the new one works, the reset
// is logged for both players, the version does not move, and it survives
// a reload.
func TestRelinkRevokes(t *testing.T) {
	h := newHarness(t)
	code, a, b := onlinePair(h, "Ann", "Bob")
	old := b.link(1, false).Claim
	phone := h.client("user-p", "tab-p", "Phone")
	phone.joinClaim(code, old)
	h.snapshot(code)
	a.current()
	b.current()

	l := b.link(1, true)
	if l.Claim == old || l.Claim != h.secret.Claim(code, "user-b", 1) {
		t.Fatalf("the reset link: %+v", l)
	}
	h.snapshot(code)
	// The phone was using the old link: it is told and let go.
	var told bool
	for _, m := range phone.drain() {
		if e, ok := m.(wire.Error); ok && e.Code == wire.ErrorCodeBadClaim {
			told = true
		}
	}
	if !told {
		t.Fatalf("the device on the old link was not told")
	}
	if err := h.m.Act(context.Background(), phone.Client, wire.NewAct(code, 9, 0, "x", nil)); err != nil {
		t.Fatalf("Act: %v", err)
	}
	phone.expectReject(9, wire.RejectCodeBadRequest)

	for _, c := range []*tclient{a, b} {
		st := c.current()
		if st.Version != 0 || len(st.RoomLog) != 1 {
			t.Fatalf("%s: version %d, log %+v", c.ClientID, st.Version, st.RoomLog)
		}
		e := st.RoomLog[0]
		if e.Kind != wire.RoomLogLink || e.Seat == nil || *e.Seat != 1 || e.Target == nil || *e.Target != 1 || len(e.On) != 0 || len(e.Off) != 0 {
			t.Fatalf("%s: the log entry %+v", c.ClientID, e)
		}
	}

	// The old link fails, the new one works.
	p2 := h.client("user-q", "tab-q", "Tablet")
	if _, err := p2.joinClaimErr(code, old); !errors.Is(err, room.ErrBadClaim) {
		t.Fatalf("the old link after the reset: %v", err)
	}
	if st := p2.joinClaim(code, l.Claim); len(st.You.Seats) != 1 || st.You.Seats[0] != 1 {
		t.Fatalf("the new link: %+v", st.You)
	}

	// The admin resets the other seat's link too; the epoch is stored.
	a.link(1, true)
	h.snapshot(code)
	snap := h.snapshot(code)
	if snap.Seats[1].Epoch != 2 || snap.Seats[0].Epoch != 0 || len(snap.RoomLog) != 2 || snap.RoomLog[1].Seat != 0 {
		t.Fatalf("after the admin's reset: %+v %+v", snap.Seats, snap.RoomLog)
	}
	saved, err := h.st.PeekHeader(code)
	if err != nil || saved.Seats[1].Epoch != 2 {
		t.Fatalf("the saved epoch: %+v %v", saved, err)
	}
}

// TestRelinkHotseatRevokesBothSeats checks that resetting the link of a
// hot-seat room, whose two seats belong to one user, revokes the old claim
// for both seats: an old claim must not live on through the other seat.
func TestRelinkHotseatRevokesBothSeats(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", "Omid"}))
	old := a.link(0, false).Claim
	if a.link(1, false).Claim != old {
		t.Fatalf("the two seats of a hot-seat room have different links")
	}
	fresh := a.link(0, true).Claim
	h.snapshot(code)
	if fresh == old || a.link(1, false).Claim != fresh {
		t.Fatalf("after the reset: old %s, fresh %s", old, fresh)
	}
	c := h.client("user-c", "tab-c", "Phone")
	if _, err := c.joinClaimErr(code, old); !errors.Is(err, room.ErrBadClaim) {
		t.Fatalf("the old hot-seat link after the reset: %v", err)
	}
	if st := c.joinClaim(code, fresh); len(st.You.Seats) != 2 {
		t.Fatalf("the new hot-seat link: %+v", st.You)
	}
}

// TestRelinkBySelfOnClaimKeepsDevice checks that a device that joined with
// a claim and resets that same link keeps its place.
func TestRelinkBySelfOnClaimKeepsDevice(t *testing.T) {
	h := newHarness(t)
	code, a, _ := onlinePair(h, "Ann", "Bob")
	claim := a.link(0, false).Claim
	c := h.client("user-c", "tab-c", "Phone")
	c.joinClaim(code, claim)
	l := c.link(0, true)
	h.snapshot(code)
	if st := c.current(); len(st.RoomLog) != 1 {
		t.Fatalf("the resetting device: %+v", st.RoomLog)
	}
	if l.Claim == claim {
		t.Fatalf("the reset kept the claim")
	}
}

// TestRematchCarriesClaim checks that a connection on a claim learns its
// seat's claim in the rematch room, and that it works there.
func TestRematchCarriesClaim(t *testing.T) {
	h := newHarness(t)
	code, a, b := onlinePair(h, "Ann", "Bob")
	claim := b.link(1, false).Claim
	phone := h.client("user-p", "tab-p", "Phone")
	phone.joinClaim(code, claim)
	phone.act("resign:1", 0, nil)
	h.snapshot(code)
	next, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil {
		t.Fatalf("Rematch: %v", err)
	}
	h.snapshot(code)
	var got *wire.Created
	for _, m := range phone.drain() {
		if c, ok := m.(wire.Created); ok {
			got = &c
		}
	}
	if got == nil || got.Code != next || got.Claim == nil || *got.Claim != h.secret.Claim(next, "user-b", 0) {
		t.Fatalf("the phone's created frame: %+v", got)
	}
	for _, m := range b.drain() {
		if c, ok := m.(wire.Created); ok && c.Claim != nil {
			t.Fatalf("the player's own device got a claim: %+v", c)
		}
	}
	if st := phone.joinClaim(next, *got.Claim); len(st.You.Seats) != 1 || st.You.Seats[0] != 1 {
		t.Fatalf("the rematch claim: %+v", st.You)
	}
}

// nameNos returns the seats' name numbers of a state frame.
func nameNos(st wire.State) [2]int {
	var out [2]int
	for _, s := range st.Room.Seats {
		out[s.Seat] = s.NameNo
	}
	return out
}

// TestDuplicateNamesNumberedStably checks the name numbers of an online
// room: the second "Sara" (without case and outer spaces) is "Sara 2",
// the numbers survive leaving and rejoining, and a rename recomputes only
// the renamer's.
func TestDuplicateNamesNumberedStably(t *testing.T) {
	h := newHarness(t)
	code, a, b := onlinePair(h, "Sara", " sara ")
	if got := nameNos(a.current()); got != [2]int{0, 2} {
		t.Fatalf("numbers %v, want [0 2]", got)
	}

	// Both leave and come back, the second first.
	h.m.Disconnect(b.Client)
	h.m.Disconnect(a.Client)
	b2 := h.client("user-b", "tab-b2", " sara ")
	if got := nameNos(b2.join(code)); got != [2]int{0, 2} {
		t.Fatalf("after leaving: %v", got)
	}
	a2 := h.client("user-a", "tab-a2", "Sara")
	if got := nameNos(a2.join(code)); got != [2]int{0, 2} {
		t.Fatalf("after rejoining: %v", got)
	}

	// The first holder renames: only its own number is recomputed, so the
	// second keeps "Sara 2".
	h.m.Disconnect(a2.Client)
	a3 := h.client("user-a", "tab-a3", "Ann")
	st := a3.join(code)
	if got := nameNos(st); got != [2]int{0, 2} || st.Room.Seats[0].Name != "Ann" {
		t.Fatalf("after the first renamed: %v %+v", got, st.Room.Seats)
	}
	// And back: Sara is now the later one, after "Sara 2".
	h.m.Disconnect(a3.Client)
	a4 := h.client("user-a", "tab-a4", "SARA")
	if got := nameNos(a4.join(code)); got != [2]int{3, 2} {
		t.Fatalf("after renaming back: %v", got)
	}
	saved, err := h.st.PeekHeader(code)
	if err != nil || saved.Seats[0].NameNo != 3 || saved.Seats[1].NameNo != 2 {
		t.Fatalf("saved numbers: %+v %v", saved, err)
	}

	// A claim joins under the seat's name, whatever its hello says.
	claim := a4.link(1, false).Claim
	c := h.client("user-c", "tab-c", "Sara")
	if st := c.joinClaim(code, claim); st.Room.Seats[1].Name != " sara " || nameNos(st) != [2]int{3, 2} {
		t.Fatalf("a claim join changed the names: %+v", st.Room.Seats)
	}
}

// TestHotseatEqualNamesNumbered checks that a hot-seat room created with
// two equal names numbers the second seat, and that a bot room has no
// numbers.
func TestHotseatEqualNamesNumbered(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	_, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", "ann"}))
	if got := nameNos(st); got != [2]int{0, 2} {
		t.Fatalf("hot-seat numbers %v", got)
	}
	_, st = a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", ""}))
	if got := nameNos(st); got != [2]int{0, 0} {
		t.Fatalf("hot-seat with a blank second name: %v", got)
	}
	bot := wire.BotLevelEasy
	_, st = a.create(wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil))
	if got := nameNos(st); got != [2]int{0, 0} {
		t.Fatalf("bot room numbers %v", got)
	}
}

// TestOldRoomFileIdentity checks a room file written before name numbers
// and link epochs existed: equal names load numbered (seat 0 first), and
// the epoch-0 claims work.
func TestOldRoomFileIdentity(t *testing.T) {
	h := newHarness(t)
	const code = "XJDN22"
	f := store.New(code, store.ModeOnline, engine.Options{}, 77, roomEpoch)
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Sara"},
		{Kind: store.SeatKindHuman, UserID: "user-b", Name: "Sara"},
	}
	f.Status = store.StatusPlaying
	if err := h.st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c := h.client("user-c", "tab-c", "Phone")
	st := c.joinClaim(code, h.secret.Claim(code, "user-b", 0))
	if got := nameNos(st); got != [2]int{0, 2} || len(st.You.Seats) != 1 || st.You.Seats[0] != 1 {
		t.Fatalf("the old room: numbers %v, you %+v", got, st.You)
	}
}
