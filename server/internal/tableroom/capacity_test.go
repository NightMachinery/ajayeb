package tableroom

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
)

// Both tables finish full Scholar games while two Duel actors keep requesting
// full Apprentice decisions. Replay the same Duel game counts without tables.
// This complements the short alternating mixed probe, whose Duel games can
// finish while Scholar is still choosing sides.
func TestCapacityMixedEvidence(t *testing.T) {
	if os.Getenv("HAFTGANE_CAPACITY") != "1" {
		t.Skip("opt-in host capacity")
	}
	limit := capacityEnv("HAFTGANE_CAPACITY_ROOMS", 1)
	duelStats, haftStats := newEvidenceStats(), newEvidenceStats()
	pool, gate := ai.NewPool(2), ai.NewPool(limit)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var duelWG sync.WaitGroup
	counts := [2]int{}
	duelErr := make(chan error, 2)
	for i := range 2 {
		d := newEvidenceDuel(t, pool, duelStats, 90, i)
		duelWG.Add(1)
		go func() {
			defer duelWG.Done()
			for ctx.Err() == nil {
				if err := d.play(ctx); err != nil {
					if ctx.Err() == nil {
						duelErr <- err
					}
					return
				}
				counts[i]++
			}
		}()
	}
	tableResults := make(chan error, 2)
	for i := range 2 {
		work, done := haftStats.hooks()
		f := newFixture(t, Config{Pool: pool, RoomWork: gate, BotWork: work, BotFinished: done})
		s := newEvidenceStore(f.store, haftStats)
		f.tables.Bind(f.m, s)
		sp := soloSpec(7, uint64(27942+i*1000), BotNormal)
		sp.Sides = "choose"
		p, err := f.openInproc(ctx, sp)
		if err != nil {
			t.Fatal(err)
		}
		go func() { tableResults <- evidencePlay(ctx, &th{t: t, ctx: ctx, f: f, p: p}, haftStats, s) }()
	}
	for range 2 {
		if err := <-tableResults; err != nil {
			t.Error(err)
		}
	}
	cancel()
	duelWG.Wait()
	close(duelErr)
	for err := range duelErr {
		t.Error(err)
	}
	duelStats.log(t, fmt.Sprintf("complete-mixed-duel-limit-%d", limit))
	haftStats.log(t, fmt.Sprintf("complete-mixed-haftgane-limit-%d", limit))
	baseStats := newEvidenceStats()
	baseCtx, baseCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer baseCancel()
	results := make(chan error, 2)
	baselinePool := ai.NewPool(2)
	for i := range 2 {
		d := newEvidenceDuel(t, baselinePool, baseStats, 90, i)
		go func() {
			for game := 0; game < counts[i]; game++ {
				if err := d.play(baseCtx); err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
	baseStats.log(t, fmt.Sprintf("complete-duel-only-limit-%d", limit))
	t.Logf("capacity limit=%d complete_duel_games=%v", limit, counts)
	if haftStats.fallbacks != 0 {
		t.Fatal("ordinary fallback")
	}
}
