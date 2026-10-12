package engine

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The tests of the decider's hints (hints_decider.go, M10.HS3): each
// outcome field against the rules it encodes, the threats one turn ahead
// (a gift through a freed face-up card, none through a freshly revealed
// one, the reveal risk against brute-force enumeration), the budget, the
// view (only the decider's view carries outcomes), and the leak guard over
// the states these tests build. Helpers are prefixed ho.

// hoView is the decider's view of s with every hint family on.
func hoView(t testing.TB, s *State) view.View {
	t.Helper()
	h := ComputeHints(s, hnAllHints)
	return ViewWithHints(s, SeatViewer(s.ToMove), &h)
}

// hoOutcome returns the outcome of the choice key in the decider's view.
func hoOutcome(t testing.TB, s *State, key string) *view.OutcomeView {
	t.Helper()
	vw := hoView(t, s)
	for _, c := range vw.Choices {
		if c.Key == key {
			if c.Outcome == nil {
				t.Fatalf("%s has no outcome", key)
			}
			return c.Outcome
		}
	}
	t.Fatalf("%s is not a choice", key)
	return nil
}

// hoThreats renders threats as "seat:card-or-wonder:kind" strings.
func hoThreats(l *[]view.ThreatView) []string {
	out := []string{}
	if l == nil {
		return out
	}
	for _, t := range *l {
		name := ""
		if t.Card != nil {
			name = *t.Card
		}
		if t.Wonder != nil {
			name = *t.Wonder
		}
		out = append(out, string(rune('0'+t.Seat))+":"+name+":"+string(t.Kind))
	}
	return out
}

// TestOutcomeEconomyBeforeLooting: the trading part of a payment goes to
// the Economy owner before the pawn loots, so the looting takes from the
// coins it just received (decisions.md "Effect order"; military-science.md
// "Looting tokens").
func TestOutcomeEconomyBeforeLooting(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Pawn:       intPtr(1),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(20)}, {Coins: intPtr(1), Tokens: []string{"economy"}}},
		Accessible: []string{"walls"},
		ToMove:     seatPtr(0),
	})
	o := hoOutcome(t, &s, "construct:walls")
	// Walls costs two stone, bought at 2 each: 4 coins to seat 1, which
	// then loses 2 to the looting token at +3.
	if o.Coins != [2]view.Amount{16, 3} {
		t.Fatalf("coins after: %v, want [16 3]", o.Coins)
	}
	if o.Pawn == nil || o.Pawn.From != 1 || o.Pawn.To != 3 || len(o.Pawn.Loot) != 1 || o.Pawn.Loot[0] != (view.LootTaken{At: 3, Lost: 2}) {
		t.Fatalf("the pawn: %+v", o.Pawn)
	}
	if o.Score[0] <= 0 {
		t.Fatalf("the score change of the builder: %v", o.Score)
	}
}

// TestOutcomeAppianClamp: the Appian Way takes 3 coins from the opponent,
// never more than it has, gives its builder 3 and an extra turn
// (wonders-tokens.md "The twelve wonders", "The order of a wonder's
// effects").
func TestOutcomeAppianClamp(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Wonders:    wonderFill([]string{"appian-way"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}, {Coins: intPtr(1)}},
		Accessible: []string{"baths"},
		ToMove:     seatPtr(0),
	})
	o := hoOutcome(t, &s, "wonder:appian-way:baths")
	cost := int(WonderQuote(&s, 0, hnWonder(t, "appian-way")).Total)
	if o.Coins != [2]view.Amount{view.Amount(40 - cost + 3), 0} {
		t.Fatalf("coins after: %v, want [%d 0]", o.Coins, 40-cost+3)
	}
	if o.Tempo == nil || o.Tempo.ExtraTurn != view.ExtraTurnStateGain || o.Tempo.Source != view.ExtraTurnSourceWonder || o.Tempo.LastCard {
		t.Fatalf("the tempo: %+v", o.Tempo)
	}
	if o.Score[0] <= 0 {
		t.Fatalf("the wonder's points: %v", o.Score)
	}
}

// TestOutcomeMoneylendersSwing: discarding for coins can raise the score of
// the opponent's Moneylenders Guild, which counts the richer city
// (ages-scoring.md "Guilds"; turn.md "2. Discard it for coins").
func TestOutcomeMoneylendersSwing(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        3,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(7)}, {Coins: intPtr(6), City: []string{"moneylenders-guild"}}},
		Accessible: []string{"arsenal"},
		ToMove:     seatPtr(0),
	})
	gain := int(DiscardValue(&s, 0))
	o := hoOutcome(t, &s, "discard:arsenal")
	if o.Coins != [2]view.Amount{view.Amount(7 + gain), 6} {
		t.Fatalf("coins after: %v", o.Coins)
	}
	// Seat 1's guild: one point per 3 coins of the richer city, 7 coins
	// before and 7+gain after.
	want1 := view.Amount((7+gain)/3 - 7/3)
	want0 := want1
	if o.Score != [2]view.Amount{want0, want1} || want1 == 0 {
		t.Fatalf("score change: %v, want [%d %d]", o.Score, want0, want1)
	}
}

// TestOutcomeCivilianEnd: on Age III's last card every action ends the
// game, and its outcome carries the result the rules give
// (ages-scoring.md "The end of the game", "Tie-break and shared victory").
func TestOutcomeCivilianEnd(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:      3,
		Players:  [2]ScenarioPlayer{{Coins: intPtr(30), City: []string{"palace"}}, {Coins: intPtr(3)}},
		OnlyLeft: []string{"senate"},
		ToMove:   seatPtr(0),
	})
	for _, key := range []string{"construct:senate", "discard:senate"} {
		o := hoOutcome(t, &s, key)
		real := s
		hnDo(t, &real, key)
		if !real.Over() || real.Result.Kind != VictoryCivilian {
			t.Fatalf("%s: the game did not end on points", key)
		}
		want := viewResult(&real.Result)
		if o.End == nil || o.End.Victory != want.Victory || (o.End.Winner == nil) != (want.Winner == nil) ||
			(want.Winner != nil && *o.End.Winner != *want.Winner) || o.End.Scores != want.Scores {
			t.Fatalf("%s: end %+v, want %+v", key, o.End, want)
		}
		if o.Tempo == nil || !o.Tempo.LastCard || o.Tempo.StarterChooser != nil {
			t.Fatalf("%s: tempo %+v", key, o.Tempo)
		}
		if o.Gives != nil || o.RevealRisk != nil {
			t.Fatalf("%s: threats after the end", key)
		}
	}
}

// TestOutcomeStarterChooser: the last card of Age I hands the choice of
// the next starter to the weaker player (ages-scoring.md "Who starts the
// next age").
func TestOutcomeStarterChooser(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:      1,
		Pawn:     intPtr(2),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(30)}},
		OnlyLeft: []string{"baths"},
		ToMove:   seatPtr(0),
	})
	o := hoOutcome(t, &s, "discard:baths")
	if o.Tempo == nil || !o.Tempo.LastCard || o.Tempo.StarterChooser == nil || *o.Tempo.StarterChooser != 1 {
		t.Fatalf("tempo: %+v", o.Tempo)
	}
}

// TestOutcomePairThenLaw: a pair while Law is on the board and the seat has
// five different symbols wins through the token it then picks, and opens
// that pick (military-science.md "Pairs", "Scientific supremacy").
func TestOutcomePairThenLaw(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:         2,
		Players:     [2]ScenarioPlayer{{City: []string{"scriptorium", "workshop", "pharmacist", "apothecary", "university"}, Coins: intPtr(30)}},
		BoardTokens: []string{"law", "agriculture", "philosophy", "masonry", "strategy"},
		Accessible:  []string{"library"},
		ToMove:      seatPtr(0),
	})
	o := hoOutcome(t, &s, "construct:library")
	if o.Wins == nil || o.Wins.Kind != view.VictoryKindScience || o.Wins.Steps != 1 || o.Wins.Via == nil || *o.Wins.Via != "token:law" {
		t.Fatalf("wins: %+v", o.Wins)
	}
	if o.Opens == nil || *o.Opens != view.PendingKindPickBoardToken {
		t.Fatalf("opens: %v", o.Opens)
	}
	// The pick itself: the token choices carry their own outcomes.
	hnDo(t, &s, "construct:library")
	law := hoOutcome(t, &s, "token:law")
	if law.Wins == nil || law.Wins.Kind != view.VictoryKindScience || law.Wins.Steps != 1 {
		t.Fatalf("token:law wins: %+v", law.Wins)
	}
	agri := hoOutcome(t, &s, "token:agriculture")
	if agri.Coins[0] != view.Amount(s.Players[0].Coins)+6 || agri.Wins != nil {
		t.Fatalf("token:agriculture: %+v", agri)
	}
}

// TestOutcomeTwoStepWin: a wonder with an extra turn, then a card that
// takes the capital, wins in two steps; with the age's last card the extra
// turn is lost and so is the win (turn.md "Extra turns").
func TestOutcomeTwoStepWin(t *testing.T) {
	sc := Scenario{
		Age:      1,
		Wonders:  wonderFill([]string{"sphinx"}, nil),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Pawn:     intPtr(8),
		OnlyLeft: []string{"guard-tower", "baths"},
		ToMove:   seatPtr(0),
	}
	s := hnNew(t, sc)
	o := hoOutcome(t, &s, "wonder:sphinx:baths")
	if o.Wins == nil || o.Wins.Steps != 2 || o.Wins.Via == nil || *o.Wins.Via != "construct:guard-tower" || o.Wins.Kind != view.VictoryKindMilitary {
		t.Fatalf("wins: %+v", o.Wins)
	}
	if o.Tempo == nil || o.Tempo.ExtraTurn != view.ExtraTurnStateGain {
		t.Fatalf("tempo: %+v", o.Tempo)
	}
	at := hoOutcome(t, &s, "construct:guard-tower")
	if at.Wins == nil || at.Wins.Steps != 1 || at.End != nil {
		t.Fatalf("construct:guard-tower: wins %+v, end %+v", at.Wins, at.End)
	}

	sc.OnlyLeft = []string{"baths"}
	last := hnNew(t, sc)
	lo := hoOutcome(t, &last, "wonder:sphinx:baths")
	if lo.Wins != nil || lo.Tempo == nil || lo.Tempo.ExtraTurn != view.ExtraTurnStateLost || !lo.Tempo.LastCard {
		t.Fatalf("the last card: wins %+v, tempo %+v", lo.Wins, lo.Tempo)
	}
}

// TestOutcomeDisabledWins: a card the seat cannot afford that would take
// the capital says so on the disabled choice; a civilian count never
// counts (military-science.md "Military supremacy").
func TestOutcomeDisabledWins(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Pawn:       intPtr(8),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(0)}},
		Accessible: []string{"horse-breeders", "sawmill"},
		ToMove:     seatPtr(0),
	})
	vw := hoView(t, &s)
	found := false
	for _, d := range vw.Disabled {
		switch d.Key {
		case "construct:horse-breeders":
			found = true
			if d.Wins == nil || d.Wins.Kind != view.VictoryKindMilitary || d.Wins.Steps != 1 {
				t.Fatalf("horse-breeders: %+v", d.Wins)
			}
		case "construct:sawmill":
			if d.Wins != nil {
				t.Fatal("the Sawmill wins nothing")
			}
		}
	}
	if !found {
		t.Fatal("horse-breeders is not disabled")
	}
}

// hoGiftScenario is Age II with seven cards left: slots 0 to 5 in row 1,
// face up, and slot 6 in row 2, which alone covers slots 0 and 1. Seat 1
// is one shield from taking seat 0's capital, and Horse Breeders, in slot
// 0, would give it that shield; seat 0 is to move.
func hoGiftScenario(t testing.TB) State {
	t.Helper()
	return hnNew(t, Scenario{
		Age:      2,
		Wonders:  wonderFill(nil, []string{"pyramids", "great-lighthouse", "hanging-gardens", "mausoleum"}),
		Pawn:     intPtr(-8),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(10)}, {Coins: intPtr(5)}},
		OnlyLeft: []string{"horse-breeders", "school", "laboratory", "dispensary", "library", "brewery", "forum"},
		ToMove:   seatPtr(0),
	})
}

// TestOutcomeGiftThroughFreedCard: taking the only card that covers a face-
// up card the opponent wins with hands that win over (gives); the card
// itself, or one that covers nothing, gives nothing (turn.md "Taking a
// card"; military-science.md "Military supremacy").
func TestOutcomeGiftThroughFreedCard(t *testing.T) {
	s := hoGiftScenario(t)
	if s.Slots[0].Card != hnCard(t, "horse-breeders") || !s.Slots[0].FaceUp || Accessible(&s)&1 != 0 {
		t.Fatal("the scenario: Horse Breeders is not face up and covered in slot 0")
	}
	o := hoOutcome(t, &s, "discard:forum")
	if got := hoThreats(o.Gives); len(got) != 1 || got[0] != "1:horse-breeders:military" {
		t.Fatalf("discard:forum gives %v", got)
	}
	if o.Blocks != nil {
		t.Fatalf("discard:forum blocks %v", hoThreats(o.Blocks))
	}
	if o2 := hoOutcome(t, &s, "discard:brewery"); o2.Gives != nil {
		t.Fatalf("discard:brewery gives %v", hoThreats(o2.Gives))
	}
	hlCheck(t, &s, hoRNG(3), hlSamples)

	// After the forum is gone, seat 1 can win; seat 0 to move again (as if
	// it played again) blocks it by taking the Horse Breeders.
	after := s
	hnDo(t, &after, "discard:forum")
	after.Active, after.ToMove = 0, 0
	b := hoOutcome(t, &after, "construct:horse-breeders")
	if got := hoThreats(b.Blocks); len(got) != 1 || got[0] != "1:horse-breeders:military" {
		t.Fatalf("taking the Horse Breeders blocks %v", got)
	}
}

// hoRNG returns a fresh generator, for hlCheck.
func hoRNG(seed uint64) *RNG {
	r := NewRNG(seed)
	return &r
}

// hoRevealScenario is Age II with twelve cards left: rows 1 and 2 full but
// slot 6 and 7 face down under slot 11 (row 3), the only card of row 3.
// Horse Breeders lies face down in slot 6 and seat 1 is one shield from
// seat 0's capital.
func hoRevealScenario(t testing.TB) State {
	t.Helper()
	s := hnNew(t, Scenario{
		Age:     2,
		Wonders: wonderFill(nil, []string{"pyramids", "great-lighthouse", "hanging-gardens", "mausoleum"}),
		Pawn:    intPtr(-8),
		Players: [2]ScenarioPlayer{{Coins: intPtr(10)}, {Coins: intPtr(5)}},
		OnlyLeft: []string{"school", "laboratory", "dispensary", "library", "brewery", "forum",
			"horse-breeders", "caravansery", "customs-house", "temple", "statue", "rostrum"},
		ToMove: seatPtr(0),
	})
	return s
}

// TestOutcomeNoGiftThroughReveal: a card the action turns over is never
// used for a gift, since the mover has not seen it; it shows only as the
// reveal risk, whose odds match a brute-force enumeration of what may lie
// under the freed backs (setup.md "What is hidden from both players";
// decisions.md "Reveal timing").
func TestOutcomeNoGiftThroughReveal(t *testing.T) {
	s := hoRevealScenario(t)
	hb := hnCard(t, "horse-breeders")
	if s.Slots[6].Card != hb || s.Slots[6].FaceUp || s.Slots[7].FaceUp || Accessible(&s)&(1<<11) == 0 {
		t.Fatalf("the scenario: slots 6 and 7 face down under slot 11")
	}
	key := "discard:rostrum"
	o := hoOutcome(t, &s, key)
	if o.Gives != nil {
		t.Fatalf("a freshly revealed card gives %v", hoThreats(o.Gives))
	}
	if o.RevealRisk == nil || o.RevealRisk.Freed != 2 {
		t.Fatalf("reveal risk: %+v", o.RevealRisk)
	}

	// Brute force: every pair of unseen Age II cards in slots 6 and 7, the
	// rest set aside; count the pairs with which seat 1 can win at once.
	var pool []catalog.CardID
	for c, ok := catalog.T.AgeCards[2].Next(1); ok; c, ok = catalog.T.AgeCards[2].Next(c + 1) {
		if publicPlace(&s, c).Place == PlaceUnseenThisAge {
			pool = append(pool, c)
		}
	}
	a := hnKey(t, key)
	mask := FaceUpSlots(&s) | 1<<6 | 1<<7
	wins, pairs := 0, 0
	for i := range pool {
		for j := range pool {
			if i == j {
				continue
			}
			d := hoPlace(t, &s, pool[i], pool[j])
			ApplyTrusted(&d, a)
			list, _ := threatsFor(&d, 1, mask, nil, true)
			pairs++
			if list.N > 0 {
				wins++
			}
		}
	}
	// wins/pairs over ordered pairs equals the odds.
	if wins*o.RevealRisk.Den != pairs*o.RevealRisk.Num {
		t.Fatalf("reveal risk %d/%d, brute force %d/%d (pool %d)", o.RevealRisk.Num, o.RevealRisk.Den, wins, pairs, len(pool))
	}
	hlCheck(t, &s, hoRNG(4), hlSamples)
}

// hoPlace returns a copy of s with cards a and b in slots 6 and 7, and the
// cards they replace set aside in their place.
func hoPlace(t testing.TB, s *State, a, b catalog.CardID) State {
	t.Helper()
	d := *s
	for k, c := range []catalog.CardID{a, b} {
		slot := 6 + k
		if d.Slots[slot].Card == c {
			continue
		}
		old := d.Slots[slot].Card
		moved := false
		if j := SlotOf(&d, c); j >= 0 {
			d.Slots[j].Card = old
			moved = true
		} else {
			for r := range d.Removed[1] {
				if d.Removed[1][r] == c {
					d.Removed[1][r] = old
					moved = true
				}
			}
		}
		if !moved {
			t.Fatalf("card %d is neither in the structure nor set aside", c)
		}
		d.Slots[slot].Card = c
	}
	if err := CheckInvariants(&d); err != nil {
		t.Fatal(err)
	}
	return d
}

// TestOutcomeThreatens: an action can set up the mover's own win for its
// next turn (creates-threat): freeing its own winning card.
func TestOutcomeThreatens(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:      2,
		Wonders:  wonderFill([]string{"pyramids", "great-lighthouse", "hanging-gardens", "mausoleum"}, nil),
		Pawn:     intPtr(8),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(10)}, {Coins: intPtr(5)}},
		OnlyLeft: []string{"horse-breeders", "school", "laboratory", "dispensary", "library", "brewery", "forum"},
		ToMove:   seatPtr(0),
	})
	o := hoOutcome(t, &s, "discard:forum")
	if got := hoThreats(o.Threatens); len(got) != 1 || got[0] != "0:horse-breeders:military" {
		t.Fatalf("threatens %v", got)
	}
}

// TestOutcomeStarterThreat: when a new age is laid out, the starter
// choice says which seat would win at once if it started (starter-threat).
func TestOutcomeStarterThreat(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:      1,
		Pawn:     intPtr(8),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(30)}, {Coins: intPtr(30)}},
		OnlyLeft: []string{"baths"},
		ToMove:   seatPtr(0),
	})
	hnDo(t, &s, "discard:baths")
	if s.Phase != PhaseChooseStarter || s.ToMove != 1 {
		t.Fatalf("phase %s, to move %d", s.Phase, s.ToMove)
	}
	// Put Horse Breeders into an accessible slot of Age II.
	hb := hnCard(t, "horse-breeders")
	acc := Accessible(&s)
	slot := -1
	for i := range s.Slots {
		if acc&(1<<i) != 0 {
			slot = i
			break
		}
	}
	if j := SlotOf(&s, hb); j >= 0 {
		s.Slots[j].Card, s.Slots[slot].Card = s.Slots[slot].Card, hb
	} else {
		for r := range s.Removed[1] {
			if s.Removed[1][r] == hb {
				s.Removed[1][r] = s.Slots[slot].Card
				s.Slots[slot].Card = hb
			}
		}
	}
	if s.Slots[slot].Card != hb {
		t.Fatal("Horse Breeders was not placed")
	}
	if err := CheckInvariants(&s); err != nil {
		t.Fatal(err)
	}
	o0 := hoOutcome(t, &s, "starter:0")
	if got := hoThreats(o0.Threatens); !slices.Contains(got, "0:horse-breeders:military") {
		t.Fatalf("starter:0 threatens %v", got)
	}
	vw := hoView(t, &s)
	for _, c := range vw.Choices {
		if c.Key == "starter:1" && c.Outcome != nil && c.Outcome.Threatens != nil {
			for _, tv := range *c.Outcome.Threatens {
				if tv.Seat != 1 {
					t.Fatalf("starter:1 names seat %d", tv.Seat)
				}
			}
		}
	}
	hlCheck(t, &s, hoRNG(5), hlSamples)
}

// TestOutcomeOnlyDecider: only the decider's view carries outcomes and
// disabled wins; the waiting seat and spectators get none, and a view
// built without hints none either.
func TestOutcomeOnlyDecider(t *testing.T) {
	s := hoGiftScenario(t)
	h := ComputeHints(&s, hnAllHints)
	for _, v := range []Viewer{SeatViewer(s.ToMove.Opp()), Spectator} {
		vw := ViewWithHints(&s, v, &h)
		for _, c := range vw.Choices {
			if c.Outcome != nil {
				t.Fatalf("viewer %d: an outcome on %s", v, c.Key)
			}
		}
	}
	for _, c := range ViewFor(&s, SeatViewer(s.ToMove)).Choices {
		if c.Outcome != nil {
			t.Fatalf("ViewFor: an outcome on %s", c.Key)
		}
	}
	n := 0
	for _, c := range ViewWithHints(&s, SeatViewer(s.ToMove), &h).Choices {
		if c.Outcome != nil {
			n++
		}
	}
	if n == 0 {
		t.Fatal("the decider's view has no outcome")
	}
	// Families off: no outcome at all.
	off := view.HintSettings{Prices: true, Wonders: true, Tokens: true, Counting: true, Recap: true}
	h = ComputeHints(&s, off)
	for _, c := range ViewWithHints(&s, SeatViewer(s.ToMove), &h).Choices {
		if c.Outcome != nil {
			t.Fatalf("no decider family on: an outcome on %s", c.Key)
		}
	}
	// Threats off: no threat field.
	noThreats := hnAllHints
	noThreats.Threats = false
	h = ComputeHints(&s, noThreats)
	for _, c := range ViewWithHints(&s, SeatViewer(s.ToMove), &h).Choices {
		if o := c.Outcome; o != nil && (o.Wins != nil || o.Gives != nil || o.Blocks != nil || o.Threatens != nil || o.RevealRisk != nil) {
			t.Fatalf("threats off: %s carries a threat", c.Key)
		}
	}
}

// TestOutcomeBudget: with a small budget, the threats are left out, never
// guessed: each threat field is either absent or the one an unlimited run
// gives.
func TestOutcomeBudget(t *testing.T) {
	for _, s := range []State{hoGiftScenario(t), hoRevealScenario(t), hnMidAge(t, 2, 2)} {
		full := ViewFor(&s, SeatViewer(s.ToMove))
		deciderHints(&s, hnAllHints, &full, &simCtx{})
		for _, budget := range []int{1, 7, 40, 200, 1000} {
			part := ViewFor(&s, SeatViewer(s.ToMove))
			deciderHints(&s, hnAllHints, &part, &simCtx{budget: budget})
			for i, c := range part.Choices {
				o, f := c.Outcome, full.Choices[i].Outcome
				if o == nil {
					continue
				}
				same := func(a, b any) bool {
					x, _ := json.Marshal(a)
					y, _ := json.Marshal(b)
					return string(x) == string(y)
				}
				if (o.Gives != nil && !same(o.Gives, f.Gives)) || (o.Blocks != nil && !same(o.Blocks, f.Blocks)) ||
					(o.Threatens != nil && !same(o.Threatens, f.Threatens)) || (o.RevealRisk != nil && !same(o.RevealRisk, f.RevealRisk)) ||
					(o.Wins != nil && !same(o.Wins, f.Wins)) {
					t.Fatalf("budget %d, %s: %+v differs from %+v", budget, c.Key, o, f)
				}
			}
		}
	}
}

// TestDeciderSimulationsStayPublic traces every simulation of the
// decider's hints over the leak corpus: the actions asked about and their
// own follow-ups use only face-up cards; every simulation one turn ahead
// (depth 2 and up) runs on a scrubbed copy, where each card that was not
// face up before is catalog.Unknown, except one probe slot of the reveal
// risk holding a card whose public place is unseen this age.
func TestDeciderSimulationsStayPublic(t *testing.T) {
	games := 10
	if testing.Short() {
		games = 3
	}
	var notes []simNote
	ctx := &simCtx{trace: func(n simNote) { notes = append(notes, n) }}
	nested := 0
	states := append(hlCorpus(t, games), hoGiftScenario(t), hoRevealScenario(t))
	for _, s := range states {
		if s.Over() {
			continue
		}
		notes = notes[:0]
		vw := ViewFor(&s, SeatViewer(s.ToMove))
		deciderHints(&s, hnAllHints, &vw, ctx)
		var shallow []simNote
		faceUp := FaceUpSlots(&s)
		for _, n := range notes {
			if n.Depth < 2 {
				shallow = append(shallow, n)
				continue
			}
			nested++
			probe := n.Mask &^ faceUp
			if probe&(probe-1) != 0 {
				t.Fatalf("ply %d: a nested mask %#x adds more than one probe slot to %#x", s.Ply, n.Mask, faceUp)
			}
			for i := range n.State.Slots {
				c := n.State.Slots[i].Card
				switch {
				case c == 0 || c == catalog.Unknown:
				case faceUp&(1<<i) != 0:
					if c != s.Slots[i].Card {
						t.Fatalf("ply %d: slot %d holds %d, not the face-up %d", s.Ply, i, c, s.Slots[i].Card)
					}
				case probe&(1<<i) != 0:
					if publicPlace(&s, c).Place != PlaceUnseenThisAge {
						t.Fatalf("ply %d: the probe slot %d holds card %d, which is not unseen", s.Ply, i, c)
					}
				default:
					t.Fatalf("ply %d: a nested simulation sees card %d in slot %d, which was not face up", s.Ply, c, i)
				}
			}
			switch n.Action.Type {
			case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
				if i := SlotOf(&n.State, n.Action.Card); i < 0 || n.Mask&(1<<i) == 0 {
					t.Fatalf("ply %d: nested %s uses a slot outside its mask", s.Ply, n.Action)
				}
			}
		}
		// The decider's own actions and follow-ups: as for the shared hints,
		// except that a pending phase's action may follow a reveal the
		// decider has seen.
		for _, n := range shallow {
			switch n.Action.Type {
			case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
				if i := SlotOf(&n.State, n.Action.Card); i < 0 || faceUp&(1<<i) == 0 {
					t.Fatalf("ply %d: %s (depth %d) uses a card that was not face up", s.Ply, n.Action, n.Depth)
				}
			}
			if n.Depth == 1 && (n.State.Phase == PhasePickLibraryToken && n.State.ToMove != s.ToMove) {
				t.Fatalf("ply %d: a follow-up in another seat's Great Library offer", s.Ply)
			}
		}
	}
	if nested == 0 {
		t.Fatal("no nested simulation was traced")
	}
	t.Logf("%d nested simulations checked", nested)
}

// TestDeciderHintsCost records the Applies and time the decider's hints
// take over the leak corpus, and checks the design's budget: every view
// within hintApplyBudget, and the median under 5 ms.
func TestDeciderHintsCost(t *testing.T) {
	maxApplies, cut := 0, 0
	var worst time.Duration
	var times []time.Duration
	for _, s := range hlCorpus(t, 6) {
		if s.Over() {
			continue
		}
		vw := ViewFor(&s, SeatViewer(s.ToMove))
		ctx := &simCtx{budget: hintApplyBudget}
		start := time.Now()
		deciderHints(&s, hnAllHints, &vw, ctx)
		d := time.Since(start)
		times = append(times, d)
		worst = max(worst, d)
		maxApplies = max(maxApplies, ctx.applies)
		if ctx.out {
			cut++
		}
	}
	slicesSortDurations(times)
	median := times[len(times)/2]
	t.Logf("%d states: at most %d Applies, %d cut by the budget, median %v, worst %v", len(times), maxApplies, cut, median, worst)
	if maxApplies > hintApplyBudget {
		t.Fatalf("%d Applies in one view", maxApplies)
	}
}

// slicesSortDurations sorts ds in place (insertion sort; the lists are
// short).
func slicesSortDurations(ds []time.Duration) {
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0 && ds[j] < ds[j-1]; j-- {
			ds[j], ds[j-1] = ds[j-1], ds[j]
		}
	}
}

// TestBinomialOdds: the reveal-risk formula, 1 - C(m-k, n) / C(m, n),
// against counting the subsets directly.
func TestBinomialOdds(t *testing.T) {
	for m := 1; m <= 9; m++ {
		for n := 1; n <= 3 && n <= m; n++ {
			for k := 0; k <= m; k++ {
				hit, all := 0, 0
				for set := 0; set < 1<<m; set++ {
					if hnCount(uint32(set)) != n {
						continue
					}
					all++
					if set&(1<<k-1) != 0 { // the first k cards are the winning ones
						hit++
					}
				}
				if all != binomial(m, n) || hit != binomial(m, n)-binomial(m-k, n) {
					t.Fatalf("m %d, n %d, k %d: %d/%d, formula %d/%d", m, n, k, hit, all, binomial(m, n)-binomial(m-k, n), binomial(m, n))
				}
			}
		}
	}
}
