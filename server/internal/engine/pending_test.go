package engine_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Tests of the pending choices that wonders open (destroy a card, build from
// the discard pile, the Great Library pick) and of the board token choice
// when a rebuilt card opens it: how each choice is entered, which seat
// decides, how the turn goes on once it is made (to the opponent, to an
// extra turn, to the end of an age and its starter choice under both
// centre-pawn options, to the end of the game), when the cards the turn
// freed are revealed, and the victories that come through these choices.
// The last test checks that the property sweep reaches every pending phase.
// Helpers come from wonders_test.go (wd).

func TestPendingRevealWaitsForTheChoice(t *testing.T) {
	// Taking the tuck card from the bottom row of Age II frees a face-down
	// card. It turns over only once the Mausoleum's choice is made.
	s := wdNew(t).Wonders(0, "mausoleum").Discard("baths").State()
	acc := engine.Accessible(&s)
	ev := wdBuild(t, &s, "mausoleum")
	if r := kindsOf(ev, engine.EvReveal); len(r) != 0 {
		t.Fatalf("revealed before the choice: %+v", r)
	}
	freed := engine.Accessible(&s) &^ acc
	if freed == 0 {
		t.Fatal("the tuck card freed no card")
	}
	for i := range s.Slots {
		if freed&(1<<i) != 0 && s.Slots[i].FaceUp {
			t.Fatalf("slot %d is face up during the choice", i)
		}
	}
	ev = enginetest.Do(t, &s, "revive baths")
	if len(ev) < 2 || ev[0].Kind != engine.EvRebuild || ev[len(ev)-1].Kind != engine.EvReveal {
		t.Fatalf("events %+v: want the rebuild first and the reveal last", ev)
	}
	for _, e := range kindsOf(ev, engine.EvReveal) {
		if freed&(1<<e.A) == 0 || !s.Slots[e.A].FaceUp {
			t.Fatalf("reveal %+v of a slot the turn did not free", e)
		}
	}
}

func TestPendingRebuiltCardOpensAPair(t *testing.T) {
	// The Mausoleum rebuilds the Laboratory next to the Workshop: the pair
	// opens the board token choice, still in the builder's turn, and the
	// turn ends after the token.
	s := wdNew(t).Wonders(0, "mausoleum").City(0, "workshop").Discard("laboratory").
		BoardTokens("agriculture", "economy", "law", "philosophy", "theology").State()
	wdBuild(t, &s, "mausoleum")
	ev := enginetest.Do(t, &s, "revive laboratory")
	wdEvents(t, "the rebuild", ev, []engine.Event{
		{Kind: engine.EvRebuild, Seat: 0, Card: cardID(t, "laboratory")},
		{Kind: engine.EvPair, Seat: 0, A: int16(catalog.SetSquare)},
	})
	if s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("phase %s, seat %d to move", s.Phase, s.ToMove)
	}
	if got := enginetest.Keys(&s); len(got) != 5 || got[0] != "token:agriculture" {
		t.Fatalf("legal %v", got)
	}
	coins := s.Players[0].Coins
	enginetest.Do(t, &s, "token agriculture")
	if s.Players[0].Coins != coins+6 || s.BoardTokens.Count() != 4 {
		t.Fatalf("coins %d, board %v", s.Players[0].Coins, s.BoardTokens)
	}
	wdTurnPassed(t, &s, 0)
}

func TestPendingDestroyedCardCanBeRebuilt(t *testing.T) {
	// The Statue of Zeus sends the opponent's Quarry to the discard pile;
	// the opponent's Mausoleum then builds it back for free.
	s := wdNew(t).Wonders(0, "statue-of-zeus").Wonders(1, "mausoleum").Coins(1, 40).City(1, "quarry").State()
	wdBuild(t, &s, "statue-of-zeus")
	enginetest.Do(t, &s, "destroy quarry")
	c := s.Slots[firstAccessible(&s)].Card
	enginetest.Do(t, &s, "wonder:mausoleum:"+c.String())
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"revive:quarry"}) {
		t.Fatalf("legal %v", got)
	}
	enginetest.Do(t, &s, "revive quarry")
	if !s.Players[1].City.Has(cardID(t, "quarry")) || !s.Discard.Empty() || s.Players[1].Prod[catalog.Stone] != 1 {
		t.Fatalf("city %v, discard %v", s.Players[1].City, s.Discard)
	}
	wdTurnPassed(t, &s, 1)
}

func TestPendingLibraryShortBox(t *testing.T) {
	// Only in scenario games can the box hold fewer than three tokens. The
	// Great Library offers what there is, and skips an empty box.
	board := []string{"agriculture", "economy", "law", "philosophy", "theology"}
	t.Run("two tokens", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "great-library").BoardTokens(board...).
			Tokens(1, "masonry", "mathematics", "strategy").State()
		arch, urb := wdToken(t, "architecture"), wdToken(t, "urbanism")
		if s.BoxTokens[2] != 0 || s.BoxTokens[0] == 0 || s.BoxTokens[1] == 0 {
			t.Fatalf("the box is %v", s.BoxTokens)
		}
		first, second := s.BoxTokens[0], s.BoxTokens[1]
		ev := wdBuild(t, &s, "great-library")
		if draws := kindsOf(ev, engine.EvLibraryDraw); len(draws) != 2 {
			t.Fatalf("draws %+v", draws)
		}
		if s.Library != [engine.LibraryDraw]catalog.TokenID{first, second} || s.BoxTokens != ([engine.NumBoxTokens]catalog.TokenID{}) {
			t.Fatalf("offer %v, box %v", s.Library, s.BoxTokens)
		}
		if got := enginetest.Keys(&s); !slices.Equal(got, []string{"token:" + first.String(), "token:" + second.String()}) {
			t.Fatalf("legal %v (the box held %s and %s)", got, arch, urb)
		}
		enginetest.Do(t, &s, "token "+second.String())
		if s.BoxTokens != [engine.NumBoxTokens]catalog.TokenID{first} || !s.Players[0].Tokens.Has(second) {
			t.Fatalf("box %v, tokens %v", s.BoxTokens, s.Players[0].Tokens)
		}
		wdTurnPassed(t, &s, 0)
	})
	t.Run("an empty box", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "great-library").BoardTokens(board...).
			Tokens(1, "architecture", "masonry", "mathematics", "strategy", "urbanism").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderGreatLibrary)
		ev := wdBuild(t, &s, "great-library")
		wdEvents(t, "the great library", ev, append(wdPaid(t, 0, "great-library", q),
			engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Wonder: catalog.WonderGreatLibrary, A: int16(engine.PhasePickLibraryToken)}))
		wdTurnPassed(t, &s, 0)
	})
}

func TestPendingChoiceAtTheEndOfAnAge(t *testing.T) {
	// The wonder is built with the age's last card. The age ends only after
	// the choice: an extra turn is lost then, and the next age's starter is
	// chosen by the weaker player, or with the pawn in the centre by the
	// builder (who simply starts under LastActiveStarts).
	cases := []struct {
		name    string
		wonder  string
		age     int
		pawn    int
		literal bool     // Options.CenterPawn = LastActiveStarts
		tokens  []string // the builder's
		step    string   // resolves the choice
		phase   engine.Phase
		toMove  engine.Seat
		lost    bool // an extra turn was lost
	}{
		{"mausoleum, centre", "mausoleum", 2, 0, false, nil, "revive baths", engine.PhaseChooseStarter, 0, false},
		{"mausoleum, centre, literal", "mausoleum", 2, 0, true, nil, "revive baths", engine.PhasePlay, 0, false},
		{"zeus pushes the pawn: the opponent chooses", "statue-of-zeus", 2, 0, false, nil, "destroy quarry", engine.PhaseChooseStarter, 1, false},
		{"zeus pushes the pawn: the opponent chooses (literal)", "statue-of-zeus", 2, 0, true, nil, "destroy quarry", engine.PhaseChooseStarter, 1, false},
		{"library, the builder is weaker, Theology's turn is lost", "great-library", 2, -2, false, []string{"theology"}, "token law", engine.PhaseChooseStarter, 0, true},
		{"circus maximus in Age I", "circus-maximus", 1, 3, false, nil, "destroy glassworks", engine.PhaseChooseStarter, 1, false},
		{"circus maximus ends Age III", "circus-maximus", 3, 0, false, []string{"theology"}, "destroy glassworks", engine.PhaseGameOver, 0, true},
	}
	last := [engine.NumAges + 1]string{1: "altar", 2: wdTuck, 3: "palace"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := engine.Options{}
			if c.literal {
				opts.CenterPawn = engine.LastActiveStarts
			}
			b := enginetest.New(t, 22).Options(opts).Age(c.age).FirstPlayer(1).ToMove(0).Coins(0, 40).
				Wonders(0, c.wonder).OnlyLeft(last[c.age]).Pawn(c.pawn).
				City(1, "quarry", "glassworks").Discard("baths").
				BoardTokens("agriculture", "economy", "masonry", "philosophy", "urbanism").BoxOrder("law")
			if c.tokens != nil {
				b.Tokens(0, c.tokens...)
			}
			s := b.State()
			enginetest.Do(t, &s, "wonder:"+c.wonder+":"+last[c.age])
			if s.Age != uint8(c.age) || s.Phase == engine.PhasePlay || s.ToMove != 0 {
				t.Fatalf("after the wonder: age %d, phase %s, seat %d to move", s.Age, s.Phase, s.ToMove)
			}
			ev := enginetest.Do(t, &s, c.step)
			if got := len(kindsOf(ev, engine.EvExtraTurnLost)) == 1; got != c.lost {
				t.Fatalf("extra turn lost: %v, want %v (events %+v)", got, c.lost, ev)
			}
			if s.Phase != c.phase || s.ExtraTurn {
				t.Fatalf("phase %s (extra turn %v), want %s", s.Phase, s.ExtraTurn, c.phase)
			}
			if c.phase == engine.PhaseGameOver {
				if s.Result.Kind != engine.VictoryCivilian || s.Result.Scores != engine.Scores(&s) {
					t.Fatalf("result %+v", s.Result)
				}
				return
			}
			if s.Age != uint8(c.age+1) || s.ToMove != c.toMove || s.Active != 0 {
				t.Fatalf("age %d, seat %d to move, active %d; want age %d, seat %d to move", s.Age, s.ToMove, s.Active, c.age+1, c.toMove)
			}
			if age := kindsOf(ev, engine.EvAgeStart); len(age) != 1 || ev[len(ev)-1] != age[0] {
				t.Fatalf("events %+v: the new age is not the last event", ev)
			}
		})
	}
}

func TestPendingVictoryScienceFromTheLibrary(t *testing.T) {
	s := wdNew(t).Wonders(0, "great-library").City(0, scienceFive...).
		BoardTokens("agriculture", "economy", "masonry", "philosophy", "theology").BoxOrder("law").State()
	wdBuild(t, &s, "great-library")
	if s.Over() || s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase %s", s.Phase)
	}
	ev := enginetest.Do(t, &s, "token law")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	if s.Library != ([engine.LibraryDraw]catalog.TokenID{}) || s.BoxTokens[4] != 0 || s.BoxTokens[3] == 0 {
		t.Fatalf("the offer %v was not put away before the game ended (box %v)", s.Library, s.BoxTokens)
	}
	if last := ev[len(ev)-1]; last.Kind != engine.EvGameOver || len(kindsOf(ev, engine.EvReveal)) != 0 {
		t.Fatalf("events %+v", ev)
	}
}

func TestPendingVictoryScienceFromARebuild(t *testing.T) {
	// A sixth symbol from the discard pile wins at once.
	s := wdNew(t).Wonders(0, "mausoleum").City(0, scienceFive...).Discard("university", "baths").State()
	wdBuild(t, &s, "mausoleum")
	ev := enginetest.Do(t, &s, "revive university")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	wdEvents(t, "the rebuild", ev, []engine.Event{
		{Kind: engine.EvRebuild, Seat: 0, Card: cardID(t, "university")},
		{Kind: engine.EvGameOver, A: int16(engine.VictoryScience), B: 0},
	})
}

func TestPendingVictoryLawAfterARebuiltPair(t *testing.T) {
	// The rebuilt Laboratory pairs the set square (five symbols still), and
	// Law from the board then makes six.
	s := wdNew(t).Wonders(0, "mausoleum").City(0, scienceFive...).Discard("laboratory").
		BoardTokens("agriculture", "economy", "law", "philosophy", "theology").State()
	wdBuild(t, &s, "mausoleum")
	enginetest.Do(t, &s, "revive laboratory")
	if s.Over() || s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s", s.Phase)
	}
	enginetest.Do(t, &s, "token law")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
}

func TestPendingVictoryMilitaryFromARebuild(t *testing.T) {
	// Seat 1 rebuilds the Garrison with the pawn one space from seat 0's
	// capital.
	s := wdNew(t).ToMove(1).Coins(1, 40).Wonders(1, "mausoleum").Discard("garrison").Pawn(-8).State()
	wdBuild(t, &s, "mausoleum")
	ev := enginetest.Do(t, &s, "revive garrison")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 1 || s.Military != -9 {
		t.Fatalf("phase %s, result %+v, pawn %d", s.Phase, s.Result, s.Military)
	}
	if last := ev[len(ev)-1]; last != (engine.Event{Kind: engine.EvGameOver, A: int16(engine.VictoryMilitary), B: 1}) {
		t.Fatalf("events %+v", ev)
	}
}

func TestPendingVictoryOnTheLastCard(t *testing.T) {
	// The Colossus, built with the age's last card, reaches the capital:
	// the game ends with no new age.
	s := enginetest.New(t, 23).Age(1).FirstPlayer(0).Coins(0, 40).Wonders(0, "colossus").
		OnlyLeft("altar").Pawn(7).State()
	ev := enginetest.Do(t, &s, "wonder colossus altar")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Age != 1 {
		t.Fatalf("phase %s, result %+v, age %d", s.Phase, s.Result, s.Age)
	}
	if len(kindsOf(ev, engine.EvAgeStart)) != 0 {
		t.Fatalf("a new age started: %+v", ev)
	}
}

func TestPendingExtraTurnOnTheLastCard(t *testing.T) {
	// The Sphinx built with the age's last card: the extra turn is lost.
	s := enginetest.New(t, 24).Age(1).FirstPlayer(0).Coins(0, 40).Wonders(0, "sphinx").OnlyLeft("altar").State()
	ev := enginetest.Do(t, &s, "wonder sphinx altar")
	want := []engine.EventKind{engine.EvWonder, engine.EvPay, engine.EvExtraTurn, engine.EvExtraTurnLost, engine.EvAgeStart}
	if got := kinds(ev); !slices.Equal(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	if s.ExtraTurn || s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 {
		t.Fatalf("extra turn %v, age %d, phase %s, seat %d to move", s.ExtraTurn, s.Age, s.Phase, s.ToMove)
	}
}

// TestPendingSweepReachesEveryChoice plays the games of the property sweep
// (TestPropertyRandomGames and TestPropertySimpleGames: the same seeds,
// policies and options) and checks that they reach every pending phase, so
// that the sweep's checks (invariants after every action, legality, illegal
// actions, replay) cover every choice. It also checks what the sweep cannot
// see from the phases alone: every Great Library of a real game offers
// three tokens, and a pending choice always has the builder to move.
func TestPendingSweepReachesEveryChoice(t *testing.T) {
	nRandom, nSimple := sweepSize(t)
	var reached [engine.PhaseGameOver + 1]int
	var built [catalog.NumWonders + 1]int
	skipped := 0
	play := func(seed uint64, pols [2]enginetest.Policy) {
		s := engine.NewGame(seed, sweepOptions(seed))
		legal := make([]engine.Action, 0, engine.MaxLegalActions)
		var ev []engine.Event
		for !s.Over() && !t.Failed() {
			legal = engine.LegalActions(&s, legal[:0])
			a := pols[s.ToMove].Choose(&s, legal)
			ev = ev[:0]
			if err := engine.Apply(&s, a, &ev); err != nil {
				t.Fatalf("seed %d: %s: %v", seed, a, err)
			}
			reached[s.Phase]++
			if a.Type == engine.ActConstructWonder {
				built[a.Wonder]++
				if a.Wonder == catalog.WonderGreatLibrary && !s.Over() &&
					(s.Phase != engine.PhasePickLibraryToken || len(kindsOf(ev, engine.EvLibraryDraw)) != engine.LibraryDraw) {
					t.Fatalf("seed %d: the great library offers %v in phase %s", seed, s.Library, s.Phase)
				}
			}
			switch s.Phase {
			case engine.PhaseDestroyCard, engine.PhaseBuildFromDiscard, engine.PhasePickBoardToken, engine.PhasePickLibraryToken:
				if s.ToMove != s.Active {
					t.Fatalf("seed %d: seat %d decides seat %d's choice", seed, s.ToMove, s.Active)
				}
			}
			skipped += len(kindsOf(ev, engine.EvEffectSkipped))
		}
	}
	for i := range nRandom {
		seed := uint64(1_000 + i)
		pol := enginetest.NewRandomPolicy(seed ^ 0x5eed)
		play(seed, [2]enginetest.Policy{pol, pol})
	}
	for i := range nSimple {
		seed := uint64(50_000 + i)
		play(seed, [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)})
	}
	t.Logf("%d games: destroy %d, build from discard %d, board token %d, library token %d, starter %d; %d choices skipped; wonders built %v",
		nRandom+nSimple, reached[engine.PhaseDestroyCard], reached[engine.PhaseBuildFromDiscard], reached[engine.PhasePickBoardToken],
		reached[engine.PhasePickLibraryToken], reached[engine.PhaseChooseStarter], skipped, built[1:])
	for _, p := range []engine.Phase{engine.PhaseDestroyCard, engine.PhaseBuildFromDiscard, engine.PhasePickBoardToken,
		engine.PhasePickLibraryToken, engine.PhaseChooseStarter} {
		if reached[p] == 0 {
			t.Errorf("the sweep never reaches %s", p)
		}
	}
	for w := 1; w <= catalog.NumWonders; w++ {
		if built[w] == 0 {
			t.Errorf("the sweep never builds %s", catalog.WonderID(w))
		}
	}
}
