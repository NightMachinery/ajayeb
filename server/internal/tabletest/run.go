package tabletest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Config is one schedule of a whole game.
type Config struct {
	Seats       int
	PickSeed    uint64 // the keys each seat submits
	ArrivalSeed uint64 // the order the submissions arrive in
	Script      ScriptOptions
	// AfterPoint, if set, runs after each point resolves (for example the
	// leak oracle over every viewer's frames); index counts from 0.
	AfterPoint func(ctx context.Context, index int) error
}

// Arrival is one delivered submission and the table's answer.
type Arrival struct {
	Point   string  `json:"point"`
	Op      Op      `json:"op"`
	Outcome Outcome `json:"outcome"`
}

// Run is one schedule's result.
type Run struct {
	Trace []Arrival
	// Expected is the batches the trace implies: at each point every
	// required seat's last accepted key when the completing confirmation
	// arrived.
	Expected  []Batch
	Canonical Canonical
}

// TraceBytes encodes the trace, one JSON line per arrival.
func (r Run) TraceBytes() []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, a := range r.Trace {
		_ = enc.Encode(a)
	}
	return b.Bytes()
}

// maxGamePoints bounds a game, so a table that never ends fails.
const maxGamePoints = 1000

// Play drives one whole game: at each point it takes every seat's snapshot,
// checks they agree on one frozen point, builds the required seats'
// scripts, delivers them in the arrival order and checks every outcome.
// At the end it checks the table's canonical batches against the ones the
// trace implies and, without races, against every seat's final key.
func Play(ctx context.Context, t Table, cfg Config) (Run, error) {
	var run Run
	seqs := Seqs{}
	applied := map[ClientID]uint64{}
	clients := make([]ClientID, cfg.Seats)
	for s := range clients {
		clients[s] = SeatClient(s)
	}
	for idx := 0; ; idx++ {
		if idx > maxGamePoints {
			return run, errors.New("tabletest: the game did not end")
		}
		snaps, err := common(ctx, t, clients)
		if err != nil {
			return run, err
		}
		if snaps[0].Over {
			break
		}
		scripts, err := Scripts(snaps, clients, cfg.PickSeed, cfg.Script, seqs)
		if err != nil {
			return run, err
		}
		order := Interleave(scripts, mix(cfg.ArrivalSeed, uint64(idx)))
		arrivals, err := deliver(ctx, t, snaps[0].Point, order)
		if err != nil {
			return run, err
		}
		run.Trace = append(run.Trace, arrivals...)
		batch, err := CheckPoint(snaps, arrivals, applied)
		if err != nil {
			return run, fmt.Errorf("point %s (schedule %d/%d): %w", snaps[0].Point, cfg.PickSeed, cfg.ArrivalSeed, err)
		}
		if !cfg.Script.Race {
			if err := checkFinals(batch, scripts); err != nil {
				return run, fmt.Errorf("point %s: %w", snaps[0].Point, err)
			}
		}
		run.Expected = append(run.Expected, batch)
		if cfg.AfterPoint != nil {
			if err := cfg.AfterPoint(ctx, idx); err != nil {
				return run, err
			}
		}
	}
	canon, err := t.Canonical(ctx)
	if err != nil {
		return run, err
	}
	run.Canonical = canon
	if err := sameBatches(canon.Batches, run.Expected); err != nil {
		return run, fmt.Errorf("the table's canonical batches differ from its trace: %w", err)
	}
	return run, nil
}

// common takes every client's snapshot and checks that they show one frozen
// point: the same reference, logical id and required seats, an offer for
// each required seat and none for the others.
func common(ctx context.Context, t Table, clients []ClientID) ([]Snapshot, error) {
	snaps := make([]Snapshot, len(clients))
	for i, c := range clients {
		s, err := t.Snapshot(ctx, c)
		if err != nil {
			return nil, err
		}
		snaps[i] = s
	}
	first := snaps[0]
	for i, s := range snaps {
		if s.Ref != first.Ref || s.Point != first.Point || s.Over != first.Over || !slices.Equal(s.Required, first.Required) {
			return nil, fmt.Errorf("tabletest: no common snapshot: %s sees %+v, %s sees %+v", clients[0], first, clients[i], s)
		}
		if contains(s.Required, s.Seat) == (len(s.Legal) == 0) && !s.Over {
			return nil, fmt.Errorf("tabletest: %s (seat %d, required %v) is offered %d keys", clients[i], s.Seat, s.Required, len(s.Legal))
		}
	}
	return snaps, nil
}

func deliver(ctx context.Context, t Table, point string, order []Op) ([]Arrival, error) {
	out := make([]Arrival, 0, len(order))
	for _, op := range order {
		o, err := t.Submit(ctx, op.Client, op.Sub)
		if err != nil {
			return out, fmt.Errorf("tabletest: submitting %+v: %w", op, err)
		}
		out = append(out, Arrival{Point: point, Op: op, Outcome: o})
	}
	return out, nil
}

// CheckPoint checks one point's arrivals against the acceptance rules and
// returns the batch they imply. snaps is the common snapshot (one per seat
// client, in seat order); applied holds each client's highest applied seq
// across the game, and is updated. The rules, each a necessary condition
// any correct table meets:
//
//   - exactly one arrival resolves the point; it confirms, and at that
//     moment every required seat's last accepted submission confirmed;
//   - before it, an accepted submission names the open reference, an
//     offered key, and a seq no older than the client's newest accepted one
//     at this point; once every seat has confirmed, the point resolves;
//   - after it, every arrival is decision-closed or a duplicate;
//   - a duplicate's seq was applied before; a stale submission's client had
//     a newer accepted pick at this point; an illegal key was not offered to
//     a required seat; decision-closed before resolution needs another
//     reference.
func CheckPoint(snaps []Snapshot, arrivals []Arrival, applied map[ClientID]uint64) (Batch, error) {
	sn := snaps[0]
	legal := map[int][]string{}
	for _, s := range snaps {
		legal[s.Seat] = s.Legal
	}
	type slot struct {
		key     string
		confirm bool
	}
	slots := map[int]slot{}
	newest := map[ClientID]uint64{}
	allConfirmed := func() bool {
		for _, s := range sn.Required {
			if !slots[s].confirm {
				return false
			}
		}
		return true
	}
	resolved := false
	for i, a := range arrivals {
		op, sub := a.Op, a.Op.Sub
		where := fmt.Sprintf("arrival %d (%s seq %d %q confirm=%t): %s", i, op.Client, sub.Seq, sub.Key, sub.Confirm, a.Outcome)
		switch a.Outcome {
		case Accepted, Resolved:
			switch {
			case resolved:
				return Batch{}, errors.New(where + ": accepted after the point resolved")
			case sub.Ref != sn.Ref:
				return Batch{}, errors.New(where + ": accepted for another point reference")
			case !contains(sn.Required, op.Seat) || !slices.Contains(legal[op.Seat], sub.Key):
				return Batch{}, errors.New(where + ": accepted a key not offered to the seat")
			case sub.Seq < newest[op.Client]:
				return Batch{}, errors.New(where + ": an older seq changed the pick (want stale)")
			}
			slots[op.Seat] = slot{sub.Key, sub.Confirm}
			newest[op.Client] = sub.Seq
			applied[op.Client] = max(applied[op.Client], sub.Seq)
			if a.Outcome == Resolved {
				if !allConfirmed() {
					return Batch{}, errors.New(where + ": resolved while a required seat had not confirmed")
				}
				resolved = true
			} else if allConfirmed() {
				return Batch{}, errors.New(where + ": every required seat had confirmed but the point stayed open")
			}
		case Duplicate:
			if sub.Seq > applied[op.Client] {
				return Batch{}, errors.New(where + ": a duplicate of a seq never applied")
			}
		case Stale:
			if resolved || sub.Seq >= newest[op.Client] {
				return Batch{}, errors.New(where + ": stale without a newer pick from this client at this point")
			}
		case Illegal:
			if resolved || (contains(sn.Required, op.Seat) && slices.Contains(legal[op.Seat], sub.Key)) {
				return Batch{}, errors.New(where + ": an offered key refused as illegal")
			}
		case Closed:
			if !resolved && sub.Ref == sn.Ref {
				return Batch{}, errors.New(where + ": decision-closed while the point was open")
			}
		default:
			return Batch{}, errors.New(where + ": unexpected outcome")
		}
	}
	if !resolved {
		return Batch{}, errors.New("the point did not resolve")
	}
	b := Batch{Point: sn.Point}
	for _, s := range sn.Required {
		b.Picks = append(b.Picks, Pick{Seat: s, Key: slots[s].key})
	}
	return b, nil
}

// checkFinals checks that a batch holds every seat's scripted final key.
func checkFinals(b Batch, scripts []Script) error {
	for _, s := range scripts {
		for _, p := range b.Picks {
			if p.Seat == s.Seat && p.Key != s.Final {
				return fmt.Errorf("seat %d resolved with %q, its final key is %q", s.Seat, p.Key, s.Final)
			}
		}
	}
	return nil
}

func sameBatches(got, want []Batch) error {
	if len(got) != len(want) {
		return fmt.Errorf("%d batches, want %d", len(got), len(want))
	}
	for i := range got {
		if !got[i].Equal(want[i]) {
			return fmt.Errorf("batch %d is %+v, want %+v", i, got[i], want[i])
		}
	}
	return nil
}

// CompareSchedules plays the same game once per arrival seed, each on a
// fresh table, and without races requires byte-identical canonical batches
// and hashes. It returns the runs.
func CompareSchedules(ctx context.Context, open Factory, cfg Config, arrivalSeeds []uint64) ([]Run, error) {
	var runs []Run
	for _, seed := range arrivalSeeds {
		t, err := open(ctx, nil)
		if err != nil {
			return runs, err
		}
		c := cfg
		c.ArrivalSeed = seed
		r, err := Play(ctx, t, c)
		if err != nil {
			return runs, err
		}
		if len(runs) > 0 && !cfg.Script.Race && !bytes.Equal(r.Canonical.Bytes(), runs[0].Canonical.Bytes()) {
			return runs, fmt.Errorf("arrival seeds %d and %d gave different canonical histories:\n%s\n%s",
				arrivalSeeds[0], seed, runs[0].Canonical.Bytes(), r.Canonical.Bytes())
		}
		runs = append(runs, r)
	}
	return runs, nil
}

// ExhaustPoint opens a table at the given prefix, builds the open point's
// scripts from its common snapshot, and replays the point on a fresh table
// for every arrival order that keeps each client's own order, checking each
// one. Without races every order must give the same batch. It refuses when
// there are more than limit orders, and returns how many it ran and the
// distinct batches seen.
func ExhaustPoint(ctx context.Context, open Factory, prefix []Batch, cfg Config, limit uint64) (int, []Batch, error) {
	clients := make([]ClientID, cfg.Seats)
	for s := range clients {
		clients[s] = SeatClient(s)
	}
	t, err := open(ctx, prefix)
	if err != nil {
		return 0, nil, err
	}
	snaps, err := common(ctx, t, clients)
	if err != nil {
		return 0, nil, err
	}
	if snaps[0].Over {
		return 0, nil, errors.New("tabletest: the prefix ends the game")
	}
	scripts, err := Scripts(snaps, clients, cfg.PickSeed, cfg.Script, Seqs{})
	if err != nil {
		return 0, nil, err
	}
	if n := CountInterleavings(scripts, limit); n > limit {
		return 0, nil, fmt.Errorf("tabletest: more than %d arrival orders", limit)
	}
	var distinct []Batch
	count := 0
	var failure error
	Interleavings(scripts, func(order []Op) bool {
		t, err := open(ctx, prefix)
		if err == nil {
			var arrivals []Arrival
			arrivals, err = deliver(ctx, t, snaps[0].Point, order)
			if err == nil {
				var b Batch
				b, err = CheckPoint(snaps, arrivals, map[ClientID]uint64{})
				if err == nil && !cfg.Script.Race {
					err = checkFinals(b, scripts)
				}
				if err == nil && !slices.ContainsFunc(distinct, b.Equal) {
					distinct = append(distinct, b)
				}
			}
		}
		if err != nil {
			failure = fmt.Errorf("arrival order %d: %w", count, err)
			return false
		}
		count++
		return true
	})
	return count, distinct, failure
}
