package bot

import (
	"context"
	"errors"
	"os"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// These probes use checked development histories only. A seat is compared
// with its own observation stripped of pre-root memory; no opponent notebook
// is supplied to ChooseAge or to a runtime policy.
func TestScholarApproximationMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_MEASURE") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_MEASURE=1")
	}
	for n := uint8(3); n <= 7; n++ {
		observations, policyCalls, actionChanges, valueChanges := 0, 0, 0, 0
		pairs, memoryRankChanges, scoreRankChanges, finalRankChanges := 0, 0, 0, 0
		var maxValueDifference int64
		for seed := uint64(0); seed < 8; seed++ {
			history := ageHistory(t, rules.Setup{Seats: n, Seed: 0x73636870726f6265 + seed, Options: rules.Options{Sides: rules.SideRule(seed % 3)}}, seed+0x2026, true)
			terminal := history[len(history)-1]
			for _, g := range history {
				if g.State.Phase != rules.PhaseTurn || g.State.Turn < 3 && (g.State.Age <= 1 || g.State.Turn != 1) {
					continue
				}
				var full, empty, exact, final [rules.MaxSeats]int64
				var coins, finalCoins [rules.MaxSeats]int16
				for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
					o := g.Observe(seat)
					stripped := rules.Observation{Now: o.Now}
					value, coin, ok := cutoffValue(o)
					other, _, otherOK := cutoffValue(stripped)
					facts, factsOK := o.Now.Facts(seat)
					if !ok || !otherOK || !factsOK {
						t.Fatal("invalid checked probe")
					}
					full[seat], empty[seat], coins[seat] = value, other, coin
					exact[seat] = int64(facts.Score.Total) * 1000
					final[seat] = int64(terminal.State.Result.Scores[seat].Total) * 1000
					finalCoins[seat] = terminal.State.Cities[seat].Coins
					observations++
					difference := value - other
					if difference != 0 {
						valueChanges++
					}
					if difference < 0 {
						difference = -difference
					}
					maxValueDifference = max(maxValueDifference, difference)
					cs := g.Legal(seat)
					for pref := balancedPreference; pref < numRolloutPreferences; pref++ {
						entropy := seed*10000 + uint64(g.State.Points)*8 + uint64(seat)
						a, err := rolloutPolicy(context.Background(), o, cs, pref, entropy)
						b, otherErr := rolloutPolicy(context.Background(), stripped, cs, pref, entropy)
						if err != nil || otherErr != nil {
							t.Fatal(err, otherErr)
						}
						policyCalls++
						if a != b {
							actionChanges++
						}
					}
				}
				compare := func(a, b int64, ac, bc int16) int {
					if a < b || a == b && ac < bc {
						return -1
					}
					if a > b || a == b && ac > bc {
						return 1
					}
					return 0
				}
				for a := 0; a < int(n); a++ {
					for b := a + 1; b < int(n); b++ {
						pairs++
						ordering := compare(full[a], full[b], coins[a], coins[b])
						if ordering != compare(empty[a], empty[b], coins[a], coins[b]) {
							memoryRankChanges++
						}
						if ordering != compare(exact[a], exact[b], coins[a], coins[b]) {
							scoreRankChanges++
						}
						if ordering != compare(final[a], final[b], finalCoins[a], finalCoins[b]) {
							finalRankChanges++
						}
					}
				}
			}
		}
		t.Logf("approximation n%d histories8 observations%d policy_pairs%d changed_actions%d changed_cutoffs%d max_abs_cutoff_milli_vp%d seat_pairs%d memory_order_changes%d exact_current_order_changes%d eventual_final_order_changes%d", n, observations, policyCalls, actionChanges, valueChanges, maxValueDifference, pairs, memoryRankChanges, scoreRankChanges, finalRankChanges)
	}
}

func scholarLaterPositions(t testing.TB) []rules.Game {
	t.Helper()
	var positions []rules.Game
	for n := uint8(3); n <= 7; n++ {
		history := ageHistory(t, rules.Setup{Seats: n, Seed: 0x73636870726f6265, Options: rules.Options{Sides: rules.SidesChoose}}, 0x2026, true)
		for _, g := range history {
			if g.State.Phase == rules.PhaseTurn && g.State.Turn == 3 {
				positions = append(positions, g)
			}
		}
	}
	if len(positions) != 15 {
		t.Fatal("missing later development positions", len(positions))
	}
	return positions
}
func TestScholarLaterNoiseMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_MEASURE") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_MEASURE=1")
	}
	var totalPaired, totalIndependent int64
	for _, g := range scholarLaterPositions(t) {
		out, err := ChooseAge(context.Background(), g.Observe(0), g.Legal(0), 0x1234, AgeWork{Attempts: 64, Candidates: 8, SamplerChecks: rules.MaxSamplerChecks})
		if err != nil || out.Fallback || out.Counters.CompletedSweeps != 64 {
			t.Fatal(out.Counters, err)
		}
		var reference AgeCandidate
		for _, c := range out.Ranked[:out.Count] {
			if c.Key == out.ReferenceKey {
				reference = c
			}
		}
		var paired, independent int64
		for _, c := range out.Ranked[:out.Count] {
			if c.Key != out.ReferenceKey {
				paired += 64*c.DeltaSquares - c.DeltaSum*c.DeltaSum
				independent += 64*c.SumSquares - c.Sum*c.Sum + 64*reference.SumSquares - reference.Sum*reference.Sum
			}
		}
		totalPaired += paired
		totalIndependent += independent
		t.Logf("later_noise n%d age%d turn3 samples64 candidates%d paired%d independent%d ratio=%.5f", g.State.Seats, g.State.Age, out.Count, paired, independent, float64(paired)/float64(independent))
	}
	t.Logf("later_noise aggregate_ratio=%.5f paired%d independent%d", float64(totalPaired)/float64(totalIndependent), totalPaired, totalIndependent)
}
func scholarCPULoad(n int) func() {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var sink atomic.Uint64
	for worker := 0; worker < n; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := uint64(1)
			for {
				select {
				case <-stop:
					sink.Add(x)
					return
				default:
					for k := 0; k < 1000; k++ {
						x = x*6364136223846793005 + 1
					}
				}
			}
		}()
	}
	return func() { close(stop); wg.Wait() }
}
func TestScholarCancellationMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_MEASURE") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_MEASURE=1")
	}
	positions := scholarLaterPositions(t)
	positions = append(positions, ageTail(t))
	quantile := func(ds []time.Duration, p int) float64 {
		slices.Sort(ds)
		return float64(ds[(len(ds)*p+99)/100-1]) / 1e6
	}
	for _, load := range []int{0, 1} {
		func() {
			stop := scholarCPULoad(load)
			defer stop()
			var gaps, latencies []time.Duration
			early := 0
			for _, g := range positions {
				seat := rules.Seat(0)
				for !g.Point().Required.Has(seat) {
					seat++
				}
				o, cs := g.Observe(seat), g.Legal(seat)
				for tape := uint64(0); tape < 8; tape++ {
					clock := &ageGapContext{Context: context.Background(), last: time.Now()}
					out, err := ChooseAge(clock, o, cs, tape, AgeWork{Attempts: 2, Candidates: 8, SamplerChecks: rules.MaxSamplerChecks})
					clock.finish()
					if err != nil || out.Fallback {
						t.Fatal(err, out.Counters)
					}
					gaps = append(gaps, clock.gap)
					ctx, cancel := context.WithCancel(context.Background())
					sent := make(chan time.Time, 1)
					go func() { time.Sleep(time.Millisecond); at := time.Now(); cancel(); sent <- at }()
					_, err = ChooseAge(ctx, o, cs, tape, AgeWork{Attempts: 2, Candidates: 8, SamplerChecks: rules.MaxSamplerChecks})
					returned := time.Now()
					at := <-sent
					cancel()
					switch {
					case errors.Is(err, context.Canceled):
						latencies = append(latencies, returned.Sub(at))
					case err == nil:
						early++
					default:
						t.Fatal(err)
					}
				}
			}
			if len(latencies) == 0 {
				t.Fatal("no cancelled calls")
			}
			t.Logf("cancel GOMAXPROCS%d load_workers%d complete_calls%d poll_gap_p50_ms=%.3f poll_gap_p95_ms=%.3f poll_gap_p99_ms=%.3f poll_gap_max_ms=%.3f cancelled_calls%d timer_cancel_to_return_p50_ms=%.3f timer_cancel_to_return_p95_ms=%.3f timer_cancel_to_return_p99_ms=%.3f timer_cancel_to_return_max_ms=%.3f completed_before_timer%d", runtime.GOMAXPROCS(0), load, len(gaps), quantile(gaps, 50), quantile(gaps, 95), quantile(gaps, 99), quantile(gaps, 100), len(latencies), quantile(latencies, 50), quantile(latencies, 95), quantile(latencies, 99), quantile(latencies, 100), early)
			if quantile(gaps, 99) >= 50 {
				t.Errorf("whole-call poll-gap p99 exceeds 50 ms under load_workers%d", load)
			}
		}()
	}
}

// Cancellation inside a structural quote set must discard its unfinished
// ranking, rather than wait for the rest of the candidate's quotes.
type scholarQuoteStop struct {
	context.Context
	calls int
}

func (c *scholarQuoteStop) Err() error {
	c.calls++
	if c.calls >= 3 {
		return context.Canceled
	}
	return nil
}
func TestScholarQuoteCancellation(t *testing.T) {
	g := ageGame(t, 7, 814)
	o := g.Observe(0)
	var counts Counters
	e, err := prepare(context.Background(), o, &counts)
	if err != nil {
		t.Fatal(err)
	}
	counts = Counters{}
	ctx := &scholarQuoteStop{Context: context.Background()}
	_, err = e.snapshot(ctx, o.Now, e.own, &counts)
	if !errors.Is(err, context.Canceled) || counts.Quotes != 1 {
		t.Fatal("did not stop between quotes", counts, err)
	}
}
