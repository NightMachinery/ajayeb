package room_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Both seats retain all their actual draft picks until the first opponent
// summary that can show them: after the opening card for the second seat,
// and after the reply for the opening seat. Automatic awards are excluded.
func TestReviewDraftPicksReachFirstSummary(t *testing.T) {
	for first := range 2 {
		t.Run(string(rune('0'+first)), func(t *testing.T) {
			h := newHarness(t)
			a := h.client("user-a", "tab-a", "Sun")
			req := wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(first), nil)
			hints := room.DefaultHints(store.ModeOnline)
			hints.Advisor = true
			hints.AdvisorStrength = view.AdvisorStrengthQuick
			req.Hints = &hints
			code, _ := a.create(req)
			b := h.client("user-b", "tab-b", "Moon")
			b.join(code)
			// Requested seat controls the creator's seat, not the draft's
			// first player. Read both assignments from the actual views.
			byseat := [2]*tclient{}
			h.snapshot(code)
			for _, c := range []*tclient{a, b} {
				byseat[c.current().You.Seats[0]] = c
			}
			drafts := [2][]string{{}, {}}
			lastDraft := [2]int{-1, -1}
			for {
				h.snapshot(code)
				st := a.current()
				if st.View.Phase != view.PhaseWonderDraft {
					break
				}
				seat := *st.View.ToMove
				mover := byseat[seat]
				key := mover.current().View.Choices[0].Key
				drafts[seat] = append(drafts[seat], key)
				lastDraft[seat] = st.View.Ply
				mover.act(key, st.Version, nil)
			}
			h.snapshot(code)
			opening := *a.current().View.ToMove
			other := 1 - opening
			checkNoReview(t, "opening seat before its card", byseat[opening].current())
			playCard := func(seat int) int {
				t.Helper()
				h.snapshot(code)
				st := byseat[seat].current()
				for _, choice := range st.View.Choices {
					if strings.HasPrefix(choice.Key, "discard:") {
						byseat[seat].act(choice.Key, st.Version, nil)
						h.snapshot(code)
						return st.View.Ply
					}
				}
				t.Fatal("no discard choice for the opening turn")
				return -1
			}
			checkDrafts := func(seat, ply, count int) {
				t.Helper()
				rv := awaitReview(t, h, byseat[seat], "draft seat", ply)
				got := []string{}
				for _, entry := range rv.Entries {
					if strings.HasPrefix(entry.Played, "draft:") {
						got = append(got, entry.Played)
					}
				}
				if !reflect.DeepEqual(got, drafts[seat]) || len(rv.Entries) != count {
					t.Fatalf("seat %d: entries %+v, want every draft pick %v and %d entries", seat, rv.Entries, drafts[seat], count)
				}
			}
			openingPly := playCard(opening)
			checkDrafts(other, lastDraft[other], len(drafts[other]))
			playCard(other)
			checkDrafts(opening, openingPly, len(drafts[opening])+1)
		})
	}
}

// The move review (the advisor family, review.go): after a seat's play
// action, the room reviews it for that seat only. Only that seat's own
// view carries it, until the seat's next play action, and only while the
// room's advisor family is on: on by default against the bot, off in
// hot-seat and online games, where the admin may turn it on. At the
// default strength, two-stage, the review is computed off the room actor
// and comes in a later frame than the move's; a quick review comes with
// the move's own frame.

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

// awaitReview waits until c's latest state frame carries a review of the
// move at ply, and checks it: a two-stage or deep review comes in a later
// frame than the move's.
func awaitReview(t *testing.T, h *harness, c *tclient, who string, ply int) *view.ReviewView {
	t.Helper()
	h.waitFor(who+"'s review", func() bool {
		h.snapshot(c.code)
		rv := c.current().View.Review
		return rv != nil && rv.Ply == ply
	})
	return checkReview(t, who, c.current(), ply)
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

	// Draft decisions are reviewed for their own seats too.
	lastDraft := [2]int{-1, -1}
	for {
		h.snapshot(code)
		st := a.current()
		if st.View.Phase != view.PhaseWonderDraft {
			break
		}
		lastDraft[*st.View.ToMove] = st.View.Ply
		playOnline(t, h, a, b, 1)
	}
	h.snapshot(code)
	byseat := [2]*tclient{a, b}
	plies := [2]int{-1, -1}
	for seat, c := range byseat {
		if seat == *a.current().View.ToMove {
			checkNoReview(t, "opening seat before its card", c.current())
			continue
		}
		plies[seat] = lastDraft[seat]
		awaitReview(t, h, c, "completed draft", plies[seat])
	}
	checkNoReview(t, "spectator after draft", sp.current())
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
			if plies[s] < 0 {
				checkNoReview(t, c.ClientID, c.current())
				continue
			}
			if rv := awaitReview(t, h, c, c.ClientID, plies[s]); rv.Strength != view.AdvisorStrengthTwoStage {
				t.Fatalf("%s: a review of strength %q, want the default", c.ClientID, rv.Strength)
			}
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
	// The other seat has only draft decisions to review so far.
	h.waitFor("other seat's draft review", func() bool {
		h.snapshot(code)
		return a.current().View.Review != nil
	})
	if a.current().View.Review.Ply >= first {
		t.Fatal("other seat received the first mover's review")
	}

	// The other seat moves: the view is now the first mover's, with its
	// review.
	play()
	st = a.current()
	if *st.View.ToMove == firstSeat {
		awaitReview(t, h, a, "the first mover", first)
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
		if st.View.Phase == view.PhasePlay {
			ply = st.View.Ply
			a.act(st.View.Choices[0].Key, st.Version, nil)
			break
		}
		v := st.Version
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version > v })
	}
	a.waitState(func(s wire.State) bool { return s.View.Ply > ply })
	awaitReview(t, h, a, "after the move", ply)
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
	if st.View.Review != nil && st.View.Review.Ply >= ply {
		t.Fatal("undo kept the undone move's review")
	}
}

// firstPlayAction plays the human seat of a bot room (the bot answering)
// until its first play action, and returns that action's ply.
func firstPlayAction(t *testing.T, a *tclient, st wire.State) int {
	t.Helper()
	for {
		for !st.You.CanAct {
			v := st.Version
			st = a.waitState(func(s wire.State) bool { return s.Version > v })
		}
		v := st.Version
		a.act(st.View.Choices[0].Key, v, nil)
		if st.View.Phase == view.PhasePlay {
			return st.View.Ply
		}
		st = a.waitState(func(s wire.State) bool { return s.Version > v })
	}
}

// TestReviewQuickSetting checks the quick strength through the create
// protocol. Frame timing and the deadline exception are tested separately
// without depending on how fast Greedy runs on the host.
func TestReviewQuickSetting(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	bot := wire.BotLevelEasy
	req := wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil)
	hints := room.DefaultHints(store.ModeAI)
	hints.AdvisorStrength = view.AdvisorStrengthQuick
	req.Hints = &hints
	_, st := a.create(req)
	if st.Room.Hints.AdvisorStrength != view.AdvisorStrengthQuick {
		t.Fatalf("the room's strength is %q", st.Room.Hints.AdvisorStrength)
	}
}

// TestAdvisorStrengthSetting checks the strength in the game's hint
// settings: two-stage by default, changed by the admin's hints frame
// without a room log entry, kept by a frame that leaves it empty, and used
// by the next review.
func TestAdvisorStrengthSetting(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	bot := wire.BotLevelEasy
	code, st := a.create(wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil))
	if got := st.Room.Hints.AdvisorStrength; got != view.AdvisorStrengthTwoStage || room.DefaultAdvisorStrength != view.AdvisorStrengthTwoStage {
		t.Fatalf("a new room's strength is %q", got)
	}

	deep := room.DefaultHints(store.ModeAI)
	deep.AdvisorStrength = view.AdvisorStrengthDeep
	seq := a.sendHints(deep)
	st = a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	if st.Room.Hints != deep || len(st.RoomLog) != 0 || h.snapshot(code).Hints != deep {
		t.Fatalf("after the change: hints %+v, log %+v", st.Room.Hints, st.RoomLog)
	}

	// A client that does not know the strength leaves it empty: the
	// game keeps its own.
	silent := withHint(room.DefaultHints(store.ModeAI), wire.HintFamilyTempo, false)
	silent.AdvisorStrength = ""
	seq = a.sendHints(silent)
	st = a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	if st.Room.Hints.AdvisorStrength != view.AdvisorStrengthDeep || st.Room.Hints.Tempo {
		t.Fatalf("after a change that left the strength out: %+v", st.Room.Hints)
	}

	ply := firstPlayAction(t, a, st)
	if rv := awaitReview(t, h, a, "a deep review", ply); rv.Strength != view.AdvisorStrengthDeep || (rv.Better != nil && rv.Gap != 0) {
		t.Fatalf("the review %+v, want a deep one without points", *rv)
	}
}
