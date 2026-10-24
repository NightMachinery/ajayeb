package tableroom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// frameConfig allows a frame's own legitimate paths: the viewer's offered
// keys (in the frame and in the view rebuilt from it), display names, and
// user and client ids. The pending pick is not among them: frameLeaks
// leaves it out of the scanned frame only once the envelope check has
// vouched for it.
var frameConfig = leakcheck.Config{
	KeyPaths:  []string{"view.you.choices[].key", "you.choices[].key"},
	TextPaths: []string{"humans[].name"},
	HexPaths:  []string{"seats[].userId", "humans[].userId", "humans[].promotedBy", "you.userId", "you.previousSeats[].userId", "ack.clientId", "roomLog[].by", "roomLog[].userId"},
}

// frameViewer is who received a frame: a seat, or leakcheck.Spectator,
// the connection's client id, which an acknowledgement must name, whether
// its user was an admin then, the only viewers the previous-seat hint may
// reach, and the seat its user last held, if any, the only former seat its
// frames may name.
type frameViewer struct {
	Seat   int
	Client string
	Admin  bool
	Former *int
}

// pendingWant is the pending pick the test's own model says the viewer's
// seat had accepted when a frame was sent. Known is false where the test
// cannot tell, as over the socket.
type pendingWant struct {
	Known   bool
	Pending *wire.TablePending
}

// frameLeaks checks one viewer's frames. Each frame's envelope is checked
// against the viewer and, where wants knows it, the accepted pending pick:
// you names the viewer's seat, an acknowledgement the viewer's own client,
// a spectator gets no pending pick, and a seat only its own (required at
// the point, offered to it, its confirmation the one the public list
// shows). Then the oracle checks the frames' documents. Only the pending
// key the envelope vouched for is left out of the whole-frame scan, and
// only once the raw frame holds it exactly (withoutVouchedKey): every
// other field, the rest of the pending object included, is scanned. raws,
// when given, are the frames as received, scanned instead of the
// re-encoded frames; wants may be nil.
func frameLeaks(t testing.TB, c *leakcheck.Checker, label string, v frameViewer, frames []wire.TableState, raws [][]byte, wants []pendingWant) []leakcheck.Leak {
	t.Helper()
	vs := make([]frameViewer, len(frames))
	for i := range vs {
		vs[i] = v
	}
	return viewerLeaks(t, c, label, vs, frames, raws, wants)
}

// viewerLeaks is frameLeaks with each frame's own viewer, vs[i], for a
// connection whose seat or admin status changed. A frame for another seat
// than the frame before it must carry the whole history, since the
// connection's events so far were redacted for the old seat.
func viewerLeaks(t testing.TB, c *leakcheck.Checker, label string, vs []frameViewer, frames []wire.TableState, raws [][]byte, wants []pendingWant) []leakcheck.Leak {
	t.Helper()
	var leaks []leakcheck.Leak
	scanned := make([][]byte, len(frames))
	seats := make([]int, len(frames))
	for i, fr := range frames {
		var w pendingWant
		if i < len(wants) {
			w = wants[i]
		}
		where := fmt.Sprintf("%s frame %d", label, i)
		bad := envelope(where, vs[i], fr, w, c.Truth)
		if i > 0 && vs[i].Seat != vs[i-1].Seat && fr.History == nil {
			bad = append(bad, leakcheck.Leak{Rule: leakcheck.Envelope, Doc: where, Path: "history", Detail: "a frame for another seat without the history"})
		}
		seats[i] = vs[i].Seat
		leaks = append(leaks, bad...)
		raw, err := json.Marshal(fr)
		if err != nil {
			t.Fatal(err)
		}
		if i < len(raws) {
			raw = raws[i]
		}
		if len(bad) == 0 && fr.Pending != nil {
			stripped, problem := withoutVouchedKey(raw, *fr.Pending)
			if problem != "" {
				leaks = append(leaks, leakcheck.Leak{Rule: leakcheck.Envelope, Doc: where, Path: "pending", Detail: problem})
			} else {
				raw = stripped
			}
		}
		scanned[i] = raw
	}
	return append(leaks, c.Check(leakDocs(t, label, seats, frames, scanned, c.Truth)...)...)
}

// envelope checks a frame's private fields against its viewer.
func envelope(where string, v frameViewer, fr wire.TableState, w pendingWant, truth *leakcheck.Truth) []leakcheck.Leak {
	var out []leakcheck.Leak
	bad := func(path, detail string) {
		out = append(out, leakcheck.Leak{Rule: leakcheck.Envelope, Doc: where, Path: path, Detail: detail})
	}
	switch {
	case v.Seat < 0 && fr.You.Seat != nil:
		bad("you.seat", "a seat in a spectator's frame")
	case v.Seat >= 0 && (fr.You.Seat == nil || *fr.You.Seat != v.Seat):
		bad("you.seat", fmt.Sprintf("not the viewer's seat %d", v.Seat))
	}
	if fr.Ack != nil && fr.Ack.ClientID != v.Client {
		bad("ack.clientId", "another connection's acknowledgement")
	}
	if ps := fr.You.PreviousSeats; ps != nil && !v.Admin {
		bad("you.previousSeats", "the previous-seat hint in a non-admin's frame")
	} else if ps != nil {
		for _, p := range *ps {
			if slices.ContainsFunc(fr.Seats, func(s wire.PlatformSeat) bool { return s.UserID != nil && *s.UserID == p.UserID }) {
				bad("you.previousSeats", "a seated user in the previous-seat hint")
			}
		}
	}
	var former *int
	if v.Seat < 0 && fr.Status == "playing" {
		former = v.Former
	}
	if (fr.You.FormerSeat == nil) != (former == nil) || (former != nil && *fr.You.FormerSeat != *former) {
		bad("you.formerSeat", "not the seat this unseated viewer held last")
	}
	if fr.You.CanPromote != nil && !v.Admin {
		bad("you.canPromote", "the admin cap in a non-admin's frame")
	}
	if pd := fr.Pending; pd != nil {
		switch {
		case v.Seat < 0:
			bad("pending", "a pending pick in a spectator's frame")
		case fr.Point == nil || !slices.Contains(fr.Point.Required, v.Seat):
			bad("pending", "a pending pick while the viewer's seat is not required")
		case fr.Batch < 0 || fr.Batch >= len(truth.Moments) || !slices.Contains(truth.Moments[fr.Batch].Legal[v.Seat], pd.Key):
			bad("pending.key", "a key not offered to the viewer's seat")
		case pd.Confirm != slices.Contains(fr.Point.Confirmed, v.Seat):
			bad("pending.confirm", "not the confirmation the public list shows")
		}
	}
	if w.Known && !samePending(fr.Pending, w.Pending) {
		bad("pending", "not the seat's accepted pick")
	}
	return out
}

func samePending(a, b *wire.TablePending) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

// leakDocs turns one viewer's frames into the oracle's documents: the
// whole frame (raws[i], as frameLeaks prepared it), its engine
// view rebuilt in the engine view's shape (the frame's view plus its point
// and, as the engine view carries them, the latest resolution's events, or
// every resolution's since the viewer's previous frame where an outbox
// merged frames), the frame's own events wherever it carries events the
// rebuilt view did not take from it, and its history. Frame i's viewer is
// seats[i].
func leakDocs(t testing.TB, label string, seats []int, frames []wire.TableState, raws [][]byte, truth *leakcheck.Truth) []leakcheck.Doc {
	t.Helper()
	var docs []leakcheck.Doc
	events := []wire.TableEvent{}
	batch, span := 0, 1 // events are those of the span resolutions up to batch
	for i, fr := range frames {
		raw, err := json.Marshal(fr)
		if err != nil {
			t.Fatal(err)
		}
		if i < len(raws) {
			raw = raws[i]
		}
		where := fmt.Sprintf("%s frame %d", label, i)
		viewer := seats[i]
		docs = append(docs, leakcheck.Doc{Label: where, Kind: leakcheck.Frame, Viewer: viewer, Data: raw})
		own := true // the rebuilt view takes its events from this frame
		switch {
		case fr.History != nil && fr.Batch == historyBatch(fr):
			// A join, resync or undo frame: the latest entry of its history.
			events, span, own = lastEvents(fr), 1, false
		case fr.History != nil:
			// A history frame an outbox merged with the live frames after
			// it: its events follow the history's last entry.
			events, span = fr.Events, fr.Batch-historyBatch(fr)
		case fr.Batch > batch:
			// A resolution frame carries that resolution's events, or, where
			// an outbox merged frames, every resolution's since the last.
			events, span = fr.Events, fr.Batch-batch
		case fr.Batch != batch:
			events, span = fr.Events, 1
		default:
			// The view still carries the events it took last.
			own = false
		}
		batch = fr.Batch
		docs = append(docs, leakcheck.Doc{Label: where + " view", Kind: leakcheck.View, Viewer: viewer, Moment: fr.Batch, Span: span,
			Data: engineViewJSON(t, fr, events)})
		if !own && len(fr.Events) > 0 {
			docs = append(docs, leakcheck.Doc{Label: where + " events", Kind: leakcheck.View, Viewer: viewer, Moment: fr.Batch,
				Data: engineViewJSON(t, fr, fr.Events)})
		}
		if fr.History != nil {
			var h []map[string]any
			for _, e := range *fr.History {
				point := rules.HistoryStartPoint
				if e.Batch > 0 {
					point = truth.Moments[e.Batch-1].Point
				}
				h = append(h, map[string]any{"point": point, "events": e.Events})
			}
			if h == nil {
				h = []map[string]any{}
			}
			data, _ := json.Marshal(h)
			docs = append(docs, leakcheck.Doc{Label: where + " history", Kind: leakcheck.History, Viewer: viewer, Moment: historyBatch(fr), Data: data})
		}
	}
	return docs
}

// pendingCanaries plants, beside a frame's legitimate pending pick, what
// must never ride along with it, and requires the oracle to report each:
// another seat's observation hash, another seat's raw key, and the key
// named twice (a decoder keeps only the second). raw is the frame as
// received; each planted frame is decoded from its bytes as the socket
// adapter does, and the oracle gets both, as it gets the socket's frames.
func pendingCanaries(t testing.TB, c *leakcheck.Checker, label string, v frameViewer, raw []byte) {
	t.Helper()
	var fr wire.TableState
	if err := json.Unmarshal(raw, &fr); err != nil || fr.Pending == nil || v.Seat < 0 {
		t.Fatalf("%s: no frame with the viewer's pending pick: %v", label, err)
	}
	check := func(name string, planted []byte, rule leakcheck.Rule) {
		t.Helper()
		var got wire.TableState
		if err := json.Unmarshal(planted, &got); err != nil {
			t.Fatalf("%s, %s: %v", label, name, err)
		}
		leaks := frameLeaks(t, c, label+" "+name, v, []wire.TableState{got}, [][]byte{planted}, nil)
		switch {
		case rule == "" && len(leaks) != 0:
			t.Errorf("%s, %s: %v", label, name, leaks)
		case rule != "" && !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == rule }):
			t.Errorf("%s, %s: %s did not report it; got %v", label, name, rule, leaks)
		}
	}
	plant := func(field string, value any) []byte {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var tree map[string]any
		if err := dec.Decode(&tree); err != nil {
			t.Fatal(err)
		}
		tree["pending"].(map[string]any)[field] = value
		out, err := json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	other := (v.Seat + 1) % c.Truth.Seats
	m := &c.Truth.Moments[fr.Batch]
	name := fmt.Sprintf("seat %d's observation hash at moment %d", other, fr.Batch)
	i := slices.IndexFunc(c.Truth.Secrets, func(s leakcheck.Secret) bool { return s.Name == name })
	j := slices.IndexFunc(m.Legal[other], func(k string) bool { return !slices.Contains(m.Legal[v.Seat], k) })
	if i < 0 || j < 0 {
		t.Fatalf("%s: no %s or key of seat %d's own", label, name, other)
	}
	foreign, _ := json.Marshal(m.Legal[other][j])
	check("unplanted", raw, "")
	check("an observation hash beside the pick", plant("observationHash", c.Truth.Secrets[i].Value), leakcheck.Hash)
	check("another seat's key beside the pick", plant("also", m.Legal[other][j]), leakcheck.RawKey)
	twice := bytes.Replace(raw, []byte(`"pending":{`), append(append([]byte(`"pending":{"key":`), foreign...), ','), 1)
	if bytes.Equal(twice, raw) {
		t.Fatalf("%s: no pending object in the raw frame", label)
	}
	check("the key named twice", twice, leakcheck.Envelope)
}

// TestPendingExtrasStayScanned: vouching for a pending pick exempts only
// its key from the whole-frame scan, never the fields beside it, in a
// frame of the in-process table (the socket test runs the same canaries
// on frames it received).
func TestPendingExtrasStayScanned(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 2029))
	own := h.snap(tabletest.SeatClient(0))
	h.expect(h.send(tabletest.SeatClient(0), 1, own.Ref, own.Legal[0], false), tabletest.Accepted, "own draft")
	truth, err := leakcheck.FromReal(makeDaySetup(2029, 3, false, false), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(h.latest(tabletest.SeatClient(0)))
	if err != nil {
		t.Fatal(err)
	}
	pendingCanaries(t, leakcheck.New(truth, frameConfig), "in process", frameViewer{Seat: 0, Client: string(tabletest.SeatClient(0)), Admin: true}, raw)
}

// withoutVouchedKey removes exactly the vouched pending key from a frame's
// raw bytes, and keeps every other field and number for the whole-frame
// scan: the confirmation and anything else beside the key included. It
// removes nothing, and says why, unless the raw frame names no field twice
// at any level (decoding would keep only one of them, hiding the other
// from the scan) and its one pending object holds that key as a string and
// that confirmation as a boolean.
func withoutVouchedKey(data []byte, want wire.TablePending) ([]byte, string) {
	if dup := duplicateField(data); dup != "" {
		return nil, "the raw frame names " + dup + " twice"
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, "the raw frame is not a JSON object"
	}
	pd, ok := m["pending"].(map[string]any)
	if !ok {
		return nil, "the raw pending pick is not an object"
	}
	if key, ok := pd["key"].(string); !ok || key != want.Key {
		return nil, "the raw pending key is not the vouched one"
	}
	if confirm, ok := pd["confirm"].(bool); !ok || confirm != want.Confirm {
		return nil, "the raw confirmation is not the vouched one"
	}
	delete(pd, "key")
	out, err := json.Marshal(m)
	if err != nil {
		return nil, "the raw frame does not encode again"
	}
	return out, ""
}

// duplicateField returns the path of a field name that appears twice in
// one object of data, or "".
func duplicateField(data []byte) string {
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func(path string) string
	walk = func(path string) string {
		tok, err := dec.Token()
		if err != nil {
			return path + " (unreadable)"
		}
		switch tok {
		case json.Delim('{'):
			seen := map[string]bool{}
			for dec.More() {
				name, err := dec.Token()
				if err != nil {
					return path + " (unreadable)"
				}
				k, _ := name.(string)
				if seen[k] {
					return path + "." + k
				}
				seen[k] = true
				if d := walk(path + "." + k); d != "" {
					return d
				}
			}
			_, _ = dec.Token()
		case json.Delim('['):
			for dec.More() {
				if d := walk(path + "[]"); d != "" {
					return d
				}
			}
			_, _ = dec.Token()
		}
		return ""
	}
	return walk("")
}

// historyBatch is the batch a frame's history reaches: its last entry's.
func historyBatch(fr wire.TableState) int {
	if fr.History == nil || len(*fr.History) == 0 {
		return 0
	}
	return (*fr.History)[len(*fr.History)-1].Batch
}

func lastEvents(fr wire.TableState) []wire.TableEvent {
	if fr.History == nil || len(*fr.History) == 0 {
		return []wire.TableEvent{}
	}
	return (*fr.History)[len(*fr.History)-1].Events
}

// engineViewJSON rebuilds the engine's view from a frame.
func engineViewJSON(t testing.TB, fr wire.TableState, events []wire.TableEvent) []byte {
	// Transport cosmetics omit events; the pure day/random-side view at
	// batch zero still describes Age I. Reconstruct this public setup fact
	// just as point/kind/required below, without exempting any raw field.
	if fr.Batch == 0 && fr.View.Age == 1 && len(events) == 0 {
		events = []wire.TableEvent{{Kind: "age-start", Age: 1}}
	}

	t.Helper()
	data, err := json.Marshal(fr.View)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	m["point"], m["kind"], m["required"] = "", kindName(rules.PointNone), []int{}
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
	p.track = true
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
	var bs []rules.Batch
	for _, b := range can.Batches {
		eb := rules.Batch{Point: b.Point}
		for _, pk := range b.Picks {
			eb.Picks = append(eb.Picks, rules.Pick{Seat: rules.Seat(pk.Seat), Key: pk.Key})
		}
		bs = append(bs, eb)
	}
	truth, err := leakcheck.FromReal(makeDaySetup(seed, n, false, hide), bs)
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
// events, history on join and resync, passes the oracle, each pending pick
// checked against the pick the test saw its seat's accepted submissions
// set. The race build plays 3 and 7 seats only (SweepSeats).
func TestFramesLeakNothing(t *testing.T) {
	for _, n := range SweepSeats() {
		for _, hide := range []bool{false, true} {
			f := newFixture(t, Config{})
			seed := uint64(500 + n)
			p, truth := playForLeaks(t, f, n, seed, hide)
			c := leakcheck.New(truth, frameConfig)
			total, picks := 0, 0
			for id, frames := range p.frames {
				v := frameViewer{Seat: viewerOf(id), Client: string(id), Admin: viewerOf(id) == 0}
				if leaks := frameLeaks(t, c, fmt.Sprintf("%d seats hide=%t %s", n, hide, id), v, frames, nil, p.wants[id]); len(leaks) != 0 {
					t.Fatalf("%d seats, hide=%t, %s: %v", n, hide, id, leaks[:min(len(leaks), 5)])
				}
				total += len(frames)
				for i, w := range p.wants[id] {
					if w.Known && frames[i].Pending != nil {
						picks++
					}
				}
			}
			if total < 20 || picks == 0 {
				t.Fatalf("%d seats: only %d frames, %d with a pending pick checked against the model", n, total, picks)
			}
		}
	}
}

// TestSwitchFramesLeakNothing: across seat switches, role changes and a
// resignation, every frame passes the oracle checked against its viewer
// as the test's model says it was when the frame was sent. A user who
// loses their seat sees no hand from then on, the spectator who takes a
// seat sees that seat's, each connection's first frame for a new seat
// carries the history redacted for it (planted: without it, the oracle
// reports it), the previous-seat hint reaches a user while, and only
// while, they are an admin, and a former player's own frames, and no one
// else's, name the seat it held last.
func TestSwitchFramesLeakNothing(t *testing.T) {
	const n, seed = 4, 67
	h := newTH(t, Config{BotWork: blockBots}, humans(n, seed))
	p := h.p
	p.former = map[string]int{}
	admin, c1, c2 := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	seqs := map[tabletest.ClientID]int{}
	seq := func(id tabletest.ClientID) int { seqs[id]++; return seqs[id] }

	// A whole batch first, so the switches land on a table with history,
	// then a pick of seat 1's own that stays pending.
	for s := range n {
		id := tabletest.SeatClient(s)
		if sn := h.snap(id); len(sn.Legal) > 0 {
			if o := h.send(id, uint64(seq(id)), sn.Ref, sn.Legal[0], true); o != tabletest.Accepted && o != tabletest.Resolved {
				t.Fatalf("seat %d's confirmed pick: %s", s, o)
			}
		}
	}
	if b := h.latest(tabletest.SpectatorClient).Batch; b != 1 {
		t.Fatalf("batch %d after every seat confirmed", b)
	}
	if sn := h.snap(c1); len(sn.Legal) > 0 {
		h.expect(h.send(c1, uint64(seq(c1)), sn.Ref, sn.Legal[0], false), tabletest.Accepted, "seat 1's pick")
	}

	// change runs one accepted request after the frames sent before it are
	// drained under the old model and the model is moved to the new one.
	change := func(what string, model func(), id tabletest.ClientID, f wire.ClientMessage) {
		t.Helper()
		p.drain()
		model()
		h.want(id, f, outAccepted, what)
	}
	change("seat 1 to a bot", func() { delete(p.seats, userOf(1)); p.former[userOf(1)] = 1 }, admin, toBot(seq(admin), 1, BotEasy))
	if fs := h.latest(c1).You.FormerSeat; fs == nil || *fs != 1 {
		t.Fatalf("the former player's own seat after the switch: %v", fs)
	}
	if ps := h.latest(admin).You.PreviousSeats; ps == nil || len(*ps) != 1 || (*ps)[0] != (wire.PreviousSeat{UserID: userOf(1), Seat: 1}) {
		t.Fatalf("the admin's hint after the switch: %v", ps)
	}
	change("the watcher to seat 1", func() { p.seats["watcher"] = 1 }, admin, toHuman(seq(admin), 1, "watcher"))
	change("user-1's promotion", func() { p.admins[userOf(1)] = true }, admin, roleOp(seq(admin), "promote", userOf(1)))
	change("user-1 steps down", func() { p.admins[userOf(1)] = false }, c1, roleOp(seq(c1), "step-down", ""))
	change("seat 2 resigns", func() { delete(p.seats, userOf(2)); p.former[userOf(2)] = 2 }, c2, wire.TableResign{T: "table-resign", Seq: seq(c2), Confirm: true})
	change("user-2's promotion", func() { p.admins[userOf(2)] = true }, admin, roleOp(seq(admin), "promote", userOf(2)))
	change("user-2's demotion", func() { p.admins[userOf(2)] = false }, admin, roleOp(seq(admin), "demote", userOf(2)))
	for _, c := range p.clients {
		a, err := p.actor()
		if err == nil {
			err = a.call(h.ctx, func() { a.inbound(c, wire.NewResync(p.code)) })
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	p.drain()

	c := leakcheck.New(truthOf(t, p, n, seed, false), frameConfig)
	hinted := map[tabletest.ClientID]int{}
	unseated, formers := 0, 0
	moved := 0
	for id, frames := range p.frames {
		vs := p.viewers[id]
		if leaks := viewerLeaks(t, c, string(id), vs, frames, nil, nil); len(leaks) != 0 {
			t.Fatalf("%s: %v", id, leaks[:min(len(leaks), 5)])
		}
		// Planted: the first frame for a new seat without its history.
		if i := slices.IndexFunc(vs, func(v frameViewer) bool { return v.Seat != vs[0].Seat }); i > 0 {
			moved++
			planted := slices.Clone(frames)
			planted[i].History = nil
			if leaks := viewerLeaks(t, c, string(id)+" planted", vs, planted, nil, nil); !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool {
				return l.Rule == leakcheck.Envelope && l.Path == "history"
			}) {
				t.Errorf("%s: a seat change without history was not reported: %v", id, leaks)
			}
		}
		for i, fr := range frames {
			if ps := fr.You.PreviousSeats; ps != nil && len(*ps) > 0 {
				hinted[id]++
			}
			if vs[i].Seat < 0 && id != tabletest.SpectatorClient {
				unseated++
			}
			if fr.You.FormerSeat != nil {
				formers++
			}
		}
	}
	if hinted[admin] == 0 || hinted[c1] == 0 || hinted[c2] == 0 || unseated == 0 || formers == 0 || moved < 3 {
		t.Fatalf("too little checked: hinted frames %v, %d frames of an unseated user, %d naming a former seat, %d connections moved", hinted, unseated, formers, moved)
	}
}

// TestPlantedEnvelopeLeaks: a frame's private fields are checked against
// the viewer and the accepted state, and a pending pick is left out of the
// raw-key scan only once that check vouches for it. Planted in real
// frames, each of these is reported: another seat's pending pick sent to
// the spectator, another seat's key as the viewer's pick, the viewer's own
// key that is not its accepted pick, a confirmation the public list does
// not show, another seat's you, another connection's acknowledgement, a
// hidden card in the events of a history frame, the previous-seat hint
// where it does not belong (the admin's own frame given to a non-admin, a
// hint planted in the spectator's frame, a seated user in the admin's
// hint), a former seat sent to a seated viewer, to a spectator who held no
// seat, or naming another seat than the viewer's own, and the admin cap
// sent to a non-admin.
func TestPlantedEnvelopeLeaks(t *testing.T) {
	f := newFixture(t, Config{})
	p, truth := playForLeaks(t, f, 3, 2028, false)
	c := leakcheck.New(truth, frameConfig)
	spec := frameViewer{Seat: leakcheck.Spectator, Client: string(tabletest.SpectatorClient)}
	seat0 := frameViewer{Seat: 0, Client: string(tabletest.SeatClient(0)), Admin: true}
	sf, s0 := p.frames[tabletest.SpectatorClient][0], p.frames[tabletest.SeatClient(0)][0]
	m := &truth.Moments[0]
	if sf.Batch != 0 || s0.Batch != 0 || s0.Pending != nil || s0.Point == nil || !slices.Contains(s0.Point.Required, 0) || slices.Contains(s0.Point.Confirmed, 0) {
		t.Fatalf("the first frames: %+v, %+v", sf.Point, s0.Point)
	}
	for _, clean := range []struct {
		v  frameViewer
		fr wire.TableState
	}{{spec, sf}, {seat0, s0}} {
		if leaks := frameLeaks(t, c, "clean", clean.v, []wire.TableState{clean.fr}, nil, nil); len(leaks) != 0 {
			t.Fatalf("an unplanted frame: %v", leaks)
		}
	}
	foreign := m.Legal[1][slices.IndexFunc(m.Legal[1], func(k string) bool { return !slices.Contains(m.Legal[0], k) })]
	pending := func(fr wire.TableState, key string, confirm bool) wire.TableState {
		fr.Pending = &wire.TablePending{Key: key, Confirm: confirm}
		return fr
	}
	other := 1
	ack := s0
	ack.Ack = &wire.PlatformAck{ClientID: string(tabletest.SeatClient(1)), Seq: 1}
	you := s0
	you.You.Seat = &other
	if s0.You.PreviousSeats == nil || sf.You.PreviousSeats != nil {
		t.Fatal("the hint is not in the admin's frame alone")
	}
	notAdmin := seat0
	notAdmin.Admin = false
	hinted := sf
	hinted.You.PreviousSeats = &[]wire.PreviousSeat{}
	seated := s0
	seated.You.PreviousSeats = &[]wire.PreviousSeat{{UserID: *s0.Seats[1].UserID, Seat: 1}}
	one, two := 1, 2
	if s0.You.CanPromote == nil || !*s0.You.CanPromote || sf.You.CanPromote != nil || s0.You.FormerSeat != nil || sf.You.FormerSeat != nil {
		t.Fatal("the admin cap is not in the admin's frame alone, or a former seat is sent")
	}
	formerSeated := s0
	formerSeated.You.FormerSeat = &one
	formerWatcher := sf
	formerWatcher.You.FormerSeat = &one
	formerOf2 := spec
	formerOf2.Former = &two
	capped := sf
	capped.You.CanPromote = new(bool)

	// A history frame whose own events name a card seat 0 may not see.
	var hist wire.TableState
	for _, fr := range p.frames[tabletest.SeatClient(0)] {
		if fr.Batch == 1 && len(fr.Events) > 0 {
			hist = fr
			break
		}
	}
	hidden := slices.IndexFunc(hist.Events, func(e wire.TableEvent) bool { return e.Design == "" })
	if hist.Batch != 1 || hidden < 0 {
		t.Fatal("no resolution frame with an event seat 0 may not see")
	}
	entry := []wire.TableHistoryEntry{{Batch: 1, Events: hist.Events}}
	hist.History = &entry
	hist.Events = slices.Clone(hist.Events)
	hist.Events[hidden].Design = truth.Moments[1].Hands[1][0]

	for _, tc := range []struct {
		name  string
		v     frameViewer
		fr    wire.TableState
		wants []pendingWant
		rules []leakcheck.Rule
	}{
		{"seat 1's pick sent to the spectator", spec, pending(sf, m.Legal[1][0], false), nil, []leakcheck.Rule{leakcheck.Envelope, leakcheck.RawKey}},
		{"seat 1's key as seat 0's pick", seat0, pending(s0, foreign, false), nil, []leakcheck.Rule{leakcheck.Envelope, leakcheck.RawKey}},
		{"seat 0's own key, never accepted", seat0, pending(s0, m.Legal[0][0], false), []pendingWant{{Known: true}}, []leakcheck.Rule{leakcheck.Envelope}},
		{"a confirmation the public list does not show", seat0, pending(s0, m.Legal[0][0], true), nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"another seat's you", seat0, you, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"another connection's acknowledgement", seat0, ack, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"a hidden card in a history frame's events", seat0, hist, nil, []leakcheck.Rule{leakcheck.HiddenCard}},
		{"the admin's frame given to a non-admin", notAdmin, s0, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"a hint in the spectator's frame", spec, hinted, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"a seated user in the admin's hint", seat0, seated, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"a former seat in a seated viewer's frame", seat0, formerSeated, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"another user's former seat sent to the spectator", spec, formerWatcher, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"a former seat that is not the viewer's own", formerOf2, formerWatcher, nil, []leakcheck.Rule{leakcheck.Envelope}},
		{"the admin cap in a non-admin's frame", spec, capped, nil, []leakcheck.Rule{leakcheck.Envelope}},
	} {
		leaks := frameLeaks(t, c, tc.name, tc.v, []wire.TableState{tc.fr}, nil, tc.wants)
		for _, rule := range tc.rules {
			if !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == rule }) {
				t.Errorf("%s: %s did not report it; got %v", tc.name, rule, leaks)
			}
		}
	}
}

// TestPlantedLeaksInFrames: a leak planted in a real frame fails its own
// detector: another seat's hand card, a raw key in an event,
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
			{Label: "view", Kind: leakcheck.View, Viewer: 0, Moment: 1, Data: engineViewJSON(t, st, ev)},
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
	g, _ := replay(makeDaySetup(77, 4, false, false), nil)
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
	data := engineViewJSON(t, base, events)
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
	g, _ := replay(makeDaySetup(seed, 4, false, false), nil)
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
	g, err := replay(makeDaySetup(l.Setup.Seed, l.Setup.Seats, l.Setup.RandomSeating, l.Setup.HideAtEnd), engineBatches(l.Batches))
	if err != nil || g.Fingerprint() != l.Hash || l.Format != LogFormat || l.Game != Game || l.Rules != RulesVersion {
		t.Fatalf("the log does not replay: %v", err)
	}
	raw, _ := json.Marshal(l)
	var tree any
	_ = json.Unmarshal(raw, &tree)
	if got, want := fieldNames(tree), "batches format game hash hideAtEnd key picks point randomSeating rules seat seats seed setup sides"; got != want {
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
