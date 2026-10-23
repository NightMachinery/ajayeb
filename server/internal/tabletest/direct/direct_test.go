package direct_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
)

type harness struct {
	t  *testing.T
	tb *direct.Table
}

func newHarness(t *testing.T, seats uint8, seqClients int) *harness {
	t.Helper()
	tb, err := direct.New(t.Context(), direct.Options{Setup: fakeengine.Setup{Seed: 77, Seats: seats}, SeqClients: seqClients})
	if err != nil {
		t.Fatal(err)
	}
	return &harness{t, tb}
}

func (h *harness) snap(c tabletest.ClientID) tabletest.Snapshot {
	h.t.Helper()
	sn, err := h.tb.Snapshot(h.t.Context(), c)
	if err != nil {
		h.t.Fatal(err)
	}
	return sn
}

func (h *harness) send(c tabletest.ClientID, seq uint64, ref, key string, confirm bool) tabletest.Outcome {
	h.t.Helper()
	o, err := h.tb.Submit(h.t.Context(), c, tabletest.Submission{Seq: seq, Ref: ref, Key: key, Confirm: confirm})
	if err != nil {
		h.t.Fatal(err)
	}
	return o
}

func (h *harness) expect(got, want tabletest.Outcome, what string) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("%s: %s, want %s", what, got, want)
	}
}

func TestAcceptanceRules(t *testing.T) {
	h := newHarness(t, 3, 32)
	a, b, c := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	sa := h.snap(a)
	ref := sa.Ref
	if len(sa.Legal) < 2 {
		t.Fatal("seat 0 needs two offered keys")
	}
	k0, k1 := sa.Legal[0], sa.Legal[1]

	h.expect(h.send(tabletest.SpectatorClient, 1, ref, k0, true), tabletest.Forbidden, "a spectator")
	h.expect(h.send(a, 1, "age-1-turn-1@99", k0, false), tabletest.Closed, "another reference")
	h.expect(h.send(a, 1, ref, "salvage:amber", false), tabletest.Illegal, "an unoffered key")
	h.expect(h.send(a, 1, ref, k0, true), tabletest.Accepted, "a pick (refusals used up no seq)")
	h.expect(h.send(a, 1, ref, k1, false), tabletest.Duplicate, "the same seq again")
	if sn := h.snap(a); sn.Pick != k0 || !sn.Confirm {
		t.Fatalf("a duplicate changed the pick: %+v", sn)
	}
	h.expect(h.send(a, 2, ref, k1, false), tabletest.Accepted, "a change, withdrawing the confirmation")
	if sn := h.snap(b); len(sn.Confirmed) != 0 || sn.Pick != "" {
		t.Fatalf("seat 1 sees %+v: another seat's pick, or a withdrawn confirmation", sn)
	}
	h.expect(h.send(a, 3, ref, k1, true), tabletest.Accepted, "confirming again")
	if sn := h.snap(b); !slices.Equal(sn.Confirmed, []int{0}) || sn.Pick != "" {
		t.Fatalf("seat 1 sees %+v; want seat 0's confirmation and no pick", sn)
	}
	h.expect(h.send(b, 1, ref, h.snap(b).Legal[0], true), tabletest.Accepted, "seat 1")
	h.expect(h.send(c, 1, ref, h.snap(c).Legal[0], true), tabletest.Resolved, "the completing confirmation")
	h.expect(h.send(a, 4, ref, k0, true), tabletest.Closed, "a change after the last confirmation")
	can, _ := h.tb.Canonical(t.Context())
	if len(can.Batches) != 1 || can.Batches[0].Picks[0].Key != k1 {
		t.Fatalf("batches %+v; want one, with seat 0's confirmed change", can.Batches)
	}
	if sn := h.snap(a); sn.Ref == ref || sn.Pick != "" || len(sn.Confirmed) != 0 {
		t.Fatalf("the next point %+v reuses the reference or keeps a pick", sn)
	}
}

// TestStaleOnceForgotten: the seq memory is per user, so another seat's
// client never pushes a user's seq out. With room for one client per user,
// a user's second tab does; the pick's floor still answers for the first:
// the seq that last changed the pick is a duplicate, an older one is
// stale, and neither reverts the newer pick. Seq 0 is a seq like any other.
func TestStaleOnceForgotten(t *testing.T) {
	h := newHarness(t, 3, 1)
	a, b, tab := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.TabClient(0)
	h.tb.Attach(tab, 0)
	sa := h.snap(a)
	h.expect(h.send(a, 0, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "a draft at seq 0")
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[1], false), tabletest.Accepted, "a change")
	h.expect(h.send(b, 1, sa.Ref, h.snap(b).Legal[0], false), tabletest.Accepted, "another user's client")
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[1], false), tabletest.Duplicate, "the change again, still remembered")
	h.expect(h.send(tab, 0, sa.Ref, sa.Legal[2], false), tabletest.Accepted, "the second tab, pushing the first out")
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[1], false), tabletest.Duplicate, "the forgotten last seq")
	h.expect(h.send(a, 0, sa.Ref, sa.Legal[0], false), tabletest.Stale, "the forgotten older seq")
	if sn := h.snap(a); sn.Pick != sa.Legal[2] {
		t.Fatalf("a forgotten seq changed the pick to %q", sn.Pick)
	}
	h.expect(h.send(a, 2, sa.Ref, sa.Legal[1], true), tabletest.Accepted, "a newer seq")
}

// TestPickFloorBound: a pick keeps floors for at most PickClients clients;
// a further client is refused with rate and changes nothing, while a
// client with a floor still changes the pick, and the next point starts
// afresh.
func TestPickFloorBound(t *testing.T) {
	tb, err := direct.New(t.Context(), direct.Options{Setup: fakeengine.Setup{Seed: 77, Seats: 3}, PickClients: 2})
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t, tb}
	a, tab, third := tabletest.SeatClient(0), tabletest.TabClient(0), tabletest.ClientID("seat-0-tab-3")
	h.tb.Attach(tab, 0)
	h.tb.Attach(third, 0)
	sa := h.snap(a)
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "the first client")
	h.expect(h.send(tab, 1, sa.Ref, sa.Legal[1], false), tabletest.Accepted, "the second client")
	h.expect(h.send(third, 1, sa.Ref, sa.Legal[2], true), tabletest.Rate, "a third client")
	if sn := h.snap(a); sn.Pick != sa.Legal[1] || sn.Confirm {
		t.Fatalf("a refused client changed the pick: %+v", sn)
	}
	h.expect(h.send(a, 2, sa.Ref, sa.Legal[2], true), tabletest.Accepted, "a client with a floor")
	for _, s := range []int{1, 2} {
		o := h.snap(tabletest.SeatClient(s))
		h.send(tabletest.SeatClient(s), 1, o.Ref, o.Legal[0], true)
	}
	next := h.snap(third)
	h.expect(h.send(third, 1, next.Ref, next.Legal[0], false), tabletest.Accepted, "the third client at the next point")
}

// TestSecondTab: two tabs of one seat keep separate seqs; a lower seq from
// the other tab is not stale.
func TestSecondTab(t *testing.T) {
	h := newHarness(t, 3, 32)
	a := tabletest.SeatClient(0)
	h.tb.Attach("seat-0-tab-2", 0)
	sa := h.snap(a)
	h.expect(h.send(a, 5, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "tab one")
	h.expect(h.send("seat-0-tab-2", 1, sa.Ref, sa.Legal[1], true), tabletest.Accepted, "tab two")
	if sn := h.snap("seat-0-tab-2"); sn.Pick != sa.Legal[1] || !sn.Confirm {
		t.Fatalf("tab two sees %+v", sn)
	}
}

func TestOneSeatPointRefusesOthers(t *testing.T) {
	for seed := uint64(0); seed < 300; seed++ {
		tb, err := direct.New(t.Context(), direct.Options{Setup: fakeengine.Setup{Seed: seed, Seats: 4}})
		if err != nil {
			t.Fatal(err)
		}
		seq := uint64(0)
		for range 12 {
			seq++
			sn, _ := tb.Snapshot(t.Context(), tabletest.SeatClient(0))
			if sn.Over {
				break
			}
			if len(sn.Required) == 1 {
				other := (sn.Required[0] + 1) % 4
				osn, _ := tb.Snapshot(t.Context(), tabletest.SeatClient(other))
				o, _ := tb.Submit(t.Context(), tabletest.SeatClient(other), tabletest.Submission{Seq: 100, Ref: sn.Ref, Key: "sell:amber", Confirm: true})
				if o != tabletest.Illegal || len(osn.Legal) != 0 {
					t.Fatalf("a seat the point does not require: %s, offered %v", o, osn.Legal)
				}
				return
			}
			for _, s := range sn.Required {
				ssn, _ := tb.Snapshot(t.Context(), tabletest.SeatClient(s))
				if _, err := tb.Submit(t.Context(), tabletest.SeatClient(s), tabletest.Submission{Seq: seq, Ref: ssn.Ref, Key: ssn.Legal[0], Confirm: true}); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	t.Fatal("no game reached a one-seat point")
}

// TestSwitch: a seat switched to the spectator is the spectator's from then
// on, its pending pick kept and still changeable, and its former client is
// forbidden; a seat switched to a bot has its pending pick confirmed, which
// resolves the point when it was the last. Switching a seat to a client
// that controls one is refused, and changes nothing.
func TestSwitch(t *testing.T) {
	h := newHarness(t, 3, 32)
	ctx := t.Context()
	a, b, c, spec := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SpectatorClient
	ref := h.snap(a).Ref
	k0, k1 := h.snap(a).Legal[0], h.snap(b).Legal[0]
	h.expect(h.send(a, 1, ref, k0, false), tabletest.Accepted, "seat 0's draft")
	if _, err := h.tb.Switch(ctx, 0, b); err == nil {
		t.Fatal("a seated client took a second seat")
	}
	if o, err := h.tb.Switch(ctx, 0, spec); err != nil || o != tabletest.Accepted {
		t.Fatalf("seat 0 to the spectator: %s %v", o, err)
	}
	if sn := h.snap(spec); sn.Seat != 0 || sn.Pick != k0 || sn.Confirm {
		t.Fatalf("the spectator in seat 0: %+v", sn)
	}
	h.expect(h.send(a, 2, ref, k0, true), tabletest.Forbidden, "the former controller")
	h.expect(h.send(spec, 1, ref, k0, true), tabletest.Accepted, "the new controller confirms")
	h.expect(h.send(b, 1, ref, k1, false), tabletest.Accepted, "seat 1's draft")
	if o, err := h.tb.Switch(ctx, 1); err != nil || o != tabletest.Accepted {
		t.Fatalf("seat 1 to a bot: %s %v", o, err)
	}
	if sn := h.snap(c); !slices.Equal(sn.Confirmed, []int{0, 1}) {
		t.Fatalf("the bot did not confirm seat 1's pick: %+v", sn)
	}
	h.expect(h.send(c, 1, ref, h.snap(c).Legal[0], false), tabletest.Accepted, "seat 2's draft")
	if o, err := h.tb.Switch(ctx, 2); err != nil || o != tabletest.Resolved {
		t.Fatalf("seat 2 to a bot, the last to confirm: %s %v", o, err)
	}
	can, _ := h.tb.Canonical(ctx)
	if len(can.Batches) != 1 || can.Batches[0].Picks[1].Key != k1 {
		t.Fatalf("batches %+v", can.Batches)
	}
}
