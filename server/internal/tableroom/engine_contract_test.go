package tableroom

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

// Real setup, side decisions and batch-zero history are checked at the actor,
// with private drafts and restart, rather than only at the pure engine. The
// race build plays one seven-seat game, under choose, whose side point the
// others lack; every rule at three seats keeps the restart under the race
// detector, and every pair runs without it.
func TestEngineContractSidesAndReplay(t *testing.T) {
	for _, sides := range []string{"day", "random", "choose"} {
		for _, n := range []int{3, 7} {
			t.Run(sides+string(rune('0'+n)), func(t *testing.T) {
				if RaceEnabled && n == 7 && sides != "choose" {
					t.Skip("race build: seven seats under choose only")
				}
				seed := uint64(1000 + n)
				sp := humans(n, seed)
				sp.Sides = sides
				h := newTH(t, Config{}, sp)
				me := tabletest.SeatClient(0)
				first := h.p.frames[me][0]
				if first.History == nil || len(*first.History) != 1 || (*first.History)[0].Batch != 0 {
					t.Fatal("missing batch-zero history")
				}
				if sides == "choose" && (first.Point.Kind != "sides" || len(first.View.You.Hand) != 0 || len(first.View.You.Choices) != 2) {
					t.Fatal("not a side point")
				}
				sn := h.snap(me)
				h.expect(h.send(me, 1, sn.Ref, sn.Legal[0], false), tabletest.Accepted, "draft before restart")
				before := h.latest(me)
				h.f.restart()
				h.rejoin()
				after := h.latest(me)
				x, _ := json.Marshal(before.View)
				y, _ := json.Marshal(after.View)
				if !bytes.Equal(x, y) || before.Point.Ref != after.Point.Ref || !samePending(before.Pending, after.Pending) {
					t.Fatal("restart changed real view, pending or point")
				}
				// Independently drive a full game from the same setup. The
				// restored draft above uses a separate client seq lifetime.
				cfg := tabletest.Config{Seats: n, PickSeed: seed, ArrivalSeed: seed + 1, Script: tabletest.ScriptOptions{Drafts: 40, Resends: 30}}
				p, err := h.f.openInproc(h.ctx, sp)
				if err != nil {
					t.Fatal(err)
				}
				p.track = true
				run, err := tabletest.Play(h.ctx, p, cfg)
				if err != nil {
					t.Fatal(err)
				}
				su, err := recordSetup(SetupRecord{Seed: seed, Seats: n, Sides: sides})
				if err != nil {
					t.Fatal(err)
				}
				truth, err := leakcheck.FromReal(su, engineBatches(recordBatches(run.Canonical.Batches)))
				if err != nil {
					t.Fatal(err)
				}
				checker := leakcheck.New(truth, frameConfig)
				for id, frames := range p.frames {
					v := frameViewer{Seat: viewerOf(id), Client: string(id), Admin: viewerOf(id) == 0}
					if got := frameLeaks(t, checker, string(id), v, frames, nil, p.wants[id]); len(got) > 0 {
						t.Fatal(got[:min(5, len(got))])
					}
				}
				a, _ := p.actor()
				r := a.st.rec.clone()
				restored, hashOK, err := restore(&r, defaultBounds)
				if err != nil || !hashOK {
					t.Fatalf("schema-2 replay: %v hash=%v", err, hashOK)
				}
				for s := -1; s < n; s++ {
					v := spectatorViewer()
					if s >= 0 {
						v = seatViewer(s)
					}
					if !reflect.DeepEqual(project(&a.st.game, v), project(&restored.game, v)) || !reflect.DeepEqual(history(&a.st.game, v, nil, ""), history(&restored.game, v, nil, "")) {
						t.Fatal("replay frame or history changed")
					}
				}

			})
		}
	}
}

func recordBatches(bs []tabletest.Batch) []BatchRecord {
	out := make([]BatchRecord, 0, len(bs))
	for _, b := range bs {
		br := BatchRecord{Point: b.Point, Picks: []PickRecord{}}
		for _, p := range b.Picks {
			br.Picks = append(br.Picks, PickRecord{Seat: p.Seat, Key: p.Key})
		}
		out = append(out, br)
	}
	return out
}

func TestEngineContractLegacyRules(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 77))
	a, _ := h.p.actor()
	var r Record
	if err := a.call(h.ctx, func() { r = a.st.rec.clone() }); err != nil {
		t.Fatal(err)
	}
	// Development records must never be replayed under unrelated real rules.
	r.Rules = 1000
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	verdict, st, err := assess(data, defaultBounds)
	if err != nil || st != nil || verdict.Archive != store.ArchiveRulesVersion {
		t.Fatalf("fake rules must archive explicitly: %+v %v", verdict, err)
	}
	r.Rules = RulesVersion
	r.Setup.Sides = "unknown"
	data, _ = r.encode()
	verdict, _, err = assess(data, defaultBounds)
	if err != nil || verdict.Archive != store.ArchiveReplayFailed {
		t.Fatal("unknown setup was accepted")
	}
}

// Report actual retained actor/connection heap and the last-confirmation
// transaction time (store write plus all eight real-view projections). The
// latter includes the in-process driver's dispatch/drain overhead and is an
// upper bound on the actor work, not a pure Resolve microbenchmark.
func TestEngineContractMeasurements(t *testing.T) {
	f := newFixture(t, Config{})
	ctx := testContextFor(t, 2*time.Minute)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	tables := make([]*inproc, 0, 32)
	for i := range 32 {
		p, err := f.openInproc(ctx, humans(7, uint64(9000+i)))
		if err != nil {
			t.Fatal(err)
		}
		p.frames = nil
		p.viewers = nil
		p.wants = nil
		tables = append(tables, p)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(tables)
	heap := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	t.Logf("seven-seat retained heap: %d bytes/table (%d tables; empty outboxes, 7 humans + spectator); rules.Game=%d bytes", heap/32, 32, unsafe.Sizeof(rules.Game{}))
	p, err := f.openInproc(ctx, humans(7, 9081))
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(9081, 77))
	seqs := make([]uint64, 7)
	var elapsed time.Duration
	batches := 0
	maxKeys := 0
	for {
		sn, err := p.Snapshot(ctx, tabletest.SeatClient(0))
		if err != nil {
			t.Fatal(err)
		}
		if sn.Over {
			break
		}
		for _, s := range sn.Required {
			id := tabletest.SeatClient(int(s))
			fr := p.frames[id][len(p.frames[id])-1]
			choices := fr.View.You.Choices
			maxKeys = max(maxKeys, len(choices))
			if len(choices) > rules.MaxTurnChoices {
				t.Fatal("offer exceeds proven bound")
			}
			payments := map[string]int{}
			for _, ch := range choices {
				if len(ch.Key) > rules.MaxKeyBytes {
					t.Fatal("key exceeds bound")
				}
				if ch.Total != ch.Left+ch.Right+ch.Printed {
					t.Fatal("payment arithmetic changed")
				}
				payments[ch.Kind+":"+ch.Design]++
			}
			for _, count := range payments {
				if count > rules.MaxPaymentOutcomes {
					t.Fatal("construction frontier exceeds bound")
				}
			}
			cur, _ := p.Snapshot(ctx, id)
			seqs[s]++
			key := cur.Legal[rng.IntN(len(cur.Legal))]
			start := time.Now()
			out, err := p.Submit(ctx, id, tabletest.Submission{Seq: seqs[s], Ref: cur.Ref, Key: key, Confirm: true})
			if err != nil {
				t.Fatal(err)
			}
			if out == tabletest.Resolved {
				elapsed += time.Since(start)
				batches++
			}
		}
	}
	if batches < 18 || rules.MaxPaymentOutcomes != 2*catalog.MaxCostUnits+1 || rules.MaxTurnChoices != 14*rules.MaxPaymentOutcomes+7 {
		t.Fatal("contract bounds or completed game")
	}
	t.Logf("seven-seat real batch: %s average over %d resolved batches; largest observed offer=%d; proven frontier=%d, turn offer=%d", elapsed/time.Duration(batches), batches, maxKeys, rules.MaxPaymentOutcomes, rules.MaxTurnChoices)
}

// A natural, replayable turn-six prefix puts both abilities on the actor's
// path. No scenario state or arbitrary engine checkpoint enters persistence.
func TestEngineContractAbilityOrder(t *testing.T) {
	var prefix []BatchRecord
	var chosen batch
	var seed uint64
	found := false
	for candidate := uint64(0); candidate < 128 && !found; candidate++ {
		su := makeDaySetup(candidate, 7, false, false)
		su.Options.Sides = rules.SidesChoose
		g, err := newGame(su)
		if err != nil {
			t.Fatal(err)
		}
		bs := []BatchRecord{}
		for !pointOver(g.Point()) {
			pt := g.Point()
			b := batch{Point: pt.ID}
			br := BatchRecord{Point: pt.ID, Picks: []PickRecord{}}
			halik, babylon := -1, -1
			for s := 0; s < 7; s++ {
				switch g.State.Cities[s].Board {
				case catalog.BoardHalikarnassos:
					halik = s
				case catalog.BoardBabylon:
					babylon = s
				}
			}
			for _, s := range pt.Required.Seats() {
				cs := g.Legal(s)
				pick := cs[0]
				if pt.Kind == rules.PointSides {
					pick = cs[1]
				} else if pt.Kind == rules.PointTurn {
					// Establish public production first. Babylon builds its ability;
					// Halikarnassos waits until turn six for its first discard build.
					for _, c := range cs {
						if c.Kind == rules.ActBuild {
							pick = c
							break
						}
					}
					for _, c := range cs {
						d, _ := catalog.DesignByID(c.Design)
						if c.Kind == rules.ActBuild && (d.Color == catalog.Brown || d.Color == catalog.Grey) {
							pick = c
							break
						}
					}
					for _, c := range cs {
						if c.Kind == rules.ActStage && ((int(s) == babylon && g.State.Cities[s].Stages < 2) || (int(s) == halik && g.State.Turn == 6)) {
							pick = c
							break
						}
					}
				}
				b.Picks = append(b.Picks, rules.Pick{Seat: s, Key: pick.Key()})
				br.Picks = append(br.Picks, PickRecord{Seat: int(s), Key: pick.Key()})
			}
			n, _, err := g.Resolve(b)
			if err != nil {
				t.Fatal(err)
			}
			if pt.Kind == rules.PointTurn && g.State.Turn == 6 && n.Point().Kind == rules.PointDiscardBuild && n.State.Cities[babylon].Kept != 0 {
				seed, prefix, chosen, found = candidate, bs, b, true
				break
			}
			g = n
			bs = append(bs, br)
		}
	}
	if !found {
		t.Fatal("no natural sixth-turn discard/kept-card prefix")
	}
	sp := humans(7, seed)
	sp.Sides = "choose"
	sp.Prefix = prefix
	h := newTH(t, Config{}, sp)
	for _, p := range chosen.Picks {
		sn := h.snap(tabletest.SeatClient(int(p.Seat)))
		_, err := h.p.Submit(h.ctx, tabletest.SeatClient(int(p.Seat)), tabletest.Submission{Seq: 1, Ref: sn.Ref, Key: p.Key, Confirm: true})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"discard-build", "last-card"} {
		a, _ := h.p.actor()
		var r Record
		if err := a.call(h.ctx, func() { r = a.st.rec.clone() }); err != nil {
			t.Fatal(err)
		}
		restored, ok, err := restore(&r, defaultBounds)
		if err != nil || !ok || pointKind(restored.game.Point()) != want {
			t.Fatalf("%s replay: %v", want, err)
		}
		pt := restored.game.Point()
		if pt.Required.Count() != 1 {
			t.Fatal("ability is not a one-seat point")
		}
		for _, s := range pt.Required.Seats() {
			id := tabletest.SeatClient(int(s))
			sn := h.snap(id)
			_, err := h.p.Submit(h.ctx, id, tabletest.Submission{Seq: 2 + uint64(len(r.Batches)), Ref: sn.Ref, Key: sn.Legal[0], Confirm: true})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if fr := h.latest(tabletest.SeatClient(0)); fr.View.Age < 2 || fr.Point.Kind != "turn" {
		t.Fatal("age did not advance after ability queue")
	}
	t.Logf("natural seed %d, %d prefix batches: turn six -> discard build -> last card -> next Age", seed, len(prefix))
}
