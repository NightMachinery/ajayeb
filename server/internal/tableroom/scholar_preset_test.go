package tableroom

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestScholarStoredPresetAndLegacyContinuation(t *testing.T) {
	for _, old := range []bool{false, true} {
		t.Run(map[bool]string{false: "Scholar", true: "legacy"}[old], func(t *testing.T) {
			hook := faults.NewWork(true)
			h := newTH(t, Config{BotWork: hook.Run}, soloSpec(3, 1821, BotNormal))
			<-hook.Started()
			r := h.record()
			if old {
				r.BotPresets = legacyPresets()
				r.Seats[2].Bot, r.Start[r.Assignment[2]].Bot = BotHard, BotHard
				for i := range r.Work {
					r.Work[i], _ = workRow(&r, r.Point.ID, r.Work[i].Seat)
				}
			}
			h.f.close()
			data, err := r.encode()
			if err != nil {
				t.Fatal(err)
			}
			// Simulate the exact historical JSON, before the Scholar fields existed.
			if old {
				var raw map[string]any
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Fatal(err)
				}
				for _, field := range []string{"botPresets", "work"} {
					for _, value := range raw[field].([]any) {
						row := value.(map[string]any)
						delete(row, "candidates")
						delete(row, "samplerChecks")
					}
				}
				data, err = json.Marshal(raw)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := h.f.store.SaveGame(r.Code, Game, data); err != nil {
				t.Fatal(err)
			}
			h.f.start()
			h.rejoin()
			got := h.record()
			if !reflect.DeepEqual(got.BotPresets, r.BotPresets) || !reflect.DeepEqual(got.Work, r.Work) {
				t.Fatal("restore replaced frozen work")
			}
			frame := h.latest(tabletest.SeatClient(0))
			wantLevels := []string{BotEasy, BotNormal}
			if old {
				wantLevels = append(wantLevels, BotHard)
			}
			if !slices.Equal(frame.BotLevels, wantLevels) {
				t.Fatal("offered levels", frame.BotLevels)
			}
			su, _ := recordSetup(r.Setup)
			g, _ := replay(su, nil)
			var keys [3]string
			for seat := 1; seat < 3; seat++ {
				in := policyInput(&r, &g, seat)
				d, err := runBot(context.Background(), in)
				if err != nil {
					t.Fatal(err)
				}
				if old {
					greedy, err := bot.Fallback(context.Background(), in.observation, in.offered)
					if err != nil || d.Decision != greedy || d.ageCounters != (bot.AgeCounters{}) {
						t.Fatal("legacy normal/hard changed", d, err)
					}
				} else if d.ageCounters.SampleAttempts != 2 || d.ageCounters.CompletedSweeps != 2 {
					t.Fatal("not full Scholar", d)
				}
				keys[seat] = d.Key
			}
			h.pickSelf(1, h.latest(tabletest.SeatClient(0)).View.You.Choices[0].Key)
			hook.Release()
			hook.Release()
			h.until(tabletest.SeatClient(0), "continued frozen batch", func(st wire.TableState) bool { return st.Batch == 1 })
			for _, pick := range h.record().Batches[0].Picks {
				if pick.Seat > 0 && pick.Key != keys[pick.Seat] {
					t.Fatal("restored worker changed key")
				}
			}
		})
	}
}

func TestScholarNewTableRefusesHard(t *testing.T) {
	f := newFixture(t, Config{})
	if _, _, err := f.tables.newRecord(soloSpec(3, 1822, BotHard)); err == nil {
		t.Fatal("new table offered Master")
	}
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 1823))
	h.want(tabletest.SeatClient(0), toBot(1, 1, BotHard), outBad, "unfrozen Master refused")
}

func TestScholarLevelSwitchUsesFrozenRows(t *testing.T) {
	for _, old := range []bool{false, true} {
		t.Run(map[bool]string{false: "Scholar", true: "legacy"}[old], func(t *testing.T) {
			h := newTH(t, Config{BotWork: blockBots}, humans(3, 1824))
			if old {
				r := h.record()
				h.f.close()
				r.BotPresets = legacyPresets()
				data, err := r.encode()
				if err != nil {
					t.Fatal(err)
				}
				if err := h.f.store.SaveGame(r.Code, Game, data); err != nil {
					t.Fatal(err)
				}
				h.f.start()
				h.rejoin()
			}
			levels := []string{BotNormal, BotEasy, BotNormal, BotEasy}
			if old {
				levels = append(levels, BotHard, BotNormal)
			}
			me := tabletest.SeatClient(0)
			for i, level := range levels {
				h.want(me, toBot(i+1, 1, level), outAccepted, "switch frozen level")
				r := h.record()
				want, _ := frozenPreset(&r, level)
				if len(r.Work) != 1 || r.Work[0].BotPreset != want {
					t.Fatal("switch changed preset", r.Work)
				}
			}
		})
	}
}

func TestScholarFallbackAndCancellation(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, soloSpec(3, 1825, BotNormal))
	r := h.record()
	su, _ := recordSetup(r.Setup)
	g, _ := replay(su, nil)
	in := policyInput(&r, &g, 1)
	// No legal past can have this pick with no Seen hand. This forces the
	// sampler's no-world outcome while keeping the current Greedy offer valid.
	in.observation.Know.Picks[1][0] = in.observation.Now.Hand[0]
	d, err := runBot(context.Background(), in)
	greedy, greedyErr := bot.Fallback(context.Background(), in.observation, in.offered)
	if err != nil || greedyErr != nil || d.Key != greedy.Key || d.fallback != "no-world" || d.ageCounters.GreedyFallbacks != 1 {
		t.Fatal("sample fallback", d, err)
	}
	// Both safe diagnostic reasons must survive a stored pending point.
	for _, reason := range []string{"no-world", "scholar-error"} {
		copy := r.clone()
		copy.Pending = append(copy.Pending, PendingSlot{Seat: 1, Key: greedy.Key, Confirmed: true, Seqs: []SeqRecord{}})
		copy.BotFallbacks = append(copy.BotFallbacks, BotFallback{At: 0, Point: copy.Point.ID, Seat: 1, Reason: reason})
		if _, _, err := restore(&copy, defaultBounds); err != nil {
			t.Fatal("fallback diagnostic did not restore", reason, err)
		}
	}
	for _, failure := range []struct {
		decision bot.AgeDecision
		err      error
	}{
		{err: errors.New("inert decision failure")}, {decision: bot.AgeDecision{Key: "inert-incomplete-key"}}, {},
	} {
		failed, err := finishScholar(context.Background(), in, failure.decision, failure.err)
		if err != nil || failed.Key != greedy.Key || failed.fallback != "scholar-error" {
			t.Fatal("decision failure skipped Greedy", failed, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err = runBot(ctx, in)
	if err != context.Canceled || d != (roomDecision{}) {
		t.Fatal("structural cancel accepted fallback", d, err)
	}
}
