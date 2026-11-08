package tableroom

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Shadow full candidate work under the real pool admission and watchdog. The
// hook cannot choose the accepted action: released Greedy still plays the
// trajectory. This measures worker completion, not Scholar playing strength.
func TestScholarWorkerWatchdogMeasurements(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_WORKER") != "1" {
		t.Skip("opt-in dormant Scholar worker measurement")
	}
	t.Logf("go=%s cpus=%d gomaxprocs=%d pool_permits=1 other_tables=0 competing_cpu_workers=0 watchdog=30s", runtime.Version(), runtime.NumCPU(), runtime.GOMAXPROCS(0))
	for _, seed := range []uint64{942, 1942, 2942} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			ready := make(chan struct{})
			reports := make(chan BotReport, 512)
			decisions := make(chan struct{}, 512)
			var h *th
			hook := func(ctx context.Context, seat int, point string) error {
				select {
				case <-ready:
				case <-ctx.Done():
					return ctx.Err()
				}
				a, err := h.p.actor()
				if err != nil {
					return err
				}
				var in botInput
				samePoint := false
				if err := a.call(ctx, func() { in = policyInput(&a.st.rec, &a.st.game, seat); samePoint = a.st.game.Point().ID == point }); err != nil {
					return err
				}
				if !samePoint {
					return fmt.Errorf("shadow input point changed")
				}
				d, err := bot.ChooseAge(ctx, in.observation, in.offered, in.entropy, bot.AgeWork{Attempts: 2, Candidates: 8, SamplerChecks: rules.MaxSamplerChecks})
				if err != nil {
					return err
				}
				if !d.Complete || d.Fallback || d.Counters.SampleAttempts != 2 || d.Counters.CompletedSweeps == 0 {
					return fmt.Errorf("shadow work incomplete or used Greedy fallback")
				}
				decisions <- struct{}{}
				return nil
			}
			sp := soloSpec(7, seed, BotNormal)
			sp.Sides = "choose"
			opened := time.Now()
			h = newTH(t, Config{Pool: ai.NewPool(1), BotWork: hook, BotFinished: func(r BotReport) { reports <- r }}, sp)
			close(ready)
			waits := playPresetTable(t, h, opened)
			wall := time.Since(opened)
			var active, poolWait, maxActive time.Duration
			count := 0
			for len(reports) > 0 {
				r := <-reports
				if r.Cancelled || r.Fallback != "" {
					t.Fatal("ordinary worker cancelled or fell back", r)
				}
				active += r.Active
				poolWait += r.PoolWait
				maxActive = max(maxActive, r.Active)
				count++
			}
			if count < 108 || len(decisions) != count || len(h.record().BotFallbacks) != 0 {
				t.Fatal("missing full shadow calls or durable watchdog fallback", count, len(decisions))
			}
			t.Logf("shadow_calls=%d human_point_waits=%d game_ms=%.3f active_ms=%.3f max_active_ms=%.3f pool_wait_ms=%.3f watchdogs=0 greedy_sample_fallbacks=0 accepted_policy=Greedy", count, len(waits), float64(wall)/float64(time.Millisecond), float64(active)/float64(time.Millisecond), float64(maxActive)/float64(time.Millisecond), float64(poolWait)/float64(time.Millisecond))
			h.f.close()
		})
	}
}
