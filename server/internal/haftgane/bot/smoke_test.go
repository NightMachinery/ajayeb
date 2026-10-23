package bot

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Declared before strength measurement. Release seeds are never calibration.
const developmentDealBase uint64 = 0x6d170001

func smokeInterval(blocks []float64) [3]float64 {
	var out [3]float64
	for _, x := range blocks {
		out[0] += x / float64(len(blocks))
	}
	r := rng.NewRNG(0x626f6f7473747261)
	values := make([]float64, 2048)
	for i := range values {
		for range blocks {
			values[i] += blocks[r.IntN(len(blocks))] / float64(len(blocks))
		}
	}
	sort.Float64s(values)
	out[1], out[2] = values[51], values[1996]
	return out
}
func smokeReward(g *rules.Game, focal int) (float64, float64) {
	result := g.State.Result
	low, ties := 0, 0
	for s := 0; s < int(g.State.Seats); s++ {
		if s == focal {
			continue
		}
		if result.Rank[s] > result.Rank[focal] {
			low++
		}
		if result.Rank[s] == result.Rank[focal] {
			ties++
		}
	}
	credit := 0.0
	if result.Rank[focal] == 1 {
		credit = 1 / float64(ties+1)
	}
	return (float64(low) + float64(ties)/2) / float64(g.State.Seats-1), credit
}
func TestStrengthSmoke(t *testing.T) {
	if os.Getenv("M17_BOT_SMOKE") != "1" {
		t.Skip("opt-in development smoke; tournament release gates are separate")
	}
	var durations [NumLevels][rules.NumPointKinds][]int64
	games := 0
	for n := uint8(3); n <= 7; n++ {
		for _, matchup := range [][2]Level{{Greedy, Random}, {Apprentice, Random}, {Greedy, Apprentice}} {
			var ranks, wins []float64
			for block := uint64(0); block < 4; block++ {
				rank, win := 0.0, 0.0
				for focal := 0; focal < int(n); focal++ {
					g, err := rules.NewGame(rules.Setup{Seed: rng.MixSeed(developmentDealBase+block, uint64(n)), Seats: n, Options: rules.Options{Sides: rules.SideRule(block % uint64(rules.NumSideRules))}})
					if err != nil {
						t.Fatal(err)
					}
					for g.State.Phase != rules.PhaseOver {
						batch := rules.Batch{Point: g.Point().ID(), Picks: []rules.Pick{}}
						for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
							if !g.Point().Required.Has(seat) {
								continue
							}
							level := matchup[1]
							if int(seat) == focal {
								level = matchup[0]
							}
							offers := g.Legal(seat)
							o := g.Observe(seat)
							start := time.Now()
							d, err := Choose(context.Background(), level, o, offers, rng.MixSeed(0x706f6c6963797631+block, uint64(seat)), DefaultWork())
							durations[level][g.Point().Kind] = append(durations[level][g.Point().Kind], time.Since(start).Nanoseconds())
							if err != nil || !d.Complete {
								t.Fatal(d, err)
							}
							batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: d.Key})
						}
						g, _, err = g.Resolve(batch)
						if err != nil {
							t.Fatal(err)
						}
					}
					r, w := smokeReward(&g, focal)
					rank += r / float64(n)
					win += w / float64(n)
					games++
				}
				ranks = append(ranks, rank)
				wins = append(wins, win)
			}
			t.Logf("SMOKE n=%d candidate=%s field=%s blocks=4 rank=%v win=%v (mean,95%% block-bootstrap low,high)", n, matchup[0], matchup[1], smokeInterval(ranks), smokeInterval(wins))
		}
	}
	for level := Random; level < NumLevels; level++ {
		for kind := rules.PointKind(1); kind < rules.NumPointKinds; kind++ {
			xs := durations[level][kind]
			if len(xs) == 0 {
				continue
			}
			sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
			t.Logf("LATENCY level=%s point=%d samples=%d p50_us=%.3f p95_us=%.3f p99_us=%.3f", level, kind, len(xs), float64(xs[len(xs)/2])/1000, float64(xs[(len(xs)-1)*95/100])/1000, float64(xs[(len(xs)-1)*99/100])/1000)
		}
	}
	t.Logf("SMOKE complete games=%d, fallback=0; one full prescribed pass per normal ranked decision", games)
}
func BenchmarkSevenSeatPolicy(b *testing.B) {
	g, err := rules.NewGame(rules.Setup{Seed: developmentDealBase, Seats: 7, Options: rules.Options{Sides: rules.SidesRandom}})
	if err != nil {
		b.Fatal(err)
	}
	for g.State.Turn < 3 {
		batch, _ := policyBatch(b, &g, [rules.MaxSeats]Level{Greedy, Greedy, Greedy, Greedy, Greedy, Greedy, Greedy}, 71)
		g, _, err = g.Resolve(batch)
		if err != nil {
			b.Fatal(err)
		}
	}
	o, cs := g.Observe(0), g.Legal(0)
	for level := Random; level < NumLevels; level++ {
		b.Run(level.String(), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Choose(context.Background(), level, o, cs, 71, DefaultWork()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
