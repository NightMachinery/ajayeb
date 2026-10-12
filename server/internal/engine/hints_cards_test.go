package engine

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The tests of the per-card hints (hints.go, "Per-card hints", M10.HS2):
// each condition against the rules it encodes, for both seats, on covered
// cards, with an empty token board and on the Mausoleum's pile; the family
// switches; and the view: the hints are the same in the three views and
// absent once the game is over. Helpers are prefixed hc.

// hcSlotHints returns the per-card hints of card slug in the view of s
// built with the settings f, for viewer v.
func hcSlotHints(t *testing.T, s *State, f view.HintSettings, v Viewer, slug string) *[2]view.CardHints {
	t.Helper()
	h := ComputeHints(s, f)
	vw := ViewWithHints(s, v, &h)
	for _, sv := range vw.Structure {
		if sv.Card != nil && *sv.Card == slug {
			return sv.Hints
		}
	}
	t.Fatalf("%s is not face up", slug)
	return nil
}

// hcEff returns a card's effect, or the zero effect when none was sent.
func hcEff(ch view.CardHints) view.EffectView {
	if ch.Effect == nil {
		return view.EffectView{}
	}
	return *ch.Effect
}

// hcCodes returns a card's codes for a seat as strings.
func hcCodes(ch view.CardHints) []string {
	out := []string{}
	for _, c := range ch.Codes {
		out = append(out, string(c))
	}
	return out
}

// hcCovered moves card slug into a face-up slot that is still covered,
// swapping it with that slot's card, and returns the slot.
func hcCovered(t *testing.T, s *State, slug string) int {
	t.Helper()
	c := hnCard(t, slug)
	from := SlotOf(s, c)
	occ := Occupied(s)
	for i := range s.Slots {
		if i != from && s.Slots[i].FaceUp && s.Slots[i].Card != 0 && CoverMask(s.Age, i)&occ != 0 {
			s.Slots[i].Card, s.Slots[from].Card = s.Slots[from].Card, s.Slots[i].Card
			if err := CheckInvariants(s); err != nil {
				t.Fatal(err)
			}
			return i
		}
	}
	t.Fatal("no covered face-up slot")
	return -1
}

// TestCardHintsPair: a card that completes a seat's science pair while a
// token is on the board earns that seat the pair pip, solid for the seat
// it helps and seen as the opponent's by the other; the same on a covered
// card, as if it were accessible; and no pip once the board is empty,
// though the effect still says it pairs (military-science.md "Pairs").
func TestCardHintsPair(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"scriptorium"}, Coins: intPtr(20)}, {City: []string{"pharmacist"}, Coins: intPtr(20)}},
		Accessible: []string{"library", "dispensary"},
		ToMove:     seatPtr(0),
	})
	lib := hcSlotHints(t, &s, hnAllHints, 0, "library")
	if lib == nil {
		t.Fatal("no hints on the Library")
	}
	if got := hcCodes(lib[0]); !slices.Equal(got, []string{"pair"}) {
		t.Fatalf("seat 0's codes on the Library: %v, want [pair]", got)
	}
	if sc := hcEff(lib[0]).Science; sc == nil || sc.Symbol != "quill" || sc.Kind != "pair" || !sc.TokenOnBoard || sc.DistinctAfter != 1 {
		t.Fatalf("seat 0's science effect: %+v", sc)
	}
	if got := hcCodes(lib[1]); len(got) != 0 {
		t.Fatalf("seat 1's codes on the Library: %v", got)
	}
	if sc := hcEff(lib[1]).Science; sc == nil || sc.Kind != "new" || sc.DistinctAfter != 2 {
		t.Fatalf("seat 1's science effect: %+v", sc)
	}
	// The Dispensary pairs seat 1's mortar: the opponent's pair, a deny
	// target for seat 0.
	disp := hcSlotHints(t, &s, hnAllHints, 0, "dispensary")
	if got := hcCodes(disp[1]); !slices.Equal(got, []string{"pair"}) || len(disp[0].Codes) != 0 {
		t.Fatalf("the Dispensary: seat 0 %v, seat 1 %v", hcCodes(disp[0]), hcCodes(disp[1]))
	}

	covered := s
	i := hcCovered(t, &covered, "library")
	if covered.Slots[i].Card != hnCard(t, "library") {
		t.Fatal("the swap")
	}
	if got := hcCodes(hcSlotHints(t, &covered, hnAllHints, 1, "library")[0]); !slices.Equal(got, []string{"pair"}) {
		t.Fatalf("a covered Library: %v", got)
	}

	empty := s
	empty.BoardTokens = 0
	e := hcSlotHints(t, &empty, hnAllHints, 0, "library")
	if len(e[0].Codes) != 0 || hcEff(e[0]).Science == nil || hcEff(e[0]).Science.Kind != "pair" || hcEff(e[0]).Science.TokenOnBoard {
		t.Fatalf("an empty board: codes %v, science %+v", hcCodes(e[0]), hcEff(e[0]).Science)
	}
}

// TestCardHintsWin: a pair while Law is on the board and the seat holds
// five different symbols wins through the token (the win pip beside the
// pair), and a card that pushes the pawn into the capital wins at once
// (military-science.md "Scientific supremacy", "Military supremacy"). An
// unaffordable winning card shows the win in its effect but earns no pip.
func TestCardHintsWin(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:         2,
		Players:     [2]ScenarioPlayer{{City: []string{"scriptorium", "workshop", "pharmacist", "apothecary", "university"}, Coins: intPtr(30)}},
		BoardTokens: []string{"law", "agriculture", "philosophy", "masonry", "strategy"},
		Accessible:  []string{"library"},
		ToMove:      seatPtr(1),
	})
	lib := hcSlotHints(t, &s, hnAllHints, 1, "library")
	if got := hcCodes(lib[0]); !slices.Equal(got, []string{"win", "pair"}) {
		t.Fatalf("seat 0's codes: %v, want [win pair]", got)
	}
	w := hcEff(lib[0]).Wins
	if w == nil || w.Kind != "science" || w.Steps != 1 || w.Via == nil || *w.Via != "token:law" {
		t.Fatalf("the win: %+v", w)
	}

	capital := hnNew(t, Scenario{Age: 1, Pawn: intPtr(8), Accessible: []string{"guard-tower"}})
	gt := hcSlotHints(t, &capital, hnAllHints, 0, "guard-tower")
	if got := hcCodes(gt[0]); !slices.Equal(got, []string{"win"}) || hcEff(gt[0]).Wins == nil || hcEff(gt[0]).Wins.Kind != "military" {
		t.Fatalf("Guard Tower into the capital: %v %+v", hcCodes(gt[0]), hcEff(gt[0]).Wins)
	}
	if hcEff(gt[1]).Wins != nil || slices.Contains(hcCodes(gt[1]), "win") {
		t.Fatalf("seat 1 pushing back wins nothing: %+v", gt[1])
	}

	poor := hnNew(t, Scenario{Age: 2, Pawn: intPtr(8), Accessible: []string{"horse-breeders"}, Players: [2]ScenarioPlayer{{Coins: intPtr(0)}}})
	hb := hcSlotHints(t, &poor, hnAllHints, 0, "horse-breeders")
	if hcEff(hb[0]).Paid || hcEff(hb[0]).Wins == nil || slices.Contains(hcCodes(hb[0]), "win") {
		t.Fatalf("an unaffordable win: paid %v, wins %+v, codes %v", hcEff(hb[0]).Paid, hcEff(hb[0]).Wins, hcCodes(hb[0]))
	}
}

// TestCardHintsLoot: a red card whose shields take a looting token earns
// its builder the loot pip, with the coins the victim actually loses, and
// the other seat, pushing the pawn the other way, none
// (military-science.md "Looting tokens").
func TestCardHintsLoot(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(5)}, {Coins: intPtr(1)}},
		Pawn:       intPtr(2),
		Accessible: []string{"garrison"},
	})
	g := hcSlotHints(t, &s, hnAllHints, Spectator, "garrison")
	if got := hcCodes(g[0]); !slices.Equal(got, []string{"loot"}) {
		t.Fatalf("seat 0's codes: %v, want [loot]", got)
	}
	m := hcEff(g[0]).Military
	if m == nil || m.From != 2 || m.To != 3 || len(m.Loot) != 1 || m.Loot[0] != (view.LootTaken{At: 3, Lost: 1}) {
		t.Fatalf("seat 0's pawn move: %+v", m)
	}
	if hcEff(g[0]).ScoreDelta != 0 {
		t.Fatalf("a red card's worth is not sent: %+v", g[0].Effect)
	}
	if got := hcCodes(g[1]); len(got) != 0 || hcEff(g[1]).Military == nil || hcEff(g[1]).Military.To != 1 || len(hcEff(g[1]).Military.Loot) != 0 {
		t.Fatalf("seat 1: codes %v, move %+v", got, hcEff(g[1]).Military)
	}
}

// TestCardHintsEconomy: the trade prices a card moves, for both seats; the
// coins a producer saves in view (only producers carry it); the last
// producer of a resource, certain or only in view; and the worth of a
// yellow card now (economy.md "Trading", "Resources", "Coin rewards").
func TestCardHintsEconomy(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"clay-pool", "glassworks"}}, {}},
		Accessible: []string{"brickyard", "glass-blower"},
	})
	other := hnNew(t, Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"clay-pool", "glassworks"}}, {}},
		Accessible: []string{"forum", "library"},
	})
	get := func(slug string) [2]view.CardHints {
		for _, st := range []*State{&s, &other} {
			h := ComputeHints(st, hnAllHints)
			vw := ViewWithHints(st, 0, &h)
			for _, sv := range vw.Structure {
				if sv.Card != nil && *sv.Card == slug {
					return *sv.Hints
				}
			}
		}
		t.Fatalf("%s is not face up", slug)
		return [2]view.CardHints{}
	}
	brick := get("brickyard")
	if want := []view.PriceShift{{Seat: 1, Resource: "clay", From: 3, To: 5}}; !slices.Equal(brick[0].PriceShift, want) {
		t.Fatalf("Brickyard for seat 0: %+v", brick[0].PriceShift)
	}
	if want := []view.PriceShift{{Seat: 0, Resource: "clay", From: 2, To: 4}}; !slices.Equal(brick[1].PriceShift, want) {
		t.Fatalf("Brickyard for seat 1: %+v", brick[1].PriceShift)
	}
	if brick[0].SavesInView == nil || brick[1].SavesInView == nil {
		t.Fatal("a brown card carries savesInView for both seats")
	}
	if lib := get("library"); lib[0].SavesInView != nil || lib[0].LastProducer != nil {
		t.Fatalf("a green card is no producer: %+v", lib[0])
	}
	if forum := get("forum"); forum[0].SavesInView == nil || len(forum[0].PriceShift) != 0 {
		t.Fatalf("the Forum saves, and moves no price: %+v", forum[0])
	}
	// Glassworks is in seat 0's city, so the Glass-Blower is the last glass
	// producer for certain (no other glass card exists).
	gb := get("glass-blower")
	if gb[0].LastProducer == nil || *gb[0].LastProducer != view.LastProducerCertain || gb[1].LastProducer == nil {
		t.Fatalf("the Glass-Blower: %+v", gb[0].LastProducer)
	}
	// The Brickyard: the Clay Pool is in a city and the Clay Pit of Age I,
	// an age that is over, was never seen, so no other clay producer can
	// come.
	if lp := brick[0].LastProducer; lp == nil || *lp != view.LastProducerCertain {
		t.Fatalf("the Brickyard: %v", lp)
	}

	// In Age I the Glassworks has the Glass-Blower of Age II still to
	// come: last in view only.
	a1 := hnNew(t, Scenario{Age: 1, Accessible: []string{"glassworks"}})
	gw := hcSlotHints(t, &a1, hnAllHints, 0, "glassworks")
	if gw[0].LastProducer == nil || *gw[0].LastProducer != view.LastProducerVisible {
		t.Fatalf("the Glassworks in Age I: %v", gw[0].LastProducer)
	}
	// A discarded Glassworks still counts while a Mausoleum is open, and
	// not once none is.
	sc := Scenario{
		Age:        2,
		Wonders:    wonderFill(nil, []string{"mausoleum"}),
		Discard:    []string{"glassworks"},
		Accessible: []string{"glass-blower"},
	}
	open := hnNew(t, sc)
	if lp := hcSlotHints(t, &open, hnAllHints, 0, "glass-blower")[0].LastProducer; lp != nil {
		t.Fatalf("a discarded Glassworks with an open Mausoleum: %v", *lp)
	}
	sc.Wonders = wonderFill([]string{"pyramids", "colossus", "sphinx", "circus-maximus"}, []string{"great-library", "appian-way", "statue-of-zeus", "piraeus"})
	shut := hnNew(t, sc)
	if lp := hcSlotHints(t, &shut, hnAllHints, 0, "glass-blower")[0].LastProducer; lp == nil || *lp != view.LastProducerCertain {
		t.Fatalf("a discarded Glassworks and no Mausoleum: %v", lp)
	}
	// Worth now: the Tavern gives 4 coins.
	tav := hnNew(t, Scenario{Age: 1, Accessible: []string{"tavern"}})
	if e := hcEff(hcSlotHints(t, &tav, hnAllHints, 0, "tavern")[0]); e.Coins != 4 || !e.Paid {
		t.Fatalf("the Tavern: %+v", e)
	}
}

// TestCardHintsFamilies: each part follows its family's switch, and with
// every per-card family off the slots carry no hints at all.
func TestCardHintsFamilies(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Players:    [2]ScenarioPlayer{{City: []string{"scriptorium"}, Coins: intPtr(9)}, {Coins: intPtr(3)}},
		Pawn:       intPtr(2),
		Accessible: []string{"garrison", "lumber-yard", "tavern"},
	})
	scienceOnly := view.HintSettings{Science: true}
	g := hcSlotHints(t, &s, scienceOnly, 0, "garrison")
	if g == nil || len(g[0].Codes) != 0 || hcEff(g[0]).Military != nil || hcEff(g[0]).ScoreDelta != 0 {
		t.Fatalf("military and scoring off: %+v", g)
	}
	if ly := hcSlotHints(t, &s, scienceOnly, 0, "lumber-yard"); len(ly[0].PriceShift) != 0 || ly[0].SavesInView != nil || ly[0].LastProducer != nil {
		t.Fatalf("economy and counting off: %+v", ly[0])
	}
	if tav := hcSlotHints(t, &s, scienceOnly, 0, "tavern"); hcEff(tav[0]).Coins != 0 {
		t.Fatalf("scoring off: %+v", tav[0].Effect)
	}
	military := view.HintSettings{Military: true}
	if g := hcSlotHints(t, &s, military, 0, "garrison"); !slices.Equal(hcCodes(g[0]), []string{"loot"}) {
		t.Fatalf("military on: %v", hcCodes(g[0]))
	}
	off := view.HintSettings{Prices: true, Tempo: true, Wonders: true, Tokens: true, Recap: true, Advisor: true}
	if g := hcSlotHints(t, &s, off, 0, "garrison"); g != nil {
		t.Fatalf("every per-card family off: %+v", g)
	}
}

// TestDiscardHints: the Mausoleum's pool, for the seat whose Mausoleum is
// open and for the seat choosing its rebuild: a pair the rebuild would
// complete, its price for the seat, and nothing for a seat without an open
// Mausoleum or with the wonders family off (wonders-tokens.md "The
// Mausoleum").
func TestDiscardHints(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Wonders:    wonderFill([]string{"mausoleum"}, nil),
		Players:    [2]ScenarioPlayer{{City: []string{"scriptorium"}, Coins: intPtr(40)}, {Coins: intPtr(40)}},
		Discard:    []string{"library", "baths", "garrison"},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
	})
	h := ComputeHints(&s, hnAllHints)
	d := h.board.Discard
	// Baths has nothing to say beyond its price, so it is left out.
	if len(d) != 2 || d[0].Seat != 0 || d[1].Seat != 0 || slices.ContainsFunc(d, func(x view.DiscardHintView) bool { return x.Card == "baths" }) {
		t.Fatalf("the pool for seat 0 only, without the Baths: %+v", d)
	}
	var lib *view.DiscardHintView
	for i := range d {
		if d[i].Card == "library" {
			lib = &d[i]
		}
	}
	if lib == nil || !slices.Equal(hcCodes(lib.Hints), []string{"pair"}) || !lib.ChainFree || lib.Total != 0 || hcEff(lib.Hints).Paid {
		t.Fatalf("the Library in the pile: %+v", lib)
	}
	if off := ComputeHints(&s, view.HintSettings{Science: true}); len(off.board.Discard) != 0 {
		t.Fatal("the wonders family is off")
	}
	// Building the Mausoleum opens the rebuild: the pool is still listed
	// for its builder, whose Mausoleum is no longer open.
	hnDo(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != PhaseBuildFromDiscard {
		t.Fatalf("phase %s", s.Phase)
	}
	h = ComputeHints(&s, hnAllHints)
	if len(h.board.Discard) != 2 {
		t.Fatalf("during the rebuild: %+v", h.board.Discard)
	}
	for _, v := range hlViewers {
		vw := ViewWithHints(&s, v, &h)
		if vw.Hints == nil || len(vw.Hints.Discard) != 2 {
			t.Fatalf("viewer %d: %+v", v, vw.Hints)
		}
	}
}

// TestDestroyHints: during a destroy choice, each candidate says how its
// loss would lower the destroyer's prices, what it would save the
// destroyer in view, and whether it is the victim's only producer of its
// resource; both seats see it (wonders-tokens.md "The Statue of Zeus and
// Circus Maximus"; economy.md "Trading").
func TestDestroyHints(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        2,
		Wonders:    wonderFill([]string{"statue-of-zeus"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}, {City: []string{"quarry", "clay-pool", "clay-pit"}}},
		Accessible: []string{"rostrum", "statue"},
		ToMove:     seatPtr(0),
	})
	hnDo(t, &s, "wonder:statue-of-zeus:rostrum")
	if s.Phase != PhaseDestroyCard {
		t.Fatalf("phase %s", s.Phase)
	}
	h := ComputeHints(&s, hnAllHints)
	byCard := map[string]view.PendingCardHint{}
	for _, v := range hlViewers {
		vw := ViewWithHints(&s, v, &h)
		if vw.Pending == nil || vw.Pending.CardHints == nil {
			t.Fatalf("viewer %d: no card hints in the destroy choice", v)
		}
		got := *vw.Pending.CardHints
		if len(got) != len(vw.Pending.Cards) {
			t.Fatalf("viewer %d: %d hints for %d candidates", v, len(got), len(vw.Pending.Cards))
		}
		for i, c := range got {
			if c.Card != vw.Pending.Cards[i] {
				t.Fatalf("viewer %d: hint %d is for %s, candidate %s", v, i, c.Card, vw.Pending.Cards[i])
			}
			byCard[c.Card] = c
		}
	}
	q := byCard["quarry"]
	if !q.OnlySource || !slices.Equal(q.PriceShift, []view.PriceShift{{Seat: 0, Resource: "stone", From: 3, To: 2}}) {
		t.Fatalf("Quarry: %+v", q)
	}
	cp := byCard["clay-pool"]
	if cp.OnlySource || !slices.Equal(cp.PriceShift, []view.PriceShift{{Seat: 0, Resource: "clay", From: 4, To: 3}}) {
		t.Fatalf("Clay Pool: %+v", cp)
	}
	// The Statue (2 clay) is face up: losing a clay producer saves the
	// destroyer one coin on each clay unit it would buy.
	if cp.SavesInView <= 0 {
		t.Fatalf("Clay Pool saves %d in view", cp.SavesInView)
	}
	if off := ComputeHints(&s, view.HintSettings{Science: true}); off.pending != nil {
		t.Fatal("the economy family is off")
	}
}

// TestCardHintsView: the per-card hints are identical in the three views
// of a state, every face-up slot carries them and no other slot does, and
// a finished game's view, or a view built without settings, carries none.
func TestCardHintsView(t *testing.T) {
	for seed := uint64(1); seed <= 4; seed++ {
		s := hnMidAge(t, seed, 2)
		h := ComputeHints(&s, hnAllHints)
		var first []byte
		for _, v := range hlViewers {
			vw := ViewWithHints(&s, v, &h)
			slots := []*[2]view.CardHints{}
			for _, sv := range vw.Structure {
				if (sv.State == view.SlotStateFaceUp) != (sv.Hints != nil) {
					t.Fatalf("seed %d, viewer %d: slot %d (%s) hints %v", seed, v, sv.Index, sv.State, sv.Hints != nil)
				}
				slots = append(slots, sv.Hints)
			}
			b, err := json.Marshal(slots)
			if err != nil {
				t.Fatal(err)
			}
			if first == nil {
				first = b
			} else if string(b) != string(first) {
				t.Fatalf("seed %d: viewer %d's per-card hints differ from seat 0's", seed, v)
			}
			if plain := ViewFor(&s, v); slices.ContainsFunc(plain.Structure, func(sv view.SlotView) bool { return sv.Hints != nil }) {
				t.Fatal("ViewFor carries per-card hints")
			}
		}
		end := hnPlayRandom(t, seed, Options{}, func(*State) bool { return false })
		he := ComputeHints(&end, hnAllHints)
		for _, v := range hlViewers {
			vw := ViewWithHints(&end, v, &he)
			if vw.Hints != nil || slices.ContainsFunc(vw.Structure, func(sv view.SlotView) bool { return sv.Hints != nil }) {
				t.Fatalf("seed %d: a finished game's view carries hints", seed)
			}
			if vw.Pending != nil && vw.Pending.CardHints != nil {
				t.Fatalf("seed %d: a finished game's pending carries hints", seed)
			}
		}
	}
}

// TestCardHintsSweep checks the per-card hints over random games against
// their definitions: a pip only with its fact (pair with a token on the
// board, loot with a token taken, win with a win), the same price shifts
// as priceShift, and previews that never change the state.
func TestCardHintsSweep(t *testing.T) {
	buf := make([]Action, 0, MaxLegalActions)
	seen := map[string]int{}
	for seed := uint64(1); seed <= 40; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		for !s.Over() {
			before := s
			h := ComputeHints(&s, hnAllHints)
			hnUnchanged(t, &before, &s, "ComputeHints")
			if h.slots != nil {
				for i := range s.Slots {
					if h.cards&(1<<i) == 0 {
						continue
					}
					for seat := range Seat(2) {
						ch := h.slots[i][seat]
						for _, c := range ch.Codes {
							seen[string(c)]++
							switch c {
							case "pair":
								if hcEff(ch).Science == nil || hcEff(ch).Science.Kind != "pair" || !hcEff(ch).Science.TokenOnBoard {
									t.Fatalf("seed %d ply %d: a pair pip without its pair: %+v", seed, s.Ply, ch.Effect)
								}
							case "loot":
								if hcEff(ch).Military == nil || len(hcEff(ch).Military.Loot) == 0 {
									t.Fatalf("seed %d ply %d: a loot pip without looting", seed, s.Ply)
								}
							case "win":
								if hcEff(ch).Wins == nil {
									t.Fatalf("seed %d ply %d: a win pip without a win", seed, s.Ply)
								}
							}
						}
						if want := viewPriceShifts(priceShift(&s, seat, s.Slots[i].Card)); !slices.Equal(ch.PriceShift, want) {
							t.Fatalf("seed %d ply %d: price shifts %+v, want %+v", seed, s.Ply, ch.PriceShift, want)
						}
					}
				}
			}
			seen["discard"] += len(h.board.Discard)
			if h.pending != nil {
				seen["destroy"] += len(*h.pending)
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
	}
	for _, k := range []string{"pair", "loot"} {
		if seen[k] == 0 {
			t.Errorf("the sweep never saw a %s pip", k)
		}
	}
	t.Logf("pips and pool entries seen: %v", seen)
}
