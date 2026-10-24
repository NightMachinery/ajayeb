package tableroom

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"runtime"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestLogDownloadEverySideRule(t *testing.T) {
	for _, sides := range []string{"", "day", "random", "choose"} {
		t.Run("sides="+sides, func(t *testing.T) {
			f := newFixture(t, Config{})
			ctx := testContextFor(t, 2*time.Minute)
			sp := humans(3, 4242)
			sp.Sides = sides
			p, err := f.openInproc(ctx, sp)
			if err != nil {
				t.Fatal(err)
			}
			run, err := tabletest.Play(ctx, p, tabletest.Config{Seats: 3, PickSeed: 4242})
			if err != nil {
				t.Fatal(err)
			}
			p.drain()
			frames := p.frames[tabletest.SeatClient(0)]
			last := frames[len(frames)-1]
			if last.Status != "finished" || !last.View.CanDownload {
				t.Fatal("finished revealed game must offer its log")
			}
			l, err := f.tables.ReadTableLog(p.code)
			if err != nil {
				t.Fatal(err)
			}
			wantSide := sides
			if wantSide == "" {
				wantSide = "day"
			}
			if l.Setup.Sides != wantSide || l.Hash != run.Canonical.Hash || !bytes.Equal(canonicalOf(&Record{Batches: l.Batches, Hash: l.Hash}).Bytes(), run.Canonical.Bytes()) {
				t.Fatal("download changed effective setup or canonical game")
			}
			// An unknown persisted rule must not silently fall back to day.
			a, err := p.actor()
			if err != nil {
				t.Fatal(err)
			}
			if err := a.call(ctx, func() {
				bad := a.st.rec.clone()
				bad.Setup.Sides = "unknown"
				data, err := json.Marshal(bad)
				if err != nil {
					t.Error(err)
					return
				}
				if err := f.tables.store().SaveGame(p.code, Game, data); err != nil {
					t.Error(err)
				}
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := f.tables.ReadTableLog(p.code); err != ErrLogReplay {
				t.Fatalf("unknown side: %v, want ErrLogReplay", err)
			}
		})
	}
}

func TestHistoryErrorLogIsSeedless(t *testing.T) {
	var captured bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&captured, nil))
	g := game{}
	if got := history(&g, spectatorViewer(), log, "HSTAAA"); len(got) != 0 {
		t.Fatal("invalid history must return empty")
	}
	var entry map[string]any
	if err := json.Unmarshal(captured.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["msg"] != "table history failed" || entry["room"] != "HSTAAA" || entry["viewer"] != float64(-1) {
		t.Fatal(entry)
	}
	for k := range entry {
		switch k {
		case "time", "level", "msg", "room", "viewer":
		default:
			t.Fatalf("history diagnostic contains unexpected field %q", k)
		}
	}
}

func TestEngineContractFinishedMemory(t *testing.T) {
	f := newFixture(t, Config{})
	ctx := testContextFor(t, 5*time.Minute)
	count := 8
	if RaceEnabled {
		// The measurement is the plain build's (the race detector's shadow
		// memory inflates the heap); two tables keep the caches' checks.
		count = 2
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	keep := make([]*inproc, 0, count)
	cached, largest := 0, 0
	for i := range count {
		seed := uint64(9300 + i)
		p, err := f.openInproc(ctx, humans(7, seed))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tabletest.Play(ctx, p, tabletest.Config{Seats: 7, PickSeed: seed}); err != nil {
			t.Fatal(err)
		}
		a, err := p.actor()
		if err != nil {
			t.Fatal(err)
		}
		for _, cl := range p.clients {
			if err := a.call(ctx, func() { a.inbound(cl, wire.NewResync(p.code)) }); err != nil {
				t.Fatal(err)
			}
		}
		p.drain()
		if err := a.call(ctx, func() {
			if a.st.rec.Status != store.StatusFinished || len(a.engineViews) != 8 {
				t.Error("all eight finished role caches must be populated")
			}
			for _, v := range a.engineViews {
				if v.history == nil {
					t.Error("resync did not cache history")
					continue
				}
				view, err := json.Marshal(v.proj.view)
				if err != nil {
					t.Error(err)
				}
				hist, err := json.Marshal(*v.history)
				if err != nil {
					t.Error(err)
				}
				one := len(view) + len(hist)
				cached += one
				largest = max(largest, one)
			}
		}); err != nil {
			t.Fatal(err)
		}
		p.frames, p.viewers, p.wants, p.rejects, p.other = nil, nil, nil, nil, nil
		keep = append(keep, p)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(keep)
	t.Logf("finished seven-seat tables: %d retained heap bytes/table (%d tables, eight roles resynced, drained outboxes); cached JSON %d bytes/table; largest one-role view+history %d bytes", (int64(after.HeapAlloc)-int64(before.HeapAlloc))/int64(count), count, cached/count, largest)
}
