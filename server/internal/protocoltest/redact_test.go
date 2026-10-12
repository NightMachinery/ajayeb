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
//
// One exception narrows a hidden set, and only inside one kind of event:
// the tokens a viewer drew from the Great Library itself (secrets.drawn).
// The rules show them to the drawer at the pickLibraryToken step, which
// trueHidden already allows at that version, and the drawer's own
// libraryDraw events record them, live and in every later history. So a
// drawn token is allowed in the token field of a libraryDraw event of the
// viewer's own seat, and still reported anywhere else in any later frame,
// once the two it did not keep are back in the box.
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
	who     string          // the label of its failures
	scanned int             // chunks of c.raw already judged
	hidden  map[int]secrets // by room version: what its viewer must not see then
	latest  int             // the newest version in hidden
}

// secrets is what one viewer must not see at one room version.
type secrets struct {
	hidden map[string]bool // the slugs that must not appear in a frame
	seat   int             // the viewer's seat, or -1 for a spectator (and once the game is over)
	drawn  map[string]bool // the Great Library tokens this seat drew, allowed in its own libraryDraw events
}

// only returns secrets that hide hidden everywhere, with no drawn tokens.
func only(hidden map[string]bool) secrets { return secrets{hidden: hidden, seat: -1} }

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

// check records sec as what c's viewer must not see at the room's
// version, then judges c's frames received since the last check (see
// leakTracker), labeling failures with who.
func (lt *leakTracker) check(c *tclient, who string, version int, sec secrets) {
	lt.t.Helper()
	tc := lt.conns[c]
	if tc == nil {
		tc = &trackedConn{hidden: map[int]secrets{}}
		lt.conns[c] = tc
	}
	tc.who = who
	tc.hidden[version] = sec
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
		sec := tc.hidden[tc.latest]
		if v, ok := stateVersion(chunk); ok {
			if h, recorded := tc.hidden[v]; recorded {
				sec = h
			} else if v > tc.latest {
				if !final {
					return
				}
				lt.t.Errorf("leak check: %s received a state frame at version %d, newer than its last check (version %d), so nothing judged it: %s", tc.who, v, tc.latest, chunk)
				sec = secrets{}
			}
		}
		tc.scanned++
		lt.judge(tc.who, chunk, sec)
	}
}

// judge reports each slug of sec.hidden that chunk contains (as a quoted
// JSON string, so "law" cannot spuriously match inside a longer word), and
// the seed. The tokens of sec.drawn are allowed only in the viewer's own
// libraryDraw events (see leakTracker).
func (lt *leakTracker) judge(who string, chunk []byte, sec secrets) {
	lt.t.Helper()
	named := withoutOwnDraws(withoutHintNames(chunk), sec)
	for slug := range sec.hidden {
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
//
// It also drops the board hints that list candidate sets by name (see
// candidateHints), which name hidden ids by design, among public ones.
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
	if b, ok := withoutCandidateHints(frame["view"]); ok {
		frame["view"] = b
	}
	out, err := json.Marshal(frame)
	if err != nil {
		return chunk
	}
	return out
}

// candidateHints are the board hint fields (M10.HS4) that name a public
// candidate set: the cards of an age not yet seen (view.hints.unseen), both
// cards of each science pair with their public place (view.hints.pairs),
// the progress tokens not on the board or in a city (view.hints.boxTokens),
// the guilds that may still come (view.hints.guildPreview), and the cards a
// seat's chains make free (players[].hints.chainTargets). Each set is the
// catalog minus what both players have seen, so it names the set-aside
// cards, the box guilds and the box tokens among the rest without saying
// which is which, exactly as a player can work out. A byte scan cannot
// tell that apart from a leak; the engine's determinization guard
// (engine/hints_leak_test.go) checks these fields instead, by recomputing
// them from a determinized observation and requiring the same value.
var candidateHints = struct{ shared, seat []string }{
	shared: []string{"unseen", "pairs", "boxTokens", "guildPreview"},
	seat:   []string{"chainTargets"},
}

// withoutCandidateHints returns a view without the fields of
// candidateHints, and false when raw is not a view with hints.
func withoutCandidateHints(raw json.RawMessage) (json.RawMessage, bool) {
	var v map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return nil, false
	}
	drop := func(raw json.RawMessage, keys []string) (json.RawMessage, bool) {
		var m map[string]json.RawMessage
		if len(raw) == 0 || json.Unmarshal(raw, &m) != nil || m == nil {
			return nil, false
		}
		for _, k := range keys {
			delete(m, k)
		}
		b, err := json.Marshal(m)
		return b, err == nil
	}
	if b, ok := drop(v["hints"], candidateHints.shared); ok {
		v["hints"] = b
	}
	var players []map[string]json.RawMessage
	if json.Unmarshal(v["players"], &players) == nil {
		for _, p := range players {
			if b, ok := drop(p["hints"], candidateHints.seat); ok {
				p["hints"] = b
			}
		}
		if b, err := json.Marshal(players); err == nil {
			v["players"] = b
		}
	}
	b, err := json.Marshal(v)
	return b, err == nil
}

// withoutOwnDraws returns a state frame without the token of each
// libraryDraw event, live or in its history, that the viewer's own seat drew
// and that sec.drawn lists. Every other frame, and every other event, comes
// back as it is, so a drawn token anywhere else is still judged.
func withoutOwnDraws(chunk []byte, sec secrets) []byte {
	if sec.seat < 0 || len(sec.drawn) == 0 {
		return chunk
	}
	var frame map[string]json.RawMessage
	if err := json.Unmarshal(chunk, &frame); err != nil || string(frame["t"]) != `"state"` {
		return chunk
	}
	strip := func(raw json.RawMessage) json.RawMessage {
		var events []map[string]json.RawMessage
		if json.Unmarshal(raw, &events) != nil {
			return raw
		}
		for _, e := range events {
			var kind, token string
			var seat *int
			if json.Unmarshal(e["kind"], &kind) != nil || kind != "libraryDraw" ||
				json.Unmarshal(e["seat"], &seat) != nil || seat == nil || *seat != sec.seat ||
				json.Unmarshal(e["token"], &token) != nil || !sec.drawn[token] {
				continue
			}
			delete(e, "token")
		}
		b, err := json.Marshal(events)
		if err != nil {
			return raw
		}
		return b
	}
	if raw, ok := frame["events"]; ok {
		frame["events"] = strip(raw)
	}
	if raw, ok := frame["history"]; ok {
		var history []map[string]json.RawMessage
		if json.Unmarshal(raw, &history) == nil {
			for _, entry := range history {
				if ev, ok := entry["events"]; ok {
					entry["events"] = strip(ev)
				}
			}
			if b, err := json.Marshal(history); err == nil {
				frame["history"] = b
			}
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
			lt.check(c, "seat0", 1, only(deciderV1))
			c.raw.add(frame(1, "law", "philosophy", "strategy")) // the presence broadcast, read late
			c.raw.add(frame(2, "law"))                           // the ack of token:law
			lt.check(c, "seat0", 2, only(deciderV2))
		}, nil},
		{"a token put back, at the frame's own version", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat0", 1, only(deciderV1))
			c.raw.add(frame(2, "law", "philosophy"))
			lt.check(c, "seat0", 2, only(deciderV2))
		}, []string{`seat0 received a hidden id "philosophy"`}},
		{"the offer read late by the other seat", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat1", 1, only(otherV1))
			c.raw.add(frame(1, "law", "philosophy", "strategy"))
			c.raw.add(frame(2, "law"))
			lt.check(c, "seat1", 2, only(otherV2))
		}, []string{`seat1 received a hidden id "law"`, `seat1 received a hidden id "philosophy"`, `seat1 received a hidden id "strategy"`}},
		{"a frame newer than the snapshot waits for its version", func(lt *leakTracker, c *tclient) {
			c.raw.add(frame(2, "law")) // law is public from v2 on, hidden at v1
			lt.check(c, "seat1", 1, only(otherV1))
			lt.check(c, "seat1", 2, only(otherV2))
		}, nil},
		{"a frame that waited is still judged", func(lt *leakTracker, c *tclient) {
			c.raw.add(frame(2, "philosophy"))
			lt.check(c, "seat1", 1, only(otherV1))
			lt.check(c, "seat1", 2, only(otherV2))
		}, []string{`seat1 received a hidden id "philosophy"`}},
		{"a frame no check reached", func(lt *leakTracker, c *tclient) {
			lt.check(c, "seat1", 1, only(otherV1))
			c.raw.add(frame(2, "law"))
		}, []string{"so nothing judged it"}},
		{"the seed", func(lt *leakTracker, c *tclient) {
			c.raw.add([]byte(`{"t":"welcome","note":` + strconv.Itoa(seed) + `}`))
			lt.check(c, "seat0", 1, only(deciderV1))
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
