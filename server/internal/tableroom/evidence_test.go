package tableroom

// Opt-in diagnostics for the internal baseline milestone. Only aggregates of
// times, sizes and counters leave this harness, never keys or private views.
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type evidenceQuantiles struct {
	N     int   `json:"n"`
	P50   int64 `json:"p50"`
	P95   int64 `json:"p95"`
	P99   int64 `json:"p99"`
	Max   int64 `json:"max"`
	Total int64 `json:"total"`
}

func evidenceQuantify(values []int64) evidenceQuantiles {
	v := slices.Clone(values)
	slices.Sort(v)
	if len(v) == 0 {
		return evidenceQuantiles{}
	}
	q := func(p int) int64 { return v[max(0, (len(v)*p+99)/100-1)] }
	out := evidenceQuantiles{N: len(v), P50: q(50), P95: q(95), P99: q(99), Max: v[len(v)-1]}
	for _, x := range v {
		out.Total += x
	}
	return out
}

type evidenceTask struct {
	point string
	seat  int
}

type evidenceStats struct {
	mu        sync.Mutex
	values    map[string][]int64
	starts    map[evidenceTask]time.Time
	fallbacks int
	cancelled int
}

func newEvidenceStats() *evidenceStats {
	return &evidenceStats{values: map[string][]int64{}, starts: map[evidenceTask]time.Time{}}
}
func (e *evidenceStats) add(name string, x int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.values[name] = append(e.values[name], x)
}
func (e *evidenceStats) hooks() (func(context.Context, int, string) error, func(BotReport)) {
	return func(_ context.Context, seat int, point string) error {
			now := time.Now()
			e.mu.Lock()
			e.starts[evidenceTask{point, seat}] = now
			e.mu.Unlock()
			return nil
		}, func(r BotReport) {
			now := time.Now()
			kind := "turn"
			switch {
			case r.Point == "sides":
				kind = "sides"
			case strings.Contains(r.Point, "discard-build"):
				kind = "discard-build"
			case strings.Contains(r.Point, "last-card"):
				kind = "last-card"
			}
			e.mu.Lock()
			defer e.mu.Unlock()
			if r.Cancelled {
				e.cancelled++
				return
			}
			if r.Fallback != "" {
				e.fallbacks++
			}
			e.values["active_ns."+kind] = append(e.values["active_ns."+kind], int64(r.Active))
			e.values["pool_wait_ns."+kind] = append(e.values["pool_wait_ns."+kind], int64(r.PoolWait))
			key := evidenceTask{r.Point, r.Seat}
			if start, ok := e.starts[key]; ok {
				e.values["pool_hold_bracket_ns."+kind] = append(e.values["pool_hold_bracket_ns."+kind], int64(now.Sub(start)))
				delete(e.starts, key)
			}
		}
}
func (e *evidenceStats) log(t *testing.T, label string) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	summary := map[string]evidenceQuantiles{}
	for k, v := range e.values {
		summary[k] = evidenceQuantify(v)
	}
	data, err := json.Marshal(struct {
		Case      string                       `json:"case"`
		Metrics   map[string]evidenceQuantiles `json:"metrics"`
		Fallbacks int                          `json:"fallbacks"`
		Cancelled int                          `json:"cancelled"`
	}{label, summary, e.fallbacks, e.cancelled})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("room-evidence %s", data)
}

type evidenceReveal struct {
	batch    int
	accepted []time.Time
	seen     bool
}
type evidenceStore struct {
	Storage
	stats    *evidenceStats
	mu       sync.Mutex
	accepted map[string]map[int]time.Time
	resolved map[string]int
	reveals  map[string][]evidenceReveal
}

func newEvidenceStore(s Storage, e *evidenceStats) *evidenceStore {
	return &evidenceStore{Storage: s, stats: e, accepted: map[string]map[int]time.Time{}, resolved: map[string]int{}, reveals: map[string][]evidenceReveal{}}
}
func (s *evidenceStore) CreateGame(code, game string, data []byte) ([]byte, error) {
	out, err := s.Storage.CreateGame(code, game, data)
	if err != nil {
		return out, err
	}
	var r Record
	if err = json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.resolved[code] = len(r.Batches)
	s.mu.Unlock()
	return out, nil
}
func (s *evidenceStore) SaveGame(code, game string, data []byte) error {
	start := time.Now()
	err := s.Storage.SaveGame(code, game, data)
	saved := time.Now()
	s.stats.add("store_save_ns", int64(saved.Sub(start)))
	if err != nil {
		return err
	}
	var r Record
	if err = json.Unmarshal(data, &r); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	remember := func(point string, seat int) {
		k := code + "/" + point
		if s.accepted[k] == nil {
			s.accepted[k] = map[int]time.Time{}
		}
		if _, ok := s.accepted[k][seat]; !ok {
			s.accepted[k][seat] = saved
		}
	}
	if r.Point != nil {
		for _, p := range r.Pending {
			if p.Confirmed {
				remember(r.Point.ID, p.Seat)
			}
		}
	}
	for i := s.resolved[code]; i < len(r.Batches); i++ {
		b := r.Batches[i]
		reveal := evidenceReveal{batch: i + 1}
		for _, p := range b.Picks {
			remember(b.Point, p.Seat)
			reveal.accepted = append(reveal.accepted, s.accepted[code+"/"+b.Point][p.Seat])
		}
		s.reveals[code] = append(s.reveals[code], reveal)
	}
	s.resolved[code] = len(r.Batches)
	return nil
}
func (s *evidenceStore) seen(code string, batch int, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.reveals[code] {
		r := &s.reveals[code][i]
		if !r.seen && r.batch <= batch {
			r.seen = true
			for _, a := range r.accepted {
				s.stats.add("accepted_to_outbox_ns", int64(now.Sub(a)))
			}
		}
	}
}

// evidencePlay uses actual actor submissions, persistence, pool work and
// outboxes. Marshal time is timed separately after receipt, not in transport.
func evidencePlay(ctx context.Context, h *th, e *evidenceStats, s *evidenceStore) error {
	seq := map[int]uint64{}
	last := map[tabletest.ClientID]wire.TableState{}
	for {
		h.p.drain()
		for id, frames := range h.p.frames {
			for _, fr := range frames {
				now := time.Now()
				start := time.Now()
				data, err := json.Marshal(fr)
				if err != nil {
					return err
				}
				e.add("frame_marshal_ns", int64(time.Since(start)))
				e.add("frame_bytes", int64(len(data)))
				if id == tabletest.SpectatorClient {
					s.seen(h.p.code, fr.Batch, now)
				}
				last[id] = fr
			}
			h.p.frames[id] = nil
			h.p.viewers[id] = nil
			h.p.wants[id] = nil
		}
		watcher := last[tabletest.SpectatorClient]
		if watcher.Status == "finished" {
			return nil
		}
		submitted := false
		for id, fr := range last {
			if fr.Point == nil || fr.You.Seat == nil || fr.Pending != nil || fr.View.You == nil || len(fr.View.You.Choices) == 0 {
				continue
			}
			seat := *fr.You.Seat
			if !slices.Contains(fr.Point.Required, seat) {
				continue
			}
			seq[seat]++
			start := time.Now()
			out, err := h.p.Submit(ctx, id, tabletest.Submission{Seq: seq[seat], Ref: refString(fr.Point.Ref), Key: fr.View.You.Choices[0].Key, Confirm: true})
			e.add("human_submit_ns", int64(time.Since(start)))
			if err != nil {
				return err
			}
			if out != tabletest.Accepted && out != tabletest.Resolved {
				return fmt.Errorf("unexpected human outcome %s", out)
			}
			submitted = true
			break
		}
		if submitted {
			continue
		}
		select {
		case <-h.p.clients[tabletest.SpectatorClient].Out.Ready():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func evidenceEnabled() bool { return os.Getenv("M17_ROOM_EVIDENCE") == "1" }

func TestRoomEvidence(t *testing.T) {
	if !evidenceEnabled() {
		t.Skip("opt-in room measurements")
	}
	t.Logf("measurement environment go=%s os=%s arch=%s cpus=%d gomaxprocs=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.GOMAXPROCS(0))
	cases := []struct {
		name, level string
		mix, all    bool
		kind        string
	}{
		{name: "six-easy", level: BotEasy}, {name: "six-normal", level: BotNormal}, {name: "six-hard", level: BotHard},
		{name: "six-mixed-choose", level: BotHard, mix: true, kind: "sides"}, {name: "all-bot-continuation", level: BotHard, mix: true, all: true},
		{name: "discard-build", level: BotHard, mix: true, kind: "discard-build"}, {name: "last-card", level: BotHard, mix: true, kind: "last-card"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newEvidenceStats()
			work, done := e.hooks()
			pool := ai.NewPool(1)
			f := newFixture(t, Config{Pool: pool, BotWork: work, BotFinished: done})
			s := newEvidenceStore(f.store, e)
			f.tables.Bind(f.m, s)
			sp := soloSpec(7, 942, tc.level)
			human := 0
			if tc.mix {
				for seat := 1; seat < 7; seat++ {
					sp.Participants[seat].Bot = []string{BotEasy, BotNormal, BotHard}[seat%3]
				}
			}
			if tc.kind == "sides" {
				sp.Sides = "choose"
			}
			if tc.kind == "discard-build" || tc.kind == "last-card" {
				kind := rulesPointKind(tc.kind)
				b := boundaryPrefix(7, kind, 0)
				if !b.ok {
					t.Fatal("missing natural ability prefix")
				}
				su, err := recordSetup(SetupRecord{Seed: b.seed, Seats: 7, Sides: b.sides})
				if err != nil {
					t.Fatal(err)
				}
				g, err := replay(su, engineBatches(b.prefix))
				if err != nil {
					t.Fatal(err)
				}
				human = (int(g.Point().Required.Seats()[0]) + 1) % 7
				sp = humans(7, b.seed)
				sp.Creator = userOf(human)
				sp.Prefix = b.prefix
				sp.Sides = b.sides
				for seat := 0; seat < 7; seat++ {
					if seat != human {
						sp.Participants[seat] = Participant{Bot: []string{BotEasy, BotNormal, BotHard}[seat%3]}
					}
				}
			}
			ctx := testContextFor(t, 5*time.Minute)
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			opened := time.Now()
			p, err := f.openInproc(ctx, sp)
			if err != nil {
				t.Fatal(err)
			}
			e.add("create_join_ns", int64(time.Since(opened)))
			h := &th{t: t, ctx: ctx, f: f, p: p}
			if tc.all {
				a, err := p.actor()
				if err != nil {
					t.Fatal(err)
				}
				if err := a.call(ctx, func() {
					a.inbound(p.clients[tabletest.SeatClient(human)], wire.TableResign{T: "table-resign", Code: p.code, Seq: 1, Confirm: true})
				}); err != nil {
					t.Fatal(err)
				}
			}
			if err := evidencePlay(ctx, h, e, s); err != nil {
				t.Fatal(err)
			}
			a, err := p.actor()
			if err != nil {
				t.Fatal(err)
			}
			if err := a.call(ctx, func() {
				if len(a.st.rec.BotFallbacks) != 0 {
					t.Error("ordinary watchdog")
				}
				picks := 0
				for _, b := range a.st.rec.Batches[len(sp.Prefix):] {
					picks += len(b.Picks)
				}
				e.mu.Lock()
				measured := len(e.values["accepted_to_outbox_ns"])
				e.mu.Unlock()
				if measured != picks {
					t.Errorf("receipt samples=%d, accepted picks=%d", measured, picks)
				}
			}); err != nil {
				t.Fatal(err)
			}
			runtime.ReadMemStats(&after)
			e.add("allocated_bytes_game", int64(after.TotalAlloc-before.TotalAlloc))
			e.add("allocations_game", int64(after.Mallocs-before.Mallocs))
			p.frames = nil
			p.viewers = nil
			p.wants = nil
			runtime.GC()
			runtime.ReadMemStats(&after)
			runtime.KeepAlive(p)
			e.add("retained_heap_delta_bytes_game", int64(after.HeapAlloc)-int64(before.HeapAlloc))
			p.frames = map[tabletest.ClientID][]wire.TableState{}
			coldStart := time.Now()
			f.restart()
			h.rejoin()
			e.add("cold_finished_start_join_ns", time.Since(coldStart).Nanoseconds())
			if pool.InUse() != 0 || pool.Waiting() != 0 {
				t.Fatal("permit leak")
			}
			e.log(t, tc.name)
		})
	}
}

func rulesPointKind(kind string) rules.PointKind {
	if kind == "discard-build" {
		return rules.PointDiscardBuild
	}
	return rules.PointLastCard
}

// Quantiles use nearest rank and never interpolate a tiny sample into a tail.
func TestEvidenceQuantiles(t *testing.T) {
	if got := evidenceQuantify([]int64{4, 1, 3, 2}); got.N != 4 || got.P50 != 2 || got.P95 != 4 || got.P99 != 4 || got.Total != 10 {
		t.Fatal(got)
	}
	if got := evidenceQuantify(nil); got.N != 0 {
		t.Fatal(got)
	}
}
