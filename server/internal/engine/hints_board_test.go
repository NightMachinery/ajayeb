package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// Tests of the board-level hints (hints_board.go, M10.HS4), written from
// docs/rules/military-science.md, wonders-tokens.md, ages-scoring.md and
// turn.md. Helpers are prefixed hb.

// hbHints returns the board hints and the hints about each seat of s, with
// every family on, as the views carry them.
func hbHints(t testing.TB, s *State) (view.HintsView, [2]view.PlayerHintsView) {
	t.Helper()
	h := ComputeHints(s, hnAllHints)
	vw := ViewWithHints(s, Spectator, &h)
	if vw.Hints == nil {
		t.Fatal("no hints in the view")
	}
	var ps [2]view.PlayerHintsView
	for seat := range ps {
		if vw.Players[seat].Hints == nil {
			t.Fatalf("seat %d has no hints", seat)
		}
		ps[seat] = *vw.Players[seat].Hints
	}
	return *vw.Hints, ps
}

// hbSymbol returns seat's hint about one symbol.
func hbSymbol(t testing.TB, ph view.PlayerHintsView, sym string) view.SymbolHint {
	t.Helper()
	for _, s := range ph.Symbols {
		if string(s.Symbol) == sym {
			return s
		}
	}
	t.Fatalf("no symbol %q in %+v", sym, ph.Symbols)
	return view.SymbolHint{}
}

// TestSymbolStates covers the symbol states of one seat
// (military-science.md "Science symbols", "Pairs"): a pair, a single whose
// partner is face up (live), a single whose partner is tucked under a
// wonder (dead: a tucked card never counts), a missing symbol whose cards
// may come (open) and one whose cards are both gone (dead).
func TestSymbolStates(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     2,
		Wonders: wonderFill([]string{"pyramids"}, []string{"sphinx"}),
		Players: [2]ScenarioPlayer{
			{City: []string{"scriptorium", "pharmacist", "dispensary", "workshop"}, Built: []ScenarioWonder{{Wonder: "pyramids", Tucked: "laboratory"}}},
			{City: []string{"apothecary"}, Built: []ScenarioWonder{{Wonder: "sphinx", Tucked: "school"}}},
		},
		BoardTokens: []string{"law", "agriculture", "economy", "masonry", "mathematics"},
		Accessible:  []string{"library"},
	})
	_, ps := hbHints(t, &s)
	want := map[string]view.SymbolState{
		"quill":      view.SymbolStateLive,   // Scriptorium, and Library face up
		"mortar":     view.SymbolStatePaired, // Pharmacist and Dispensary
		"set-square": view.SymbolStateOwned,  // Workshop, and Laboratory tucked: dead
		"wheel":      view.SymbolStateDead,   // Apothecary in Moon's city, School tucked
		"globe":      view.SymbolStateOpen,   // both Age III
		"sundial":    view.SymbolStateOpen,
	}
	for sym, w := range want {
		if got := hbSymbol(t, ps[0], sym); got.State != w {
			t.Errorf("seat 0 %s: %s, want %s", sym, got.State, w)
		}
	}
	if got := hbSymbol(t, ps[0], "quill").OpenCards; got != 1 {
		t.Errorf("quill open cards %d, want 1", got)
	}
	if got := hbSymbol(t, ps[1], "wheel").State; got != view.SymbolStateOwned {
		t.Errorf("seat 1 wheel: %s, want owned (School tucked)", got)
	}
	if got := hbSymbol(t, ps[1], "quill").State; got != view.SymbolStateOpen {
		t.Errorf("seat 1 quill: %s, want open (Library face up)", got)
	}
	// Seat 0 owns three symbols, so it needs 3 more: globe and sundial may
	// still come, and the scales through Law on the board, since a pair
	// (quill) is live.
	if got := hbSymbol(t, ps[0], "scales").State; got != view.SymbolStateOpen {
		t.Errorf("seat 0 scales %s, want open", got)
	}
	if ps[0].Need != 3 || !ps[0].Reachable {
		t.Errorf("seat 0 need %d reachable %v, want 3 true", ps[0].Need, ps[0].Reachable)
	}
}

// TestSymbolDeadAfterAge: a partner never seen in an age that is over is
// certainly set aside, so a single of that symbol is dead (setup.md "What
// is hidden from both players"), in a scenario and in real games.
func TestSymbolDeadAfterAge(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     2,
		Players: [2]ScenarioPlayer{{City: []string{"library"}}},
	})
	if p := publicPlace(&s, hnCard(t, "scriptorium")).Place; p != PlaceRemovedKnown {
		t.Fatalf("scriptorium: %v, want removedKnown", p)
	}
	_, ps := hbHints(t, &s)
	if got := hbSymbol(t, ps[0], "quill").State; got != view.SymbolStateOwned {
		t.Errorf("quill with Scriptorium gone: %s, want owned (dead partner)", got)
	}

	// Real games: whenever a seat owns one card of a symbol whose other card
	// is in a past age's set-aside cards, the state is owned, never live.
	checked := 0
	for seed := uint64(1); seed <= 40; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		for !s.Over() {
			if s.Age >= 2 && s.Phase == PhasePlay && s.Ply%4 == 0 {
				_, ps := hbHints(t, &s)
				for seat := range Seat(2) {
					for _, c := range s.Removed[0] {
						sym := catalog.T.Sym[c]
						if c == 0 || sym == catalog.NoSymbol || s.Players[seat].Sci[sym] != 1 {
							continue
						}
						checked++
						if got := hbSymbol(t, ps[seat], sym.String()).State; got != view.SymbolStateOwned {
							t.Errorf("seed %d ply %d: seat %d %s with %s set aside in Age I: %s, want owned", seed, s.Ply, seat, sym, catalog.Card(c).Slug, got)
						}
					}
				}
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
	}
	if checked == 0 {
		t.Fatal("no set-aside partner checked")
	}
	t.Logf("%d set-aside partners checked", checked)
}

// TestScalesNeedLivePair: Law on the board can only be taken through a
// pair, so the scales are open only while the seat has a pair it may still
// complete (military-science.md "Pairs"), and Law wins only then.
func TestScalesNeedLivePair(t *testing.T) {
	five := []string{"scriptorium", "pharmacist", "workshop", "apothecary", "study"}
	board := []string{"agriculture", "architecture", "economy", "masonry", "law"}
	// Academy (the other sundial) face up: a live pair.
	live := hnNew(t, Scenario{
		Age: 3, Players: [2]ScenarioPlayer{{City: five}}, BoardTokens: board, Accessible: []string{"academy"},
	})
	_, ps := hbHints(t, &live)
	if got := hbSymbol(t, ps[0], "scales"); got.State != view.SymbolStateOpen || got.OpenCards != 1 {
		t.Errorf("scales with a live pair: %+v, want open", got)
	}
	if !ps[0].LawWins || ps[0].Need != 1 || !ps[0].Reachable {
		t.Errorf("live pair: lawWins %v need %d reachable %v", ps[0].LawWins, ps[0].Need, ps[0].Reachable)
	}
	// Every partner gone (the Age I and II partners were never placed, so
	// they are out of play, and Academy is in Moon's city), and one globe
	// is in Moon's city too: no pair can complete, so Law cannot come.
	dead := hnNew(t, Scenario{
		Age: 3, Players: [2]ScenarioPlayer{{City: five}, {City: []string{"academy", "university"}}}, BoardTokens: board,
	})
	_, ps = hbHints(t, &dead)
	if got := hbSymbol(t, ps[0], "scales"); got.State != view.SymbolStateDead {
		t.Errorf("scales without a live pair: %+v, want dead", got)
	}
	if ps[0].LawWins {
		t.Error("Law wins without a live pair")
	}
	for _, sym := range []string{"quill", "mortar", "set-square", "wheel", "sundial"} {
		if got := hbSymbol(t, ps[0], sym).State; got != view.SymbolStateOwned {
			t.Errorf("%s: %s, want owned", sym, got)
		}
	}
}

// TestCapitalSafeStrategy: the bound of the shields the attacker can still
// place counts Strategy's extra shield on every red building that may come
// unless the defender owns Strategy (wonders-tokens.md "The ten tokens"),
// so capital-safe never claims safety while Strategy is possible.
func TestCapitalSafeStrategy(t *testing.T) {
	mk := func(pawn int, defenderStrategy bool) State {
		sc := Scenario{
			Age:         3,
			Wonders:     wonderFill([]string{"pyramids"}, nil),
			BoardTokens: []string{"agriculture", "architecture", "economy", "masonry", "strategy"},
			OnlyLeft:    []string{"arsenal", "senate"},
			Pawn:        &pawn,
		}
		if defenderStrategy {
			sc.BoardTokens = []string{"agriculture", "architecture", "economy", "masonry", "mathematics"}
			sc.Players[1].Tokens = []string{"strategy"}
		}
		return hnNew(t, sc)
	}
	with := shieldsStillPossible(ptr(mk(0, false)), 0)
	without := shieldsStillPossible(ptr(mk(0, true)), 0)
	reds := 0
	s := mk(0, false)
	for c, ok := catalog.T.ByColor[catalog.Red].Next(1); ok; c, ok = catalog.T.ByColor[catalog.Red].Next(c + 1) {
		if catalog.T.Shields[c] > 0 && mayComeTo(&s, 0, publicPlace(&s, c)) {
			reds++
		}
	}
	if reds == 0 || with != without+reds {
		t.Fatalf("bound with Strategy possible %d, with the defender owning it %d, red cards that may come %d", with, without, reds)
	}
	if with > 17 {
		t.Skipf("bound %d too large for a scenario pawn", with)
	}
	pawn := MilitaryCapital - with // seat 1's capital is exactly `with` away
	_, ps := hbHints(t, ptr(mk(pawn, false)))
	if ps[1].ToCapital != with || ps[1].CapitalSafe {
		t.Errorf("Strategy possible: toCapital %d (want %d), safe %v (want false)", ps[1].ToCapital, with, ps[1].CapitalSafe)
	}
	_, ps = hbHints(t, ptr(mk(pawn, true)))
	if !ps[1].CapitalSafe {
		t.Errorf("defender owns Strategy: bound %d < %d, want safe", without, with)
	}
}

func ptr(s State) *State { return &s }

// hbSym returns the science symbol of a wire name.
func hbSym(name string) catalog.Symbol {
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		if sym.String() == name {
			return sym
		}
	}
	return catalog.NoSymbol
}

// TestBoardHintsSound plays random games to the end and checks, at every
// state, that each claim of the board hints holds for the rest of that
// game: a capital called safe is never taken, a seat whose science is not
// reachable never wins by science, a dead symbol never enters that seat's
// science, and a seat that Law would make win has a pair it can complete.
func TestBoardHintsSound(t *testing.T) {
	games := 40
	if testing.Short() {
		games = 10
	}
	claims := 0
	for g := range games {
		seed := uint64(500 + g)
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		var safe, unreachable [2]bool
		var dead [2][catalog.NumSymbols]bool
		var why [2][catalog.NumSymbols]string
		for !s.Over() {
			if s.Phase != PhaseWonderDraft && s.Ply%3 == 0 {
				_, ps := hbHints(t, &s)
				for seat := range ps {
					safe[seat] = safe[seat] || ps[seat].CapitalSafe
					if !ps[seat].Reachable && ps[seat].Need > 0 {
						unreachable[seat] = true
					}
					for _, sh := range ps[seat].Symbols {
						if sh.State == view.SymbolStateDead && !dead[seat][hbSym(string(sh.Symbol))] {
							sym := hbSym(string(sh.Symbol))
							dead[seat][sym] = true
							cards, n := pairCards(sym)
							why[seat][sym] = fmt.Sprintf("ply %d (%s)", s.Ply, s.Phase)
							for _, c := range cards[:n] {
								pl := publicPlace(&s, c)
								why[seat][sym] += fmt.Sprintf(" %s:%s/%d", catalog.Card(c).Slug, pl.Place, pl.Seat)
							}
							claims++
						}
					}
				}
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
			for seat := range Seat(2) {
				for sym := range dead[seat] {
					if dead[seat][sym] && s.Players[seat].Sci[sym] > 0 {
						t.Fatalf("seed %d ply %d: seat %d gained %s, called dead at %s", seed, s.Ply, seat, catalog.Symbol(sym), why[seat][sym])
					}
				}
			}
		}
		if w := s.Result.Winner; w == 0 || w == 1 {
			winner := Seat(w)
			if s.Result.Kind == VictoryMilitary && safe[winner.Opp()] {
				t.Errorf("seed %d: seat %d's capital was called safe, then taken", seed, winner.Opp())
			}
			if s.Result.Kind == VictoryScience && unreachable[winner] {
				t.Errorf("seed %d: seat %d won by science after it was called unreachable", seed, winner)
			}
		}
	}
	if claims == 0 {
		t.Fatal("no dead symbol was ever claimed")
	}
	t.Logf("%d dead-symbol claims held", claims)
}

// TestGuildCandidates: in Age III the guild back's pool and the guild
// preview always include the guilds never put in the game, so neither tells
// which guilds are in it (decisions.md "Guild back"), and the pool's size
// less the face-down guild backs is the set-aside count.
func TestGuildCandidates(t *testing.T) {
	checked := 0
	for seed := uint64(1); seed <= 30; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		for !s.Over() {
			if s.Age == NumAges && s.Phase == PhasePlay {
				b, _ := hbHints(t, &s)
				var preview []string
				for _, g := range b.GuildPreview {
					preview = append(preview, g.Guild)
				}
				var pool *view.UnseenView
				for i := range b.Unseen {
					if b.Unseen[i].Back == view.BackGuild {
						pool = &b.Unseen[i]
					}
				}
				for _, c := range s.BoxGuilds {
					slug := catalog.Card(c).Slug
					if !slices.Contains(preview, slug) {
						t.Fatalf("seed %d ply %d: box guild %s missing from the preview %v", seed, s.Ply, slug, preview)
					}
					if pool != nil && !slices.Contains(pool.Cards, slug) {
						t.Fatalf("seed %d ply %d: box guild %s missing from the guild back's pool %v", seed, s.Ply, slug, pool.Cards)
					}
				}
				if pool != nil && pool.SetAside != NumBoxGuilds {
					t.Fatalf("seed %d ply %d: guild pool %+v, want %d set aside", seed, s.Ply, pool, NumBoxGuilds)
				}
				checked++
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
	}
	if checked == 0 {
		t.Fatal("no Age III state checked")
	}
}

// TestLastCardTo checks lastCardTo against the rest of each age in random
// games: from every play state and every pending choice, when nothing
// grants an extra turn until the age's last card is taken, that card goes
// to the seat lastCardTo named. While a new age's starter is chosen it
// names no one (turn.md "Whose turn it is", "Extra turns").
func TestLastCardTo(t *testing.T) {
	var pending, play, starter int
	for seed := uint64(1); seed <= 40; seed++ {
		var states []State
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		for !s.Over() {
			if s.Age >= 1 {
				states = append(states, s)
			}
			legal := LegalActions(&s, buf[:0])
			// Prefer actions that grant no extra turn, so most playouts check.
			a := legal[r.IntN(len(legal))]
			ApplyTrusted(&s, a)
		}
		for _, x := range states {
			got, ok := lastCardTo(&x)
			if x.Phase == PhaseChooseStarter {
				if ok {
					t.Fatalf("seed %d ply %d: a starter choice names %d", seed, x.Ply, got)
				}
				starter++
				continue
			}
			if !ok {
				continue
			}
			taker, clean := hbLastTaker(&x, seed)
			if !clean {
				continue
			}
			if taker != got {
				t.Fatalf("seed %d ply %d (%s): lastCardTo %d, the last card went to %d", seed, x.Ply, x.Phase, got, taker)
			}
			if x.Phase == PhasePlay {
				play++
			} else {
				pending++
			}
		}
	}
	if pending == 0 || play == 0 || starter == 0 {
		t.Fatalf("checked %d play, %d pending and %d starter states; each kind needs one", play, pending, starter)
	}
	t.Logf("checked %d play, %d pending and %d starter states", play, pending, starter)
}

// hbLastTaker plays the rest of x's age with random actions that grant no
// extra turn, and returns the seat that took the age's last card; false
// when no such playout exists (only extra-turn actions were legal) or the
// game ended first.
func hbLastTaker(x *State, seed uint64) (Seat, bool) {
	s := *x
	r := NewRNG(seed ^ uint64(s.Ply))
	buf := make([]Action, 0, MaxLegalActions)
	age := s.Age
	for !s.Over() && s.Age == age && s.Phase != PhaseChooseStarter {
		legal := LegalActions(&s, buf[:0])
		var ok []Action
		for _, a := range legal {
			if st, _ := extraTurnOutcome(&s, a); st != ExtraTurnGain {
				ok = append(ok, a)
			}
		}
		if len(ok) == 0 {
			return 0, false
		}
		a := ok[r.IntN(len(ok))]
		mover := s.ToMove
		last := s.Phase == PhasePlay && Occupied(&s)&(Occupied(&s)-1) == 0
		ApplyTrusted(&s, a)
		if last {
			return mover, true
		}
	}
	return 0, false
}

// TestLibraryTokenValuesDecider: during the Great Library offer only the
// seat choosing from it gets the offer's values; the other seat and a
// spectator get none, and nobody gets threats (the phase gate).
func TestLibraryTokenValuesDecider(t *testing.T) {
	sc := NamedScenario("great-library")
	s, err := NewGameFromScenario(3, Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	hnDo(t, &s, sc.Trigger[0])
	if s.Phase != PhasePickLibraryToken {
		t.Fatalf("phase %s", s.Phase)
	}
	h := ComputeHints(&s, hnAllHints)
	for _, v := range hlViewers {
		vw := ViewWithHints(&s, v, &h)
		for seat := range vw.Players {
			if ph := vw.Players[seat].Hints; ph == nil || ph.Threats != nil {
				t.Errorf("viewer %d: seat %d's threats during the offer: %+v", v, seat, ph)
			}
		}
		got := vw.Pending.TokenValues
		if seat, ok := v.Seat(); ok && seat == s.ToMove {
			if got == nil || len(*got) != 3 {
				t.Fatalf("the decider's token values: %+v", got)
			}
			law := slices.IndexFunc(*got, func(x view.TokenValueView) bool { return x.Token == "law" })
			if law < 0 || (*got)[law].Seat != int(seat) {
				t.Errorf("no Law for the decider among the values %+v", *got)
			}
			continue
		}
		if got != nil {
			t.Errorf("viewer %d sees the offer's values %+v", v, *got)
		}
		b, err := json.Marshal(vw.Pending)
		if err != nil {
			t.Fatal(err)
		}
		if len(vw.Pending.Tokens) != 0 || slices.Contains(vw.Pending.Tokens, "law") || len(b) == 0 {
			t.Errorf("viewer %d sees the offer %v", v, vw.Pending.Tokens)
		}
	}
	// The box is named by elimination for everyone, the same list.
	b0, _ := hbHints(t, &s)
	if len(b0.BoxTokens) != 0 {
		// The Great Library is built, so no seat has one open: not sent.
		t.Errorf("box tokens named with no Great Library open: %v", b0.BoxTokens)
	}
}

// TestBoxTokensNamed: while a seat's Great Library is open, the box is
// named by elimination (owner answer Q1 B), in id order, never in draw
// order.
func TestBoxTokensNamed(t *testing.T) {
	sc := NamedScenario("great-library")
	s, err := NewGameFromScenario(3, Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	b, ps := hbHints(t, &s)
	want := []string{}
	for tk := catalog.TokenID(1); tk <= catalog.NumTokens; tk++ {
		if !s.BoardTokens.Has(tk) && !s.Players[0].Tokens.Has(tk) && !s.Players[1].Tokens.Has(tk) {
			want = append(want, catalog.Token(tk).Slug)
		}
	}
	if !slices.Equal(b.BoxTokens, want) {
		t.Errorf("box %v, want %v", b.BoxTokens, want)
	}
	// Law is in the box and seat 0's Great Library is open: the scales may
	// still come to seat 0, never to seat 1.
	if got := hbSymbol(t, ps[0], "scales").State; got != view.SymbolStateOpen {
		t.Errorf("seat 0 scales %s, want open (its Great Library may draw Law)", got)
	}
	if got := hbSymbol(t, ps[1], "scales").State; got != view.SymbolStateDead {
		t.Errorf("seat 1 scales %s, want dead", got)
	}
}

// TestWonderEffectView: an open wonder's effect names its extra turn, the
// pawn move and its special choice, and the wonders left; a built wonder
// carries none.
func TestWonderEffectView(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     2,
		Wonders: wonderFill([]string{"colossus", "piraeus"}, []string{"great-library"}),
		Players: [2]ScenarioPlayer{{Coins: intPtr(30), Built: []ScenarioWonder{{Wonder: "piraeus", Tucked: "baths"}}}, {Coins: intPtr(30)}},
	})
	h := ComputeHints(&s, hnAllHints)
	vw := ViewWithHints(&s, SeatViewer(0), &h)
	if vw.Hints.WondersLeft != MaxWondersBuilt-int(s.WondersBuilt) {
		t.Errorf("wondersLeft %d", vw.Hints.WondersLeft)
	}
	for _, w := range vw.Players[0].Wonders {
		switch {
		case w.Built && w.Effect != nil:
			t.Errorf("built %s has an effect", w.Wonder)
		case w.Wonder == "colossus":
			if w.Effect == nil || w.Effect.Pawn == nil || w.Effect.Pawn.To != int(s.Military)+2 {
				t.Errorf("colossus: %+v", w.Effect)
			}
		}
	}
	for _, w := range vw.Players[1].Wonders {
		if w.Wonder == "great-library" {
			if w.Effect == nil || w.Effect.Opens == nil || *w.Effect.Opens != view.PendingKindPickLibraryToken || w.Effect.Pool != 3 {
				t.Errorf("great library: %+v", w.Effect)
			}
		}
	}
}

// TestLootAndCapital: each looting token says what it would take now (no
// more than the victim has), and toCapital counts from the pawn.
func TestLootAndCapital(t *testing.T) {
	pawn := 2
	s := hnNew(t, Scenario{Age: 1, Pawn: &pawn, Players: [2]ScenarioPlayer{{Coins: intPtr(1)}, {Coins: intPtr(4)}}})
	b, ps := hbHints(t, &s)
	got := map[int]int{}
	for _, l := range b.Looting {
		got[l.At] = l.WouldLose
	}
	want := map[int]int{-6: 1, -3: 1, 3: 2, 6: 4}
	for at, w := range want {
		if got[at] != w {
			t.Errorf("token at %d: would lose %d, want %d", at, got[at], w)
		}
	}
	if ps[1].ToCapital != 7 || ps[0].ToCapital != 11 {
		t.Errorf("toCapital %d and %d, want 11 and 7", ps[0].ToCapital, ps[1].ToCapital)
	}
}

// TestHintsLeakGuardCoverage makes sure the leak guard's corpus
// (hlCorpus, TestHintsLeakGuard) reaches every part of the board-level
// hints with something to compare: a guard over empty fields would prove
// nothing.
func TestHintsLeakGuardCoverage(t *testing.T) {
	var threats, pairs, tokens, box, guilds, unseen, library, effects, lastCard, chains int
	for _, s := range hlCorpus(t, 6) {
		if s.Over() || s.Phase == PhaseWonderDraft {
			continue
		}
		h := ComputeHints(&s, hnAllHints)
		vw := ViewWithHints(&s, SeatViewer(s.ToMove), &h)
		b := vw.Hints
		for _, p := range vw.Players {
			if p.Hints.Threats != nil && len(*p.Hints.Threats) > 0 {
				threats++
			}
			chains += len(p.Hints.ChainTargets)
			for _, w := range p.Wonders {
				if w.Effect != nil {
					effects++
				}
			}
		}
		pairs += len(b.Pairs)
		tokens += len(b.Tokens)
		box += len(b.BoxTokens)
		guilds += len(b.GuildPreview)
		unseen += len(b.Unseen)
		if b.LastCardTo != nil {
			lastCard++
		}
		if vw.Pending != nil && vw.Pending.TokenValues != nil {
			library++
		}
	}
	for name, n := range map[string]int{
		"threats": threats, "pairs": pairs, "tokens": tokens, "boxTokens": box, "guildPreview": guilds,
		"unseen": unseen, "tokenValues": library, "wonder effects": effects, "lastCardTo": lastCard, "chainTargets": chains,
	} {
		if n == 0 {
			t.Errorf("the leak guard's corpus never reaches %s", name)
		}
	}
}
