package protocoltest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/bits"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// trueHidden returns the slugs that must not be visible to viewer right
// now, reconstructed directly from the engine's own exported state fields
// (not from view.go, which is the production code these tests exist to
// check). viewer nil means a spectator, the strictest case: a spectator
// never sees the Great Library offer either.
//
// This is a best-effort reconstruction, not a certified mirror of view.go's
// whitelist (see doc.go): it covers every category view.go's own doc
// comment lists (face-down slots, removed cards, later ages' deals, box
// guilds, box wonders, draft round two before round one ends, and the
// Great Library offer), which is precise enough for an integration leak
// check, since every id is unique within one game. Once the game is over,
// RevealView exposes all of this (the rooms this package builds use the
// default Options, so HideAtEnd is false), so trueHidden returns empty.
func trueHidden(s *engine.State, viewer *engine.Seat) map[string]bool {
	hidden := map[string]bool{}
	if s.Over() {
		return hidden
	}
	addCard := func(id catalog.CardID) {
		if id == 0 {
			return
		}
		if d := catalog.Card(id); d != nil {
			hidden[d.Slug] = true
		}
	}
	addWonder := func(id catalog.WonderID) {
		if id == 0 {
			return
		}
		if d := catalog.Wonder(id); d != nil {
			hidden[d.Slug] = true
		}
	}
	addToken := func(id catalog.TokenID) {
		if id == 0 {
			return
		}
		if d := catalog.Token(id); d != nil {
			hidden[d.Slug] = true
		}
	}

	for _, slot := range s.Slots {
		if !slot.FaceUp {
			addCard(slot.Card)
		}
	}
	for _, deal := range s.Deals {
		for _, id := range deal {
			addCard(id)
		}
	}
	// The cards set aside from every age's deck, not just the current one's:
	// an age that is over keeps its removed cards hidden until the game is.
	for _, removed := range s.Removed {
		for _, id := range removed {
			addCard(id)
		}
	}
	for _, id := range s.BoxGuilds {
		addCard(id)
	}
	for _, id := range s.BoxWonders {
		addWonder(id)
	}
	// Draft round two stays hidden until round one's four wonders are all
	// taken (view.go's viewDraft: round := draftRound(draftPicks(s)), where
	// draftPicks is popcount(DraftTaken)); this is not simply "fewer than
	// four actions applied" because the round's last wonder can be taken
	// automatically (no real choice left) in the same Apply call as the
	// third, so Ply alone undercounts picks taken near the round boundary.
	if s.Phase == engine.PhaseWonderDraft && bits.OnesCount8(s.DraftTaken) < 4 {
		for _, id := range s.DraftOrder[4:] {
			addWonder(id)
		}
	}
	for _, id := range s.BoxTokens {
		addToken(id)
	}
	for id := range s.OutOfPlay.All() {
		addCard(id)
	}

	// The Great Library offer: drawLibrary moves the drawn tokens out of
	// BoxTokens into Library, so they are not covered by the box above.
	// Only the deciding seat may see them; the other seat and a spectator
	// see only how many there are.
	decider := viewer != nil && s.Phase == engine.PhasePickLibraryToken && *viewer == s.ToMove
	if !decider {
		for _, id := range s.Library {
			addToken(id)
		}
	}
	return hidden
}

// leakTracker checks every frame a connection receives against what was
// hidden from that connection's viewer at the version the room built the
// frame for, and every byte against the room's seed (the view package never
// includes the seed at all, not even once the game is over, so there is no
// "before game over" exception to carve out).
//
// A frame's own version is what matters, not the moment the test reads it.
// The room builds a state frame from its state at one version and the
// connection's writer sends it later, so a frame can reach the test after
// the room has moved on: a presence broadcast at version v (another
// connection joining) can arrive after the test has checked version v and
// played v+1. And what is hidden from a viewer does not only shrink: the
// Great Library's decider sees its offer at v, and the two tokens it does
// not keep go back into the box, hidden again, at v+1. Judging that late
// frame by v+1's hidden set reports a leak that never happened.
//
// So each check records the hidden set at its snapshot's version, and a
// state frame is judged by the version it carries:
//
//   - a version some check recorded: exactly that version's hidden set;
//   - a version newer than any recorded (the room moved on between the
//     snapshot and the scan, as a bot's move can): left, with everything
//     after it, for a later check, and reported as unchecked if no check
//     reaches it before the test ends;
//   - an older version no check recorded (a bot moved twice between two
//     checks): the hidden set of the first check that reached it.
//
// Frames other than state frames carry no version and no view; they are
// judged by the hidden set of the check that scans them.
type leakTracker struct {
	t         reporter
	seed      []byte // the seed's decimal digits, as they would appear unquoted in JSON
	seedFound bool
	conns     map[*tclient]*trackedConn
}

// reporter is the part of *testing.T a leakTracker reports through, so the
// tracker's own tests can capture what it reports.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// trackedConn is what a leakTracker knows about one connection.
type trackedConn struct {
	who     string                  // the label of its failures
	scanned int                     // chunks of c.raw already judged
	hidden  map[int]map[string]bool // by room version: the slugs hidden from its viewer then
	latest  int                     // the newest version in hidden
}

// newLeakTracker returns a tracker for one room whose seed is seed. When the
// test ends it judges whatever arrived after the last check (see finish).
func newLeakTracker(t *testing.T, seed uint64) *leakTracker {
	lt := newLeakTrackerFor(t, seed)
	t.Cleanup(func() {
		if !t.Failed() {
			lt.finish()
		}
	})
	return lt
}

func newLeakTrackerFor(r reporter, seed uint64) *leakTracker {
	return &leakTracker{t: r, seed: []byte(strconv.FormatUint(seed, 10)), conns: map[*tclient]*trackedConn{}}
}

// check records hidden as what c's viewer must not see at the room's
// version, then judges c's frames received since the last check (see
// leakTracker), labeling failures with who.
func (lt *leakTracker) check(c *tclient, who string, version int, hidden map[string]bool) {
	lt.t.Helper()
	tc := lt.conns[c]
	if tc == nil {
		tc = &trackedConn{hidden: map[int]map[string]bool{}}
		lt.conns[c] = tc
	}
	tc.who = who
	tc.hidden[version] = hidden
	tc.latest = max(tc.latest, version)
	lt.scan(c, tc, false)
}

// finish judges what every connection received after its last check. A
// state frame newer than every recorded version is reported as unchecked.
func (lt *leakTracker) finish() {
	lt.t.Helper()
	for c, tc := range lt.conns {
		lt.scan(c, tc, true)
	}
}

// scan judges c's unscanned chunks in order. Unless final, it stops at the
// first state frame newer than any recorded version.
func (lt *leakTracker) scan(c *tclient, tc *trackedConn, final bool) {
	lt.t.Helper()
	chunks, _ := c.raw.since(tc.scanned)
	for _, chunk := range chunks {
		hidden := tc.hidden[tc.latest]
		if v, ok := stateVersion(chunk); ok {
			if h, recorded := tc.hidden[v]; recorded {
				hidden = h
			} else if v > tc.latest {
				if !final {
					return
				}
				lt.t.Errorf("leak check: %s received a state frame at version %d, newer than its last check (version %d), so nothing judged it: %s", tc.who, v, tc.latest, chunk)
				hidden = nil
			}
		}
		tc.scanned++
		lt.judge(tc.who, chunk, hidden)
	}
}

// judge reports each slug of hidden that chunk contains (as a quoted JSON
// string, so "law" cannot spuriously match inside a longer word), and the
// seed.
func (lt *leakTracker) judge(who string, chunk []byte, hidden map[string]bool) {
	lt.t.Helper()
	named := withoutHintNames(chunk)
	for slug := range hidden {
		if pat := []byte(`"` + slug + `"`); bytes.Contains(named, pat) {
			lt.t.Errorf("leak: %s received a hidden id %s: %s", who, pat, chunk)
		}
	}
	if !lt.seedFound && bytes.Contains(chunk, lt.seed) {
		lt.seedFound = true
		lt.t.Errorf("leak: %s received the room's seed: %s", who, chunk)
	}
}

// withoutHintNames returns a state frame without its hint settings
// (room.hints) and its room log, whose fixed vocabulary of hint family
// names includes "economy", also the id of a progress token. Every other
// frame comes back as it is. The wire decoder and the hygiene tests pin
// that vocabulary, so nothing hidden can travel there.
func withoutHintNames(chunk []byte) []byte {
	var frame map[string]json.RawMessage
	if err := json.Unmarshal(chunk, &frame); err != nil || string(frame["t"]) != `"state"` {
		return chunk
	}
	delete(frame, "roomLog")
	var room map[string]json.RawMessage
	if err := json.Unmarshal(frame["room"], &room); err == nil {
		delete(room, "hints")
		if b, err := json.Marshal(room); err == nil {
			frame["room"] = b
		}
	}
	out, err := json.Marshal(frame)
	if err != nil {
		return chunk
	}
	return out
}

// stateVersion returns the room version of a state frame, and false for any
// other frame.
func stateVersion(chunk []byte) (int, bool) {
	var head struct {
		T       string `json:"t"`
		Version int    `json:"version"`
	}
	if json.Unmarshal(chunk, &head) != nil || head.T != "state" {
		return 0, false
	}
	return head.Version, true
}

// leakRecorder is a reporter that keeps what a leakTracker reports.
type leakRecorder struct{ errs []string }

func (r *leakRecorder) Helper() {}

func (r *leakRecorder) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

// TestLeakTrackerJudgesEachFrameByItsVersion feeds a leakTracker state
// frames the way the Great Library test receives them, without a server,
// so that the order of frames and checks is fixed. The first case is the
// sequence that failed on CI when every frame was judged by the newest
// hidden set: seat 0's offer, in the spectator's presence broadcast at
// version 1, read only after seat 0 had kept law at version 2. The other
// cases are leaks the tracker must still report.
func TestLeakTrackerJudgesEachFrameByItsVersion(t *testing.T) {
	const seed = 9_950_000_000_000
	// What each seat must not see before (v1) and after (v2) seat 0 keeps
	// law from an offer of law, philosophy and strategy, with masonry and
	// theology left in the box.
	deciderV1 := map[string]bool{"masonry": true, "theology": true}
	deciderV2 := map[string]bool{"masonry": true, "theology": true, "philosophy": true, "strategy": true}
	otherV1 := map[string]bool{"masonry": true, "theology": true, "law": true, "philosophy": true, "strategy": true}
	otherV2 := deciderV2
	frame := func(version int, tokens ...string) []byte {
		b, err := json.Marshal(map[string]any{
			"t": "state", "code": "LIBRARY", "version": version,
			"view": map[string]any{"pending": map[string]any{"kind": "pickLibraryToken", "tokens": tokens}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	cases := []struct {
		name string
		play func(lt *leakTracker, c *tclient)
		want []string // one per report expected: a part of its text
	}{
		{"the decider's offer read after the take", func(lt *leakTracker, c *tclient) {
			c.raw.add(frame(1, "law", "philosophy", "strategy")) // the join
			lt.check(c, "seat0", 1, deciderV1)
			c.raw.add(frame(1, "law", "philosophy", "strategy")) // the presence broadcast, read late
			c.raw.add(frame(2, "law"))                           // the ack of token:law
			lt.check(c, "seat0", 2, deciderV2)
		}, nil},
		{"a token put back, at the frame's own version", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat0", 1, deciderV1)
			c.raw.add(frame(2, "law", "philosophy"))
			lt.check(c, "seat0", 2, deciderV2)
		}, []string{`seat0 received a hidden id "philosophy"`}},
		{"the offer read late by the other seat", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat1", 1, otherV1)
			c.raw.add(frame(1, "law", "philosophy", "strategy"))
			c.raw.add(frame(2, "law"))
			lt.check(c, "seat1", 2, otherV2)
		}, []string{`seat1 received a hidden id "law"`, `seat1 received a hidden id "philosophy"`, `seat1 received a hidden id "strategy"`}},
		{"a frame newer than the snapshot waits for its version", func(lt *leakTracker, c *tclient) {
			c.raw.add(frame(2, "law")) // law is public from v2 on, hidden at v1
			lt.check(c, "seat1", 1, otherV1)
			lt.check(c, "seat1", 2, otherV2)
		}, nil},
		{"a frame that waited is still judged", func(lt *leakTracker, c *tclient) {
			c.raw.add(frame(2, "philosophy"))
			lt.check(c, "seat1", 1, otherV1)
			lt.check(c, "seat1", 2, otherV2)
		}, []string{`seat1 received a hidden id "philosophy"`}},
		{"a frame no check reached", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat1", 1, otherV1)
			c.raw.add(frame(2, "law"))
		}, []string{"so nothing judged it"}},
		{"the seed", func(lt *leakTracker, c *tclient) {
			c.raw.add([]byte(`{"t":"welcome","note":` + strconv.Itoa(seed) + `}`))
			lt.check(c, "seat0", 1, deciderV1)
		}, []string{"seat0 received the room's seed"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &leakRecorder{}
			lt := newLeakTrackerFor(rec, seed)
			tc.play(lt, &tclient{raw: &rawCapture{}})
			lt.finish()
			unmatched := append([]string{}, rec.errs...)
			for _, want := range tc.want {
				i := slices.IndexFunc(unmatched, func(e string) bool { return strings.Contains(e, want) })
				if i < 0 {
					t.Errorf("no report contains %q; reports: %q", want, rec.errs)
					continue
				}
				unmatched = slices.Delete(unmatched, i, i+1)
			}
			for _, e := range unmatched {
				t.Errorf("unexpected report: %s", e)
			}
		})
	}
}
