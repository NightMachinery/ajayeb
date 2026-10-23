package fakeengine

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview"
)

// randomBatch picks a uniformly random offered key for every required seat.
func randomBatch(t *testing.T, g *Game, rng *rand.Rand) Batch {
	t.Helper()
	p := g.Point()
	b := Batch{Point: p.ID}
	for _, s := range p.Required.Seats() {
		cs := g.Legal(s)
		if len(cs) == 0 {
			t.Fatalf("%s: seat %d is required but offered nothing", p.ID, s)
		}
		b.Picks = append(b.Picks, Pick{Seat: s, Key: cs[rng.IntN(len(cs))].Key})
	}
	return b
}

// sweepStats counts what a sweep reached.
type sweepStats struct {
	salvage, lastCard, twoInARow, salvageNothing, duplicateInHand int
}

// playout plays one game with random keys, checking after every point.
func playout(t *testing.T, s Setup, rng *rand.Rand, stats *sweepStats) (Game, []Batch) {
	t.Helper()
	g, err := NewGame(s)
	if err != nil {
		t.Fatalf("NewGame(%+v): %v", s, err)
	}
	if err := g.CheckInvariants(); err != nil {
		t.Fatalf("setup %+v: %v", s, err)
	}
	var batches []Batch
	var live [][]Event
	prevOneSeat := false
	for g.Point().Kind != PointNone {
		if len(batches) >= MaxPoints {
			t.Fatalf("setup %+v: more than %d points", s, MaxPoints)
		}
		p := g.Point()
		oneSeat := p.Kind == PointSalvage || p.Kind == PointLastCard
		if stats != nil {
			switch p.Kind {
			case PointSalvage:
				stats.salvage++
			case PointLastCard:
				stats.lastCard++
			}
			if oneSeat && prevOneSeat {
				stats.twoInARow++
			}
			for i := Seat(0); uint8(i) < s.Seats; i++ {
				h := g.State.Hands[i]
				for j := 1; j < HandSize; j++ {
					if h[j] != 0 && h[j] == h[j-1] {
						stats.duplicateInHand++
					}
				}
			}
		}
		prevOneSeat = oneSeat
		b := randomBatch(t, &g, rng)
		next, evs, err := g.Resolve(b)
		if err != nil {
			t.Fatalf("setup %+v: Resolve(%+v): %v", s, b, err)
		}
		if len(evs) > MaxEvents {
			t.Fatalf("%d events, more than MaxEvents", len(evs))
		}
		for _, e := range evs {
			if stats != nil && e.Kind == EvSalvageNothing {
				stats.salvageNothing++
			}
		}
		g = next
		if err := g.CheckInvariants(); err != nil {
			t.Fatalf("setup %+v after %s: %v", s, b.Point, err)
		}
		batches = append(batches, b)
		live = append(live, evs)
	}
	if !g.State.Result.Over {
		t.Fatalf("setup %+v: no result", s)
	}
	// History equals the live streams, filtered the same way.
	for _, v := range viewers(s.Seats) {
		h := g.History(v)
		if len(h) != len(live) {
			t.Fatalf("history has %d entries, want %d", len(h), len(live))
		}
		for i := range live {
			if !reflect.DeepEqual(h[i].Events, filterEvents(live[i], v)) {
				t.Fatalf("history entry %d for %+v differs from the live events", i, v)
			}
		}
	}
	return g, batches
}

func viewers(n uint8) []Viewer {
	vs := []Viewer{SpectatorViewer()}
	for s := Seat(0); uint8(s) < n; s++ {
		vs = append(vs, SeatViewer(s))
	}
	return vs
}

func TestDeckSizes(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		if got := len(deckFor(1, n)); got != HandSize*int(n) {
			t.Errorf("Age 1 at %d seats has %d copies, want %d", n, got, HandSize*int(n))
		}
		if got := len(deckFor(2, n)); got != HandSize*int(n)+MaxBox {
			t.Errorf("Age 2 at %d seats has %d copies, want %d", n, got, HandSize*int(n)+MaxBox)
		}
	}
	seen := map[string]bool{}
	for _, d := range Designs() {
		if seen[d.Slug] || strings.ContainsAny(d.Slug, "0123456789-:#") {
			t.Errorf("slug %q is repeated or not a plain word", d.Slug)
		}
		seen[d.Slug] = true
	}
	for _, b := range BoardSlugs() {
		if seen[b] {
			t.Errorf("board slug %q is also a design slug", b)
		}
	}
}

// TestSweep plays seeded random games at every seat count under both end
// policies and both seating rules, checking invariants after every point,
// history against the live events, replay against play, and the views'
// JSON shape; and requires the sweep to reach both ability points, two
// one-seat points in a row, a salvage with nothing to take and a hand with
// two copies of one design.
func TestSweep(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		var stats sweepStats
		rng := rand.New(rand.NewPCG(uint64(n), 7))
		for i := 0; i < 400; i++ {
			s := Setup{Seed: uint64(1000*int(n) + i), Seats: n, Options: Options{HideAtEnd: i%2 == 1, Seating: SeatingRule(i % 3 / 2)}}
			g, batches := playout(t, s, rng, &stats)
			r, err := Replay(s, batches)
			if err != nil {
				t.Fatalf("Replay: %v", err)
			}
			if r != g || r.Fingerprint() != g.Fingerprint() {
				t.Fatalf("setup %+v: replay differs from play", s)
			}
			for _, v := range viewers(n) {
				checkViewJSON(t, g.ViewFor(v), v, s.Options.HideAtEnd)
			}
		}
		if stats.salvage == 0 || stats.lastCard == 0 || stats.twoInARow == 0 || stats.salvageNothing == 0 || stats.duplicateInHand == 0 {
			t.Errorf("%d seats: the sweep missed a path: %+v", n, stats)
		}
	}
}

// checkViewJSON checks a final view's shape: no null slices, You only for a
// seat, Reveal exactly when the end reveal is on.
func checkViewJSON(t *testing.T, tv fakeview.Table, v Viewer, hide bool) {
	t.Helper()
	data, err := json.Marshal(tv)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "null") {
		t.Fatalf("view for %+v has a null: %s", v, data)
	}
	if (tv.You != nil) == v.Spectator {
		t.Fatalf("view for %+v: You present %t", v, tv.You != nil)
	}
	if tv.Result == nil || (tv.Reveal != nil) == hide {
		t.Fatalf("final view for %+v: result %v, reveal %v with HideAtEnd %t", v, tv.Result, tv.Reveal, hide)
	}
}

func TestFingerprintDependsOnSeed(t *testing.T) {
	a, _ := NewGame(Setup{Seed: 1, Seats: 5})
	b, _ := NewGame(Setup{Seed: 1, Seats: 5})
	c, _ := NewGame(Setup{Seed: 2, Seats: 5})
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("the same setup gave two fingerprints")
	}
	if a.Fingerprint() == c.Fingerprint() {
		t.Fatal("two seeds gave one fingerprint")
	}
	if len(a.Fingerprint()) != 16 {
		t.Fatalf("fingerprint %q is not 16 hex digits", a.Fingerprint())
	}
}

func TestNewGameRejectsBadSetup(t *testing.T) {
	for _, s := range []Setup{{Seats: 2}, {Seats: 8}, {Seats: 4, Options: Options{Seating: 9}}} {
		if _, err := NewGame(s); !errors.Is(err, ErrSetup) {
			t.Errorf("NewGame(%+v) = %v, want ErrSetup", s, err)
		}
	}
}

func TestResolveRejects(t *testing.T) {
	g, _ := NewGame(Setup{Seed: 42, Seats: 4})
	before := g
	p := g.Point()
	good := Batch{Point: p.ID}
	for _, s := range p.Required.Seats() {
		good.Picks = append(good.Picks, Pick{Seat: s, Key: g.Legal(s)[0].Key})
	}
	swap := Batch{Point: p.ID, Picks: append([]Pick{}, good.Picks...)}
	swap.Picks[0], swap.Picks[1] = swap.Picks[1], swap.Picks[0]
	dup := Batch{Point: p.ID, Picks: append([]Pick{}, good.Picks...)}
	dup.Picks[1] = dup.Picks[0]
	illegal := Batch{Point: p.ID, Picks: append([]Pick{}, good.Picks...)}
	illegal.Picks[2].Key = "salvage:amber"
	cases := []struct {
		name string
		b    Batch
		want error
	}{
		{"another point", Batch{Point: "age-1-turn-2", Picks: good.Picks}, ErrWrongPoint},
		{"a seat missing", Batch{Point: p.ID, Picks: good.Picks[:3]}, ErrBatch},
		{"unsorted", swap, ErrBatch},
		{"a seat twice", dup, ErrBatch},
		{"an unoffered key", illegal, ErrIllegal},
	}
	for _, c := range cases {
		if _, _, err := g.Resolve(c.b); !errors.Is(err, c.want) {
			t.Errorf("%s: Resolve = %v, want %v", c.name, err, c.want)
		}
		if g != before {
			t.Fatalf("%s: a refused batch changed the game", c.name)
		}
	}
	if _, _, err := g.Resolve(good); err != nil {
		t.Fatalf("the good batch: %v", err)
	}
	over, _ := playout(t, Setup{Seed: 3, Seats: 3}, rand.New(rand.NewPCG(1, 2)), nil)
	if _, _, err := over.Resolve(Batch{}); !errors.Is(err, ErrOver) {
		t.Fatalf("Resolve after the end = %v, want ErrOver", err)
	}
	if len(over.Legal(0)) != 0 || over.Point().Kind != PointNone || over.Point().ID != "" {
		t.Fatalf("a finished game offers %v at %+v", over.Legal(0), over.Point())
	}
}

func TestParseKey(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 5))
	for i := 0; i < 50; i++ {
		g, _ := NewGame(Setup{Seed: uint64(i), Seats: 7})
		for g.Point().Kind != PointNone {
			for _, s := range g.Point().Required.Seats() {
				for _, c := range g.Legal(s) {
					a, ok := ParseKey(c.Key)
					if !ok || a.Key() != c.Key || a != (Action{Kind: c.Kind, Design: c.Design, Left: c.Left, Right: c.Right}) {
						t.Fatalf("ParseKey(%q) = %+v, %t", c.Key, a, ok)
					}
				}
			}
			g, _, _ = g.Resolve(randomBatch(t, &g, rng))
		}
	}
	for _, k := range []string{"", "sell", "sell:", "sell:amber:", "SELL:amber", "sell:copper", "build:amber:01:0",
		"build:amber:0", "build:amber:0:0:0", "tuck:amber:0:0", "build:amber:-1:1", "salvage:amber#2"} {
		if _, ok := ParseKey(k); ok {
			t.Errorf("ParseKey(%q) accepted a non-canonical key", k)
		}
	}
}

func TestSeatOrder(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		chosen := SeatOrder(9, n, SeatingChosen)
		random := SeatOrder(9, n, SeatingRandom)
		if random != SeatOrder(9, n, SeatingRandom) {
			t.Fatal("SeatOrder is not deterministic")
		}
		var seen [MaxSeats]bool
		for s := uint8(0); s < MaxSeats; s++ {
			if s >= n {
				if chosen[s] != 0 || random[s] != 0 {
					t.Fatalf("entries beyond %d seats are not zero: %v %v", n, chosen, random)
				}
				continue
			}
			if chosen[s] != s || random[s] >= n || seen[random[s]] {
				t.Fatalf("SeatOrder(%d) chosen %v random %v is not a permutation", n, chosen, random)
			}
			seen[random[s]] = true
		}
	}
	if SeatOrder(1, 2, SeatingRandom) != [MaxSeats]uint8{} {
		t.Fatal("an invalid seat count gave an assignment")
	}
}

// TestNoninterference changes what a seat may not know (other hands of the
// same sizes, Age 2's deal, the box and the pile's composition) and checks
// that the seat's view, observation and observation fingerprint stay the
// same, and that the spectator's view does too.
func TestNoninterference(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 12))
	for i := 0; i < 200; i++ {
		g, _ := NewGame(Setup{Seed: uint64(i), Seats: uint8(MinSeats + i%5)})
		steps := rng.IntN(4) // four points at least, so the game is never over here
		for k := 0; k < steps && g.Point().Kind != PointNone; k++ {
			g, _, _ = g.Resolve(randomBatch(t, &g, rng))
		}
		for s := Seat(0); uint8(s) < g.State.Seats; s++ {
			h := g
			for o := Seat(0); uint8(o) < h.State.Seats; o++ {
				if o == s {
					continue
				}
				for j, d := range h.State.Hands[o] {
					if d != 0 {
						h.State.Hands[o][j] = 15 + DesignID(j)%4 // relic ids, sorted
					}
				}
				for j := range h.State.Deals[o] {
					if h.State.Deals[o][j] != 0 {
						h.State.Deals[o][j] = 8
					}
				}
			}
			h.State.Box = [MaxBox]DesignID{1, 1, 1, 1}
			var pileCount uint8
			for d := range h.State.Pile {
				pileCount += h.State.Pile[d]
				h.State.Pile[d] = 0
			}
			h.State.Pile[2] = pileCount
			if g.State.Phase != PhaseSalvage || g.State.Decider != s {
				compareView(t, &g, &h, SeatViewer(s))
				og, oh := g.Observe(s), h.Observe(s)
				if og != oh || og.Fingerprint() != oh.Fingerprint() {
					t.Fatalf("seat %d's observation depends on hidden cards", s)
				}
			}
			compareView(t, &g, &h, SpectatorViewer())
		}
	}
}

func compareView(t *testing.T, g, h *Game, v Viewer) {
	t.Helper()
	a, _ := json.Marshal(g.ViewFor(v))
	b, _ := json.Marshal(h.ViewFor(v))
	if string(a) != string(b) {
		t.Fatalf("the view for %+v depends on hidden cards:\n%s\n%s", v, a, b)
	}
}

// TestViewTags checks the DTO rules: every field has an explicit JSON name,
// and omitempty appears only on pointer fields.
func TestViewTags(t *testing.T) {
	for _, v := range []any{fakeview.Table{}, fakeview.Seat{}, fakeview.Play{}, fakeview.Private{}, fakeview.Choice{},
		fakeview.OfferEntry{}, fakeview.SeenEntry{}, fakeview.Event{}, fakeview.HistoryEntry{}, fakeview.Result{},
		fakeview.Score{}, fakeview.Reveal{}} {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tag := f.Tag.Get("json")
			if tag == "" || strings.HasPrefix(tag, ",") {
				t.Errorf("%s.%s has no JSON name", typ.Name(), f.Name)
			}
			if strings.Contains(tag, "omitempty") && f.Type.Kind() != reflect.Pointer {
				t.Errorf("%s.%s uses omitempty on a non-pointer", typ.Name(), f.Name)
			}
		}
	}
}

// TestPrivateSections spot-checks the whitelist: a seat sees its own hand
// and offer only, and the salvage offer appears only to its decider during
// its point.
func TestPrivateSections(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	found := false
	for i := 0; i < 300 && !found; i++ {
		g, _ := NewGame(Setup{Seed: uint64(i), Seats: 6})
		for g.Point().Kind != PointNone {
			for s := Seat(0); s < 6; s++ {
				you := g.ViewFor(SeatViewer(s)).You
				if strings.Join(you.Hand, ",") != strings.Join(designSlugs(g.State.Hands[s][:]), ",") || len(you.Choices) != len(g.Legal(s)) {
					t.Fatalf("seat %d's private section is not its own", s)
				}
				if len(you.Offer) > 0 != (g.State.Phase == PhaseSalvage && g.State.Decider == s) {
					t.Fatalf("seat %d sees an offer %v outside its salvage point", s, you.Offer)
				}
				found = found || len(you.Offer) > 0
			}
			g, _, _ = g.Resolve(randomBatch(t, &g, rng))
		}
	}
	if !found {
		t.Fatal("no salvage offer was reached")
	}
}
