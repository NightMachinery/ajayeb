package bot

import (
	"bytes"
	"context"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// A preference is sampled once per seat and comparison sweep, then persists
// for the entire rollout and across its root alternatives. These experimental
// policies have no access to the runner's sampled state or other seats' memory.
type rolloutPreference uint8

const (
	balancedPreference rolloutPreference = iota
	economicPreference
	sciencePreference
	militaryPreference
	numRolloutPreferences
)

type seatPolicy func(context.Context, rules.Observation, []rules.Choice, rolloutPreference, uint64) (rules.Action, error)

func rolloutPolicy(ctx context.Context, o rules.Observation, offered []rules.Choice, preference rolloutPreference, entropy uint64) (rules.Action, error) {
	if len(offered) == 0 || len(offered) > rules.MaxTurnChoices || !o.Now.Seat.Valid(o.Now.Seats) {
		return rules.Action{}, ErrInput
	}
	var counts Counters
	e, err := prepare(ctx, o, &counts)
	if err != nil {
		return rules.Action{}, err
	}
	var chosen rules.Action
	var best int64
	var key, bestKey [rules.MaxKeyBytes]byte
	bestLen := 0
	for _, choice := range offered {
		if err := ctx.Err(); err != nil {
			return rules.Action{}, err
		}
		features, err := e.evaluate(ctx, choice, &counts)
		if err != nil {
			return rules.Action{}, err
		}
		encoded := choice.AppendKey(key[:0])
		value := features.Total()
		switch preference {
		case economicPreference:
			value += features.Economy + features.Liquidity + features.Affordability + features.Chains
		case sciencePreference:
			value += features.Science
			if sciencePlacement(choice) {
				value += features.Points
			}
		case militaryPreference:
			value += features.Military
		}
		// Key-indexed noise shares a draw for a common action even when a
		// different candidate changes the offered set or its ranking order.
		r := rng.NewRNG2(entropy, rng.Digest(encoded))
		value += int64(r.IntN(2*ApprenticeTemperature+1) - ApprenticeTemperature)
		if bestLen == 0 || value > best || value == best && bytes.Compare(encoded, bestKey[:bestLen]) < 0 {
			chosen = choice.Action
			best = value
			bestLen = copy(bestKey[:], encoded)
		}
	}
	if err := ctx.Err(); err != nil {
		return rules.Action{}, err
	}
	return chosen, nil
}

// cutoffValue estimates this seat's final score from its own observation.
// At the terminal point all future terms disappear and exact scoring wins.
func cutoffValue(o rules.Observation) (int64, int16, bool) {
	value, coins, err := cutoffValueControlled(context.Background(), o)
	return value, coins, err == nil
}

func cutoffValueControlled(ctx context.Context, o rules.Observation) (int64, int16, error) {
	var counts Counters
	e, err := prepare(ctx, o, &counts)
	if err != nil {
		return 0, 0, err
	}
	return int64(e.own.Score.Total)*1000 + e.before.Total(), o.Now.Cities[o.Now.Seat].Coins, nil
}

// Each term is exact at this scale for N=3..7 and every possible winner tie.
// The 3/4 rank + 1/4 win blend is exploratory, not a released preset.
const AgeRewardScale int64 = 3360

func ageReward(values [rules.MaxSeats]int64, coins [rules.MaxSeats]int16, seats uint8, focal rules.Seat) int64 {
	below, tied, winners := int64(0), int64(0), int64(0)
	first := true
	for s := rules.Seat(0); s < rules.Seat(seats); s++ {
		if values[s] > values[focal] || values[s] == values[focal] && coins[s] > coins[focal] {
			first = false
		}
		if s == focal {
			continue
		}
		if values[s] < values[focal] || values[s] == values[focal] && coins[s] < coins[focal] {
			below++
		} else if values[s] == values[focal] && coins[s] == coins[focal] {
			tied++
		}
	}
	if first {
		winners = tied + 1
	}
	reward := 3 * AgeRewardScale * (2*below + tied) / (8 * int64(seats-1))
	if winners != 0 {
		reward += AgeRewardScale / (4 * winners)
	}
	return reward
}

// A science specialist also values immediate science points. Use only the
// policy's offered design, never any unobserved hand or sampled future deal.
func sciencePlacement(c rules.Choice) bool {
	d, ok := catalog.DesignByID(c.Design)
	return ok && c.Kind == rules.ActBuild && d.Color == catalog.Green
}
