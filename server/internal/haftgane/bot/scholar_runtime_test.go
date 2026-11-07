package bot

import (
	"context"
	"os"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Runtime controls are diagnostic only. Neither a GC-off result nor the
// synthetic floor replaces the normal-GC, unchanged cancellation gate.
func TestScholarGCFreeDiagnostic(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_RUNTIME") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_RUNTIME=1")
	}
	if os.Getenv("GOGC") != "off" {
		t.Fatal("diagnostic requires separate GOGC=off process")
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	defer func() {
		runtime.ReadMemStats(&after)
		t.Logf("gc_free_diagnostic gc_before%d gc_after%d allocated_bytes%d heap_after_bytes%d", before.NumGC, after.NumGC, after.TotalAlloc-before.TotalAlloc, after.HeapAlloc)
	}()
	TestScholarCancellationMeasurements(t)
}

var scholarFloorSink atomic.Uint64

func scholarFloorLoop(clock *ageGapContext, chunks int, seed uint64) uint64 {
	for chunk := 0; chunk < chunks; chunk++ {
		_ = clock.Err()
		for i := 0; i < 256; i++ {
			seed = seed*6364136223846793005 + 1
		}
	}
	return seed
}
func TestScholarSchedulerFloorDiagnostic(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_RUNTIME") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_RUNTIME=1")
	}
	var target []time.Duration
	for _, g := range append(scholarLaterPositions(t), ageTail(t)) {
		seat := rules.Seat(0)
		for !g.Point().Required.Has(seat) {
			seat++
		}
		o, cs := g.Observe(seat), g.Legal(seat)
		for tape := uint64(0); tape < 8; tape++ {
			start := time.Now()
			out, err := ChooseAge(context.Background(), o, cs, tape, AgeWork{Attempts: 2, Candidates: 8, SamplerChecks: rules.MaxSamplerChecks})
			target = append(target, time.Since(start))
			if err != nil || out.Fallback {
				t.Fatal(err, out.Counters)
			}
		}
	}
	// Calibrate fixed loop counts before introducing load. Clocks here select
	// only a synthetic control duration, never Scholar's decision work.
	const unitChunks = 4096
	unitClock := &ageGapContext{Context: context.Background(), last: time.Now()}
	start := time.Now()
	scholarFloorSink.Store(scholarFloorLoop(unitClock, unitChunks, 1))
	unit := time.Since(start)
	chunks := make([]int, len(target))
	var targetTotal time.Duration
	for i, d := range target {
		chunks[i] = max(1, int(int64(d)*unitChunks/int64(unit)))
		targetTotal += d
	}
	gaps := make([]time.Duration, 0, len(target))
	runtime.GC()
	stop := scholarCPULoad(1)
	defer stop()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start = time.Now()
	for i, count := range chunks {
		clock := &ageGapContext{Context: context.Background(), last: time.Now()}
		value := scholarFloorLoop(clock, count, uint64(i)+1)
		clock.finish()
		scholarFloorSink.Store(value)
		gaps = append(gaps, clock.gap)
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	slices.Sort(gaps)
	q := func(p int) float64 { return float64(gaps[(len(gaps)*p+99)/100-1]) / 1e6 }
	t.Logf("scheduler_floor GOMAXPROCS%d load_workers1 calls%d target_idle_total_ms=%.3f loaded_total_ms=%.3f calibrated_poll_interval_ns=%.1f gap_p50_ms=%.3f gap_p95_ms=%.3f gap_p99_ms=%.3f gap_max_ms=%.3f gc_before%d gc_after%d allocated_bytes%d", runtime.GOMAXPROCS(0), len(target), float64(targetTotal)/1e6, float64(elapsed)/1e6, float64(unit)/unitChunks, q(50), q(95), q(99), q(100), before.NumGC, after.NumGC, after.TotalAlloc-before.TotalAlloc)
}
