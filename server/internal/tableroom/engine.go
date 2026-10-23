package tableroom

// The actor's concrete binding to the real engine. The small wrappers keep
// transport point references outside rules, and never reconstruct an action.
import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	hview "github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type game struct{ rules.Game }
type setup = rules.Setup
type seatNum = rules.Seat
type batch = rules.Batch
type pick = rules.Pick
type viewer = rules.Seat
type seatSet rules.SeatSet

func (s seatSet) Has(v seatNum) bool { return rules.SeatSet(s).Has(v) }
func (s seatSet) Count() int         { return rules.SeatSet(s).Count() }
func (s seatSet) Seats() []seatNum {
	out := []seatNum{}
	for v := seatNum(0); v < MaxSeats; v++ {
		if s.Has(v) {
			out = append(out, v)
		}
	}
	return out
}

type point struct {
	ID       string
	Kind     rules.PointKind
	Required seatSet
}

func (g *game) Point() point { p := g.Game.Point(); return point{p.ID(), p.Kind, seatSet(p.Required)} }
func (g *game) Resolve(b batch) (game, []rules.Event, error) {
	n, e, err := g.Game.Resolve(b)
	return game{n}, e, err
}
func newGame(s setup) (game, error)           { g, e := rules.NewGame(s); return game{g}, e }
func replay(s setup, b []batch) (game, error) { g, e := rules.Replay(s, b); return game{g}, e }

const (
	RulesVersion = rules.RulesVersion
	MinSeats     = rules.MinSeats
	MaxSeats     = rules.MaxSeats
	maxBatches   = rules.MaxPoints
	// Exact offered-key fallback only. M17.fixed-preset-room-bots replaces
	// this policy and versions the persisted work rows.
	policyVersion = "offered-key-1"
)

var seatOrder = rules.SeatOrder

func pointOver(p point) bool { return p.Kind == rules.PointNone }
func seatingRule(random bool) rules.SeatingRule {
	if random {
		return rules.SeatingRandom
	}
	return rules.SeatingChosen
}
func makeSetup(seed uint64, seats int, random, hideAtEnd bool) setup {
	return setup{Seed: seed, Seats: uint8(seats), Options: rules.Options{Sides: rules.SidesDay, Seating: seatingRule(random), HideAtEnd: hideAtEnd}}
}
func recordSetup(r SetupRecord) (setup, error) {
	s := makeSetup(r.Seed, r.Seats, r.RandomSeating, r.HideAtEnd)
	switch r.Sides {
	case "", "day":
	case "choose":
		s.Options.Sides = rules.SidesChoose
	case "random":
		s.Options.Sides = rules.SidesRandom
	default:
		return setup{}, invalid("the side rule")
	}
	return s, nil
}
func seatViewer(s int) viewer { return viewer(s) }
func spectatorViewer() viewer { return rules.NoSeat }
func legalKeys(g *game, s int) []string {
	cs := g.Legal(seatNum(s))
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Key())
	}
	return out
}
func observationHash(g *game, s int) string { return g.Observe(seatNum(s)).Fingerprint() }

type projection struct {
	view   wire.TableView
	events []wire.TableEvent
}

func project(g *game, v viewer) projection {
	t := g.ViewFor(v)
	return projection{wire.TableView{Phase: t.Phase, Age: t.Age, Turn: t.Turn, Pass: t.Pass, Pile: t.Pile, Seats: t.Seats, You: t.You, Result: t.Result, Reveal: t.Reveal, CanDownload: t.CanDownload}, wireEvents(t.Events)}
}

// Entry i is batch i, including HistoryStartPoint at batch zero, for every
// side rule. Restore and every transaction validate the full rules history.
func history(g *game, v viewer) []wire.TableHistoryEntry {
	h, err := g.History(v)
	if err != nil {
		return []wire.TableHistoryEntry{}
	}
	out := make([]wire.TableHistoryEntry, 0, len(h))
	for i, e := range h {
		out = append(out, wire.TableHistoryEntry{Batch: i, Events: wireEvents(e.Events)})
	}
	return out
}
func wireEvents(evs []hview.Event) []wire.TableEvent {
	out := make([]wire.TableEvent, 0, len(evs))
	for _, e := range evs {
		pass := ""
		if e.Kind == "hands-passed" {
			pass = "left"
			if e.Age == 2 {
				pass = "right"
			}
		}
		out = append(out, wire.TableEvent{Kind: e.Kind, Seat: e.Seat, Other: e.Other, Age: e.Age, Turn: e.Turn, Play: e.Play, Design: e.Design, Stage: e.Stage, Side: e.Side, Left: e.Left, Right: e.Right, Printed: e.Printed, Delta: e.Delta, Source: e.Source, Outcome: e.Outcome, Pass: pass})
	}
	return out
}
func pointKind(p point) string { return kindName(p.Kind) }
func kindName(k rules.PointKind) string {
	if k >= rules.NumPointKinds {
		return ""
	}
	return [...]string{"none", "sides", "turn", "discard-build", "last-card"}[k]
}
