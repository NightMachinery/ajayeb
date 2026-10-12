package protocoltest

import (
	"bytes"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestDeviceLinkOverTheSocket drives a device link (docs/protocol.md, link,
// linked and join's claim) through the real transport: the player asks for
// its seat's link, a third socket joins with the claim and plays the seat,
// the claim never reaches the other player, and a reset revokes it.
func TestDeviceLinkOverTheSocket(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_960_000_000_001)
	a := dial(t, srv, "Sara")
	a.hello()
	a.create(wire.ModeOnline, nil, wire.SeatNumber(0))
	b := dial(t, srv, "Sara")
	b.hello()
	bst := b.join(a.code)
	if bst.Room.Seats[1].NameNo != 2 || bst.Room.Seats[0].NameNo != 0 {
		t.Fatalf("two Saras: %+v", bst.Room.Seats)
	}

	a.send(wire.NewLink(a.code, 0))
	l := a.until("the linked frame", func(f any) bool { _, ok := f.(wire.Linked); return ok }).(wire.Linked)
	if l.Seat != 0 || len(l.Claim) != 22 {
		t.Fatalf("linked %+v", l)
	}

	phone := dial(t, srv, "Phone")
	phone.hello()
	join := wire.NewJoin(a.code)
	join.Claim = &l.Claim
	phone.send(join)
	phone.code = a.code
	pst := phone.until("the claimed room's state", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Code == a.code
	}).(wire.State)
	if len(pst.You.Seats) != 1 || pst.You.Seats[0] != 0 || !pst.You.Admin {
		t.Fatalf("the phone: you %+v", pst.You)
	}

	// The other player asks for seat 0's link: refused, and the claim
	// never reaches its socket.
	b.send(wire.NewLink(b.code, 0))
	if r := b.until("the refusal", func(f any) bool { _, ok := f.(wire.Reject); return ok }).(wire.Reject); r.Code != wire.RejectCodeNotAdmin {
		t.Fatalf("the opponent's link request: %+v", r)
	}
	chunks, _ := b.raw.since(0)
	for _, c := range chunks {
		if bytes.Contains(c, []byte(l.Claim)) {
			t.Fatalf("the opponent received the claim: %s", c)
		}
	}

	// A reset: the phone is let go, and the old link fails.
	a.send(wire.NewRelink(a.code, 0))
	nl := a.until("the new link", func(f any) bool { _, ok := f.(wire.Linked); return ok }).(wire.Linked)
	if nl.Claim == l.Claim {
		t.Fatalf("the reset kept the claim")
	}
	if e := phone.until("the revocation", func(f any) bool { _, ok := f.(wire.Error); return ok }).(wire.Error); e.Code != wire.ErrorCodeBadClaim {
		t.Fatalf("the phone after the reset: %+v", e)
	}
	tablet := dial(t, srv, "Tablet")
	tablet.hello()
	tablet.send(join)
	if e := tablet.until("the refusal", func(f any) bool { _, ok := f.(wire.Error); return ok }).(wire.Error); e.Code != wire.ErrorCodeBadClaim {
		t.Fatalf("the old link after the reset: %+v", e)
	}
}

// TestBadClaimsAreRateLimited checks that joins with bad claims use up the
// address's join limit, as joins of unknown rooms do: past it, every join
// of the address is refused with reject rate.
func TestBadClaimsAreRateLimited(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_970_000_000_001)
	a := dial(t, srv, "Owner")
	a.hello()
	a.create(wire.ModeOnline, nil, wire.SeatNumber(0))

	g := dial(t, srv, "Guesser")
	g.hello()
	bad := "AAAAAAAAAAAAAAAAAAAAAA"
	join := wire.NewJoin(a.code)
	join.Claim = &bad
	errs := 0
	for range 40 {
		g.send(join)
		switch f := g.until("an answer", func(f any) bool {
			switch f.(type) {
			case wire.Error, wire.Reject:
				return true
			}
			return false
		}).(type) {
		case wire.Error:
			if f.Code != wire.ErrorCodeBadClaim {
				t.Fatalf("a bad claim: %+v", f)
			}
			errs++
		case wire.Reject:
			if f.Code != wire.RejectCodeRate {
				t.Fatalf("want reject rate, got %+v", f)
			}
			if errs == 0 {
				t.Fatalf("rate limited before any bad claim was answered")
			}
			// Even a plain join of a real room is refused now.
			g.send(wire.NewJoin(a.code))
			if r := g.until("the refusal", func(f any) bool { _, ok := f.(wire.Reject); return ok }).(wire.Reject); r.Code != wire.RejectCodeRate {
				t.Fatalf("a plain join past the limit: %+v", r)
			}
			return
		}
	}
	t.Fatalf("40 bad claims and no rate limit")
}
