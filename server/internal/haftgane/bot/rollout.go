package bot

import (
	"context"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// AgeWork bounds one sampled Age decision. An attempt is one bounded sample
// and, on success, one paired sweep. ScholarWork is the released row; offline
// experiments may supply other rows.
type AgeWork struct {
	Attempts      uint16
	Candidates    uint16
	SamplerChecks int
}

type AgeCounters struct {
	SampleAttempts, Worlds, SamplerChecks, NoWorldSamples, BudgetOutSamples int
	CompletedSweeps, Comparisons, Batches, PolicyChoices                    int
	GreedyRankings, GreedyFallbacks, Excluded                               int
}

type AgeCandidate struct {
	Key                                     string
	Sum, SumSquares, DeltaSum, DeltaSquares int64
}

// All candidates have Counters.CompletedSweeps observations. Delta statistics
// compare against ReferenceKey on the same worlds/tapes, for paired variance.
// Only complete sweeps enter these arrays. Cancellation returns the zero value.
type AgeDecision struct {
	Key, ReferenceKey string
	Complete          bool
	Fallback          bool
	Count             uint16
	Ranked            [rules.MaxTurnChoices]AgeCandidate
	Counters          AgeCounters
}

type stateSampler func(context.Context, rules.Observation, uint64, int) (rules.State, rules.SamplerCost, bool, error)

// ChooseAge is the admitted-worker entry point for Scholar. Policies see
// only Project plus their own
// rollout memory; only this runner owns the sampled root and simulated state.
func ChooseAge(ctx context.Context, o rules.Observation, offered []rules.Choice, entropy uint64, work AgeWork) (AgeDecision, error) {
	return chooseAge(ctx, o, offered, entropy, work, sampledState, rolloutPolicy)
}

func chooseAge(ctx context.Context, o rules.Observation, offered []rules.Choice, entropy uint64, work AgeWork, sample stateSampler, policy seatPolicy) (AgeDecision, error) {
	if err := ctx.Err(); err != nil {
		return AgeDecision{}, err
	}
	if !o.Now.Seat.Valid(o.Now.Seats) || len(offered) == 0 || len(offered) > rules.MaxTurnChoices {
		return AgeDecision{}, ErrInput
	}
	if work.Candidates == 0 || work.Candidates > rules.MaxTurnChoices || work.SamplerChecks < 0 || work.SamplerChecks > rules.MaxSamplerChecks {
		return AgeDecision{}, ErrWork
	}
	var rankedBuf [rules.MaxTurnChoices]Ranked
	ranked, _, err := Rank(ctx, o, offered, rankedBuf[:0])
	if err != nil {
		return AgeDecision{}, err
	}
	var candidates [rules.MaxTurnChoices]rules.Choice
	chosen := ageShortlist(ranked, offered, int(work.Candidates), candidates[:0])
	out := AgeDecision{ReferenceKey: ranked[0].Key}
	out.Counters.GreedyRankings = 1
	out.Counters.Excluded = len(offered) - len(chosen)
	var legalScratch []rules.Choice
	for attempt := uint16(0); attempt < work.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return AgeDecision{}, err
		}
		tape := rng.MixSeed(entropy, uint64(attempt)+0x6167657377656570)
		st, cost, ok, err := sample(ctx, o, tape, work.SamplerChecks)
		if err != nil {
			return AgeDecision{}, err
		}
		if err := ctx.Err(); err != nil {
			return AgeDecision{}, err
		}
		out.Counters.SampleAttempts++
		out.Counters.SamplerChecks += cost.Checks
		if !ok {
			out.Counters.NoWorldSamples++
			if cost.Exhausted {
				out.Counters.BudgetOutSamples++
			}
			continue
		}
		if rules.Project(&st, o.Now.Seat) != o.Now {
			return AgeDecision{}, ErrInput
		}
		out.Counters.Worlds++
		if legalScratch == nil {
			// A policy receives a borrowed offer and returns an Action value. Reuse
			// one bounded legal buffer for every seat, batch, candidate and attempt.
			legalScratch = make([]rules.Choice, 0, rules.MaxTurnChoices)
		}
		var rewards [rules.MaxTurnChoices]int64
		for i, c := range chosen {
			if err := ctx.Err(); err != nil {
				return AgeDecision{}, err
			}
			reward, err := simulateAge(ctx, st, o.Now.Seat, o.Know, c.Action, tape, policy, &out.Counters, legalScratch)
			if err != nil {
				return AgeDecision{}, err
			}
			rewards[i] = reward
		}
		if err := ctx.Err(); err != nil {
			return AgeDecision{}, err
		}
		// Commit together: an incomplete candidate set never enters the means.
		for i, c := range chosen {
			x := &out.Ranked[i]
			if x.Key == "" {
				x.Key = c.Key()
			}
			x.Sum += rewards[i]
			x.SumSquares += rewards[i] * rewards[i]
			delta := rewards[i] - rewards[0]
			x.DeltaSum += delta
			x.DeltaSquares += delta * delta
		}
		out.Counters.CompletedSweeps++
		out.Counters.Comparisons += len(chosen)
	}
	if err := ctx.Err(); err != nil {
		return AgeDecision{}, err
	}
	if out.Counters.CompletedSweeps == 0 {
		// The shortlist's one exact Greedy pass already did this ranking.
		// Reuse it once, regardless of how many samples failed.
		out.Key, out.Complete, out.Fallback = ranked[0].Key, true, true
		out.Counters.GreedyFallbacks = 1
		return out, nil
	}
	out.Count = uint16(len(chosen))
	for i := 1; i < len(chosen); i++ {
		for j := i; j > 0 && (out.Ranked[j].Sum > out.Ranked[j-1].Sum || out.Ranked[j].Sum == out.Ranked[j-1].Sum && out.Ranked[j].Key < out.Ranked[j-1].Key); j-- {
			out.Ranked[j], out.Ranked[j-1] = out.Ranked[j-1], out.Ranked[j]
		}
	}
	if err := ctx.Err(); err != nil {
		return AgeDecision{}, err
	}
	out.Key, out.Complete = out.Ranked[0].Key, true
	return out, nil
}

func ageShortlist(ranked []Ranked, offered []rules.Choice, cap int, dst []rules.Choice) []rules.Choice {
	var keys [rules.MaxTurnChoices]string
	for i, c := range offered {
		keys[i] = c.Key()
	}
	add := func(key string) {
		if len(dst) == cap {
			return
		}
		for i, offeredKey := range keys[:len(offered)] {
			if offeredKey == key {
				c := offered[i]
				for _, selected := range dst {
					if selected.Action == c.Action {
						return
					}
				}
				dst = append(dst, c)
				return
			}
		}
	}
	add(ranked[0].Key)
	for _, kind := range []rules.ActionKind{rules.ActBuild, rules.ActStage, rules.ActSell} {
		for _, x := range ranked {
			found := false
			for i, c := range offered {
				if c.Kind == kind && keys[i] == x.Key {
					add(x.Key)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	for _, x := range ranked {
		add(x.Key)
	}
	return dst
}

func policyTape(sweep uint64, c rules.Current, seat rules.Seat) uint64 {
	point := uint64(c.Age)<<24 | uint64(c.Turn)<<16 | uint64(c.Phase)<<8 | uint64(c.Decider)
	return rng.MixSeed(rng.MixSeed(sweep, point), uint64(seat)+0x73656174706f6c)
}

func simulateAge(ctx context.Context, st rules.State, focal rules.Seat, own rules.Knowledge, root rules.Action, tape uint64, policy seatPolicy, counts *AgeCounters, legalScratch []rules.Choice) (int64, error) {
	horizon := max(uint8(1), st.Age)
	var memory [rules.MaxSeats]rules.Knowledge
	memory[focal] = own
	var preferences [rules.MaxSeats]rolloutPreference
	for seat := rules.Seat(0); seat < rules.Seat(st.Seats); seat++ {
		r := rng.NewRNG2(tape, uint64(seat)+0x707265666572)
		preferences[seat] = rolloutPreference(r.IntN(int(numRolloutPreferences)))
	}
	for batch := 0; batch < rules.MaxPoints; batch++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if st.Phase == rules.PhaseOver || st.Age > horizon {
			var values [rules.MaxSeats]int64
			var coins [rules.MaxSeats]int16
			for seat := rules.Seat(0); seat < rules.Seat(st.Seats); seat++ {
				if err := ctx.Err(); err != nil {
					return 0, err
				}
				o := rules.Observation{Now: rules.Project(&st, seat), Know: memory[seat]}
				var err error
				values[seat], coins[seat], err = cutoffValueControlled(ctx, o)
				if err != nil {
					return 0, err
				}
			}
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			return ageReward(values, coins, st.Seats, focal), nil
		}
		var observations [rules.MaxSeats]rules.Observation
		for seat := rules.Seat(0); seat < rules.Seat(st.Seats); seat++ {
			observations[seat] = rules.Observation{Now: rules.Project(&st, seat), Know: memory[seat]}
		}
		var picks [rules.MaxSeats]rules.Action
		for seat := rules.Seat(0); seat < rules.Seat(st.Seats); seat++ {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			o := observations[seat]
			if st.Phase != rules.PhaseTurn && st.Phase != rules.PhaseSides && st.Decider != seat {
				continue
			}
			legal := rules.LegalState(&st, seat, legalScratch[:0])
			if len(legal) == 0 {
				return 0, ErrInput
			}
			var a rules.Action
			if batch == 0 && seat == focal {
				a = root
			} else {
				var err error
				a, err = policy(ctx, o, legal, preferences[seat], policyTape(tape, o.Now, seat))
				if err != nil {
					return 0, err
				}
				counts.PolicyChoices++
			}
			found := false
			for _, c := range legal {
				found = found || c.Action == a
			}
			if !found {
				return 0, ErrInput
			}
			picks[seat] = a
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		var events [rules.MaxResolutionEvents]rules.Event
		next, _, err := rules.ResolveState(&st, &picks, events[:0])
		if err != nil {
			return 0, err
		}
		counts.Batches++
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		rememberRound(&memory, observations, picks, &next)
		st = next
	}
	return 0, ErrWork
}

// The runner records only public plays and each seat's own sightings/picks.
// A private sale/tuck identity is copied only into its owner's memory.
func rememberRound(memory *[rules.MaxSeats]rules.Knowledge, before [rules.MaxSeats]rules.Observation, picks [rules.MaxSeats]rules.Action, after *rules.State) {
	point := before[0].Now
	if point.Phase == rules.PhaseSides {
		return
	}
	age, turn := point.Age-1, point.Turn-1
	for viewer := rules.Seat(0); viewer < rules.Seat(point.Seats); viewer++ {
		k := &memory[viewer]
		if point.Phase == rules.PhaseDiscardBuild {
			k.Rebuilt[age][turn][point.Decider] = picks[point.Decider].Design
			continue
		}
		for seat := rules.Seat(0); seat < rules.Seat(point.Seats); seat++ {
			a := picks[seat]
			if a.Kind == rules.ActNone {
				continue
			}
			play := rules.Play{Kind: a.Kind}
			if a.Kind == rules.ActBuild {
				play.Design = a.Design
			}
			if a.Kind == rules.ActBuild || a.Kind == rules.ActStage {
				play.Left, play.Right = a.Left, a.Right
			}
			if a.Kind == rules.ActStage {
				play.Stage = point.Cities[seat].Stages + 1
			}
			k.Plays[age][turn][seat] = play
			if seat == viewer {
				k.Seen[age][turn] = before[viewer].Now.Hand
				if point.Phase == rules.PhaseLastCard {
					k.Seen[age][turn] = [rules.HandSize]catalog.DesignID{before[viewer].Now.Kept}
				}
				k.Picks[age][turn] = a.Design
			}
		}
		if picks[viewer].Kind != rules.ActStage {
			continue
		}
		city := after.Cities[viewer]
		board, _ := catalog.BoardByID(city.Board)
		for _, effect := range board.Sides[city.Side-1].Stages[city.Stages-1].Effects {
			if effect.Kind != catalog.EffBuildFromDiscard {
				continue
			}
			for i := range k.PileLooks {
				if k.PileLooks[i].Age == 0 {
					k.PileLooks[i] = rules.PileLook{Age: point.Age, Turn: point.Turn, Count: after.Discard}
					break
				}
			}
		}
	}
}
