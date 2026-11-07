package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func ageGame(t testing.TB, seats uint8, seed uint64) rules.Game {
	t.Helper()
	g, err := rules.NewGame(rules.Setup{Seats: seats, Seed: seed, Options: rules.Options{Sides: rules.SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestAgePairedWorkAndFallback(t *testing.T) {
	for n := uint8(3); n <= 7; n++ {
		g := ageGame(t, n, 814)
		o, cs := g.Observe(0), g.Legal(0)
		work := AgeWork{Attempts: 2, Candidates: 4, SamplerChecks: rules.MaxSamplerChecks}
		a, err := ChooseAge(context.Background(), o, cs, 91, work)
		if err != nil || !a.Complete || a.Fallback || a.Counters.CompletedSweeps != 2 || a.Counters.Worlds != 2 || a.Counters.SampleAttempts != 2 || a.Counters.Comparisons != 2*int(a.Count) || a.Counters.GreedyRankings != 1 {
			t.Fatal("paired work", n, a, err)
		}
		for i, x := range a.Ranked[:a.Count] {
			found := false
			for _, c := range cs {
				found = found || c.Key() == x.Key
			}
			if !found || x.Sum < 0 || x.Sum > 2*AgeRewardScale || x.SumSquares < 0 || x.DeltaSquares < 0 {
				t.Fatal("invalid comparison", x)
			}
			if i > 0 && (x.Sum > a.Ranked[i-1].Sum || x.Sum == a.Ranked[i-1].Sum && x.Key < a.Ranked[i-1].Key) {
				t.Fatal("ranking order")
			}
		}
		reverse := append([]rules.Choice{}, cs...)
		for i, j := 0, len(reverse)-1; i < j; i, j = i+1, j-1 {
			reverse[i], reverse[j] = reverse[j], reverse[i]
		}
		b, err := ChooseAge(context.Background(), o, reverse, 91, work)
		if err != nil || a != b {
			t.Fatal("not repeatable under offer order", n, err)
		}
		greedy, _ := Fallback(context.Background(), o, cs)
		work.SamplerChecks = 0
		failed, err := ChooseAge(context.Background(), o, cs, 91, work)
		if err != nil || failed.Key != greedy.Key || !failed.Fallback || !failed.Complete || failed.Count != 0 || failed.Counters.NoWorldSamples != 2 || failed.Counters.BudgetOutSamples != 2 || failed.Counters.GreedyFallbacks != 1 || failed.Counters.GreedyRankings != 1 || failed.Counters.Comparisons != 0 {
			t.Fatal("once-per-decision fallback", failed, err)
		}
		work.Attempts = 0
		failed, err = ChooseAge(context.Background(), o, cs, 91, work)
		if err != nil || !failed.Fallback || failed.Counters.SampleAttempts != 0 || failed.Counters.GreedyFallbacks != 1 {
			t.Fatal("zero comparison budget", failed, err)
		}
	}
}

func TestAgePartialSamplesAndCancellation(t *testing.T) {
	g := ageGame(t, 7, 831)
	o, cs := g.Observe(0), g.Legal(0)
	work := AgeWork{Attempts: 3, Candidates: 4, SamplerChecks: rules.MaxSamplerChecks}
	calls := 0
	sample := func(ctx context.Context, o rules.Observation, entropy uint64, limit int) (rules.State, rules.SamplerCost, bool, error) {
		calls++
		if calls == 2 {
			return rules.State{}, rules.SamplerCost{Checks: 12, Exhausted: true}, false, nil
		}
		return sampledState(ctx, o, entropy, limit)
	}
	a, err := chooseAge(context.Background(), o, cs, 91, work, sample, rolloutPolicy)
	if err != nil || a.Fallback || a.Counters.NoWorldSamples != 1 || a.Counters.BudgetOutSamples != 1 || a.Counters.Worlds != 2 || a.Counters.CompletedSweeps != 2 || a.Counters.Comparisons != 2*int(a.Count) || a.Counters.GreedyFallbacks != 0 || a.Counters.GreedyRankings != 1 {
		t.Fatal("partial sample semantics", a, err)
	}
	// Even a completed first sweep cannot become a cancelled incumbent.
	completedCtx, completedCancel := context.WithCancel(context.Background())
	defer completedCancel()
	calls = 0
	afterSweep := func(ctx context.Context, o rules.Observation, entropy uint64, limit int) (rules.State, rules.SamplerCost, bool, error) {
		calls++
		if calls == 2 {
			completedCancel()
			return rules.State{}, rules.SamplerCost{}, false, ctx.Err()
		}
		return sampledState(ctx, o, entropy, limit)
	}
	if out, err := chooseAge(completedCtx, o, cs, 91, work, afterSweep, rolloutPolicy); calls != 2 || !errors.Is(err, context.Canceled) || out != (AgeDecision{}) {
		t.Fatal("completed incumbent escaped cancellation", calls, out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if out, err := ChooseAge(ctx, o, cs, 91, work); !errors.Is(err, context.Canceled) || out != (AgeDecision{}) {
		t.Fatal("pre-cancel exposed result", out, err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	picks := 0
	policy := func(ctx context.Context, o rules.Observation, cs []rules.Choice, pref rolloutPreference, entropy uint64) (rules.Action, error) {
		picks++
		if picks == 5 {
			cancel()
		}
		return rolloutPolicy(ctx, o, cs, pref, entropy)
	}
	if out, err := chooseAge(ctx, o, cs, 91, work, sampledState, policy); picks != 5 || !errors.Is(err, context.Canceled) || out != (AgeDecision{}) {
		t.Fatal("in-rollout cancel exposed partial sweep", picks, out, err)
	}
}

func TestAgeFrozenRoundAndHorizon(t *testing.T) {
	g := ageGame(t, 7, 814)
	root := g.Legal(0)[0].Action
	var first [rules.MaxSeats]rules.Action
	var seen [rules.MaxSeats]bool
	policy := func(ctx context.Context, o rules.Observation, cs []rules.Choice, pref rolloutPreference, entropy uint64) (rules.Action, error) {
		if o.Now.Age != 1 {
			t.Fatal("policy ran beyond Age horizon", o.Now.Age)
		}
		a, err := rolloutPolicy(ctx, o, cs, pref, entropy)
		if o.Now.Turn == 1 {
			if o != (rules.Observation{Now: rules.Project(&g.State, o.Now.Seat)}) {
				t.Fatal("simultaneous pick saw a placement or private memory")
			}
			if seen[o.Now.Seat] && first[o.Now.Seat] != a {
				t.Fatal("root candidate changed a simultaneous opponent pick")
			}
			seen[o.Now.Seat], first[o.Now.Seat] = true, a
		}
		return a, err
	}
	var counts AgeCounters
	var scratch [rules.MaxTurnChoices]rules.Choice
	if _, err := simulateAge(context.Background(), g.State, 0, g.Observe(0).Know, root, 914, policy, &counts, scratch[:0]); err != nil {
		t.Fatal(err)
	}
	if _, err := simulateAge(context.Background(), g.State, 0, g.Observe(0).Know, g.Legal(0)[1].Action, 914, policy, &counts, scratch[:0]); err != nil {
		t.Fatal(err)
	}
	for s := rules.Seat(1); s < 7; s++ {
		if !seen[s] {
			t.Fatal("missing simultaneous seat", s)
		}
	}
	c := g.Observe(1).Now
	tape := policyTape(2, c, 1)
	c.Points++ // Runtime point counts cannot shift a logical policy tape.
	if policyTape(2, c, 1) != tape {
		t.Fatal("point counter changed tape")
	}
}

func TestAgeMemoryMatchesCheckedHistory(t *testing.T) {
	var phases [rules.NumPhases]int
	looks := 0
	for seed := uint64(0); seed < 40; seed++ {
		g, err := rules.NewGame(rules.Setup{Seats: 7, Seed: seed + 218})
		if err != nil {
			t.Fatal(err)
		}
		var memory [rules.MaxSeats]rules.Knowledge
		for g.State.Phase != rules.PhaseOver {
			phases[g.State.Phase]++
			var observations [rules.MaxSeats]rules.Observation
			var actions [rules.MaxSeats]rules.Action
			batch := rules.Batch{Point: g.Point().ID(), Picks: []rules.Pick{}}
			for seat := rules.Seat(0); seat < 7; seat++ {
				observations[seat] = g.Observe(seat)
				if memory[seat] != observations[seat].Know {
					t.Fatal("memory differs from checked knowledge", seed, g.Point(), seat)
				}
				if !g.Point().Required.Has(seat) {
					continue
				}
				cs := g.Legal(seat)
				r := rng.NewRNG2(seed, uint64(g.State.Points)*8+uint64(seat))
				pick := cs[r.IntN(len(cs))]
				// Exercise ability stages as soon as they become affordable.
				for _, c := range cs {
					if c.Kind == rules.ActStage || c.Kind == rules.ActSide && c.Side == catalog.Night {
						pick = c
						break
					}
				}
				actions[seat] = pick.Action
				batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: pick.Key()})
			}
			next, _, err := g.Resolve(batch)
			if err != nil {
				t.Fatal(err)
			}
			rememberRound(&memory, observations, actions, &next.State)
			g = next
		}
		for seat := rules.Seat(0); seat < 7; seat++ {
			value, coins, ok := cutoffValue(g.Observe(seat))
			if !ok || value != int64(g.State.Result.Scores[seat].Total)*1000 || coins != g.State.Cities[seat].Coins {
				t.Fatal("terminal cutoff is not exact", value, coins)
			}
			if memory[seat] != g.Observe(seat).Know {
				t.Fatal("terminal memory differs", seed, seat)
			}
			for _, look := range memory[seat].PileLooks {
				if look.Age != 0 {
					looks++
				}
			}
		}
	}
	if phases[rules.PhaseDiscardBuild] == 0 || phases[rules.PhaseLastCard] == 0 || looks == 0 {
		t.Fatal("missing ability coverage", phases, looks)
	}
	t.Log("checked memory phases", phases, "owner looks", looks)
}

func TestAgeRewardAndCutoff(t *testing.T) {
	var values [rules.MaxSeats]int64
	var coins [rules.MaxSeats]int16
	for n := uint8(3); n <= 7; n++ {
		if got := ageReward(values, coins, n, 0); got != 3*AgeRewardScale/8+AgeRewardScale/(4*int64(n)) {
			t.Fatal("tied reward", n, got)
		}
	}
	values[0] = 1
	if ageReward(values, coins, 7, 0) != AgeRewardScale || ageReward(values, coins, 7, 1) == AgeRewardScale {
		t.Fatal("winner reward")
	}
	values[0], coins[0] = 0, 1
	if ageReward(values, coins, 7, 0) != AgeRewardScale {
		t.Fatal("coins tie break")
	}
}

func TestAgeRootMemoryAndSides(t *testing.T) {
	history := ageHistory(t, rules.Setup{Seats: 7, Seed: 218}, 77, true)
	g := history[2]
	if g.State.Phase != rules.PhaseTurn {
		t.Fatal("memory fixture")
	}
	own := g.Observe(0).Know
	rootSeen, opponentSeen := false, false
	policy := func(ctx context.Context, o rules.Observation, cs []rules.Choice, pref rolloutPreference, entropy uint64) (rules.Action, error) {
		if o.Now.Seat == 0 {
			rootSeen = true
			for age := range own.Picks {
				for turn, pick := range own.Picks[age] {
					if pick != 0 && (o.Know.Picks[age][turn] != pick || o.Know.Seen[age][turn] != own.Seen[age][turn]) {
						t.Fatal("root forgot prior legitimate knowledge")
					}
				}
			}
		} else if o.Now.Points == g.State.Points {
			opponentSeen = true
			if o.Know != (rules.Knowledge{}) {
				t.Fatal("opponent borrowed pre-root notebook")
			}
		}
		return rolloutPolicy(ctx, o, cs, pref, entropy)
	}
	var counts AgeCounters
	var scratch [rules.MaxTurnChoices]rules.Choice
	if _, err := simulateAge(context.Background(), g.State, 0, own, g.Legal(0)[0].Action, 51, policy, &counts, scratch[:0]); err != nil || !rootSeen || !opponentSeen {
		t.Fatal("memory boundary", rootSeen, opponentSeen, err)
	}
	sides := history[0]
	count := 0
	policy = func(ctx context.Context, o rules.Observation, cs []rules.Choice, pref rolloutPreference, entropy uint64) (rules.Action, error) {
		if o.Now.Phase == rules.PhaseSides {
			count++
			if o.Now.Hand != ([rules.HandSize]catalog.DesignID{}) || o.Now.Kept != 0 || o.Know != (rules.Knowledge{}) {
				t.Fatal("side choice saw future cards")
			}
		}
		return rolloutPolicy(ctx, o, cs, pref, entropy)
	}
	counts = AgeCounters{}
	if _, err := simulateAge(context.Background(), sides.State, 0, sides.Observe(0).Know, sides.Legal(0)[0].Action, 51, policy, &counts, scratch[:0]); err != nil || count != 6 {
		t.Fatal("side boundary", count, err)
	}
}

func TestAgeTerminalResolutionAndWorkValidation(t *testing.T) {
	history := ageHistory(t, rules.Setup{Seats: 7, Seed: 218}, 77, true)
	var final rules.Game
	for _, g := range history {
		if g.State.Age == 3 && g.State.Turn == 6 && g.State.Phase == rules.PhaseTurn {
			final = g
			break
		}
	}
	if final.State.Seats != 7 {
		t.Fatal("terminal fixture")
	}
	pick := func(cs []rules.Choice) rules.Action {
		for _, c := range cs {
			if c.Kind == rules.ActSell {
				return c.Action
			}
		}
		return cs[0].Action
	}
	policy := func(_ context.Context, _ rules.Observation, cs []rules.Choice, _ rolloutPreference, _ uint64) (rules.Action, error) {
		return pick(cs), nil
	}
	var counts AgeCounters
	var scratch [rules.MaxTurnChoices]rules.Choice
	reward, err := simulateAge(context.Background(), final.State, 0, final.Observe(0).Know, pick(final.Legal(0)), 51, policy, &counts, scratch[:0])
	if err != nil {
		t.Fatal(err)
	}
	for final.State.Phase != rules.PhaseOver {
		batch := rules.Batch{Point: final.Point().ID()}
		for seat := rules.Seat(0); seat < 7; seat++ {
			if final.Point().Required.Has(seat) {
				batch.Picks = append(batch.Picks, rules.Pick{Seat: seat, Key: pick(final.Legal(seat)).Key()})
			}
		}
		final, _, err = final.Resolve(batch)
		if err != nil {
			t.Fatal(err)
		}
	}
	var values [rules.MaxSeats]int64
	var coins [rules.MaxSeats]int16
	for seat := rules.Seat(0); seat < 7; seat++ {
		values[seat], coins[seat] = int64(final.State.Result.Scores[seat].Total)*1000, final.State.Cities[seat].Coins
	}
	if reward != ageReward(values, coins, 7, 0) {
		t.Fatal("Age horizon skipped final conflicts/abilities", reward)
	}
	g := history[2]
	for _, work := range []AgeWork{{Candidates: 0}, {Candidates: rules.MaxTurnChoices + 1}, {Candidates: 8, SamplerChecks: -1}, {Candidates: 8, SamplerChecks: rules.MaxSamplerChecks + 1}} {
		if out, err := ChooseAge(context.Background(), g.Observe(0), g.Legal(0), 91, work); !errors.Is(err, ErrWork) || out != (AgeDecision{}) {
			t.Fatal("invalid work", out, err)
		}
	}
}
