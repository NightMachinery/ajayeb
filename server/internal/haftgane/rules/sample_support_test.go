package rules

import (
	"os"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

//  1. The true world must be inside the constraint model's support at every
//     seat and prefix (completeness on the real past).
//  2. On positions with discard opportunities, permute up to six hidden places
//     (up to five pile entries from any Age, then other hidden places) and
//     compare model support with checked replay (samplerReplayOracle).
func TestSamplerSupportSweep(t *testing.T) {
	games := 10
	if text := os.Getenv("HAFTGANE_SAMPLER_GAMES"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 100000 {
			t.Fatal("invalid HAFTGANE_SAMPLER_GAMES")
		}
		games = n
	}
	truth, enumerated, assignments, accepted := 0, 0, 0, 0
	var ages [NumAges]int
	rebuildSources := 0
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for seed := 0; seed < games; seed++ {
			setup := Setup{Seats: n, Seed: uint64(seed) + 0x9a1, Options: Options{Sides: SideRule(seed % 3), HideAtEnd: seed%2 == 1}}
			run := samplerSweepPlay(t, setup, uint64(seed)+311, (seed/3)%3)
			for prefix, g := range run.games {
				for s := Seat(0); s < Seat(n); s++ {
					o := g.Observe(s)
					actual := samplerWitness(t, g)
					var initial sampleModel
					initial.limit = MaxSamplerChecks
					if !initial.prepare(&o) {
						t.Fatalf("prepare n%d seed%d prefix%d seat%d", n, seed, prefix, s)
					}
					m := initial
					for i := range m.places[:m.np] {
						p := &m.places[i]
						if p.turn == 0 {
							continue
						}
						p.base = catalog.DesignSet{}
						p.base.Add(actual.cards[p.age-1][p.packet][p.turn-1])
					}
					r := rng.NewRNG(uint64(prefix))
					ok := m.sources(0, &r)
					if ok {
						_, ok = m.world(&o)
					}
					if !ok {
						t.Fatalf("true world refused n%d seed%d prefix%d seat%d", n, seed, prefix, s)
					}
					truth++
					if initial.no == 0 || (int(s)+prefix)%3 != 0 {
						continue
					}
					var pile, other []int
					for i, p := range initial.places[:initial.np] {
						if p.turn == 0 || p.fixed != 0 {
							continue
						}
						if p.birth != 0 {
							pile = append(pile, i)
						} else if p.age <= g.State.Age {
							other = append(other, i)
						}
					}
					rr := rng.NewRNG2(uint64(seed), uint64(prefix)*8+uint64(s))
					rng.ShuffleSlice(&rr, pile)
					rng.ShuffleSlice(&rr, other)
					variable := append([]int{}, pile...)
					if len(variable) > 5 {
						variable = variable[:5]
					}
					for _, i := range other {
						if len(variable) >= 6 {
							break
						}
						variable = append(variable, i)
					}
					if len(variable) < 2 {
						continue
					}
					base := initial
					for i := range base.places[:base.np] {
						p := &base.places[i]
						if p.turn == 0 {
							continue
						}
						chosen := false
						for _, v := range variable {
							chosen = chosen || v == i
						}
						if chosen {
							continue
						}
						p.base = catalog.DesignSet{}
						p.base.Add(actual.cards[p.age-1][p.packet][p.turn-1])
					}
					values := make([]catalog.DesignID, len(variable))
					for j, i := range variable {
						p := base.places[i]
						values[j] = actual.cards[p.age-1][p.packet][p.turn-1]
					}
					enumerated++
					for _, i := range variable {
						p := base.places[i]
						ages[p.age-1]++
						if p.birth == 0 {
							continue
						}
						id := actual.cards[p.age-1][p.packet][p.turn-1]
						for _, op := range base.ops[:base.no] {
							if op.design == id && p.birth <= op.moment {
								rebuildSources++
							}
						}
					}
					var enumerate func(int)
					enumerate = func(at int) {
						if at == len(values) {
							cards := actual.cards
							for j, i := range variable {
								p := base.places[i]
								cards[p.age-1][p.packet][p.turn-1] = values[j]
							}
							_, replayOK := samplerReplayOracle(run.games[0], run.batches[:prefix], cards, actual.state.BoxGuilds, o)
							mm := base
							for j, i := range variable {
								mm.places[i].base = catalog.DesignSet{}
								mm.places[i].base.Add(values[j])
							}
							r := rng.NewRNG(91)
							solverOK := mm.sources(0, &r)
							if solverOK {
								_, solverOK = mm.world(&o)
							}
							assignments++
							if replayOK {
								accepted++
							}
							if replayOK != solverOK {
								t.Fatalf("support differs n%d seed%d prefix%d seat%d values%v replay%v solver%v", n, seed, prefix, s, values, replayOK, solverOK)
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
				}
			}
		}
	}
	t.Logf("truth=%d enumerated=%d assignments=%d accepted=%d variable_ages=%v rebuild_sources=%d", truth, enumerated, assignments, accepted, ages, rebuildSources)
	if rebuildSources == 0 || ages[0] == 0 || ages[1] == 0 || ages[2] == 0 {
		t.Fatal("missing cross-Age or rebuild-source coverage")
	}
}
