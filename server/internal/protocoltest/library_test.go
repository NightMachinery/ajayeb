package protocoltest

import (
	"bytes"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// TestGreatLibraryOfferStaysPrivate starts an online room, with a
// spectator, right as seat 0 builds the Great Library (the named scenario
// great-library), so the three tokens drawn from the box are on offer to
// seat 0 alone. Random full games reach this choice only sometimes; this
// test reaches it every time. Seat 0 must see the offer; seat 1 and the
// spectator must not, before or after seat 0 keeps one, since the two it
// puts back return to the box.
func TestGreatLibraryOfferStaysPrivate(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_950_000_000_001)
	open := store.Seat{Kind: store.SeatKindOpen}
	code, err := srv.rooms.CreateFromSpec(room.Spec{
		Mode:     store.ModeOnline,
		Seats:    [2]store.Seat{open, open},
		Seed:     9_950_000_000_000,
		Scenario: "great-library",
		Actions:  []string{"wonder:great-library:rostrum"},
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}

	a := dial(t, srv, "A")
	a.hello()
	a.join(code)
	b := dial(t, srv, "B")
	b.hello()
	b.join(code)
	spec := dial(t, srv, "Spectator")
	spec.hello()
	spec.join(code)
	conns := []*tclient{a, b, spec}
	syncTo(conns, maxVersion(conns))

	if len(a.latest.You.Seats) != 1 || a.latest.You.Seats[0] != 0 {
		t.Fatalf("A holds seats %v, want [0] (the first joiner claims the first open seat)", a.latest.You.Seats)
	}
	snap := srv.snapshot(code)
	if snap.State.Phase != engine.PhasePickLibraryToken || snap.State.ToMove != 0 {
		t.Fatalf("the room is in phase %s with seat %d to move, want seat 0 choosing from the Great Library offer", snap.State.Phase, snap.State.ToMove)
	}
	offer := trueHidden(&snap.State, seatRef(1))
	for _, slug := range []string{"law", "philosophy", "strategy"} {
		if !offer[slug] {
			t.Fatalf("the hidden set for seat 1 lacks the offered token %s: %v", slug, offer)
		}
	}
	// The decider sees the offer: a positive control that the scan below
	// would find the slugs if they were sent.
	sawLaw := false
	chunks, _ := a.raw.since(0)
	for _, chunk := range chunks {
		sawLaw = sawLaw || bytes.Contains(chunk, []byte(`"law"`))
	}
	if !sawLaw {
		t.Fatalf("seat 0, the decider, never received the offered token law")
	}

	lt := newLeakTracker(t, snap.State.Seed)
	checkAll := func() {
		s := srv.snapshot(code)
		lt.check(a, "seat0", trueHidden(&s.State, seatRef(0)))
		lt.check(b, "seat1", trueHidden(&s.State, seatRef(1)))
		lt.check(spec, "spectator", trueHidden(&s.State, nil))
	}
	checkAll()

	seq := a.act("token:law", nil)
	a.untilAck(seq)
	syncTo(conns, maxVersion(conns))
	checkAll()
}
