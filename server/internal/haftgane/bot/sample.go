package bot

import (
	"context"
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// SampleCounters are per-call diagnostics for later calibration. They contain
// no seed, keys, private witness, fingerprint or persisted preset information.
type SampleCounters struct {
	Attempts, Worlds, Checks, NoWorldFallbacks, BudgetFallbacks int
}

// RootSample belongs only to the root search worker. State is never passed to
// a simulated seat policy: that seat must receive rules.Project instead. This
// preparation API enables no level, performs no rollout and retains no cache.
type RootSample struct {
	state    rules.State
	Ready    bool
	Fallback Decision
	Counters SampleCounters
}

func (s RootSample) State() (rules.State, bool) { return s.state, s.Ready }

// SampleRoot consumes only permitted input, independent policy entropy and an
// explicit deterministic check allowance. The worker calls it after admission;
// it must never run on the room actor. A failed sample is tried once, then
// invokes exact observation-only Greedy. Cancellation yields no replacement.
// A later sampled preset must freeze the allowance; this dormant API changes
// neither DefaultWork nor any released policy/preset/version.
func SampleRoot(ctx context.Context, o rules.Observation, offered []rules.Choice, entropy uint64, maxChecks int) (RootSample, error) {
	if err := ctx.Err(); err != nil {
		return RootSample{}, err
	}
	if !o.Now.Seat.Valid(o.Now.Seats) || len(offered) == 0 || len(offered) > rules.MaxTurnChoices || maxChecks < 0 || maxChecks > rules.MaxSamplerChecks {
		return RootSample{}, ErrInput
	}
	hash, err := strconv.ParseUint(o.Fingerprint(), 16, 64)
	if err != nil {
		return RootSample{}, ErrInput
	}
	// A separate domain keeps future sampling independent of baseline noise.
	r := rng.NewRNG2(rng.MixSeed(entropy, 0x73616d706c657631), hash)
	w, cost, ok := rules.DeterminizeControlled(o, &r, maxChecks, func() bool { return ctx.Err() != nil })
	if err := ctx.Err(); err != nil {
		return RootSample{}, err
	}
	out := RootSample{Counters: SampleCounters{Attempts: 1, Checks: cost.Checks}}
	if ok {
		out.state, out.Ready = w.State(), true
		out.Counters.Worlds = 1
		return out, nil
	}
	out.Counters.NoWorldFallbacks = 1
	if cost.Exhausted {
		out.Counters.BudgetFallbacks = 1
	}
	out.Fallback, err = Fallback(ctx, o, offered)
	if err != nil {
		return RootSample{}, err
	}
	return out, nil
}
