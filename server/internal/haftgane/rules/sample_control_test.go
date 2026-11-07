package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
)

func TestSamplerControlledIdentityAndStop(t *testing.T) {
	run := samplerSweepPlay(t, Setup{Seats: 7, Seed: 1218 + 0x517}, 1218+127, 1)
	for _, g := range run.games {
		if g.State.Points != 21 {
			continue
		}
		for seat := Seat(0); seat < 7; seat++ {
			o := g.Observe(seat)
			a, b := rng.NewRNG(71), rng.NewRNG(71)
			wa, oka := Determinize(o, &a)
			polls := 0
			wb, cost, okb := DeterminizeControlled(o, &b, MaxSamplerChecks, func() bool { polls++; return false })
			if !oka || !okb || wa != wb || a != b || cost.Stopped || cost.Exhausted || polls < 3 {
				t.Fatal("control changed uninterrupted sampling", seat, cost, polls)
			}
			for at := 1; at <= 8; at++ {
				r := rng.NewRNG(71)
				calls := 0
				w, cost, ok := DeterminizeControlled(o, &r, MaxSamplerChecks, func() bool { calls++; return calls >= at })
				if ok || w != (World{}) || !cost.Stopped || cost.Exhausted || cost.Checks > max(0, at-2) {
					t.Fatal("stopped call exposed a world", seat, at, cost)
				}
			}
			r := rng.NewRNG(71)
			w, cost, ok := DeterminizeControlled(o, &r, 16, nil)
			if ok || w != (World{}) || cost.Stopped || !cost.Exhausted || cost.Checks != 16 {
				t.Fatal("budget outcome", cost)
			}
			r = rng.NewRNG(71)
			if n := testing.AllocsPerRun(3, func() { DeterminizeControlled(o, &r, MaxSamplerChecks, func() bool { return false }) }); n != 0 {
				t.Fatal("controlled sampler allocates", n)
			}
		}
		return
	}
	t.Fatal("tail fixture missing")
}
