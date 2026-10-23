package bot

import (
	"context"
	"errors"
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

type Level uint8

const (
	Random Level = iota
	Greedy
	Apprentice
	NumLevels
)
const (
	PolicyVersion         = 1
	PresetVersion         = 1
	ApprenticeTemperature = 1500 // milli-VP; fixed-point softmax, frozen v1.
	MaxProbes             = 12
	MaxQuoteCalls         = (rules.MaxTurnChoices + 1) * MaxProbes
)

func (l Level) String() string {
	switch l {
	case Random:
		return "random"
	case Greedy:
		return "greedy"
	case Apprentice:
		return "apprentice"
	}
	return ""
}
func ParseLevel(s string) (Level, error) {
	for l := Random; l < NumLevels; l++ {
		if l.String() == s {
			return l, nil
		}
	}
	return 0, ErrInput
}

// Work is the persisted, versioned per-bot allowance. These baseline policies
// prescribe one complete bounded pass, never a table-wide divided allowance.
type Work struct{ PolicyVersion, PresetVersion, Iterations uint16 }

func DefaultWork() Work { return Work{PolicyVersion, PresetVersion, 1} }

type Counters struct{ Iterations, Ranked, Previews, Quotes, RandomDraws int }
type Decision struct {
	Key      string
	Complete bool
	Counters Counters
}
type Features struct{ Points, Military, Science, Economy, Liquidity, Chains, Wonder, Affordability, Passing, Externality int64 }

func (f Features) Total() int64 {
	return f.Points + f.Military + f.Science + f.Economy + f.Liquidity + f.Chains + f.Wonder + f.Affordability + f.Passing + f.Externality
}

type Ranked struct {
	Key      string
	Value    int64
	Features Features
}

var (
	ErrInput = errors.New("haftgane bot: invalid policy input")
	ErrWork  = errors.New("haftgane bot: unsupported policy work row")
)

// Choose has no Game/State, clock, pending key, task generation or root seed.
// Entropy is policy-only input supplied by the caller. All results/counters
// depend solely on permitted observation, offered choices, entropy and Work.
// Cancellation returns no partial key; delivery still requires actor checks.
func Choose(ctx context.Context, level Level, o rules.Observation, offered []rules.Choice, entropy uint64, work Work) (Decision, error) {
	if work != DefaultWork() {
		return Decision{}, ErrWork
	}
	if level >= NumLevels || len(offered) == 0 || len(offered) > rules.MaxTurnChoices || !o.Now.Seat.Valid(o.Now.Seats) {
		return Decision{}, ErrInput
	}
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	hash, err := strconv.ParseUint(o.Fingerprint(), 16, 64)
	if err != nil {
		return Decision{}, ErrInput
	}
	r := rng.NewRNG2(rng.MixSeed(entropy, 0x706f6c6963797631), hash)
	if level == Random {
		var ordered [rules.MaxTurnChoices]rules.Choice
		var keys [rules.MaxTurnChoices]string
		copy(ordered[:], offered)
		for i, c := range offered {
			keys[i] = c.Key()
		}
		for i := 1; i < len(offered); i++ {
			for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
				ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
				keys[j], keys[j-1] = keys[j-1], keys[j]
			}
		}
		index, draws, ok := randomIndex(&r, o, ordered[:len(offered)])
		if !ok {
			return Decision{}, ErrInput
		}
		if err := ctx.Err(); err != nil {
			return Decision{}, err
		}
		return Decision{Key: keys[index], Complete: true, Counters: Counters{Iterations: 1, RandomDraws: draws}}, nil
	}
	var buf [rules.MaxTurnChoices]Ranked
	ranked, counters, err := Rank(ctx, o, offered, buf[:0])
	if err != nil {
		return Decision{}, err
	}
	chosen := 0
	if level == Apprentice && len(ranked) > 1 {
		// Quantize exp(-delta / 1.5 VP) at 1/16 exponent steps. The integer
		// recurrence approximates e^-1/16; no libm or cross-architecture ties.
		var weights [rules.MaxTurnChoices]uint64
		var exp [257]uint64
		exp[0] = 1 << 40
		for i := 1; i < len(exp); i++ {
			exp[i] = exp[i-1] * 61565 / 65536
		}
		var total uint64
		for i, x := range ranked {
			step := min(int((ranked[0].Value-x.Value)*16/ApprenticeTemperature), 256)
			weights[i] = max(uint64(1), exp[step])
			total += weights[i]
		}
		pick := r.Uint64N(total)
		counters.RandomDraws++
		for i, w := range weights[:len(ranked)] {
			if pick < w {
				chosen = i
				break
			}
			pick -= w
		}
	}
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	return Decision{Key: ranked[chosen].Key, Complete: true, Counters: counters}, nil
}

// Fallback uses the exact same deterministic Greedy and supplied engine keys.
// It never reconstructs an action from guessed costs or an incomplete search.
func Fallback(ctx context.Context, o rules.Observation, offered []rules.Choice) (Decision, error) {
	return Choose(ctx, Greedy, o, offered, 0, DefaultWork())
}
func randomIndex(r *rng.RNG, o rules.Observation, offers []rules.Choice) (int, int, bool) {
	if o.Now.Phase != rules.PhaseTurn {
		return r.IntN(len(offers)), 1, true
	}
	var hand [rules.HandSize]catalog.DesignID
	n := 0
	for _, id := range o.Now.Hand {
		if id == 0 {
			continue
		}
		found := false
		for _, c := range offers {
			if c.Design == id {
				found = true
				break
			}
		}
		if found {
			hand[n] = id
			n++
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	id := hand[r.IntN(n)]
	var kinds [rules.NumActionKinds]rules.ActionKind
	used := 0
	for k := rules.ActionKind(1); k < rules.NumActionKinds; k++ {
		for _, c := range offers {
			if c.Design == id && c.Kind == k {
				kinds[used] = k
				used++
				break
			}
		}
	}
	if used == 0 {
		return 0, 0, false
	}
	kind := kinds[r.IntN(used)]
	var indices [rules.MaxTurnChoices]int
	count := 0
	for i, c := range offers {
		if c.Design == id && c.Kind == kind {
			indices[count] = i
			count++
		}
	}
	return indices[r.IntN(count)], 3, true
}
