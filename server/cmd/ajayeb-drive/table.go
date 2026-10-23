package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

// tableOptions are the flags of ajayeb-drive table, the N-seat driver for
// simultaneous decision points (docs/testing.md, "The N-seat table
// harness").
type tableOptions struct {
	seats      []int
	seed       uint64
	games      int
	schedules  int
	exhaust    uint64
	seqClients int
	drafts     int
	resends    int
	race       bool
	hideAtEnd  bool
	leaks      bool
}

func parseTableFlags(args []string, stderr io.Writer) (tableOptions, error) {
	fs := flag.NewFlagSet("ajayeb-drive table", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: ajayeb-drive table [flags]")
		fmt.Fprintln(stderr, "Plays N-seat games of the fake hand game on an in-process table, each under")
		fmt.Fprintln(stderr, "several seeded arrival schedules, and requires one canonical history.")
		fs.PrintDefaults()
	}
	var o tableOptions
	seats := fs.String("seats", "3,4,5,6,7", "comma-separated seat counts, each 3 to 7")
	fs.Uint64Var(&o.seed, "seed", 1, "base seed; game g at n seats uses seed+1000*n+g")
	fs.IntVar(&o.games, "games", 2, "games per seat count")
	fs.IntVar(&o.schedules, "schedules", 8, "arrival schedules per game, compared for identical canonical batches")
	fs.Uint64Var(&o.exhaust, "exhaust", 0, "also replay the first point under every arrival order, refusing above this many orders (0: off)")
	fs.IntVar(&o.seqClients, "seq-clients", 2, "clients whose last seq the table remembers (small values make stale seqs)")
	fs.IntVar(&o.drafts, "drafts", 50, "percent of seats that send a draft before their final pick")
	fs.IntVar(&o.resends, "resends", 40, "percent of submissions re-sent later with the same seq")
	fs.BoolVar(&o.race, "race", false, "seats confirm, then change or withdraw, so arrival decides the batch")
	fs.BoolVar(&o.hideAtEnd, "hide-at-end", false, "play with the end reveal off")
	fs.BoolVar(&o.leaks, "leaks", true, "run the leak oracle over every view, history, snapshot and log line")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return o, err
		}
		return o, errBadFlags
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	for _, f := range strings.Split(*seats, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n < fakeengine.MinSeats || n > fakeengine.MaxSeats {
			return o, fmt.Errorf("-seats: %q is not a seat count from %d to %d", f, fakeengine.MinSeats, fakeengine.MaxSeats)
		}
		o.seats = append(o.seats, n)
	}
	switch {
	case o.games < 1:
		return o, errors.New("-games must be at least 1")
	case o.schedules < 1:
		return o, errors.New("-schedules must be at least 1")
	case o.seqClients < 1:
		return o, errors.New("-seq-clients must be at least 1")
	case o.drafts < 0 || o.drafts > 100 || o.resends < 0 || o.resends > 100:
		return o, errors.New("-drafts and -resends are percentages")
	}
	return o, nil
}

// runTable is ajayeb-drive table. It prints one summary line per game, then
// any failure, and exits 1 if a game failed.
func runTable(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	o, err := parseTableFlags(args, stderr)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errBadFlags):
		return 2
	case err != nil:
		fmt.Fprintf(stderr, "ajayeb-drive table: %v\n", err)
		return 2
	}
	failed := 0
	for _, n := range o.seats {
		for g := range o.games {
			line, err := tableGame(ctx, o, n, g)
			fmt.Fprintln(stdout, line)
			if err != nil {
				fmt.Fprintf(stderr, "ajayeb-drive table: %d seats, game %d: %v\n", n, g, err)
				failed++
			}
		}
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// tableGame plays one game under every schedule, optionally every arrival
// order of its first point, and the leak oracle.
func tableGame(ctx context.Context, o tableOptions, n, game int) (string, error) {
	seed := o.seed + uint64(1000*n+game)
	setup := fakeengine.Setup{Seed: seed, Seats: uint8(n), Options: fakeengine.Options{HideAtEnd: o.hideAtEnd}}
	summary := fmt.Sprintf("table seats=%d game=%d seed=%d", n, game, seed)

	// The in-process adapter. A WebSocket adapter plugs in here: a Factory
	// whose Table sends each submission as a frame and reads the reply.
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{ReplaceAttr: dropTime}))
	var current tabletest.Table
	var frames []leakcheck.Doc
	open := func(ctx context.Context, prefix []tabletest.Batch) (tabletest.Table, error) {
		t, err := direct.Factory(direct.Options{Setup: setup, SeqClients: o.seqClients, Logger: logger})(ctx, prefix)
		current = t
		return t, err
	}
	cfg := tabletest.Config{
		Seats: n, PickSeed: seed,
		Script: tabletest.ScriptOptions{Drafts: o.drafts, Resends: o.resends, Race: o.race},
	}
	if o.leaks {
		cfg.AfterPoint = func(ctx context.Context, idx int) error {
			for s := -1; s < n; s++ {
				c := tabletest.SpectatorClient
				if s >= 0 {
					c = tabletest.SeatClient(s)
				}
				sn, err := current.Snapshot(ctx, c)
				if err != nil {
					return err
				}
				data, err := json.Marshal(sn)
				if err != nil {
					return err
				}
				frames = append(frames, leakcheck.Doc{
					Label: fmt.Sprintf("%s's snapshot after point %d", c, idx), Kind: leakcheck.Frame,
					Viewer: s, Moment: idx + 1, Data: data,
				})
			}
			return nil
		}
	}

	arrivals := make([]uint64, o.schedules)
	for i := range arrivals {
		arrivals[i] = seed<<8 | uint64(i+1)
	}
	runs, err := tabletest.CompareSchedules(ctx, open, cfg, arrivals)
	if err != nil {
		return summary + " status=fail", err
	}
	c := runs[0].Canonical
	var outcomes [8]int
	names := []tabletest.Outcome{tabletest.Accepted, tabletest.Resolved, tabletest.Duplicate, tabletest.Stale,
		tabletest.Illegal, tabletest.Closed, tabletest.Forbidden, tabletest.StorageError}
	total := 0
	for _, r := range runs {
		for _, a := range r.Trace {
			for i, nm := range names {
				if a.Outcome == nm {
					outcomes[i]++
				}
			}
			total++
		}
	}
	summary += fmt.Sprintf(" points=%d schedules=%d submissions=%d", len(c.Batches), len(runs), total)
	for i, nm := range names {
		if outcomes[i] > 0 {
			summary += fmt.Sprintf(" %s=%d", nm, outcomes[i])
		}
	}
	summary += " hash=" + c.Hash

	if o.exhaust > 0 {
		ecfg := cfg
		ecfg.AfterPoint = nil
		count, distinct, err := tabletest.ExhaustPoint(ctx, open, nil, ecfg, o.exhaust)
		summary += fmt.Sprintf(" orders=%d batches=%d", count, len(distinct))
		if err != nil {
			return summary + " status=fail", err
		}
	}

	if o.leaks {
		leaks, docs, err := checkTableLeaks(setup, c.Batches, frames, logs.Bytes())
		summary += fmt.Sprintf(" docs=%d leaks=%d", docs, len(leaks))
		if err != nil {
			return summary + " status=fail", err
		}
		if len(leaks) > 0 {
			var b strings.Builder
			for i, l := range leaks {
				if i == 5 {
					fmt.Fprintf(&b, "\n  and %d more", len(leaks)-5)
					break
				}
				fmt.Fprintf(&b, "\n  %s", l)
			}
			return summary + " status=fail", fmt.Errorf("the leak oracle failed:%s", b.String())
		}
	}
	return summary + " status=ok", nil
}

// checkTableLeaks runs the leak oracle over every viewer's view and
// history at every moment of the canonical game, every snapshot taken, and
// the table's log.
func checkTableLeaks(setup fakeengine.Setup, batches []tabletest.Batch, frames []leakcheck.Doc, logs []byte) ([]leakcheck.Leak, int, error) {
	var bs []fakeengine.Batch
	for _, b := range batches {
		eb := fakeengine.Batch{Point: b.Point}
		for _, p := range b.Picks {
			eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: fakeengine.Seat(p.Seat), Key: p.Key})
		}
		bs = append(bs, eb)
	}
	truth, err := leakcheck.FromFake(setup, bs)
	if err != nil {
		return nil, 0, err
	}
	docs := append([]leakcheck.Doc{}, frames...)
	docs = append(docs, leakcheck.Doc{Label: "the table's log", Kind: leakcheck.Log, Data: logs})
	for k := 0; k <= len(bs); k++ {
		g, err := fakeengine.Replay(setup, bs[:k])
		if err != nil {
			return nil, 0, err
		}
		for s := -1; s < int(setup.Seats); s++ {
			v := fakeengine.SpectatorViewer()
			if s >= 0 {
				v = fakeengine.SeatViewer(fakeengine.Seat(s))
			}
			view, err := json.Marshal(g.ViewFor(v))
			if err != nil {
				return nil, 0, err
			}
			hist, err := json.Marshal(g.History(v))
			if err != nil {
				return nil, 0, err
			}
			docs = append(docs,
				leakcheck.Doc{Label: fmt.Sprintf("view of %d at moment %d", s, k), Kind: leakcheck.View, Viewer: s, Moment: k, Data: view},
				leakcheck.Doc{Label: fmt.Sprintf("history of %d at moment %d", s, k), Kind: leakcheck.History, Viewer: s, Moment: k, Data: hist})
		}
	}
	cfg := leakcheck.DefaultConfig
	cfg.KeyPaths = append(append([]string{}, cfg.KeyPaths...), "legal[]", "pick")
	return leakcheck.New(truth, cfg).Check(docs...), len(docs), nil
}

// dropTime removes the time from captured log lines, so a log is a
// function of the game alone.
func dropTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
