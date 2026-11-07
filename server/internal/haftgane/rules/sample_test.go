package rules

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func TestSamplerSweep(t *testing.T) {
	seeds := 2
	if text := os.Getenv("HAFTGANE_SAMPLER_GAMES"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 100000 {
			t.Fatal("invalid HAFTGANE_SAMPLER_GAMES")
		}
		seeds = n
	}
	positions, samples, worst := 0, 0, 0
	worstAt := ""
	var phases [PhaseOver + 1]int
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for seed := 0; seed < seeds; seed++ {
			setup := Setup{Seats: n, Seed: uint64(seed) + 0x517, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: seed%2 == 1}}
			run := samplerSweepPlay(t, setup, uint64(seed)+127, (seed/3)%3)
			for _, g := range run.games {
				positions++
				phases[g.State.Phase]++
				for s := Seat(0); s < Seat(n); s++ {
					o := g.Observe(s)
					r := rng.NewRNG2(uint64(seed), uint64(g.State.Points)*31+uint64(s))
					w, checks, ok := determinize(o, &r, MaxSamplerChecks)
					if checks > worst {
						worst = checks
						worstAt = fmt.Sprintf("n%d seed%d point%d seat%d", n, seed, g.State.Points, s)
					}
					if !ok {
						t.Fatalf("no sample n%d seed%d point%d phase%d age%d turn%d seat%d checks%d", n, seed, g.State.Points, g.State.Phase, g.State.Age, g.State.Turn, s, checks)
					}
					if !MatchesObservation(o, w) {
						t.Fatalf("invalid sample n%d seed%d point%d phase%d age%d turn%d seat%d checks%d", n, seed, g.State.Points, g.State.Phase, g.State.Age, g.State.Turn, s, checks)
					}
					checkSampleNotebook(t, o, w)
					samples++
				}
			}
		}
	}
	t.Logf("positions=%d samples=%d worst_checks=%d limit=%d phases=%v worst_at=%s", positions, samples, worst, MaxSamplerChecks, phases, worstAt)
}

func checkSampleNotebook(t *testing.T, o Observation, w World) {
	t.Helper()
	stats := o.Statistics()
	for _, p := range stats.Deductions[:stats.NumDeductions] {
		origin := packetOrigin(p.Seat, o.Now.Seats, p.Age, min(p.Turn, uint8(TurnsPerAge)))
		if w.cards[p.Age-1][origin][p.Turn-1] != p.Design {
			t.Fatal("false notebook deduction", p)
		}
	}
	c := &o.Now
	if c.Age == 0 || c.Phase == PhaseOver || c.Phase == PhaseLastCard || c.Turn >= TurnsPerAge {
		return
	}
	resolved := c.Turn - 1
	if c.Phase == PhaseDiscardBuild {
		resolved = c.Turn
	}
	var remembered [MaxSeats]bool
	for turn := uint8(1); turn <= resolved; turn++ {
		remembered[packetOrigin(c.Seat, c.Seats, c.Age, turn)] = true
	}
	var colors [catalog.NumColors]uint8
	var returning [catalog.NumDesigns + 1]uint8
	for packet := Seat(0); packet < Seat(c.Seats); packet++ {
		if !remembered[packet] {
			continue
		}
		for _, id := range w.cards[c.Age-1][packet][resolved:] {
			d, _ := catalog.DesignByID(id)
			colors[d.Color]++
		}
		// A sampled future includes arbitrary future picks, so the exact
		// return packet is available without invoking the statistics code.
		next := c.Turn + 1
		for next <= TurnsPerAge && packetHolder(packet, c.Seats, c.Age, next) != c.Seat {
			next++
		}
		if next <= TurnsPerAge && c.Seats <= 5 {
			for _, id := range w.cards[c.Age-1][packet][next-1:] {
				returning[id]++
			}
		}
	}
	for color, b := range stats.Circulating {
		if colors[color] < b.Certain || colors[color] > b.AtMost {
			t.Fatalf("circulating bound %+v actual%d color%d", b, colors[color], color)
		}
	}
	for id, b := range stats.Returning {
		if returning[id] < b.Certain || returning[id] > b.AtMost {
			t.Fatalf("returning bound %+v actual%d id%d", b, returning[id], id)
		}
	}
}

func TestSamplerValueAndFailure(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[World]()} {
		if bad := puritycheck.FixedShapeViolations(typ, typ.Name()); len(bad) != 0 {
			t.Fatal(bad)
		}
		t.Logf("%s size=%d bytes", typ.Name(), typ.Size())
	}
	g, err := NewGame(Setup{Seats: 7, Seed: 91})
	if err != nil {
		t.Fatal(err)
	}
	o := g.Observe(0)
	r := rng.NewRNG(12)
	if w, _, ok := determinize(o, &r, 0); ok || w != (World{}) {
		t.Fatal("exhaustion returned world")
	}
	if w, ok := Determinize(o, nil); ok || w != (World{}) {
		t.Fatal("nil RNG accepted")
	}
	bad := o
	bad.Now.Seat = NoSeat
	if _, ok := Determinize(bad, &r); ok {
		t.Fatal("invalid seat accepted")
	}
	w, ok := Determinize(o, &r)
	if !ok {
		t.Fatal("no sample")
	}
	st := w.State()
	st.Seed = 93
	if w.State().Seed != 0 {
		t.Fatal("state aliases witness")
	}
	w.state.Seed = 1
	if MatchesObservation(o, w) {
		t.Fatal("seed accepted")
	}
	w.state.Seed = 0
	w.cards[0][0][0] = 0
	if MatchesObservation(o, w) {
		t.Fatal("bad witness accepted")
	}
	allocs := testing.AllocsPerRun(20, func() { Determinize(o, &r) })
	if allocs != 0 {
		t.Fatalf("sampler allocates: %v", allocs)
	}
}

func FuzzMatchesObservation(f *testing.F) {
	for _, seed := range []uint64{0, 1, 7, 491, 98734} {
		f.Add(seed, uint8(3), uint8(9), uint64(21))
	}
	f.Fuzz(func(t *testing.T, seed uint64, size, prefix uint8, tape uint64) {
		n := uint8(3 + size%5)
		g, err := NewGame(Setup{Seats: n, Seed: seed, Options: Options{Sides: SideRule(seed % 3)}})
		if err != nil {
			t.Fatal(err)
		}
		for step := 0; step < int(prefix%26) && g.State.Phase != PhaseOver; step++ {
			next, _, err := g.Resolve(privacySweepBatch(g, seed, step))
			if err != nil {
				t.Fatal(err)
			}
			g = next
		}
		o := g.Observe(Seat(tape % uint64(n)))
		r := rng.NewRNG(tape)
		w, ok := Determinize(o, &r)
		if !ok || !MatchesObservation(o, w) {
			t.Fatalf("sample invalid success=%v n%d seed%d point%d", ok, n, seed, g.State.Points)
		}
		checkSampleNotebook(t, o, w)
	})
}

func BenchmarkDeterminizeSevenSeats(b *testing.B) {
	g, err := NewGame(Setup{Seats: 7, Seed: 91})
	if err != nil {
		b.Fatal(err)
	}
	o := g.Observe(0)
	r := rng.NewRNG(17)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := Determinize(o, &r); !ok {
			b.Fatal("no sample")
		}
	}
}

// The existing rules leak fixtures are usable here without importing tabletest
// (which depends on rules). Swap real future packets and a box guild, plus the
// seed. Identical permitted input must reproduce the entire opaque witness.
func TestSamplerNoninterference(t *testing.T) {
	pairs := 0
	for n := uint8(3); n <= 7; n++ {
		run := privacySweepPlay(t, Setup{Seats: n, Seed: 91, Options: Options{Sides: SidesRandom}}, 213)
		for _, a := range run.games {
			if a.State.Age >= 3 {
				continue
			}
			b := a
			b.State.Seed ^= 0x123456
			b.State.Deals[2][1], b.State.Deals[2][2] = b.State.Deals[2][2], b.State.Deals[2][1]
			boxSwap := false
			for packet := 0; packet < int(n) && !boxSwap; packet++ {
				for i, id := range b.State.Deals[2][packet] {
					d, _ := catalog.DesignByID(id)
					if d.Color == catalog.Purple {
						b.State.Deals[2][packet][i], b.State.BoxGuilds[0] = b.State.BoxGuilds[0], id
						boxSwap = true
						break
					}
				}
				canonicalIDs(b.State.Deals[2][packet][:])
			}
			canonicalIDs(b.State.BoxGuilds[:])
			for s := Seat(0); s < Seat(n); s++ {
				oa, ob := a.Observe(s), b.Observe(s)
				if oa != ob {
					t.Fatal("forbidden future changed observation")
				}
				ra, rb := rng.NewRNG(197), rng.NewRNG(197)
				wa, oka := Determinize(oa, &ra)
				wb, okb := Determinize(ob, &rb)
				if !oka || !okb || wa != wb || ra != rb {
					t.Fatal("sampler noninterference")
				}
				pairs++
			}
		}
	}
	if pairs == 0 {
		t.Fatal("no leak pairs")
	}
	t.Logf("equal complete samples across %d hidden-state pairs", pairs)
}

func TestSamplerSkippedLooksAndAllocations(t *testing.T) {
	run := samplerSkipRun(t)
	for prefix := 3; prefix < len(run.games); prefix++ {
		for seat := Seat(0); seat < 7; seat++ {
			o := run.games[prefix].Observe(seat)
			r := rng.NewRNG(397)
			for trial := 0; trial < 16; trial++ {
				w, ok := Determinize(o, &r)
				if !ok || !MatchesObservation(o, w) {
					t.Fatalf("skip/look sample prefix%d seat%d", prefix, seat)
				}
			}
			if allocs := testing.AllocsPerRun(5, func() { Determinize(o, &r) }); allocs != 0 {
				t.Fatal("constrained sampler allocates", allocs)
			}
		}
	}
}

func samplerSweepPlay(t *testing.T, setup Setup, seed uint64, policy int) privacySweepRun {
	t.Helper()
	if policy == 0 {
		return privacySweepPlay(t, setup, seed)
	}
	g, err := NewGame(setup)
	if err != nil {
		t.Fatal(err)
	}
	run := privacySweepRun{games: []Game{g}}
	r := rng.NewRNG(seed)
	name := "random"
	if policy == 2 {
		name = "simple"
	}
	for g.State.Phase != PhaseOver {
		b := Batch{Point: g.Point().ID()}
		for seat := Seat(0); seat < Seat(g.State.Seats); seat++ {
			if !g.Point().Required.Has(seat) {
				continue
			}
			b.Picks = append(b.Picks, Pick{Seat: seat, Key: gateChoice(&g, seat, name, &r).Key()})
		}
		next, _, err := g.Resolve(b)
		if err != nil {
			t.Fatal(err)
		}
		g = next
		run.batches = append(run.batches, b)
		run.games = append(run.games, g)
	}
	return run
}

func BenchmarkDeterminizeSevenSeatsConstrained(b *testing.B) {
	// Use a real natural prefix with discard opportunities, without a testing.T
	// fixture or a retained authoritative game in the measured call.
	g, err := NewGame(Setup{Seats: 7, Seed: 0, Options: Options{Sides: SidesRandom}})
	if err != nil {
		b.Fatal(err)
	}
	for step := 0; step < 16 && g.State.Phase != PhaseOver; step++ {
		next, _, err := g.Resolve(privacySweepBatch(g, 117, step))
		if err != nil {
			b.Fatal(err)
		}
		g = next
	}
	o := g.Observe(0)
	r := rng.NewRNG(17)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := Determinize(o, &r); !ok {
			b.Fatal("no sample")
		}
	}
}

func TestSamplerDiscardSourceRegression(t *testing.T) {
	const seed = 172
	run := samplerSweepPlay(t, Setup{Seats: 5, Seed: seed + 0x517, Options: Options{Sides: SideRule(seed % 3)}}, seed+127, 0)
	for _, g := range run.games {
		if g.State.Points != 15 {
			continue
		}
		o := g.Observe(0)
		r := rng.NewRNG2(seed, uint64(g.State.Points)*31)
		w, checks, ok := determinize(o, &r, MaxSamplerChecks)
		if !ok || !MatchesObservation(o, w) {
			t.Fatalf("regression success%v checks%d", ok, checks)
		}
		t.Logf("five-seat source regression checks=%d", checks)
		return
	}
	t.Fatal("regression point missing")
}

func TestSamplerClassPermutationRegression(t *testing.T) {
	const seed = 1434
	run := samplerSweepPlay(t, Setup{Seats: 6, Seed: seed + 0x517, Options: Options{Sides: SideRule(seed % 3)}}, seed+127, (seed/3)%3)
	for _, g := range run.games {
		if g.State.Points != 16 {
			continue
		}
		o := g.Observe(0)
		r := rng.NewRNG2(seed, uint64(g.State.Points)*31)
		w, checks, ok := determinize(o, &r, MaxSamplerChecks)
		if !ok || !MatchesObservation(o, w) {
			t.Fatalf("regression success%v checks%d", ok, checks)
		}
		t.Logf("six-seat class regression checks=%d", checks)
		return
	}
	t.Fatal("regression point missing")
}

// The still-open discard point forces a legal pile card even for nonowners.
func TestSamplerPendingDiscardRegression(t *testing.T) {
	const seed = 519
	run := samplerSweepPlay(t, Setup{Seats: 5, Seed: seed + 0x517, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: true}}, seed+127, (seed/3)%3)
	for _, g := range run.games {
		if g.State.Points != 14 {
			continue
		}
		o := g.Observe(0)
		for tape := uint64(0); tape < 64; tape++ {
			r := rng.NewRNG(tape)
			w, ok := Determinize(o, &r)
			if !ok || !MatchesObservation(o, w) {
				t.Fatalf("pending discard tape%d success%v", tape, ok)
			}
		}
		return
	}
	t.Fatal("regression point missing")
}

func TestSamplerSevenSeatBoundRegression(t *testing.T) {
	const seed = 1218
	run := samplerSweepPlay(t, Setup{Seats: 7, Seed: seed + 0x517, Options: Options{Sides: SideRule(seed % 3)}}, seed+127, (seed/3)%3)
	for _, g := range run.games {
		if g.State.Points != 21 {
			continue
		}
		o := g.Observe(1)
		r := rng.NewRNG2(seed, uint64(g.State.Points)*31+1)
		w, checks, ok := determinize(o, &r, MaxSamplerChecks)
		if !ok || !MatchesObservation(o, w) {
			t.Fatalf("regression success%v checks%d", ok, checks)
		}
		t.Logf("seven-seat regression checks=%d", checks)
		return
	}
	t.Fatal("regression point missing")
}

// Three feasible assignments demonstrate the search distribution explicitly:
// AB/AC/ABC with supply ABC chooses BAC and BCA a quarter each, ACB a half.
// Exact uniformity would put one third on each. No game-policy posterior is
// claimed; this is an enumerable constraint-system measurement.
func TestSamplerConstrainedDistribution(t *testing.T) {
	var counts [3]int
	const trials = 12000
	for seed := uint64(0); seed < trials; seed++ {
		var m sampleModel
		m.np, m.ng, m.no, m.limit = 3, 1, 1, MaxSamplerChecks
		for _, id := range []catalog.DesignID{2, 5, 7} {
			m.groups[0].supply[id] = 1
		}
		domains := [3]catalog.DesignSet{catalog.DesignSetOf(2, 5), catalog.DesignSetOf(2, 7), catalog.DesignSetOf(2, 5, 7)}
		for i, d := range domains {
			m.places[i] = samplePlace{allowed: d, base: d, turn: 1, class: uint8(i)}
		}
		r := rng.NewRNG(seed)
		if !m.fill(&r) {
			t.Fatal("synthetic no sample")
		}
		switch [3]catalog.DesignID{m.places[0].value, m.places[1].value, m.places[2].value} {
		case [3]catalog.DesignID{2, 7, 5}:
			counts[0]++
		case [3]catalog.DesignID{5, 2, 7}:
			counts[1]++
		case [3]catalog.DesignID{5, 7, 2}:
			counts[2]++
		default:
			t.Fatal("outside exact synthetic support")
		}
	}
	tv := 0.0
	for i, count := range counts {
		want := 0.25
		if i == 0 {
			want = 0.5
		}
		p := float64(count) / trials
		if math.Abs(p-want) > 0.03 {
			t.Fatal("distribution changed", counts)
		}
		tv += math.Abs(p-1.0/3) / 2
	}
	t.Logf("trials%d counts%v empirical_TV_uniform=%.5f exact_TV_uniform=1/6", trials, counts, tv)
}
