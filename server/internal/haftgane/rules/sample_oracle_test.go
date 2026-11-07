package rules

import (
	"math"
	"os"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// A complete conserving setup and legal prefix, rather than a partial record:
// two nonempty duplicate-only skips, then a new pile addition and a rebuild.
// This pins rulings 3 and 23 without sharing the sampler's skip constraints.
func samplerSkipRun(t *testing.T) privacySweepRun {
	t.Helper()
	g, err := NewGame(Setup{Seats: 7, Seed: 27, Options: Options{Sides: SidesChoose}})
	if err != nil {
		t.Fatal(err)
	}
	boards := [MaxSeats]catalog.BoardID{catalog.BoardHalikarnassos, catalog.BoardBabylon, catalog.BoardAlexandria, catalog.BoardEphesos, catalog.BoardGizah, catalog.BoardRhodos, catalog.BoardOlympia}
	for s, b := range boards {
		g.State.Cities[s], err = cityCaches(City{Board: b, Coins: 3})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Each entry names a removal, not a physical copy or an original hand.
	planned := [5][MaxSeats]catalog.DesignID{
		{2, 11, 12, 17, 19, 20, 13},
		{14, 25, 21, 2, 24, 22, 25},
		{14, 5, 7, 7, 8, 8, 5},
		{23, 16, 17, 19, 20, 23, 22},
		{15},
	}
	var supply [catalog.NumDesigns + 1]int
	for _, h := range g.State.Deals[0] {
		for _, id := range h {
			if id != 0 {
				supply[id]++
			}
		}
	}
	var packets [MaxSeats][HandSize]catalog.DesignID
	for ti, row := range planned {
		for s, id := range row {
			if id == 0 {
				continue
			}
			origin := (s - ti + MaxSeats) % MaxSeats
			packets[origin][ti] = id
			supply[id]--
			if supply[id] < 0 {
				t.Fatal("fixture overdraw", id)
			}
		}
	}
	for s := range packets {
		for i, id := range packets[s] {
			if id == 0 {
				for d := 1; d <= catalog.NumDesigns; d++ {
					if supply[d] > 0 {
						packets[s][i] = catalog.DesignID(d)
						supply[d]--
						break
					}
				}
			}
		}
		canonicalIDs(packets[s][:])
	}
	g.State.Deals[0] = packets
	g, err = NewGameFromScenario(0, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal(err)
	}
	run := privacySweepRun{games: []Game{g}}
	step := func(b Batch) []Event {
		next, evs, e := g.Resolve(b)
		if e != nil {
			t.Fatal(e)
		}
		g = next
		run.batches = append(run.batches, b)
		run.games = append(run.games, g)
		return evs
	}
	sides := Batch{Point: g.Point().ID()}
	for s := Seat(0); s < 7; s++ {
		side := catalog.Day
		if s == 0 {
			side = catalog.Night
		}
		sides.Picks = append(sides.Picks, Pick{Seat: s, Key: (Action{Kind: ActSide, Side: side}).Key()})
	}
	step(sides)
	skips := 0
	for ti, row := range planned {
		b := Batch{Point: g.Point().ID()}
		for s := Seat(0); s < 7; s++ {
			kind := ActBuild
			if s == 0 && (ti == 1 || ti == 2 || ti == 4) {
				kind = ActStage
			}
			if ti == 1 && s == 3 || ti == 3 && s == 2 || ti == 4 && s != 0 {
				kind = ActSell
			}
			id := row[s]
			if id == 0 {
				id = g.State.Hands[s][0]
			}
			found := false
			for _, ch := range g.Legal(s) {
				if ch.Kind != kind || ch.Design != id {
					continue
				}
				if ti == 1 && s == 1 && ch.Right != 2 {
					continue
				}
				if ti == 1 && s == 6 && ch.Left != 2 {
					continue
				}
				b.Picks = append(b.Picks, Pick{Seat: s, Key: ch.Key()})
				found = true
				break
			}
			if !found {
				t.Fatalf("fixture no action turn%d seat%d kind%d id%d", ti+1, s, kind, id)
			}
		}
		for _, e := range step(b) {
			if e.Kind == EvDiscardBuildNothing {
				skips++
			}
		}
		if ti == 1 || ti == 2 {
			if g.State.Discard[2] != 1 || g.State.Phase != PhaseTurn {
				t.Fatal("expected duplicate-only skipped pile")
			}
		}
	}
	if skips != 2 || g.State.Phase != PhaseDiscardBuild {
		t.Fatal("skip/rebuild fixture coverage", skips, g.State.Phase)
	}
	b := Batch{Point: g.Point().ID(), Picks: []Pick{{Seat: 0, Key: (Action{Kind: ActBuildDiscarded, Design: 17}).Key()}}}
	step(b)
	return run
}

func samplerWitness(t *testing.T, g Game) World {
	t.Helper()
	w := World{state: g.State}
	w.state.Seed = 0
	for a := 0; a < NumAges; a++ {
		packets := g.Record.Packets[a]
		if packets == ([MaxSeats][HandSize]catalog.DesignID{}) {
			packets = g.State.Deals[a]
		}
		for p := 0; p < int(g.State.Seats); p++ {
			h := packets[p]
			for ti := 0; ti < TurnsPerAge; ti++ {
				dir := 1
				if a == 1 {
					dir = -1
				}
				holder := ((p+dir*ti)%int(g.State.Seats) + int(g.State.Seats)) % int(g.State.Seats)
				id := g.Record.Picks[a][ti][holder]
				if id != 0 {
					w.cards[a][p][ti] = id
					if !takeCard(&h, id) {
						t.Fatal("bad witness fixture")
					}
				}
			}
			at := 0
			for ti := range w.cards[a][p] {
				if w.cards[a][p][ti] == 0 {
					w.cards[a][p][ti] = h[at]
					at++
				}
			}
		}
	}
	return w
}

// This oracle uses only full deals, original public keys and modular packet
// rotation, and asks checked Resolve whether the whole candidate past exists.
// It shares neither group supplies, pile intervals nor skip/look predicates.
func samplerReplayOracle(root Game, batches []Batch, cards [NumAges][MaxSeats][HandSize]catalog.DesignID, box [MaxBoxGuilds]catalog.DesignID, o Observation) (World, bool) {
	g := root
	g.State.Seed = 0
	g.State.BoxGuilds = box
	for a := range cards {
		packets := cards[a]
		for s := range packets {
			canonicalIDs(packets[s][:])
		}
		if a == 0 && g.State.Phase == PhaseTurn {
			g.State.Hands = packets
			g.Record.Packets[0] = packets
		} else {
			g.State.Deals[a] = packets
		}
	}
	var err error
	g, err = NewGameFromScenario(0, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		return World{}, false
	}
	for _, old := range batches {
		b := Batch{Point: g.Point().ID()}
		for _, pick := range old.Picks {
			a, e := ParseKey(pick.Key)
			if e != nil {
				return World{}, false
			}
			if a.Kind == ActBuild || a.Kind == ActStage || a.Kind == ActSell {
				ti := int(g.State.Turn) - 1
				rot := min(ti, 5)
				dir := 1
				if g.State.Age == 2 {
					dir = -1
				}
				packet := ((int(pick.Seat)-dir*rot)%int(g.State.Seats) + int(g.State.Seats)) % int(g.State.Seats)
				a.Design = cards[g.State.Age-1][packet][ti]
			}
			b.Picks = append(b.Picks, Pick{Seat: pick.Seat, Key: a.Key()})
		}
		next, _, e := g.Resolve(b)
		if e != nil {
			return World{}, false
		}
		g = next
	}
	return World{state: g.State, cards: cards}, g.Observe(o.Now.Seat) == o
}

func TestSamplerEnumerableReplay(t *testing.T) {
	run := samplerSkipRun(t)
	trials := 256
	if os.Getenv("HAFTGANE_SAMPLER_GAMES") != "" {
		trials = 2048
	}
	// Inspect both sides of each skip, the later addition, the pending rebuild
	// and the completed rebuild, from an owner and a nonowner's knowledge.
	for prefix := 3; prefix < len(run.games); prefix++ {
		g := run.games[prefix]
		for _, seat := range []Seat{0, 4} {
			o := g.Observe(seat)
			actual := samplerWitness(t, g)
			if !MatchesObservation(o, actual) {
				t.Fatal("actual witness fails", prefix, seat)
			}
			var initial sampleModel
			initial.limit = MaxSamplerChecks
			if !initial.prepare(&o) {
				t.Fatal("prepare", prefix, seat)
			}
			// Freeze the complement of a six-place universe. This conditions
			// both solvers on identical explicit assignments, not on a shared
			// interpretation of the observation's constraints.
			var variable []int
			for i, p := range initial.places[:initial.np] {
				if p.age != 1 || p.fixed != 0 {
					continue
				}
				if p.birth != 0 || p.turn <= g.State.Turn {
					variable = append(variable, i)
				}
			}
			if len(variable) > 6 {
				variable = variable[:6]
			}
			if len(variable) < 2 {
				t.Fatal("too few enumerable places")
			}
			for i := range initial.places[:initial.np] {
				p := &initial.places[i]
				chosen := false
				for _, v := range variable {
					chosen = chosen || v == i
				}
				if chosen {
					continue
				}
				if p.turn == 0 {
					continue
				}
				id := actual.cards[p.age-1][p.packet][p.turn-1]
				p.base = catalog.DesignSet{}
				p.base.Add(id)
			}
			var values []catalog.DesignID
			for _, i := range variable {
				p := initial.places[i]
				values = append(values, actual.cards[p.age-1][p.packet][p.turn-1])
			}
			accepted := map[string]bool{}
			var enumerate func(int)
			enumerate = func(at int) {
				if at == len(values) {
					cards := actual.cards
					for j, i := range variable {
						p := initial.places[i]
						cards[p.age-1][p.packet][p.turn-1] = values[j]
					}
					w, replayOK := samplerReplayOracle(run.games[0], run.batches[:prefix], cards, actual.state.BoxGuilds, o)
					m := initial
					for j, i := range variable {
						m.places[i].base = catalog.DesignSet{}
						m.places[i].base.Add(values[j])
					}
					r := rng.NewRNG(91)
					solverOK := m.sources(0, &r)
					if solverOK {
						_, solverOK = m.world(&o)
					}
					if replayOK != solverOK {
						t.Fatalf("support differs prefix%d seat%d assignment%v replay%v solver%v", prefix, seat, values, replayOK, solverOK)
					}
					if replayOK {
						accepted[string(values)] = true
						checkSampleNotebook(t, o, w)
					}
					return
				}
				var tried catalog.DesignSet
				for j := at; j < len(values); j++ {
					if tried.Has(values[j]) {
						continue
					}
					tried.Add(values[j])
					values[at], values[j] = values[j], values[at]
					enumerate(at + 1)
					values[at], values[j] = values[j], values[at]
				}
			}
			enumerate(0)
			seen := map[string]int{}
			for seed := uint64(0); seed < uint64(trials); seed++ {
				m := initial
				r := rng.NewRNG(seed)
				if !m.sources(0, &r) {
					t.Fatalf("enumerable failure prefix%d seat%d checks%d", prefix, seat, m.checks)
				}
				m.shuffleClasses(&r)
				w, ok := m.world(&o)
				if !ok || !MatchesObservation(o, w) {
					t.Fatal("enumerable bad world", prefix, seat)
				}
				for j, i := range variable {
					p := m.places[i]
					values[j] = w.cards[p.age-1][p.packet][p.turn-1]
				}
				key := string(values)
				if !accepted[key] {
					t.Fatal("sampler outside replay support", key)
				}
				seen[key]++
			}
			if len(accepted) != len(seen) {
				t.Fatal("sampler missed accepted worlds", prefix, seat, len(accepted), len(seen))
			}
			lo, hi := trials, 0
			tv := 0.0
			for _, count := range seen {
				lo = min(lo, count)
				hi = max(hi, count)
				tv += math.Abs(float64(count)/float64(trials)-1/float64(len(seen))) / 2
			}
			t.Logf("prefix%d seat%d places%d support%d trials%d frequency_min%d max%d empirical_TV_uniform=%.5f", prefix, seat, len(variable), len(seen), trials, lo, hi, tv)
		}
	}
}
