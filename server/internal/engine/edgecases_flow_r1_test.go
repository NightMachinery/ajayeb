package engine_test

import (
	"math/bits"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// This file is a black-box rules-adversary pass (lens: Flow) written from
// docs/rules/** and the rules reference alone. It does not read the engine's
// implementation files. Helper names are prefixed flowR1 to avoid colliding
// with helpers other slices define in this same package.

func flowR1HasEvent(ev []engine.Event, kind engine.EventKind) bool {
	for _, e := range ev {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func flowR1CountEvent(ev []engine.Event, kind engine.EventKind) int {
	n := 0
	for _, e := range ev {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func flowR1HasKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}

func flowR1MustCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("test setup: %s is not a registered card", slug)
	}
	return id
}

func flowR1FirstKeyWithPrefix(t *testing.T, keys []string, prefix string) string {
	t.Helper()
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			return k
		}
	}
	t.Fatalf("test setup: no legal key with prefix %q, got %v", prefix, keys)
	return ""
}

// TestAdvFlowR1_WonderDraftPickPatternAndAutoLastPick checks the wonder
// draft's A-B-B-A / B-A-A-B pick pattern (rules-reference.md section 2
// "Setup"; setup.md "The wonder draft"), that the last pick of each round
// is made automatically by the engine with no key of its own ("The
// automatic last pick of each round is made by the engine itself; no key
// is sent for it"), that round 2's wonders stay hidden until round 1 ends,
// and that Age I is laid out only once the whole draft ends.
func TestAdvFlowR1_WonderDraftPickPatternAndAutoLastPick(t *testing.T) {
	s := engine.NewGame(777, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})

	if s.Phase != engine.PhaseWonderDraft {
		t.Fatalf("expected PhaseWonderDraft at the start, got %v", s.Phase)
	}
	if s.ToMove != 0 {
		t.Fatalf("expected seat 0 (the first player) to pick first, got seat %d", s.ToMove)
	}
	if engine.Occupied(&s) != 0 || s.Age != 0 {
		t.Fatalf("expected no age dealt during the draft, got age %d with %d occupied slots",
			s.Age, bits.OnesCount32(engine.Occupied(&s)))
	}

	round1 := enginetest.Keys(&s)
	if len(round1) != 4 {
		t.Fatalf("expected 4 wonders on offer in round 1, got %v", round1)
	}

	// Pick 1: seat 0 (A) takes one.
	ev := enginetest.Do(t, &s, round1[0])
	if flowR1CountEvent(ev, engine.EvDraft) != 1 {
		t.Fatalf("expected exactly one draft pick event, got %v", ev)
	}
	if s.ToMove != 1 {
		t.Fatalf("expected seat 1 (B) to move next, got seat %d", s.ToMove)
	}
	legal := enginetest.Keys(&s)
	if len(legal) != 3 {
		t.Fatalf("expected 3 wonders left on offer, got %v", legal)
	}
	for _, k := range legal {
		if !flowR1HasKey(round1, k) {
			t.Fatalf("round 2's wonders must stay hidden during round 1, but saw %s", k)
		}
	}

	// Pick 2: seat 1 (B) takes the first of its two.
	ev = enginetest.Do(t, &s, legal[0])
	if flowR1CountEvent(ev, engine.EvDraft) != 1 {
		t.Fatalf("expected exactly one draft pick event, got %v", ev)
	}
	if s.ToMove != 1 {
		t.Fatalf("expected seat 1 (B) to pick again (B takes two in round 1), got seat %d", s.ToMove)
	}
	legal = enginetest.Keys(&s)
	if len(legal) != 2 {
		t.Fatalf("expected 2 wonders left on offer, got %v", legal)
	}
	for _, k := range legal {
		if !flowR1HasKey(round1, k) {
			t.Fatalf("round 2's wonders must stay hidden during round 1, but saw %s", k)
		}
	}

	// Pick 3: seat 1 (B) takes the second of its two; round 1's last wonder
	// must be assigned to seat 0 (A) automatically, in this same action.
	ev = enginetest.Do(t, &s, legal[0])
	manual, auto := 0, 0
	var manualSeat, autoSeat engine.Seat = 255, 255
	for _, e := range ev {
		if e.Kind != engine.EvDraft {
			continue
		}
		if e.A == 1 {
			auto++
			autoSeat = e.Seat
		} else {
			manual++
			manualSeat = e.Seat
		}
	}
	if manual != 1 || auto != 1 {
		t.Fatalf("expected one manual and one automatic draft pick in this action, got %v", ev)
	}
	if manualSeat != 1 {
		t.Fatalf("expected the manual pick to be seat 1's, got seat %d", manualSeat)
	}
	if autoSeat != 0 {
		t.Fatalf("expected round 1's automatic last pick to go to seat 0 (A), got seat %d", autoSeat)
	}
	if !flowR1HasEvent(ev, engine.EvDraftRound) {
		t.Fatalf("expected round 2's wonders to be revealed as soon as round 1 ends, got %v", ev)
	}
	if s.ToMove != 1 {
		t.Fatalf("expected round 2 to start with seat 1 (B), got seat %d", s.ToMove)
	}
	round2 := enginetest.Keys(&s)
	if len(round2) != 4 {
		t.Fatalf("expected 4 new wonders on offer in round 2, got %v", round2)
	}
	for _, k := range round2 {
		if flowR1HasKey(round1, k) {
			t.Fatalf("round 2 must offer 4 wonders never seen in round 1, but saw %s again", k)
		}
	}

	// Pick 4: seat 1 (B) takes its one pick of round 2.
	enginetest.Do(t, &s, round2[0])
	if s.ToMove != 0 {
		t.Fatalf("expected seat 0 (A) to move next in round 2, got seat %d", s.ToMove)
	}
	legal = enginetest.Keys(&s)
	if len(legal) != 3 {
		t.Fatalf("expected 3 wonders left in round 2, got %v", legal)
	}

	// Pick 5: seat 0 (A) takes the first of its two.
	enginetest.Do(t, &s, legal[0])
	if s.ToMove != 0 {
		t.Fatalf("expected seat 0 (A) to pick again (A takes two in round 2), got seat %d", s.ToMove)
	}
	legal = enginetest.Keys(&s)
	if len(legal) != 2 {
		t.Fatalf("expected 2 wonders left in round 2, got %v", legal)
	}

	// Pick 6: seat 0 (A) takes the second of its two; the last wonder must
	// go to seat 1 (B) automatically, the draft ends and Age I is dealt.
	ev = enginetest.Do(t, &s, legal[0])
	manual, auto = 0, 0
	manualSeat, autoSeat = 255, 255
	for _, e := range ev {
		if e.Kind != engine.EvDraft {
			continue
		}
		if e.A == 1 {
			auto++
			autoSeat = e.Seat
		} else {
			manual++
			manualSeat = e.Seat
		}
	}
	if manual != 1 || auto != 1 {
		t.Fatalf("expected one manual and one automatic draft pick in this action, got %v", ev)
	}
	if manualSeat != 0 || autoSeat != 1 {
		t.Fatalf("expected seat 0's manual pick and seat 1's automatic last pick, got manual=%d auto=%d",
			manualSeat, autoSeat)
	}
	if !flowR1HasEvent(ev, engine.EvAgeStart) {
		t.Fatalf("expected Age I to be dealt as soon as the draft ends, got %v", ev)
	}
	if s.Phase != engine.PhasePlay {
		t.Fatalf("expected PhasePlay once the draft ends, got %v", s.Phase)
	}
	if s.Age != 1 {
		t.Fatalf("expected Age 1, got %d", s.Age)
	}
	if s.ToMove != 0 {
		t.Fatalf("expected the first player to start Age I, got seat %d", s.ToMove)
	}
	if got := bits.OnesCount32(engine.Accessible(&s)); got != 6 {
		t.Fatalf("expected 6 accessible cards at the start of Age I, got %d", got)
	}
	for seat := 0; seat < 2; seat++ {
		n := 0
		for _, w := range s.Players[seat].Wonders {
			if w.ID != 0 {
				n++
			}
		}
		if n != 4 {
			t.Fatalf("expected seat %d to end the draft with 4 wonders, got %d", seat, n)
		}
	}
}

// TestAdvFlowR1_AccessibleCountsAtStartOfEachAge pins setup.md "Card
// structures": 6, 2 and 2 cards are accessible at the start of Ages I, II
// and III.
func TestAdvFlowR1_AccessibleCountsAtStartOfEachAge(t *testing.T) {
	want := map[int]int{1: 6, 2: 2, 3: 2}
	for age, n := range want {
		s := enginetest.New(t, uint64(100+age)).Age(age).State()
		if got := bits.OnesCount32(engine.Accessible(&s)); got != n {
			t.Errorf("age %d: expected %d accessible cards at the start, got %d", age, n, got)
		}
		if got := bits.OnesCount32(engine.Occupied(&s)); got != engine.SlotsPerAge {
			t.Errorf("age %d: expected all %d slots occupied at the start, got %d", age, engine.SlotsPerAge, got)
		}
	}
}

// TestAdvFlowR1_PartialCoverageLeavesCardInaccessible pins setup.md "The
// overlap rule": a card at (r, c) is accessible only when neither (r+1,
// c-1) nor (r+1, c+1) still holds a card. Removing just one of its two
// coverers must leave it inaccessible; removing both must free it.
func TestAdvFlowR1_PartialCoverageLeavesCardInaccessible(t *testing.T) {
	s := enginetest.New(t, 55).Age(2).State()

	target := -1
	var covA, covB int
	accStart := engine.Accessible(&s)
	for i := 0; i < engine.SlotsPerAge; i++ {
		if accStart&(1<<uint(i)) != 0 {
			continue // already accessible: not useful as our covered target
		}
		cov := engine.Coverers(2, i)
		if cov[0] < 0 || cov[1] < 0 {
			continue
		}
		if accStart&(1<<uint(cov[0])) != 0 && accStart&(1<<uint(cov[1])) != 0 {
			target, covA, covB = i, int(cov[0]), int(cov[1])
			break
		}
	}
	if target < 0 {
		t.Fatalf("test setup: found no slot covered by exactly two currently-accessible coverers")
	}
	wasFaceUp := s.Slots[target].FaceUp

	firstCard := s.Slots[covA].Card
	ev := enginetest.Do(t, &s, engine.DiscardAction(firstCard).Key())
	if engine.Accessible(&s)&(1<<uint(target)) != 0 {
		t.Fatalf("slot %d became accessible after only one of its two coverers was removed", target)
	}
	for _, e := range ev {
		if e.Kind == engine.EvReveal && e.A == int16(target) {
			t.Fatalf("slot %d must not be revealed while still partly covered, got %v", target, ev)
		}
	}

	secondCard := s.Slots[covB].Card
	ev = enginetest.Do(t, &s, engine.DiscardAction(secondCard).Key())
	if engine.Accessible(&s)&(1<<uint(target)) == 0 {
		t.Fatalf("slot %d is still inaccessible after both of its coverers were taken", target)
	}
	if !wasFaceUp {
		found := false
		for _, e := range ev {
			if e.Kind == engine.EvReveal && e.A == int16(target) {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a reveal event for slot %d once it was fully freed, got %v", target, ev)
		}
		if !s.Slots[target].FaceUp {
			t.Fatalf("expected slot %d to be face up once revealed", target)
		}
	}
}

// TestAdvFlowR1_CoveredFaceUpCardNeedsNoRevealEvent pins setup.md "Face-up
// and face-down rows" together with "the cards at the ends of row 1 are
// each overlapped by a single card of row 2" for Age II: uncovering an
// already face-up card (an odd row) must not produce a reveal event, since
// EvReveal marks a face-down-to-face-up transition, not a mere change of
// accessibility.
func TestAdvFlowR1_CoveredFaceUpCardNeedsNoRevealEvent(t *testing.T) {
	s := enginetest.New(t, 9).Age(2).OnlyLeft(
		"sawmill", "shelf-quarry", "brickyard", "glass-blower", "drying-room", "horse-breeders", // row 1
		"barracks", // the first card of row 2
	).ToMove(0).State()

	accBefore := engine.Accessible(&s)
	const row1Mask = uint32(0x3F) // slots 0-5
	coveredMask := row1Mask &^ accBefore
	if coveredMask == 0 {
		t.Fatalf("test setup: expected the lone row-2 card to cover at least one row-1 slot")
	}
	for i := 0; i < 6; i++ {
		if coveredMask&(1<<uint(i)) != 0 && !s.Slots[i].FaceUp {
			t.Fatalf("row 1 is dealt face up; slot %d should already be face up", i)
		}
	}

	card6 := s.Slots[6].Card
	ev := enginetest.Do(t, &s, engine.DiscardAction(card6).Key())

	if engine.Accessible(&s)&coveredMask == 0 {
		t.Fatalf("expected the covered row-1 slot(s) to become accessible once their only coverer is gone")
	}
	if flowR1HasEvent(ev, engine.EvReveal) {
		t.Fatalf("uncovering an already face-up card must not fire a reveal event, got %v", ev)
	}
}

// TestAdvFlowR1_RevealWaitsForPendingChoice pins turn.md "The end of a
// turn": pending choices are settled before newly accessible cards are
// revealed ("The reveal comes after the pending choices on purpose"). An
// action that both completes a science pair and frees a new face-down card
// must not reveal that card until the resulting token choice is resolved.
func TestAdvFlowR1_RevealWaitsForPendingChoice(t *testing.T) {
	s := enginetest.New(t, 11).
		Age(2).
		City(0, "pharmacist").
		Coins(0, 30).
		Accessible("dispensary").
		ToMove(0).
		State()

	accBefore := bits.OnesCount32(engine.Accessible(&s))

	ev := enginetest.Do(t, &s, "construct:dispensary")

	accAfter := bits.OnesCount32(engine.Accessible(&s))
	if accAfter < accBefore {
		t.Fatalf("test setup: constructing dispensary did not free a new card (accessible count %d -> %d)",
			accBefore, accAfter)
	}
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("expected PhasePickBoardToken after completing the mortar pair, got %v", s.Phase)
	}
	if flowR1HasEvent(ev, engine.EvReveal) {
		t.Fatalf("expected the reveal to wait for the pending token choice, got %v", ev)
	}

	foundHidden := false
	acc := engine.Accessible(&s)
	for i := 0; i < engine.SlotsPerAge; i++ {
		if acc&(1<<uint(i)) != 0 && s.Slots[i].Card != 0 && !s.Slots[i].FaceUp {
			foundHidden = true
		}
	}
	if !foundHidden {
		t.Fatalf("test setup: expected a freed card still waiting to be revealed during the pending choice")
	}

	legal := enginetest.Keys(&s)
	if len(legal) == 0 {
		t.Fatalf("test setup: expected at least one board token to choose from")
	}
	ev2 := enginetest.Do(t, &s, legal[0])
	if !flowR1HasEvent(ev2, engine.EvReveal) {
		t.Fatalf("expected the freed card to be revealed once the pending choice was resolved, got %v", ev2)
	}
}

// TestAdvFlowR1_ThereIsAlwaysALegalDiscard pins turn.md "There is always a
// move": discarding is always legal, whatever the card and whatever the
// player's coins.
func TestAdvFlowR1_ThereIsAlwaysALegalDiscard(t *testing.T) {
	s := enginetest.New(t, 3).Age(1).Coins(0, 0).ToMove(0).State()
	acc := engine.Accessible(&s)
	n := 0
	for i := 0; i < engine.SlotsPerAge; i++ {
		if acc&(1<<uint(i)) == 0 {
			continue
		}
		n++
		key := engine.DiscardAction(s.Slots[i].Card).Key()
		if !enginetest.IsLegal(&s, key) {
			t.Errorf("expected %s to be legal even with 0 coins", key)
		}
	}
	if n == 0 {
		t.Fatalf("test setup: no accessible cards found")
	}
}

// TestAdvFlowR1_WonderTuckIgnoresTuckedCardsOwnCost pins turn.md "3. Build
// a wonder" ("Any accessible card can be used, including one you could not
// afford to construct") and rules-reference.md section 8 item 1. With
// coins set to exactly the wonder's own cost, every accessible card
// (whatever its own price) must still be legal to tuck.
func TestAdvFlowR1_WonderTuckIgnoresTuckedCardsOwnCost(t *testing.T) {
	probe := enginetest.New(t, 2024).Age(1).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Coins(0, 999).State()
	cost := engine.WonderQuote(&probe, 0, catalog.WonderPyramids).Total
	if cost <= 0 {
		t.Fatalf("test setup: unexpected pyramids cost %d", cost)
	}

	s := enginetest.New(t, 2024).Age(1).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Coins(0, int(cost)).ToMove(0).State()

	legal := enginetest.Keys(&s)
	acc := engine.Accessible(&s)
	checked := 0
	for i := 0; i < engine.SlotsPerAge; i++ {
		if acc&(1<<uint(i)) == 0 {
			continue
		}
		slug := s.Slots[i].Card.String()
		key := "wonder:pyramids:" + slug
		if !flowR1HasKey(legal, key) {
			t.Errorf("expected %s to be legal with exactly the wonder's own cost (%d coins); "+
				"the tucked card's own price must not matter", key, cost)
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("test setup: no accessible cards found")
	}
}

// TestAdvFlowR1_EndOfAgeCenteredPawnChooserAndExtraTurnLost pins turn.md
// "Lost at the end of an age" and ages-scoring.md "Who starts the next
// age": an extra turn earned on the age's last card is lost; the next
// age's structure is laid out before the starter is chosen; with the pawn
// centred, the last active player chooses (decisions.md "Centred pawn").
func TestAdvFlowR1_EndOfAgeCenteredPawnChooserAndExtraTurnLost(t *testing.T) {
	s := enginetest.New(t, 4).
		Age(1).
		Wonders(0, "sphinx", "mausoleum", "great-library", "pyramids").
		Coins(0, 30).
		OnlyLeft("lumber-yard").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:sphinx:lumber-yard")

	if !flowR1HasEvent(ev, engine.EvExtraTurnLost) {
		t.Fatalf("expected the sphinx's extra turn to be lost on the age's last card, got %v", ev)
	}
	if !flowR1HasEvent(ev, engine.EvAgeStart) {
		t.Fatalf("expected Age II to be dealt immediately, before the starter is chosen, got %v", ev)
	}
	if s.Age != 2 {
		t.Fatalf("expected Age 2, got %d", s.Age)
	}
	if s.Phase != engine.PhaseChooseStarter {
		t.Fatalf("expected PhaseChooseStarter with the pawn centred, got %v", s.Phase)
	}
	if got := bits.OnesCount32(engine.Accessible(&s)); got != 2 {
		t.Fatalf("expected Age II's structure to already be laid out (2 accessible cards) before the choice, got %d", got)
	}
	if s.ToMove != 0 {
		t.Fatalf("expected the last active player (seat 0) to choose with the pawn centred, got seat %d", s.ToMove)
	}
	legal := enginetest.Keys(&s)
	if len(legal) != 2 || !flowR1HasKey(legal, "starter:0") || !flowR1HasKey(legal, "starter:1") {
		t.Fatalf("expected both starter:0 and starter:1 to be offered, got %v", legal)
	}

	ev2 := enginetest.Do(t, &s, "starter:1")
	if !flowR1HasEvent(ev2, engine.EvStarterChosen) {
		t.Fatalf("expected an EvStarterChosen event, got %v", ev2)
	}
	if s.Phase != engine.PhasePlay {
		t.Fatalf("expected PhasePlay once the starter is chosen, got %v", s.Phase)
	}
	if s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("expected seat 1 to start Age II, got ToMove=%d Active=%d", s.ToMove, s.Active)
	}
}

// TestAdvFlowR1_OffCenterPawnWeakerPlayerChoosesEvenIfNotLastActive pins
// ages-scoring.md "Who starts the next age": off-centre, the weaker player
// (the one on whose side the pawn stands) chooses, even when that is not
// the player who took the age's last card.
func TestAdvFlowR1_OffCenterPawnWeakerPlayerChoosesEvenIfNotLastActive(t *testing.T) {
	s := enginetest.New(t, 5).
		Age(1).
		Pawn(3). // seat 0 is ahead; seat 1 is the weaker player
		OnlyLeft("lumber-yard").
		ToMove(0). // seat 0 will be the last active player
		State()

	enginetest.Do(t, &s, "discard:lumber-yard")

	if s.Phase != engine.PhaseChooseStarter {
		t.Fatalf("expected PhaseChooseStarter, got %v", s.Phase)
	}
	if s.Active != 0 {
		t.Fatalf("expected the last active player to remain seat 0, got %d", s.Active)
	}
	if s.ToMove != 1 {
		t.Fatalf("expected the weaker seat (1) to choose, even though seat 0 was last active, got seat %d", s.ToMove)
	}
}

// TestAdvFlowR1_LastActiveStartsOptionSkipsTheChoicePhase pins
// decisions.md "Centred pawn", alternative: with CenterPawn set to
// LastActiveStarts, the last active player simply starts the next age and
// no choice phase opens at all.
func TestAdvFlowR1_LastActiveStartsOptionSkipsTheChoicePhase(t *testing.T) {
	s := enginetest.New(t, 6).
		Options(engine.Options{CenterPawn: engine.LastActiveStarts}).
		Age(1).
		OnlyLeft("lumber-yard").
		ToMove(1).
		State()

	ev := enginetest.Do(t, &s, "discard:lumber-yard")

	if flowR1HasEvent(ev, engine.EvStarterChosen) {
		t.Fatalf("did not expect a starter-choice event under LastActiveStarts, got %v", ev)
	}
	if s.Phase != engine.PhasePlay {
		t.Fatalf("expected PhasePlay directly (no PhaseChooseStarter) under LastActiveStarts, got %v", s.Phase)
	}
	if s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("expected the last active player (seat 1) to start the next age directly, got ToMove=%d Active=%d",
			s.ToMove, s.Active)
	}
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "starter:") {
			t.Fatalf("did not expect any starter: key under LastActiveStarts, got %s", k)
		}
	}
}

// TestAdvFlowR1_ExtraTurnHonoredThenLostAtAgeEnd pins turn.md "Extra
// turns": an extra turn earned mid-age is honoured (the same player plays
// on), while "Lost at the end of an age" says one earned on the very last
// card of the age is lost instead.
func TestAdvFlowR1_ExtraTurnHonoredThenLostAtAgeEnd(t *testing.T) {
	s := enginetest.New(t, 7).
		Age(1).
		Wonders(0, "sphinx", "mausoleum", "great-library", "pyramids").
		Coins(0, 30).
		OnlyLeft("lumber-yard", "quarry").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:sphinx:lumber-yard")
	if !flowR1HasEvent(ev, engine.EvExtraTurn) {
		t.Fatalf("expected the sphinx's extra turn to be honoured mid-age, got %v", ev)
	}
	if flowR1HasEvent(ev, engine.EvExtraTurnLost) {
		t.Fatalf("did not expect the extra turn to be lost mid-age, got %v", ev)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("expected seat 0 to play again immediately, got Phase=%v ToMove=%d", s.Phase, s.ToMove)
	}
	if got := bits.OnesCount32(engine.Accessible(&s)); got != 1 {
		t.Fatalf("expected exactly 1 card left accessible, got %d", got)
	}

	ev = enginetest.Do(t, &s, "discard:quarry")
	if !flowR1HasEvent(ev, engine.EvAgeStart) {
		t.Fatalf("expected the age to end and Age II to be dealt, got %v", ev)
	}
	if s.Age != 2 {
		t.Fatalf("expected Age 2, got %d", s.Age)
	}
}

// TestAdvFlowR1_CarryOverAcrossAges pins ages-scoring.md "The end of an
// age": "Nothing is reset between ages. Coins, cities, wonders, progress
// tokens, the pawn's position and the looting tokens still on the track
// all carry over."
func TestAdvFlowR1_CarryOverAcrossAges(t *testing.T) {
	s := enginetest.New(t, 8).
		Age(1).
		City(0, "lumber-yard").
		Coins(0, 15).
		Tokens(0, "theology").
		Pawn(2).
		OnlyLeft("quarry").
		ToMove(1).
		State()

	coinsBefore := s.Players[0].Coins
	cityBefore := s.Players[0].City
	tokensBefore := s.Players[0].Tokens
	pawnBefore := s.Military
	lootBefore := s.Looting

	enginetest.Do(t, &s, "discard:quarry")

	if s.Age != 2 {
		t.Fatalf("expected Age 2, got %d", s.Age)
	}
	if s.Players[0].Coins != coinsBefore {
		t.Errorf("seat 0's coins changed across the age boundary: %d -> %d", coinsBefore, s.Players[0].Coins)
	}
	if s.Players[0].City != cityBefore {
		t.Errorf("seat 0's city changed across the age boundary")
	}
	if s.Players[0].Tokens != tokensBefore {
		t.Errorf("seat 0's tokens changed across the age boundary")
	}
	if s.Military != pawnBefore {
		t.Errorf("the pawn's position changed across the age boundary: %d -> %d", pawnBefore, s.Military)
	}
	if s.Looting != lootBefore {
		t.Errorf("the looting tokens changed across the age boundary: %08b -> %08b", lootBefore, s.Looting)
	}
}

// TestAdvFlowR1_MilitarySupremacySkipsLaterPartOfTheSameAction pins
// decisions.md "Supremacy timing": a military supremacy ends the game
// inside the pawn's move, so nothing later in the same action happens, for
// example the Statue of Zeus's destroy choice.
func TestAdvFlowR1_MilitarySupremacySkipsLaterPartOfTheSameAction(t *testing.T) {
	s := enginetest.New(t, 12).
		Age(1).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		Coins(0, 30).
		City(1, "lumber-yard").
		Pawn(8). // one more shield from seat 0 reaches seat 1's capital (9)
		ToMove(0).
		State()

	key := flowR1FirstKeyWithPrefix(t, enginetest.Keys(&s), "wonder:statue-of-zeus:")
	ev := enginetest.Do(t, &s, key)

	if !s.Over() {
		t.Fatalf("expected the game to be over")
	}
	if s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
		t.Fatalf("expected a military victory for seat 0, got kind=%v winner=%d", s.Result.Kind, s.Result.Winner)
	}
	if s.Phase != engine.PhaseGameOver {
		t.Fatalf("expected PhaseGameOver, got %v", s.Phase)
	}
	if flowR1HasEvent(ev, engine.EvDestroy) {
		t.Fatalf("did not expect the destroy choice to run after the game already ended, got %v", ev)
	}
	lumberYard := flowR1MustCard(t, "lumber-yard")
	if !s.Players[1].City.Has(lumberYard) {
		t.Fatalf("expected the opponent's brown card to survive, since the destroy choice never ran")
	}
}

// TestAdvFlowR1_ScientificSupremacyFromConstructingTheSixthSymbolCard pins
// military-science.md "Scientific supremacy": owning six different science
// symbols wins at once, and "the sixth symbol can come from constructing a
// green card."
func TestAdvFlowR1_ScientificSupremacyFromConstructingTheSixthSymbolCard(t *testing.T) {
	s := enginetest.New(t, 13).
		Age(3).
		City(0, "scriptorium", "workshop", "apothecary", "academy", "pharmacist").
		Coins(0, 30).
		Accessible("university").
		ToMove(0).
		State()

	if got := engine.DistinctSymbols(&s.Players[0]); got != 5 {
		t.Fatalf("test setup: expected 5 distinct symbols before, got %d", got)
	}

	enginetest.Do(t, &s, "construct:university")

	if !s.Over() {
		t.Fatalf("expected the game to be over")
	}
	if s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("expected a scientific victory for seat 0, got kind=%v winner=%d", s.Result.Kind, s.Result.Winner)
	}
}

// TestAdvFlowR1_ScientificSupremacyFromTheLawTokenAfterADuplicatePair pins
// military-science.md "Scientific supremacy": "A card that completes a
// pair repeats a symbol you already had, so it never adds a sixth
// different symbol itself. The token you then pick can, if it is Law."
func TestAdvFlowR1_ScientificSupremacyFromTheLawTokenAfterADuplicatePair(t *testing.T) {
	s := enginetest.New(t, 14).
		Age(2).
		City(0, "scriptorium", "workshop", "apothecary", "academy", "pharmacist").
		Coins(0, 30).
		BoardTokens("law", "agriculture", "architecture", "masonry", "economy").
		Accessible("dispensary").
		ToMove(0).
		State()

	if got := engine.DistinctSymbols(&s.Players[0]); got != 5 {
		t.Fatalf("test setup: expected 5 distinct symbols before, got %d", got)
	}

	ev := enginetest.Do(t, &s, "construct:dispensary")
	if s.Over() {
		t.Fatalf("completing a duplicate pair must not itself trigger scientific supremacy, but the game ended: %v", ev)
	}
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("expected PhasePickBoardToken after completing the mortar pair, got %v", s.Phase)
	}
	if !flowR1HasKey(enginetest.Keys(&s), "token:law") {
		t.Fatalf("expected token:law to be offered, got %v", enginetest.Keys(&s))
	}

	enginetest.Do(t, &s, "token:law")

	if !s.Over() {
		t.Fatalf("expected taking the Law token to trigger scientific supremacy")
	}
	if s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("expected a scientific victory for seat 0, got kind=%v winner=%d", s.Result.Kind, s.Result.Winner)
	}
}

// TestAdvFlowR1_SupremacyOnTheLastCardOfAnAgeEndsTheGameWithNoNewAge pins
// ages-scoring.md "The end of an age": "If a supremacy happens on the last
// card of an age, the game is over at once and no new age begins."
func TestAdvFlowR1_SupremacyOnTheLastCardOfAnAgeEndsTheGameWithNoNewAge(t *testing.T) {
	s := enginetest.New(t, 15).
		Age(1).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		Coins(0, 30).
		City(1, "lumber-yard").
		Pawn(8).
		OnlyLeft("stable"). // tucked under the wonder, never constructed
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:stable")

	if !s.Over() {
		t.Fatalf("expected the game to be over")
	}
	if s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
		t.Fatalf("expected a military victory for seat 0, got kind=%v winner=%d", s.Result.Kind, s.Result.Winner)
	}
	if s.Phase != engine.PhaseGameOver {
		t.Fatalf("expected PhaseGameOver, got %v", s.Phase)
	}
	if s.Age != 1 {
		t.Fatalf("expected no new age to begin, age stayed at 1, got %d", s.Age)
	}
	if flowR1HasEvent(ev, engine.EvAgeStart) {
		t.Fatalf("did not expect Age II to be dealt, got %v", ev)
	}
	if flowR1HasEvent(ev, engine.EvStarterChosen) {
		t.Fatalf("did not expect a starter choice, got %v", ev)
	}
}

// TestAdvFlowR1_GuildCountsTheOpponentsCityWhenItIsLarger pins
// ages-scoring.md "Guilds": "a guild looks at both cities... the city with
// the most cards of that kind, which may be the opponent's city."
func TestAdvFlowR1_GuildCountsTheOpponentsCityWhenItIsLarger(t *testing.T) {
	s := enginetest.New(t, 16).
		Age(3).
		City(1, "tavern", "brewery"). // 2 yellow cards, none for seat 0
		Coins(0, 30).
		Accessible("merchants-guild").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:merchants-guild")

	merchantsGuild := flowR1MustCard(t, "merchants-guild")
	gotGain := int16(0)
	found := false
	for _, e := range ev {
		if e.Kind == engine.EvCoins && e.Card == merchantsGuild && e.Seat == 0 {
			gotGain = e.A
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a coin reward from the merchants guild, got %v", ev)
	}
	if gotGain != 2 {
		t.Fatalf("expected 2 coins (1 per yellow card in the bigger city, the opponent's), got %d", gotGain)
	}

	scores := engine.Scores(&s)
	if scores[0].Purple != 2 {
		t.Fatalf("expected 2 purple points from the merchants guild, got %d", scores[0].Purple)
	}
}

// TestAdvFlowR1_ShipownersGuildCombinesBrownAndGreyWithinOneCityOnly pins
// ages-scoring.md "Guilds": "Shipowners Guild: brown and grey cards
// together... counted in the single city with the most brown and grey
// cards combined. Brown cards from one city cannot be added to grey cards
// from the other."
func TestAdvFlowR1_ShipownersGuildCombinesBrownAndGreyWithinOneCityOnly(t *testing.T) {
	s := enginetest.New(t, 17).
		Age(3).
		City(0, "lumber-yard", "stone-pit").                        // 2 brown, seat 0's city
		City(1, "clay-pit", "glassworks", "press", "glass-blower"). // 1 brown + 3 grey = 4, one city
		Coins(0, 30).
		Accessible("shipowners-guild").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:shipowners-guild")

	shipownersGuild := flowR1MustCard(t, "shipowners-guild")
	gotGain := int16(-1)
	for _, e := range ev {
		if e.Kind == engine.EvCoins && e.Card == shipownersGuild && e.Seat == 0 {
			gotGain = e.A
		}
	}
	if gotGain != 4 {
		t.Fatalf("expected 4 coins (the opponent's single-city total, not a cross-city sum or the builder's own total), got %d",
			gotGain)
	}

	scores := engine.Scores(&s)
	if scores[0].Purple != 4 {
		t.Fatalf("expected 4 purple points from the shipowners guild, got %d", scores[0].Purple)
	}
}

// TestAdvFlowR1_BuildersGuildPaysNoCoins pins ages-scoring.md "Guilds":
// "Builders Guild: 2 points per wonder built by the player who has built
// the most wonders. It pays no coins."
func TestAdvFlowR1_BuildersGuildPaysNoCoins(t *testing.T) {
	s := enginetest.New(t, 18).
		Age(3).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").
		Built(0, "colossus", "").
		Built(1, "mausoleum", "").
		Coins(0, 30).
		Accessible("builders-guild").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:builders-guild")

	if flowR1HasEvent(ev, engine.EvCoins) {
		t.Fatalf("the builders guild must pay no coins when built, got %v", ev)
	}
	scores := engine.Scores(&s)
	if scores[0].Purple != 4 {
		t.Fatalf("expected 4 purple points (2 per wonder of the city with the most wonders, seat 0's 2), got %d",
			scores[0].Purple)
	}
}

// TestAdvFlowR1_MoneylendersGuildScoresByTheRicherPlayerAtGameEnd pins
// ages-scoring.md "Guilds": "Moneylenders Guild: 1 point per full 3 coins
// held by the richer player at the end. It pays no coins." The richer
// player can be the opponent, not the guild's owner.
func TestAdvFlowR1_MoneylendersGuildScoresByTheRicherPlayerAtGameEnd(t *testing.T) {
	probe := enginetest.New(t, 19).Age(3).Coins(0, 999).Accessible("moneylenders-guild").State()
	cost := engine.BuildingQuote(&probe, 0, flowR1MustCard(t, "moneylenders-guild")).Total

	s := enginetest.New(t, 19).
		Age(3).
		Coins(0, int(cost)+2).
		Coins(1, 50).
		Accessible("moneylenders-guild").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:moneylenders-guild")

	if flowR1HasEvent(ev, engine.EvCoins) {
		t.Fatalf("the moneylenders guild must pay no coins when built, got %v", ev)
	}
	if s.Players[0].Coins >= s.Players[1].Coins {
		t.Fatalf("test setup: expected seat 0 to end up poorer than seat 1 (%d vs %d)",
			s.Players[0].Coins, s.Players[1].Coins)
	}

	scores := engine.Scores(&s)
	want := s.Players[1].Coins / 3
	if scores[0].Purple != want {
		t.Fatalf("expected %d purple points, from the richer player's (seat 1's %d coins) count, got %d",
			want, s.Players[1].Coins, scores[0].Purple)
	}
	if scores[1].Purple != 0 {
		t.Fatalf("expected seat 1 (who does not own the guild) to score 0 purple points, got %d", scores[1].Purple)
	}
}

// TestAdvFlowR1_TieGoesToMoreBlueCardPoints pins ages-scoring.md "Tie-break
// and shared victory": "If the totals are equal, the player with more
// points from blue cards (category 1) wins."
func TestAdvFlowR1_TieGoesToMoreBlueCardPoints(t *testing.T) {
	s := enginetest.New(t, 20).
		Age(3).
		City(0, "baths"). // 3 blue VP
		Coins(0, 1).      // +2 from discarding below -> 3 coins -> 1 coin point; total 4
		Coins(1, 12).     // 4 coin points, no blue; total 4
		OnlyLeft("obelisk").
		ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:obelisk")

	if !s.Over() {
		t.Fatalf("expected the game to be over (the last Age III card was taken)")
	}
	if s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("expected a civilian victory, got %v", s.Result.Kind)
	}
	scores := s.Result.Scores
	if scores[0].Total != scores[1].Total {
		t.Fatalf("test setup: expected tied totals, got %d vs %d", scores[0].Total, scores[1].Total)
	}
	if scores[0].Blue <= scores[1].Blue {
		t.Fatalf("test setup: expected seat 0 to have more blue points, got %d vs %d", scores[0].Blue, scores[1].Blue)
	}
	if s.Result.Winner != 0 {
		t.Fatalf("expected seat 0 to win on the blue-card tie-break, got winner %d", s.Result.Winner)
	}
}

// TestAdvFlowR1_EqualBlueTooSharesVictory pins ages-scoring.md "Tie-break
// and shared victory": "If those are equal too, both players share the
// victory."
func TestAdvFlowR1_EqualBlueTooSharesVictory(t *testing.T) {
	s := enginetest.New(t, 21).
		Age(3).
		City(0, "baths"). // 3 blue VP
		City(1, "altar"). // 3 blue VP
		Coins(0, 1).      // +2 from discarding below -> 3 coins -> 1 coin point
		Coins(1, 3).      // 1 coin point
		OnlyLeft("obelisk").
		ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:obelisk")

	if !s.Over() {
		t.Fatalf("expected the game to be over")
	}
	scores := s.Result.Scores
	if scores[0].Total != scores[1].Total || scores[0].Blue != scores[1].Blue {
		t.Fatalf("test setup: expected a full tie, got %+v vs %+v", scores[0], scores[1])
	}
	if s.Result.Winner != engine.SharedVictory {
		t.Fatalf("expected a shared victory, got winner %d", s.Result.Winner)
	}
}

// TestAdvFlowR1_SevenWonderRuleRemovesTheOtherPlayersLastWonder pins
// wonders-tokens.md "The seven-wonder rule": "When the seventh wonder is
// built, by either player, the one wonder still unbuilt goes back to the
// box at once, whichever player drafted it... From then on, no player can
// build a wonder."
func TestAdvFlowR1_SevenWonderRuleRemovesTheOtherPlayersLastWonder(t *testing.T) {
	s := enginetest.New(t, 22).
		Age(1).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").
		Built(0, "colossus", "").
		Built(0, "great-lighthouse", "").
		Built(1, "mausoleum", "").
		Built(1, "great-library", "").
		Built(1, "piraeus", "").
		Coins(0, 30).
		ToMove(0).
		State()

	if s.WondersBuilt != 6 {
		t.Fatalf("test setup: expected 6 wonders built already, got %d", s.WondersBuilt)
	}

	key := flowR1FirstKeyWithPrefix(t, enginetest.Keys(&s), "wonder:temple-of-artemis:")
	ev := enginetest.Do(t, &s, key)

	if s.WondersBuilt != 7 {
		t.Fatalf("expected 7 wonders built, got %d", s.WondersBuilt)
	}

	var removedEvent engine.Event
	foundRemoved := false
	for _, e := range ev {
		if e.Kind == engine.EvWonderRemoved {
			removedEvent = e
			foundRemoved = true
		}
	}
	if !foundRemoved {
		t.Fatalf("expected an EvWonderRemoved event for the eighth, never-built wonder, got %v", ev)
	}
	if removedEvent.Seat != 1 {
		t.Fatalf("expected the removed wonder to belong to seat 1 (hanging-gardens), got seat %d", removedEvent.Seat)
	}

	var hgSlot *engine.WonderSlot
	for i := range s.Players[1].Wonders {
		if s.Players[1].Wonders[i].ID.String() == "hanging-gardens" {
			hgSlot = &s.Players[1].Wonders[i]
		}
	}
	if hgSlot == nil {
		t.Fatalf("test setup: could not find seat 1's hanging-gardens wonder slot")
	}
	if !hgSlot.Removed || hgSlot.Built {
		t.Fatalf("expected hanging-gardens to be marked Removed and not Built, got %+v", *hgSlot)
	}

	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "wonder:") {
			t.Fatalf("expected no further wonder construction to be legal, got %s", k)
		}
	}
}

// TestAdvFlowR1_StatueOfZeusSkipsDestroyChoiceWhenOpponentHasNoMatchingColour
// pins wonders-tokens.md "The Statue of Zeus and Circus Maximus": "If the
// opponent has no card of that colour, nothing is destroyed," together
// with decisions.md "Forced choices": the choice phase is skipped and the
// wonder is still built, with an EvEffectSkipped event naming the phase it
// would have opened.
func TestAdvFlowR1_StatueOfZeusSkipsDestroyChoiceWhenOpponentHasNoMatchingColour(t *testing.T) {
	s := enginetest.New(t, 23).
		Age(1).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		Coins(0, 30).
		City(1, "glassworks"). // grey, no brown card at all
		ToMove(0).
		State()

	key := flowR1FirstKeyWithPrefix(t, enginetest.Keys(&s), "wonder:statue-of-zeus:")
	ev := enginetest.Do(t, &s, key)

	if !flowR1HasEvent(ev, engine.EvWonder) {
		t.Fatalf("expected the wonder to still be built, got %v", ev)
	}
	if s.Phase != engine.PhasePlay {
		t.Fatalf("expected the destroy choice to be skipped and play to continue normally, got phase %v", s.Phase)
	}
	if flowR1HasEvent(ev, engine.EvDestroy) {
		t.Fatalf("expected nothing to be destroyed (the opponent has no brown card), got %v", ev)
	}
	found := false
	for _, e := range ev {
		if e.Kind == engine.EvEffectSkipped && e.Seat == 0 && e.A == int16(engine.PhaseDestroyCard) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an EvEffectSkipped event naming PhaseDestroyCard, got %v", ev)
	}
	if s.Military != 1 {
		t.Fatalf("expected the shield to still move the pawn by 1, got military=%d", s.Military)
	}
}

// TestAdvFlowR1_GreatLibraryOffersExactlyTheBoxOrderFirstThree pins
// wonders-tokens.md "The Great Library" ("Only the builder sees them")
// together with decisions.md "The Great Library draw": "the order of the
// box is shuffled when the game is created, and the Great Library takes
// the first three."
func TestAdvFlowR1_GreatLibraryOffersExactlyTheBoxOrderFirstThree(t *testing.T) {
	s := enginetest.New(t, 24).
		Age(1).
		Wonders(0, "great-library", "pyramids", "colossus", "sphinx").
		Coins(0, 30).
		BoardTokens("agriculture", "masonry", "strategy", "urbanism", "economy").
		BoxOrder("philosophy", "mathematics", "theology", "architecture", "law").
		ToMove(0).
		State()

	key := flowR1FirstKeyWithPrefix(t, enginetest.Keys(&s), "wonder:great-library:")
	ev := enginetest.Do(t, &s, key)

	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("expected PhasePickLibraryToken, got %v", s.Phase)
	}
	want := map[string]bool{"token:philosophy": true, "token:mathematics": true, "token:theology": true}
	got := enginetest.Keys(&s)
	if len(got) != 3 {
		t.Fatalf("expected exactly 3 tokens offered, got %v", got)
	}
	for _, k := range got {
		if !want[k] {
			t.Fatalf("expected the offer to be exactly the box's first three (philosophy, mathematics, theology), got %v", got)
		}
	}
	n := 0
	for _, e := range ev {
		if e.Kind == engine.EvLibraryDraw {
			n++
			if !e.Private || e.Seat != 0 {
				t.Fatalf("expected each library-draw event to be private to seat 0, got %+v", e)
			}
		}
	}
	if n != 3 {
		t.Fatalf("expected 3 EvLibraryDraw events, got %d", n)
	}
}
