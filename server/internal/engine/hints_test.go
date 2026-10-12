package engine

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The tests of the hints engine (hints.go): each helper against the rules
// it encodes (docs/rules/), the core view fields that came with it (shields,
// playingAgain, the draft order), the ViewFor and ViewWithHints split, and
// the benchmarks whose numbers docs/architecture.md records. The leak guard
// is in hints_leak_test.go. Helpers are prefixed hn.

// hnAllHints switches every family on.
var hnAllHints = view.HintSettings{
	Prices: true, Economy: true, Science: true, Military: true, Threats: true, Tempo: true,
	Wonders: true, Tokens: true, Scoring: true, Counting: true, Recap: true, Advisor: true,
}

// hnNew builds a scenario position and checks its invariants.
func hnNew(t testing.TB, sc Scenario) State {
	t.Helper()
	s, err := NewGameFromScenario(7, Options{}, &sc)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckInvariants(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func hnCard(t testing.TB, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("no card %q", slug)
	}
	return c
}

func hnWonder(t testing.TB, slug string) catalog.WonderID {
	t.Helper()
	w, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("no wonder %q", slug)
	}
	return w
}

func hnToken(t testing.TB, slug string) catalog.TokenID {
	t.Helper()
	k, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("no token %q", slug)
	}
	return k
}

func hnKey(t testing.TB, key string) Action {
	t.Helper()
	a, err := ParseKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// hnDo applies action keys, checking the invariants after each.
func hnDo(t testing.TB, s *State, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := Apply(s, hnKey(t, k), nil); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if err := CheckInvariants(s); err != nil {
			t.Fatalf("after %s: %v", k, err)
		}
	}
}

// hnUnchanged fails when a helper changed the state it was given.
func hnUnchanged(t *testing.T, before, after *State, what string) {
	t.Helper()
	if *before != *after {
		t.Fatalf("%s changed the state", what)
	}
}

// hnPlayRandom plays seed's game with random legal actions until stop says
// so or the game ends, and returns the state.
func hnPlayRandom(t testing.TB, seed uint64, opts Options, stop func(*State) bool) State {
	t.Helper()
	s := NewGame(seed, opts)
	r := NewRNG(seed ^ 0x9e3779b97f4a7c15)
	buf := make([]Action, 0, MaxLegalActions)
	for !s.Over() && !stop(&s) {
		legal := LegalActions(&s, buf[:0])
		if err := Apply(&s, legal[r.IntN(len(legal))], nil); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// --- The derived counters and the hash ---

// TestHashIgnoresDerivedCounters pins the layout that keeps every hash
// recorded before the derived counters existed: they are the last fields
// of State, and Hash digests none of their bytes but every byte before.
func TestHashIgnoresDerivedCounters(t *testing.T) {
	rt := reflect.TypeFor[State]()
	n := rt.NumField()
	if rt.Field(n-2).Name != "Shields" || rt.Field(n-1).Name != "PlayingAgain" {
		t.Fatalf("State must end with Shields, PlayingAgain; it ends with %s, %s", rt.Field(n-2).Name, rt.Field(n-1).Name)
	}
	if want := binary.Size(State{}) - 3; stateHashedSize != want {
		t.Fatalf("Hash digests %d bytes, want %d", stateHashedSize, want)
	}
	s := NewGame(5, Options{})
	h := Hash(&s)
	x := s
	x.Shields = [2]int8{4, 9}
	x.PlayingAgain = true
	if Hash(&x) != h {
		t.Error("the derived counters changed the hash")
	}
	x = s
	x.WondersBuilt++
	if Hash(&x) == h {
		t.Error("the field before the counters is not hashed")
	}
}

// TestShieldsCounter: every shield a seat places counts, from buildings
// (with Strategy's extra shield) and from wonders (without it)
// (military-science.md "Where shields come from").
func TestShieldsCounter(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Wonders:    wonderFill([]string{"colossus"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(30), Tokens: []string{"strategy"}}, {Coins: intPtr(30)}},
		Accessible: []string{"guard-tower", "stable", "baths"},
		ToMove:     seatPtr(0),
	})
	hnDo(t, &s, "construct:guard-tower") // 1 + Strategy's 1
	if s.Shields != [2]int8{2, 0} || s.Military != 2 {
		t.Fatalf("after Guard Tower with Strategy: shields %v, pawn %d; want [2 0], 2", s.Shields, s.Military)
	}
	hnDo(t, &s, "construct:stable") // seat 1, no Strategy
	if s.Shields != [2]int8{2, 1} || s.Military != 1 {
		t.Fatalf("after seat 1's Stable: shields %v, pawn %d; want [2 1], 1", s.Shields, s.Military)
	}
	hnDo(t, &s, "wonder:colossus:baths") // 2, Strategy adds nothing to a wonder
	if s.Shields != [2]int8{4, 1} || s.Military != 3 {
		t.Fatalf("after the Colossus: shields %v, pawn %d; want [4 1], 3", s.Shields, s.Military)
	}
	if v := ViewFor(&s, 1); v.Players[0].Shields != 4 || v.Players[1].Shields != 1 {
		t.Fatalf("view shields %d, %d", v.Players[0].Shields, v.Players[1].Shields)
	}
}

// TestShieldsMatchPawn: in real games the pawn is the difference of the
// shields placed, until a capital stops it.
func TestShieldsMatchPawn(t *testing.T) {
	for seed := uint64(1); seed <= 60; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		for !s.Over() {
			if got := int(s.Shields[0]) - int(s.Shields[1]); got != int(s.Military) {
				t.Fatalf("seed %d ply %d: shields %v but the pawn is at %d", seed, s.Ply, s.Shields, s.Military)
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
		if s.Result.Kind != VictoryMilitary && int(s.Shields[0])-int(s.Shields[1]) != int(s.Military) {
			t.Fatalf("seed %d at the end: shields %v, pawn %d", seed, s.Shields, s.Military)
		}
	}
}

// TestPlayingAgain: a turn granted by an extra turn is marked, the next
// one is not, and an extra turn lost on an age's last card marks nothing
// (turn.md "Extra turns").
func TestPlayingAgain(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Wonders:    wonderFill([]string{"sphinx"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Accessible: []string{"theater", "baths"},
		ToMove:     seatPtr(0),
	})
	if s.PlayingAgain || ViewFor(&s, 0).PlayingAgain {
		t.Fatal("playing again before any extra turn")
	}
	hnDo(t, &s, "wonder:sphinx:theater")
	if !s.PlayingAgain || s.ToMove != 0 || !ViewFor(&s, Spectator).PlayingAgain {
		t.Fatalf("after the Sphinx: playing again %v, to move %d", s.PlayingAgain, s.ToMove)
	}
	hnDo(t, &s, "discard:baths")
	if s.PlayingAgain || s.ToMove != 1 {
		t.Fatalf("after the extra turn: playing again %v, to move %d", s.PlayingAgain, s.ToMove)
	}

	sc := NamedScenario("extra-turn-lost")
	lost := hnNew(t, *sc)
	hnDo(t, &lost, sc.Trigger...)
	if lost.PlayingAgain || lost.Phase != PhaseChooseStarter || ViewFor(&lost, 0).PlayingAgain {
		t.Fatalf("an extra turn lost at the age's end: playing again %v in %s", lost.PlayingAgain, lost.Phase)
	}
}

// TestDraftOrder: the draft view lists the round's pickers, first player
// relative (setup.md "The wonder draft").
func TestDraftOrder(t *testing.T) {
	for _, first := range []FirstPlayerRule{FirstPlayerSeat0, FirstPlayerSeat1} {
		s := NewGame(3, Options{FirstPlayer: first})
		f := int(s.FirstSeat)
		o := 1 - f
		if got := ViewFor(&s, 0).Draft.Order; !slices.Equal(got, []int{f, o, o, f}) {
			t.Fatalf("first player %d, round 1 order %v", f, got)
		}
		buf := make([]Action, 0, MaxLegalActions)
		for draftPicks(&s) < 4 {
			ApplyTrusted(&s, LegalActions(&s, buf[:0])[0])
		}
		if got := ViewFor(&s, Spectator).Draft.Order; !slices.Equal(got, []int{o, f, f, o}) {
			t.Fatalf("first player %d, round 2 order %v", f, got)
		}
	}
}

// --- ViewFor and ViewWithHints ---

// TestViewWithHints: ViewFor is ViewWithHints without hints; with hints a
// live game's view carries them for every viewer, and a finished game's
// view carries none.
func TestViewWithHints(t *testing.T) {
	s := hnPlayRandom(t, 21, Options{}, func(s *State) bool { return s.Age == 2 })
	h := ComputeHints(&s, hnAllHints)
	if h.Settings() != hnAllHints {
		t.Fatal("the settings are not kept")
	}
	for _, v := range []Viewer{0, 1, Spectator} {
		plain := ViewFor(&s, v)
		if plain.Hints != nil {
			t.Fatal("ViewFor carries hints")
		}
		a, _ := json.Marshal(plain)
		b, _ := json.Marshal(ViewWithHints(&s, v, nil))
		if string(a) != string(b) {
			t.Fatal("ViewFor differs from ViewWithHints(nil)")
		}
		if ViewWithHints(&s, v, &h).Hints == nil {
			t.Fatalf("viewer %d: no hints in a live game", v)
		}
	}
	end := hnPlayRandom(t, 21, Options{}, func(*State) bool { return false })
	h = ComputeHints(&end, hnAllHints)
	for _, v := range []Viewer{0, 1, Spectator} {
		if vw := ViewWithHints(&end, v, &h); vw.Hints != nil || vw.PlayingAgain {
			t.Fatalf("viewer %d: a finished game's view carries hints", v)
		}
	}
}

// TestHintEnumsMatchView checks that the hint vocabularies of
// internal/view/hints.go list exactly the engine's names, in order.
func TestHintEnumsMatchView(t *testing.T) {
	path := filepath.Join("..", "view", "hints.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			typ := vs.Type.(*ast.Ident)
			for _, v := range vs.Values {
				s, err := strconv.Unquote(v.(*ast.BasicLit).Value)
				if err != nil {
					t.Fatal(err)
				}
				got[typ.Name] = append(got[typ.Name], s)
			}
		}
	}
	want := map[string][]string{
		"PublicPlace":       publicPlaceNames[:],
		"ExtraTurnState":    extraTurnStateNames[:],
		"ExtraTurnSource":   extraTurnSourceNames[:],
		"ScienceEffectKind": scienceEffectNames[:],
		"HintCode":          hintCodeNames[:],
		"LastProducer":      lastProducerNames[:],
		// The engine names these with the view's own constants
		// (hints_board.go), so the list here is the vocabulary itself.
		"SymbolState": {"paired", "live", "owned", "open", "dead"},
		"TieBreak":    {"total", "blue", "shared"},
	}
	if len(got) != len(want) {
		t.Errorf("view/hints.go has vocabularies %v; this test checks %d", reflect.ValueOf(got).MapKeys(), len(want))
	}
	for typ, w := range want {
		if !slices.Equal(got[typ], w) {
			t.Errorf("view.%s is %v, the engine's names are %v", typ, got[typ], w)
		}
	}
}

// --- publicPlace ---

// TestPublicPlaceScenario covers the places a scenario can set: face up,
// a city, tucked, the discard pile, and an unseen card of this age and of
// a later one (setup.md "What is hidden from both players").
func TestPublicPlaceScenario(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Wonders:    wonderFill([]string{"pyramids"}, nil),
		Players:    [2]ScenarioPlayer{{City: []string{"quarry"}, Built: []ScenarioWonder{{Wonder: "pyramids", Tucked: "baths"}}}, {City: []string{"press"}}},
		Discard:    []string{"altar"},
		Accessible: []string{"garrison"},
	})
	check := func(slug string, want PublicPlace, seat Seat) {
		t.Helper()
		before := s
		got := publicPlace(&s, hnCard(t, slug))
		hnUnchanged(t, &before, &s, "publicPlace")
		if got.Place != want || ((want == PlaceCity || want == PlaceTucked) && got.Seat != seat) {
			t.Errorf("%s: %v (seat %d), want %v (seat %d)", slug, got.Place, got.Seat, want, seat)
		}
	}
	check("quarry", PlaceCity, 0)
	check("press", PlaceCity, 1)
	check("baths", PlaceTucked, 0)
	check("altar", PlaceDiscard, 0)
	check("library", PlaceUnseenLaterAge, 0)
	check("arsenal", PlaceUnseenLaterAge, 0)
	check("tacticians-guild", PlaceUnseenLaterAge, 0)
	g := publicPlace(&s, hnCard(t, "garrison"))
	if g.Place != PlaceFaceUp || s.Slots[g.Slot].Card != hnCard(t, "garrison") {
		t.Errorf("garrison: %+v", g)
	}
	for i := range s.Slots {
		if sl := s.Slots[i]; sl.Card != 0 && !sl.FaceUp {
			if p := publicPlace(&s, sl.Card); p.Place != PlaceUnseenThisAge {
				t.Errorf("face-down slot %d: %v, want unseenThisAge", i, p.Place)
			}
		}
	}
	for _, c := range s.Removed[0] {
		if c == 0 {
			continue // a scenario's hole
		}
		if p := publicPlace(&s, c); p.Place != PlaceUnseenThisAge {
			t.Errorf("the set-aside %s: %v, want unseenThisAge", catalog.Card(c).Slug, p.Place)
		}
	}
}

// TestPublicPlaceSweep checks every card in real games against the full
// state: a place is a fact about the card, and removedKnown occurs only
// for a past age's set-aside cards.
func TestPublicPlaceSweep(t *testing.T) {
	for seed := uint64(1); seed <= 30; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		buf := make([]Action, 0, MaxLegalActions)
		for !s.Over() {
			if s.Ply%5 == 0 {
				hnCheckPlaces(t, &s, seed)
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
	}
}

func hnCheckPlaces(t *testing.T, s *State, seed uint64) {
	t.Helper()
	inSlot := func(c catalog.CardID, up bool) bool {
		for i := range s.Slots {
			if s.Slots[i].Card == c && s.Slots[i].FaceUp == up {
				return true
			}
		}
		return false
	}
	for c, ok := catalog.T.All.Next(1); ok; c, ok = catalog.T.All.Next(c + 1) {
		p := publicPlace(s, c)
		age := catalog.T.Age[c]
		removed := slices.Contains(s.Removed[max(age, 1)-1][:], c) || slices.Contains(s.BoxGuilds[:], c)
		var bad bool
		switch p.Place {
		case PlaceFaceUp:
			bad = !inSlot(c, true) || s.Slots[p.Slot].Card != c
		case PlaceCity:
			bad = !s.Players[p.Seat].City.Has(c)
		case PlaceTucked:
			bad = !slices.ContainsFunc(s.Players[p.Seat].Wonders[:], func(w WonderSlot) bool { return w.Built && w.Tucked == c })
		case PlaceDiscard:
			bad = !s.Discard.Has(c)
		case PlaceUnseenThisAge:
			bad = age != s.Age || (!inSlot(c, false) && !removed)
		case PlaceUnseenLaterAge:
			bad = age <= s.Age
		case PlaceRemovedKnown:
			bad = age >= s.Age || !removed
		default:
			bad = true
		}
		if bad {
			t.Fatalf("seed %d ply %d (age %d): %s is %v (seat %d, slot %d)", seed, s.Ply, s.Age, catalog.Card(c).Slug, p.Place, p.Seat, p.Slot)
		}
	}
}

// --- previewBuild ---

// TestPreviewBuildPair: the second card of a symbol completes a pair, which
// opens the board token choice while the board has a token and gives
// nothing once it is empty; a partner tucked under a wonder never counts
// (military-science.md "Pairs").
func TestPreviewBuildPair(t *testing.T) {
	sc := Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"scriptorium"}, Coins: intPtr(20)}, {Coins: intPtr(20)}},
		Accessible: []string{"library"},
	}
	s := hnNew(t, sc)
	lib := hnCard(t, "library")
	before := s
	p := previewBuild(&s, 0, lib, false)
	hnUnchanged(t, &before, &s, "previewBuild")
	if !p.HasScience || p.Science.Kind != ScienceEffectPair || !p.Science.TokenOnBoard || p.Opens != PhasePickBoardToken || p.Science.DistinctAfter != 1 {
		t.Fatalf("seat 0 completing the quill pair: %+v", p)
	}
	if !p.Paid || p.Wins {
		t.Fatalf("Library is free by the book chain and wins nothing: %+v", p)
	}
	if q := previewBuild(&s, 1, lib, false); q.Science.Kind != ScienceEffectNew || q.Opens != PhasePlay {
		t.Fatalf("seat 1, a new symbol: %+v", q)
	}

	empty := s
	empty.BoardTokens = 0
	if q := previewBuild(&empty, 0, lib, false); q.Science.Kind != ScienceEffectPair || q.Science.TokenOnBoard || q.Opens != PhasePlay {
		t.Fatalf("a pair with an empty board: %+v", q)
	}

	sc.Players[0].City = nil
	sc.Players[0].Built = []ScenarioWonder{{Wonder: "pyramids", Tucked: "scriptorium"}}
	sc.Wonders = wonderFill([]string{"pyramids"}, nil)
	tucked := hnNew(t, sc)
	if q := previewBuild(&tucked, 0, lib, false); q.Science.Kind != ScienceEffectNew || q.Opens != PhasePlay {
		t.Fatalf("a tucked partner counts for nothing: %+v", q)
	}
}

// TestPreviewBuildScienceWins: a sixth different symbol wins at once, and
// a pair while Law is on the board wins through the token it lets the seat
// take, the scales being the sixth (military-science.md "Scientific
// supremacy").
func TestPreviewBuildScienceWins(t *testing.T) {
	five := []string{"scriptorium", "workshop", "pharmacist", "apothecary", "university"}
	s := hnNew(t, Scenario{
		Age:         2,
		Players:     [2]ScenarioPlayer{{City: five, Coins: intPtr(30)}},
		BoardTokens: []string{"law", "agriculture", "philosophy", "masonry", "strategy"},
		Accessible:  []string{"library", "school"},
		ToMove:      seatPtr(0),
	})
	p := previewBuild(&s, 0, hnCard(t, "library"), false)
	want := Win{Kind: VictoryScience, Steps: 1, Via: TokenAction(hnToken(t, "law"))}
	if !p.Wins || p.Win != want || p.Science.DistinctAfter != 5 {
		t.Fatalf("a pair with Law on the board: %+v, want the win %+v", p, want)
	}
	if v := tokenPreview(&s, 0, hnToken(t, "law")); !v.Wins {
		t.Fatalf("Law at five symbols: %+v", v)
	}
	if v := tokenPreview(&s, 1, hnToken(t, "law")); v.Wins {
		t.Fatalf("Law for seat 1, with no symbol: %+v", v)
	}
	// School repeats the wheel of the Apothecary: a pair too.
	if q := previewBuild(&s, 0, hnCard(t, "school"), false); !q.Wins || q.Win != want {
		t.Fatalf("School: %+v", q)
	}
	w, ok := WinsNow(&s, hnKey(t, "construct:library"), FaceUpSlots(&s))
	if !ok || w != want {
		t.Fatalf("WinsNow construct:library = %+v %v, want %+v", w, ok, want)
	}

	six := hnNew(t, Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"workshop", "pharmacist", "apothecary", "university", "academy"}, Coins: intPtr(30)}},
		Accessible: []string{"library"},
	})
	if q := previewBuild(&six, 0, hnCard(t, "library"), false); !q.Wins || q.Win != (Win{Kind: VictoryScience, Steps: 1}) || q.Science.DistinctAfter != 6 {
		t.Fatalf("a sixth symbol: %+v", q)
	}
}

// TestPreviewBuildLooting: the trading coins reach an opponent who owns
// Economy before the looting takes coins from them, and the loss is
// clamped to the coins they have (military-science.md "Looting tokens";
// decisions.md "Effect order").
func TestPreviewBuildLooting(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:        1,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(5)}, {Coins: intPtr(0), Tokens: []string{"economy"}}},
		Pawn:       intPtr(2),
		Accessible: []string{"garrison"},
	})
	p := previewBuild(&s, 0, hnCard(t, "garrison"), false)
	if !p.Paid || !p.Moved || p.Pawn.From != 2 || p.Pawn.To != 3 {
		t.Fatalf("Garrison: %+v", p)
	}
	if p.Pawn.NLoot != 1 || p.Pawn.Loot[0] != (lootTaken{At: 3, Lost: 2}) {
		t.Fatalf("the looting: %+v, want 2 lost at 3 (the Economy income first)", p.Pawn)
	}
	if p.CoinsAfter != [2]int{3, 0} {
		t.Fatalf("coins after: %v, want [3 0]", p.CoinsAfter)
	}
	// Strategy: two spaces, and one move takes both tokens of a side only
	// when it reaches both; here the second is at 6.
	s.Players[0].Tokens.Add(hnToken(t, "strategy"))
	if q := previewBuild(&s, 0, hnCard(t, "garrison"), false); q.Pawn.To != 4 || q.Pawn.NLoot != 1 {
		t.Fatalf("with Strategy: %+v", q.Pawn)
	}

	capital := hnNew(t, Scenario{Age: 1, Pawn: intPtr(8), Accessible: []string{"guard-tower"}})
	if q := previewBuild(&capital, 0, hnCard(t, "guard-tower"), false); !q.Wins || q.Win != (Win{Kind: VictoryMilitary, Steps: 1}) {
		t.Fatalf("the capital: %+v", q)
	}
	if q := previewBuild(&capital, 1, hnCard(t, "guard-tower"), false); q.Wins || q.Pawn.To != 7 {
		t.Fatalf("seat 1 pushing back: %+v", q)
	}
}

// TestPreviewBuildCoins: a free chain build pays Urbanism's bonus, a
// Mausoleum rebuild never does, and an unaffordable card is previewed
// without payment (wonders-tokens.md "The ten tokens"; decisions.md
// "Urbanism and the Mausoleum").
func TestPreviewBuildCoins(t *testing.T) {
	sc := Scenario{
		Age:        2,
		Players:    [2]ScenarioPlayer{{City: []string{"scriptorium"}, Tokens: []string{"urbanism"}, Coins: intPtr(0)}},
		Accessible: []string{"library", "courthouse"},
	}
	s := hnNew(t, sc)
	if p := previewBuild(&s, 0, hnCard(t, "library"), false); !p.Paid || p.Coins != 4 || p.CoinsAfter[0] != 4 {
		t.Fatalf("a chain build with Urbanism: %+v", p)
	}
	if p := previewBuild(&s, 0, hnCard(t, "courthouse"), false); p.Paid || p.CoinsAfter[0] != 0 || p.ScoreDelta[0] != 5 {
		t.Fatalf("an unaffordable Courthouse: %+v", p)
	}
	sc.Accessible = nil
	sc.Discard = []string{"library"}
	r := hnNew(t, sc)
	if p := previewBuild(&r, 0, hnCard(t, "library"), true); p.Coins != 0 || p.Paid {
		t.Fatalf("a rebuild pays no Urbanism bonus: %+v", p)
	}
}

// --- previewWonder ---

func hnSevenWonders(t *testing.T) State {
	return hnNew(t, Scenario{
		Age:     2,
		Wonders: wonderFill([]string{"pyramids", "colossus", "sphinx", "mausoleum"}, []string{"great-library", "appian-way", "statue-of-zeus", "piraeus"}),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(30), Built: []ScenarioWonder{{Wonder: "pyramids"}, {Wonder: "colossus"}, {Wonder: "sphinx"}}},
			{Coins: intPtr(30), Built: []ScenarioWonder{{Wonder: "great-library"}, {Wonder: "appian-way"}, {Wonder: "statue-of-zeus"}}},
		},
	})
}

// TestPreviewWonderSeventh: building the seventh wonder sends the last
// unbuilt one back to the box (wonders-tokens.md "The seven-wonder rule").
func TestPreviewWonderSeventh(t *testing.T) {
	s := hnSevenWonders(t)
	if s.WondersBuilt != 6 {
		t.Fatalf("%d wonders built", s.WondersBuilt)
	}
	before := s
	p := previewWonder(&s, 0, hnWonder(t, "mausoleum"))
	hnUnchanged(t, &before, &s, "previewWonder")
	if p.Removes != hnWonder(t, "piraeus") {
		t.Fatalf("the Mausoleum as the seventh removes %v", p.Removes)
	}
	if q := previewWonder(&s, 1, hnWonder(t, "piraeus")); q.Removes != hnWonder(t, "mausoleum") || q.ExtraTurn != ExtraTurnSourceWonder {
		t.Fatalf("Piraeus as the seventh: %+v", q)
	}
	if q := previewWonder(&s, 0, hnWonder(t, "pyramids")); q != (wonderPreview{}) {
		t.Fatalf("a built wonder has no preview: %+v", q)
	}
}

// TestPreviewWonderGreatLibrary: the preview names the draw without making
// it, so it depends on the box's count, never on its order or content
// (wonders-tokens.md "The Great Library").
func TestPreviewWonderGreatLibrary(t *testing.T) {
	s := hnNew(t, Scenario{Age: 1, Wonders: wonderFill(nil, []string{"great-library"}), Players: [2]ScenarioPlayer{{}, {Coins: intPtr(30)}}})
	box := s.BoxTokens
	p := previewWonder(&s, 1, hnWonder(t, "great-library"))
	if p.Opens != PhasePickLibraryToken || p.Pool != LibraryDraw || !p.Paid {
		t.Fatalf("the Great Library: %+v", p)
	}
	if s.BoxTokens != box || s.Phase != PhasePlay {
		t.Fatal("the preview drew from the real box")
	}
	other := s
	slices.Reverse(other.BoxTokens[:])
	if q := previewWonder(&other, 1, hnWonder(t, "great-library")); q != p {
		t.Fatalf("the box order changed the preview: %+v and %+v", p, q)
	}
}

// TestPreviewWonderEffects: coins, shields, the Appian Way's loss, the
// destroy choice's size and the extra turn, in the rules' order
// (wonders-tokens.md "The twelve wonders", "The order of a wonder's
// effects").
func TestPreviewWonderEffects(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     1,
		Wonders: wonderFill([]string{"colossus", "pyramids", "mausoleum", "temple-of-artemis"}, []string{"appian-way", "statue-of-zeus", "circus-maximus", "sphinx"}),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(2), City: []string{"quarry", "clay-pool", "lumber-yard", "glassworks", "press"}},
			{Coins: intPtr(30), Tokens: []string{"theology"}},
		},
		Discard: []string{"baths"},
	})
	if p := previewWonder(&s, 1, hnWonder(t, "appian-way")); !p.Paid || p.CoinsDelta[0] != -2 || p.ExtraTurn != ExtraTurnSourceWonder {
		t.Fatalf("the Appian Way: %+v (seat 0 has only 2 coins to lose)", p)
	}
	if p := previewWonder(&s, 1, hnWonder(t, "statue-of-zeus")); p.Opens != PhaseDestroyCard || p.Pool != 3 || p.Pawn.To != -1 || p.ExtraTurn != ExtraTurnSourceTheology {
		t.Fatalf("the Statue of Zeus: %+v", p)
	}
	if p := previewWonder(&s, 1, hnWonder(t, "circus-maximus")); p.Opens != PhaseDestroyCard || p.Pool != 2 {
		t.Fatalf("Circus Maximus: %+v", p)
	}
	p := previewWonder(&s, 0, hnWonder(t, "colossus"))
	if p.Paid || !p.Moved || p.Pawn.To != 2 || p.ExtraTurn != ExtraTurnSourceNone || p.ScoreDelta[0] != 3+2 {
		t.Fatalf("the Colossus, unaffordable for seat 0: %+v", p)
	}
	if p := previewWonder(&s, 0, hnWonder(t, "temple-of-artemis")); p.CoinsDelta[0] != 12 || p.ExtraTurn != ExtraTurnSourceWonder {
		t.Fatalf("the Temple of Artemis: %+v", p)
	}
	if p := previewWonder(&s, 0, hnWonder(t, "mausoleum")); p.Opens != PhaseBuildFromDiscard || p.Pool != 1 || p.Wins {
		t.Fatalf("the Mausoleum: %+v", p)
	}
}

// TestPreviewWonderMausoleumWin: the Mausoleum wins through a rebuild that
// gives a sixth symbol (wonders-tokens.md "The Mausoleum").
func TestPreviewWonderMausoleumWin(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     2,
		Wonders: wonderFill([]string{"mausoleum"}, nil),
		Players: [2]ScenarioPlayer{{Coins: intPtr(30), City: []string{"workshop", "pharmacist", "apothecary", "university", "academy"}}},
		Discard: []string{"baths", "library"},
	})
	p := previewWonder(&s, 0, hnWonder(t, "mausoleum"))
	want := Win{Kind: VictoryScience, Steps: 1, Via: ReviveAction(hnCard(t, "library"))}
	if !p.Wins || p.Win != want || p.Pool != 2 {
		t.Fatalf("the Mausoleum rebuilding Library: %+v", p)
	}
}

// --- extraTurnOutcome ---

// TestExtraTurnOutcome: a wonder grants one extra turn, by its own effect
// or by Theology, never two, and an extra turn taken with the age's last
// card is lost (turn.md "Extra turns").
func TestExtraTurnOutcome(t *testing.T) {
	sc := Scenario{
		Age:      1,
		Wonders:  wonderFill([]string{"sphinx", "pyramids"}, nil),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(40)}},
		OnlyLeft: []string{"theater", "baths"},
		ToMove:   seatPtr(0),
	}
	s := hnNew(t, sc)
	check := func(s *State, key string, state ExtraTurnState, src ExtraTurnSource) {
		t.Helper()
		if gs, gsrc := extraTurnOutcome(s, hnKey(t, key)); gs != state || gsrc != src {
			t.Errorf("%s: %v, %v; want %v, %v", key, gs, gsrc, state, src)
		}
	}
	check(&s, "wonder:sphinx:theater", ExtraTurnGain, ExtraTurnSourceWonder)
	check(&s, "wonder:pyramids:theater", ExtraTurnNone, ExtraTurnSourceNone)
	check(&s, "construct:theater", ExtraTurnNone, ExtraTurnSourceNone)
	s.Players[0].Tokens.Add(hnToken(t, "theology"))
	check(&s, "wonder:pyramids:theater", ExtraTurnGain, ExtraTurnSourceTheology)
	check(&s, "wonder:sphinx:theater", ExtraTurnGain, ExtraTurnSourceWonder)

	sc.OnlyLeft = []string{"theater"}
	last := hnNew(t, sc)
	check(&last, "wonder:sphinx:theater", ExtraTurnLost, ExtraTurnSourceWonder)
	// The static answer agrees with the rules engine.
	hnDo(t, &last, "wonder:sphinx:theater")
	if last.PlayingAgain || last.Age != 2 {
		t.Fatal("the extra turn was not lost")
	}
}

// --- priceShift and savesInView ---

// TestPriceShift: fixed production raises the other seat's price, a fixed
// price lowers the builder's own, and flexible production changes nothing
// (economy.md "Trading", "Fixed prices", "Flexible production").
func TestPriceShift(t *testing.T) {
	s := hnNew(t, Scenario{Age: 2, Players: [2]ScenarioPlayer{{City: []string{"clay-pool"}}, {}}})
	got := priceShift(&s, 0, hnCard(t, "brickyard"))
	if got.N != 1 || got.E[0] != (priceShiftEntry{Seat: 1, Res: catalog.Clay, From: 3, To: 5}) {
		t.Fatalf("Brickyard: %+v", got.E[:got.N])
	}
	got = priceShift(&s, 1, hnCard(t, "customs-house"))
	if got.N != 2 || got.E[0] != (priceShiftEntry{Seat: 1, Res: catalog.Glass, From: 2, To: 1}) || got.E[1].Res != catalog.Papyrus {
		t.Fatalf("the Customs House: %+v", got.E[:got.N])
	}
	if got = priceShift(&s, 0, hnCard(t, "forum")); got.N != 0 {
		t.Fatalf("the Forum: %+v", got.E[:got.N])
	}
	if got = priceShift(&s, 0, hnCard(t, "clay-pool")); got.N != 0 {
		t.Fatal("a card already in the city")
	}
}

// TestSavesInView: a producer saves its builder the trading coins on the
// cards and wonders in view (economy.md "The cheapest payment").
func TestSavesInView(t *testing.T) {
	s := hnNew(t, Scenario{
		Age:     1,
		Wonders: wonderFill([]string{"pyramids"}, nil),
		Players: [2]ScenarioPlayer{{Coins: intPtr(30)}, {City: []string{"quarry"}}},
	})
	targets := []catalog.CardID{hnCard(t, "baths"), hnCard(t, "garrison")}
	// Seat 0's open wonders are the Pyramids (3 stone, 1 papyrus), the
	// Colossus (3 clay, 1 glass), the Great Lighthouse (1 wood, 1 stone, 2
	// papyrus) and the Temple of Artemis (1 each of wood, stone, glass and
	// papyrus). Seat 1's Quarry makes seat 0's stone cost 3.
	// Stone Pit saves one stone (3 coins) on Baths, the Pyramids, the Great
	// Lighthouse and the Temple of Artemis, and nothing on Garrison (clay).
	if got := savesInView(&s, 0, hnCard(t, "stone-pit"), targets); got != 4*3 {
		t.Fatalf("Stone Pit: saves %d, want 12", got)
	}
	// Press makes one papyrus per payment: it saves 2 coins on the
	// Pyramids, the Great Lighthouse (which still buys its second papyrus)
	// and the Temple of Artemis.
	if got := savesInView(&s, 0, hnCard(t, "press"), targets); got != 3*2 {
		t.Fatalf("Press: saves %d, want 6", got)
	}
	ids, n := faceUpTargets(&s)
	if n == 0 || n > SlotsPerAge || ids[0] == 0 {
		t.Fatalf("face-up targets: %d", n)
	}
}

// --- WinsNow ---

// TestWinsNow: a win at once, a win in the mover's own extra turn, no win
// when the extra turn is lost, the face-up mask, and the budget.
func TestWinsNow(t *testing.T) {
	sc := Scenario{
		Age:      1,
		Wonders:  wonderFill([]string{"sphinx"}, nil),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Pawn:     intPtr(8),
		OnlyLeft: []string{"guard-tower", "baths"},
		ToMove:   seatPtr(0),
	}
	s := hnNew(t, sc)
	mask := FaceUpSlots(&s)
	check := func(key string, mask SlotMask, want Win, ok bool) {
		t.Helper()
		before := s
		got, gotOK := WinsNow(&s, hnKey(t, key), mask)
		hnUnchanged(t, &before, &s, "WinsNow")
		if got != want || gotOK != ok {
			t.Errorf("%s: %+v %v, want %+v %v", key, got, gotOK, want, ok)
		}
	}
	check("construct:guard-tower", mask, Win{Kind: VictoryMilitary, Steps: 1}, true)
	check("wonder:sphinx:baths", mask, Win{Kind: VictoryMilitary, Steps: 2, Via: hnKey(t, "construct:guard-tower")}, true)
	check("discard:baths", mask, Win{}, false)
	tower := SlotOf(&s, hnCard(t, "guard-tower"))
	check("wonder:sphinx:baths", mask&^(1<<tower), Win{}, false)
	check("construct:guard-tower", mask&^(1<<tower), Win{}, false)

	ctx := &simCtx{budget: 1}
	if _, ok := winsNow(&s, hnKey(t, "wonder:sphinx:baths"), mask, ctx); ok || ctx.applies != 1 {
		t.Fatalf("a budget of one Apply: %v after %d", ok, ctx.applies)
	}

	sc.OnlyLeft = []string{"baths"}
	sc.Pawn = intPtr(8)
	last := hnNew(t, sc)
	if _, ok := WinsNow(&last, hnKey(t, "wonder:sphinx:baths"), FaceUpSlots(&last)); ok {
		t.Fatal("an extra turn lost on the last card wins nothing")
	}
}

// --- Frame size and benchmarks ---

// hnMidAge returns seed's game played by random moves until ten cards of
// the given age are left.
func hnMidAge(t testing.TB, seed uint64, age uint8) State {
	return hnPlayRandom(t, seed, Options{}, func(s *State) bool {
		return s.Age == age && s.Phase == PhasePlay && hnCount(Occupied(s)) <= 10
	})
}

func hnCount(m uint32) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// TestHintsFrameSize logs the size of mid-age views with and without the
// hints, for docs/architecture.md, and bounds the growth: the shared hints
// (the per-card hints of M10.HS2, both seats, every face-up card, and the
// board-level hints of M10.HS4) may add at most 60% to the view, and the
// decider's outcomes of M10.HS3 (Choice.outcome, DisabledChoice.wins) at
// most a sixth more. The gzip sizes are logged for comparison.
func TestHintsFrameSize(t *testing.T) {
	for seed := uint64(1); seed <= 3; seed++ {
		for _, age := range []uint8{1, 2} {
			s := hnMidAge(t, seed, age)
			h := ComputeHints(&s, hnAllHints)
			plain, _ := json.Marshal(ViewFor(&s, SeatViewer(s.ToMove)))
			vw := ViewWithHints(&s, SeatViewer(s.ToMove), &h)
			with, _ := json.Marshal(vw)
			vw.Choices = slices.Clone(vw.Choices)
			for i := range vw.Choices {
				vw.Choices[i].Outcome = nil
			}
			vw.Disabled = slices.Clone(vw.Disabled)
			for i := range vw.Disabled {
				vw.Disabled[i].Wins = nil
			}
			shared, _ := json.Marshal(vw)
			t.Logf("seed %d, age %d, ply %d: decider view %d bytes, %d with the shared hints (+%d), %d with the outcomes (+%d more, %d choices)",
				seed, age, s.Ply, len(plain), len(shared), len(shared)-len(plain), len(with), len(with)-len(shared), len(vw.Choices))
			// The board-level hints of M10.HS4 on their own: HintsView without
			// the Mausoleum pool, PlayerView.hints and WonderView.effect.
			board := *vw.Hints
			board.Discard = nil
			nb, _ := json.Marshal(board)
			np, nw := 0, 0
			for i := range vw.Players {
				b, _ := json.Marshal(vw.Players[i].Hints)
				np += len(b)
				for _, w := range vw.Players[i].Wonders {
					if w.Effect != nil {
						b, _ := json.Marshal(w.Effect)
						nw += len(b)
					}
				}
			}
			t.Logf("  of which the board hints (HS4): %d bytes: HintsView %d, the players' hints %d, the wonder effects %d; gzip %d to %d",
				len(nb)+np+nw, len(nb), np, nw, hnGzip(plain), hnGzip(with))
			if 5*len(shared) > 8*len(plain) {
				t.Errorf("the shared hints add %d bytes to a %d-byte view", len(shared)-len(plain), len(plain))
			}
			if 6*(len(with)-len(shared)) > len(plain) {
				t.Errorf("the outcomes add %d bytes to a %d-byte view", len(with)-len(shared), len(plain))
			}
		}
	}
}

// BenchmarkComputeHints measures the shared hints of one broadcast: the
// hints themselves (the per-card hints since M10.HS2) with the three views of a frame, and
// the projected work of the later slices with the helpers above: a
// previewBuild, a priceShift and a savesInView for each face-up card and
// seat, a previewWonder for each open wonder, a tokenPreview for each board
// token and seat, and publicPlace for every card.
func BenchmarkComputeHints(b *testing.B) {
	s := hnMidAge(b, 2, 2)
	b.Run("frame", func(b *testing.B) {
		for b.Loop() {
			h := ComputeHints(&s, hnAllHints)
			for _, v := range []Viewer{Spectator, 0, 1} {
				_ = ViewWithHints(&s, v, &h)
			}
		}
	})
	b.Run("projected", func(b *testing.B) {
		for b.Loop() {
			ids, n := faceUpTargets(&s)
			for seat := range Seat(2) {
				for _, c := range ids[:n] {
					_ = previewBuild(&s, seat, c, false)
					_ = priceShift(&s, seat, c)
					_ = savesInView(&s, seat, c, ids[:n])
				}
				for k := range s.Players[seat].Wonders {
					if wonderOpen(&s, seat, k) {
						_ = previewWonder(&s, seat, s.Players[seat].Wonders[k].ID)
					}
				}
				for t, ok := s.BoardTokens.Next(1); ok; t, ok = s.BoardTokens.Next(t + 1) {
					_ = tokenPreview(&s, seat, t)
				}
			}
			for c, ok := catalog.T.All.Next(1); ok; c, ok = catalog.T.All.Next(c + 1) {
				_ = publicPlace(&s, c)
			}
		}
	})
}

// hnBusiest returns the play state with the most legal actions in the
// first states of seed's game: a heavy case for the decider's hints.
func hnBusiest(tb testing.TB, seed uint64) State {
	var best State
	most := -1
	buf := make([]Action, 0, MaxLegalActions)
	hnPlayRandom(tb, seed, Options{}, func(s *State) bool {
		if n := len(LegalActions(s, buf[:0])); s.Phase == PhasePlay && n > most {
			best, most = *s, n
		}
		return false
	})
	return best
}

// BenchmarkDeciderHints measures the decider's hints of one view
// (hints_decider.go): every legal action's outcome, the disabled wins, the
// threats one turn ahead and the reveal risk, under the Apply budget, in
// the play state with the most legal actions of a seeded game ("busiest")
// and in a mid-Age II state ("midAge").
func BenchmarkDeciderHints(b *testing.B) {
	for _, c := range []struct {
		name string
		s    State
	}{{"busiest", hnBusiest(b, 2)}, {"midAge", hnMidAge(b, 2, 2)}} {
		b.Run(c.name, func(b *testing.B) {
			s := c.s
			base := ViewFor(&s, SeatViewer(s.ToMove))
			applies := 0
			for b.Loop() {
				vw := base
				vw.Choices = slices.Clone(base.Choices)
				vw.Disabled = slices.Clone(base.Disabled)
				ctx := &simCtx{budget: hintApplyBudget}
				deciderHints(&s, hnAllHints, &vw, ctx)
				applies = ctx.applies
			}
			b.ReportMetric(float64(len(base.Choices)), "actions")
			b.ReportMetric(float64(applies), "applies")
		})
	}
}

// hnGzip returns the size of b compressed with gzip at the default level.
func hnGzip(b []byte) int {
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	_, _ = z.Write(b)
	_ = z.Close()
	return buf.Len()
}
