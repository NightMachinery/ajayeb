package tableroom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func policyInput(r *Record, g *game, seat int) botInput {
	row, ok := workRow(r, g.Point().ID, seat)
	if !ok {
		panic("seat has no frozen preset")
	}
	o := g.Observe(seatNum(seat))
	return botInput{row: row, observation: o, offered: g.Legal(seatNum(seat)), entropy: botTape(r.Setup.Seed, g.Point().ID, seat, o.Fingerprint(), row.BotPreset)}
}

func TestFixedPresetsRejectUnsupportedWork(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, soloSpec(7, 501, BotHard))
	base := h.record()
	if len(base.BotPresets) != 3 || len(base.Work) != 6 {
		t.Fatal("missing frozen configuration", base.Work)
	}
	for _, row := range base.Work {
		p, _ := frozenPreset(&base, row.Level)
		if row.BotPreset != p || row.Allowance != 1 {
			t.Fatal("partial/divided preset", row)
		}
	}
	for name, change := range map[string]func(*Record){
		"policy family":  func(r *Record) { r.Work[0].Policy = "future" },
		"policy version": func(r *Record) { r.Work[0].PolicyVersion++ },
		"preset version": func(r *Record) { r.Work[0].Preset++ },
		"allowance":      func(r *Record) { r.Work[0].Allowance++ },
		"algorithm":      func(r *Record) { r.Work[0].Algorithm = "random" },
		"quote bound":    func(r *Record) { r.Work[0].MaxQuotes++ },
		"probe bound":    func(r *Record) { r.Work[0].MaxProbes++ },
		"offer bound":    func(r *Record) { r.Work[0].MaxChoices++ },
		"future config":  func(r *Record) { r.BotPresets[0].PolicyVersion++ },
		"missing config": func(r *Record) { r.BotPresets = nil },
	} {
		t.Run(name, func(t *testing.T) {
			r := base.clone()
			change(&r)
			if _, _, err := restore(&r, defaultBounds); err == nil {
				t.Fatal("unsupported policy was reinterpreted")
			}
		})
	}
	// A table frozen under preset set 1 (easy Random, normal Apprentice)
	// archives on load rather than playing on under set 2.
	v1 := base.clone()
	for i := range v1.BotPresets {
		v1.BotPresets[i].Policy = "haftgane-bot-1"
	}
	v1.BotPresets[0].Algorithm, v1.BotPresets[1].Algorithm = bot.Random.String(), bot.Apprentice.String()
	for i := range v1.Work {
		v1.Work[i].Policy = "haftgane-bot-1"
	}
	data, _ := v1.encode()
	if verdict, _, err := assess(data, defaultBounds); err != nil || verdict.Archive != store.ArchiveReplayFailed {
		t.Fatal("preset set 1 was not archived", verdict, err)
	}
	data, _ = base.encode()
	var old map[string]any
	_ = json.Unmarshal(data, &old)
	old["format"] = 1
	delete(old, "botPresets")
	data, _ = json.Marshal(old)
	verdict, _, err := assess(data, defaultBounds)
	if err != nil || verdict.Archive == "" {
		t.Fatal("old fallback configuration silently upgraded", verdict, err)
	}
}

// Each room code plays its persona's policy where one exists; normal and hard
// share Greedy until Scholar, and Random is never a room level (D46).
func TestRoomPresetMapping(t *testing.T) {
	got := map[string]string{}
	for _, p := range releasedPresets() {
		if p.Policy != "haftgane-bot-2" {
			t.Fatal("preset set", p)
		}
		got[p.Level] = p.Algorithm
	}
	want := map[string]string{BotEasy: "apprentice", BotNormal: "greedy", BotHard: "greedy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("room mapping", got)
	}
}

// Levels are checked against the table's frozen presets, not the build's:
// a level the table never froze gets no work row and no seat.
func TestLevelsComeFromFrozenPresets(t *testing.T) {
	r := Record{BotPresets: releasedPresets()[:1], Seats: []SeatRecord{{Seat: 0, Kind: SeatBot, Bot: BotEasy}, {Seat: 1, Kind: SeatBot, Bot: BotHard}}}
	if row, ok := workRow(&r, "p", 0); !ok || row.Seat != 0 || row.BotPreset != r.BotPresets[0] {
		t.Fatal("frozen level", row)
	}
	if _, ok := workRow(&r, "p", 1); ok {
		t.Fatal("unfrozen level got a work row")
	}
	if checkSeats(&r) == nil {
		t.Fatal("unfrozen level kept a seat")
	}
}

// Real permitted inputs, not a state hash or pending picks, determine the
// complete decision and its counters. Scenario fixtures vary only future
// packets and are never used as persisted room checkpoints.
func TestRoomPolicyRepeatabilityAndForbiddenFacts(t *testing.T) {
	for n := 3; n <= 7; n++ {
		su := makeDaySetup(uint64(700+n), n, false, false)
		g, err := newGame(su)
		if err != nil {
			t.Fatal(err)
		}
		sc := rules.ScenarioFromGame(g.Game)
		sc.State.Deals[1][1][0], sc.State.Deals[1][2][0] = sc.State.Deals[1][2][0], sc.State.Deals[1][1][0]
		other, err := rules.NewGameFromScenario(su.Seed, g.State.Opts, sc)
		if err != nil {
			t.Fatal(err)
		}
		for _, preset := range releasedPresets() {
			r := Record{Setup: SetupRecord{Seed: su.Seed}, BotPresets: releasedPresets(), Seats: make([]SeatRecord, n)}
			r.Seats[0] = SeatRecord{Kind: SeatBot, Bot: preset.Level}
			in := policyInput(&r, &g, 0)
			altered := policyInput(&r, &game{other}, 0)
			if in.observation != altered.observation || in.entropy != altered.entropy || !reflect.DeepEqual(in.offered, altered.offered) {
				t.Fatal("forbidden facts changed inputs")
			}
			want, err := runBot(context.Background(), in)
			if err != nil || !want.Complete {
				t.Fatal(err)
			}
			for i := 0; i < 4; i++ {
				got, err := runBot(context.Background(), altered)
				if err != nil || got != want {
					t.Fatal("same observation/row/entropy changed key or counters", err)
				}
			}
			r.Pending = []PendingSlot{{Seat: 1, Key: "inert-private-draft", Confirmed: true}}
			got, err := runBot(context.Background(), policyInput(&r, &g, 0))
			if err != nil || got != want {
				t.Fatal("another pending pick changed decision", err)
			}
			if want.Counters.Iterations != in.row.Allowance || !slices.Contains(legalKeys(&g, 0), want.Key) {
				t.Fatal("not a full offered-key decision", want)
			}
		}
	}
}

// All 64 subsets of six accepted bot picks cold-load through the real room
// manager. No accepted bot is searched again, and every missing bot retains
// its full frozen row. Human none/draft/confirmed variants also restore.
func TestBotEveryPendingSubsetRestarts(t *testing.T) {
	for mask := 0; mask < 64; mask++ {
		t.Run(fmt.Sprint(mask), func(t *testing.T) {
			hook := faults.NewWork(true)
			sp := soloSpec(7, 812, BotHard)
			for s := 1; s < 7; s++ {
				sp.Participants[s].Bot = []string{BotEasy, BotNormal, BotHard}[s%3]
			}
			h := newTH(t, Config{BotWork: hook.Run}, sp)
			<-hook.Started()
			r := h.record()
			su, _ := recordSetup(r.Setup)
			g, _ := replay(su, nil)
			for seat := 1; seat < 7; seat++ {
				if mask&(1<<(seat-1)) != 0 {
					d, err := runBot(context.Background(), policyInput(&r, &g, seat))
					if err != nil {
						t.Fatal(err)
					}
					r.Pending = append(r.Pending, PendingSlot{Seat: seat, Key: d.Key, Confirmed: true, Seqs: []SeqRecord{}})
				}
			}
			for _, human := range []int{0, 1, 2} {
				c := r.clone()
				if human != 0 {
					c.Pending = append([]PendingSlot{{Seat: 0, Key: legalKeys(&g, 0)[0], Confirmed: human == 2, Seqs: []SeqRecord{}}}, c.Pending...)
				}
				restored, ok, err := restore(&c, defaultBounds)
				if human == 2 && mask == 63 {
					if err == nil {
						t.Fatal("fully confirmed point stayed open")
					}
					continue
				}
				if err != nil || !ok || !reflect.DeepEqual(restored.rec.Work, r.Work) || !reflect.DeepEqual(restored.rec.Pending, c.Pending) {
					t.Fatal("subset changed", mask, human, err)
				}
			}
			h.f.close()
			data, err := r.encode()
			if err != nil {
				t.Fatal(err)
			}
			if err := h.f.store.SaveGame(r.Code, Game, data); err != nil {
				t.Fatal(err)
			}
			restartStart := time.Now()
			h.f.start()
			h.rejoin()
			if evidenceEnabled() {
				t.Logf("room-restore-evidence subset=%d cold_start_join_ns=%d", mask, time.Since(restartStart).Nanoseconds())
			}
			got := h.record()
			if !reflect.DeepEqual(got.Pending, r.Pending) || !reflect.DeepEqual(got.Work, r.Work) {
				t.Fatal("cold load changed accepted subset/row", mask)
			}
			if mask != 63 {
				next := <-hook.Started()
				// The old cancelled search had one Start; the new one must be missing.
				if mask&(1<<(next.Seat-1)) != 0 {
					t.Fatal("accepted seat searched again", mask, next.Seat)
				}
			} else if h.botBusy() {
				t.Fatal("all accepted bots searched again")
			}
		})
	}
}

func TestBotUndoRepeatsActualPicks(t *testing.T) {
	hook := faults.NewWork(true)
	h := newTH(t, Config{BotWork: hook.Run}, soloSpec(3, 910, BotNormal))
	me := tabletest.SeatClient(0)
	first := h.latest(me)
	humanKey := first.View.You.Choices[0].Key
	h.pickSelf(1, humanKey)
	hook.Release()
	hook.Release()
	fr := h.until(me, "first batch", func(st wire.TableState) bool { return st.Batch == 1 })
	original := h.record().Batches[0]
	<-hook.Started()
	<-hook.Started()
	<-hook.Started() // two finished, next point blocked
	if got := h.undo(2, fr.Revision); got != outAccepted {
		t.Fatal(got)
	}
	<-hook.Started() // reopened point worker, after old cancellation acknowledgement
	h.pickSelf(3, humanKey)
	hook.Release()
	hook.Release()
	h.until(me, "repeated batch", func(st wire.TableState) bool { return st.Batch == 1 })
	if got := h.record().Batches[0]; !reflect.DeepEqual(got, original) {
		t.Fatal("undo changed bot choices", got, original)
	}
}

func TestBotLateResultsAndUnrelatedChanges(t *testing.T) {
	hook := faults.NewWork(true)
	h := newTH(t, Config{BotWork: hook.Run}, soloSpec(4, 911, BotNormal))
	start := <-hook.Started()
	a, _ := h.p.actor()
	var late msgBotResult
	_ = a.call(h.ctx, func() {
		late = msgBotResult{owner: a, gen: a.botGen, point: start.Point, seat: start.Seat, key: legalKeys(&a.st.game, start.Seat)[0]}
	})
	h.want(tabletest.SeatClient(0), roleOp(1, "promote", "watcher"), outAccepted, "unrelated role change")
	_ = a.call(h.ctx, func() {
		if a.botGen != late.gen {
			t.Error("role revision invalidated search")
		}
	})
	h.want(tabletest.SeatClient(0), toHuman(2, start.Seat, "watcher"), outAccepted, "human cancels searched seat")
	<-hook.Started() // next bot, after physical cancellation completed
	before := h.raw()
	_ = a.call(h.ctx, func() { a.botResult(late) })
	if !bytes.Equal(before, h.raw()) {
		t.Fatal("late changed-seat answer wrote a pick")
	}
	h.f.restart()
	h.rejoin()
	before = h.raw()
	a, _ = h.p.actor()
	var generation uint64
	_ = a.call(h.ctx, func() { generation = a.botGen })
	_ = a.call(h.ctx, func() { a.botResult(late) })
	if !bytes.Equal(before, h.raw()) {
		t.Fatal("pre-restart answer changed restored room")
	}
	_ = a.call(h.ctx, func() {
		if a.botGen != generation {
			t.Error("old actor answer cancelled replacement work")
		}
	})
	// Closed point and impossible seat results cannot affect a current worker.
	late.point = "closed-point"
	late.gen = ^uint64(0)
	late.seat = -1
	_ = a.call(h.ctx, func() { a.botResult(late) })
	if !bytes.Equal(before, h.raw()) {
		t.Fatal("closed-point late answer changed room")
	}
}

func TestCancelledWorkerFinishesBeforeReplacement(t *testing.T) {
	started := make(chan int, 4)
	release := make(chan struct{}, 4)
	var active, maxActive atomic.Int32
	hook := func(ctx context.Context, seat int, _ string) error {
		now := active.Add(1)
		for old := maxActive.Load(); now > old && !maxActive.CompareAndSwap(old, now); old = maxActive.Load() {
		}
		started <- seat
		<-release // deliberately hold cleanup after cancellation
		active.Add(-1)
		return ctx.Err()
	}
	pool := ai.NewPool(2)
	h := newTH(t, Config{Pool: pool, BotWork: hook}, soloSpec(3, 912, BotNormal))
	t.Cleanup(func() { release <- struct{}{}; release <- struct{}{} })
	seat := <-started
	h.want(tabletest.SeatClient(0), toHuman(1, seat, "watcher"), outAccepted, "replace active bot")
	if active.Load() != 1 || pool.InUse() != 1 || pool.Waiting() != 0 {
		t.Fatal("cancelled cleanup overlapped replacement")
	}
	release <- struct{}{}
	<-started
	if maxActive.Load() != 1 {
		t.Fatal("two room workers overlapped")
	}
}

func TestBotQueuedCancellationReleasesWaiter(t *testing.T) {
	pool := ai.NewPool(1)
	if err := pool.AcquireFor(context.Background(), "blocker"); err != nil {
		t.Fatal(err)
	}
	h := newTH(t, Config{Pool: pool}, soloSpec(3, 913, BotHard))
	waitFor(t, "queued bot", func() bool { return pool.Waiting() == 1 })
	h.leave(tabletest.SeatClient(0))
	waitFor(t, "removed waiter", func() bool { return pool.Waiting() == 0 })
	pool.ReleaseFor("blocker")
	waitFor(t, "no permits", func() bool { return pool.InUse() == 0 })
}

func TestBotWatchdogHasDurableSafeReason(t *testing.T) {
	pool := ai.NewPool(1)
	h := newTH(t, Config{Pool: pool, BotWatchdog: time.Millisecond, BotWork: blockBots}, soloSpec(3, 914, BotNormal))
	h.until(tabletest.SeatClient(0), "fallback confirmations", func(st wire.TableState) bool { return st.Point != nil && len(st.Point.Confirmed) == 2 })
	r := h.record()
	su, _ := recordSetup(r.Setup)
	g, _ := replay(su, nil)
	if len(r.BotFallbacks) != 2 {
		t.Fatal("missing watchdog reasons", r.BotFallbacks)
	}
	for _, f := range r.BotFallbacks {
		d, err := bot.Fallback(context.Background(), g.Observe(seatNum(f.Seat)), g.Legal(seatNum(f.Seat)))
		if err != nil || f.Reason != "watchdog" || pendingOf(&r, f.Seat).Key != d.Key {
			t.Fatal("fallback not exact deterministic greedy", err)
		}
	}
	h.f.restart()
	h.rejoin()
	if len(h.record().BotFallbacks) != 2 {
		t.Fatal("fallback diagnostic lost on restart")
	}
	if pool.InUse() != 0 || pool.Waiting() != 0 {
		t.Fatal("watchdog leaked permit")
	}
}

// Drive mixed tables with bot confirmations first. With six bots, each
// sample measures serialized wait including pool, actor/store and real views.
func playPresetTable(t *testing.T, h *th, opened time.Time) []time.Duration {
	t.Helper()
	h.ctx = testContextFor(t, 3*time.Minute)
	seq := make([]uint64, 7)
	samples := []time.Duration{}
	lastPoint := ""
	for steps := 0; steps < 200; steps++ {
		fr := h.until(tabletest.SeatClient(0), "bots and human decision or end", func(st wire.TableState) bool {
			if st.Status == "finished" {
				return true
			}
			if st.Point == nil {
				return false
			}
			human := false
			for _, s := range st.Point.Required {
				if st.Seats[s].Kind == SeatBot {
					if !slices.Contains(st.Point.Confirmed, s) {
						return false
					}
				} else if !slices.Contains(st.Point.Confirmed, s) {
					human = true
				}
			}
			return human
		})
		if fr.Status == "finished" {
			return samples
		}
		if fr.Point.Ref.ID != lastPoint {
			samples = append(samples, time.Since(opened))
			lastPoint = fr.Point.Ref.ID
		}
		for _, s := range fr.Point.Required {
			if fr.Seats[s].Kind == SeatHuman && !slices.Contains(fr.Point.Confirmed, s) {
				id := tabletest.SeatClient(s)
				sn := h.snap(id)
				seq[s]++
				opened = time.Now()
				outcome := h.send(id, seq[s], sn.Ref, sn.Legal[0], true)
				if outcome != tabletest.Accepted && outcome != tabletest.Resolved {
					t.Fatal(outcome)
				}
				break
			}
		}
	}
	t.Fatal("mixed table did not end")
	return nil
}

func TestPresetTablesAndSixBotWait(t *testing.T) {
	for count := 0; count <= 6; count++ {
		t.Run(fmt.Sprintf("%d bots", count), func(t *testing.T) {
			reports := make(chan BotReport, 512)
			sp := humans(7, uint64(920+count))
			for s := 7 - count; s < 7; s++ {
				sp.Participants[s] = Participant{Bot: []string{BotEasy, BotNormal, BotHard}[s%3]}
			}
			pool := ai.NewPool(1)
			opened := time.Now()
			h := newTH(t, Config{Pool: pool, BotFinished: func(r BotReport) { reports <- r }}, sp)
			samples := playPresetTable(t, h, opened)
			if len(h.record().BotFallbacks) != 0 {
				t.Fatal("ordinary watchdog fallback")
			}
			completed := 0
			var active, poolWait time.Duration
			for len(reports) > 0 {
				r := <-reports
				if r.Cancelled || r.Fallback != "" || r.Counters.Iterations != 1 {
					t.Fatal("incomplete or fallback ordinary work", r)
				}
				completed++
				active += r.Active
				poolWait += r.PoolWait
			}
			if pool.InUse() != 0 || pool.Waiting() != 0 {
				t.Fatal("permit leaked")
			}
			if count > 0 && completed == 0 {
				t.Fatal("no released policy calls")
			}
			if count == 6 {
				var sum, maxWait time.Duration
				for _, v := range samples {
					sum += v
					if v > maxWait {
						maxWait = v
					}
				}
				t.Logf("six mixed bots: %d point waits mean=%s max=%s; %d decisions active total=%s pool wait total=%s; watchdogs=0", len(samples), sum/time.Duration(len(samples)), maxWait, completed, active, poolWait)
			}
		})
	}
}

func TestSevenBotContinuationWithReleasedPolicies(t *testing.T) {
	h := newTH(t, Config{Pool: ai.NewPool(1)}, soloSpec(7, 931, BotHard))
	h.want(tabletest.SeatClient(0), wire.TableResign{T: "table-resign", Seq: 1, Confirm: true}, outAccepted, "last human resigns and watches")
	h.until(tabletest.SeatClient(0), "seven-bot continuation", func(st wire.TableState) bool { return st.Status == "finished" })
	if r := h.record(); len(r.BotFallbacks) != 0 || r.Seats[0].Bot != DefaultBotLevel {
		t.Fatal("continuation policy/fallback", r.BotFallbacks)
	}
}

func TestPresetSwitchKeepsPickAndFullRow(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(4, 941))
	me, c1 := tabletest.SeatClient(0), tabletest.SeatClient(1)
	sn := h.snap(c1)
	h.expect(h.send(c1, 1, sn.Ref, sn.Legal[1], false), tabletest.Accepted, "human draft")
	check := func(level string) {
		r := h.record()
		preset, _ := frozenPreset(&r, level)
		i := slices.IndexFunc(r.Work, func(w WorkRow) bool { return w.Seat == 1 })
		if i < 0 || r.Work[i].BotPreset != preset || pendingOf(&r, 1).Key != sn.Legal[1] || !pendingOf(&r, 1).Confirmed {
			t.Fatal("switch lost kept pick or full row")
		}
	}
	h.want(me, toBot(1, 1, BotHard), outAccepted, "admin selects hard")
	check(BotHard)
	h.want(me, toHuman(2, 1, userOf(1)), outAccepted, "former human returns")
	if fr := h.latest(c1); fr.Pending == nil || fr.Pending.Key != sn.Legal[1] {
		t.Fatal("human lost pick")
	}
	h.expect(h.send(c1, 2, sn.Ref, sn.Legal[1], false), tabletest.Accepted, "human can withdraw")
	h.want(me, toBot(3, 1, BotNormal), outAccepted, "switch with full normal row")
	check(BotNormal)
	h.f.restart()
	h.rejoin()
	check(BotNormal)
}

func TestReleasedRoomPoliciesAtSideAndAbilities(t *testing.T) {
	for _, n := range []int{3, 7} {
		for _, kind := range []rules.PointKind{rules.PointSides, rules.PointDiscardBuild, rules.PointLastCard} {
			t.Run(fmt.Sprintf("%d/%d", n, kind), func(t *testing.T) {
				seed := uint64(941)
				prefix := []BatchRecord{}
				sides := "choose"
				if kind != rules.PointSides {
					b := boundaryPrefix(n, kind, 0)
					if !b.ok {
						t.Fatal("missing natural boundary")
					}
					seed, prefix, sides = b.seed, b.prefix, b.sides
				}
				su, _ := recordSetup(SetupRecord{Seed: seed, Seats: n, Sides: sides})
				g, err := replay(su, engineBatches(prefix))
				if err != nil {
					t.Fatal(err)
				}
				for _, level := range []string{BotEasy, BotNormal, BotHard} {
					sp := humans(n, seed)
					sp.Sides = sides
					sp.Prefix = prefix
					target := int(g.Point().Required.Seats()[0])
					human := (target + 1) % n
					if kind == rules.PointSides {
						target, human = 1, 0
					}
					sp.Creator = userOf(human)
					sp.Participants[target] = Participant{Bot: level}
					hook := faults.NewWork(true)
					h := newTH(t, Config{BotWork: hook.Run}, sp)
					started := <-hook.Started()
					if started.Seat != target {
						t.Fatal("wrong ability controller")
					}
					r := h.record()
					input := policyInput(&r, &g, target)
					want, err := runBot(context.Background(), input)
					if err != nil {
						t.Fatal(err)
					}
					hook.Release()
					waitFor(t, "side/ability policy pick", func() bool { r = h.record(); return len(r.Batches) > len(prefix) || pendingOf(&r, target) != nil })
					actual := ""
					if len(r.Batches) > len(prefix) {
						for _, p := range r.Batches[len(prefix)].Picks {
							if p.Seat == target {
								actual = p.Key
							}
						}
					} else {
						actual = pendingOf(&r, target).Key
					}
					if actual != want.Key || !want.Complete {
						t.Fatal("actor did not use released policy at point", kind, level)
					}
					h.f.close()
				}
			})
		}
	}
}

func TestSixBotPresetWaitMeasurements(t *testing.T) {
	for _, level := range []string{BotEasy, BotNormal, BotHard} {
		t.Run(level, func(t *testing.T) {
			reports := make(chan BotReport, 512)
			opened := time.Now()
			h := newTH(t, Config{Pool: ai.NewPool(1), BotFinished: func(r BotReport) { reports <- r }}, soloSpec(7, 942, level))
			samples := playPresetTable(t, h, opened)
			var active, poolWait, sum, maxWait time.Duration
			decisions := 0
			for len(reports) > 0 {
				r := <-reports
				if r.Cancelled || r.Fallback != "" || r.Counters.Iterations != 1 {
					t.Fatal("ordinary truncated/fallback work", r)
				}
				active += r.Active
				poolWait += r.PoolWait
				decisions++
			}
			for _, d := range samples {
				sum += d
				maxWait = max(maxWait, d)
			}
			if len(h.record().BotFallbacks) != 0 || decisions < 6*18 {
				t.Fatal("missing full policy work")
			}
			t.Logf("six %s bots: %d human-point waits mean=%s max=%s, %d completed decisions active=%s pool-wait=%s watchdogs=0", level, len(samples), sum/time.Duration(len(samples)), maxWait, decisions, active, poolWait)
		})
	}
}

// The actual room calls, not just repeated pure calls, produce identical
// accepted keys and complete counters when human arrival and restarts differ.
func TestBotSchedulesAndRestartSameBatch(t *testing.T) {
	var first BatchRecord
	var firstCounts map[int]bot.Counters
	for mode := 0; mode < 2; mode++ {
		hook := faults.NewWork(true)
		reports := make(chan BotReport, 64)
		sp := soloSpec(7, 950, BotNormal)
		for seat := 1; seat < 7; seat++ {
			sp.Participants[seat].Bot = []string{BotEasy, BotNormal, BotHard}[seat%3]
		}
		h := newTH(t, Config{BotWork: hook.Run, BotFinished: func(r BotReport) { reports <- r }}, sp)
		me := tabletest.SeatClient(0)
		sn := h.snap(me)
		humanKey := sn.Legal[0]
		a, _ := h.p.actor()
		var late msgBotResult
		_ = a.call(h.ctx, func() {
			late = msgBotResult{owner: a, gen: a.botGen, seat: a.botSeat, point: a.botPoint, key: legalKeys(&a.st.game, a.botSeat)[0]}
		})
		if mode == 0 {
			h.pickSelf(1, humanKey)
		} else {
			h.expect(h.send(me, 1, sn.Ref, sn.Legal[1], false), tabletest.Accepted, "private draft before bot commits")
			for i := 0; i < 3; i++ {
				hook.Release()
			}
			h.until(me, "three durable bots", func(st wire.TableState) bool { return st.Point != nil && len(st.Point.Confirmed) == 3 })
			h.f.restart()
			h.rejoin()
			h.expect(h.send(me, 2, sn.Ref, humanKey, false), tabletest.Accepted, "replace draft after restart")
		}
		left := 6
		if mode != 0 {
			left = 3
		}
		for i := 0; i < left; i++ {
			hook.Release()
		}
		if mode != 0 {
			h.until(me, "all six bot confirmations", func(st wire.TableState) bool { return st.Point != nil && len(st.Point.Confirmed) == 6 })
			h.pickSelf(3, humanKey)
		}
		h.until(me, "resolved same batch", func(st wire.TableState) bool { return st.Batch == 1 })
		r := h.record()
		counts := map[int]bot.Counters{}
		for len(reports) > 0 {
			report := <-reports
			if report.Point == late.point && !report.Cancelled {
				if _, exists := counts[report.Seat]; exists {
					t.Fatal("accepted seat searched twice")
				}
				counts[report.Seat] = report.Counters
			}
		}
		if len(counts) != 6 {
			t.Fatal("missing complete counters", counts)
		}
		if mode == 0 {
			first, firstCounts = r.Batches[0], counts
		} else if !reflect.DeepEqual(first, r.Batches[0]) || !reflect.DeepEqual(firstCounts, counts) {
			t.Fatal("arrival/restart changed accepted batch or work", first, r.Batches[0], firstCounts, counts)
		}
		a, _ = h.p.actor()
		before := h.raw()
		_ = a.call(h.ctx, func() { a.botResult(late) })
		if !bytes.Equal(before, h.raw()) {
			t.Fatal("answer from closed point changed record")
		}
		h.f.close()
	}
}
