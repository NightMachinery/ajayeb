package engine_test

import (
	"slices"
	"sort"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Tests of the named scenarios (scenarios_named.go): one test per scenario,
// each running its Trigger and checking the state and events it predicts.
// Helpers are prefixed ns, this file's own (see AGENTS.md on prefixing a
// slice's test helpers so they never collide with another file's).

// nsID, nsTokenID and nsWonderID resolve a slug, failing the test if it is
// not registered.
func nsID(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("no card %q", slug)
	}
	return c
}

func nsTokenID(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("no token %q", slug)
	}
	return tk
}

func nsWonderID(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	w, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("no wonder %q", slug)
	}
	return w
}

// nsWonder returns p's slot of the named wonder, failing the test if p does
// not have it.
func nsWonder(t *testing.T, p *engine.Player, slug string) engine.WonderSlot {
	t.Helper()
	id := nsWonderID(t, slug)
	for _, w := range p.Wonders {
		if w.ID == id {
			return w
		}
	}
	t.Fatalf("does not have wonder %q: %+v", slug, p.Wonders)
	return engine.WonderSlot{}
}

// nsRun looks up the named scenario, builds its state with seed, runs its
// Trigger in full, and returns the resulting state and the trigger's events.
// It fails the test if the scenario is unknown or malformed, or its Trigger
// does not apply in full.
func nsRun(t *testing.T, name string, seed uint64) (engine.State, []engine.Event) {
	t.Helper()
	sc := engine.NamedScenario(name)
	if sc == nil {
		t.Fatalf("no named scenario %q", name)
	}
	if sc.Name != name {
		t.Fatalf("scenario %q: Name is %q", name, sc.Name)
	}
	if sc.Expect == "" {
		t.Fatalf("scenario %q: Expect is empty", name)
	}
	if len(sc.Trigger) == 0 {
		t.Fatalf("scenario %q: Trigger is empty", name)
	}
	s, err := engine.NewGameFromScenario(seed, engine.Options{}, sc)
	if err != nil {
		t.Fatalf("scenario %q: %v", name, err)
	}
	var ev []engine.Event
	if err := engine.RunTrigger(&s, sc, &ev); err != nil {
		t.Fatalf("scenario %q: trigger: %v", name, err)
	}
	if err := engine.CheckInvariants(&s); err != nil {
		t.Fatalf("scenario %q: after trigger: %v", name, err)
	}
	return s, ev
}

// nsHasEvent and nsHasKind scan a trigger's events.
func nsHasEvent(ev []engine.Event, want engine.Event) bool {
	for _, e := range ev {
		if e == want {
			return true
		}
	}
	return false
}

func nsHasKind(ev []engine.Event, k engine.EventKind) bool {
	for _, e := range ev {
		if e.Kind == k {
			return true
		}
	}
	return false
}

// nsInBox reports whether token tk sits anywhere in the box.
func nsInBox(s *engine.State, tk catalog.TokenID) bool {
	for _, x := range s.BoxTokens {
		if x == tk {
			return true
		}
	}
	return false
}

func TestNamedScenarioMausoleum(t *testing.T) {
	s, _ := nsRun(t, "mausoleum", 101)
	baths := nsID(t, "baths")
	if !s.Players[0].City.Has(baths) {
		t.Fatalf("baths was not rebuilt into seat 0's city: %v", s.Players[0].City)
	}
	if s.Discard.Has(baths) {
		t.Fatalf("baths is still in the discard pile")
	}
	w := nsWonder(t, &s.Players[0], "mausoleum")
	if !w.Built || w.Tucked != nsID(t, "rostrum") {
		t.Fatalf("mausoleum: %+v", w)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("phase %s, toMove %d, active %d; want seat 1's turn", s.Phase, s.ToMove, s.Active)
	}
}

func TestNamedScenarioMausoleumEmpty(t *testing.T) {
	s, ev := nsRun(t, "mausoleum-empty", 102)
	w := nsWonder(t, &s.Players[0], "mausoleum")
	if !w.Built || w.Tucked != nsID(t, "rostrum") {
		t.Fatalf("mausoleum: %+v", w)
	}
	want := engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Wonder: nsWonderID(t, "mausoleum"), A: int16(engine.PhaseBuildFromDiscard)}
	if !nsHasEvent(ev, want) {
		t.Fatalf("events %+v: want %+v", ev, want)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, toMove %d; want seat 1's turn", s.Phase, s.ToMove)
	}
}

func TestNamedScenarioGreatLibrary(t *testing.T) {
	s, _ := nsRun(t, "great-library", 103)
	law := nsTokenID(t, "law")
	if !s.Players[0].Tokens.Has(law) {
		t.Fatalf("seat 0 does not hold law: %v", s.Players[0].Tokens)
	}
	if s.BoardTokens.Has(law) {
		t.Fatalf("law is still on the board")
	}
	for _, slug := range []string{"philosophy", "strategy"} {
		if !nsInBox(&s, nsTokenID(t, slug)) {
			t.Fatalf("%s was not returned to the box: %v", slug, s.BoxTokens)
		}
	}
	if s.Library != ([engine.LibraryDraw]catalog.TokenID{}) {
		t.Fatalf("the library offer was not cleared: %v", s.Library)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, toMove %d; want seat 1's turn", s.Phase, s.ToMove)
	}
}

func TestNamedScenarioZeusDestroy(t *testing.T) {
	s, ev := nsRun(t, "zeus-destroy", 104)
	quarry := nsID(t, "quarry")
	if s.Players[1].City.Has(quarry) {
		t.Fatalf("seat 1 still has quarry")
	}
	if !s.Discard.Has(quarry) {
		t.Fatalf("quarry is not in the discard pile")
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvDestroy, Seat: 0, Card: quarry}) {
		t.Fatalf("events %+v: want EvDestroy", ev)
	}
	if s.Military != 1 {
		t.Fatalf("military %d, want 1", s.Military)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, toMove %d; want seat 1's turn", s.Phase, s.ToMove)
	}
}

func TestNamedScenarioZeusNoTarget(t *testing.T) {
	s, ev := nsRun(t, "zeus-no-target", 105)
	want := engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Wonder: nsWonderID(t, "statue-of-zeus"), A: int16(engine.PhaseDestroyCard)}
	if !nsHasEvent(ev, want) {
		t.Fatalf("events %+v: want %+v", ev, want)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, toMove %d; want seat 1's turn", s.Phase, s.ToMove)
	}
}

func TestNamedScenarioCircusMaximus(t *testing.T) {
	s, ev := nsRun(t, "circus-maximus", 106)
	glassworks := nsID(t, "glassworks")
	if s.Players[1].City.Has(glassworks) {
		t.Fatalf("seat 1 still has glassworks")
	}
	if !s.Discard.Has(glassworks) {
		t.Fatalf("glassworks is not in the discard pile")
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvDestroy, Seat: 0, Card: glassworks}) {
		t.Fatalf("events %+v: want EvDestroy", ev)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, toMove %d; want seat 1's turn", s.Phase, s.ToMove)
	}
}

func TestNamedScenarioSciencePair(t *testing.T) {
	s, ev := nsRun(t, "science-pair", 107)
	if s.Over() || s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, over %v; want the board token choice open", s.Phase, s.Over())
	}
	if s.ToMove != 0 {
		t.Fatalf("toMove %d, want seat 0", s.ToMove)
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvPair, Seat: 0, A: int16(catalog.Quill)}) {
		t.Fatalf("events %+v: want EvPair of quill", ev)
	}
	if n := s.BoardTokens.Count(); n != engine.NumBoardTokens {
		t.Fatalf("board tokens %d, want all %d untouched", n, engine.NumBoardTokens)
	}
}

func TestNamedScenarioScienceSupremacy(t *testing.T) {
	s, _ := nsRun(t, "science-supremacy", 108)
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
}

func TestNamedScenarioLawSupremacy(t *testing.T) {
	s, ev := nsRun(t, "law-supremacy", 109)
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	if !s.Players[0].Tokens.Has(nsTokenID(t, "law")) {
		t.Fatalf("seat 0 does not hold law: %v", s.Players[0].Tokens)
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvPair, Seat: 0, A: int16(catalog.Sundial)}) {
		t.Fatalf("events %+v: want EvPair of sundial before the token pick", ev)
	}
}

func TestNamedScenarioMilitarySupremacy(t *testing.T) {
	s, _ := nsRun(t, "military-supremacy", 110)
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("phase %s, result %+v, military %d", s.Phase, s.Result, s.Military)
	}
}

func TestNamedScenarioMilitaryViaWonder(t *testing.T) {
	s, ev := nsRun(t, "military-via-wonder", 111)
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("phase %s, result %+v, military %d", s.Phase, s.Result, s.Military)
	}
	if nsHasKind(ev, engine.EvDestroy) {
		t.Fatalf("events %+v: the destroy choice should never have resolved", ev)
	}
	for _, e := range ev {
		if e.Kind == engine.EvEffectSkipped && e.A == int16(engine.PhaseDestroyCard) {
			t.Fatalf("events %+v: the destroy choice should never have been reached, skipped or not", ev)
		}
	}
}

func TestNamedScenarioLootingDouble(t *testing.T) {
	s, ev := nsRun(t, "looting-double", 112)
	if s.Over() || s.Military != 6 {
		t.Fatalf("over %v, military %d, want 6 and not over", s.Over(), s.Military)
	}
	if s.Players[1].Coins != 13 {
		t.Fatalf("seat 1's coins %d, want 13 (20 - 2 - 5)", s.Players[1].Coins)
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvLoot, Seat: 1, A: 2, B: 2}) {
		t.Fatalf("events %+v: want the 2-coin looting token taken", ev)
	}
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvLoot, Seat: 1, A: 5, B: 5}) {
		t.Fatalf("events %+v: want the 5-coin looting token taken", ev)
	}
}

func TestNamedScenarioCivilianEnd(t *testing.T) {
	s, _ := nsRun(t, "civilian-end", 113)
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	sc := s.Result.Scores
	if sc[0].Total == sc[1].Total {
		t.Fatalf("scores are tied, want a clean win: %+v", sc)
	}
}

func TestNamedScenarioTieBlue(t *testing.T) {
	s, _ := nsRun(t, "tie-blue", 114)
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	sc := s.Result.Scores
	if sc[0].Total != sc[1].Total {
		t.Fatalf("totals are not tied, want the tie-break exercised: %+v", sc)
	}
	if sc[0].Blue <= sc[1].Blue {
		t.Fatalf("seat 0's blue points do not break the tie: %+v", sc)
	}
}

func TestNamedScenarioDraw(t *testing.T) {
	s, _ := nsRun(t, "draw", 115)
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != engine.SharedVictory {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	sc := s.Result.Scores
	if sc[0].Total != sc[1].Total || sc[0].Blue != sc[1].Blue {
		t.Fatalf("scores are not a full tie: %+v", sc)
	}
}

func TestNamedScenarioTheologyChain(t *testing.T) {
	s, _ := nsRun(t, "theology-chain", 116)
	p := nsWonder(t, &s.Players[0], "pyramids")
	if !p.Built || p.Tucked != nsID(t, "rostrum") {
		t.Fatalf("pyramids: %+v", p)
	}
	c := nsWonder(t, &s.Players[0], "colossus")
	if !c.Built || c.Tucked != nsID(t, "statue") {
		t.Fatalf("colossus: %+v", c)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 || s.ExtraTurn {
		t.Fatalf("phase %s, toMove %d, active %d, extraTurn %v; want seat 0 still to move",
			s.Phase, s.ToMove, s.Active, s.ExtraTurn)
	}
}

func TestNamedScenarioAppianEconomy(t *testing.T) {
	s, _ := nsRun(t, "appian-economy", 117)
	if s.Players[0].Coins != 8 {
		t.Fatalf("seat 0's coins %d, want 8 (15 - 10 trade + 3 of its own)", s.Players[0].Coins)
	}
	if s.Players[1].Coins != 7 {
		t.Fatalf("seat 1's coins %d, want 7 (0 + 10 trade - 3 lost, never floored to 0)", s.Players[1].Coins)
	}
}

func TestNamedScenarioAgeTransitionChooser(t *testing.T) {
	s, ev := nsRun(t, "age-transition-chooser", 118)
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvStarterChosen, Seat: 1, A: 0}) {
		t.Fatalf("events %+v: want seat 1, the trailing side, choosing seat 0 to start", ev)
	}
	if s.Phase != engine.PhasePlay || s.Age != 2 || s.Active != 0 || s.ToMove != 0 {
		t.Fatalf("phase %s, age %d, active %d, toMove %d; want age 2, seat 0 to move",
			s.Phase, s.Age, s.Active, s.ToMove)
	}
}

func TestNamedScenarioExtraTurnLost(t *testing.T) {
	s, ev := nsRun(t, "extra-turn-lost", 119)
	if !nsHasEvent(ev, engine.Event{Kind: engine.EvExtraTurnLost, Seat: 0}) {
		t.Fatalf("events %+v: want the extra turn lost", ev)
	}
	w := nsWonder(t, &s.Players[0], "sphinx")
	if !w.Built || w.Tucked != nsID(t, "theater") {
		t.Fatalf("sphinx: %+v", w)
	}
	if s.Phase != engine.PhaseChooseStarter || s.Age != 2 || s.ToMove != 0 {
		t.Fatalf("phase %s, age %d, toMove %d; want age 2's starter choice, seat 0 to decide",
			s.Phase, s.Age, s.ToMove)
	}
}

// TestNamedScenarioNames checks the dev endpoint's catalog: every scenario
// this file tests is listed, sorted, and resolves; nothing else does.
func TestNamedScenarioNames(t *testing.T) {
	want := []string{
		"age-transition-chooser", "appian-economy", "civilian-end", "circus-maximus",
		"draw", "extra-turn-lost", "great-library", "law-supremacy",
		"looting-double", "mausoleum", "mausoleum-empty", "military-supremacy",
		"military-via-wonder", "science-pair", "science-supremacy", "theology-chain",
		"tie-blue", "zeus-destroy", "zeus-no-target",
	}
	sort.Strings(want)
	got := engine.NamedScenarioNames()
	if !slices.Equal(got, want) {
		t.Fatalf("names:\n  %v\nwant\n  %v", got, want)
	}
	if !slices.IsSorted(got) {
		t.Fatalf("names are not sorted: %v", got)
	}
	for _, name := range got {
		if engine.NamedScenario(name) == nil {
			t.Fatalf("%q is listed but NamedScenario returns nil", name)
		}
	}
	if engine.NamedScenario("not-a-named-scenario") != nil {
		t.Fatalf("an unknown name should return nil")
	}
}
