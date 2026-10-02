package engine_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// This file is an adversarial pass (lens: military and science) written from
// the rules alone: rules-reference.md sections 2-8, docs/rules/*.md. It does
// not read the engine's implementation files.

// TestAdvMilSciR1_MilitaryPointsZoneOneAndTwoEqualTwo checks that the pawn
// standing on space 1 or on space 2 of either side always scores exactly 2
// military points for the player who is ahead, and 0 for the other.
// Rule: docs/rules/military-science.md "Spaces 1 and 2: 2 VP"; rules-reference.md section 4.
func TestAdvMilSciR1_MilitaryPointsZoneOneAndTwoEqualTwo(t *testing.T) {
	for _, pawn := range []int{1, 2} {
		st := enginetest.New(t, 101).Age(1).Pawn(pawn).State()
		sc := engine.Scores(&st)
		if sc[0].Military != 2 || sc[1].Military != 0 {
			t.Errorf("pawn=%d: got seat0=%d seat1=%d, want 2/0", pawn, sc[0].Military, sc[1].Military)
		}
	}
	for _, pawn := range []int{-1, -2} {
		st := enginetest.New(t, 101).Age(1).Pawn(pawn).State()
		sc := engine.Scores(&st)
		if sc[1].Military != 2 || sc[0].Military != 0 {
			t.Errorf("pawn=%d: got seat0=%d seat1=%d, want 0/2", pawn, sc[0].Military, sc[1].Military)
		}
	}
}

// TestAdvMilSciR1_MilitaryPointsCentreNeitherPlayerScores checks that a
// centred pawn gives military points to neither player.
// Rule: docs/rules/ages-scoring.md "Military: ... nothing for the other player";
// docs/rules/military-science.md "Military points at the end".
func TestAdvMilSciR1_MilitaryPointsCentreNeitherPlayerScores(t *testing.T) {
	st := enginetest.New(t, 103).Age(1).Pawn(0).State()
	sc := engine.Scores(&st)
	if sc[0].Military != 0 || sc[1].Military != 0 {
		t.Fatalf("centred pawn: got seat0=%d seat1=%d, want 0/0", sc[0].Military, sc[1].Military)
	}
}

// TestAdvMilSciR1_MilitaryZoneBoundaryPoints exercises the zone table at and
// around its boundaries (3, 5, 6, 8) on both sides of the track, since a
// boundary is where an off-by-one bug would show.
// Rule: docs/rules/military-science.md zone table ("3 to 5: 5 points",
// "6 to 8: 10 points").
func TestAdvMilSciR1_MilitaryZoneBoundaryPoints(t *testing.T) {
	cases := []struct {
		pawn int
		pts  int16
	}{
		{3, 5}, {4, 5}, {5, 5},
		{6, 10}, {7, 10}, {8, 10},
	}
	for _, c := range cases {
		st := enginetest.New(t, 104).Age(1).Pawn(c.pawn).State()
		sc := engine.Scores(&st)
		if sc[0].Military != c.pts || sc[1].Military != 0 {
			t.Errorf("pawn=%d: got seat0=%d seat1=%d, want %d/0", c.pawn, sc[0].Military, sc[1].Military, c.pts)
		}

		stN := enginetest.New(t, 104).Age(1).Pawn(-c.pawn).State()
		scN := engine.Scores(&stN)
		if scN[1].Military != c.pts || scN[0].Military != 0 {
			t.Errorf("pawn=%d: got seat0=%d seat1=%d, want 0/%d", -c.pawn, scN[0].Military, scN[1].Military, c.pts)
		}
	}
}

// TestAdvMilSciR1_LootingTokenNeverRetriggersAfterRetreatAndReturn checks
// that a looting token already taken does not fire again if the pawn leaves
// its zone and later re-enters it.
// Rule: docs/rules/military-science.md "A token triggers only once. If the
// pawn retreats and enters the zone again later, nothing happens."
func TestAdvMilSciR1_LootingTokenNeverRetriggersAfterRetreatAndReturn(t *testing.T) {
	st := enginetest.New(t, 105).
		Age(1).
		Pawn(-3). // the -3 token has just been "reached", so it is already gone
		Coins(0, 7).
		Coins(1, 7).
		Accessible("guard-tower", "palisade").
		ToMove(0).
		State()

	ev1 := enginetest.Do(t, &st, "construct:guard-tower") // seat0, free, 1 shield: -3 -> -2
	for _, e := range ev1 {
		if e.Kind == engine.EvLoot {
			t.Fatalf("unexpected loot moving away from the zone: %+v", e)
		}
	}
	if st.Military != -2 {
		t.Fatalf("after guard-tower: military = %d, want -2", st.Military)
	}

	ev2 := enginetest.Do(t, &st, "construct:palisade") // seat1, 2 coins, 1 shield: -2 -> -3
	for _, e := range ev2 {
		if e.Kind == engine.EvLoot {
			t.Fatalf("token at -3 retriggered on return: %+v", e)
		}
	}
	if st.Military != -3 {
		t.Fatalf("after palisade: military = %d, want -3", st.Military)
	}
	if st.Players[0].Coins != 7 {
		t.Fatalf("seat0 coins changed to %d, want unchanged 7 (no retrigger)", st.Players[0].Coins)
	}
}

// TestAdvMilSciR1_OneMoveTriggersBothLootingTokensOnASide reproduces the
// worked example of a single move crossing both looting tokens on one side.
// Rule: docs/rules/military-science.md "One move can trigger both tokens on
// a side ... pawn on space 2 of B's side, A constructs Arsenal (3 shields)
// while owning Strategy (1 more). The pawn moves 4 spaces to space 6 ...
// B loses 2 coins and then 5 coins."
func TestAdvMilSciR1_OneMoveTriggersBothLootingTokensOnASide(t *testing.T) {
	st := enginetest.New(t, 106).
		Age(3).
		City(0, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Tokens(0, "strategy").
		Pawn(2).
		Coins(1, 7).
		Accessible("arsenal").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "construct:arsenal")

	var loots []engine.Event
	for _, e := range ev {
		if e.Kind == engine.EvLoot {
			loots = append(loots, e)
		}
	}
	if len(loots) != 2 {
		t.Fatalf("got %d loot events, want 2: %+v", len(loots), loots)
	}
	if loots[0].B != 2 || loots[1].B != 5 {
		t.Fatalf("loot token values = %d, %d; want 2, 5", loots[0].B, loots[1].B)
	}
	for _, l := range loots {
		if l.Seat != 1 {
			t.Fatalf("loot taken from seat %d, want seat 1", l.Seat)
		}
	}
	if st.Military != 6 {
		t.Fatalf("military = %d, want 6 (2 base shields from Arsenal... +1 from Strategy, from space 2)", st.Military)
	}
	if st.Players[1].Coins != 0 {
		t.Fatalf("seat1 coins = %d, want 0 (7 - 2 - 5)", st.Players[1].Coins)
	}
}

// TestAdvMilSciR1_StrategyGivesNoBonusToWonderShields checks that Strategy's
// extra shield applies only to buildings, never to a wonder, even one whose
// own effect is shields (the Colossus).
// Rule: docs/rules/wonders-tokens.md "Strategy: every red building you
// construct afterwards gives 1 extra shield. Wonders get no extra shield.";
// docs/rules/decisions.md "Effect order" / rules-reference.md section 8 item 4.
func TestAdvMilSciR1_StrategyGivesNoBonusToWonderShields(t *testing.T) {
	st := enginetest.New(t, 107).
		Wonders(0, "colossus").
		City(0, "clay-pool", "clay-pit", "brickyard", "glassworks").
		Tokens(0, "strategy").
		Pawn(0).
		Age(1).
		Accessible("quarry").
		ToMove(0).
		State()

	enginetest.Do(t, &st, "wonder:colossus:quarry")

	if st.Military != 2 {
		t.Fatalf("military = %d, want 2 (Colossus's own shields only, no Strategy bonus)", st.Military)
	}
}

// TestAdvMilSciR1_StrategyAppliesToRedCardRebuiltByMausoleum checks that
// Strategy's bonus shield applies to a red card rebuilt for free by the
// Mausoleum, not only to a card bought in the ordinary way.
// Rule: docs/rules/military-science.md "The Strategy token: ... This
// includes a red building rebuilt with the Mausoleum."
func TestAdvMilSciR1_StrategyAppliesToRedCardRebuiltByMausoleum(t *testing.T) {
	st := enginetest.New(t, 108).
		Wonders(0, "mausoleum").
		City(0, "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Tokens(0, "strategy").
		Discard("garrison").
		Pawn(0).
		Age(1).
		Accessible("stone-pit").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "wonder:mausoleum:stone-pit", "revive:garrison")

	var pawnEvents []engine.Event
	for _, e := range ev {
		if e.Kind == engine.EvPawn {
			pawnEvents = append(pawnEvents, e)
		}
	}
	if len(pawnEvents) != 1 {
		t.Fatalf("got %d pawn events, want exactly 1 (from the revived garrison): %+v", len(pawnEvents), pawnEvents)
	}
	if pawnEvents[0].A != 0 || pawnEvents[0].B != 2 {
		t.Fatalf("pawn moved %d -> %d, want 0 -> 2 (1 shield + 1 from Strategy)", pawnEvents[0].A, pawnEvents[0].B)
	}
	if st.Military != 2 {
		t.Fatalf("military = %d, want 2", st.Military)
	}
}

// TestAdvMilSciR1_TakingStrategyTokenGrantsNoImmediateShields checks that
// picking up Strategy is not itself a one-off effect: it must not move the
// pawn, even though the picking player already owns a red building.
// Rule: docs/rules/wonders-tokens.md "A token is kept for the rest of the
// game. One-off effects happen at once; lasting effects apply to everything
// the player does afterwards, never to what they did before."
func TestAdvMilSciR1_TakingStrategyTokenGrantsNoImmediateShields(t *testing.T) {
	st := enginetest.New(t, 109).
		City(0, "stable", "pharmacist", "quarry", "clay-pool", "clay-pit").
		BoardTokens("strategy", "agriculture", "philosophy", "masonry", "mathematics").
		Pawn(1).
		Age(2).
		Accessible("dispensary").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "construct:dispensary", "token:strategy")

	for _, e := range ev {
		if e.Kind == engine.EvPawn {
			t.Fatalf("unexpected pawn movement while completing a pair and picking Strategy: %+v", e)
		}
	}
	if st.Military != 1 {
		t.Fatalf("military = %d, want unchanged 1", st.Military)
	}
	tid, ok := catalog.TokenBySlug("strategy")
	if !ok {
		t.Fatalf("strategy token slug not found")
	}
	if !st.Players[0].Tokens.Has(tid) {
		t.Fatalf("seat0 does not own Strategy after picking it")
	}
}

// TestAdvMilSciR1_StrategyShieldCanTriggerLootingThatWouldNotOtherwiseHappen
// checks that Strategy's extra shield is a real shield for every purpose,
// including crossing a looting boundary that the card's own shield would not
// have reached.
// Rule: docs/rules/military-science.md "Where shields come from" (Strategy)
// and "Looting tokens" (entering space 3 for the first time).
func TestAdvMilSciR1_StrategyShieldCanTriggerLootingThatWouldNotOtherwiseHappen(t *testing.T) {
	st := enginetest.New(t, 110).
		City(0, "lumber-yard").
		Tokens(0, "strategy").
		Pawn(1).
		Coins(1, 7).
		Age(1).
		Accessible("stable").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "construct:stable")

	var loot *engine.Event
	for i, e := range ev {
		if e.Kind == engine.EvLoot {
			loot = &ev[i]
		}
	}
	if loot == nil {
		t.Fatalf("expected a loot event from the Strategy-boosted shield, got none: %+v", ev)
	}
	if loot.Seat != 1 || loot.A != 2 || loot.B != 2 {
		t.Fatalf("loot = %+v, want seat1 losing 2 (token worth 2)", *loot)
	}
	if st.Military != 3 {
		t.Fatalf("military = %d, want 3 (1 + 1 from Strategy)", st.Military)
	}
	if st.Players[1].Coins != 5 {
		t.Fatalf("seat1 coins = %d, want 5 (7 - 2)", st.Players[1].Coins)
	}
}

// TestAdvMilSciR1_LootingCapsAtZeroButEventRecordsFullTokenValue checks the
// A-vs-B distinction on EvLoot: A is the actual, capped loss, B is always the
// token's face value.
// Rule: docs/rules/military-science.md "A player who has fewer coins loses
// all they have and no more."; events.go doc: "EvLoot ... Seat lost A coins
// (the token is worth B)".
func TestAdvMilSciR1_LootingCapsAtZeroButEventRecordsFullTokenValue(t *testing.T) {
	st := enginetest.New(t, 111).
		City(0, "lumber-yard").
		Pawn(2).
		Coins(1, 1).
		Age(1).
		Accessible("stable").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "construct:stable")

	var loot *engine.Event
	for i, e := range ev {
		if e.Kind == engine.EvLoot {
			loot = &ev[i]
		}
	}
	if loot == nil {
		t.Fatalf("expected a loot event, got none: %+v", ev)
	}
	if loot.A != 1 {
		t.Fatalf("loot.A (actual coins lost) = %d, want 1 (capped)", loot.A)
	}
	if loot.B != 2 {
		t.Fatalf("loot.B (token face value) = %d, want 2", loot.B)
	}
	if st.Players[1].Coins != 0 {
		t.Fatalf("seat1 coins = %d, want 0 (never negative)", st.Players[1].Coins)
	}
}

// TestAdvMilSciR1_PairCompletionNeverIncreasesDistinctSymbolCount checks
// that completing a pair (necessarily the second card of a symbol the player
// already had) never changes the count of distinct symbols owned, since a
// repeated symbol cannot be a new one.
// Rule: docs/rules/military-science.md "A card that completes a pair
// repeats a symbol you already had, so it never adds a sixth different
// symbol itself."
func TestAdvMilSciR1_PairCompletionNeverIncreasesDistinctSymbolCount(t *testing.T) {
	st := enginetest.New(t, 112).
		City(0, "apothecary", "lumber-yard", "press", "drying-room").
		Age(2).
		Accessible("school").
		ToMove(0).
		State()

	before := engine.DistinctSymbols(&st.Players[0])

	ev := enginetest.Do(t, &st, "construct:school")

	foundPair := false
	for _, e := range ev {
		if e.Kind == engine.EvPair {
			foundPair = true
		}
	}
	if !foundPair {
		t.Fatalf("expected a pair event when the second wheel card is built, got none: %+v", ev)
	}
	after := engine.DistinctSymbols(&st.Players[0])
	if after != before {
		t.Fatalf("distinct symbols changed from %d to %d on a pair completion", before, after)
	}
	if st.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase = %v, want pickBoardToken (the pair is still forced)", st.Phase)
	}

	keys := enginetest.Keys(&st)
	if len(keys) == 0 {
		t.Fatalf("expected at least one board token on offer")
	}
	enginetest.Do(t, &st, keys[0]) // resolve the forced choice so the state stays valid
}

// TestAdvMilSciR1_CentrePawnGivesStarterChoiceToLastActivePlayer checks the
// default centre-pawn ruling: with the pawn centred, the last active player
// (not a "weaker" player, since neither is weaker) chooses who starts next,
// and may choose either seat.
// Rule: docs/rules/ages-scoring.md "If the pawn is on the centre space,
// neither player is weaker, and the last active player chooses."
// docs/rules/decisions.md "Centred pawn".
func TestAdvMilSciR1_CentrePawnGivesStarterChoiceToLastActivePlayer(t *testing.T) {
	st := enginetest.New(t, 113).
		Age(1).
		OnlyLeft("stone-pit").
		Pawn(0).
		ToMove(0).
		State()

	ev1 := enginetest.Do(t, &st, "discard:stone-pit")

	foundAgeStart := false
	for _, e := range ev1 {
		if e.Kind == engine.EvAgeStart && e.A == 2 {
			foundAgeStart = true
		}
	}
	if !foundAgeStart {
		t.Fatalf("expected age 2 to be dealt before the starter choice: %+v", ev1)
	}
	if st.Phase != engine.PhaseChooseStarter {
		t.Fatalf("phase = %v, want chooseStarter", st.Phase)
	}
	if st.Active != 0 || st.ToMove != 0 {
		t.Fatalf("active=%d toMove=%d, want both 0 (the last active player, centred pawn)", st.Active, st.ToMove)
	}

	keys := enginetest.Keys(&st)
	slices.Sort(keys)
	want := []string{"starter:0", "starter:1"}
	if !slices.Equal(keys, want) {
		t.Fatalf("legal keys = %v, want %v", keys, want)
	}

	ev2 := enginetest.Do(t, &st, "starter:1")
	foundChoice := false
	for _, e := range ev2 {
		if e.Kind == engine.EvStarterChosen && e.Seat == 0 && e.A == 1 {
			foundChoice = true
		}
	}
	if !foundChoice {
		t.Fatalf("expected seat0 to choose seat1 to start: %+v", ev2)
	}
	if st.Age != 2 || st.ToMove != 1 || st.Active != 1 || st.Phase != engine.PhasePlay {
		t.Fatalf("after the choice: age=%d toMove=%d active=%d phase=%v", st.Age, st.ToMove, st.Active, st.Phase)
	}
}

// TestAdvMilSciR1_OffCentrePawnGivesStarterChoiceToWeakerPlayerNotLastActive
// checks that off-centre the choice goes strictly by military weakness, not
// by who happened to take the age's last card: here the last active player
// is the stronger one, and the choice must still go to the weaker opponent.
// Rule: docs/rules/ages-scoring.md "The weaker player chooses who starts the
// next age."; docs/rules/turn.md "Two seats can matter at once ... They can
// differ only while a player other than the active one chooses who starts
// the next age."
func TestAdvMilSciR1_OffCentrePawnGivesStarterChoiceToWeakerPlayerNotLastActive(t *testing.T) {
	st := enginetest.New(t, 114).
		Age(1).
		OnlyLeft("stone-pit").
		Pawn(3). // positive: seat0 ahead, seat1 weaker
		ToMove(0).
		State()

	enginetest.Do(t, &st, "discard:stone-pit")

	if st.Phase != engine.PhaseChooseStarter {
		t.Fatalf("phase = %v, want chooseStarter", st.Phase)
	}
	if st.Active != 0 {
		t.Fatalf("active = %d, want 0 (seat0 took the age's last card)", st.Active)
	}
	if st.ToMove != 1 {
		t.Fatalf("toMove = %d, want 1 (seat1 is weaker and must choose)", st.ToMove)
	}

	keys := enginetest.Keys(&st)
	slices.Sort(keys)
	want := []string{"starter:0", "starter:1"}
	if !slices.Equal(keys, want) {
		t.Fatalf("legal keys = %v, want %v (the weaker player may choose either seat)", keys, want)
	}

	enginetest.Do(t, &st, "starter:0")
	if st.ToMove != 0 || st.Active != 0 || st.Phase != engine.PhasePlay {
		t.Fatalf("after seat1 chose seat0: toMove=%d active=%d phase=%v", st.ToMove, st.Active, st.Phase)
	}
}

// TestAdvMilSciR1_GreatLibraryDrawIsPrivateAndLawGrantsScales checks two
// things at once: the three drawn tokens are reported only through private
// events in their drawn order, and taking Law immediately counts as the
// scales symbol.
// Rule: docs/rules/wonders-tokens.md "For developers": "The Great Library
// draw is not random at play time ... The offer appears only in the
// builder's view, and the event announcing the draw is filtered the same
// way."; events.go: "EvLibraryDraw // Private to Seat ... A = its place in
// the offer (0 to 2)"; docs/rules/military-science.md "For developers":
// "Science is kept as a count per symbol ... with Law adding the scales."
func TestAdvMilSciR1_GreatLibraryDrawIsPrivateAndLawGrantsScales(t *testing.T) {
	st := enginetest.New(t, 115).
		Wonders(0, "great-library").
		City(0, "lumber-yard", "logging-camp", "sawmill", "glassworks", "press").
		BoardTokens("architecture", "economy", "masonry", "mathematics", "philosophy").
		BoxOrder("strategy", "law", "agriculture").
		Pawn(0).
		Age(1).
		Accessible("stone-pit").
		ToMove(0).
		State()

	before := engine.DistinctSymbols(&st.Players[0])

	ev := enginetest.Do(t, &st, "wonder:great-library:stone-pit")

	var draws []engine.Event
	for _, e := range ev {
		if e.Kind == engine.EvLibraryDraw {
			draws = append(draws, e)
		}
	}
	if len(draws) != 3 {
		t.Fatalf("got %d library-draw events, want 3: %+v", len(draws), draws)
	}
	byPlace := map[int16]engine.Event{}
	for _, d := range draws {
		byPlace[d.A] = d
	}
	wantTokens := []string{"strategy", "law", "agriculture"}
	for i, slug := range wantTokens {
		d, ok := byPlace[int16(i)]
		if !ok {
			t.Fatalf("missing library-draw event for place %d", i)
		}
		if !d.Private || d.Seat != 0 {
			t.Fatalf("draw at place %d = %+v, want private to seat0", i, d)
		}
		id, ok := catalog.TokenBySlug(slug)
		if !ok || d.Token != id {
			t.Fatalf("draw at place %d token = %v, want %s", i, d.Token, slug)
		}
		if st.Library[i] != id {
			t.Fatalf("Library[%d] = %v, want %s", i, st.Library[i], slug)
		}
	}
	if st.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase = %v, want pickLibraryToken", st.Phase)
	}

	keys := enginetest.Keys(&st)
	slices.Sort(keys)
	wantKeys := []string{"token:agriculture", "token:law", "token:strategy"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("legal keys = %v, want %v", keys, wantKeys)
	}

	ev2 := enginetest.Do(t, &st, "token:law")
	foundToken := false
	for _, e := range ev2 {
		if e.Kind == engine.EvToken {
			foundToken = true
			if e.Private {
				t.Fatalf("the kept token's own event must be public, got private: %+v", e)
			}
			if e.A != 1 {
				t.Fatalf("EvToken.A = %d, want 1 (came from the Great Library)", e.A)
			}
		}
	}
	if !foundToken {
		t.Fatalf("expected an EvToken event for the kept token: %+v", ev2)
	}
	after := engine.DistinctSymbols(&st.Players[0])
	if after != before+1 {
		t.Fatalf("distinct symbols went from %d to %d, want +1 for the scales", before, after)
	}
}

// TestAdvMilSciR1_MilitarySupremacyMidWonderSkipsStatueOfZeusDestroyChoice
// checks that when the Statue of Zeus's own shield carries the pawn into the
// capital, the game ends immediately and the destroy choice never opens,
// even though the opponent does own a brown card (so the choice is skipped
// by the supremacy, not for lack of a target). The wonder itself must still
// be recorded as built, since payment and building happen before the shield.
// Rule: docs/rules/decisions.md "Supremacy timing": "if the Statue of
// Zeus's shield carries the pawn into the capital, the game ends before its
// builder chooses a card to destroy."; docs/rules/wonders-tokens.md "The
// order of a wonder's effects".
func TestAdvMilSciR1_MilitarySupremacyMidWonderSkipsStatueOfZeusDestroyChoice(t *testing.T) {
	loggingCamp, ok := catalog.CardBySlug("logging-camp")
	if !ok {
		t.Fatalf("logging-camp slug not found")
	}
	wid, ok := catalog.WonderBySlug("statue-of-zeus")
	if !ok {
		t.Fatalf("statue-of-zeus slug not found")
	}

	st := enginetest.New(t, 116).
		Wonders(0, "statue-of-zeus").
		City(0, "press", "drying-room", "clay-pool", "lumber-yard", "quarry").
		City(1, "logging-camp").
		Pawn(8).
		Age(1).
		Accessible("stone-pit").
		ToMove(0).
		State()

	enginetest.Do(t, &st, "wonder:statue-of-zeus:stone-pit")

	if !st.Over() {
		t.Fatalf("game should be over, military = %d", st.Military)
	}
	if st.Phase != engine.PhaseGameOver {
		t.Fatalf("phase = %v, want gameOver (the destroy choice must never open)", st.Phase)
	}
	if st.Result.Kind != engine.VictoryMilitary || st.Result.Winner != 0 {
		t.Fatalf("result = %+v, want military victory for seat0", st.Result)
	}
	built := false
	for _, w := range st.Players[0].Wonders {
		if w.ID == wid && w.Built {
			built = true
		}
	}
	if !built {
		t.Fatalf("Statue of Zeus should still be recorded as built, even though the choice was skipped")
	}
	if !st.Players[1].City.Has(loggingCamp) {
		t.Fatalf("seat1's brown card should be untouched, since the destroy choice never happened")
	}
}

// TestAdvMilSciR1_MilitarySupremacyOnLastCardSkipsNewAgeEntirely checks that
// a supremacy on the age's last card ends the game at once, with no new age
// dealt and no starter choice opened.
// Rule: docs/rules/ages-scoring.md "The end of an age": "If a supremacy
// happens on the last card of an age, the game is over at once and no new
// age begins."
func TestAdvMilSciR1_MilitarySupremacyOnLastCardSkipsNewAgeEntirely(t *testing.T) {
	st := enginetest.New(t, 117).
		Wonders(0, "statue-of-zeus").
		City(0, "press", "drying-room", "clay-pool", "lumber-yard", "quarry").
		Pawn(8).
		Age(1).
		OnlyLeft("stone-pit").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "wonder:statue-of-zeus:stone-pit")

	if st.Phase != engine.PhaseGameOver || st.Age != 1 {
		t.Fatalf("phase=%v age=%d, want gameOver and age unchanged at 1", st.Phase, st.Age)
	}
	for _, e := range ev {
		if e.Kind == engine.EvAgeStart {
			t.Fatalf("no new age should start: %+v", e)
		}
		if e.Kind == engine.EvStarterChosen {
			t.Fatalf("no starter choice should ever open: %+v", e)
		}
	}
}

// TestAdvMilSciR1_ShieldsOvershootingCapitalClampToNine checks that shields
// carrying the pawn past the capital land exactly on space 9, not beyond.
// Rule: docs/rules/military-science.md track diagram (space 9 is the
// capital; "reaching it wins the game at once"); engine.State doc: "Military
// int8 // conflict pawn, -9 to 9".
func TestAdvMilSciR1_ShieldsOvershootingCapitalClampToNine(t *testing.T) {
	st := enginetest.New(t, 118).
		City(0, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Pawn(7).
		Age(3).
		Accessible("arsenal").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "construct:arsenal")

	var pawnEv *engine.Event
	for i, e := range ev {
		if e.Kind == engine.EvPawn {
			pawnEv = &ev[i]
		}
	}
	if pawnEv == nil || pawnEv.B != 9 {
		t.Fatalf("pawn event = %+v, want to land exactly on 9", pawnEv)
	}
	if st.Military != 9 {
		t.Fatalf("military = %d, want 9 (clamped, not 10)", st.Military)
	}
	if st.Result.Kind != engine.VictoryMilitary || st.Result.Winner != 0 {
		t.Fatalf("result = %+v, want military victory for seat0", st.Result)
	}
}

// TestAdvMilSciR1_EconomyCreditArrivesBeforeLootingDebit checks the ordering
// of two effects of one action that both move coins for the opponent: the
// Economy credit for the trading payment is applied before the looting debit
// that the same action's shield triggers. Seat1 starts at 0 coins, so the
// two possible orders give different final totals (1 if credited first, 3 if
// debited first), which makes this a sharp, non-vacuous check.
// Rule: docs/rules/decisions.md "Effect order": "an opponent who owns
// Economy receives the trading coins before a looting token ... takes coins
// from them."
func TestAdvMilSciR1_EconomyCreditArrivesBeforeLootingDebit(t *testing.T) {
	st := enginetest.New(t, 119).
		City(1, "clay-pool").
		Tokens(1, "economy").
		Coins(0, 5).
		Coins(1, 0).
		Pawn(2).
		Age(1).
		Accessible("garrison").
		ToMove(0).
		State()

	enginetest.Do(t, &st, "construct:garrison")

	if st.Players[1].Coins != 1 {
		t.Fatalf("seat1 coins = %d, want 1 (received 3 via Economy, then lost 2 to looting)", st.Players[1].Coins)
	}
	if st.Players[0].Coins != 2 {
		t.Fatalf("seat0 coins = %d, want 2 (5 - 3 traded)", st.Players[0].Coins)
	}
	if st.Military != 3 {
		t.Fatalf("military = %d, want 3", st.Military)
	}
}

// TestAdvMilSciR1_StatueOfZeusSkippedWhenOpponentHasNoBrownCard checks that
// the destroy choice is skipped (not offered against a card of the wrong
// colour) when the opponent's only building is grey, and that this skip is
// reported through EvEffectSkipped.
// Rule: docs/rules/wonders-tokens.md "The Statue of Zeus and Circus Maximus":
// "If the opponent has no card of that colour, nothing is destroyed.";
// docs/rules/decisions.md "Forced choices": "When a choice has no option, it
// is skipped and the wonder is still built."
func TestAdvMilSciR1_StatueOfZeusSkippedWhenOpponentHasNoBrownCard(t *testing.T) {
	glassworks, ok := catalog.CardBySlug("glassworks")
	if !ok {
		t.Fatalf("glassworks slug not found")
	}
	st := enginetest.New(t, 120).
		Wonders(0, "statue-of-zeus").
		City(0, "press", "drying-room", "clay-pool", "lumber-yard", "quarry").
		City(1, "glassworks").
		Pawn(0).
		Age(1).
		Accessible("stone-pit").
		ToMove(0).
		State()

	ev := enginetest.Do(t, &st, "wonder:statue-of-zeus:stone-pit")

	foundSkip := false
	for _, e := range ev {
		if e.Kind == engine.EvEffectSkipped && engine.Phase(e.A) == engine.PhaseDestroyCard {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Fatalf("expected the destroy choice to be reported as skipped: %+v", ev)
	}
	if st.Phase == engine.PhaseDestroyCard {
		t.Fatalf("the destroy phase must never open when the opponent has no brown card")
	}
	if !st.Players[1].City.Has(glassworks) {
		t.Fatalf("seat1's grey card should be untouched")
	}
	if st.Military != 1 {
		t.Fatalf("military = %d, want 1 (the shield still applies)", st.Military)
	}
}

// TestAdvMilSciR1_CircusMaximusOnlyOffersGreyTargetsNotBrown checks that the
// destroy choice is scoped to the wonder's own colour even when the opponent
// owns a card of the other colour too: only the grey card may be targeted.
// Rule: docs/rules/wonders-tokens.md "Circus Maximus: 1 shield. Then destroy
// one grey card in your opponent's city."
func TestAdvMilSciR1_CircusMaximusOnlyOffersGreyTargetsNotBrown(t *testing.T) {
	glassworks, ok := catalog.CardBySlug("glassworks")
	if !ok {
		t.Fatalf("glassworks slug not found")
	}
	clayPool, ok := catalog.CardBySlug("clay-pool")
	if !ok {
		t.Fatalf("clay-pool slug not found")
	}

	st := enginetest.New(t, 121).
		Wonders(0, "circus-maximus").
		City(0, "quarry", "stone-pit", "lumber-yard", "glass-blower").
		City(1, "clay-pool", "glassworks").
		Pawn(0).
		Age(1).
		Accessible("baths").
		ToMove(0).
		State()

	enginetest.Do(t, &st, "wonder:circus-maximus:baths")

	if st.Phase != engine.PhaseDestroyCard {
		t.Fatalf("phase = %v, want destroyCard", st.Phase)
	}
	if st.Destroy != catalog.Grey {
		t.Fatalf("destroy colour = %v, want grey", st.Destroy)
	}
	keys := enginetest.Keys(&st)
	slices.Sort(keys)
	want := []string{"destroy:glassworks"}
	if !slices.Equal(keys, want) {
		t.Fatalf("legal keys = %v, want %v (only the grey card)", keys, want)
	}

	enginetest.Do(t, &st, "destroy:glassworks")

	if st.Players[1].City.Has(glassworks) {
		t.Fatalf("glassworks should have been destroyed")
	}
	if !st.Discard.Has(glassworks) {
		t.Fatalf("the destroyed card should sit on the discard pile")
	}
	if !st.Players[1].City.Has(clayPool) {
		t.Fatalf("the brown card must be untouched")
	}
}

// TestAdvMilSciR1_NegativeSideShieldsAlsoClampAtMinusNine is the mirror of
// the positive-side overshoot test: the track is symmetric, so the clamp at
// seat0's capital (-9) must behave exactly like the clamp at seat1's capital
// (+9), guarding against a sign-only bug.
// Rule: docs/rules/military-science.md track diagram; engine.State doc.
func TestAdvMilSciR1_NegativeSideShieldsAlsoClampAtMinusNine(t *testing.T) {
	st := enginetest.New(t, 122).
		City(1, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Pawn(-7).
		Age(3).
		Accessible("arsenal").
		ToMove(1).
		State()

	ev := enginetest.Do(t, &st, "construct:arsenal")

	var pawnEv *engine.Event
	for i, e := range ev {
		if e.Kind == engine.EvPawn {
			pawnEv = &ev[i]
		}
	}
	if pawnEv == nil || pawnEv.B != -9 {
		t.Fatalf("pawn event = %+v, want to land exactly on -9", pawnEv)
	}
	if st.Military != -9 {
		t.Fatalf("military = %d, want -9 (clamped, not -10)", st.Military)
	}
	if st.Result.Kind != engine.VictoryMilitary || st.Result.Winner != 1 {
		t.Fatalf("result = %+v, want military victory for seat1", st.Result)
	}
}
