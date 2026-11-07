package bot

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

type ageGapContext struct {
	context.Context
	last time.Time
	gap  time.Duration
}

func (c *ageGapContext) Err() error {
	now := time.Now()
	c.gap = max(c.gap, now.Sub(c.last))
	c.last = now
	return c.Context.Err()
}
func (c *ageGapContext) finish() { c.gap = max(c.gap, time.Since(c.last)) }

func TestAgeMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_ROLLOUT_MEASURE") != "1" {
		t.Skip("set HAFTGANE_ROLLOUT_MEASURE=1")
	}
	tail := ageTail(t)
	t.Log("saved tail decision point", tail.Point(), "required mask", tail.Point().Required)
	for _, load := range []int{0, 2} {
		var gaps []time.Duration
		stop := make(chan struct{})
		var wg sync.WaitGroup
		var sink atomic.Uint64
		for worker := 0; worker < load; worker++ {
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
		func() {
			defer func() { close(stop); wg.Wait() }()
			measure := func(g rules.Game, seat rules.Seat, cap uint16, tape uint64) (time.Duration, AgeCounters) {
				ctx := &ageGapContext{Context: context.Background(), last: time.Now()}
				start := ctx.last
				out, err := ChooseAge(ctx, g.Observe(seat), g.Legal(seat), tape, AgeWork{Attempts: 2, Candidates: cap, SamplerChecks: rules.MaxSamplerChecks})
				ctx.finish()
				elapsed := time.Since(start)
				if err != nil || out.Fallback || out.Counters.CompletedSweeps != 2 {
					t.Fatal("measured decision", out.Counters, err)
				}
				gaps = append(gaps, ctx.gap)
				return elapsed, out.Counters
			}
			for _, cap := range []uint16{8, 16, rules.MaxTurnChoices} {
				for seat := rules.Seat(0); seat < 7; seat++ {
					if !tail.Point().Required.Has(seat) {
						continue
					}
					var elapsed []time.Duration
					var total time.Duration
					var counters AgeCounters
					for tape := uint64(0); tape < 8; tape++ {
						d, c := measure(tail, seat, cap, tape)
						elapsed = append(elapsed, d)
						total += d
						counters = c
					}
					slices.Sort(elapsed)
					t.Logf("tail load%d cap%d seat%d attempts2 calls8 mean_ms=%.3f max_ms=%.3f last_counters=%+v", load, cap, seat, float64(total)/8e6, float64(elapsed[7])/1e6, counters)
				}
			}
			for n := uint8(3); n <= 7; n++ {
				for seed := 0; seed < ageSweepGames(t); seed++ {
					history := ageHistory(t, rules.Setup{Seats: n, Seed: uint64(seed) + 0x71ac}, uint64(seed)+781, true)
					for _, g := range history {
						for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
							if g.Point().Required.Has(seat) {
								measure(g, seat, 8, uint64(seed))
								break
							}
						}
					}
				}
			}
		}()
		slices.Sort(gaps)
		p99 := gaps[(len(gaps)*99+99)/100-1]
		t.Logf("whole_decision_poll_gap load%d GOMAXPROCS%d calls%d p50_ms=%.3f p99_ms=%.3f max_ms=%.3f", load, runtime.GOMAXPROCS(0), len(gaps), float64(gaps[len(gaps)/2])/1e6, float64(p99)/1e6, float64(gaps[len(gaps)-1])/1e6)
		if p99 >= 50*time.Millisecond {
			t.Fatal("rollout poll-gap p99 above target", load, p99)
		}
	}
	t.Logf("value_bytes state=%d knowledge=%d current=%d choice=%d ranking=%d candidate=%d decision=%d", unsafe.Sizeof(rules.State{}), unsafe.Sizeof(rules.Knowledge{}), unsafe.Sizeof(rules.Current{}), unsafe.Sizeof(rules.Choice{}), unsafe.Sizeof(Ranked{}), unsafe.Sizeof(AgeCandidate{}), unsafe.Sizeof(AgeDecision{}))
}

func TestAgePairedNoiseMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_ROLLOUT_MEASURE") != "1" {
		t.Skip("set HAFTGANE_ROLLOUT_MEASURE=1")
	}
	var allPaired, allIndependent int64
	for n := uint8(3); n <= 7; n++ {
		g := ageGame(t, n, 0x8171)
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
			if c.Key == reference.Key {
				continue
			}
			// N^2 times empirical variance. Independent draws retain these
			// marginals but have zero cross-candidate covariance.
			paired += 64*c.DeltaSquares - c.DeltaSum*c.DeltaSum
			independent += 64*c.SumSquares - c.Sum*c.Sum + 64*reference.SumSquares - reference.Sum*reference.Sum
		}
		allPaired += paired
		allIndependent += independent
		t.Logf("paired_noise n%d samples64 candidates%d variance_numerator paired%d independent%d ratio=%.4f", n, out.Count, paired, independent, float64(paired)/float64(independent))
	}
	if allIndependent == 0 || allPaired >= allIndependent {
		t.Fatal("paired comparison did not reduce aggregate measured noise", allPaired, allIndependent)
	}
	t.Logf("aggregate paired_noise_ratio=%.4f", float64(allPaired)/float64(allIndependent))
}

func BenchmarkAgeDecision(b *testing.B) {
	for _, cap := range []uint16{8, 16, rules.MaxTurnChoices} {
		b.Run(fmt.Sprintf("cap%d", cap), func(b *testing.B) {
			g := ageGame(b, 7, 814)
			o, cs := g.Observe(0), g.Legal(0)
			actual := 0
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, err := ChooseAge(context.Background(), o, cs, uint64(i), AgeWork{Attempts: 2, Candidates: cap, SamplerChecks: rules.MaxSamplerChecks})
				if err != nil || out.Fallback {
					b.Fatal(err)
				}
				actual = int(out.Count)
			}
			b.ReportMetric(float64(actual), "candidates/op")
		})
	}
}

// Storage sampling is separate from latency: ReadMemStats adds overhead to
// polls. This is a sampled Go-heap high-water mark, not process peak RSS.
type ageStorageContext struct {
	context.Context
	polls int
	peak  uint64
}

func (c *ageStorageContext) Err() error {
	c.polls++
	if c.polls%128 == 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c.peak = max(c.peak, m.HeapAlloc)
	}
	return c.Context.Err()
}
func TestAgeStorageMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_ROLLOUT_MEASURE") != "1" {
		t.Skip("set HAFTGANE_ROLLOUT_MEASURE=1")
	}
	g := ageGame(t, 7, 814)
	o, cs := g.Observe(0), g.Legal(0)
	for _, cap := range []uint16{8, 16, rules.MaxTurnChoices} {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		ctx := &ageStorageContext{Context: context.Background(), peak: before.HeapAlloc}
		out, err := ChooseAge(ctx, o, cs, 51, AgeWork{Attempts: 2, Candidates: cap, SamplerChecks: rules.MaxSamplerChecks})
		runtime.ReadMemStats(&after)
		ctx.peak = max(ctx.peak, after.HeapAlloc)
		if err != nil || out.Fallback {
			t.Fatal(err)
		}
		t.Logf("storage cap%d candidates%d attempts2 polls%d baseline_heap_bytes%d sampled_peak_heap_bytes%d sampled_peak_increase_bytes%d allocated_bytes%d allocations%d", cap, out.Count, ctx.polls, before.HeapAlloc, ctx.peak, ctx.peak-before.HeapAlloc, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs)
	}
}
