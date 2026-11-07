package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	hbot "github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
)

type scPlannedN struct {
	N            int `json:"n"`
	Comparisons  int `json:"comparisons_per_block"`
	Games        int `json:"unique_games_per_block"`
	ScholarSeats int `json:"scholar_seat_games_per_block"`
}

func scPlanSummary(c scConfig) any {
	var current, full []scPlannedN
	for _, n := range c.Seats {
		for _, suite := range []string{c.Suite, "all"} {
			plan := scPlan(n, suite)
			row := scPlannedN{N: n, Comparisons: len(plan), Games: len(scLineups(plan))}
			for _, line := range scLineups(plan) {
				row.ScholarSeats += strings.Count(line, "S")
			}
			if len(current) == len(full) {
				current = append(current, row)
			} else {
				full = append(full, row)
			}
		}
	}
	return struct {
		Config   scConfig     `json:"config"`
		Assigned []scPlannedN `json:"assigned_per_n"`
		Full     []scPlannedN `json:"full_gate_manifest_per_block"`
	}{c, current, full}
}

type scContrast struct {
	Profile           string `json:"profile"`
	N                 int    `json:"n"`
	Family            string `json:"family"`
	Blocks            int    `json:"independent_blocks"`
	Rank              haCI   `json:"rank_reward"`
	Win               haCI   `json:"win_credit"`
	Delta             haCI   `json:"paired_rank_improvement"`
	IntervalEstimable bool   `json:"interval_estimable"`
}
type scTiming struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
	P50   int64  `json:"p50_ns"`
	P95   int64  `json:"p95_ns"`
	P99   int64  `json:"p99_ns"`
	Max   int64  `json:"max_ns"`
}

func scQuantiles(key string, ns []int64) scTiming {
	if len(ns) == 0 {
		return scTiming{Key: key}
	}
	slices.Sort(ns)
	q := func(p int) int64 { return ns[(len(ns)*p+99)/100-1] }
	return scTiming{key, len(ns), q(50), q(95), q(99), ns[len(ns)-1]}
}

type scEstimate struct {
	Profile      string  `json:"profile"`
	N            int     `json:"n"`
	Games        int     `json:"games_at_100_blocks"`
	Hours        float64 `json:"projected_hours"`
	Optimistic   float64 `json:"observed_rate_low_hours"`
	Conservative float64 `json:"observed_rate_high_hours"`
}
type scProfileEstimate struct {
	Profile        string  `json:"profile"`
	Hours          float64 `json:"projected_full_gate_hours"`
	Low            float64 `json:"observed_rate_low_hours"`
	High           float64 `json:"observed_rate_high_hours"`
	OverEightHours bool    `json:"over_eight_hour_limit"`
}
type scSummary struct {
	Config           scConfig                    `json:"config"`
	Complete         bool                        `json:"complete"`
	Blocks           int                         `json:"completed_blocks"`
	Games            int                         `json:"games"`
	Eligible         bool                        `json:"heldout_eligible"`
	GatesPassed      bool                        `json:"gates_passed"`
	Combined         []scContrast                `json:"combined_suite"`
	Contrasts        []scContrast                `json:"contrasts"`
	Timings          []scTiming                  `json:"timings"`
	Estimates        []scEstimate                `json:"per_n_runtime_projection"`
	ProfileEstimates []scProfileEstimate         `json:"full_gate_runtime_projection"`
	Counters         map[string]hbot.AgeCounters `json:"aggregate_counters"`
	Allocated        map[string]uint64           `json:"allocated_bytes_all_games"`
	Allocs           map[string]uint64           `json:"allocations_all_games"`
	Heap             map[string]uint64           `json:"sampled_heap_after_game_max_bytes"`
}
type scBlockMeans struct{ rank, win, delta []float64 }
type scRates struct {
	games                       int
	ns, scholarNS, scholarSeats int64
	low, high                   float64
}

func scInterval(x []float64) haCI {
	if len(x) < 2 {
		mean := 0.0
		if len(x) == 1 {
			mean = x[0]
		}
		return haCI{mean, 0, 1}
	}
	return haInterval(x)
}
func scAddCounters(a, b hbot.AgeCounters) hbot.AgeCounters {
	return hbot.AgeCounters{
		SampleAttempts: a.SampleAttempts + b.SampleAttempts, Worlds: a.Worlds + b.Worlds, SamplerChecks: a.SamplerChecks + b.SamplerChecks, NoWorldSamples: a.NoWorldSamples + b.NoWorldSamples, BudgetOutSamples: a.BudgetOutSamples + b.BudgetOutSamples,
		CompletedSweeps: a.CompletedSweeps + b.CompletedSweeps, Comparisons: a.Comparisons + b.Comparisons, Batches: a.Batches + b.Batches, PolicyChoices: a.PolicyChoices + b.PolicyChoices, GreedyRankings: a.GreedyRankings + b.GreedyRankings, GreedyFallbacks: a.GreedyFallbacks + b.GreedyFallbacks, Excluded: a.Excluded + b.Excluded,
	}
}
func scSummarize(dir string, c scConfig) (scSummary, error) {
	s := scSummary{Config: c, Complete: true, Contrasts: []scContrast{}, Timings: []scTiming{}, Estimates: []scEstimate{}, ProfileEstimates: []scProfileEstimate{}, Counters: map[string]hbot.AgeCounters{}, Allocated: map[string]uint64{}, Allocs: map[string]uint64{}, Heap: map[string]uint64{}}
	means := map[string]*scBlockMeans{}
	times := map[string][]int64{}
	rates := map[string]*scRates{}
	for _, p := range c.Profiles {
		for _, n := range c.Seats {
			for index := 0; index < c.Blocks; index++ {
				b, err := scReadBlock(scBlockPath(dir, p, n, index), c, p)
				if os.IsNotExist(err) {
					s.Complete = false
					continue
				}
				if err != nil {
					return s, err
				}
				s.Blocks++
				s.Games += len(b.Games)
				games := map[string]scGame{}
				key := fmt.Sprintf("%s/n%d", p.ID, n)
				rate := rates[key]
				if rate == nil {
					rate = &scRates{}
					rates[key] = rate
				}
				for _, g := range b.Games {
					games[g.Lineup] = g
					rate.games++
					rate.ns += g.WallNS
					scholarNS := int64(0)
					for _, pick := range g.Picks {
						if !pick.Scholar {
							continue
						}
						scholarNS += pick.NS
						s.Counters[key] = scAddCounters(s.Counters[key], pick.Counters)
						times[key+"/decision"] = append(times[key+"/decision"], pick.NS)
						times[key+"/poll-gap"] = append(times[key+"/poll-gap"], pick.GapNS)
						tag := fmt.Sprintf("%s/phase%d/decision", key, pick.Phase)
						times[tag] = append(times[tag], pick.NS)
					}
					roles := strings.Count(g.Lineup, "S")
					rate.scholarNS += scholarNS
					rate.scholarSeats += int64(roles)
					if roles > 0 {
						r := float64(scholarNS) / float64(roles)
						if rate.low == 0 || r < rate.low {
							rate.low = r
						}
						rate.high = max(rate.high, r)
					}
					if g.Lineup == "G"+strings.Repeat("S", n-1) {
						times[key+"/one-human-table/game"] = append(times[key+"/one-human-table/game"], g.WallNS)
						times[key+"/one-human-table/turn"] = append(times[key+"/one-human-table/turn"], g.Turns...)
						for _, pick := range g.Picks {
							if pick.Scholar {
								times[key+"/one-human-table/decision"] = append(times[key+"/one-human-table/decision"], pick.NS)
							}
						}
					}
					s.Allocated[key] += g.Allocated
					s.Allocs[key] += g.Allocs
					s.Heap[key] = max(s.Heap[key], g.HeapAfter)
				}
				for _, family := range []string{"main", "inverse", "mixed"} {
					var rank, win, delta float64
					count := 0
					for _, compare := range b.Comparisons {
						if compare.Family != family {
							continue
						}
						candidate, baseline := games[compare.Candidate], games[compare.Baseline]
						ca := haGame{Result: candidate.Result, Coins: candidate.Coins}
						ba := haGame{Result: baseline.Result, Coins: baseline.Coins}
						cr, br := haGameReward(&ca, n, compare.Focal), haGameReward(&ba, n, compare.Focal)
						rank += cr.Rank
						win += cr.Win
						delta += cr.Rank - br.Rank
						count++
					}
					mk := key + "/" + family
					m := means[mk]
					if m == nil {
						m = &scBlockMeans{}
						means[mk] = m
					}
					m.rank = append(m.rank, rank/float64(count))
					m.win = append(m.win, win/float64(count))
					m.delta = append(m.delta, delta/float64(count))
				}
			}
		}
	}
	var timeKeys []string
	for key := range times {
		timeKeys = append(timeKeys, key)
	}
	slices.Sort(timeKeys)
	for _, key := range timeKeys {
		s.Timings = append(s.Timings, scQuantiles(key, times[key]))
	}
	gates := true
	// The combined suite weights player counts equally, then families equally.
	// Keep all rotations/reflections within each independent deal block.
	for _, p := range c.Profiles {
		combined := scBlockMeans{}
		for index := 0; index < c.Blocks; index++ {
			rank, win, delta, cells := 0.0, 0.0, 0.0, 0
			for _, n := range c.Seats {
				for _, family := range []string{"main", "inverse", "mixed"} {
					m := means[fmt.Sprintf("%s/n%d/%s", p.ID, n, family)]
					if m != nil && len(m.rank) == c.Blocks {
						rank += m.rank[index]
						win += m.win[index]
						delta += m.delta[index]
						cells++
					}
				}
			}
			if cells == len(c.Seats)*3 {
				combined.rank = append(combined.rank, rank/float64(cells))
				combined.win = append(combined.win, win/float64(cells))
				combined.delta = append(combined.delta, delta/float64(cells))
			}
		}
		row := scContrast{p.ID, 0, "combined", len(combined.rank), scInterval(combined.rank), scInterval(combined.win), scInterval(combined.delta), len(combined.rank) >= 2}
		if !row.IntervalEstimable {
			row.Delta.Low = -1
		}
		s.Combined = append(s.Combined, row)
		gates = gates && row.IntervalEstimable && haThresholdPass(row.Delta, 0, true)
	}
	for _, p := range c.Profiles {
		pe := scProfileEstimate{Profile: p.ID}
		for _, n := range c.Seats {
			key := fmt.Sprintf("%s/n%d", p.ID, n)
			for _, family := range []string{"main", "inverse", "mixed"} {
				m := means[key+"/"+family]
				if m == nil {
					gates = false
					continue
				}
				row := scContrast{p.ID, n, family, len(m.rank), scInterval(m.rank), scInterval(m.win), scInterval(m.delta), len(m.rank) >= 2}
				if !row.IntervalEstimable {
					row.Delta.Low = -1
				}
				s.Contrasts = append(s.Contrasts, row)
				if !row.IntervalEstimable || family == "main" && (!haThresholdPass(row.Delta, 0, true) || !haThresholdPass(row.Rank, 0.65, false) || !haThresholdPass(row.Win, 1/float64(n)+0.10, false)) || family != "main" && !haThresholdPass(row.Delta, 0, false) {
					gates = false
				}
			}
			rate := rates[key]
			if rate == nil || rate.scholarSeats == 0 {
				continue
			}
			lines := scLineups(scPlan(n, "all"))
			roles := 0
			for _, line := range lines {
				roles += strings.Count(line, "S")
			}
			overhead := float64(rate.ns-rate.scholarNS) / float64(rate.games)
			mean := float64(rate.scholarNS) / float64(rate.scholarSeats)
			toHours := func(roleCost float64) float64 {
				return 100 * (float64(len(lines))*overhead + float64(roles)*roleCost) / 3.6e12
			}
			e := scEstimate{p.ID, n, 100 * len(lines), toHours(mean), toHours(rate.low), toHours(rate.high)}
			s.Estimates = append(s.Estimates, e)
			pe.Hours += e.Hours
			pe.Low += e.Optimistic
			pe.High += e.Conservative
		}
		pe.OverEightHours = pe.Hours > 8
		s.ProfileEstimates = append(s.ProfileEstimates, pe)
	}
	s.Eligible = s.Complete && c.Partition == "heldout" && c.Frozen && c.Blocks >= 100 && reflectAllCounts(c.Seats) && c.Suite == "all" && len(c.Profiles) == 1
	if c.Partition == "heldout" && !s.Complete {
		// Progress and cost may be inspected before the one final strength look.
		s.Contrasts, s.Combined = nil, nil
	}
	s.GatesPassed = s.Eligible && gates
	return s, nil
}
func reflectAllCounts(ns []int) bool { return slices.Equal(ns, []int{3, 4, 5, 6, 7}) }
