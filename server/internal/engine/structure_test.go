package engine

import (
	"math/bits"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Tests of the structure (dealing, accessibility, reveals) and of the city
// bookkeeping in city.go. They are internal tests: some build states mid-turn
// by hand.

// rowMask returns the slots of the given rows of an age.
func rowMask(age uint8, rows ...int8) uint32 {
	var m uint32
	for i := range SlotsPerAge {
		for _, r := range rows {
			if SlotPos(age, i).Row == r {
				m |= 1 << i
			}
		}
	}
	return m
}

// faceUpSlots returns the slots of s that are face up.
func faceUpSlots(s *State) uint32 {
	var m uint32
	for i := range s.Slots {
		if s.Slots[i].FaceUp {
			m |= 1 << i
		}
	}
	return m
}

// structureAt returns a state in the play phase at the start of an age, with
// the draft completed through the real flow (each pick takes the first
// wonder on offer) and the earlier ages played by discarding the first
// accessible card each turn.
func structureAt(t *testing.T, seed uint64, age uint8) State {
	t.Helper()
	s := NewGame(seed, Options{})
	buf := make([]Action, 0, MaxLegalActions)
	for s.Phase == PhaseWonderDraft {
		ApplyTrusted(&s, LegalActions(&s, buf[:0])[0])
	}
	for s.Age < age || s.Phase != PhasePlay {
		legal := LegalActions(&s, buf[:0])
		a := legal[0]
		for _, x := range legal {
			if x.Type == ActDiscardBuilding {
				a = x
				break
			}
		}
		if err := Apply(&s, a, nil); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

// TestInitialStructures checks each age's structure as dealt: all 20 cards,
// the odd rows face up and the even rows face down, and 6, 2 and 2
// accessible cards, all face up.
func TestInitialStructures(t *testing.T) {
	cases := []struct {
		age        uint8
		up, down   []int8 // rows
		accessible uint32
	}{
		{1, []int8{1, 3, 5}, []int8{2, 4}, rowMask(1, 5)},
		{2, []int8{1, 3, 5}, []int8{2, 4}, rowMask(2, 5)},
		{3, []int8{1, 3, 5, 7}, []int8{2, 4, 6}, rowMask(3, 7)},
	}
	wantAcc := [4]int{0, 6, 2, 2}
	for _, c := range cases {
		for seed := uint64(1); seed <= 5; seed++ {
			s := structureAt(t, seed, c.age)
			if s.Age != c.age || Occupied(&s) != allSlots {
				t.Fatalf("age %d seed %d: age %d with %d cards", c.age, seed, s.Age, bits.OnesCount32(Occupied(&s)))
			}
			if got, want := faceUpSlots(&s), rowMask(c.age, c.up...); got != want {
				t.Errorf("age %d seed %d: face up %020b, want %020b", c.age, seed, got, want)
			}
			if got := allSlots &^ faceUpSlots(&s); got != rowMask(c.age, c.down...) || bits.OnesCount32(got) != 8 {
				t.Errorf("age %d seed %d: face down %020b", c.age, seed, got)
			}
			acc := Accessible(&s)
			if acc != c.accessible || bits.OnesCount32(acc) != wantAcc[c.age] {
				t.Errorf("age %d seed %d: accessible %020b, want %020b", c.age, seed, acc, c.accessible)
			}
			if acc&^faceUpSlots(&s) != 0 {
				t.Errorf("age %d seed %d: an accessible card is face down", c.age, seed)
			}
			guilds := 0
			for i, sl := range s.Slots {
				if sl.Back != cardBack(sl.Card) || catalog.Card(sl.Card).Age != c.age {
					t.Errorf("age %d seed %d: slot %d holds %s with back %s", c.age, seed, i, sl.Card, sl.Back)
				}
				if sl.Back == catalog.GuildBack {
					guilds++
				}
			}
			if want := map[bool]int{true: GuildsInPlay, false: 0}[c.age == 3]; guilds != want {
				t.Errorf("age %d seed %d: %d guild backs, want %d", c.age, seed, guilds, want)
			}
			if s.Deals[c.age-1] != ([SlotsPerAge]catalog.CardID{}) {
				t.Errorf("age %d seed %d: the deal was not cleared", c.age, seed)
			}
		}
	}
}

// TestStartAgeCopiesTheDeal checks that startAge lays the deal out in
// layout order and reports the age.
func TestStartAgeCopiesTheDeal(t *testing.T) {
	for age := uint8(1); age <= NumAges; age++ {
		s := NewGame(11, Options{})
		deal := s.Deals[age-1]
		var ev []Event
		startAge(&s, age, &ev)
		for i := range s.Slots {
			if s.Slots[i].Card != deal[i] {
				t.Fatalf("age %d: slot %d holds %s, the deal had %s", age, i, s.Slots[i].Card, deal[i])
			}
		}
		if len(ev) != 1 || ev[0] != (Event{Kind: EvAgeStart, A: int16(age)}) {
			t.Errorf("age %d: events %+v", age, ev)
		}
	}
}

// takeAt discards the card at (row, col) for the player to move and returns
// the events.
func takeAt(t *testing.T, s *State, row, col int8) []Event {
	t.Helper()
	i := mustSlot(t, s.Age, row, col)
	var ev []Event
	if err := Apply(s, DiscardAction(s.Slots[i].Card), &ev); err != nil {
		t.Fatalf("discard (%d,%d): %v", row, col, err)
	}
	return ev
}

// reveals returns the slots revealed by the events, in event order.
func reveals(ev []Event) []int {
	var out []int
	for _, e := range ev {
		if e.Kind == EvReveal {
			out = append(out, int(e.A))
		}
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRevealSequencesInPlay plays known take orders and checks which cards turn
// face up, and when.
func TestRevealSequencesInPlay(t *testing.T) {
	type take struct {
		row, col int8
		reveal   [][2]int8 // the slots (row, col) revealed at the end of this turn
	}
	cases := []struct {
		age   uint8
		takes []take
	}{
		{1, []take{
			{5, 1, nil},
			{5, 3, [][2]int8{{4, 2}}},
			{5, 5, [][2]int8{{4, 4}}},
			{4, 2, nil}, // (3,3) is on a face-up row
			{5, 11, nil},
			{5, 9, [][2]int8{{4, 10}}},
		}},
		{2, []take{
			{5, 5, [][2]int8{{4, 4}}},
			{5, 7, [][2]int8{{4, 6}, {4, 8}}},
			{4, 4, nil}, // frees (3,3), on a face-up row
			{3, 3, [][2]int8{{2, 2}}},
		}},
		{3, []take{
			{7, 5, [][2]int8{{6, 4}}},
			{7, 7, [][2]int8{{6, 6}, {6, 8}}},
			{6, 4, nil},
			{5, 3, nil},
			{6, 6, nil},
			{5, 5, [][2]int8{{4, 4}}}, // the gap row: (4,4) is freed by (5,3) and (5,5)
			{4, 4, nil},               // frees (3,3) and (3,5), which are face up
		}},
	}
	for _, c := range cases {
		s := structureAt(t, 21, c.age)
		for k, tk := range c.takes {
			var want []int
			for _, p := range tk.reveal {
				want = append(want, mustSlot(t, c.age, p[0], p[1]))
			}
			got := reveals(takeAt(t, &s, tk.row, tk.col))
			if !sameInts(got, want) {
				t.Fatalf("age %d take %d (%d,%d): revealed slots %v, want %v", c.age, k, tk.row, tk.col, got, want)
			}
			for _, i := range want {
				if !s.Slots[i].FaceUp {
					t.Fatalf("age %d: revealed slot %d is still face down", c.age, i)
				}
			}
			if err := CheckInvariants(&s); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestRevealsAreTheNewlyAccessibleFaceDownCards plays random games and
// checks that each turn reveals exactly the face-down cards it made
// accessible, in slot order, and nothing else.
func TestRevealsAreTheNewlyAccessibleFaceDownCards(t *testing.T) {
	buf := make([]Action, 0, MaxLegalActions)
	for seed := uint64(1); seed <= 40; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		for !s.Over() {
			before := s
			legal := LegalActions(&s, buf[:0])
			var ev []Event
			if err := Apply(&s, legal[r.IntN(len(legal))], &ev); err != nil {
				t.Fatal(err)
			}
			if s.Age != before.Age || before.Age == 0 {
				continue // a new age was laid out (or the draft ended): nothing to compare
			}
			var want []int
			acc := Accessible(&s)
			for i := range s.Slots {
				if acc&(1<<i) != 0 && !before.Slots[i].FaceUp {
					want = append(want, i)
				}
			}
			if got := reveals(ev); !sameInts(got, want) {
				t.Fatalf("seed %d ply %d: revealed %v, want %v", seed, s.Ply, got, want)
			}
			for _, e := range ev {
				if e.Kind == EvReveal && s.Slots[e.A].Card != e.Card {
					t.Fatalf("seed %d: reveal of slot %d names %s, the slot holds %s", seed, e.A, e.Card, s.Slots[e.A].Card)
				}
			}
		}
	}
}

// TestRevealWaitsForThePendingChoice builds a turn that opened a pending
// choice and checks that the card it freed turns face up only once the
// choice is made, after the choice's own events.
func TestRevealWaitsForThePendingChoice(t *testing.T) {
	s := structureAt(t, 5, 1)
	takeAt(t, &s, 5, 1) // seat A
	takeAt(t, &s, 5, 7) // seat B, frees nothing face down
	// Seat A takes (5,3), which frees the face-down (4,2), and its effect
	// opens a token choice (built by hand: the S4 hooks open none).
	seat := s.Active
	i := mustSlot(t, 1, 5, 3)
	c := takeSlot(&s, i)
	addCard(&s, seat, c)
	s.Phase = PhasePickBoardToken
	s42 := mustSlot(t, 1, 4, 2)
	if Accessible(&s)&(1<<s42) == 0 || s.Slots[s42].FaceUp {
		t.Fatal("(4,2) should be accessible and still face down while the choice is open")
	}
	if Accessible(&s) == 0 {
		t.Fatal("no accessible card")
	}
	tok, _ := s.BoardTokens.Next(1)
	var ev []Event
	if err := Apply(&s, TokenAction(tok), &ev); err != nil {
		t.Fatal(err)
	}
	got := reveals(ev)
	if !sameInts(got, []int{s42}) || !s.Slots[s42].FaceUp {
		t.Fatalf("after the choice: revealed %v, want [%d]", got, s42)
	}
	if ev[len(ev)-1].Kind != EvReveal {
		t.Errorf("the reveal is not the last event of the turn: %+v", ev)
	}
	if s.Phase != PhasePlay || s.ToMove != seat.Opp() {
		t.Errorf("after the choice: phase %s, seat %d to move", s.Phase, s.ToMove)
	}
}

func TestAccessibleAndSlotOf(t *testing.T) {
	s := NewGame(3, Options{})
	if Accessible(&s) != 0 || Occupied(&s) != 0 {
		t.Fatal("the draft has a structure")
	}
	if SlotOf(&s, 0) != -1 {
		t.Fatal("SlotOf(0) found a slot")
	}
	s = structureAt(t, 3, 2)
	for i, sl := range s.Slots {
		if SlotOf(&s, sl.Card) != i {
			t.Fatalf("SlotOf(%s) = %d, want %d", sl.Card, SlotOf(&s, sl.Card), i)
		}
	}
	takeSlot(&s, 18)
	if Accessible(&s) != AccessibleMask(2, Occupied(&s)) || Occupied(&s) != allSlots&^(1<<18) {
		t.Fatal("Accessible disagrees with AccessibleMask")
	}
	if s.Slots[18] != (Slot{}) {
		t.Fatal("a taken slot is not zero")
	}
}

// cityState returns a state in the play phase of Age II with empty cities.
func cityState(t *testing.T) State {
	t.Helper()
	s, err := NewGameFromScenario(1, Options{}, &Scenario{Age: 2})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func cardID(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("no card %q", slug)
	}
	return c
}

// checkCaches compares p's caches with a recompute.
func checkCaches(t *testing.T, p *Player, what string) {
	t.Helper()
	fresh := *p
	recomputeCaches(&fresh)
	if fresh != *p {
		t.Fatalf("%s: caches %s, recompute gives %s", what, cacheString(p), cacheString(&fresh))
	}
}

// TestCityCaches adds and removes cards, tokens and wonders and checks each
// cache against its definition.
func TestCityCaches(t *testing.T) {
	s := cityState(t)
	p := &s.Players[0]
	for _, slug := range []string{"lumber-yard", "sawmill", "glassworks", "forum", "stone-reserve", "customs-house", "scriptorium", "library", "stable", "caravansery"} {
		addCard(&s, 0, cardID(t, slug))
		checkCaches(t, p, "after "+slug)
	}
	if p.Prod != [catalog.NumResources]uint8{catalog.Wood: 3, catalog.Glass: 1} {
		t.Errorf("Prod = %v", p.Prod)
	}
	if p.NChoice != 2 || p.Choice[0] != catalog.Manufactured || p.Choice[1] != catalog.RawGoods {
		t.Errorf("Choice = %v (n %d): want the Forum (39) then the Caravansery (40)", p.Choice, p.NChoice)
	}
	if p.TradeFix != catalog.MaskOf(catalog.Stone, catalog.Glass, catalog.Papyrus) {
		t.Errorf("TradeFix = %#x", p.TradeFix)
	}
	if p.ColorN[catalog.Brown] != 2 || p.ColorN[catalog.Yellow] != 4 || p.ColorN[catalog.Green] != 2 || p.ColorN[catalog.Red] != 1 {
		t.Errorf("ColorN = %v", p.ColorN)
	}
	if p.Sci[catalog.Quill] != 2 {
		t.Errorf("Sci = %v", p.Sci)
	}
	if p.Chains&catalog.ChainBook.Bit() == 0 || p.Chains&catalog.ChainHorseshoe.Bit() == 0 {
		t.Errorf("Chains = %#x", p.Chains)
	}

	removeCard(&s, 0, cardID(t, "sawmill"))
	removeCard(&s, 0, cardID(t, "forum"))
	checkCaches(t, p, "after removing")
	if p.Prod[catalog.Wood] != 1 || p.NChoice != 1 || p.Choice[0] != catalog.RawGoods {
		t.Errorf("after removing: Prod %v, Choice %v", p.Prod, p.Choice)
	}

	addToken(&s, 0, catalog.TokenLaw)
	if p.Sci[catalog.Scales] != 1 {
		t.Errorf("Law gave Sci %v", p.Sci)
	}
	addToken(&s, 0, catalog.TokenMasonry)
	checkCaches(t, p, "after tokens")

	// A producing wonder joins the flexible producers after the cards.
	k := 1
	p.Wonders[k] = WonderSlot{ID: catalog.WonderGreatLighthouse}
	buildWonder(&s, 0, k, cardID(t, "tavern"))
	checkCaches(t, p, "after the wonder")
	if s.WondersBuilt != 1 || !p.Wonders[k].Built || p.Wonders[k].Tucked != cardID(t, "tavern") {
		t.Errorf("buildWonder: built %d, slot %+v", s.WondersBuilt, p.Wonders[k])
	}
	if p.NChoice != 2 || p.Choice[1] != catalog.RawGoods {
		t.Errorf("Choice after the Great Lighthouse = %v", p.Choice)
	}
}

// TestSevenWonderRule checks the bookkeeping of the seven-wonder rule.
func TestSevenWonderRule(t *testing.T) {
	s := cityState(t)
	var ev []Event
	built := 0
	for seat := range Seat(2) {
		for k := range s.Players[seat].Wonders {
			if built == 6 {
				break
			}
			buildWonder(&s, seat, k, catalog.CardID(1+built))
			built++
		}
	}
	sevenWonderRule(&s, &ev)
	if len(ev) != 0 {
		t.Fatalf("the rule acted after six wonders: %+v", ev)
	}
	// Seat 1 holds two unbuilt wonders; build one, the other is removed.
	last, other := -1, -1
	for k, ws := range s.Players[1].Wonders {
		if !ws.Built {
			if last < 0 {
				last = k
			} else {
				other = k
			}
		}
	}
	buildWonder(&s, 1, last, catalog.CardID(7))
	sevenWonderRule(&s, &ev)
	sevenWonderRule(&s, &ev) // idempotent
	w := s.Players[1].Wonders[other]
	if !w.Removed || w.Built {
		t.Fatalf("the last wonder is %+v", w)
	}
	if len(ev) != 1 || ev[0] != (Event{Kind: EvWonderRemoved, Seat: 1, Wonder: w.ID}) {
		t.Fatalf("events %+v", ev)
	}
}

// TestGainCoinsSaturates checks the coin cap.
func TestGainCoinsSaturates(t *testing.T) {
	var s State
	s.Players[0].Coins = maxInt16 - 1
	gainCoins(&s, 0, 5)
	if s.Players[0].Coins != maxInt16 {
		t.Fatalf("coins = %d", s.Players[0].Coins)
	}
}

// midGame plays a seeded random game until Age II is under way (four cards
// taken) with a card in each city and at least one wonder built.
func midGame(t *testing.T) State {
	t.Helper()
	buf := make([]Action, 0, MaxLegalActions)
	for seed := uint64(1); seed <= 50; seed++ {
		s := NewGame(seed, Options{})
		r := NewRNG(seed)
		for !s.Over() {
			if s.Age == 2 && s.Phase == PhasePlay && s.WondersBuilt > 0 && bits.OnesCount32(Occupied(&s)) <= SlotsPerAge-4 &&
				!s.Players[0].City.Empty() && !s.Players[1].City.Empty() && !s.BoardTokens.Empty() && s.BoxTokens[1] != 0 {
				return s
			}
			legal := LegalActions(&s, buf[:0])
			if err := Apply(&s, legal[r.IntN(len(legal))], nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Fatal("no random game reached the wanted Age II position")
	return State{}
}

// TestInvariantsCatchCorruption breaks one rule at a time in otherwise good
// states and checks that CheckInvariants names it. Without it, a checker
// that always passed would let every other test pass too.
func TestInvariantsCatchCorruption(t *testing.T) {
	mid := midGame(t)
	draft := NewGame(1, Options{})
	for _, base := range []State{mid, draft} {
		if err := CheckInvariants(&base); err != nil {
			t.Fatalf("the base state is broken: %v", err)
		}
	}
	occupied := func(s *State) int {
		for i, sl := range s.Slots {
			if sl.Card != 0 {
				return i
			}
		}
		t.Fatal("no occupied slot")
		return -1
	}
	empty := func(s *State) int {
		for i, sl := range s.Slots {
			if sl.Card == 0 {
				return i
			}
		}
		t.Fatal("no empty slot")
		return -1
	}
	accessible := func(s *State) int {
		acc := Accessible(s)
		return bits.TrailingZeros32(acc)
	}
	builtWonder := func(s *State) *WonderSlot {
		for seat := range s.Players {
			for k := range s.Players[seat].Wonders {
				if s.Players[seat].Wonders[k].Built {
					return &s.Players[seat].Wonders[k]
				}
			}
		}
		t.Fatal("no built wonder")
		return nil
	}
	boardToken := func(s *State) catalog.TokenID {
		tk, _ := s.BoardTokens.Next(0)
		return tk
	}
	cases := []struct {
		name   string
		base   *State
		mutate func(s *State)
		want   string
	}{
		{"a card in two places", &mid, func(s *State) { s.Discard.Add(s.Slots[occupied(s)].Card) }, "is in"},
		{"a card lost", &mid, func(s *State) { s.Removed[0][0] = 0 }, "is nowhere"},
		{"a future deal short", &mid, func(s *State) { s.OutOfPlay.Add(s.Deals[2][0]); s.Deals[2][0] = 0 }, "the age 3 deal holds 19"},
		{"a card of the wrong age", &mid, func(s *State) {
			i := occupied(s)
			s.Slots[i].Card, s.Deals[2][0] = s.Deals[2][0], s.Slots[i].Card
		}, "not a card of age 2"},
		{"a wrong back", &mid, func(s *State) { s.Slots[occupied(s)].Back = catalog.BackGuild }, "shows back"},
		{"an empty slot with data", &mid, func(s *State) { s.Slots[empty(s)].FaceUp = true }, "is not zero"},
		{"an accessible card face down", &mid, func(s *State) { s.Slots[accessible(s)].FaceUp = false }, "face down"},
		{"negative coins", &mid, func(s *State) { s.Players[0].Coins = -1 }, "-1 coins"},
		{"a stale cache", &mid, func(s *State) { s.Players[1].Prod[0]++ }, "caches are stale"},
		{"the pawn off the track", &mid, func(s *State) { s.Military = 10 }, "the pawn is at 10"},
		{"the pawn at a capital", &mid, func(s *State) { s.Military = -9; s.Looting = 0 }, "capital"},
		{"looting out of order", &mid, func(s *State) { s.Military = 0; s.Looting = LootPlus3 }, "+6 looting token is gone"},
		{"a looting token passed", &mid, func(s *State) { s.Military = 4; s.Looting = LootAll }, "+3 looting token is still there"},
		{"a wrong wonder count", &mid, func(s *State) { s.WondersBuilt++ }, "WondersBuilt"},
		{"a wonder built with no card", &mid, func(s *State) {
			w := builtWonder(s)
			s.OutOfPlay.Add(w.Tucked)
			w.Tucked = 0
		}, "no card tucked"},
		{"a wonder removed early", &mid, func(s *State) {
			for k := range s.Players[0].Wonders {
				if w := &s.Players[0].Wonders[k]; !w.Built {
					w.Removed = true
					return
				}
			}
		}, "removed with only"},
		{"a drafted wonder back on offer", &mid, func(s *State) { s.DraftTaken &^= 1 }, "still on offer"},
		{"a token twice", &mid, func(s *State) { s.Players[1].Tokens.Add(boardToken(s)) }, "appears 2 times"},
		{"a gap in the box", &mid, func(s *State) { s.BoxTokens[0] = 0 }, "gap"},
		{"the wrong phase for the age", &mid, func(s *State) { s.Phase = PhaseWonderDraft }, "in age 2"},
		{"the wrong seat to move", &mid, func(s *State) { s.ToMove = s.Active.Opp() }, "is to move in phase"},
		{"a destroy colour outside its phase", &mid, func(s *State) { s.Destroy = catalog.Brown }, "Destroy is"},
		{"a library offer outside its phase", &mid, func(s *State) {
			tk := boardToken(s)
			s.BoardTokens.Remove(tk)
			s.Library[0] = tk
		}, "Great Library offer"},
		{"a result during play", &mid, func(s *State) { s.Result.Winner = 1 }, "a result is set"},
		{"a slot during the draft", &draft, func(s *State) { s.Slots[0].FaceUp = true }, "during the wonder draft"},
		{"the wrong drafter", &draft, func(s *State) { s.ToMove = s.ToMove.Opp(); s.Active = s.ToMove }, "draft pick 0"},
		{"an extra turn in the draft", &draft, func(s *State) { s.ExtraTurn = true }, "extra turn is pending"},
	}
	for _, c := range cases {
		s := *c.base
		c.mutate(&s)
		err := CheckInvariants(&s)
		if err == nil {
			t.Errorf("%s: CheckInvariants passed", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v\n  does not mention %q", c.name, err, c.want)
		}
	}
}
