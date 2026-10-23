package tableroom

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// frameConfig allows a frame's own legitimate paths: the viewer's offered
// keys (in the frame and in the view rebuilt from it) and own pending pick,
// display names, and user and client ids.
var frameConfig = leakcheck.Config{
	KeyPaths:  []string{"view.you.choices[].key", "you.choices[].key", "pending.key"},
	TextPaths: []string{"humans[].name"},
	HexPaths:  []string{"seats[].userId", "humans[].userId", "you.userId", "ack.clientId", "roomLog[].by", "roomLog[].userId"},
}

// leakDocs turns one viewer's frames into the oracle's documents: the
// whole frame, its engine view rebuilt in the fake view's shape (the
// frame's view plus its point and, as the fake view carries them, the
// latest resolution's events), and its history.
func leakDocs(t testing.TB, label string, viewer int, frames []wire.TableState, truth *leakcheck.Truth) []leakcheck.Doc {
	t.Helper()
	var docs []leakcheck.Doc
	events := []wire.TableEvent{}
	batch := 0
	for i, fr := range frames {
		raw, err := json.Marshal(fr)
		if err != nil {
			t.Fatal(err)
		}
		where := fmt.Sprintf("%s frame %d", label, i)
		docs = append(docs, leakcheck.Doc{Label: where, Kind: leakcheck.Frame, Viewer: viewer, Data: raw})
		switch {
		case fr.History != nil:
			// A join, resync or undo frame: the latest entry of its history.
			events = lastEvents(fr)
		case fr.Batch != batch:
			// A resolution frame carries that resolution's events.
			events = fr.Events
		}
		batch = fr.Batch
		docs = append(docs, leakcheck.Doc{Label: where + " view", Kind: leakcheck.View, Viewer: viewer, Moment: fr.Batch,
			Data: fakeViewJSON(t, fr, events)})
		if fr.History != nil {
			var h []map[string]any
			for _, e := range *fr.History {
				h = append(h, map[string]any{"point": truth.Moments[e.Batch-1].Point, "events": e.Events})
			}
			if h == nil {
				h = []map[string]any{}
			}
			data, _ := json.Marshal(h)
			docs = append(docs, leakcheck.Doc{Label: where + " history", Kind: leakcheck.History, Viewer: viewer, Moment: fr.Batch, Data: data})
		}
	}
	return docs
}

func lastEvents(fr wire.TableState) []wire.TableEvent {
	if fr.History == nil || len(*fr.History) == 0 {
		return []wire.TableEvent{}
	}
	return (*fr.History)[len(*fr.History)-1].Events
}

// fakeViewJSON rebuilds the fake engine's view from a frame.
func fakeViewJSON(t testing.TB, fr wire.TableState, events []wire.TableEvent) []byte {
	t.Helper()
	data, err := json.Marshal(fr.View)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	m["point"], m["kind"], m["required"] = "", fakeengine.PointNone.String(), []int{}
	if fr.Point != nil {
		m["point"], m["kind"], m["required"] = fr.Point.Ref.ID, fr.Point.Kind, fr.Point.Required
	}
	m["events"] = events
	out, _ := json.Marshal(m)
	return out
}

// playForLeaks plays one in-process game with drafts and resends and
// returns the table, its truth and every client's frames.
func playForLeaks(t *testing.T, f *fixture, n int, seed uint64, hide bool) (*inproc, *leakcheck.Truth) {
	t.Helper()
	ctx := testContext(t)
	sp := humans(n, seed)
	sp.HideAtEnd = hide
	p, err := f.openInproc(ctx, sp)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tabletest.Play(ctx, p, tabletest.Config{Seats: n, PickSeed: seed, ArrivalSeed: seed + 1,
		Script: tabletest.ScriptOptions{Drafts: 60, Resends: 30}}); err != nil {
		t.Fatal(err)
	}
	// A late resync: the whole safe history again.
	for _, c := range p.clients {
		a, _ := p.actor()
		if err := a.call(ctx, func() { a.inbound(c, wire.NewResync(p.code)) }); err != nil {
			t.Fatal(err)
		}
	}
	p.drain()
	return p, truthOf(t, p, n, seed, hide)
}

func truthOf(t *testing.T, p *inproc, n int, seed uint64, hide bool) *leakcheck.Truth {
	t.Helper()
	can, err := p.Canonical(testContext(t))
	if err != nil {
		t.Fatal(err)
	}
	return truthFrom(t, can, n, seed, hide)
}

// truthFrom is the oracle's ground truth of a game played to can's
// batches, under chosen seating.
func truthFrom(t testing.TB, can tabletest.Canonical, n int, seed uint64, hide bool) *leakcheck.Truth {
	t.Helper()
	var bs []fakeengine.Batch
	for _, b := range can.Batches {
		eb := fakeengine.Batch{Point: b.Point}
		for _, pk := range b.Picks {
			eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: fakeengine.Seat(pk.Seat), Key: pk.Key})
		}
		bs = append(bs, eb)
	}
	truth, err := leakcheck.FromFake(makeSetup(seed, n, false, hide), bs)
	if err != nil {
		t.Fatal(err)
	}
	return truth
}

func viewerOf(id tabletest.ClientID) int {
	if id == tabletest.SpectatorClient {
		return leakcheck.Spectator
	}
	s, _ := strconv.Atoi(strings.TrimPrefix(string(id), "seat-"))
	return s
}

// TestFramesLeakNothing: every frame every seat and the spectator received
// over whole games at 3 to 7 seats, under both end policies, with live
// events, history on join and resync, passes the oracle.
func TestFramesLeakNothing(t *testing.T) {
	for n := MinSeats; n <= MaxSeats; n++ {
		for _, hide := range []bool{false, true} {
			f := newFixture(t, Config{})
			seed := uint64(500 + n)
			p, truth := playForLeaks(t, f, n, seed, hide)
			c := leakcheck.New(truth, frameConfig)
			total := 0
			for id, frames := range p.frames {
				docs := leakDocs(t, fmt.Sprintf("%d seats hide=%t %s", n, hide, id), viewerOf(id), frames, truth)
				total += len(docs)
				if leaks := c.Check(docs...); len(leaks) != 0 {
					t.Fatalf("%d seats, hide=%t, %s: %v", n, hide, id, leaks[:min(len(leaks), 5)])
				}
			}
			if total < 50 {
				t.Fatalf("%d seats: only %d documents", n, total)
			}
		}
	}
}

// TestPlantedLeaksInFrames: a leak planted in a real frame fails its own
// detector: another seat's pending key or hand card, a raw key in an event,
// a game fingerprint, a copy-suffixed card id, a deal-ordered hand,
// another seat's offer and an unknown field.
func TestPlantedLeaksInFrames(t *testing.T) {
	f := newFixture(t, Config{})
	p, truth := playForLeaks(t, f, 4, 77, false)
	c := leakcheck.New(truth, frameConfig)
	seat0 := p.frames[tabletest.SeatClient(0)]
	// The first frame after the first resolution: seat 0 holds a hand and
	// the frame carries events.
	var base wire.TableState
	var events []wire.TableEvent
	for _, fr := range seat0 {
		if fr.Batch == 1 && len(fr.Events) > 0 {
			base, events = fr, fr.Events
			break
		}
	}
	if base.View.You == nil || len(base.View.You.Hand) < 2 {
		t.Fatal("no frame with a hand after the first resolution")
	}
	clean := func(st wire.TableState, ev []wire.TableEvent) []leakcheck.Doc {
		raw, _ := json.Marshal(st)
		return []leakcheck.Doc{
			{Label: "frame", Kind: leakcheck.Frame, Viewer: 0, Data: raw},
			{Label: "view", Kind: leakcheck.View, Viewer: 0, Moment: 1, Data: fakeViewJSON(t, st, ev)},
		}
	}
	if leaks := c.Check(clean(base, events)...); len(leaks) != 0 {
		t.Fatalf("the unplanted frame: %v", leaks)
	}
	m := &truth.Moments[1]
	other := slices.IndexFunc(m.Hands[1], func(card string) bool { return !slices.Contains(base.View.You.Hand, card) })
	if other < 0 {
		t.Fatal("seat 1 holds no card seat 0 lacks")
	}
	g, _ := replay(makeSetup(77, 4, false, false), nil)
	for _, tc := range []struct {
		name  string
		plant func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent
		rule  leakcheck.Rule
	}{
		{"another seat's hand card", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			st.View.You.Hand = append(slices.Clone(st.View.You.Hand), m.Hands[1][other])
			slices.Sort(st.View.You.Hand)
			return ev
		}, leakcheck.HiddenCard},
		{"a raw key in an event", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			ev = slices.Clone(ev)
			ev[0].Play = "sell:" + m.Hands[1][0]
			st.Events = ev
			return ev
		}, leakcheck.RawKey},
		{"a fingerprint in the frame", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			st.RoomLog = []wire.PlatformLogEntry{{Kind: "seat", By: g.Fingerprint()}}
			return ev
		}, leakcheck.Hash},
		{"a copy-suffixed card id", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			st.View.You.Hand = append([]string{}, st.View.You.Hand...)
			st.View.You.Hand[0] += "#2"
			return ev
		}, leakcheck.ShuffleID},
		{"a deal-ordered hand", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			h := slices.Clone(st.View.You.Hand)
			slices.Reverse(h)
			st.View.You.Hand = h
			return ev
		}, leakcheck.DealOrder},
		{"another seat's offer", func(st *wire.TableState, ev []wire.TableEvent) []wire.TableEvent {
			ch := slices.Clone(st.View.You.Choices)
			for _, k := range m.Legal[1] {
				if !slices.ContainsFunc(ch, func(c wire.TableChoice) bool { return c.Key == k }) {
					ch = append(ch, wire.TableChoice{Key: k, Kind: "sell", Design: strings.TrimPrefix(k, "sell:")})
					break
				}
			}
			st.View.You.Choices = ch
			return ev
		}, leakcheck.ForeignOffer},
	} {
		st := base
		you := *base.View.You
		st.View.You = &you
		ev := tc.plant(&st, events)
		if leaks := c.Check(clean(st, ev)...); !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == tc.rule }) {
			t.Errorf("%s: %s did not report it; got %v", tc.name, tc.rule, leaks)
		}
	}
	// An unknown field in the view.
	data := fakeViewJSON(t, base, events)
	data = append(data[:len(data)-1], []byte(`,"seed":1}`)...)
	if leaks := c.Check(leakcheck.Doc{Label: "unknown", Kind: leakcheck.View, Viewer: 0, Moment: 1, Data: data}); !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == leakcheck.UnknownField }) {
		t.Errorf("an unknown field: got %v", leaks)
	}
}

// TestLogCapture: a whole game's captured log names no key, card, seed,
// fingerprint or observation hash. Hidden sentinels (a distinctive seed,
// every key the game used, each rules hash it had) are searched for
// verbatim, then a planted key and hash must fail the oracle.
func TestLogCapture(t *testing.T) {
	const seed = 0x51ed270b27f3a1c9
	f := newFixture(t, Config{SeqClients: 2})
	p, truth := playForLeaks(t, f, 4, seed, false)
	logs := f.logs.String()
	if strings.Count(logs, "\n") < 20 {
		t.Fatalf("only %d log lines", strings.Count(logs, "\n"))
	}
	c := leakcheck.New(truth, leakcheck.DefaultConfig)
	if leaks := c.Check(leakcheck.Doc{Label: "log", Kind: leakcheck.Log, Data: []byte(logs)}); len(leaks) != 0 {
		t.Fatalf("the log leaks: %v", leaks[:min(5, len(leaks))])
	}
	can, _ := p.Canonical(testContext(t))
	sentinels := []string{strconv.FormatUint(seed, 10), strconv.FormatUint(seed, 16), can.Hash}
	g, _ := replay(makeSetup(seed, 4, false, false), nil)
	for s := range 4 {
		sentinels = append(sentinels, observationHash(&g, s))
	}
	for _, b := range can.Batches {
		for _, pk := range b.Picks {
			sentinels = append(sentinels, pk.Key)
		}
	}
	for _, s := range sentinels {
		if strings.Contains(logs, s) {
			t.Fatalf("the log holds a hidden sentinel %q", s)
		}
	}
	for _, plant := range []struct {
		line string
		rule leakcheck.Rule
	}{
		{`{"msg":"table","key":"` + can.Batches[0].Picks[0].Key + `"}`, leakcheck.RawKey},
		{`{"msg":"table","hash":"` + can.Hash + `"}`, leakcheck.Hash},
	} {
		leaks := c.Check(leakcheck.Doc{Label: "planted", Kind: leakcheck.Log, Data: []byte(plant.line + "\n")})
		if !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == plant.rule }) {
			t.Errorf("%s: %s did not report it; got %v", plant.line, plant.rule, leaks)
		}
	}
}

// TestGameLog: the ajayeb-log/2 download is refused while the game runs
// and under the reveal-off policy, and once over it replays to its hash
// with no seqs, arrivals, work rows or references.
func TestGameLog(t *testing.T) {
	f := newFixture(t, Config{})
	ctx := testContext(t)
	running, err := f.openInproc(ctx, humans(3, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.tables.ReadTableLog(running.code); !errors.Is(err, ErrLogRunning) {
		t.Fatalf("a running table: %v", err)
	}
	if _, err := f.tables.ReadTableLog("ABCDEF"); !errors.Is(err, ErrNotTable) {
		t.Fatalf("a missing code: %v", err)
	}
	p, _ := playForLeaks(t, f, 3, 4, false)
	l, err := f.tables.ReadTableLog(p.code)
	if err != nil {
		t.Fatal(err)
	}
	g, err := replay(makeSetup(l.Setup.Seed, l.Setup.Seats, l.Setup.RandomSeating, l.Setup.HideAtEnd), engineBatches(l.Batches))
	if err != nil || g.Fingerprint() != l.Hash || l.Format != LogFormat || l.Game != Game || l.Rules != RulesVersion {
		t.Fatalf("the log does not replay: %v", err)
	}
	raw, _ := json.Marshal(l)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	if got, want := fieldNames(tree), "batches format game hash hideAtEnd key picks point randomSeating rules seat seats seed setup"; got != want {
		t.Fatalf("the log's fields: %s, want %s", got, want)
	}
	hidden, _ := playForLeaks(t, f, 3, 5, true)
	if _, err := f.tables.ReadTableLog(hidden.code); !errors.Is(err, ErrLogHidden) {
		t.Fatalf("reveal off: %v", err)
	}
}

// fieldNames lists every field name in a JSON tree, sorted and distinct.
func fieldNames(v any) string {
	seen := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, e := range x {
				seen[k] = true
				walk(e)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(v)
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	slices.Sort(out)
	return strings.Join(out, " ")
}
