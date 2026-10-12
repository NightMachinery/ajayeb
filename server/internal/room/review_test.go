package room_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The move review (the advisor family, review.go): after a seat's play
// action, the room reviews it for that seat only. Only that seat's own
// view carries it, until the seat's next play action, and only while the
// room's advisor family is on: on by default against the bot, off in
// hot-seat and online games, where the admin may turn it on.

// checkReview fails unless st's view carries a review of the move at ply.
func checkReview(t *testing.T, who string, st wire.State, ply int) *view.ReviewView {
	t.Helper()
	rv := st.View.Review
	if rv == nil {
		t.Fatalf("%s: no review, want one of ply %d", who, ply)
	}
	if rv.Ply != ply || rv.Played == "" || rv.Reason == "" {
		t.Fatalf("%s: review %+v, want one of ply %d", who, *rv, ply)
	}
	if (rv.Better == nil) != (rv.Reason == view.ReviewReasonNone) || (rv.Better == nil && rv.Gap != 0) {
		t.Fatalf("%s: inconsistent review %+v", who, *rv)
	}
	return rv
}

// checkNoReview fails if st's view carries a review.
func checkNoReview(t *testing.T, who string, st wire.State) {
	t.Helper()
	if st.View.Review != nil {
		t.Fatalf("%s: a review %+v, want none", who, *st.View.Review)
	}
}

// TestReviewOnlyInMoversView checks an online room whose admin turned the
// advisor on: after each play action only the mover's view carries its
// review, the opponent's and a spectator's never do, and a seat keeps its
// review while the other seat moves.
func TestReviewOnlyInMoversView(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	sp := h.client("user-c", "tab-c", "Cy")
	sp.join(code)
	if h.snapshot(code).Hints.Advisor {
		t.Fatal("an online room starts with the advisor on")
	}
	a.sendHints(withHint(room.DefaultHints(store.ModeOnline), wire.HintFamilyAdvisor, true))

	// The draft is not reviewed.
	for {
		h.snapshot(code)
		if a.current().View.Phase != view.PhaseWonderDraft {
			break
		}
		playOnline(t, h, a, b, 1)
	}
	h.snapshot(code)
	for _, c := range []*tclient{a, b, sp} {
		checkNoReview(t, c.ClientID+" after the draft", c.current())
	}

	byseat := [2]*tclient{a, b}
	plies := [2]int{-1, -1}
	for range 4 {
		h.snapshot(code)
		st := a.current()
		b.current()
		sp.current()
		if st.View.Phase != view.PhasePlay {
			break
		}
		seat := *st.View.ToMove
		plies[seat] = st.View.Ply
		mover := byseat[seat]
		mover.act(mover.last.View.Choices[0].Key, st.Version, nil)
		h.snapshot(code)
		for s, c := range byseat {
			got := c.current()
			if plies[s] < 0 {
				checkNoReview(t, c.ClientID, got)
				continue
			}
			checkReview(t, c.ClientID, got, plies[s])
		}
		checkNoReview(t, "the spectator", sp.current())
	}
	if plies[0] < 0 || plies[1] < 0 {
		t.Fatalf("both seats should have moved: %v", plies)
	}
}

// TestReviewFollowsAdvisor checks a hot-seat room: no review while the
// advisor is off (the default); once the admin turns it on, the view of
// the seat to move carries that seat's own review; off again, none.
func TestReviewFollowsAdvisor(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if st.Room.Hints.Advisor {
		t.Fatal("a hot-seat room starts with the advisor on")
	}
	play := func() (int, int) {
		h.snapshot(code)
		st := a.current()
		ply, seat := st.View.Ply, *st.View.ToMove
		a.act(st.View.Choices[0].Key, st.Version, nil)
		h.snapshot(code)
		return ply, seat
	}
	for {
		h.snapshot(code)
		if a.current().View.Phase != view.PhaseWonderDraft {
			break
		}
		play()
	}
	first, firstSeat := play()
	checkNoReview(t, "advisor off", a.current())

	on := withHint(room.DefaultHints(store.ModeHotseat), wire.HintFamilyAdvisor, true)
	a.sendHints(on)
	h.snapshot(code)
	st = a.current()
	if *st.View.ToMove == firstSeat {
		t.Skip("the first play action kept the turn")
	}
	// The view is the other seat's, which has not played yet.
	checkNoReview(t, "the other seat", st)

	// The other seat moves: the view is now the first mover's, with its
	// review.
	play()
	st = a.current()
	if *st.View.ToMove == firstSeat {
		checkReview(t, "the first mover", st, first)
	}

	a.sendHints(room.DefaultHints(store.ModeHotseat))
	h.snapshot(code)
	checkNoReview(t, "advisor off again", a.current())
}

// TestReviewInBotRoom checks a game against the bot, where the advisor is
// on by default: the player's frames carry the review of their play action
// through the bot's reply, and an undo of that move takes it away.
func TestReviewInBotRoom(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	bot := wire.BotLevelEasy
	code, st := a.create(wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil))
	if !st.Room.Hints.Advisor {
		t.Fatal("a bot room starts with the advisor off")
	}
	// Play the human seat until its first play action.
	var ply int
	for {
		for !st.You.CanAct {
			v := st.Version
			st = a.waitState(func(s wire.State) bool { return s.Version > v })
		}
		checkNoReview(t, "before the first play action", st)
		if st.View.Phase == view.PhasePlay {
			ply = st.View.Ply
			a.act(st.View.Choices[0].Key, st.Version, nil)
			break
		}
		v := st.Version
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version > v })
	}
	st = a.waitState(func(s wire.State) bool { return s.View.Ply > ply })
	checkReview(t, "after the move", st, ply)
	// Through the bot's reply.
	h.waitFor("the bot's reply", func() bool { return len(h.snapshot(code).Actions) > ply+1 })
	h.snapshot(code)
	st = a.current()
	if st.View.Ply > ply+1 {
		checkReview(t, "after the bot's reply", st, ply)
	}

	// Undo takes the move back, and its review with it.
	a.undo(st.Version)
	h.snapshot(code)
	st = a.current()
	if st.View.Ply != ply {
		t.Fatalf("undo went back to ply %d, want %d", st.View.Ply, ply)
	}
	checkNoReview(t, "after the undo", st)
}
