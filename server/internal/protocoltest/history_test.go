package protocoltest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The history (docs/protocol.md, the state frame) re-sends every applied
// action's events, filtered for the viewer exactly like live events. The
// leak model therefore has one more thing a viewer may see: the Great
// Library tokens that viewer drew itself. Its own libraryDraw events name
// them, and the two it did not keep go back into the box, hidden from
// everyone else again but not secret from the drawer, who saw them.
// hiddenFor is trueHidden minus those tokens; the drive tests resync every
// connection after every turn, so every history a game produces goes
// through the same leak check as the live frames.

// seenPrivately returns the tokens viewer drew from the Great Library so
// far in the room's current line, replayed from the room's start. viewer
// nil (a spectator) draws nothing.
func seenPrivately(t *testing.T, snap room.Snapshot, viewer *engine.Seat) map[string]bool {
	t.Helper()
	seen := map[string]bool{}
	if viewer == nil {
		return seen
	}
	s := engine.NewGame(snap.State.Seed, snap.State.Opts)
	if snap.Scenario != "" {
		var err error
		if s, err = engine.NewGameFromScenario(snap.State.Seed, snap.State.Opts, engine.NamedScenario(snap.Scenario)); err != nil {
			t.Fatalf("seenPrivately: scenario %s: %v", snap.Scenario, err)
		}
	}
	for i, key := range snap.Actions {
		a, err := engine.ParseKey(key)
		if err != nil {
			t.Fatalf("seenPrivately: action %d: %v", i, err)
		}
		var ev []engine.Event
		if err := engine.Apply(&s, a, &ev); err != nil {
			t.Fatalf("seenPrivately: action %d (%s): %v", i, key, err)
		}
		for _, e := range ev {
			if e.Kind == engine.EvLibraryDraw && e.Seat == *viewer {
				if d := catalog.Token(e.Token); d != nil {
					seen[d.Slug] = true
				}
			}
		}
	}
	return seen
}

// hiddenFor is what viewer must not see in snap's position: trueHidden,
// less the tokens it drew itself (seenPrivately).
func hiddenFor(t *testing.T, snap room.Snapshot, viewer *engine.Seat) map[string]bool {
	t.Helper()
	hidden := trueHidden(&snap.State, viewer)
	for slug := range seenPrivately(t, snap, viewer) {
		delete(hidden, slug)
	}
	return hidden
}

// resync asks for a fresh state and returns it: a frame with the history,
// which must hold one entry per action of the game so far.
func (c *tclient) resync() wire.State {
	c.t.Helper()
	c.send(wire.NewResync(c.code))
	st := c.until("the resync's state", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Code == c.code && st.History != nil
	}).(wire.State)
	if len(*st.History) != st.View.Ply {
		c.t.Fatalf("%s: a history of %d entries at ply %d", c.name, len(*st.History), st.View.Ply)
	}
	for i, e := range *st.History {
		if e.Ply != i || (e.Seat != 0 && e.Seat != 1) || len(e.Events) == 0 {
			c.t.Fatalf("%s: history entry %d: %+v", c.name, i, e)
		}
	}
	return st
}

// resyncAll resyncs every connection.
func resyncAll(conns ...*tclient) {
	for _, c := range conns {
		c.resync()
	}
}

// TestLeakTrackerCatchesLeaksInHistory plants a hidden id inside a history
// entry's event, where no live frame of the same version carries it, and
// checks that the tracker reports it; the same frame without the planted
// id passes.
func TestLeakTrackerCatchesLeaksInHistory(t *testing.T) {
	const seed = 9_960_000_000_000
	hidden := map[string]bool{"baths": true, "pyramids": true}
	frame := func(card string) []byte {
		st := wire.NewState("HISTRY", 3, wire.NewRoomInfo(wire.ModeOnline, wire.RoomStatusPlaying, nil, 0),
			wire.NewYouInfo(nil, false), nil, nil, view.View{Ply: 3})
		history := []wire.HistoryEntry{
			{Ply: 0, Seat: 0, Events: []view.GameEvent{{Kind: view.EventKindBuild, Seat: seatInt(0), Card: &card}}},
		}
		st.History = &history
		b, err := json.Marshal(st)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, tc := range []struct {
		name, card string
		leak       bool
	}{{"clean", "lumber-yard", false}, {"planted", "baths", true}} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &leakRecorder{}
			lt := newLeakTrackerFor(rec, seed)
			c := &tclient{raw: &rawCapture{}}
			c.raw.add(frame(tc.card))
			lt.check(c, "seat1", 3, hidden)
			lt.finish()
			found := len(rec.errs) == 1 && strings.Contains(rec.errs[0], `seat1 received a hidden id "baths"`)
			if tc.leak != found || (!tc.leak && len(rec.errs) != 0) {
				t.Fatalf("reports %q, want a leak: %v", rec.errs, tc.leak)
			}
		})
	}
}

// TestLibraryHistoryModel: after the Great Library's choice, the drawer's
// history names the three tokens it drew, two of them back in the box;
// the other seat's and the spectator's name none of them. hiddenFor allows
// exactly the drawer's own.
func TestLibraryHistoryModel(t *testing.T) {
	srv, code, a, b, spec := libraryRoom(t)
	seq := a.act("token:law", nil)
	a.untilAck(seq)
	conns := []*tclient{a, b, spec}
	syncTo(conns, maxVersion(conns))

	snap := srv.snapshot(code)
	lt := newLeakTracker(t, snap.State.Seed)
	histA, histB, histS := a.resync(), b.resync(), spec.resync()
	lt.check(a, "seat0", snap.Version, hiddenFor(t, snap, seatRef(0)))
	lt.check(b, "seat1", snap.Version, hiddenFor(t, snap, seatRef(1)))
	lt.check(spec, "spectator", snap.Version, hiddenFor(t, snap, nil))

	names := func(st wire.State) string {
		raw, err := json.Marshal(st.History)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, slug := range []string{"law", "philosophy", "strategy"} {
		if !strings.Contains(names(histA), `"`+slug+`"`) {
			t.Fatalf("the drawer's history lacks its drawn token %s", slug)
		}
	}
	for _, slug := range []string{"philosophy", "strategy"} {
		if strings.Contains(names(histB), `"`+slug+`"`) || strings.Contains(names(histS), `"`+slug+`"`) {
			t.Fatalf("a history other than the drawer's names the unkept token %s", slug)
		}
		if !trueHidden(&snap.State, seatRef(0))[slug] || hiddenFor(t, snap, seatRef(0))[slug] || !hiddenFor(t, snap, seatRef(1))[slug] {
			t.Fatalf("the model of %s is wrong", slug)
		}
	}
}
