package rules

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
)

// These opt-in observations measure both expensive full calls and cancellation
// after entering search. Local load is explicit and all goroutines are joined.
func TestSamplerCancellationMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SAMPLER_MEASURE") != "1" {
		t.Skip("set HAFTGANE_SAMPLER_MEASURE=1")
	}
	type position struct {
		n           uint8
		seed, point int
		seat        Seat
		o           Observation
	}
	var positions []position
	for _, fixture := range []struct {
		n           uint8
		seed, point int
		seat        Seat
	}{{5, 172, 15, 0}, {6, 1434, 16, 0}, {7, 1218, 21, NoSeat}} {
		setup := Setup{Seats: fixture.n, Seed: uint64(fixture.seed) + 0x517, Options: Options{Sides: SideRule(fixture.seed % 3), HideAtEnd: fixture.seed%2 == 1}}
		run := samplerSweepPlay(t, setup, uint64(fixture.seed)+127, (fixture.seed/3)%3)
		found := false
		for _, g := range run.games {
			if int(g.State.Points) != fixture.point {
				continue
			}
			found = true
			for seat := Seat(0); seat < Seat(fixture.n); seat++ {
				if fixture.seat == NoSeat || fixture.seat == seat {
					positions = append(positions, position{fixture.n, fixture.seed, fixture.point, seat, g.Observe(seat)})
				}
			}
		}
		if !found {
			t.Fatal("missing measured fixture")
		}
	}
	for _, p := range positions {
		elapsed := make([]time.Duration, 64)
		worstChecks := 0
		for tape := uint64(0); tape < 64; tape++ {
			r := rng.NewRNG2(uint64(p.seed), uint64(p.point)*31+uint64(p.seat)+tape)
			start := time.Now()
			w, cost, ok := DeterminizeControlled(p.o, &r, MaxSamplerChecks, func() bool { return false })
			elapsed[tape] = time.Since(start)
			if !ok || !MatchesObservation(p.o, w) {
				t.Fatal("measured no-world", p.n, p.seat, cost)
			}
			worstChecks = max(worstChecks, cost.Checks)
		}
		slices.Sort(elapsed)
		var total time.Duration
		for _, d := range elapsed {
			total += d
		}
		t.Logf("full n%d seed_index%d point%d seat%d calls64 mean_ms=%.3f p50_ms=%.3f p95_ms=%.3f max_ms=%.3f max_checks=%d", p.n, p.seed, p.point, p.seat, float64(total)/64e6, float64(elapsed[31])/1e6, float64(elapsed[60])/1e6, float64(elapsed[63])/1e6, worstChecks)
	}
	for _, load := range []int{0, 2} {
		stopLoad := make(chan struct{})
		var wg sync.WaitGroup
		var sink atomic.Uint64
		for worker := 0; worker < load; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				x := uint64(1)
				for {
					select {
					case <-stopLoad:
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
		var latencies []time.Duration
		func() {
			defer func() { close(stopLoad); wg.Wait() }()
			for _, p := range positions {
				for tape := uint64(0); tape < 64; tape++ {
					ctx, cancel := context.WithCancel(context.Background())
					entered := make(chan struct{}, 1)
					cancelledAt := make(chan time.Time, 1)
					go func() { <-entered; start := time.Now(); cancel(); cancelledAt <- start }()
					polls := 0
					r := rng.NewRNG2(uint64(p.seed), tape)
					w, cost, ok := DeterminizeControlled(p.o, &r, MaxSamplerChecks, func() bool {
						polls++
						if polls == 4 {
							entered <- struct{}{}
							<-ctx.Done()
							// Model cancellation just after a poll: deliberately
							// run one whole chunk before the next poll sees it.
							return false
						}
						return ctx.Err() != nil
					})
					returned := time.Now()
					if polls < 4 {
						entered <- struct{}{}
						<-cancelledAt
						cancel()
						t.Fatal("measurement never entered sampling")
					}
					at := <-cancelledAt
					cancel()
					if ok || w != (World{}) || !cost.Stopped || cost.Checks < 32 {
						t.Fatal("cancellation was not inside sampling", cost)
					}
					latencies = append(latencies, returned.Sub(at))
				}
			}
		}()
		slices.Sort(latencies)
		p99 := latencies[(len(latencies)*99+99)/100-1]
		t.Logf("cancel load_workers%d GOMAXPROCS%d calls%d p50_ms=%.3f p95_ms=%.3f p99_ms=%.3f max_ms=%.3f", load, runtime.GOMAXPROCS(0), len(latencies), float64(latencies[len(latencies)/2])/1e6, float64(latencies[len(latencies)*95/100])/1e6, float64(p99)/1e6, float64(latencies[len(latencies)-1])/1e6)
		if p99 >= 50*time.Millisecond {
			t.Fatal("cancellation p99 above target", load, p99)
		}
	}
}

// Complete calls expose every polling window, including preparation and the
// materialization tail. Record the largest window per call rather than treating
// a cancellation injected at one early poll as the slowest window of that call.
func TestSamplerPollGapMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SAMPLER_MEASURE") != "1" {
		t.Skip("set HAFTGANE_SAMPLER_MEASURE=1")
	}
	games := 10
	if text := os.Getenv("HAFTGANE_SAMPLER_GAMES"); text != "" {
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > 100000 {
			t.Fatal("invalid HAFTGANE_SAMPLER_GAMES")
		}
		games = n
	}
	var gaps []time.Duration
	var worst, slowest time.Duration
	worstAt := ""
	measure := func(label string, o Observation, r *rng.RNG) {
		start := time.Now()
		last, largest := start, time.Duration(0)
		_, _, ok := DeterminizeControlled(o, r, MaxSamplerChecks, func() bool {
			now := time.Now()
			largest = max(largest, now.Sub(last))
			last = now
			return false
		})
		end := time.Now()
		largest = max(largest, end.Sub(last))
		if !ok {
			t.Fatal("poll-gap no-world", label)
		}
		gaps = append(gaps, largest)
		if largest > worst {
			worst, worstAt = largest, label
		}
		slowest = max(slowest, end.Sub(start))
	}
	for _, fixture := range []struct {
		n           uint8
		seed, point int
	}{{5, 172, 15}, {6, 1434, 16}, {7, 1218, 21}} {
		setup := Setup{Seats: fixture.n, Seed: uint64(fixture.seed) + 0x517, Options: Options{Sides: SideRule(fixture.seed % 3), HideAtEnd: fixture.seed%2 == 1}}
		run := samplerSweepPlay(t, setup, uint64(fixture.seed)+127, (fixture.seed/3)%3)
		found := false
		for _, g := range run.games {
			if int(g.State.Points) != fixture.point {
				continue
			}
			found = true
			for seat := Seat(0); seat < Seat(fixture.n); seat++ {
				for tape := uint64(0); tape < 64; tape++ {
					r := rng.NewRNG2(tape, uint64(seat)+99)
					measure(fmt.Sprintf("fixture n%d seed_index%d point%d seat%d tape%d", fixture.n, fixture.seed, fixture.point, seat, tape), g.Observe(seat), &r)
				}
			}
		}
		if !found {
			t.Fatal("missing poll-gap fixture")
		}
	}
	fixtureCalls := len(gaps)
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for seed := 0; seed < games; seed++ {
			setup := Setup{Seats: n, Seed: uint64(seed) + 0x3c3, Options: Options{Sides: SideRule(seed % 3)}}
			run := samplerSweepPlay(t, setup, uint64(seed)+77, (seed/3)%3)
			for _, g := range run.games {
				for seat := Seat(0); seat < Seat(n); seat++ {
					r := rng.NewRNG2(uint64(seed), uint64(g.State.Points)*8+uint64(seat))
					measure(fmt.Sprintf("sweep n%d seed_index%d point%d seat%d", n, seed, g.State.Points, seat), g.Observe(seat), &r)
				}
			}
		}
	}
	slices.Sort(gaps)
	percentile := func(perThousand int) time.Duration {
		return gaps[(len(gaps)*perThousand+999)/1000-1]
	}
	t.Logf("poll_gap GOMAXPROCS%d calls%d fixture_calls%d fresh_games_per_count%d p50_ms=%.3f p99_ms=%.3f p999_ms=%.3f max_ms=%.3f worst_at=%s slowest_call_ms=%.3f", runtime.GOMAXPROCS(0), len(gaps), fixtureCalls, games, float64(percentile(500))/1e6, float64(percentile(990))/1e6, float64(percentile(999))/1e6, float64(worst)/1e6, worstAt, float64(slowest)/1e6)
	if percentile(990) >= 50*time.Millisecond {
		t.Fatal("whole-call poll-gap p99 above target", percentile(990))
	}
}
