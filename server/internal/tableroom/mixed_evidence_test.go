package tableroom

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// A transparent wrapper around the released Duel bot. made-to-complete
// includes dispatch, permitted-observation construction, pool wait and Choose.
// It excludes the actor's subsequent commit and the optional think delay.
type evidenceDuelBot struct {
	bot   room.Bot
	made  time.Time
	stats *evidenceStats
}

func (b evidenceDuelBot) Choose(ctx context.Context, o engine.Observation, budget ai.Budget, progress func(ai.Progress)) (engine.Action, error) {
	start := time.Now()
	a, err := b.bot.Choose(ctx, o, budget, progress)
	end := time.Now()
	if err != nil {
		b.stats.add("duel_policy_errors", 1)
	}
	if err == nil {
		b.stats.add("duel_active_ns", int64(end.Sub(start)))
		b.stats.add("duel_dispatch_to_complete_ns", int64(end.Sub(b.made)))
		b.stats.add("duel_dispatch_to_search_ns", int64(start.Sub(b.made)))
	}
	return a, err
}

type evidenceDuel struct {
	m     *room.Manager
	c     *room.Client
	stats *evidenceStats
}

func newEvidenceDuel(t *testing.T, pool *ai.Pool, e *evidenceStats, trial, index int) *evidenceDuel {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := room.NewManager(room.Config{Store: st, BotPool: pool, Seed: func() uint64 { return uint64(17000 + trial*8 + index) }, NewBot: func(level store.BotLevel, seed uint64) room.Bot {
		return evidenceDuelBot{bot: room.DefaultBotFactory(level, seed), made: time.Now(), stats: e}
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	c := room.NewClient(fmt.Sprintf("evidence-player-%d", index), "evidence-tab", "Evidence")
	c.Addr = fmt.Sprintf("duel-evidence-address-%d", index)
	return &evidenceDuel{m: m, c: c, stats: e}
}
func (d *evidenceDuel) play(ctx context.Context) error {
	level := wire.BotLevelEasy
	req := wire.NewCreate(wire.ModeAI, &level, wire.SeatNumber(0), nil)
	req.Hints = &view.HintSettings{}
	code, err := d.m.Create(ctx, d.c, req)
	if err != nil {
		return err
	}
	seq, sentVersion := 0, -1
	pending := 0
	var sent time.Time
	var latest wire.State
	for {
		for _, msg := range d.c.Out.Take() {
			switch fr := msg.(type) {
			case wire.State:
				if pending != 0 && fr.Ack != nil && fr.Ack.Seq == pending {
					d.stats.add("duel_action_ack_ns", int64(time.Since(sent)))
					pending = 0
				}
				latest = fr
			case wire.Reject:
				return fmt.Errorf("Duel measurement request rejected: %s", fr.Code)
			case wire.Error:
				return fmt.Errorf("Duel measurement error: %s", fr.Code)
			}
		}
		if latest.Room.Status == wire.RoomStatusFinished {
			return nil
		}
		if pending == 0 && latest.Version > sentVersion && latest.You.CanAct && len(latest.View.Choices) > 0 {
			seq++
			pending = seq
			sentVersion = latest.Version
			sent = time.Now()
			if err := d.m.Act(ctx, d.c, wire.NewAct(code, seq, latest.Version, latest.View.Choices[0].Key, nil)); err != nil {
				return err
			}
		}
		select {
		case <-d.c.Out.Ready():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Matched full-game trials alternate cohort order. Both use two real Duel
// Easy rooms, full default work, no advisor, no artificial think delay and
// one shared pool permit. Mixed adds one continuously replaced six-bot table.
// This is a queue/actor baseline on the test host, not a release capacity gate.
func TestMixedRoomEvidence(t *testing.T) {
	if !evidenceEnabled() {
		t.Skip("opt-in mixed room measurements")
	}
	const trials = 4
	const duelRooms = 2
	eBaseline, eMixed, eHaft := newEvidenceStats(), newEvidenceStats(), newEvidenceStats()
	for trial := 0; trial < trials; trial++ {
		order := []bool{false, true}
		if trial%2 == 1 {
			order = []bool{true, false}
		}
		for _, mixed := range order {
			label := "duel-only"
			e := eBaseline
			if mixed {
				label = "mixed"
				e = eMixed
			}
			t.Run(fmt.Sprintf("%d/%s", trial, label), func(t *testing.T) {
				pool := ai.NewPool(1)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				duels := make([]*evidenceDuel, 0, duelRooms)
				for index := 0; index < duelRooms; index++ {
					duels = append(duels, newEvidenceDuel(t, pool, e, trial, index))
				}
				var background sync.WaitGroup
				haftErr := make(chan error, 1)
				if mixed {
					background.Add(1)
					started := make(chan struct{})
					var once sync.Once
					go func() {
						defer background.Done()
						var runErr error
						defer func() { haftErr <- runErr }()
						for game := 0; ctx.Err() == nil; game++ {
							work, done := eHaft.hooks()
							f := newFixture(t, Config{Pool: pool, BotWork: func(c context.Context, seat int, point string) error {
								once.Do(func() { close(started) })
								return work(c, seat, point)
							}, BotFinished: done})
							s := newEvidenceStore(f.store, eHaft)
							f.tables.Bind(f.m, s)
							sp := soloSpec(7, uint64(18000+trial*100+game), BotHard)
							sp.Sides = "choose"
							p, err := f.openInproc(ctx, sp)
							if err == nil {
								err = evidencePlay(ctx, &th{t: t, ctx: ctx, f: f, p: p}, eHaft, s)
							}
							f.close()
							if err != nil {
								if ctx.Err() == nil {
									runErr = err
								}
								return
							}
							eHaft.add("completed_background_games", 1)
						}
					}()
					select {
					case <-started:
					case <-ctx.Done():
						t.Fatal("background table did not start")
					}
				}
				results := make(chan error, duelRooms)
				for _, d := range duels {
					go func() { results <- d.play(ctx) }()
				}
				var duelErr error
				for range duels {
					if err := <-results; err != nil {
						duelErr = err
					}
				}
				cancel()
				background.Wait()
				if mixed {
					if err := <-haftErr; err != nil && !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
				for _, d := range duels {
					d.m.Close()
				}
				if duelErr != nil {
					t.Fatal(duelErr)
				}
				if pool.InUse() != 0 || pool.Waiting() != 0 {
					t.Fatal("shared pool did not drain")
				}
			})
		}
	}
	eBaseline.mu.Lock()
	errorsBaseline := len(eBaseline.values["duel_policy_errors"])
	eBaseline.mu.Unlock()
	eMixed.mu.Lock()
	errorsMixed := len(eMixed.values["duel_policy_errors"])
	eMixed.mu.Unlock()
	if errorsBaseline+errorsMixed != 0 {
		t.Fatal("Duel policy errors in measurement")
	}
	eBaseline.log(t, "duel-only")
	eMixed.log(t, "mixed-duel")
	eHaft.log(t, "mixed-haftgane-background")
}
