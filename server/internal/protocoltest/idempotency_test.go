package protocoltest

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestStaleBaseAndDuplicateSeq checks the two idempotency rules of
// docs/protocol.md: an act whose base does not match the room's current
// version is rejected as stale and a fresh state follows at once, and
// resending an already-applied seq re-sends the current state without
// applying anything a second time.
func TestStaleBaseAndDuplicateSeq(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_400_000_000_001)
	c := dial(t, srv, "Hotseat")
	c.hello()
	c.create(wire.ModeHotseat, nil, wire.SeatNumber(0))

	base := c.latest.Version
	toMove := *c.latest.View.ToMove
	key := c.latest.View.Choices[0].Key

	// A base that is not the current version, in either direction.
	seq1 := c.nextSeq()
	c.actAt(seq1, base+1, key, seatInt(toMove))
	rej := c.untilReject(seq1)
	if rej.Code != wire.RejectCodeStale {
		t.Fatalf("stale base: reject code = %q, want %q", rej.Code, wire.RejectCodeStale)
	}
	// "stale -> fresh state follows": the very next frame is a state at the
	// unchanged version, not just eventually one.
	f := c.next()
	st, ok := f.(wire.State)
	if !ok {
		t.Fatalf("stale base: want a state frame right after the reject, got %#v", f)
	}
	if st.Version != base {
		t.Fatalf("stale base: the fresh state's version = %d, want unchanged %d", st.Version, base)
	}

	// A correctly based action actually applies.
	seq2 := c.act(key, seatInt(toMove))
	after := c.untilAck(seq2)
	if after.Version != base+1 {
		t.Fatalf("after the action: version = %d, want %d", after.Version, base+1)
	}

	// The exact same seq again (the client resending because it never saw
	// the first reply, or a browser double-click): not applied twice.
	c.actAt(seq2, base, key, seatInt(toMove))
	dup := c.untilAck(seq2)
	if dup.Version != after.Version {
		t.Fatalf("duplicate seq changed the version: got %d, want unchanged %d", dup.Version, after.Version)
	}

	// A key that is not in the current legal set.
	seq3 := c.nextSeq()
	nextToMove := *after.View.ToMove
	c.actAt(seq3, after.Version, "construct:this-card-does-not-exist", seatInt(nextToMove))
	rej3 := c.untilReject(seq3)
	if rej3.Code != wire.RejectCodeIllegal {
		t.Fatalf("illegal key: reject code = %q, want %q", rej3.Code, wire.RejectCodeIllegal)
	}
}

// TestNotYourTurn checks that an online room refuses an act from the
// connection that does not hold the seat to move.
func TestNotYourTurn(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_500_000_000_001)
	a := dial(t, srv, "A")
	a.hello()
	a.create(wire.ModeOnline, nil, wire.SeatRandom)
	code := a.code

	b := dial(t, srv, "B")
	b.hello()
	b.join(code)

	conns := []*tclient{a, b}
	syncTo(conns, maxVersion(conns))
	bySeat := map[int]*tclient{a.latest.You.Seats[0]: a, b.latest.You.Seats[0]: b}

	toMove := *a.latest.View.ToMove
	actor := bySeat[toMove]
	other := bySeat[1-toMove]

	key := actor.latest.View.Choices[0].Key
	seq := other.nextSeq()
	other.actAt(seq, other.latest.Version, key, nil)
	rej := other.untilReject(seq)
	if rej.Code != wire.RejectCodeNotYourTurn {
		t.Fatalf("not your turn: reject code = %q, want %q", rej.Code, wire.RejectCodeNotYourTurn)
	}
}
