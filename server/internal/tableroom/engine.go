package tableroom

// This file is the actor's only engine seam. Until
// M16.haftgane-engine-contract it binds the fake engine; that slice
// replaces the import, these aliases and the view conversion with the real
// engine's (server/internal/haftgane/rules and view). Nothing else in the
// package names fakeengine or fakeview.

import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type (
	game    = fakeengine.Game
	setup   = fakeengine.Setup
	seatNum = fakeengine.Seat
	point   = fakeengine.Point
	batch   = fakeengine.Batch
	pick    = fakeengine.Pick
	viewer  = fakeengine.Viewer
)

const (
	// RulesVersion is the bound engine's rules version, the third member
	// of D28's saved-room tuple.
	RulesVersion = fakeengine.RulesVersion
	// MinSeats and MaxSeats bound a table.
	MinSeats = fakeengine.MinSeats
	MaxSeats = fakeengine.MaxSeats
	// maxBatches bounds a game's complete batches.
	maxBatches = fakeengine.MaxPoints
	// policyVersion names the bot policy whose work rows a table persists.
	// M17 replaces the policy and versions it.
	policyVersion = "fake-1"
)

var (
	newGame   = fakeengine.NewGame
	replay    = fakeengine.Replay
	seatOrder = fakeengine.SeatOrder
)

func pointOver(p point) bool { return p.Kind == fakeengine.PointNone }

func seatingRule(random bool) fakeengine.SeatingRule {
	if random {
		return fakeengine.SeatingRandom
	}
	return fakeengine.SeatingChosen
}

func makeSetup(seed uint64, seats int, random, hideAtEnd bool) setup {
	return setup{Seed: seed, Seats: uint8(seats), Options: fakeengine.Options{Seating: seatingRule(random), HideAtEnd: hideAtEnd}}
}

func seatViewer(s int) viewer { return fakeengine.SeatViewer(seatNum(s)) }

func spectatorViewer() viewer { return fakeengine.SpectatorViewer() }

// legalKeys returns the keys offered to seat s, in canonical order.
func legalKeys(g *game, s int) []string {
	chs := g.Legal(seatNum(s))
	out := make([]string, 0, len(chs))
	for _, ch := range chs {
		out = append(out, ch.Key)
	}
	return out
}

// observationHash is the canonical hash of the seat's permitted
// observation: bot entropy only, never a frame, a log or a download.
func observationHash(g *game, s int) string {
	o := g.Observe(seatNum(s))
	return o.Fingerprint()
}

// projection is one viewer's engine view, split into the frame's parts.
type projection struct {
	view   wire.TableView
	events []wire.TableEvent // the latest resolution's events, filtered
}

func project(g *game, v viewer) projection {
	t := g.ViewFor(v)
	p := projection{
		view: wire.TableView{
			Phase: t.Phase, Age: t.Age, Turn: t.Turn, Pass: t.Pass, Pile: t.Pile,
			Seats: make([]wire.TableCity, 0, len(t.Seats)),
		},
		events: wireEvents(t.Events),
	}
	for _, s := range t.Seats {
		c := wire.TableCity{
			Seat: s.Seat, Board: s.Board, Coins: s.Coins, Built: cloneStrings(s.Built), Stages: s.Stages,
			HandSize: s.HandSize, HasKept: s.HasKept,
		}
		if s.Last != nil {
			c.Last = &wire.TablePlay{Kind: s.Last.Kind, Design: s.Last.Design, Stage: s.Last.Stage, Left: s.Last.Left, Right: s.Last.Right}
		}
		p.view.Seats = append(p.view.Seats, c)
	}
	if t.You != nil {
		y := &wire.TablePrivate{
			Seat: t.You.Seat, Hand: cloneStrings(t.You.Hand), Kept: t.You.Kept, Tucked: cloneStrings(t.You.Tucked),
			Choices: make([]wire.TableChoice, 0, len(t.You.Choices)),
			Offer:   make([]wire.TableOffer, 0, len(t.You.Offer)),
			Seen:    make([]wire.TableSeen, 0, len(t.You.Seen)),
		}
		for _, ch := range t.You.Choices {
			y.Choices = append(y.Choices, wire.TableChoice{Key: ch.Key, Kind: ch.Kind, Design: ch.Design, Left: ch.Left, Right: ch.Right})
		}
		for _, o := range t.You.Offer {
			y.Offer = append(y.Offer, wire.TableOffer{Design: o.Design, Count: o.Count, Legal: o.Legal})
		}
		for _, s := range t.You.Seen {
			y.Seen = append(y.Seen, wire.TableSeen{Point: s.Point, Hand: cloneStrings(s.Hand), Played: s.Played, Action: s.Action})
		}
		p.view.You = y
	}
	if t.Result != nil {
		r := &wire.TableResult{Scores: make([]wire.TableScore, 0, len(t.Result.Scores)), Rank: cloneInts(t.Result.Rank), Winners: cloneInts(t.Result.Winners)}
		for _, s := range t.Result.Scores {
			r.Scores = append(r.Scores, wire.TableScore{Seat: s.Seat, Built: s.Built, Stages: s.Stages, Treasure: s.Treasure, Total: s.Total})
		}
		p.view.Result = r
	}
	if t.Reveal != nil {
		r := &wire.TableReveal{Tucked: make([][]string, 0, len(t.Reveal.Tucked)), Pile: cloneStrings(t.Reveal.Pile), Box: cloneStrings(t.Reveal.Box)}
		for _, tk := range t.Reveal.Tucked {
			r.Tucked = append(r.Tucked, cloneStrings(tk))
		}
		p.view.Reveal = r
	}
	return p
}

// history is the viewer's whole history, one entry per complete batch.
func history(g *game, v viewer) []wire.TableHistoryEntry {
	h := g.History(v)
	out := make([]wire.TableHistoryEntry, 0, len(h))
	for i, e := range h {
		out = append(out, wire.TableHistoryEntry{Batch: i + 1, Events: wireEvents(e.Events)})
	}
	return out
}

func wireEvents(evs []fakeview.Event) []wire.TableEvent {
	out := make([]wire.TableEvent, 0, len(evs))
	for _, e := range evs {
		out = append(out, wire.TableEvent{
			Kind: e.Kind, Seat: e.Seat, Play: e.Play, Design: e.Design, Stage: e.Stage,
			Left: e.Left, Right: e.Right, Age: e.Age, Pass: e.Pass,
		})
	}
	return out
}

func cloneStrings(xs []string) []string { return append(make([]string, 0, len(xs)), xs...) }

func cloneInts(xs []int) []int { return append(make([]int, 0, len(xs)), xs...) }

// pointKind names a point's kind for the frame.
func pointKind(p point) string { return p.Kind.String() }
