package rules

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// The entitlement oracle collects facts when they become visible, independently
// of Observe, History and FilterEvents. It never uses a global forbidden-card
// set: the same design may be public in a city and private in another hand.
type screenMemory struct {
	seen      [MaxSeats][]view.SeenEntry
	looks     [MaxSeats][]view.PileLook
	last      [MaxSeats]*view.Play
	leftovers []view.Leftover
	events    []Event
}

func newScreenMemory() screenMemory {
	m := screenMemory{leftovers: []view.Leftover{}}
	for seat := range m.seen {
		m.seen[seat] = []view.SeenEntry{}
		m.looks[seat] = []view.PileLook{}
	}
	return m
}
func (m *screenMemory) record(before, next *Game, b Batch, events []Event) {
	p := before.Point()
	for _, pick := range b.Picks {
		a, _ := ParseKey(pick.Key)
		if p.Kind == PointTurn || p.Kind == PointLastCard {
			hand := before.State.Hands[pick.Seat]
			if p.Kind == PointLastCard {
				hand = [HandSize]catalog.DesignID{before.State.Cities[pick.Seat].Kept}
			}
			m.seen[pick.Seat] = append(m.seen[pick.Seat], view.SeenEntry{Point: p.ID(), Hand: oracleSlugs(hand[:]), Played: oracleSlug(a.Design), Action: actionVerbs[a.Kind]})
		}
		if a.Kind == ActStage && stageHasEffect(&next.State.Cities[pick.Seat], next.State.Cities[pick.Seat].Stages, catalog.EffBuildFromDiscard) {
			look := view.PileLook{Point: (Point{Kind: PointDiscardBuild, Age: p.Age, Turn: p.Turn}).ID(), Cards: []view.DesignCount{}}
			for id, n := range next.State.Discard {
				if id > 0 && n > 0 {
					look.Cards = append(look.Cards, view.DesignCount{Design: oracleSlug(catalog.DesignID(id)), Count: int(n)})
				}
			}
			m.looks[pick.Seat] = append(m.looks[pick.Seat], look)
		}
	}
	for _, e := range events {
		switch e.Kind {
		case EvPlay:
			v := &view.Play{Kind: actionVerbs[e.Action]}
			if e.Action == ActBuild {
				v.Design = oracleSlug(e.Design)
			}
			if e.Action != ActSell {
				v.Left, v.Right, v.Printed = int(e.Left), int(e.Right), int(e.Printed)
			}
			if e.Action == ActStage {
				v.Stage = int(e.Stage)
			}
			m.last[e.Seat] = v
		case EvDiscardBuild:
			m.last[e.Seat] = &view.Play{Kind: "build-discarded", Design: oracleSlug(e.Design)}
		case EvLeftoverDiscarded, EvCardKept:
			m.leftovers = append(m.leftovers, view.Leftover{Age: int(before.State.Age), Seat: int(e.Seat), Design: oracleSlug(e.Design), Kept: e.Kind == EvCardKept})
		}
	}
	m.events = append([]Event{}, events...)
}
func oracleSlug(id catalog.DesignID) string {
	d, ok := catalog.DesignByID(id)
	if !ok {
		return ""
	}
	return d.Slug
}
func oracleSlugs(ids []catalog.DesignID) []string {
	out := []string{}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for _, held := range ids {
			if held == id {
				out = append(out, oracleSlug(id))
			}
		}
	}
	return out
}
func screenProblems(v view.Table, g *Game, viewer Seat, m *screenMemory) []string {
	problems := []string{}
	bad := func(s string) { problems = append(problems, s) }
	seated := viewer.Valid(g.State.Seats)
	if (v.You != nil) != seated {
		bad("private section entitlement")
	}
	if len(v.Seats) != int(g.State.Seats) {
		bad("public city count")
		return problems
	}
	for s, c := range v.Seats {
		expected := []string{}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if g.State.Cities[s].Buildings.Has(id) {
				expected = append(expected, oracleSlug(id))
			}
		}
		if !reflect.DeepEqual(c.Built, expected) {
			bad("public building whitelist")
		}
		if !reflect.DeepEqual(c.Last, m.last[s]) {
			bad("latest public play whitelist")
		}
	}
	if v.You != nil && seated {
		p := v.You
		if !reflect.DeepEqual(p.Hand, oracleSlugs(g.State.Hands[viewer][:])) || !reflect.DeepEqual(p.Tucked, oracleSlugs(g.State.Cities[viewer].Tucked[:])) || p.Kept != oracleSlug(g.State.Cities[viewer].Kept) {
			bad("own current private holders")
		}
		if !reflect.DeepEqual(p.Notebook.Seen, m.seen[viewer]) {
			bad("own remembered hands and actions")
		}
		if !reflect.DeepEqual(p.Notebook.PileLooks, m.looks[viewer]) {
			bad("own private pile opportunities")
		}
		expectedOffer := []view.OfferEntry{}
		if g.State.Phase == PhaseDiscardBuild && g.State.Decider == viewer {
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				if n := g.State.Discard[id]; n > 0 {
					d, _ := catalog.DesignByID(id)
					expectedOffer = append(expectedOffer, view.OfferEntry{Design: d.Slug, Count: int(n), Legal: g.State.Cities[viewer].Buildings.And(catalog.T.ByName[d.NameID]).Empty()})
				}
			}
		}
		if !reflect.DeepEqual(p.Offer, expectedOffer) {
			bad("private offer entitlement")
		}
	}
	reveal := g.State.Phase == PhaseOver && !g.State.Opts.HideAtEnd
	if (v.Reveal != nil) != reveal || v.CanDownload != reveal {
		bad("end reveal/download entitlement")
	}
	if v.Reveal != nil && reveal {
		if !reflect.DeepEqual(v.Reveal.Leftovers, m.leftovers) || !reflect.DeepEqual(v.Reveal.Box, oracleSlugs(g.State.BoxGuilds[:])) {
			bad("end leftover and box whitelist")
		}
		pile := []string{}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			for n := uint8(0); n < g.State.Discard[id]; n++ {
				pile = append(pile, oracleSlug(id))
			}
		}
		if !reflect.DeepEqual(v.Reveal.Pile, pile) || len(v.Reveal.Tucked) != int(g.State.Seats) {
			bad("end pile/tucked count")
		}
		if len(v.Reveal.Tucked) == int(g.State.Seats) {
			for s := range v.Reveal.Tucked {
				if !reflect.DeepEqual(v.Reveal.Tucked[s], oracleSlugs(g.State.Cities[s].Tucked[:])) {
					bad("end tucked identities")
				}
			}
		}
	}
	if (v.Result != nil) != (g.State.Phase == PhaseOver) {
		bad("result entitlement")
	}
	if len(m.events) > 0 {
		if len(v.Events) != len(m.events) {
			bad("filtered event count")
		} else {
			for i, e := range m.events {
				expected := ""
				switch e.Kind {
				case EvDiscardBuild:
					expected = oracleSlug(e.Design)
				case EvPlay:
					if e.Action == ActBuild || e.Seat == viewer && seated {
						expected = oracleSlug(e.Design)
					}
				case EvLeftoverDiscarded, EvCardKept:
					if e.Seat == viewer && seated {
						expected = oracleSlug(e.Design)
					}
				case EvCoins:
					if e.Source == CoinDesign {
						expected = oracleSlug(e.Design)
					}
				}
				if v.Events[i].Design != expected {
					bad("event design entitlement")
				}
			}
		}
	}
	return problems
}

func TestViewPrivacyOracleNegatives(t *testing.T) {
	g, err := NewGame(Setup{Seats: 3, Seed: 71, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	m := newScreenMemory()
	if problems := screenProblems(g.ViewFor(0), &g, 0, &m); len(problems) > 0 {
		t.Fatal(problems)
	}
	for _, perturb := range []func(*view.Table){
		func(v *view.Table) { v.Seats[1].Built = append(v.Seats[1].Built, oracleSlug(g.State.Hands[1][0])) },
		func(v *view.Table) { v.You.Hand = []string{oracleSlug(g.State.Hands[1][0])} },
		func(v *view.Table) {
			v.You.Notebook.Seen = append(v.You.Notebook.Seen, view.SeenEntry{Hand: []string{"hidden-sentinel"}})
		},
		func(v *view.Table) { v.You.Offer = []view.OfferEntry{{Design: "hidden-sentinel", Count: 1}} },
		func(v *view.Table) { v.Reveal = &view.Reveal{} },
		func(v *view.Table) { v.CanDownload = true },
		func(v *view.Table) { v.You = nil },
	} {
		v := g.ViewFor(0)
		perturb(&v)
		if len(screenProblems(v, &g, 0, &m)) == 0 {
			t.Fatal("entitlement oracle accepted a hidden-field leak")
		}
	}
	v := g.ViewFor(NoSeat)
	v.You = &view.Private{}
	if len(screenProblems(v, &g, NoSeat, &m)) == 0 {
		t.Fatal("spectator leak accepted")
	}
}

func TestViewPrivacyAndQuoteSweep(t *testing.T) {
	positions, frames, quotes, maxBytes := 0, 0, 0, 0
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 6; seed++ {
			for _, hide := range []bool{false, true} {
				g, err := NewGame(Setup{Seats: n, Seed: seed, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: hide}})
				if err != nil {
					t.Fatal(err)
				}
				m := newScreenMemory()
				for {
					positions++
					observation := g.Observe(0)
					for i := 0; i <= int(n); i++ {
						viewer := Seat(i)
						if i == int(n) {
							viewer = NoSeat
						}
						v := g.ViewFor(viewer)
						assertViewSlices(t, v)
						if problems := screenProblems(v, &g, viewer, &m); len(problems) > 0 {
							t.Fatal(n, seed, hide, g.Point().ID(), viewer, problems)
						}
						b, err := json.Marshal(v)
						if err != nil || bytes.Contains(b, []byte("null")) {
							t.Fatal("invalid raw DTO", err)
						}
						maxBytes = max(maxBytes, len(b))
						frames++
					}
					var costs [catalog.NumDesigns + catalog.NumBoards*2*MaxStages]catalog.Cost
					used := 0
					for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
						d, _ := catalog.DesignByID(id)
						costs[used] = d.Cost
						used++
					}
					// All board-side stage costs, including stages not present at this table.
					for id := catalog.BoardID(1); id <= catalog.NumBoards; id++ {
						b, _ := catalog.BoardByID(id)
						for _, side := range b.Sides {
							for _, stage := range side.Stages[:side.NumStages] {
								costs[used] = stage.Cost
								used++
							}
						}
					}
					for seat := Seat(0); seat < Seat(n); seat++ {
						for _, cost := range costs[:used] {
							full := PaymentsState(&g.State, seat, cost)
							want := Quote{}
							if full.Count > 0 {
								p := full.Outcomes[0]
								want = Quote{Affordable: true, Total: uint16(full.Printed) + uint16(p.Left) + uint16(p.Right)}
							}
							if got := QuoteState(&g.State, seat, cost); got != want {
								t.Fatal("quote/full solver mismatch", n, seed, g.Point().ID(), seat, cost, got, want)
							}
							if got := observation.Quote(seat, cost); got != want {
								t.Fatal("observation/full solver mismatch", n, seed, g.Point().ID(), seat, cost, got, want)
							}
							quotes++
						}
					}
					for seat := Seat(0); seat < Seat(n); seat++ {
						checkNotebookBounds(t, &g, seat)
					}
					if g.State.Phase == PhaseOver {
						break
					}
					b := observationBatch(g, seed)
					next, events, err := g.Resolve(b)
					if err != nil {
						t.Fatal(err)
					}
					m.record(&g, &next, b, events)
					g = next
				}
			}
		}
	}
	t.Logf("%d positions, %d per-viewer frames, %d seat/cost quote/full-solver comparisons, largest reachable view %d bytes", positions, frames, quotes, maxBytes)
}

// Track packet origins by moving labels along the ring, independently of the
// notebook's subtraction algorithm. Every certain/upper bound contains the
// actual multiset, and every named neighbour pick equals the resolved record.
func checkNotebookBounds(t *testing.T, g *Game, seat Seat) {
	t.Helper()
	o := g.Observe(seat)
	s := o.Statistics()
	if g.State.Age == 0 {
		return
	}
	for _, p := range s.Deductions[:s.NumDeductions] {
		if g.Record.Picks[p.Age-1][p.Turn-1][p.Seat] != p.Design {
			t.Fatal("false certain neighbour deduction", p)
		}
	}
	var origins [MaxSeats]int
	for i := range origins {
		origins[i] = i
	}
	var seen [MaxSeats]bool
	turn := int(g.State.Turn)
	completed := turn - 1
	if g.State.Phase == PhaseDiscardBuild {
		completed = turn
	}
	if g.State.Phase == PhaseOver || g.State.Phase == PhaseLastCard {
		completed = 6
	}
	for played := 1; played <= min(completed, 5); played++ {
		if o.Know.Picks[g.State.Age-1][played-1] != 0 {
			seen[origins[seat]] = true
		}
		origins = oraclePassOrigins(origins, g.State.Seats, g.State.Age)
	}
	passes := completed
	if g.State.Phase == PhaseDiscardBuild {
		passes--
	}
	origins = [MaxSeats]int{}
	for i := range origins {
		origins[i] = i
	}
	for i := 0; i < passes; i++ {
		origins = oraclePassOrigins(origins, g.State.Seats, g.State.Age)
	}
	var colours [catalog.NumColors]uint8
	for holder := 0; holder < int(g.State.Seats); holder++ {
		if seen[origins[holder]] {
			for _, id := range g.State.Hands[holder] {
				if id != 0 {
					d, _ := catalog.DesignByID(id)
					colours[d.Color]++
				}
			}
		}
	}
	for colour, b := range s.Circulating {
		if b.Certain > colours[colour] || b.AtMost < colours[colour] {
			t.Fatal("notebook bounds exclude actual consistent world", g.Point().ID(), seat, colour, b, colours)
		}
	}
}
func oraclePassOrigins(in [MaxSeats]int, n, age uint8) [MaxSeats]int {
	var out [MaxSeats]int
	for from := 0; from < int(n); from++ {
		to := (from + 1) % int(n)
		if age == 2 {
			to = (from + int(n) - 1) % int(n)
		}
		out[to] = in[from]
	}
	return out
}

// This test policy is a deterministic Greedy witness, not the future shipping
// bot: it accepts only the permitted observation and the offered choices. It
// exercises public hypothetical quotes and remembered information in its rank.
func privacyGreedy(o Observation, choices []Choice) string {
	best, key := -1<<30, ""
	stats := o.Statistics()
	for _, c := range choices {
		value := -int(c.Total) * 8
		switch c.Kind {
		case ActBuild, ActBuildDiscarded:
			d, _ := catalog.DesignByID(c.Design)
			value += int(d.Color)*7 + int(stats.Returning[c.Design].Certain)
			q := o.Quote(o.Now.Seat, d.Cost)
			if q.Affordable {
				value -= int(q.Total)
			}
		case ActStage:
			value += 12
		case ActSell:
			value += 3
		case ActSide:
			value += int(c.Side)
		}
		if value > best {
			best, key = value, c.Key()
		}
	}
	return key
}
func assertPrivacyPair(t *testing.T, a, b Game, label string) {
	t.Helper()
	if err := a.CheckInvariants(); err != nil {
		t.Fatal(label, "first unreachable", err)
	}
	if err := b.CheckInvariants(); err != nil {
		t.Fatal(label, "second unreachable", err)
	}
	if a == b {
		t.Fatal(label, "pair does not differ")
	}
	if a.Observe(NoSeat) != b.Observe(NoSeat) {
		t.Fatal(label, "hidden difference reached spectator observation")
	}
	sa, _ := json.Marshal(a.ViewFor(NoSeat))
	sb, _ := json.Marshal(b.ViewFor(NoSeat))
	if !bytes.Equal(sa, sb) {
		t.Fatal(label, "hidden difference reached spectator view")
	}
	x, y := a.Observe(0), b.Observe(0)
	if x.Now != y.Now || x.Know != y.Know || x != y || x.Statistics() != y.Statistics() || !bytes.Equal(x.AppendEncoding(nil), y.AppendEncoding(nil)) || x.Fingerprint() != y.Fingerprint() {
		t.Fatal(label, "unknown cards changed normalized observation")
	}
	va, vb := a.ViewFor(0), b.ViewFor(0)
	ja, _ := json.Marshal(va)
	jb, _ := json.Marshal(vb)
	if !bytes.Equal(ja, jb) {
		t.Fatal(label, "unknown cards changed view", string(ja), string(jb))
	}
	if !reflect.DeepEqual(a.Legal(0), b.Legal(0)) || privacyGreedy(x, a.Legal(0)) != privacyGreedy(y, b.Legal(0)) {
		t.Fatal(label, "unknown cards changed Greedy input/choice")
	}
}

func TestReachableHiddenPlacePairs(t *testing.T) {
	a, err := NewGame(Setup{Seats: 7, Seed: 41, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("other-hands", func(t *testing.T) {
		sc := ScenarioFromGame(a)
		pos := 0
		for i, id := range sc.State.Hands[3] {
			if id != sc.State.Hands[2][0] {
				pos = i
				break
			}
		}
		sc.State.Hands[2][0], sc.State.Hands[3][pos] = sc.State.Hands[3][pos], sc.State.Hands[2][0]
		sc.Record.Packets[0] = sc.State.Hands
		b, err := NewGameFromScenario(92, a.State.Opts, sc)
		if err != nil {
			t.Fatal(err)
		}
		assertPrivacyPair(t, a, b, "other hand and setup root")
	})
	t.Run("future-deal", func(t *testing.T) {
		sc := ScenarioFromGame(a)
		pos := 0
		for i, id := range sc.State.Deals[1][2] {
			if id != sc.State.Deals[1][1][0] {
				pos = i
				break
			}
		}
		sc.State.Deals[1][1][0], sc.State.Deals[1][2][pos] = sc.State.Deals[1][2][pos], sc.State.Deals[1][1][0]
		b, err := NewGameFromScenario(92, a.State.Opts, sc)
		if err != nil {
			t.Fatal(err)
		}
		assertPrivacyPair(t, a, b, "future deal")
	})
	t.Run("box-guild", func(t *testing.T) {
		sc := ScenarioFromGame(a)
		swapped := false
		for seat := 0; seat < 7 && !swapped; seat++ {
			for card, id := range sc.State.Deals[2][seat] {
				if id >= 71 {
					sc.State.BoxGuilds[0], sc.State.Deals[2][seat][card] = id, sc.State.BoxGuilds[0]
					swapped = true
					break
				}
			}
		}
		if !swapped {
			t.Fatal("no dealt guild")
		}
		b, err := NewGameFromScenario(92, a.State.Opts, sc)
		if err != nil {
			t.Fatal(err)
		}
		assertPrivacyPair(t, a, b, "box guild")
	})
}
func commonSale(t *testing.T, a, b *Game, seat Seat, avoid ...catalog.DesignID) Choice {
	t.Helper()
	for _, ca := range a.Legal(seat) {
		if ca.Kind != ActSell {
			continue
		}
		skip := false
		for _, id := range avoid {
			if ca.Design == id {
				skip = true
			}
		}
		if skip {
			continue
		}
		for _, cb := range b.Legal(seat) {
			if cb.Action == ca.Action {
				return ca
			}
		}
	}
	t.Fatal("no common sale", seat)
	return Choice{}
}
func TestReachableHiddenResolvedChoicePairs(t *testing.T) {
	for _, kind := range []ActionKind{ActSell, ActStage} {
		t.Run(actionVerbs[kind], func(t *testing.T) {
			var first Game
			found := false
			for seed := uint64(0); seed < 1000 && !found; seed++ {
				g, err := NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesChoose}})
				if err != nil {
					t.Fatal(err)
				}
				for _, id := range g.State.Deals[0][2] {
					if id == 2 {
						first = g
						found = true
					}
				}
			}
			if !found {
				t.Fatal("no initial clay pool for Halikarnassos")
			}
			sc := ScenarioFromGame(first)
			for s := 0; s < 7; s++ {
				if sc.State.Cities[s].Board == catalog.BoardHalikarnassos {
					sc.State.Cities[2].Board, sc.State.Cities[s].Board = sc.State.Cities[s].Board, sc.State.Cities[2].Board
				}
			}
			for s := 0; s < 7; s++ {
				if sc.State.Cities[s].Board == catalog.BoardOlympia {
					sc.State.Cities[3].Board, sc.State.Cities[s].Board = sc.State.Cities[s].Board, sc.State.Cities[3].Board
				}
			}
			a, err := NewGameFromScenario(first.State.Seed, first.State.Opts, sc)
			if err != nil {
				t.Fatal(err)
			}
			sc = ScenarioFromGame(a)
			old := sc.State.Deals[0][1][0]
			pos := -1
			for i, id := range sc.State.Deals[0][3] {
				if id != old {
					pos = i
					break
				}
			}
			if pos < 0 {
				t.Fatal("no different hidden copy")
			}
			replacement := sc.State.Deals[0][3][pos]
			sc.State.Deals[0][1][0], sc.State.Deals[0][3][pos] = replacement, old
			b, err := NewGameFromScenario(first.State.Seed+1000, first.State.Opts, sc)
			if err != nil {
				t.Fatal(err)
			}
			initialA, initialB := a, b
			batchesA, batchesB := []Batch{}, []Batch{}
			side := Batch{Point: "sides"}
			for seat := Seat(0); seat < 7; seat++ {
				s := catalog.Day
				if seat == 2 {
					s = catalog.Night
				}
				side.Picks = append(side.Picks, Pick{Seat: seat, Key: (Action{Kind: ActSide, Side: s}).Key()})
			}
			for step := 0; step < 3; step++ {
				ba, bb := side, side
				if step > 0 {
					ba = Batch{Point: a.Point().ID()}
					bb = Batch{Point: b.Point().ID()}
					for seat := Seat(0); seat < 7; seat++ {
						ca, cb := Choice{}, Choice{}
						switch {
						case seat == 2 && step == 1:
							for _, c := range a.Legal(seat) {
								if c.Kind == ActBuild && c.Design == 2 {
									ca = c
									break
								}
							}
							cb = ca
						case seat == 2 && step == 2:
							for _, c := range a.Legal(seat) {
								if c.Kind == kind && c.Design == old {
									ca = c
									break
								}
							}
							for _, c := range b.Legal(seat) {
								if c.Kind == kind && c.Design == replacement && c.Left == ca.Left && c.Right == ca.Right {
									cb = c
									break
								}
							}
						default:
							ca = commonSale(t, &a, &b, seat, old, replacement)
							cb = ca
						}
						if ca.Kind == ActNone || cb.Kind == ActNone {
							t.Fatal("fixture lacks paired legal action", step, seat, kind, old, replacement)
						}
						ba.Picks = append(ba.Picks, Pick{Seat: seat, Key: ca.Key()})
						bb.Picks = append(bb.Picks, Pick{Seat: seat, Key: cb.Key()})
					}
				}
				a, _, err = a.Resolve(ba)
				if err != nil {
					t.Fatal(err)
				}
				b, _, err = b.Resolve(bb)
				if err != nil {
					t.Fatal(err)
				}
				batchesA = append(batchesA, ba)
				batchesB = append(batchesB, bb)
				assertPrivacyPair(t, a, b, "reachable "+actionVerbs[kind])
			}
			if kind == ActStage && a.State.Cities[2].Tucked == b.State.Cities[2].Tucked {
				t.Fatal("tucked identities did not differ")
			}
			if kind == ActSell && a.State.Discard == b.State.Discard {
				t.Fatal("sale identities did not differ")
			}
			for i, initial := range []Game{initialA, initialB} {
				batches, final := batchesA, a
				if i == 1 {
					batches, final = batchesB, b
				}
				setup := Setup{Seed: initial.State.Seed, Seats: 7, Options: initial.State.Opts, Scenario: ScenarioFromGame(initial)}
				replay, err := Replay(setup, batches)
				if err != nil || replay != final {
					t.Fatal("restart differs", err)
				}
				if replay.Observe(0) != final.Observe(0) || replay.Observe(0).Fingerprint() != final.Observe(0).Fingerprint() {
					t.Fatal("restart observation differs")
				}
				undo, err := Replay(setup, batches[:len(batches)-1])
				if err != nil {
					t.Fatal(err)
				}
				again, _, err := undo.Resolve(batches[len(batches)-1])
				if err != nil || again.Observe(0) != final.Observe(0) {
					t.Fatal("undo then redo differs", err)
				}
			}
		})
	}
}
