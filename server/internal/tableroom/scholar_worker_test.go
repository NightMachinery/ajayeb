package tableroom

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Twenty actual Scholar trajectories, not shadow searches. Submit the human's
// first legal pick as soon as its point is offered; measure until the resolving
// batch reaches that human's outbox. Include serialization, pool, storage and
// views. Bot-only ability points advance naturally and have no human-pick wait.
func TestScholarWorkerWatchdogMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_WORKER") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_WORKER=1")
	}
	t.Logf("go=%s os=%s arch=%s cpus=%d gomaxprocs=%d games=20 pool_permits=1 other_tables=0 competing_cpu_workers=0 watchdog=30s accepted_policy=Scholar", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.GOMAXPROCS(0))
	var aggregate []int64
	var gameTimes []int64
	watchdogs, sampleFallbacks, calls := 0, 0, 0
	for game := 0; game < 20; game++ {
		t.Run(fmt.Sprint(game), func(t *testing.T) {
			reports := make(chan BotReport, 1024)
			sp := soloSpec(7, uint64(942+game*1000), BotNormal)
			sp.Sides = []string{"day", "random", "choose"}[game%3]
			pool := ai.NewPool(1)
			opened := time.Now()
			h := newTH(t, Config{Pool: pool, BotFinished: func(r BotReport) { reports <- r }}, sp)
			h.ctx = testContextFor(t, 5*time.Minute)
			me := tabletest.SeatClient(0)
			seq, batch, sentPoint := 0, 0, ""
			var sent time.Time
			var waits []int64
			for {
				fr := h.until(me, "new human point or resolving batch", func(st wire.TableState) bool {
					return st.Status == "finished" || st.Batch > batch || st.Point != nil && st.Point.Ref.ID != sentPoint && st.View.You != nil && len(st.View.You.Choices) > 0 && slices.Contains(st.Point.Required, 0)
				})
				if !sent.IsZero() && fr.Batch > batch {
					waits = append(waits, int64(time.Since(sent)))
					sent = time.Time{}
				}
				batch = fr.Batch
				if fr.Status == "finished" {
					break
				}
				if fr.Point == nil || fr.Point.Ref.ID == sentPoint || fr.View.You == nil || len(fr.View.You.Choices) == 0 || !slices.Contains(fr.Point.Required, 0) {
					continue
				}
				seq++
				sentPoint, sent = fr.Point.Ref.ID, time.Now()
				h.pickSelf(seq, fr.View.You.Choices[0].Key)
			}
			aggregate = append(aggregate, waits...)
			gameTimes = append(gameTimes, int64(time.Since(opened)))
			for len(reports) > 0 {
				r := <-reports
				if r.Cancelled {
					t.Fatal("ordinary worker cancelled", r)
				}
				if r.Fallback == "watchdog" {
					watchdogs++
				}
				if r.Fallback == "no-world" {
					sampleFallbacks++
				}
				if r.AgeCounters.SampleAttempts != bot.ScholarAttempts {
					t.Fatal("reduced preset", r)
				}
				calls++
			}
			if pool.InUse() != 0 || pool.Waiting() != 0 {
				t.Fatal("permit leak")
			}
			t.Logf("scholar-room game=%d human_pick_to_resolve_ns=%+v", game, evidenceQuantify(waits))
			h.f.close()
		})
	}
	t.Logf("scholar-room aggregate human_pick_to_resolve_ns=%+v game_ns=%+v bot_calls=%d watchdogs=%d sample_fallbacks=%d", evidenceQuantify(aggregate), evidenceQuantify(gameTimes), calls, watchdogs, sampleFallbacks)
	if watchdogs != 0 {
		t.Error("ordinary watchdog fallback")
	}
}
