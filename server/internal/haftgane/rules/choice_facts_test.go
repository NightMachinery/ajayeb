package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Engine facts keep exact scores and placements in the rules package. Other
// seats sell, so the isolated preview and real batch have identical own income
// counts and no unknown same-round construction or incoming trade receipts.
func TestObservedChoiceFactsAgainstResolution(t *testing.T) {
	checked := 0
	for seed := uint64(0); seed < 12; seed++ {
		g, err := NewGame(Setup{Seed: seed, Seats: 3, Options: Options{Sides: SidesRandom}})
		if err != nil {
			t.Fatal(err)
		}
		for g.State.Phase != PhaseOver {
			if g.State.Phase == PhaseTurn && g.State.Turn < 6 {
				for seat := Seat(0); seat < 3; seat++ {
					o := g.Observe(seat)
					base, ok := o.Now.Facts(seat)
					if !ok || base.Score != scoreCity(&g.State, seat) {
						t.Fatal("facts do not match real score")
					}
					for _, c := range g.Legal(seat) {
						if c.Kind != ActBuild && c.Kind != ActSell && c.Kind != ActStage {
							continue
						}
						next, _, err := g.Resolve(actionBatch(g, seat, c.Action))
						if err != nil {
							t.Fatal(err)
						}
						preview, facts, ok := o.Preview(c)
						if !ok || facts.Score != scoreCity(&next.State, seat) || preview.Cities[seat].Coins != next.State.Cities[seat].Coins {
							t.Fatal("isolated preview differs from actual", seed, g.Point(), c, ok)
						}
						checked++
					}
				}
			}
			g, _, err = g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if checked < 100 {
		t.Fatal("facts coverage", checked)
	}
	t.Log("score/placement comparisons", checked)
	// Optimal wildcard science and both neighbour thresholds come from the engine.
	o := Observation{Now: Current{Seat: 0, Seats: 3, Age: 3, Turn: 6, Phase: PhaseTurn}}
	for s := range 3 {
		o.Now.Cities[s] = PublicCity{Board: catalog.BoardID(s + 1), Side: catalog.Day}
	}
	o.Now.Cities[0].Buildings = catalog.DesignSetOf(25, 26, 76)
	f, ok := o.Now.Facts(0)
	if !ok || f.Score.Green != 10 {
		t.Fatal("wildcard set", f, ok)
	}
}
