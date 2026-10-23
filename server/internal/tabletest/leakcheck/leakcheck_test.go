package leakcheck_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

// sample is one viewer's view and history at one moment of a game.
type sample struct {
	label  string
	setup  fakeengine.Setup
	truth  *leakcheck.Truth
	game   fakeengine.Game
	moment int
	viewer int
	view   []byte
	hist   []byte
}

func viewer(v int) fakeengine.Viewer {
	if v < 0 {
		return fakeengine.SpectatorViewer()
	}
	return fakeengine.SeatViewer(fakeengine.Seat(v))
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// playout plays a game with seeded random picks.
func playout(setup fakeengine.Setup, pickSeed uint64) ([]fakeengine.Batch, error) {
	g, err := fakeengine.NewGame(setup)
	if err != nil {
		return nil, err
	}
	rng := rand.New(rand.NewPCG(pickSeed, 7))
	var bs []fakeengine.Batch
	for g.Point().Kind != fakeengine.PointNone {
		p := g.Point()
		b := fakeengine.Batch{Point: p.ID}
		for _, s := range p.Required.Seats() {
			cs := g.Legal(s)
			b.Picks = append(b.Picks, fakeengine.Pick{Seat: s, Key: cs[rng.IntN(len(cs))].Key})
		}
		if g, _, err = g.Resolve(b); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, nil
}

var pool = sync.OnceValues(func() ([]sample, error) {
	var out []sample
	for n := uint8(3); n <= 7; n++ {
		for game := range uint64(4) {
			for _, hide := range []bool{false, true} {
				setup := fakeengine.Setup{Seed: 0x9e3779b97f4a7c15 ^ (100*uint64(n) + game), Seats: n, Options: fakeengine.Options{HideAtEnd: hide}}
				bs, err := playout(setup, game)
				if err != nil {
					return nil, err
				}
				truth, err := leakcheck.FromFake(setup, bs)
				if err != nil {
					return nil, err
				}
				for k := 0; k <= len(bs); k++ {
					g, err := fakeengine.Replay(setup, bs[:k])
					if err != nil {
						return nil, err
					}
					for v := -1; v < int(n); v++ {
						out = append(out, sample{
							label: fmt.Sprintf("%d seats, game %d, hide %v, moment %d, viewer %d", n, game, hide, k, v),
							setup: setup, truth: truth, game: g, moment: k, viewer: v,
							view: mustJSON(g.ViewFor(viewer(v))), hist: mustJSON(g.History(viewer(v))),
						})
					}
				}
			}
		}
	}
	return out, nil
})

func samples(t *testing.T) []sample {
	t.Helper()
	ss, err := pool()
	if err != nil {
		t.Fatal(err)
	}
	return ss
}

func check(s *sample, kind leakcheck.Kind, data []byte) []leakcheck.Leak {
	c := leakcheck.New(s.truth, leakcheck.DefaultConfig)
	return c.Check(leakcheck.Doc{Label: s.label, Kind: kind, Viewer: s.viewer, Moment: s.moment, Data: data})
}

// TestCleanFakePasses: every view and history of every viewer, at every
// moment of games at three to seven seats under both end policies, passes
// every detector, and the sweep meets salvage and last-card points and both
// endings.
func TestCleanFakePasses(t *testing.T) {
	ss := samples(t)
	var salvage, lastCard, revealed, hidden int
	for i := range ss {
		s := &ss[i]
		if leaks := append(check(s, leakcheck.View, s.view), check(s, leakcheck.History, s.hist)...); len(leaks) > 0 {
			t.Fatalf("%s: %v", s.label, leaks)
		}
		m := s.truth.Moments[s.moment]
		switch {
		case m.Decider >= 0:
			salvage++
		case m.Kind == "last-card":
			lastCard++
		case m.Over && m.Reveal:
			revealed++
		case m.Over:
			hidden++
		}
	}
	if salvage == 0 || lastCard == 0 || revealed == 0 || hidden == 0 {
		t.Fatalf("the sweep missed a case: salvage %d, last card %d, revealed %d, hidden %d", salvage, lastCard, revealed, hidden)
	}
}

// fixture plants one leak in a clean document. Exactly one of view, hist
// and raw is set; each returns false when the sample cannot carry the leak.
type fixture struct {
	name string
	rule leakcheck.Rule
	view func(s *sample, v *fakeview.Table) bool
	hist func(s *sample, h []fakeview.HistoryEntry) bool
	raw  func(s *sample, v map[string]any) bool
}

func lastDesign(s *sample, seat int) string {
	d := ""
	for _, r := range s.truth.Resolutions[:s.moment] {
		if slices.Contains(r.Required, seat) {
			d = r.Actions[seat].Design
		}
	}
	return d
}

// otherEvent finds an event of another seat's play of kind play, returning
// its index and true design.
func otherEvent(s *sample, evs []fakeview.Event, res int, play string) (int, string) {
	if res < 0 {
		return -1, ""
	}
	for i, e := range evs {
		if e.Kind == "play" && e.Play == play && e.Seat != s.viewer {
			return i, s.truth.Resolutions[res].Events[i].Design
		}
	}
	return -1, ""
}

func seatView(s *sample, seat int) fakeview.Table { return s.game.ViewFor(viewer(seat)) }

var fixtures = []fixture{
	// A shuffle-derived id.
	{name: "a hand card with a copy suffix", rule: leakcheck.ShuffleID, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || len(v.You.Hand) == 0 {
			return false
		}
		v.You.Hand[0] += "#2"
		return true
	}},
	{name: "a deal position for a notebook card", rule: leakcheck.ShuffleID, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || len(v.You.Seen) == 0 || len(v.You.Seen[0].Hand) == 0 {
			return false
		}
		v.You.Seen[0].Hand[0] = "c7"
		return true
	}},
	{name: "a built card with a copy suffix", rule: leakcheck.ShuffleID, view: func(s *sample, v *fakeview.Table) bool {
		for i := range v.Seats {
			if len(v.Seats[i].Built) > 0 {
				v.Seats[i].Built[0] += ".3"
				return true
			}
		}
		return false
	}},

	// A deal-ordered hand.
	{name: "a hand in deal order", rule: leakcheck.DealOrder, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || len(v.You.Hand) < 2 || v.You.Hand[0] == v.You.Hand[len(v.You.Hand)-1] {
			return false
		}
		slices.Reverse(v.You.Hand)
		return true
	}},

	// A raw sold or tucked key.
	{name: "a sale's key in a live event", rule: leakcheck.RawKey, view: func(s *sample, v *fakeview.Table) bool {
		i, d := otherEvent(s, v.Events, s.moment-1, "sell")
		if i < 0 {
			return false
		}
		v.Events[i].Play = "sell:" + d
		return true
	}},
	{name: "a tuck's key in a seat's latest play", rule: leakcheck.RawKey, view: func(s *sample, v *fakeview.Table) bool {
		for i := range v.Seats {
			if l := v.Seats[i].Last; l != nil && l.Kind == "tuck" {
				l.Kind = "tuck:" + lastDesign(s, i)
				return true
			}
		}
		return false
	}},
	{name: "a tuck's key in the history", rule: leakcheck.RawKey, hist: func(s *sample, h []fakeview.HistoryEntry) bool {
		for k := range h {
			if i, d := otherEvent(s, h[k].Events, k, "tuck"); i >= 0 {
				h[k].Events[i].Play = "tuck:" + d
				return true
			}
		}
		return false
	}},
	{name: "another seat's sold card in a live event", rule: leakcheck.HiddenCard, view: func(s *sample, v *fakeview.Table) bool {
		i, d := otherEvent(s, v.Events, s.moment-1, "sell")
		if i < 0 {
			return false
		}
		v.Events[i].Design = d
		return true
	}},
	{name: "another seat's tucked card in its latest play", rule: leakcheck.HiddenCard, view: func(s *sample, v *fakeview.Table) bool {
		for i := range v.Seats {
			if l := v.Seats[i].Last; l != nil && l.Kind == "tuck" && i != s.viewer {
				l.Design = lastDesign(s, i)
				return true
			}
		}
		return false
	}},
	{name: "another seat's tucked card in the history", rule: leakcheck.HiddenCard, hist: func(s *sample, h []fakeview.HistoryEntry) bool {
		for k := range h {
			if i, d := otherEvent(s, h[k].Events, k, "tuck"); i >= 0 {
				h[k].Events[i].Design = d
				return true
			}
		}
		return false
	}},
	{name: "another seat's hand", rule: leakcheck.HiddenCard, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || s.truth.Moments[s.moment].Over {
			return false
		}
		other := s.truth.Moments[s.moment].Hands[(s.viewer+1)%s.truth.Seats]
		if !slices.ContainsFunc(other, func(c string) bool { return !slices.Contains(v.You.Hand, c) }) {
			return false
		}
		v.You.Hand = other
		return true
	}},
	{name: "the pile and box before the end", rule: leakcheck.HiddenCard, view: func(s *sample, v *fakeview.Table) bool {
		m := s.truth.Moments[s.moment]
		if m.Over {
			return false
		}
		v.Reveal = &fakeview.Reveal{Tucked: m.Tucked, Pile: m.Pile, Box: m.Box}
		return true
	}},
	{name: "the reveal with the reveal off", rule: leakcheck.HiddenCard, view: func(s *sample, v *fakeview.Table) bool {
		m := s.truth.Moments[s.moment]
		if !m.Over || m.Reveal {
			return false
		}
		v.Reveal = &fakeview.Reveal{Tucked: m.Tucked, Pile: m.Pile, Box: m.Box}
		return true
	}},

	// A pending or observation hash.
	{name: "a pending hash", rule: leakcheck.Hash, raw: func(s *sample, v map[string]any) bool {
		v["pendingHash"] = s.game.Fingerprint()
		return true
	}},
	{name: "another seat's observation hash", rule: leakcheck.Hash, view: func(s *sample, v *fakeview.Table) bool {
		o := s.game.Observe(fakeengine.Seat((s.viewer + 1) % s.truth.Seats))
		v.Pass = o.Fingerprint()
		return true
	}},
	{name: "the seed as a number", rule: leakcheck.Hash, raw: func(s *sample, v map[string]any) bool {
		v["seed"] = json.Number(strconv.FormatUint(s.setup.Seed, 10))
		return true
	}},
	{name: "the game fingerprint in the history", rule: leakcheck.Hash, hist: func(s *sample, h []fakeview.HistoryEntry) bool {
		if len(h) == 0 {
			return false
		}
		h[0].Point += "@" + s.game.Fingerprint()
		return true
	}},

	// An opponent's offer.
	{name: "another seat's offered keys", rule: leakcheck.ForeignOffer, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || v.Kind != "turn" {
			return false
		}
		other := seatView(s, (s.viewer+1)%s.truth.Seats).You.Choices
		own := s.truth.Moments[s.moment].Legal[s.viewer]
		if !slices.ContainsFunc(other, func(c fakeview.Choice) bool { return !slices.Contains(own, c.Key) }) {
			return false
		}
		v.You.Choices = other
		return true
	}},
	{name: "the salvage offer shown to another seat", rule: leakcheck.ForeignOffer, view: func(s *sample, v *fakeview.Table) bool {
		d := s.truth.Moments[s.moment].Decider
		if v.You == nil || d < 0 || d == s.viewer {
			return false
		}
		v.You.Offer = seatView(s, d).You.Offer
		return len(v.You.Offer) > 0
	}},
	{name: "another seat's private section", rule: leakcheck.ForeignOffer, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil {
			return false
		}
		v.You = seatView(s, (s.viewer+1)%s.truth.Seats).You
		return true
	}},
	{name: "a private section shown to a spectator", rule: leakcheck.ForeignOffer, view: func(s *sample, v *fakeview.Table) bool {
		if s.viewer >= 0 {
			return false
		}
		v.You = seatView(s, 0).You
		return true
	}},

	// Canonical order.
	{name: "offered keys out of the engine's order", rule: leakcheck.Order, view: func(s *sample, v *fakeview.Table) bool {
		if v.You == nil || len(v.You.Choices) < 2 {
			return false
		}
		slices.Reverse(v.You.Choices)
		return true
	}},
	{name: "required seats out of seat order", rule: leakcheck.Order, view: func(s *sample, v *fakeview.Table) bool {
		if len(v.Required) < 2 {
			return false
		}
		slices.Reverse(v.Required)
		return true
	}},
	{name: "a city's built cards out of catalog order", rule: leakcheck.DealOrder, view: func(s *sample, v *fakeview.Table) bool {
		for i := range v.Seats {
			if len(v.Seats[i].Built) >= 2 {
				slices.Reverse(v.Seats[i].Built)
				return true
			}
		}
		return false
	}},
	{name: "a history out of point order", rule: leakcheck.Order, hist: func(s *sample, h []fakeview.HistoryEntry) bool {
		if len(h) < 2 {
			return false
		}
		h[0], h[1] = h[1], h[0]
		return true
	}},

	// The whitelist.
	{name: "a field outside the whitelist", rule: leakcheck.UnknownField, raw: func(s *sample, v map[string]any) bool {
		v["debug"] = "x"
		return true
	}},
}

// TestPlantedLeaksFail plants each leak in a clean document of the sweep
// and requires its own detector to report it.
func TestPlantedLeaksFail(t *testing.T) {
	ss := samples(t)
	rules := map[leakcheck.Rule]bool{}
	for _, f := range fixtures {
		planted := false
		for i := range ss {
			s := &ss[i]
			var kind leakcheck.Kind
			var data []byte
			switch {
			case f.view != nil:
				var v fakeview.Table
				if err := json.Unmarshal(s.view, &v); err != nil {
					t.Fatal(err)
				}
				if !f.view(s, &v) {
					continue
				}
				kind, data = leakcheck.View, mustJSON(v)
			case f.hist != nil:
				var h []fakeview.HistoryEntry
				if err := json.Unmarshal(s.hist, &h); err != nil {
					t.Fatal(err)
				}
				if !f.hist(s, h) {
					continue
				}
				kind, data = leakcheck.History, mustJSON(h)
			default:
				var v map[string]any
				dec := json.NewDecoder(bytes.NewReader(s.view))
				dec.UseNumber()
				if err := dec.Decode(&v); err != nil {
					t.Fatal(err)
				}
				if !f.raw(s, v) {
					continue
				}
				kind, data = leakcheck.View, mustJSON(v)
			}
			planted = true
			leaks := check(s, kind, data)
			if !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == f.rule }) {
				t.Errorf("%s (%s): %s did not report it; got %v", f.name, s.label, f.rule, leaks)
			}
			rules[f.rule] = true
			break
		}
		if !planted {
			t.Errorf("%s: no document of the sweep could carry it", f.name)
		}
	}
	for _, r := range []leakcheck.Rule{leakcheck.ShuffleID, leakcheck.DealOrder, leakcheck.RawKey, leakcheck.HiddenCard,
		leakcheck.Hash, leakcheck.ForeignOffer, leakcheck.Order, leakcheck.UnknownField} {
		if !rules[r] {
			t.Errorf("no fixture exercises %s", r)
		}
	}
}

// TestFrames: a whole frame gets the field-independent detectors, with the
// frame's own key path allowed.
func TestFrames(t *testing.T) {
	s := &samples(t)[1] // seat 0 at the first moment
	cfg := leakcheck.Config{KeyPaths: []string{"view.you.choices[].key"}}
	frame := func(extra map[string]any) []byte {
		f := map[string]any{"t": "state", "ref": s.truth.Moments[0].Point + "@0", "view": json.RawMessage(s.view)}
		for k, v := range extra {
			f[k] = v
		}
		return mustJSON(f)
	}
	c := leakcheck.New(s.truth, cfg)
	doc := func(data []byte) []leakcheck.Leak {
		return c.Check(leakcheck.Doc{Label: "frame", Kind: leakcheck.Frame, Viewer: 0, Data: data})
	}
	if leaks := doc(frame(nil)); len(leaks) != 0 {
		t.Fatalf("a clean frame: %v", leaks)
	}
	for name, tc := range map[string]struct {
		extra map[string]any
		rule  leakcheck.Rule
	}{
		"a key outside the view's choices": {map[string]any{"ack": map[string]any{"key": "sell:amber"}}, leakcheck.RawKey},
		"a pending hash":                   {map[string]any{"pending": s.game.Fingerprint()}, leakcheck.Hash},
		"a card id with a copy suffix":     {map[string]any{"card": "amber#2"}, leakcheck.ShuffleID},
	} {
		leaks := doc(frame(tc.extra))
		if !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == tc.rule }) {
			t.Errorf("%s: %s did not report it; got %v", name, tc.rule, leaks)
		}
	}
}

// TestLogCapture plays a game through the in-process table with its
// logger captured, and requires the log to name no key, card or secret;
// then plants a key and a fingerprint in the log and requires each to fail.
func TestLogCapture(t *testing.T) {
	setup := fakeengine.Setup{Seed: 0x51ed270b27f3a1c9, Seats: 4}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}))
	tb, err := direct.Factory(direct.Options{Setup: setup, Logger: logger, SeqClients: 2})(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	run, err := tabletest.Play(t.Context(), tb, tabletest.Config{Seats: 4, PickSeed: 3, ArrivalSeed: 4, Script: tabletest.ScriptOptions{Drafts: 50, Resends: 40}})
	if err != nil {
		t.Fatal(err)
	}
	var bs []fakeengine.Batch
	for _, b := range run.Canonical.Batches {
		eb := fakeengine.Batch{Point: b.Point}
		for _, p := range b.Picks {
			eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: fakeengine.Seat(p.Seat), Key: p.Key})
		}
		bs = append(bs, eb)
	}
	truth, err := leakcheck.FromFake(setup, bs)
	if err != nil {
		t.Fatal(err)
	}
	c := leakcheck.New(truth, leakcheck.DefaultConfig)
	if strings.Count(buf.String(), "\n") < len(run.Trace) {
		t.Fatalf("the log has %d lines for %d submissions", strings.Count(buf.String(), "\n"), len(run.Trace))
	}
	if leaks := c.Check(leakcheck.Doc{Label: "log", Kind: leakcheck.Log, Data: buf.Bytes()}); len(leaks) != 0 {
		t.Fatalf("the clean log: %v", leaks)
	}
	key := run.Canonical.Batches[0].Picks[0].Key
	g, _ := fakeengine.Replay(setup, bs[:1])
	for _, plant := range []struct {
		line  []any
		rules []leakcheck.Rule
	}{
		{[]any{"op", "pick", "key", key}, []leakcheck.Rule{leakcheck.RawKey, leakcheck.HiddenCard}},
		{[]any{"op", "pick", "pending", g.Fingerprint()}, []leakcheck.Rule{leakcheck.Hash}},
	} {
		buf.Reset()
		logger.Info("table", plant.line...)
		leaks := c.Check(leakcheck.Doc{Label: "log", Kind: leakcheck.Log, Data: buf.Bytes()})
		for _, r := range plant.rules {
			if !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == r }) {
				t.Errorf("%q: %s did not report it; got %v", buf.String(), r, leaks)
			}
		}
	}
}
