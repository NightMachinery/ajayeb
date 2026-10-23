package main

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

type haReward struct{ Rank, Win, Midrank float64 }

func haScoreOutcome(scores, coins []int16, focal int) haReward {
	below, tied, above := 0, 0, 0
	for i, score := range scores {
		if i == focal {
			continue
		}
		switch {
		case score < scores[focal] || score == scores[focal] && coins[i] < coins[focal]:
			below++
		case score == scores[focal] && coins[i] == coins[focal]:
			tied++
		default:
			above++
		}
	}
	win := 0.0
	if above == 0 {
		win = 1 / float64(tied+1)
	}
	return haReward{(float64(below) + float64(tied)/2) / float64(len(scores)-1), win, 1 + float64(above) + float64(tied)/2}
}

type haCI struct {
	Mean float64 `json:"mean"`
	Low  float64 `json:"low_95"`
	High float64 `json:"high_95"`
}

func haInterval(blocks []float64) haCI {
	if len(blocks) == 0 {
		return haCI{}
	}
	mean := 0.0
	for _, v := range blocks {
		mean += v / float64(len(blocks))
	}
	r := rng.NewRNG(haBootstrapBase)
	samples := make([]float64, 5000)
	for i := range samples {
		for range blocks {
			samples[i] += blocks[r.IntN(len(blocks))] / float64(len(blocks))
		}
	}
	sort.Float64s(samples)
	return haCI{mean, samples[125], samples[4874]}
}

type haBlockMetric struct {
	Index                                                int
	Comparisons                                          int
	Rank, Win, Midrank, DeltaRank, DeltaWin, DeltaPoints float64
}
type haContrast struct {
	N                 int    `json:"n"`
	Name              string `json:"name"`
	IndependentBlocks int    `json:"independent_blocks"`
	Comparisons       int    `json:"paired_comparisons"`
	Rank              haCI   `json:"rank_reward"`
	Win               haCI   `json:"win_credit"`
	Midrank           haCI   `json:"midrank"`
	DeltaRank         haCI   `json:"paired_rank_improvement"`
	DeltaWin          haCI   `json:"paired_win_improvement"`
	DeltaPoints       haCI   `json:"paired_point_improvement"`
}
type haGate struct {
	N         int     `json:"n"`
	Name      string  `json:"name"`
	Metric    haCI    `json:"metric"`
	Target    float64 `json:"target"`
	Pass      bool    `json:"pass"`
	Uncertain bool    `json:"interval_crosses_target"`
}
type haRawAggregate struct {
	Count, RankUnits, WinUnits, MidrankUnits int64
	Points, Coins, Sales, Decisions          int64
	Categories                               [8]int64
	Work                                     [5]int64
	Fallbacks                                int64
	ScoreHistogram                           map[int16]int
	CoinHistogram                            map[int16]int
}
type haBreakdown struct {
	Key             string        `json:"key"`
	Observations    int64         `json:"seat_games"`
	RankReward      float64       `json:"rank_reward"`
	WinCredit       float64       `json:"win_credit"`
	Midrank         float64       `json:"midrank"`
	MeanScore       float64       `json:"mean_score"`
	MeanCoins       float64       `json:"mean_coins"`
	ScoreCategories [8]float64    `json:"mean_categories_wonder_treasure_military_blue_yellow_green_purple_total"`
	Sales           int64         `json:"sales"`
	Decisions       int64         `json:"decisions"`
	Work            [5]int64      `json:"work_iterations_ranked_previews_quotes_draws"`
	Fallbacks       int64         `json:"fallbacks"`
	ScoreHistogram  map[int16]int `json:"score_histogram"`
	CoinHistogram   map[int16]int `json:"coin_histogram"`
}
type haRawTime struct {
	Count      int64
	Total, Max int64
	Histogram  map[int64]int64
}
type haTiming struct {
	Key     string `json:"key"`
	Count   int64  `json:"decisions"`
	TotalNS int64  `json:"active_ns"`
	MaxNS   int64  `json:"max_ns"`
	P50US   int64  `json:"p50_us"`
	P95US   int64  `json:"p95_us"`
	P99US   int64  `json:"p99_us"`
}
type haSummary struct {
	Header          haHeader      `json:"header"`
	Complete        bool          `json:"complete"`
	Blocks          int           `json:"blocks"`
	Games           int           `json:"unique_games"`
	Picks           int           `json:"accepted_picks"`
	PeakHeapBytes   uint64        `json:"peak_heap_bytes_sampled_after_block"`
	ReleaseEligible bool          `json:"release_eligible"`
	GatesPassed     bool          `json:"release_gates_passed"`
	SelfplaySanity  bool          `json:"selfplay_sanity"`
	Contrasts       []haContrast  `json:"contrasts"`
	Gates           []haGate      `json:"gates"`
	Breakdowns      []haBreakdown `json:"breakdowns"`
	Timings         []haTiming    `json:"timings_one_microsecond_bins"`
}
type haAccumulator struct {
	header               haHeader
	blocks, games, picks int
	peakHeap             uint64
	contrasts            map[string][]haBlockMetric
	aggregates           map[string]*haRawAggregate
	timings              map[string]*haRawTime
}

func haNewAccumulator(h haHeader) *haAccumulator {
	return &haAccumulator{header: h, contrasts: map[string][]haBlockMetric{}, aggregates: map[string]*haRawAggregate{}, timings: map[string]*haRawTime{}}
}
func haGameReward(g *haGame, n, focal int) haReward {
	scores := make([]int16, n)
	for s := 0; s < n; s++ {
		scores[s] = g.Result.Scores[s].Total
	}
	return haScoreOutcome(scores, g.Coins[:n], focal)
}
func (a *haAccumulator) add(b haBlock) {
	a.blocks++
	a.games += len(b.Games)
	games := map[string]*haGame{}
	for i := range b.Games {
		g := &b.Games[i]
		games[g.ID] = g
		a.picks += len(g.Diagnostics)
		var sales [rules.MaxSeats]int64
		for _, batch := range g.Batches {
			for _, pick := range batch.Picks {
				action, _ := rules.ParseKey(pick.Key)
				if action.Kind == rules.ActSell {
					sales[pick.Seat]++
				}
			}
		}
		var work [rules.MaxSeats][5]int64
		var decisions, fallbacks [rules.MaxSeats]int64
		for _, d := range g.Diagnostics {
			seat := int(d[0])
			decisions[seat]++
			for j := 0; j < 5; j++ {
				work[seat][j] += d[j+3]
			}
			if d[8] != 0 {
				fallbacks[seat]++
			}
			key := fmt.Sprintf("n=%d/policy=%s/point=%d", b.N, g.Lineup[seat], d[1])
			p := a.timings[key]
			if p == nil {
				p = &haRawTime{Histogram: map[int64]int64{}}
				a.timings[key] = p
			}
			p.Count++
			p.Total += d[2]
			p.Max = max(p.Max, d[2])
			p.Histogram[d[2]/1000]++
		}
		for seat := 0; seat < b.N; seat++ {
			reward := haGameReward(g, b.N, seat)
			keys := []string{
				fmt.Sprintf("n=%d/policy=%s", b.N, g.Lineup[seat]),
				fmt.Sprintf("n=%d/policy=%s/seat=%d/board=%s/side=%s/left=%s/right=%s", b.N, g.Lineup[seat], seat, b.Boards[seat], b.Sides[seat], g.Lineup[(seat+1)%b.N], g.Lineup[(seat+b.N-1)%b.N]),
			}
			score := g.Result.Scores[seat]
			for _, key := range keys {
				p := a.aggregates[key]
				if p == nil {
					p = &haRawAggregate{ScoreHistogram: map[int16]int{}, CoinHistogram: map[int16]int{}}
					a.aggregates[key] = p
				}
				p.Count++
				p.RankUnits += int64(math.Round(reward.Rank * float64(2*(b.N-1))))
				p.WinUnits += int64(math.Round(reward.Win * 420))
				p.MidrankUnits += int64(reward.Midrank * 2)
				p.Points += int64(score.Total)
				p.Coins += int64(g.Coins[seat])
				p.Sales += sales[seat]
				p.Decisions += decisions[seat]
				p.Fallbacks += fallbacks[seat]
				for j, v := range [8]int16{score.Wonder, score.Treasure, score.Military, score.Blue, score.Yellow, score.Green, score.Purple, score.Total} {
					p.Categories[j] += int64(v)
				}
				for j, v := range work[seat] {
					p.Work[j] += v
				}
				p.ScoreHistogram[score.Total]++
				p.CoinHistogram[g.Coins[seat]]++
			}
		}
	}
	blockMeans := map[string]*haBlockMetric{}
	for _, c := range b.Comparisons {
		candidate := games[haLineupKey(c.Candidate, b.N)]
		baseline := games[haLineupKey(c.Baseline, b.N)]
		cr, br := haGameReward(candidate, b.N, c.Focal), haGameReward(baseline, b.N, c.Focal)
		p := blockMeans[c.Name]
		if p == nil {
			p = &haBlockMetric{Index: b.Index}
			blockMeans[c.Name] = p
		}
		p.Comparisons++
		p.Rank += cr.Rank
		p.Win += cr.Win
		p.Midrank += cr.Midrank
		p.DeltaRank += cr.Rank - br.Rank
		p.DeltaWin += cr.Win - br.Win
		p.DeltaPoints += float64(candidate.Result.Scores[c.Focal].Total - baseline.Result.Scores[c.Focal].Total)
	}
	for name, p := range blockMeans {
		n := float64(p.Comparisons)
		p.Rank /= n
		p.Win /= n
		p.Midrank /= n
		p.DeltaRank /= n
		p.DeltaWin /= n
		p.DeltaPoints /= n
		key := fmt.Sprintf("%d/%s", b.N, name)
		a.contrasts[key] = append(a.contrasts[key], *p)
	}
}
func (a *haAccumulator) finish() haSummary {
	h := a.header
	out := haSummary{Header: h, Complete: true, Blocks: a.blocks, Games: a.games, Picks: a.picks, PeakHeapBytes: a.peakHeap, SelfplaySanity: true, Contrasts: []haContrast{}, Gates: []haGate{}, Breakdowns: []haBreakdown{}, Timings: []haTiming{}}
	out.ReleaseEligible = h.Partition == "release" && h.Blocks >= 100 && reflectSeatsAll(h.Seats) && (h.Suite == "release" || h.Suite == "all")
	keys := make([]string, 0, len(a.contrasts))
	for key := range a.contrasts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rows := a.contrasts[key]
		sort.Slice(rows, func(i, j int) bool { return rows[i].Index < rows[j].Index })
		var n int
		var name string
		_, _ = fmt.Sscanf(key, "%d/%s", &n, &name)
		c := haContrast{N: n, Name: name, IndependentBlocks: len(rows)}
		var rank, win, mid, dr, dw, dp []float64
		for _, p := range rows {
			c.Comparisons += p.Comparisons
			rank = append(rank, p.Rank)
			win = append(win, p.Win)
			mid = append(mid, p.Midrank)
			dr = append(dr, p.DeltaRank)
			dw = append(dw, p.DeltaWin)
			dp = append(dp, p.DeltaPoints)
		}
		c.Rank = haInterval(rank)
		c.Win = haInterval(win)
		c.Midrank = haInterval(mid)
		c.DeltaRank = haInterval(dr)
		c.DeltaWin = haInterval(dw)
		c.DeltaPoints = haInterval(dp)
		out.Contrasts = append(out.Contrasts, c)
		gate := func(metric string, ci haCI, target float64, strict bool) {
			pass := ci.Mean >= target
			if strict {
				pass = ci.Low > target
			}
			out.Gates = append(out.Gates, haGate{n, name + "/" + metric, ci, target, pass, ci.Low <= target && ci.High >= target})
		}
		switch name {
		case "greedy-random":
			gate("rank", c.Rank, .85, false)
			gate("win", c.Win, .70, false)
		case "apprentice-random":
			gate("rank", c.Rank, .75, false)
			gate("win", c.Win, .50, false)
		case "mixed-greedy-apprentice":
			gate("paired-rank", c.DeltaRank, 0, true)
		}
		if strings.HasPrefix(name, "selfplay-") && (math.Abs(c.Midrank.Mean-float64(n+1)/2) > 1e-12 || math.Abs(c.Rank.Mean-.5) > 1e-12) {
			out.SelfplaySanity = false
		}
	}
	out.GatesPassed = out.ReleaseEligible && len(out.Gates) == 25
	for _, g := range out.Gates {
		out.GatesPassed = out.GatesPassed && g.Pass
	}
	keys = keys[:0]
	for key := range a.aggregates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		p := a.aggregates[key]
		var n int
		_, _ = fmt.Sscanf(key, "n=%d/", &n)
		count := float64(p.Count)
		row := haBreakdown{Key: key, Observations: p.Count, RankReward: float64(p.RankUnits) / float64(2*(n-1)) / count, WinCredit: float64(p.WinUnits) / 420 / count, Midrank: float64(p.MidrankUnits) / 2 / count, MeanScore: float64(p.Points) / count, MeanCoins: float64(p.Coins) / count, Sales: p.Sales, Decisions: p.Decisions, Work: p.Work, Fallbacks: p.Fallbacks, ScoreHistogram: p.ScoreHistogram, CoinHistogram: p.CoinHistogram}
		for j, v := range p.Categories {
			row.ScoreCategories[j] = float64(v) / count
		}
		out.Breakdowns = append(out.Breakdowns, row)
	}
	keys = keys[:0]
	for key := range a.timings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		p := a.timings[key]
		bins := make([]int64, 0, len(p.Histogram))
		for bin := range p.Histogram {
			bins = append(bins, bin)
		}
		sort.Slice(bins, func(i, j int) bool { return bins[i] < bins[j] })
		quantile := func(percent int64) int64 {
			want := (p.Count*percent + 99) / 100
			seen := int64(0)
			for _, bin := range bins {
				seen += p.Histogram[bin]
				if seen >= want {
					return bin
				}
			}
			return 0
		}
		out.Timings = append(out.Timings, haTiming{key, p.Count, p.Total, p.Max, quantile(50), quantile(95), quantile(99)})
	}
	return out
}
func reflectSeatsAll(ns []int) bool {
	return len(ns) == 5 && ns[0] == 3 && ns[1] == 4 && ns[2] == 5 && ns[3] == 6 && ns[4] == 7
}
