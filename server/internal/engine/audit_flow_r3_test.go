package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// This file is round 2 of the M10.R1 rules audit, lens: turn flow and ages.
// It is black-box: every expected value is derived from docs/rules/ and the
// design reference (rules-reference.md), not from the engine's code. It
// avoids the cases of edgecases_flow_r1_test.go and audit_*_r2_test.go and
// aims at interactions of two or three effects around the end of an age,
// pending choices on the last card, the structure's overlap rule checked
// against an independent oracle, and what Legal offers at each moment.
// Helpers are prefixed auditR3Flow.

func auditR3FlowCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("test setup: no card %q", slug)
	}
	return c
}

func auditR3FlowCount(ev []engine.Event, kind engine.EventKind) int {
	n := 0
	for _, e := range ev {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

// auditR3FlowIndex returns the index of the first event of kind, or -1.
func auditR3FlowIndex(ev []engine.Event, kind engine.EventKind) int {
	for i, e := range ev {
		if e.Kind == kind {
			return i
		}
	}
	return -1
}

func auditR3FlowKinds(ev []engine.Event) []string {
	out := make([]string, len(ev))
	for i, e := range ev {
		out[i] = e.Kind.String()
	}
	return out
}

func auditR3FlowSorted(keys []string) []string {
	out := slices.Clone(keys)
	slices.Sort(out)
	return out
}

// auditR3FlowDiscardAny discards the first card that Legal offers to discard.
func auditR3FlowDiscardAny(t *testing.T, s *engine.State) []engine.Event {
	t.Helper()
	for _, k := range enginetest.Keys(s) {
		if strings.HasPrefix(k, "discard:") {
			return enginetest.Do(t, s, k)
		}
	}
	t.Fatalf("no discard key is legal: %v", enginetest.Keys(s))
	return nil
}

// auditR3FlowRows transcribes the three structures of rules-reference.md
// section 2 "Card structures" (the same shapes as the diagrams of setup.md
// "Card structures"): for each age, the half-card columns of each row from
// the top.
var auditR3FlowRows = [3][][]int{
	{{5, 7}, {4, 6, 8}, {3, 5, 7, 9}, {2, 4, 6, 8, 10}, {1, 3, 5, 7, 9, 11}},
	{{1, 3, 5, 7, 9, 11}, {2, 4, 6, 8, 10}, {3, 5, 7, 9}, {4, 6, 8}, {5, 7}},
	{{5, 7}, {4, 6, 8}, {3, 5, 7, 9}, {4, 8}, {3, 5, 7, 9}, {4, 6, 8}, {5, 7}},
}

// auditR3FlowPositions returns each slot's place, after checking that the
// engine's slots are exactly the places of the transcribed structure.
func auditR3FlowPositions(t *testing.T, age uint8) [engine.SlotsPerAge]engine.Pos {
	t.Helper()
	var want, got []engine.Pos
	for r, cols := range auditR3FlowRows[age-1] {
		for _, c := range cols {
			want = append(want, engine.Pos{Row: int8(r + 1), Col: int8(c)})
		}
	}
	var out [engine.SlotsPerAge]engine.Pos
	for i := range engine.SlotsPerAge {
		out[i] = engine.SlotPos(age, i)
		got = append(got, out[i])
	}
	less := func(a, b engine.Pos) int {
		if a.Row != b.Row {
			return int(a.Row) - int(b.Row)
		}
		return int(a.Col) - int(b.Col)
	}
	slices.SortFunc(want, less)
	slices.SortFunc(got, less)
	if !slices.Equal(want, got) {
		t.Fatalf("age %d: the structure's places are %v, want %v (rules-reference.md section 2)", age, got, want)
	}
	return out
}

// auditR3FlowOracle returns the accessible slots by the overlap rule of
// setup.md "Card structures": an occupied place (r, c) is accessible when
// no occupied place is at (r+1, c-1) or (r+1, c+1).
func auditR3FlowOracle(s *engine.State, pos *[engine.SlotsPerAge]engine.Pos) uint32 {
	var acc uint32
	for i := range engine.SlotsPerAge {
		if s.Slots[i].Card == 0 {
			continue
		}
		covered := false
		for j := range engine.SlotsPerAge {
			if s.Slots[j].Card == 0 || pos[j].Row != pos[i].Row+1 {
				continue
			}
			if d := pos[j].Col - pos[i].Col; d == 1 || d == -1 {
				covered = true
			}
		}
		if !covered {
			acc |= 1 << i
		}
	}
	return acc
}

// auditR3FlowSlotAt returns the slot at place (row, col) of the age.
func auditR3FlowSlotAt(t *testing.T, age uint8, row, col int) int {
	t.Helper()
	for i := range engine.SlotsPerAge {
		if p := engine.SlotPos(age, i); int(p.Row) == row && int(p.Col) == col {
			return i
		}
	}
	t.Fatalf("test setup: age %d has no place (%d, %d)", age, row, col)
	return -1
}

// auditR3FlowKeyCard returns the card a play key names, or 0.
func auditR3FlowKeyCard(t *testing.T, key string) catalog.CardID {
	t.Helper()
	a, err := engine.ParseKey(key)
	if err != nil {
		t.Fatalf("legal key %q does not parse: %v", key, err)
	}
	switch a.Type {
	case engine.ActConstructBuilding, engine.ActDiscardBuilding, engine.ActConstructWonder:
		return a.Card
	}
	return 0
}

// TestAuditR3FlowLegalCardsFollowTheOverlapRuleInEveryAge plays whole games
// from NewGame (draft, three ages, starter choices) by discarding in four
// different orders, and checks every play turn against an oracle written
// from the rules, not from the engine's own Accessible:
//
//   - setup.md "Card structures": a card is accessible when no card is left
//     at (r+1, c-1) or (r+1, c+1); odd rows are dealt face up, even rows
//     face down; 12 face up and 8 face down per age; 6, 2 and 2 cards
//     accessible at the start of Ages I, II and III;
//   - turn.md "Taking a card": only accessible cards can be taken, and every
//     accessible card is face up when the turn begins; turn.md "For
//     developers": every accessible card has a discard key, and construct
//     and wonder keys only name accessible cards;
//   - turn.md "The end of a turn" step 2: exactly the face-down cards the
//     turn freed are turned face up (one reveal each), and nothing else;
//     a card dealt face up is never "revealed";
//   - ages-scoring.md "The end of an age": the last card's turn reveals
//     nothing and the next structure is dealt fresh.
//
// The orders are: lowest slot first, highest slot first, deepest row then
// leftmost (which in Age III takes the left row-4 card while the right one
// stays, the gap case of setup.md "Age III"), and a fixed pseudo-random one.
func TestAuditR3FlowLegalCardsFollowTheOverlapRuleInEveryAge(t *testing.T) {
	startAcc := [3]int{6, 2, 2}
	for strategy := range 4 {
		for seed := uint64(301); seed <= 303; seed++ {
			s := engine.NewGame(seed, engine.Options{})
			var pos [engine.SlotsPerAge]engine.Pos
			age := uint8(0)
			lcg := uint32(seed)
			reveals := 0
			for step := 0; !s.Over(); step++ {
				if step > 200 {
					t.Fatalf("strategy %d seed %d: the game did not end", strategy, seed)
				}
				switch s.Phase {
				case engine.PhaseWonderDraft:
					enginetest.Do(t, &s, enginetest.Keys(&s)[0])
					continue
				case engine.PhaseChooseStarter:
					enginetest.Do(t, &s, "starter:0")
					continue
				case engine.PhasePlay:
				default:
					t.Fatalf("strategy %d seed %d: unexpected phase %s in a discard-only game", strategy, seed, s.Phase)
				}
				if s.Age != age {
					age = s.Age
					pos = auditR3FlowPositions(t, age)
					up, down := 0, 0
					for i := range engine.SlotsPerAge {
						if s.Slots[i].Card == 0 {
							t.Fatalf("age %d starts with an empty slot %d", age, i)
						}
						if s.Slots[i].FaceUp {
							up++
						} else {
							down++
						}
						if odd := pos[i].Row%2 == 1; s.Slots[i].FaceUp != odd {
							t.Errorf("age %d start: slot %d at row %d face up %v, want %v", age, i, pos[i].Row, s.Slots[i].FaceUp, odd)
						}
					}
					if up != 12 || down != 8 {
						t.Errorf("age %d start: %d face up, %d face down; want 12 and 8", age, up, down)
					}
					if n := auditR3FlowPopcount(auditR3FlowOracle(&s, &pos)); n != startAcc[age-1] {
						t.Errorf("age %d start: %d accessible by the overlap rule, want %d", age, n, startAcc[age-1])
					}
				}
				acc := auditR3FlowOracle(&s, &pos)
				var accCards, discardCards []catalog.CardID
				for i := range engine.SlotsPerAge {
					switch {
					case acc&(1<<i) != 0:
						accCards = append(accCards, s.Slots[i].Card)
						if !s.Slots[i].FaceUp {
							t.Errorf("age %d step %d: accessible slot %d is face down at the start of a turn", age, step, i)
						}
					case s.Slots[i].Card != 0 && pos[i].Row%2 == 1 && !s.Slots[i].FaceUp:
						t.Errorf("age %d step %d: slot %d on an odd row is face down", age, step, i)
					case s.Slots[i].Card != 0 && pos[i].Row%2 == 0 && s.Slots[i].FaceUp:
						t.Errorf("age %d step %d: covered slot %d on an even row is already face up", age, step, i)
					}
				}
				for _, k := range enginetest.Keys(&s) {
					c := auditR3FlowKeyCard(t, k)
					if c == 0 {
						t.Errorf("age %d step %d: non-play key %s in the play phase", age, step, k)
						continue
					}
					if !slices.Contains(accCards, c) {
						t.Errorf("age %d step %d: legal key %s names a card that the overlap rule says is not accessible", age, step, k)
					}
					if strings.HasPrefix(k, "discard:") {
						discardCards = append(discardCards, c)
					}
				}
				slices.Sort(accCards)
				slices.Sort(discardCards)
				if !slices.Equal(accCards, discardCards) {
					t.Fatalf("age %d step %d: discard keys for %v, accessible by the overlap rule %v", age, step, discardCards, accCards)
				}
				// Pick the slot to take.
				var slots []int
				for i := range engine.SlotsPerAge {
					if acc&(1<<i) != 0 {
						slots = append(slots, i)
					}
				}
				pick := slots[0]
				switch strategy {
				case 1:
					pick = slots[len(slots)-1]
				case 2:
					for _, i := range slots {
						if pos[i].Row > pos[pick].Row || (pos[i].Row == pos[pick].Row && pos[i].Col < pos[pick].Col) {
							pick = i
						}
					}
				case 3:
					lcg = lcg*1664525 + 1013904223
					pick = slots[int(lcg>>16)%len(slots)]
				}
				before := s
				ev := enginetest.Do(t, &s, "discard:"+s.Slots[pick].Card.String())
				if s.Over() || s.Age != before.Age {
					if n := auditR3FlowCount(ev, engine.EvReveal); n != 0 {
						t.Errorf("age %d: the last card's turn revealed %d cards", before.Age, n)
					}
					continue
				}
				after := auditR3FlowOracle(&s, &pos)
				var want, got []int
				for i := range engine.SlotsPerAge {
					if after&(1<<i) != 0 && acc&(1<<i) == 0 && pos[i].Row%2 == 0 {
						want = append(want, i)
					}
				}
				for _, e := range ev {
					if e.Kind != engine.EvReveal {
						continue
					}
					got = append(got, int(e.A))
					reveals++
					if int(e.A) < 0 || int(e.A) >= engine.SlotsPerAge || before.Slots[e.A].Card != e.Card {
						t.Errorf("age %d step %d: reveal event %+v does not name the card that was in that slot", age, step, e)
					}
				}
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Errorf("age %d step %d (took slot %d at %v): revealed slots %v, want the freed face-down slots %v",
						age, step, pick, pos[pick], got, want)
				}
			}
			// Every face-down card must be turned up before it can be taken, so
			// a game played to the end reveals each of the 3 x 8 exactly once.
			if reveals != 24 {
				t.Errorf("strategy %d seed %d: %d reveals in the game, want 24", strategy, seed, reveals)
			}
			if s.Result.Kind != engine.VictoryCivilian || s.Age != 3 {
				t.Errorf("strategy %d seed %d: a discard-only game ended with %+v in age %d, want civilian scoring after Age III",
					strategy, seed, s.Result, s.Age)
			}
		}
	}
}

func auditR3FlowPopcount(m uint32) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// TestAuditR3FlowSingleCoverersOfAgesIIAndIII pins two shapes that
// setup.md "Card structures" calls out. Age II: "The cards at the ends of
// row 1 are each overlapped by a single card of row 2", so once rows 5 to 3
// and the card at (2,2) are gone, (1,1) is accessible while (1,3), still
// overlapped by (2,4), is not. Age III: "The left one alone overlaps the two
// left cards of row 3", so once rows 7 to 5 and the left row-4 card (4,4)
// are gone, (3,3) and (3,5) are accessible while (4,8) still holds (3,7)
// and (3,9). Both are checked through the legal discard keys.
func TestAuditR3FlowSingleCoverersOfAgesIIAndIII(t *testing.T) {
	cases := []struct {
		age        uint8
		takeRows   []int
		thenTake   [2]int
		accessible [][2]int
	}{
		{2, []int{5, 4, 3}, [2]int{2, 2}, [][2]int{{1, 1}, {2, 4}, {2, 6}, {2, 8}, {2, 10}}},
		{3, []int{7, 6, 5}, [2]int{4, 4}, [][2]int{{3, 3}, {3, 5}, {4, 8}}},
	}
	for _, c := range cases {
		s := enginetest.New(t, 310).Age(int(c.age)).State()
		var order [][2]int
		for _, r := range c.takeRows {
			for _, col := range auditR3FlowRows[c.age-1][r-1] {
				order = append(order, [2]int{r, col})
			}
		}
		order = append(order, c.thenTake)
		for _, rc := range order {
			i := auditR3FlowSlotAt(t, c.age, rc[0], rc[1])
			enginetest.Do(t, &s, "discard:"+s.Slots[i].Card.String())
		}
		var want, got []string
		for _, rc := range c.accessible {
			want = append(want, "discard:"+s.Slots[auditR3FlowSlotAt(t, c.age, rc[0], rc[1])].Card.String())
		}
		for _, k := range enginetest.Keys(&s) {
			if strings.HasPrefix(k, "discard:") {
				got = append(got, k)
			}
		}
		if !slices.Equal(auditR3FlowSorted(got), auditR3FlowSorted(want)) {
			t.Errorf("age %d: discard keys %v, want exactly the cards at %v", c.age, got, c.accessible)
		}
	}
}

// TestAuditR3FlowLostExtraTurnDoesNotFollowIntoTheNextAge pins turn.md
// "Extra turns", Lost at the end of an age ("It does not carry over into
// the next age, whose starter is chosen in the normal way") with
// ages-scoring.md "Who starts the next age". The Sphinx's extra turn is
// earned on the last card of Age I and lost. Whoever then starts Age II
// plays one turn and the opponent plays next, in all three ways the next
// age can start: the last active player chooses itself (centred pawn,
// decisions.md "Centred pawn"); the last active player starts directly
// (the LastActiveStarts option); the weaker opponent (pawn on its side)
// chooses the last active player.
func TestAuditR3FlowLostExtraTurnDoesNotFollowIntoTheNextAge(t *testing.T) {
	cases := []struct {
		name    string
		opts    engine.Options
		pawn    int
		chooser engine.Seat
		choice  bool
	}{
		{"centred, chooses itself", engine.Options{}, 0, 0, true},
		{"centred, LastActiveStarts", engine.Options{CenterPawn: engine.LastActiveStarts}, 0, 0, false},
		{"weaker opponent chooses it", engine.Options{}, 2, 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := enginetest.New(t, 311).Options(c.opts).Age(1).OnlyLeft("altar").
				Wonders(0, "sphinx").Coins(0, 30).Pawn(c.pawn).ToMove(0).State()
			ev := enginetest.Do(t, &s, "wonder:sphinx:altar")
			if auditR3FlowCount(ev, engine.EvExtraTurnLost) != 1 || s.ExtraTurn {
				t.Fatalf("the Sphinx's extra turn on the last card was not lost: events %v, extra turn %v", auditR3FlowKinds(ev), s.ExtraTurn)
			}
			if s.Age != 2 {
				t.Fatalf("age %d, want Age II", s.Age)
			}
			if c.choice {
				if s.Phase != engine.PhaseChooseStarter || s.ToMove != c.chooser {
					t.Fatalf("phase %s, to move %d; want seat %d choosing", s.Phase, s.ToMove, c.chooser)
				}
				enginetest.Do(t, &s, "starter:0")
			}
			if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
				t.Fatalf("Age II start: phase %s, to move %d, active %d; want seat 0 to play", s.Phase, s.ToMove, s.Active)
			}
			auditR3FlowDiscardAny(t, &s)
			if s.ToMove != 1 || s.Active != 1 {
				t.Errorf("after seat 0's first Age II turn: to move %d, active %d; want seat 1 (the lost extra turn came back)", s.ToMove, s.Active)
			}
		})
	}
}

// TestAuditR3FlowZeusOnTheLastCardHoldsTheAgeThenTheNewWeakerPlayerChooses
// combines three rules on the last card of Age I. Seat 1, owning Theology,
// builds the Statue of Zeus: its shield moves the pawn from the centre to
// seat 0's side (military-science.md; wonders-tokens.md), Theology grants
// an extra turn (turn.md "Extra turns"), and the destroy choice opens. The
// age must stay open until the choice is made (ages-scoring.md "The end of
// an age": "including any pending choices it opened"; turn.md "The end of
// a turn", step 1 before step 3). After the destroy the extra turn is lost
// and Age II is dealt, and seat 0, now the weaker player although seat 1
// was last active, chooses (ages-scoring.md "Who starts the next age").
func TestAuditR3FlowZeusOnTheLastCardHoldsTheAgeThenTheNewWeakerPlayerChooses(t *testing.T) {
	s := enginetest.New(t, 312).Age(1).OnlyLeft("altar").
		Wonders(1, "statue-of-zeus").Tokens(1, "theology").Coins(1, 30).
		City(0, "quarry").ToMove(1).State()
	ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:altar")
	if s.Military != -1 {
		t.Fatalf("pawn %d, want -1 (one shield from seat 1 toward seat 0's capital)", s.Military)
	}
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 1 || s.Age != 1 {
		t.Fatalf("after Zeus on the last card: phase %s, to move %d, age %d; want seat 1's destroy choice in Age I", s.Phase, s.ToMove, s.Age)
	}
	if auditR3FlowCount(ev, engine.EvAgeStart) != 0 || auditR3FlowCount(ev, engine.EvExtraTurnLost) != 0 {
		t.Fatalf("the age ended before the destroy choice: events %v", auditR3FlowKinds(ev))
	}
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"destroy:quarry"}) {
		t.Fatalf("destroy choice offers %v, want only destroy:quarry", got)
	}
	ev = enginetest.Do(t, &s, "destroy:quarry")
	d, lost, age := auditR3FlowIndex(ev, engine.EvDestroy), auditR3FlowIndex(ev, engine.EvExtraTurnLost), auditR3FlowIndex(ev, engine.EvAgeStart)
	if d < 0 || lost < d || age < d {
		t.Errorf("events %v; want the destroy, then the lost extra turn and the new age", auditR3FlowKinds(ev))
	}
	if !s.Discard.Has(auditR3FlowCard(t, "quarry")) {
		t.Errorf("the destroyed quarry is not on the discard pile")
	}
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 || s.Active != 1 {
		t.Errorf("age %d, phase %s, to move %d, active %d; want Age II with seat 0 (weaker) choosing and seat 1 last active",
			s.Age, s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR3FlowMausoleumRevivesTheCardZeusDestroyed pins
// rules-reference.md section 8 item 8 (cards destroyed by Zeus or Circus
// Maximus go to the discard pile, so the Mausoleum can rebuild them) and
// wonders-tokens.md, with the extra turns of Theology (turn.md "Extra
// turns", Chaining) linking the two wonders in one sequence of turns: Zeus
// destroys seat 1's Quarry, seat 0 plays again and builds the Mausoleum,
// whose only offer is that Quarry, which enters seat 0's city (turn.md
// "Every building exists only once": it leaves the pile), and seat 0 plays
// a third time before the move passes.
func TestAuditR3FlowMausoleumRevivesTheCardZeusDestroyed(t *testing.T) {
	s := enginetest.New(t, 313).Age(1).Accessible("altar", "baths", "theater").
		Wonders(0, "statue-of-zeus", "mausoleum").Tokens(0, "theology").Coins(0, 60).
		City(1, "quarry").ToMove(0).State()
	quarry := auditR3FlowCard(t, "quarry")
	enginetest.Do(t, &s, "wonder:statue-of-zeus:altar", "destroy:quarry")
	if !s.Discard.Has(quarry) || s.Players[1].City.Has(quarry) {
		t.Fatalf("after the destroy: quarry on the pile %v, in seat 1's city %v", s.Discard.Has(quarry), s.Players[1].City.Has(quarry))
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("after Zeus with Theology: phase %s, to move %d; want seat 0 to play again", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "wonder:mausoleum:baths")
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"revive:quarry"}) {
		t.Fatalf("Mausoleum offers %v, want only revive:quarry", got)
	}
	enginetest.Do(t, &s, "revive:quarry")
	if !s.Players[0].City.Has(quarry) || s.Players[1].City.Has(quarry) || s.Discard.Has(quarry) {
		t.Fatalf("after the revive: seat 0 has quarry %v, seat 1 %v, pile %v; want it only in seat 0's city",
			s.Players[0].City.Has(quarry), s.Players[1].City.Has(quarry), s.Discard.Has(quarry))
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("after the Mausoleum with Theology: phase %s, to move %d, active %d; want seat 0 again", s.Phase, s.ToMove, s.Active)
	}
	enginetest.Do(t, &s, "discard:theater")
	if s.ToMove != 1 {
		t.Errorf("after three turns of seat 0, seat %d is to move; want seat 1", s.ToMove)
	}
}

// TestAuditR3FlowRevivedRedCardWinsOnTheLastCardOfAgeII pins
// decisions.md "Supremacy timing" ("A supremacy on the last card of an age
// ends the game with no age change") for a supremacy reached inside a
// pending choice: the Mausoleum, built with the last card of Age II by a
// Theology owner, rebuilds Walls (2 shields) with the pawn at 7, which
// reaches seat 1's capital. The game ends in Age II with a military victory:
// no Age III is dealt, the Theology extra turn is neither played nor
// reported lost, and the game-over event is the last one.
func TestAuditR3FlowRevivedRedCardWinsOnTheLastCardOfAgeII(t *testing.T) {
	s := enginetest.New(t, 314).Age(2).OnlyLeft("rostrum").
		Wonders(0, "mausoleum").Tokens(0, "theology").Coins(0, 30).
		Discard("walls").Pawn(7).ToMove(0).State()
	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard || s.Age != 2 {
		t.Fatalf("after the Mausoleum on the last card: phase %s, age %d", s.Phase, s.Age)
	}
	ev := enginetest.Do(t, &s, "revive:walls")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("result %+v, pawn %d; want seat 0's military victory at 9", s.Result, s.Military)
	}
	if s.Age != 2 || auditR3FlowCount(ev, engine.EvAgeStart) != 0 || auditR3FlowCount(ev, engine.EvExtraTurnLost) != 0 {
		t.Errorf("age %d, events %v; want the game to end in Age II with no new age and no turn bookkeeping", s.Age, auditR3FlowKinds(ev))
	}
	if len(ev) == 0 || ev[len(ev)-1].Kind != engine.EvGameOver {
		t.Errorf("events %v; want the game-over event last", auditR3FlowKinds(ev))
	}
}

// TestAuditR3FlowZeusOnTheLastCardOfAgeIIIScoresAfterTheDestroy pins
// ages-scoring.md "The end of the game" (civilian scoring after the last
// card of Age III "and that turn has finished, including its pending
// choices") with three effects of one wonder in order (decisions.md
// "Effect order"): the Statue of Zeus's shield moves the pawn from 2 to 3,
// taking seat 1's 2-coin looting token (military-science.md); the destroy
// choice removes seat 1's Quarry; only then is the game scored. Expected,
// from ages-scoring.md "The eight scoring categories" and "Guilds":
//
//   - seat 0: Zeus 3 points; 13 coins minus Zeus's price (2 papyrus at 2,
//     clay and stone at 3 each because seat 1 produces one of each, wood
//     its own; economy.md) leaves 3 coins, 1 point; the pawn at 3 gives 5
//     military points. Total 9.
//   - seat 1: the Shipowners Guild counts the city with the most brown and
//     grey cards at the end: seat 1 has Clay Pool and Glassworks left (2),
//     seat 0 has Lumber Yard (1), so 2 points; 9 coins minus the 2 looted
//     leaves 7, 2 points. Total 4.
func TestAuditR3FlowZeusOnTheLastCardOfAgeIIIScoresAfterTheDestroy(t *testing.T) {
	s := enginetest.New(t, 315).Age(3).OnlyLeft("palace").
		Wonders(0, "statue-of-zeus").Coins(0, 13).City(0, "lumber-yard").
		City(1, "quarry", "clay-pool", "glassworks", "shipowners-guild").Coins(1, 9).
		Pawn(2).ToMove(0).State()
	enginetest.Do(t, &s, "wonder:statue-of-zeus:palace")
	if s.Over() || s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("after Zeus on the last card of Age III: phase %s, result %+v; want the destroy choice first", s.Phase, s.Result)
	}
	if got := auditR3FlowSorted(enginetest.Keys(&s)); !slices.Equal(got, []string{"destroy:clay-pool", "destroy:quarry"}) {
		t.Fatalf("destroy choice %v, want seat 1's two brown cards", got)
	}
	if s.Military != 3 || s.Players[1].Coins != 7 || s.Players[0].Coins != 3 {
		t.Fatalf("pawn %d, coins %d and %d; want pawn 3, seat 0 at 3 coins, seat 1 at 7 after the looting", s.Military, s.Players[0].Coins, s.Players[1].Coins)
	}
	enginetest.Do(t, &s, "destroy:quarry")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != 0 {
		t.Fatalf("result %+v; want civilian scoring won by seat 0", s.Result)
	}
	want := [2]engine.Score{
		{Wonders: 3, Coins: 1, Military: 5, Total: 9},
		{Purple: 2, Coins: 2, Total: 4},
	}
	if s.Result.Scores != want {
		t.Errorf("scores %+v, want %+v", s.Result.Scores, want)
	}
}

// TestAuditR3FlowTheologyChainsAcrossAPlayAgainWonderWithoutDoubling pins
// turn.md "Extra turns": with Theology every wonder grants an extra turn,
// a wonder never grants two (Never doubled; rules-reference.md section 8
// item 3), and extra turns chain (Chaining; decisions.md "Extra turns in a
// row"). Seat 0 builds the Sphinx (its own extra turn plus Theology: one),
// then in that extra turn the Pyramids (Theology's only), then discards:
// three turns in a row, and then seat 1 moves.
func TestAuditR3FlowTheologyChainsAcrossAPlayAgainWonderWithoutDoubling(t *testing.T) {
	s := enginetest.New(t, 316).Age(1).Accessible("altar", "baths", "theater").
		Wonders(0, "sphinx", "pyramids").Tokens(0, "theology").Coins(0, 60).ToMove(0).State()
	for _, step := range []string{"wonder:sphinx:altar", "wonder:pyramids:baths"} {
		ev := enginetest.Do(t, &s, step)
		if n := auditR3FlowCount(ev, engine.EvExtraTurn); n != 1 {
			t.Errorf("%s: %d extra-turn events, want exactly 1", step, n)
		}
		if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
			t.Fatalf("after %s: phase %s, to move %d, active %d; want seat 0 to play again", step, s.Phase, s.ToMove, s.Active)
		}
	}
	enginetest.Do(t, &s, "discard:theater")
	if s.ToMove != 1 || s.Active != 1 {
		t.Errorf("after the chain: to move %d, active %d; want seat 1 (no doubled extra turn)", s.ToMove, s.Active)
	}
}

// TestAuditR3FlowHiddenAndOutOfStructureCardsAreNeverLegal pins turn.md
// "Taking a card" (only an accessible card of the structure can be taken)
// and "For developers" (the server accepts only keys from the legal set),
// for cards a client could name but must never reach: the cards removed
// unseen from this age and an earlier one, a card of the next age's deck, a
// guild left in the box (setup.md "What is hidden from both players"), the
// card tucked under a wonder (turn.md "3. Build a wonder": it is in no
// city and not on the pile), a card on the discard pile, and a covered
// face-down card. A wonder key is legal only for the mover's own unbuilt
// wonder: not a built one, not the opponent's, not one in the box.
func TestAuditR3FlowHiddenAndOutOfStructureCardsAreNeverLegal(t *testing.T) {
	s := enginetest.New(t, 317).Age(2).
		Wonders(0, "pyramids", "sphinx").Built(0, "sphinx", "baths").Wonders(1, "colossus").
		Coins(0, 99).Discard("altar").ToMove(0).State()
	nonzero := func(cs []catalog.CardID) catalog.CardID {
		for _, c := range cs {
			if c != 0 {
				return c
			}
		}
		t.Fatalf("test setup: no card in %v", cs)
		return 0
	}
	var covered catalog.CardID
	for i := range engine.SlotsPerAge {
		if engine.SlotPos(2, i).Row == 2 && engine.Accessible(&s)&(1<<i) == 0 && !s.Slots[i].FaceUp {
			covered = s.Slots[i].Card
			break
		}
	}
	if covered == 0 {
		t.Fatalf("test setup: no covered face-down card in Age II")
	}
	hidden := map[string]catalog.CardID{
		"removed age II":    nonzero(s.Removed[1][:]),
		"removed age I":     nonzero(s.Removed[0][:]),
		"age III deck":      nonzero(s.Deals[2][:]),
		"box guild":         nonzero(s.BoxGuilds[:]),
		"tucked":            auditR3FlowCard(t, "baths"),
		"discard pile":      auditR3FlowCard(t, "altar"),
		"covered face-down": covered,
	}
	for _, what := range []string{"removed age II", "removed age I", "age III deck", "box guild", "tucked", "discard pile", "covered face-down"} {
		c := hidden[what]
		for _, step := range []string{"construct:" + c.String(), "discard:" + c.String(), "wonder:pyramids:" + c.String()} {
			if enginetest.IsLegal(&s, step) {
				t.Errorf("%s (%s): %s is legal", what, c, step)
				continue
			}
			if r := enginetest.MustFail(t, &s, step); r != engine.ReasonNotAccessible {
				t.Errorf("%s: %s refused with %s, want notAccessible", what, step, r)
			}
		}
	}
	var acc catalog.CardID
	for i := range engine.SlotsPerAge {
		if engine.Accessible(&s)&(1<<i) != 0 {
			acc = s.Slots[i].Card
			break
		}
	}
	if !enginetest.IsLegal(&s, "wonder:pyramids:"+acc.String()) {
		t.Fatalf("test setup: the Pyramids should be buildable with 99 coins: %v", enginetest.Keys(&s))
	}
	for _, w := range []string{"sphinx", "colossus", s.BoxWonders[0].String()} {
		if r := enginetest.MustFail(t, &s, "wonder:"+w+":"+acc.String()); r != engine.ReasonWonderUnavailable {
			t.Errorf("wonder %s: refused with %s, want wonderUnavailable", w, r)
		}
	}
}

// TestAuditR3FlowStarterChoiceAndGameOverAcceptNothingElse pins
// ages-scoring.md "Who starts the next age" and its "For developers" (the
// starter choice is its own phase, keys starter:0 and starter:1): while the
// choice is open, the new age's accessible cards are laid out but cannot be
// taken, not even by the chooser. After civilian scoring (ages-scoring.md
// "The end of the game") no action of any kind is legal.
func TestAuditR3FlowStarterChoiceAndGameOverAcceptNothingElse(t *testing.T) {
	s := enginetest.New(t, 318).Age(1).OnlyLeft("altar").ToMove(0).State()
	enginetest.Do(t, &s, "discard:altar")
	if s.Phase != engine.PhaseChooseStarter || s.Age != 2 {
		t.Fatalf("test setup: phase %s, age %d", s.Phase, s.Age)
	}
	if got := auditR3FlowSorted(enginetest.Keys(&s)); !slices.Equal(got, []string{"starter:0", "starter:1"}) {
		t.Errorf("starter choice offers %v", got)
	}
	n := 0
	for i := range engine.SlotsPerAge {
		if engine.Accessible(&s)&(1<<i) == 0 {
			continue
		}
		n++
		c := s.Slots[i].Card.String()
		for _, step := range []string{"construct:" + c, "discard:" + c} {
			if r := enginetest.MustFail(t, &s, step); r != engine.ReasonWrongPhase {
				t.Errorf("%s during the starter choice: reason %s, want wrongPhase", step, r)
			}
		}
	}
	if n != 2 {
		t.Errorf("Age II shows %d accessible cards during the choice, want 2", n)
	}
	if enginetest.IsLegal(&s, "starter:2") {
		t.Errorf("starter:2 is legal")
	}

	s = enginetest.New(t, 318).Age(3).OnlyLeft("palace").ToMove(1).State()
	enginetest.Do(t, &s, "discard:palace")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("test setup: the last discard of Age III did not end the game: %+v", s.Result)
	}
	if got := enginetest.Keys(&s); len(got) != 0 {
		t.Errorf("legal keys after the game: %v", got)
	}
	for _, step := range []string{"starter:0", "token:law", "discard:palace", "draft:pyramids", "revive:palace", "destroy:palace"} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonGameOver {
			t.Errorf("%s after the game: reason %s, want gameOver", step, r)
		}
	}
}

// TestAuditR3FlowLastCardPairIsPickedByTheBuilderThenTheWeakerChooses pins
// turn.md "The end of a turn" step 1 ("The active player makes these
// choices") and ages-scoring.md "Who starts the next age": two different
// players decide the two choices at the end of Age I. Seat 0 takes the last
// card, Workshop, completing a set-square pair with Laboratory; seat 0
// picks the token, while the age stays open; then, with the pawn on seat
// 1's side (+2), seat 1 chooses the starter and seat 0 stays the last
// active player.
func TestAuditR3FlowLastCardPairIsPickedByTheBuilderThenTheWeakerChooses(t *testing.T) {
	s := enginetest.New(t, 319).Age(1).OnlyLeft("workshop").City(0, "laboratory").
		BoardTokens("philosophy", "agriculture", "economy", "masonry", "law").
		Coins(0, 10).Pawn(2).ToMove(0).State()
	enginetest.Do(t, &s, "construct:workshop")
	if s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 || s.Age != 1 {
		t.Fatalf("after the pair: phase %s, to move %d, age %d; want seat 0's token pick in Age I", s.Phase, s.ToMove, s.Age)
	}
	if r := enginetest.MustFail(t, &s, "starter:1"); r != engine.ReasonWrongPhase {
		t.Errorf("starter:1 during the token pick: reason %s, want wrongPhase", r)
	}
	enginetest.Do(t, &s, "token:philosophy")
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 || s.Active != 0 {
		t.Fatalf("after the pick: age %d, phase %s, to move %d, active %d; want seat 1 (weaker) choosing, seat 0 last active",
			s.Age, s.Phase, s.ToMove, s.Active)
	}
	enginetest.Do(t, &s, "starter:0")
	if s.ToMove != 0 || s.Active != 0 || s.Phase != engine.PhasePlay {
		t.Errorf("seat 1 chose seat 0: phase %s, to move %d, active %d", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR3FlowRevivedRedCardOnTheLastCardDecidesTheChooser pins
// ages-scoring.md "The end of an age" (the age ends after the last turn's
// pending choices) and "Who starts the next age" (the pawn's side decides),
// where the pawn moves inside the pending choice. Seat 1 builds the
// Mausoleum with the last card of Age I, the pawn at +1 (seat 1 weaker).
// Rebuilding Walls (2 shields) carries the pawn to -1, so seat 0 is now
// weaker and chooses. Rebuilding Garrison (1 shield) brings it to the
// centre; under the LastActiveStarts option (decisions.md "Centred pawn",
// alternative) seat 1, the last active player, then starts Age II with no
// choice.
func TestAuditR3FlowRevivedRedCardOnTheLastCardDecidesTheChooser(t *testing.T) {
	t.Run("walls crosses the centre", func(t *testing.T) {
		s := enginetest.New(t, 320).Age(1).OnlyLeft("altar").
			Wonders(1, "mausoleum").Coins(1, 30).Discard("walls").Pawn(1).ToMove(1).State()
		enginetest.Do(t, &s, "wonder:mausoleum:altar")
		if s.Phase != engine.PhaseBuildFromDiscard || s.Age != 1 {
			t.Fatalf("phase %s, age %d; want the revive choice in Age I", s.Phase, s.Age)
		}
		enginetest.Do(t, &s, "revive:walls")
		if s.Military != -1 {
			t.Fatalf("pawn %d, want -1", s.Military)
		}
		if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 || s.Active != 1 {
			t.Errorf("age %d, phase %s, to move %d, active %d; want seat 0 (now weaker) choosing", s.Age, s.Phase, s.ToMove, s.Active)
		}
	})
	t.Run("garrison centres the pawn under LastActiveStarts", func(t *testing.T) {
		s := enginetest.New(t, 320).Options(engine.Options{CenterPawn: engine.LastActiveStarts}).Age(1).OnlyLeft("altar").
			Wonders(1, "mausoleum").Coins(1, 30).Discard("garrison").Pawn(1).ToMove(1).State()
		ev := enginetest.Do(t, &s, "wonder:mausoleum:altar", "revive:garrison")
		if s.Military != 0 {
			t.Fatalf("pawn %d, want 0", s.Military)
		}
		if s.Age != 2 || s.Phase != engine.PhasePlay || s.ToMove != 1 || s.Active != 1 || auditR3FlowCount(ev, engine.EvStarterChosen) != 0 {
			t.Errorf("age %d, phase %s, to move %d, active %d, events %v; want seat 1 to start Age II directly",
				s.Age, s.Phase, s.ToMove, s.Active, auditR3FlowKinds(ev))
		}
	})
}

// TestAuditR3FlowChainedChoicesEndWithLawOnTheLastCardOfAgeII pins
// decisions.md "Supremacy timing" (a scientific supremacy is checked right
// after a token is taken; on the last card of an age it ends the game with
// no age change) through a chain of two pending choices (turn.md "The end
// of a turn": "One choice can open another"). Seat 0 owns five different
// symbols (quill, set-square, mortar, wheel, sundial) and Theology. The
// Mausoleum, built with the last card of Age II, rebuilds Library, whose
// quill pairs with Scriptorium (still five symbols); the pair's token is
// Law, the sixth symbol (military-science.md "Scientific supremacy"). The
// game ends in Age II: no Age III, and the Theology extra turn is not
// reported lost.
func TestAuditR3FlowChainedChoicesEndWithLawOnTheLastCardOfAgeII(t *testing.T) {
	s := enginetest.New(t, 321).Age(2).OnlyLeft("rostrum").
		Wonders(0, "mausoleum").Tokens(0, "theology").Coins(0, 30).
		City(0, "scriptorium", "workshop", "pharmacist", "apothecary", "academy").
		Discard("library").BoardTokens("law", "agriculture", "economy", "masonry", "philosophy").
		ToMove(0).State()
	all := enginetest.Do(t, &s, "wonder:mausoleum:rostrum", "revive:library")
	if s.Phase != engine.PhasePickBoardToken || s.Age != 2 || s.Over() {
		t.Fatalf("after the revive: phase %s, age %d, over %v; want the pair's token pick", s.Phase, s.Age, s.Over())
	}
	ev := enginetest.Do(t, &s, "token:law")
	all = append(all, ev...)
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("result %+v; want seat 0's science victory", s.Result)
	}
	if s.Age != 2 || auditR3FlowCount(all, engine.EvAgeStart) != 0 || auditR3FlowCount(all, engine.EvExtraTurnLost) != 0 {
		t.Errorf("age %d, events %v; want the game to end in Age II with no new age", s.Age, auditR3FlowKinds(all))
	}
	if ev[len(ev)-1].Kind != engine.EvGameOver {
		t.Errorf("events %v; want the game-over event last", auditR3FlowKinds(ev))
	}
}

// TestAuditR3FlowMilitarySupremacyOnTheLastCardOfAgeIIIBeatsPoints pins
// ages-scoring.md "The end of the game" (a supremacy ends the game "with a
// single winner and no scoring") and decisions.md "Supremacy timing" ("on
// the last card of Age III it wins without civilian scoring"). Seat 0 leads
// on points by far; seat 1 takes the last card of Age III, Pretorium (3
// shields), paying exactly its 8 coins, with the pawn at -6: the pawn
// reaches seat 0's capital and seat 1 wins by military supremacy.
func TestAuditR3FlowMilitarySupremacyOnTheLastCardOfAgeIIIBeatsPoints(t *testing.T) {
	s := enginetest.New(t, 322).Age(3).OnlyLeft("pretorium").
		City(0, "palace", "town-hall", "pantheon").Coins(0, 30).
		Coins(1, 8).Pawn(-6).ToMove(1).State()
	if sc := engine.Scores(&s); sc[0].Total <= sc[1].Total {
		t.Fatalf("test setup: seat 0 should lead on points, scores %+v", sc)
	}
	ev := enginetest.Do(t, &s, "construct:pretorium")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 1 || s.Military != -9 {
		t.Fatalf("result %+v, pawn %d; want seat 1's military victory at -9", s.Result, s.Military)
	}
	last := ev[len(ev)-1]
	if last.Kind != engine.EvGameOver || last.A != int16(engine.VictoryMilitary) || last.B != 1 {
		t.Errorf("last event %+v; want game over, military, seat 1", last)
	}
}

// TestAuditR3FlowCircusMaximusOnTheLastCardCentresThePawn pins
// ages-scoring.md "Who starts the next age" with the pawn centred by the
// last card's own wonder, read after its destroy choice: seat 0, weaker at
// -1, builds Circus Maximus with the last card of Age I; its shield brings
// the pawn to the centre, the destroy choice holds the age open, and then
// the last active player (seat 0) chooses under the default ruling
// (decisions.md "Centred pawn") or simply starts under LastActiveStarts.
func TestAuditR3FlowCircusMaximusOnTheLastCardCentresThePawn(t *testing.T) {
	for _, c := range []struct {
		name   string
		opts   engine.Options
		choose bool
	}{{"default", engine.Options{}, true}, {"LastActiveStarts", engine.Options{CenterPawn: engine.LastActiveStarts}, false}} {
		t.Run(c.name, func(t *testing.T) {
			s := enginetest.New(t, 323).Options(c.opts).Age(1).OnlyLeft("altar").
				Wonders(0, "circus-maximus").Coins(0, 30).City(1, "glassworks").Pawn(-1).ToMove(0).State()
			enginetest.Do(t, &s, "wonder:circus-maximus:altar")
			if s.Military != 0 || s.Phase != engine.PhaseDestroyCard || s.Age != 1 {
				t.Fatalf("pawn %d, phase %s, age %d; want the centred pawn and the destroy choice in Age I", s.Military, s.Phase, s.Age)
			}
			ev := enginetest.Do(t, &s, "destroy:glassworks")
			if s.Age != 2 || s.Active != 0 || s.ToMove != 0 {
				t.Fatalf("age %d, to move %d, active %d; want Age II with seat 0 deciding", s.Age, s.ToMove, s.Active)
			}
			if c.choose && s.Phase != engine.PhaseChooseStarter {
				t.Errorf("phase %s, want the starter choice", s.Phase)
			}
			if !c.choose && (s.Phase != engine.PhasePlay || auditR3FlowCount(ev, engine.EvStarterChosen) != 0) {
				t.Errorf("phase %s, events %v; want seat 0 to start directly", s.Phase, auditR3FlowKinds(ev))
			}
		})
	}
}
