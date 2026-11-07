package bot

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func ageHistory(t testing.TB, setup rules.Setup, tape uint64, stages bool) []rules.Game {
	t.Helper()
	g, err := rules.NewGame(setup)
	if err != nil {
		t.Fatal(err)
	}
	games := []rules.Game{g}
	r := rng.NewRNG(tape)
	for g.State.Phase != rules.PhaseOver {
		batch := rules.Batch{Point: g.Point().ID(), Picks: []rules.Pick{}}
		for seat := rules.Seat(0); seat < rules.Seat(g.State.Seats); seat++ {
			if !g.Point().Required.Has(seat) {
				continue
			}
			cs := g.Legal(seat)
			pick := cs[r.IntN(len(cs))]
			if stages {
				for _, c := range cs {
					if c.Kind == rules.ActStage || c.Kind == rules.ActSide && c.Side == catalog.Night {
						pick = c
						break
					}
				}
			}
			batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: pick.Key()})
		}
		g, _, err = g.Resolve(batch)
		if err != nil {
			t.Fatal(err)
		}
		if err := g.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
		games = append(games, g)
	}
	return games
}

func ageTail(t testing.TB) rules.Game {
	t.Helper()
	// Exact seven-seat regression from the sampler reviews, not a dev scenario.
	games := ageHistory(t, rules.Setup{Seats: 7, Seed: 1218 + 0x517}, 1218+127, false)
	for _, g := range games {
		if g.State.Points == 21 {
			return g
		}
	}
	t.Fatal("missing saved tail")
	return rules.Game{}
}

func ageSweepGames(t testing.TB) int {
	t.Helper()
	games := 1
	if text := os.Getenv("HAFTGANE_ROLLOUT_GAMES"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 10000 {
			t.Fatal("invalid HAFTGANE_ROLLOUT_GAMES")
		}
		games = n
	}
	return games
}

func TestAgeSweep(t *testing.T) {
	games, decisions, worlds := ageSweepGames(t), 0, 0
	var phases [rules.NumPhases]int
	checks, batches := 0, 0
	// Validate every sampled witness outside the production runner hot path.
	sampler := func(ctx context.Context, o rules.Observation, entropy uint64, limit int) (rules.State, rules.SamplerCost, bool, error) {
		hash, err := strconv.ParseUint(o.Fingerprint(), 16, 64)
		if err != nil {
			return rules.State{}, rules.SamplerCost{}, false, err
		}
		r := rng.NewRNG2(rng.MixSeed(entropy, 0x73616d706c657631), hash)
		w, cost, ok := rules.DeterminizeControlled(o, &r, limit, func() bool { return ctx.Err() != nil })
		if ok && !rules.MatchesObservation(o, w) {
			t.Fatal("inconsistent sweep world")
		}
		return w.State(), cost, ok, ctx.Err()
	}
	for n := uint8(3); n <= 7; n++ {
		for seed := 0; seed < games; seed++ {
			setup := rules.Setup{Seats: n, Seed: uint64(seed) + 0x7671, Options: rules.Options{Sides: rules.SideRule(seed % 3)}}
			history := ageHistory(t, setup, uint64(seed)+51, seed%2 == 0)
			for _, g := range history {
				for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
					if !g.Point().Required.Has(seat) {
						continue
					}
					phases[g.State.Phase]++
					o, cs := g.Observe(seat), g.Legal(seat)
					out, err := chooseAge(context.Background(), o, cs, uint64(seed)+uint64(seat)*31, AgeWork{Attempts: 1, Candidates: 4, SamplerChecks: rules.MaxSamplerChecks}, sampler, rolloutPolicy)
					if err != nil || !out.Complete || out.Fallback || out.Counters.SampleAttempts != 1 || out.Counters.CompletedSweeps != 1 || out.Counters.Worlds != 1 || out.Counters.GreedyFallbacks != 0 || out.Counters.NoWorldSamples != 0 {
						t.Fatal("sweep failed", n, seed, g.Point(), seat, out.Counters, err)
					}
					found := false
					for _, c := range cs {
						found = found || out.Key == c.Key()
					}
					if !found || out.Counters.Batches > int(out.Count)*rules.MaxPoints || out.Counters.PolicyChoices > out.Counters.Batches*int(n) {
						t.Fatal("work/membership bound", out.Counters)
					}
					decisions++
					worlds += out.Counters.Worlds
					checks += out.Counters.SamplerChecks
					batches += out.Counters.Batches
				}
			}
		}
	}
	if phases[rules.PhaseSides] == 0 || phases[rules.PhaseTurn] == 0 {
		t.Fatal("missing phases", phases)
	}
	t.Logf("games=%d decisions=%d worlds=%d phases=%v checks=%d simulated_batches=%d fallback=0", games*5, decisions, worlds, phases, checks, batches)
}
