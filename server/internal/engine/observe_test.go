package engine_test

import (
	"fmt"
	"math"
	"math/bits"
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of observe.go: Observe hides exactly the hidden places;
// Determinize returns complete, valid states that agree with the
// observation (Observe(Determinize(o)) == o), puts only guilds in
// guild-backed slots, deals the future ages as setup does, and samples the
// hidden information uniformly (chi-square bounds over 5000 samples); and a
// chooser that sees only observations makes the same choice for two states
// that differ only in hidden information (the no-cheat test).
//
// The chi-square statistics are test arithmetic, not rules code, so they
// use floating point.

const (
	obsMaxPlies = 200
	obsSamples  = 5000 // samples per uniformity table
	obsPlayouts = 2    // random playouts per legal action in the no-cheat chooser
)

// obsGame plays a seeded game with random or simple policies and returns
// the state before every action and the final state.
func obsGame(tb testing.TB, seed uint64, simple bool) []engine.State {
	tb.Helper()
	s := engine.NewGame(seed, engine.Options{FirstPlayer: engine.FirstPlayerRule(seed % 3)})
	pols := [2]enginetest.Policy{enginetest.NewRandomPolicy(seed), enginetest.NewRandomPolicy(seed ^ 0xa5a5)}
	if simple {
		pols = [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed ^ 0xa5a5)}
	}
	states := []engine.State{s}
	err := enginetest.Playout(&s, pols, obsMaxPlies, func(engine.Action, []engine.Event) error {
		states = append(states, s)
		return nil
	})
	if err != nil {
		tb.Fatalf("game %d: %v", seed, err)
	}
	return states
}

// obsLibraryOpen returns a position in which seat 0 has just built the
// Great Library and keeps one of agriculture, law and philosophy, with
// architecture and mathematics left in the box.
func obsLibraryOpen(tb testing.TB) engine.State {
	tb.Helper()
	s := enginetest.New(tb, 21).Age(2).FirstPlayer(0).ToMove(0).Coins(0, 40).Accessible("rostrum").
		Wonders(0, "great-library").BoardTokens("economy", "masonry", "strategy", "theology", "urbanism").
		BoxOrder("agriculture", "law", "philosophy").State()
	enginetest.Do(tb, &s, "wonder:great-library:rostrum")
	if s.Phase != engine.PhasePickLibraryToken || s.ToMove != 0 {
		tb.Fatalf("the Great Library did not open its offer: phase %s, seat %d to move", s.Phase, s.ToMove)
	}
	return s
}

// obsCorpus returns the states of 24 seeded games, half with random and
// half with simple policies, plus a Great Library position, so that every
// phase occurs (TestObsCorpusCoversEveryPhase checks that).
func obsCorpus(tb testing.TB) []engine.State {
	tb.Helper()
	var all []engine.State
	for seed := uint64(1); seed <= 12; seed++ {
		all = append(all, obsGame(tb, 100+seed, false)...)
		all = append(all, obsGame(tb, 200+seed, true)...)
	}
	return append(all, obsLibraryOpen(tb))
}

func TestObsCorpusCoversEveryPhase(t *testing.T) {
	var n [engine.PhaseGameOver + 1]int
	for _, s := range obsCorpus(t) {
		n[s.Phase]++
	}
	for p, k := range n {
		if k == 0 {
			t.Errorf("the corpus has no state in phase %s", engine.Phase(p))
		}
	}
	t.Logf("states per phase: %v", n)
}

// obsMapIDs replaces every id of ids with f(id).
func obsMapIDs[T ~uint8](ids []T, f func(uint8) uint8) {
	for i := range ids {
		ids[i] = T(f(uint8(ids[i])))
	}
}

// obsMapHidden applies f to the id in every place hidden from seat, as the
// list at the top of observe.go names them: the cards of face-down slots,
// the deals, the removed cards, the box guilds, draft round 2 during round
// 1, the box wonders, the box tokens, and the Great Library offer unless
// seat is choosing from it.
func obsMapHidden(s *engine.State, seat engine.Seat, f func(uint8) uint8) {
	for i := range s.Slots {
		if !s.Slots[i].FaceUp {
			s.Slots[i].Card = catalog.CardID(f(uint8(s.Slots[i].Card)))
		}
	}
	for a := range s.Deals {
		obsMapIDs(s.Deals[a][:], f)
	}
	for a := range s.Removed {
		obsMapIDs(s.Removed[a][:], f)
	}
	obsMapIDs(s.BoxGuilds[:], f)
	if bits.OnesCount8(s.DraftTaken) < 4 {
		obsMapIDs(s.DraftOrder[4:], f)
	}
	obsMapIDs(s.BoxWonders[:], f)
	obsMapIDs(s.BoxTokens[:], f)
	if s.Phase != engine.PhasePickLibraryToken || s.ToMove != seat {
		obsMapIDs(s.Library[:], f)
	}
}

// obsHideID is what Observe makes of a hidden id: 0 stays 0 (an empty
// place), anything else becomes Unknown.
func obsHideID(id uint8) uint8 {
	if id == 0 {
		return 0
	}
	return catalog.Unknown
}

// obsDiff names the parts in which two states differ.
func obsDiff(got, want *engine.State) string {
	var b strings.Builder
	part := func(name string, g, w any) {
		if g != w {
			fmt.Fprintf(&b, "\n  %s: got %v\n  %*s want %v", name, g, len(name), "", w)
		}
	}
	part("seed", got.Seed, want.Seed)
	part("slots", got.Slots, want.Slots)
	part("deals", got.Deals, want.Deals)
	part("removed", got.Removed, want.Removed)
	part("box guilds", got.BoxGuilds, want.BoxGuilds)
	part("draft order", got.DraftOrder, want.DraftOrder)
	part("box wonders", got.BoxWonders, want.BoxWonders)
	part("box tokens", got.BoxTokens, want.BoxTokens)
	part("library", got.Library, want.Library)
	if b.Len() == 0 {
		b.WriteString("\n  (a public field differs)")
	}
	return b.String()
}

func TestObserveHidesExactlyTheHiddenParts(t *testing.T) {
	for _, s := range obsCorpus(t) {
		for seat := range engine.Seat(2) {
			before := s
			o := engine.Observe(&s, seat)
			if s != before {
				t.Fatalf("ply %d: Observe changed the state", s.Ply)
			}
			if o.Seat() != seat {
				t.Fatalf("ply %d: the observation is seat %d's, want %d", s.Ply, o.Seat(), seat)
			}
			want := s
			want.Seed = 0
			obsMapHidden(&want, seat, obsHideID)
			if got := o.Public(); *got != want {
				t.Fatalf("ply %d, phase %s, seat %d: the observation differs:%s", s.Ply, s.Phase, seat, obsDiff(got, &want))
			}
		}
	}
}

// obsAgeIIIWithHiddenGuild returns the start of Age III of the first
// scenario seed that deals a guild face down.
func obsAgeIIIWithHiddenGuild(t *testing.T) engine.State {
	t.Helper()
	for seed := uint64(1); seed <= 100; seed++ {
		s := enginetest.New(t, seed).Age(3).State()
		for _, sl := range s.Slots {
			if !sl.FaceUp && sl.Back == catalog.GuildBack {
				return s
			}
		}
	}
	t.Fatal("no scenario seed up to 100 deals a guild face down")
	return engine.State{}
}

// TestObserveExamples spells out the hiding rules on concrete positions.
func TestObserveExamples(t *testing.T) {
	unknownWonder := catalog.WonderID(catalog.Unknown)
	unknownToken := catalog.TokenID(catalog.Unknown)

	t.Run("draft", func(t *testing.T) {
		s := engine.NewGame(7, engine.Options{})
		o := engine.Observe(&s, 0)
		p := o.Public()
		if p.Seed != 0 || s.Seed != 7 {
			t.Errorf("seed %d in the observation, %d in the game", p.Seed, s.Seed)
		}
		if !slices.Equal(p.DraftOrder[:4], s.DraftOrder[:4]) {
			t.Errorf("round 1 is %v, want %v", p.DraftOrder[:4], s.DraftOrder[:4])
		}
		for i := 4; i < engine.NumDraftWonders; i++ {
			if p.DraftOrder[i] != unknownWonder {
				t.Errorf("round 2 wonder %d is visible during round 1: %s", i, p.DraftOrder[i])
			}
		}
		for i, w := range p.BoxWonders {
			if w != unknownWonder {
				t.Errorf("box wonder %d is visible: %s", i, w)
			}
		}
		if p.BoardTokens != s.BoardTokens {
			t.Errorf("board tokens %v, want %v", p.BoardTokens, s.BoardTokens)
		}
		for i, tk := range p.BoxTokens {
			if tk != unknownToken {
				t.Errorf("box token %d is visible: %s", i, tk)
			}
		}
		for a := range p.Deals {
			for i, c := range p.Deals[a] {
				if c != catalog.CardID(catalog.Unknown) {
					t.Fatalf("age %d deal place %d is visible: %s", a+1, i, c)
				}
			}
		}
		// After round 1 (the fourth pick is automatic), round 2 is public.
		for range 3 {
			enginetest.Do(t, &s, engine.LegalActions(&s, nil)[0].Key())
		}
		o = engine.Observe(&s, 1)
		p = o.Public()
		if !slices.Equal(p.DraftOrder[:], s.DraftOrder[:]) {
			t.Errorf("after round 1 the draft order is %v, want %v", p.DraftOrder, s.DraftOrder)
		}
	})

	t.Run("structure", func(t *testing.T) {
		s := obsAgeIIIWithHiddenGuild(t)
		o := engine.Observe(&s, 1)
		p := o.Public()
		faceDown, guilds := 0, 0
		for i, sl := range p.Slots {
			game := s.Slots[i]
			switch {
			case game.FaceUp:
				if sl != game {
					t.Errorf("face-up slot %d is %+v, want %+v", i, sl, game)
				}
			default:
				faceDown++
				if sl.Card != catalog.CardID(catalog.Unknown) || sl.Back != game.Back || sl.FaceUp {
					t.Errorf("face-down slot %d is %+v (the card is %s)", i, sl, game.Card)
				}
				if sl.Back == catalog.GuildBack {
					guilds++
					if !catalog.Card(game.Card).IsGuild() {
						t.Errorf("slot %d shows the guild back over %s", i, game.Card)
					}
				}
			}
		}
		if faceDown != 8 || guilds == 0 {
			t.Errorf("%d face-down slots (want 8), %d of them guilds (want some)", faceDown, guilds)
		}
		for i, c := range p.Removed[2] {
			if c != catalog.CardID(catalog.Unknown) {
				t.Errorf("removed card %d is visible: %s", i, c)
			}
		}
		for i, c := range p.BoxGuilds {
			if c != catalog.CardID(catalog.Unknown) {
				t.Errorf("box guild %d is visible: %s", i, c)
			}
		}
	})

	t.Run("great library", func(t *testing.T) {
		s := obsLibraryOpen(t)
		ownObs, otherObs := engine.Observe(&s, 0), engine.Observe(&s, 1)
		own := ownObs.Public()
		if own.Library != s.Library {
			t.Errorf("the drawer sees the offer as %v, want %v", own.Library, s.Library)
		}
		other := otherObs.Public()
		want := [engine.LibraryDraw]catalog.TokenID{unknownToken, unknownToken, unknownToken}
		if other.Library != want {
			t.Errorf("the opponent sees the offer as %v", other.Library)
		}
		wantBox := [engine.NumBoxTokens]catalog.TokenID{unknownToken, unknownToken}
		if own.BoxTokens != wantBox || other.BoxTokens != wantBox {
			t.Errorf("the box is %v for the drawer and %v for the opponent, want %v", own.BoxTokens, other.BoxTokens, wantBox)
		}
	})

	t.Run("bad seat", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("Observe for seat 2 did not panic")
			}
		}()
		s := engine.NewGame(1, engine.Options{})
		engine.Observe(&s, 2)
	})
}

// TestObserveKeepsLegalActions checks that the observation of the seat to
// move is enough to list its legal actions, and that the score preview and
// the discard values are public.
func TestObserveKeepsLegalActions(t *testing.T) {
	for _, s := range obsCorpus(t) {
		for seat := range engine.Seat(2) {
			o := engine.Observe(&s, seat)
			p := o.Public()
			if seat == s.ToMove {
				got, want := enginetest.Keys(p), enginetest.Keys(&s)
				if !slices.Equal(got, want) {
					t.Fatalf("ply %d, phase %s, seat %d: legal actions %v from the observation, %v from the game", s.Ply, s.Phase, seat, got, want)
				}
			}
			if engine.Scores(p) != engine.Scores(&s) {
				t.Fatalf("ply %d: the score preview differs", s.Ply)
			}
			if engine.DiscardValue(p, seat) != engine.DiscardValue(&s, seat) {
				t.Fatalf("ply %d: the discard value differs", s.Ply)
			}
		}
	}
}

func TestDeterminizeRoundTrip(t *testing.T) {
	r := engine.NewRNG(8)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for i, s := range obsCorpus(t) {
		for seat := range engine.Seat(2) {
			o := engine.Observe(&s, seat)
			before := o
			for k := range 2 {
				d := o.Determinize(&r)
				if o != before {
					t.Fatalf("ply %d: Determinize changed the observation", s.Ply)
				}
				if err := engine.CheckInvariants(&d); err != nil {
					t.Fatalf("ply %d, phase %s, seat %d: the sample is not valid: %v", s.Ply, s.Phase, seat, err)
				}
				if back := engine.Observe(&d, seat); back != o {
					t.Fatalf("ply %d, phase %s, seat %d: Observe(Determinize(o)) != o:%s", s.Ply, s.Phase, seat, obsDiff(back.Public(), o.Public()))
				}
				// The legal actions are public, except the opponent's
				// Great Library offer.
				if s.Phase != engine.PhasePickLibraryToken || seat == s.ToMove {
					if got, want := enginetest.Keys(&d), enginetest.Keys(&s); !slices.Equal(got, want) {
						t.Fatalf("ply %d: the sample's legal actions are %v, the game's %v", s.Ply, got, want)
					}
				}
				// Now and then, the sample must play on to the end.
				if k == 0 && i%9 == 0 {
					obsPlayToEnd(t, &d, &r, buf)
				}
			}
		}
	}
}

// obsPlayToEnd plays *s to the end with random moves, checking the
// invariants after every action.
func obsPlayToEnd(t *testing.T, s *engine.State, r *engine.RNG, buf []engine.Action) {
	t.Helper()
	for n := 0; !s.Over(); n++ {
		if n >= obsMaxPlies {
			t.Fatalf("a sample is not over after %d actions", n)
		}
		buf = engine.LegalActions(s, buf[:0])
		if err := engine.Apply(s, buf[r.IntN(len(buf))], nil); err != nil {
			t.Fatalf("a sample refused its own legal action: %v", err)
		}
		if err := engine.CheckInvariants(s); err != nil {
			t.Fatalf("a sample broke an invariant at ply %d: %v", s.Ply, err)
		}
	}
}

// obsCardsOf lists the cards of a set.
func obsCardsOf(set catalog.CardSet) []catalog.CardID { return slices.Collect(set.All()) }

func TestDeterminizeGuildSlots(t *testing.T) {
	var states []engine.State
	for seed := uint64(1); seed <= 20; seed++ {
		states = append(states, enginetest.New(t, seed).Age(3).State())
	}
	for _, s := range obsCorpus(t) {
		if s.Age == 3 {
			states = append(states, s)
		}
	}
	r := engine.NewRNG(3)
	guildSlots := 0
	for _, s := range states {
		o := engine.Observe(&s, s.ToMove)
		for range 50 {
			d := o.Determinize(&r)
			for i, sl := range d.Slots {
				if sl.Card == 0 || s.Slots[i].FaceUp {
					continue
				}
				def := catalog.Card(sl.Card)
				switch sl.Back {
				case catalog.GuildBack:
					guildSlots++
					if !def.IsGuild() {
						t.Fatalf("ply %d: guild-backed slot %d received %s", s.Ply, i, sl.Card)
					}
				case catalog.BackAge3:
					if def.IsGuild() || def.Age != 3 {
						t.Fatalf("ply %d: Age III slot %d received %s", s.Ply, i, sl.Card)
					}
				default:
					t.Fatalf("ply %d: slot %d shows back %s in Age III", s.Ply, i, sl.Back)
				}
			}
			for _, c := range d.BoxGuilds {
				if c != 0 && !catalog.Card(c).IsGuild() {
					t.Fatalf("ply %d: the box guilds received %s", s.Ply, c)
				}
			}
		}
	}
	if guildSlots == 0 {
		t.Fatal("no face-down guild slot was sampled; the test checks nothing")
	}
}

// TestDeterminizeFutureDeals checks that the ages still to come are dealt
// as setup deals them: 20 cards of the age in the deal and 3 removed, and
// for Age III exactly 3 guilds in the deal and 4 in the box.
func TestDeterminizeFutureDeals(t *testing.T) {
	r := engine.NewRNG(4)
	checked := 0
	for _, s := range obsCorpus(t) {
		if s.Age >= 3 || s.Phase == engine.PhasePickLibraryToken {
			continue // the Great Library position is a scenario
		}
		checked++
		o := engine.Observe(&s, s.ToMove)
		d := o.Determinize(&r)
		for age := int(s.Age) + 1; age <= engine.NumAges; age++ {
			guilds := 0
			for _, c := range d.Deals[age-1] {
				def := catalog.Card(c)
				if def == nil || def.Age != uint8(age) {
					t.Fatalf("ply %d: the age %d deal holds %s", s.Ply, age, c)
				}
				if def.IsGuild() {
					guilds++
				}
			}
			for _, c := range d.Removed[age-1] {
				if def := catalog.Card(c); def == nil || def.Age != uint8(age) || def.IsGuild() {
					t.Fatalf("ply %d: the age %d removed cards hold %s", s.Ply, age, c)
				}
			}
			if age == 3 && guilds != engine.GuildsInPlay {
				t.Fatalf("ply %d: the Age III deal holds %d guilds, want %d", s.Ply, guilds, engine.GuildsInPlay)
			}
		}
		for _, c := range d.BoxGuilds {
			if def := catalog.Card(c); def == nil || !def.IsGuild() {
				t.Fatalf("ply %d: the box guilds hold %s", s.Ply, c)
			}
		}
	}
	if checked < 100 {
		t.Fatalf("only %d states before Age III", checked)
	}
}

// obsChi2 is a table of observed counts against the expected probability
// of each cell: one row per item, one column per place, and each sample
// puts each item in exactly one place. Cells of probability 0 must stay
// empty.
type obsChi2 struct {
	name  string
	row   [256]int // row of each item id, or -1
	items []uint8
	count [][]int
	p     [][]float64
}

func obsNewChi2(name string, items []uint8, places int) *obsChi2 {
	c := &obsChi2{name: name, items: items}
	for i := range c.row {
		c.row[i] = -1
	}
	for i, id := range items {
		c.row[id] = i
	}
	c.count = make([][]int, len(items))
	c.p = make([][]float64, len(items))
	for i := range items {
		c.count[i] = make([]int, places)
		c.p[i] = make([]float64, places)
	}
	return c
}

// obsUniformChi2 expects every item to be equally likely in every place.
func obsUniformChi2(name string, items []uint8) *obsChi2 {
	c := obsNewChi2(name, items, len(items))
	for i := range c.p {
		for j := range c.p[i] {
			c.p[i][j] = 1 / float64(len(items))
		}
	}
	return c
}

// add counts item id in place j.
func (c *obsChi2) add(t *testing.T, id uint8, j int) {
	t.Helper()
	i := c.row[id]
	if i < 0 {
		t.Fatalf("%s: id %d is not an item of the table", c.name, id)
	}
	c.count[i][j]++
}

// stat returns Pearson's statistic, its expected value (the sum over the
// rows of the row's possible places minus one) and the bound the test
// allows: the expected value plus six standard deviations of a chi-square
// variable with that many degrees of freedom. The rows of a table of
// shuffles are not independent, which only narrows the spread, so the bound
// is generous; the seeds are fixed, so the outcome is deterministic.
func (c *obsChi2) stat(t *testing.T, samples int) (x, mean, bound float64) {
	t.Helper()
	for i := range c.count {
		sum := 0
		for j, o := range c.count[i] {
			sum += o
			e := float64(samples) * c.p[i][j]
			if e == 0 {
				if o != 0 {
					t.Fatalf("%s: item %d came up %d times in impossible place %d", c.name, c.items[i], o, j)
				}
				continue
			}
			x += (float64(o) - e) * (float64(o) - e) / e
			mean += 1 - c.p[i][j]
		}
		if sum != samples {
			t.Fatalf("%s: item %d was counted %d times in %d samples", c.name, c.items[i], sum, samples)
		}
	}
	return x, mean, mean + 6*math.Sqrt(2*mean)
}

// check fails the test when the statistic is above the bound.
func (c *obsChi2) check(t *testing.T, samples int) {
	t.Helper()
	x, mean, bound := c.stat(t, samples)
	if x > bound {
		t.Errorf("%s: chi-square %.1f is above the bound %.1f (expected about %.1f)", c.name, x, bound, mean)
	}
	t.Logf("%s: chi-square %.1f, expected about %.1f, bound %.1f", c.name, x, mean, bound)
}

// obsIDs converts ids to bytes.
func obsIDs[T ~uint8](ids []T) []uint8 {
	out := make([]uint8, len(ids))
	for i, id := range ids {
		out[i] = uint8(id)
	}
	return out
}

// TestObsChi2Detects checks that the uniformity statistic tells a correct
// shuffle from a classic wrong one (Sattolo's algorithm, which never leaves
// an item in its place) at the sample size the tests use.
func TestObsChi2Detects(t *testing.T) {
	items := make([]uint8, 11)
	for i := range items {
		items[i] = uint8(i + 1)
	}
	good, bad := obsUniformChi2("shuffle", items), obsUniformChi2("sattolo", items)
	r := engine.NewRNG(5)
	for range obsSamples {
		a := slices.Clone(items)
		engine.ShuffleSlice(&r, a)
		for j, id := range a {
			good.add(t, id, j)
		}
		b := slices.Clone(items)
		for i := len(b) - 1; i > 0; i-- {
			k := r.IntN(i) // the bug: IntN(i+1) is right
			b[i], b[k] = b[k], b[i]
		}
		for j, id := range b {
			bad.add(t, id, j)
		}
	}
	good.check(t, obsSamples)
	if x, _, bound := bad.stat(t, obsSamples); x <= bound {
		t.Errorf("the statistic missed a biased shuffle: %.1f is within the bound %.1f", x, bound)
	}
}

// obsDraftDone returns a game just after the wonder draft (every pick the
// first legal one): Age I is laid out and nothing of it has been taken.
func obsDraftDone(t *testing.T, seed uint64) engine.State {
	t.Helper()
	s := engine.NewGame(seed, engine.Options{})
	for s.Phase == engine.PhaseWonderDraft {
		enginetest.Do(t, &s, engine.LegalActions(&s, nil)[0].Key())
	}
	return s
}

func TestDeterminizeUniform(t *testing.T) {
	t.Run("draft", func(t *testing.T) {
		// Everything is hidden but round 1 of the draft and the board.
		s := engine.NewGame(31, engine.Options{})
		o := engine.Observe(&s, 0)
		wonders := obsUniformChi2("wonders: draft round 2 and the box", obsIDs(append(s.DraftOrder[4:], s.BoxWonders[:]...)))
		tokens := obsUniformChi2("tokens: the box order", obsIDs(s.BoxTokens[:]))
		ageI := obsUniformChi2("Age I: deal and removed", obsIDs(obsCardsOf(catalog.T.AgeCards[1])))
		r := engine.NewRNG(31)
		for range obsSamples {
			d := o.Determinize(&r)
			for j, w := range append(d.DraftOrder[4:], d.BoxWonders[:]...) {
				wonders.add(t, uint8(w), j)
			}
			for j, tk := range d.BoxTokens {
				tokens.add(t, uint8(tk), j)
			}
			for j, c := range append(d.Deals[0][:], d.Removed[0][:]...) {
				ageI.add(t, uint8(c), j)
			}
		}
		wonders.check(t, obsSamples)
		tokens.check(t, obsSamples)
		ageI.check(t, obsSamples)
	})

	t.Run("age I", func(t *testing.T) {
		s := obsDraftDone(t, 32)
		o := engine.Observe(&s, s.ToMove)
		// The current age: face-down slots, then the removed cards.
		var places []*catalog.CardID
		var hidden []catalog.CardID
		for i := range s.Slots {
			if !s.Slots[i].FaceUp {
				places = append(places, &s.Slots[i].Card)
			}
		}
		for i := range s.Removed[0] {
			places = append(places, &s.Removed[0][i])
		}
		for _, p := range places {
			hidden = append(hidden, *p)
		}
		slices.Sort(hidden)
		current := obsUniformChi2("Age I: face-down slots and removed", obsIDs(hidden))
		ageII := obsUniformChi2("Age II: deal and removed", obsIDs(obsCardsOf(catalog.T.AgeCards[2])))
		// Age III: each guild is dealt with probability 3/7, each Age III
		// card removed with probability 3/20, and each place of the deal
		// holds a guild with probability 3/20.
		guilds := obsNewChi2("Age III: guilds dealt or boxed", obsIDs(obsCardsOf(catalog.T.Guilds)), 2)
		for i := range guilds.p {
			guilds.p[i] = []float64{3.0 / 7, 4.0 / 7}
		}
		thirds := obsNewChi2("Age III: cards dealt or removed", obsIDs(obsCardsOf(catalog.T.AgeCards[3])), 2)
		for i := range thirds.p {
			thirds.p[i] = []float64{17.0 / 20, 3.0 / 20}
		}
		var placeIDs []uint8
		for j := range engine.SlotsPerAge {
			placeIDs = append(placeIDs, uint8(j+1))
		}
		dealPlaces := obsNewChi2("Age III: guilds per place of the deal", placeIDs, 2)
		for i := range dealPlaces.p {
			dealPlaces.p[i] = []float64{3.0 / 20, 17.0 / 20}
		}

		r := engine.NewRNG(32)
		for range obsSamples {
			d := o.Determinize(&r)
			j := 0
			for i := range d.Slots {
				if !d.Slots[i].FaceUp {
					current.add(t, uint8(d.Slots[i].Card), j)
					j++
				}
			}
			for _, c := range d.Removed[0] {
				current.add(t, uint8(c), j)
				j++
			}
			for j, c := range append(d.Deals[1][:], d.Removed[1][:]...) {
				ageII.add(t, uint8(c), j)
			}
			for j, c := range d.Deals[2] {
				if catalog.Card(c).IsGuild() {
					guilds.add(t, uint8(c), 0)
					dealPlaces.add(t, uint8(j+1), 0)
				} else {
					thirds.add(t, uint8(c), 0)
					dealPlaces.add(t, uint8(j+1), 1)
				}
			}
			for _, c := range d.BoxGuilds {
				guilds.add(t, uint8(c), 1)
			}
			for _, c := range d.Removed[2] {
				thirds.add(t, uint8(c), 1)
			}
		}
		for _, c := range []*obsChi2{current, ageII, guilds, thirds, dealPlaces} {
			c.check(t, obsSamples)
		}
	})

	t.Run("age III", func(t *testing.T) {
		s := obsAgeIIIWithHiddenGuild(t)
		o := engine.Observe(&s, s.ToMove)
		var cards, guilds []catalog.CardID
		for _, sl := range s.Slots {
			switch {
			case sl.FaceUp:
			case sl.Back == catalog.GuildBack:
				guilds = append(guilds, sl.Card)
			default:
				cards = append(cards, sl.Card)
			}
		}
		cards = append(cards, s.Removed[2][:]...)
		guilds = append(guilds, s.BoxGuilds[:]...)
		slices.Sort(cards)
		slices.Sort(guilds)
		cardTable := obsUniformChi2("Age III: face-down cards and removed", obsIDs(cards))
		guildTable := obsUniformChi2("Age III: face-down guilds and the box", obsIDs(guilds))
		r := engine.NewRNG(33)
		for range obsSamples {
			d := o.Determinize(&r)
			jc, jg := 0, 0
			for _, sl := range d.Slots {
				switch {
				case sl.FaceUp:
				case sl.Back == catalog.GuildBack:
					guildTable.add(t, uint8(sl.Card), jg)
					jg++
				default:
					cardTable.add(t, uint8(sl.Card), jc)
					jc++
				}
			}
			for _, c := range d.Removed[2] {
				cardTable.add(t, uint8(c), jc)
				jc++
			}
			for _, c := range d.BoxGuilds {
				guildTable.add(t, uint8(c), jg)
				jg++
			}
		}
		cardTable.check(t, obsSamples)
		guildTable.check(t, obsSamples)
	})

	t.Run("great library", func(t *testing.T) {
		// Seen from the opponent, the offer and the box hold the five box
		// tokens in a uniformly random order.
		s := obsLibraryOpen(t)
		o := engine.Observe(&s, 1)
		tokens := append(s.Library[:], s.BoxTokens[:2]...)
		slices.Sort(tokens)
		table := obsUniformChi2("Great Library: the opponent's offer and the box", obsIDs(tokens))
		r := engine.NewRNG(34)
		for range obsSamples {
			d := o.Determinize(&r)
			for j, tk := range append(d.Library[:], d.BoxTokens[:2]...) {
				table.add(t, uint8(tk), j)
			}
		}
		table.check(t, obsSamples)
	})
}

// TestDeterminizeLibrary checks the Great Library offer from both sides and
// after the pick.
func TestDeterminizeLibrary(t *testing.T) {
	s := obsLibraryOpen(t)
	pool := catalog.TokenSetOf(append(s.Library[:], s.BoxTokens[:2]...)...)
	r := engine.NewRNG(6)

	own := engine.Observe(&s, 0)
	other := engine.Observe(&s, 1)
	var orders [2]bool
	for range 200 {
		d := own.Determinize(&r)
		if d.Library != s.Library {
			t.Fatalf("the drawer's sample changed the offer %v to %v", s.Library, d.Library)
		}
		switch {
		case d.BoxTokens == s.BoxTokens:
			orders[0] = true
		case d.BoxTokens[0] == s.BoxTokens[1] && d.BoxTokens[1] == s.BoxTokens[0]:
			orders[1] = true
		default:
			t.Fatalf("the drawer's sample has box %v, want the tokens of %v", d.BoxTokens, s.BoxTokens)
		}

		d = other.Determinize(&r)
		got := catalog.TokenSetOf(append(d.Library[:], d.BoxTokens[:2]...)...)
		if got != pool || d.BoxTokens[2] != 0 {
			t.Fatalf("the opponent's sample has offer %v and box %v, want the tokens %v", d.Library, d.BoxTokens, pool)
		}
	}
	if !orders[0] || !orders[1] {
		t.Errorf("the box order of the drawer's samples never varied: %v", orders)
	}

	enginetest.Do(t, &s, "token:law")
	for seat := range engine.Seat(2) {
		o := engine.Observe(&s, seat)
		for range 20 {
			d := o.Determinize(&r)
			var box catalog.TokenSet
			for _, tk := range d.BoxTokens[:4] {
				box.Add(tk)
			}
			want := pool
			want.Remove(catalog.TokenLaw)
			if box != want || d.BoxTokens[4] != 0 || d.Library != ([engine.LibraryDraw]catalog.TokenID{}) {
				t.Fatalf("seat %d: after the pick the sample has box %v and offer %v", seat, d.BoxTokens, d.Library)
			}
		}
	}
}

func TestDeterminizeIsDeterministic(t *testing.T) {
	s := obsDraftDone(t, 41)
	o := engine.Observe(&s, 0)
	r := engine.NewRNG(41)
	r2 := r
	a, b := o.Determinize(&r), o.Determinize(&r2)
	if a != b {
		t.Fatal("the same observation and generator state gave different samples")
	}
	differ := false
	for range 10 {
		if c := o.Determinize(&r); c != a {
			differ = true
		}
	}
	if !differ {
		t.Error("ten further samples all equal the first")
	}
}

func TestDeterminizePanicsOnAZeroObservation(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Determinize of the zero Observation did not panic")
		}
	}()
	var o engine.Observation
	r := engine.NewRNG(1)
	o.Determinize(&r)
}

func TestObserveAndDeterminizeAllocateNothing(t *testing.T) {
	for _, s := range []engine.State{engine.NewGame(51, engine.Options{}), obsDraftDone(t, 52), enginetest.New(t, 53).Age(3).State()} {
		r := engine.NewRNG(5)
		o := engine.Observe(&s, 0)
		if n := testing.AllocsPerRun(100, func() { o = engine.Observe(&s, 0) }); n != 0 {
			t.Errorf("age %d: Observe allocates %.1f times per call", s.Age, n)
		}
		if n := testing.AllocsPerRun(100, func() { _ = o.Determinize(&r) }); n != 0 {
			t.Errorf("age %d: Determinize allocates %.1f times per call", s.Age, n)
		}
	}
}

// obsShuffleHidden returns a copy of s whose hidden parts are rearranged in
// a way a different setup shuffle could have left them, while everything
// seat can see stays as it is: another seed; the cards of each back
// shuffled among the hidden places that hold a card of that back (face-down
// slots, deals, removed cards, box guilds); the hidden wonders shuffled
// among their places; the box tokens (with the Great Library offer, when it
// is hidden from seat) shuffled among theirs. It is a second, independent
// account of what Observe must hide: Observe of the copy must equal Observe
// of s.
func obsShuffleHidden(s *engine.State, seat engine.Seat, r *engine.RNG) engine.State {
	x := *s
	x.Seed = s.Seed ^ 0x5eed
	var byBack [catalog.BackGuild + 1][]*catalog.CardID
	hide := func(c *catalog.CardID) {
		if *c != 0 {
			b := catalog.Card(*c).Back
			byBack[b] = append(byBack[b], c)
		}
	}
	for i := range x.Slots {
		if !x.Slots[i].FaceUp {
			hide(&x.Slots[i].Card)
		}
	}
	for a := range x.Deals {
		for i := range x.Deals[a] {
			hide(&x.Deals[a][i])
		}
	}
	for a := range x.Removed {
		for i := range x.Removed[a] {
			hide(&x.Removed[a][i])
		}
	}
	for i := range x.BoxGuilds {
		hide(&x.BoxGuilds[i])
	}
	for _, places := range byBack {
		obsShufflePlaces(places, r)
	}

	var wonders []*catalog.WonderID
	if bits.OnesCount8(x.DraftTaken) < 4 {
		for i := 4; i < engine.NumDraftWonders; i++ {
			wonders = append(wonders, &x.DraftOrder[i])
		}
	}
	for i := range x.BoxWonders {
		wonders = append(wonders, &x.BoxWonders[i])
	}
	obsShufflePlaces(wonders, r)

	var tokens []*catalog.TokenID
	if x.Phase != engine.PhasePickLibraryToken || x.ToMove != seat {
		for i := range x.Library {
			if x.Library[i] != 0 {
				tokens = append(tokens, &x.Library[i])
			}
		}
	}
	for i := range x.BoxTokens {
		if x.BoxTokens[i] != 0 {
			tokens = append(tokens, &x.BoxTokens[i])
		}
	}
	obsShufflePlaces(tokens, r)
	return x
}

// obsShufflePlaces shuffles the values held in places among them.
func obsShufflePlaces[T any](places []*T, r *engine.RNG) {
	vals := make([]T, len(places))
	for i, p := range places {
		vals[i] = *p
	}
	engine.ShuffleSlice(r, vals)
	for i, p := range places {
		*p = vals[i]
	}
}

// obsHiddenDiffers reports whether two states differ in more than the seed.
func obsHiddenDiffers(a, b engine.State) bool {
	a.Seed, b.Seed = 0, 0
	return a != b
}

// obsChoose is the seeded random chooser of the no-cheat test, a flat Monte
// Carlo search: for each legal action, in order, it plays obsPlayouts
// random games from states drawn by sample and counts two points for a win
// of seat me and one for a shared victory; it keeps the first action with
// the most points. It also returns a fingerprint of the outcomes of all its
// playouts, so a test can tell whether two searches saw different games.
func obsChoose(legal []engine.Action, me engine.Seat, seed uint64, sample func(*engine.RNG) engine.State) (engine.Action, uint64) {
	r := engine.NewRNG(seed)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	best, bestPoints := 0, -1
	fp := uint64(14695981039346656037)
	mix := func(v int) { fp = (fp ^ uint64(v)) * 1099511628211 }
	for i, a := range legal {
		points := 0
		for range obsPlayouts {
			d := sample(&r)
			engine.ApplyTrusted(&d, a)
			for n := 0; !d.Over() && n < obsMaxPlies; n++ {
				buf = engine.LegalActions(&d, buf[:0])
				engine.ApplyTrusted(&d, buf[r.IntN(len(buf))])
			}
			switch d.Result.Winner {
			case int8(me):
				points += 2
			case engine.SharedVictory:
				points++
			}
			mix(int(d.Ply))
			mix(int(d.Result.Kind))
			mix(int(d.Result.Scores[0].Total))
			mix(int(d.Result.Scores[1].Total))
		}
		mix(points)
		if points > bestPoints {
			best, bestPoints = i, points
		}
	}
	return legal[best], fp
}

// obsHonestChoose runs the chooser on an observation: it sees the game
// only through Public and Determinize, as a bot does.
func obsHonestChoose(o *engine.Observation, seed uint64) (engine.Action, uint64) {
	legal := engine.LegalActions(o.Public(), nil)
	return obsChoose(legal, o.Seat(), seed, func(r *engine.RNG) engine.State { return o.Determinize(r) })
}

// obsCheatChoose runs the same chooser on the real state, the control that
// shows the positions of the no-cheat test differ in hidden information
// that matters to a search.
func obsCheatChoose(s *engine.State, seed uint64) (engine.Action, uint64) {
	legal := engine.LegalActions(s, nil)
	return obsChoose(legal, s.ToMove, seed, func(*engine.RNG) engine.State { return *s })
}

// TestNoCheat plays seeded games and, at positions along them, builds a
// second state that differs only in hidden information. The two states
// must give the seat to move equal observations, and a chooser that sees
// only the observation must make the same choice from the same seed, with
// the same playout outcomes. The same chooser given the real states (the
// control) sees different games, so the hidden information it is denied
// would have changed its search.
func TestNoCheat(t *testing.T) {
	r := engine.NewRNG(61)
	pairs, cheatSaw := 0, 0
	for seed := uint64(1); seed <= 4; seed++ {
		for i, s := range obsGame(t, 300+seed, seed%2 == 0) {
			if s.Over() || i%7 != 3 {
				continue
			}
			seat := s.ToMove
			s2 := obsShuffleHidden(&s, seat, &r)
			if err := engine.CheckInvariants(&s2); err != nil {
				t.Fatalf("game %d ply %d: the reshuffled state is not valid: %v", seed, s.Ply, err)
			}
			if !obsHiddenDiffers(s, s2) {
				continue // nothing hidden to rearrange
			}
			o1, o2 := engine.Observe(&s, seat), engine.Observe(&s2, seat)
			if o1 != o2 {
				t.Fatalf("game %d ply %d: states that differ only in hidden parts give different observations:%s",
					seed, s.Ply, obsDiff(o2.Public(), o1.Public()))
			}
			pairs++
			choiceSeed := seed<<16 | uint64(s.Ply)
			a1, fp1 := obsHonestChoose(&o1, choiceSeed)
			a2, fp2 := obsHonestChoose(&o2, choiceSeed)
			if a1 != a2 || fp1 != fp2 {
				t.Fatalf("game %d ply %d: equal observations gave %s and %s", seed, s.Ply, a1.Key(), a2.Key())
			}
			if engine.IsLegal(&s, a1) != nil || engine.IsLegal(&s2, a1) != nil {
				t.Fatalf("game %d ply %d: the choice %s is not legal in both states", seed, s.Ply, a1.Key())
			}
			_, c1 := obsCheatChoose(&s, choiceSeed)
			_, c2 := obsCheatChoose(&s2, choiceSeed)
			if c1 != c2 {
				cheatSaw++
			}
		}
	}
	if pairs < 20 {
		t.Fatalf("only %d position pairs were checked", pairs)
	}
	if cheatSaw*2 < pairs {
		t.Errorf("the control saw different games in only %d of %d pairs; the pairs barely differ", cheatSaw, pairs)
	}
	t.Logf("%d pairs; the control saw different games in %d", pairs, cheatSaw)
}

var obsSink uint16

func BenchmarkDeterminize(b *testing.B) {
	mid := obsGame(b, 71, true)
	var age1, age3 engine.State
	for _, s := range mid {
		if s.Age == 1 && s.Phase == engine.PhasePlay && bits.OnesCount32(engine.Occupied(&s)) == 10 {
			age1 = s
		}
		if s.Age == 3 && s.Phase == engine.PhasePlay && age3.Age == 0 {
			age3 = s
		}
	}
	cases := []struct {
		name string
		s    engine.State
	}{
		{"draft", engine.NewGame(71, engine.Options{})},
		{"age1", age1},
		{"age3", age3},
	}
	for _, c := range cases {
		if c.s.Ply == 0 && c.name != "draft" {
			b.Fatalf("game 71 has no %s position", c.name)
		}
		b.Run(c.name, func(b *testing.B) {
			o := engine.Observe(&c.s, c.s.ToMove)
			r := engine.NewRNG(71)
			b.ReportAllocs()
			for b.Loop() {
				d := o.Determinize(&r)
				obsSink = d.Ply
			}
		})
	}
}

func BenchmarkObserve(b *testing.B) {
	s := obsGame(b, 72, true)[30]
	b.ReportAllocs()
	for b.Loop() {
		o := engine.Observe(&s, 0)
		obsSink = o.Public().Ply
	}
}
