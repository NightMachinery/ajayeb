package rules

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// The discard ability lets the owner search the whole pile, and a card that
// shares a name with one of the city's buildings cannot be built (rules.md,
// "Discard ability"). The searching seat is told why each such card is not
// offered; nobody else gets a reason for a pile card.
func TestViewDiscardSearchExplainsBarredCards(t *testing.T) {
	g := abilityScenario(t, 2, 4, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2, 23}}, 1: {board: catalog.BoardOlympia}}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {28}, 6: {29}}, 46)
	b := actionBatch(g, 0, stagePick(t, g, 0, 39))
	b.Picks[1].Key = (Action{Kind: ActBuild, Design: 28}).Key()
	b.Picks[6].Key = (Action{Kind: ActBuild, Design: 29}).Key()
	pending, _, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseDiscardBuild || pending.State.Decider != 0 {
		t.Fatal("no discard search", pending.State.Phase)
	}
	you := pending.ViewFor(0).You
	var barred []view.Disabled
	for _, o := range you.Offer {
		id, _ := catalog.DesignBySlug(o.Design)
		d, _ := catalog.DesignByID(id)
		named := !pending.State.Cities[0].Buildings.And(catalog.T.ByName[d.NameID]).Empty()
		if o.Legal == named {
			t.Fatal("the name rule and the offer disagree", o)
		}
		if !o.Legal {
			barred = append(barred, view.Disabled{Kind: "build-discarded", Design: o.Design, Reason: "already-built"})
		}
	}
	if len(barred) == 0 {
		t.Fatal("the scenario's pile holds no card the city already has")
	}
	if !slices.Equal(you.Disabled, barred) {
		t.Fatalf("disabled %+v, want %+v", you.Disabled, barred)
	}
	for seat := Seat(1); seat < 7; seat++ {
		if d := pending.ViewFor(seat).You.Disabled; len(d) != 0 {
			t.Fatal("a pile reason reached a seat that is not searching", seat, d)
		}
	}
}

// keptPreview maps each offered key to the card the view says it keeps.
func keptPreview(g Game, seat Seat) map[string]string {
	out := map[string]string{}
	for _, c := range g.ViewFor(seat).You.Choices {
		if c.Kept != nil {
			out[c.Key] = *c.Kept
		}
	}
	return out
}

// keptAfter resolves the turn with seat's choice, every other seat taking
// its first offer, and returns the card the seat kept.
func keptAfter(t *testing.T, g Game, seat Seat, a Action) catalog.DesignID {
	t.Helper()
	var picks [MaxSeats]Action
	for s := Seat(0); s < Seat(g.State.Seats); s++ {
		picks[s] = g.Legal(s)[0].Action
	}
	picks[seat] = a
	next, _, err := ResolveState(&g.State, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	return next.Cities[seat].Kept
}

// Ruling 1: a Babylon night stage built on the sixth turn keeps the other
// card in that same Age. Only that stage keeps it; building or selling
// without the ability discards it. Every preview matches what resolving the
// turn with that choice keeps.
func TestViewKeptPreviewSameTurnStage(t *testing.T) {
	g := abilityScenario(t, 1, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, buildings: []catalog.DesignID{8}}, 1: {board: catalog.BoardGizah}}, [MaxSeats][]catalog.DesignID{0: {17, 18}})
	other := map[catalog.DesignID]catalog.DesignID{17: 18, 18: 17}
	preview := keptPreview(g, 0)
	stages := 0
	for _, c := range g.Legal(0) {
		want := catalog.DesignID(0)
		if c.Kind == ActStage {
			want = other[c.Design]
			stages++
		}
		if got := keptAfter(t, g, 0, c.Action); got != want {
			t.Fatal("the scenario's resolution", c.Key(), got, want)
		}
		if got, ok := preview[c.Key()]; ok != (want != 0) || (ok && got != designSlug(want)) {
			t.Fatalf("%s previews %q, want %q", c.Key(), got, designSlug(want))
		}
	}
	if stages == 0 {
		t.Fatal("no stage offered")
	}
	for seat := Seat(1); seat < 7; seat++ {
		if p := keptPreview(g, seat); len(p) != 0 {
			t.Fatal("a seat without the ability previews a kept card", seat, p)
		}
	}
	// Turn five is not the sixth turn: nothing is kept yet.
	g = abilityScenario(t, 1, 5, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, buildings: []catalog.DesignID{8}}}, [MaxSeats][]catalog.DesignID{0: {17, 18, 19}})
	if p := keptPreview(g, 0); len(p) != 0 {
		t.Fatal("a kept card before the sixth turn", p)
	}
}

// With the ability already built, every action of the sixth turn keeps the
// other card (rules.md, "Last card ability").
func TestViewKeptPreviewAbilityHeld(t *testing.T) {
	g := abilityScenario(t, 3, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, stages: 1, buildings: []catalog.DesignID{2, 28, 30, 11}}}, [MaxSeats][]catalog.DesignID{0: {51, 64}})
	other := map[catalog.DesignID]catalog.DesignID{51: 64, 64: 51}
	preview := keptPreview(g, 0)
	choices := g.Legal(0)
	if len(choices) == 0 || len(preview) != len(choices) {
		t.Fatal("not every choice keeps the other card", len(preview), len(choices))
	}
	for _, c := range choices {
		if preview[c.Key()] != designSlug(other[c.Design]) || keptAfter(t, g, 0, c.Action) != other[c.Design] {
			t.Fatal("kept preview", c.Key(), preview[c.Key()])
		}
	}
	if g.ViewFor(NoSeat).You != nil {
		t.Fatal("a spectator got choices")
	}
}

// Military conflicts (rules.md): each event names both strengths, and a
// victory is worth 1, 3 or 5 by Age, a tie nothing. The tokens the events
// give add up to each city's military score.
func TestMilitaryEventsCarryShieldsAndTokens(t *testing.T) {
	worth := [NumAges]int{1, 3, 5}
	wins, ties := 0, 0
	for seed := uint64(1); seed <= 4; seed++ {
		g, err := NewGame(Setup{Seats: 5, Seed: seed, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		var military [MaxSeats]int
		for g.State.Phase != PhaseOver {
			next, events, err := g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
			var raw []Event
			for _, e := range events {
				if e.Kind != EvMilitary {
					continue
				}
				raw = append(raw, e)
				left, right := next.State.Cities[e.Seat].Shields, next.State.Cities[e.Other].Shields
				if e.Left != left || e.Right != right {
					t.Fatal("conflict strengths", e, left, right)
				}
				switch {
				case left == right:
					ties++
					if e.Delta != 0 || e.Outcome != ConflictTie {
						t.Fatal("a tie with a token", e)
					}
				case left > right:
					wins++
					military[e.Seat] += worth[e.Age-1]
					military[e.Other]--
					if int(e.Delta) != worth[e.Age-1] || e.Outcome != ConflictSeatWins {
						t.Fatal("victory token", e)
					}
				default:
					wins++
					military[e.Other] += worth[e.Age-1]
					military[e.Seat]--
					if int(e.Delta) != worth[e.Age-1] || e.Outcome != ConflictOtherWins {
						t.Fatal("victory token", e)
					}
				}
			}
			for _, viewer := range []Seat{NoSeat, 0, 3} {
				for i, v := range FilterEvents(raw, viewer) {
					e := raw[i]
					if v.Kind != "military" || v.Left != int(e.Left) || v.Right != int(e.Right) || v.Delta != int(e.Delta) {
						t.Fatal("filtered conflict", viewer, v, e)
					}
				}
			}
			g = next
		}
		for s := Seat(0); s < 5; s++ {
			if int(g.State.Result.Scores[s].Military) != military[s] {
				t.Fatal("tokens and military score disagree", s, military[s], g.State.Result.Scores[s].Military)
			}
		}
	}
	if wins == 0 || ties == 0 {
		t.Fatal("too few conflicts of each kind", wins, ties)
	}
}

// Ruling 20: the highest total wins; on a tie the most coins win, and
// players still tied share the victory. A city here holds coins alone, so
// its total is a point per three coins.
func TestTopTie(t *testing.T) {
	for _, tc := range []struct {
		name  string
		coins []int16
		kind  string
		tied  SeatSet
	}{
		{"one seat on top", []int16{12, 9, 9}, "", 0},
		{"coins break a two-way tie", []int16{9, 10, 3}, "coins", 0b011},
		{"equal coins share", []int16{9, 9, 3}, "shared", 0b011},
		{"coins narrow a three-way tie to two", []int16{9, 11, 11}, "coins", 0b111},
		{"coins break a three-way tie", []int16{11, 9, 10}, "coins", 0b111},
		{"all three share", []int16{9, 9, 9}, "shared", 0b111},
	} {
		st := State{Seats: uint8(len(tc.coins)), Phase: PhaseOver}
		for s, c := range tc.coins {
			st.Cities[s].Coins = c
		}
		st.Result = scoreResult(&st)
		kind, tied := topTie(&st.Result, st.Seats)
		if kind != tc.kind || tied != tc.tied {
			t.Errorf("%s: %q %03b, want %q %03b", tc.name, kind, tied, tc.kind, tc.tied)
		}
	}
}

// A finished view states the tie with the seats in seat order, and an
// empty list (never null) when nobody tied.
func TestViewResultTieBreak(t *testing.T) {
	seen := map[string]bool{}
	for seed := uint64(0); seed < 40 && len(seen) < 2; seed++ {
		g, err := NewGame(Setup{Seats: 3, Seed: seed, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		for g.State.Phase != PhaseOver {
			g, _, err = g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
		}
		kind, tied := topTie(&g.State.Result, 3)
		for _, viewer := range []Seat{NoSeat, 0, 2} {
			r := g.ViewFor(viewer).Result
			want := []int{}
			for s := Seat(0); s < 3; s++ {
				if tied.Has(s) {
					want = append(want, int(s))
				}
			}
			if r == nil || r.TieBreak != kind || r.Tied == nil || !slices.Equal(r.Tied, want) {
				t.Fatal("view tie", r)
			}
		}
		seen[kind] = true
	}
	if !seen[""] {
		t.Fatal("no untied game in the sweep")
	}
}
