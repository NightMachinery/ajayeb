package tableroom

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

// tabClient is seat 0's second tab: the same user on another client.
const tabClient tabletest.ClientID = "seat-0-tab"

// boundaryKinds are where a fuzzed table opens: the start, or the first
// salvage or last-card point of a game.
var boundaryKinds = []fakeengine.PointKind{fakeengine.PointNone, fakeengine.PointSalvage, fakeengine.PointLastCard}

type boundaryKey struct {
	seats int
	kind  fakeengine.PointKind
	from  uint64
}

type boundary struct {
	seed   uint64
	prefix []BatchRecord
	ok     bool
}

var boundaries sync.Map // boundaryKey -> boundary

// boundaryPrefix finds, from seed from on, a game (chosen seating) that
// reaches a point of the kind, and returns its seed and the batches
// before that point.
func boundaryPrefix(seats int, kind fakeengine.PointKind, from uint64) boundary {
	k := boundaryKey{seats, kind, from}
	if b, ok := boundaries.Load(k); ok {
		return b.(boundary)
	}
	b := findBoundary(seats, kind, from)
	boundaries.Store(k, b)
	return b
}

func findBoundary(seats int, kind fakeengine.PointKind, from uint64) boundary {
	for seed := from; seed < from+64; seed++ {
		g, err := newGame(makeSetup(seed, seats, false, false))
		if err != nil {
			return boundary{}
		}
		rng := rand.New(rand.NewPCG(seed, 3))
		var prefix []BatchRecord
		for !pointOver(g.Point()) {
			pt := g.Point()
			if pt.Kind == kind {
				return boundary{seed: seed, prefix: prefix, ok: true}
			}
			b, br := batch{Point: pt.ID}, BatchRecord{Point: pt.ID}
			for _, s := range pt.Required.Seats() {
				keys := legalKeys(&g, int(s))
				key := keys[rng.IntN(len(keys))]
				b.Picks = append(b.Picks, pick{Seat: s, Key: key})
				br.Picks = append(br.Picks, PickRecord{Seat: int(s), Key: key})
			}
			if g, _, err = g.Resolve(b); err != nil {
				return boundary{}
			}
			prefix = append(prefix, br)
		}
	}
	return boundary{}
}

// TestAbilityBoundariesReachable: the fuzzer's salvage and last-card
// openings exist at every seat count, so its corpus does reach them.
func TestAbilityBoundariesReachable(t *testing.T) {
	for n := MinSeats; n <= MaxSeats; n++ {
		for _, kind := range boundaryKinds[1:] {
			if b := boundaryPrefix(n, kind, 0); !b.ok {
				t.Errorf("%d seats: no game reaches a %s point", n, kind)
			}
		}
	}
}

// blockBots is a bot search that never finishes on its own: the fuzzer
// decides when a bot acts, so every run is deterministic.
func blockBots(ctx context.Context, _ int, _ string) error {
	<-ctx.Done()
	return ctx.Err()
}

// FuzzTableOps drives one table through fuzzed operations: picks, changes
// and confirmations from every seat, a second tab and the spectator, new
// tabs that open, pick and close (each a new client id, so the seq memory
// and the picks' floors meet many clients of one user), resends of
// earlier submissions (a closed tab reconnecting under its id to resend),
// stale references, undo, bot picks, detaches and restarts, opening at the
// start or at a salvage or last-card point, with small seq and floor
// bounds. After every operation the table in memory must equal the stored
// record byte for byte, the stored record must restore with a matching
// rules hash, no frame revision may go backwards, and a restart must keep
// every client's pick, confirmation and reference. A refusal or a
// duplicate writes nothing.
func FuzzTableOps(f *testing.F) {
	f.Add(uint64(1), byte(0), byte(0), []byte{0, 1, 0, 3, 6, 0, 1, 0, 2, 0, 4, 0, 0, 5, 7, 9, 6, 0, 5, 2, 1, 3})
	f.Add(uint64(2), byte(0x11), byte(2), []byte{0, 3, 7, 2, 6, 0, 0, 9, 4, 0, 1, 1, 6, 0, 2, 4})
	f.Add(uint64(3), byte(0x24), byte(4), []byte{0, 1, 4, 0, 0, 3, 6, 0, 6, 0})
	f.Add(uint64(4), byte(0x30), byte(1), []byte{3, 0, 7, 1, 6, 0, 3, 1, 3, 0, 1, 0, 6, 0, 6, 0, 3, 0, 4, 0, 3, 0, 5, 0})
	f.Add(uint64(5), byte(0x02), byte(3), []byte{0, 1, 7, 0, 7, 3, 4, 0, 3, 0, 6, 0, 3, 0})
	f.Add(uint64(6), byte(0x13), byte(5), []byte{7, 0, 0, 1, 4, 0, 7, 2, 6, 0, 3, 0, 4, 0})
	finish := []byte{}
	for i := range 60 {
		finish = append(finish, 6, byte(i))
	}
	f.Add(uint64(7), byte(0x01), byte(1), finish)
	f.Add(uint64(8), byte(0x14), byte(0), finish)
	f.Add(uint64(9), byte(0x40), byte(0), []byte{0, 1, 8, 0, 8, 2, 8, 4, 1, 3, 8, 6, 1, 4, 6, 0, 8, 1, 1, 9, 4, 0, 8, 8, 1, 2})
	f.Add(uint64(10), byte(0x01), byte(0), []byte{8, 0, 8, 8, 8, 16, 8, 24, 8, 32, 1, 1, 1, 2, 8, 40, 1, 4, 6, 0})
	f.Fuzz(func(t *testing.T, seed uint64, shape, mode byte, ops []byte) {
		if len(ops) > 200 {
			ops = ops[:200]
		}
		fuzzTable(t, seed, shape, mode, ops)
	})
}

type sentSub struct {
	id  tabletest.ClientID
	sub tabletest.Submission
}

func fuzzTable(t *testing.T, seed uint64, shape, mode byte, ops []byte) {
	n := MinSeats + int(shape&0x0f)%(MaxSeats-MinSeats+1)
	solo := mode&1 == 1
	kind := boundaryKinds[int(mode>>1)%len(boundaryKinds)]
	cfg := Config{SeqClients: 1 + int(shape>>4)%4, PickClients: 1 + int(shape>>6)%4}
	seed %= 64
	sp := humans(n, seed)
	if solo {
		sp = soloSpec(n, seed, BotEasy)
		cfg.BotWork = blockBots
	}
	if kind != fakeengine.PointNone {
		if b := boundaryPrefix(n, kind, seed); b.ok {
			sp.Seed, sp.Prefix = &b.seed, b.prefix
		}
	}
	ctx := testContext(t)
	fx := newFixtureOn(t, cfg, true)
	p, err := fx.openInproc(ctx, sp)
	if err != nil {
		t.Fatal(err)
	}
	h := &th{t: t, ctx: ctx, f: fx, p: p}
	me := p.clients[tabletest.SeatClient(0)]
	if err := p.join(ctx, tabClient, me.UserID, me.Name); err != nil {
		t.Fatal(err)
	}

	seqs := map[tabletest.ClientID]uint64{}
	next := func(id tabletest.ClientID) uint64 { seqs[id]++; return seqs[id] }
	var sent []sentSub
	refs := []string{"none@0"}
	closed := map[tabletest.ClientID]*room.Client{} // tabs that opened and closed
	submit := func(id tabletest.ClientID, s tabletest.Submission) {
		t.Helper()
		before, _, _ := fx.ms.PeekGame(p.code, Game)
		o, err := p.Submit(ctx, id, s)
		if err != nil {
			t.Fatal(err)
		}
		if o == tabletest.StorageError || strings.HasPrefix(string(o), "unknown") {
			t.Fatalf("%s %+v: %s", id, s, o)
		}
		if after, _, _ := fx.ms.PeekGame(p.code, Game); o != tabletest.Accepted && o != tabletest.Resolved && !bytes.Equal(before, after) {
			t.Fatalf("%s %+v: %s changed the record", id, s, o)
		}
		sent = append(sent, sentSub{id, s})
	}
	// tab opens id for user, runs fn and closes it again.
	tab := func(id tabletest.ClientID, user, name string, fn func()) {
		t.Helper()
		if err := p.join(ctx, id, user, name); err != nil {
			t.Fatal(err)
		}
		fn()
		closed[id] = p.clients[id]
		fx.m.Leave(p.clients[id], p.code)
		delete(p.clients, id)
	}
	tabs := 0
	pickFrom := func(id tabletest.ClientID, arg int, ref string) {
		sn := h.snap(id)
		refs = append(refs, sn.Ref)
		if ref == "" {
			ref = sn.Ref
		}
		key := "build:amber:0:0" // unoffered to this client, or offered by chance
		if len(sn.Legal) > 0 {
			key = sn.Legal[(arg/2)%len(sn.Legal)]
		}
		submit(id, tabletest.Submission{Seq: next(id), Ref: ref, Key: key, Confirm: arg&1 == 1})
	}
	botAct := func(choice int) bool {
		a, err := p.actor()
		if err != nil {
			t.Fatal(err)
		}
		acted := false
		if err := a.call(ctx, func() {
			if !a.botBusy {
				return
			}
			r, g := &a.st.rec, &a.st.game
			pt := g.Point()
			for _, s := range pt.Required.Seats() {
				if ps := pendingOf(r, int(s)); r.Seats[s].Kind == SeatBot && (ps == nil || !ps.Confirmed) {
					legal := legalKeys(g, int(s))
					a.botResult(msgBotResult{gen: a.botGen, point: pt.ID, seat: int(s), key: legal[choice%len(legal)]})
					acted = true
					return
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		p.drain()
		return acted
	}
	seen := map[tabletest.ClientID]int{}
	check := func(step int) {
		t.Helper()
		a, err := p.actor()
		if err != nil {
			t.Fatal(err)
		}
		var mem []byte
		var fp string
		var encErr error
		if err := a.call(ctx, func() { mem, encErr = a.st.rec.encode(); fp = a.st.game.Fingerprint() }); err != nil {
			t.Fatal(err)
		}
		disk, _, perr := fx.ms.PeekGame(p.code, Game)
		if encErr != nil || perr != nil || !bytes.Equal(mem, disk) {
			t.Fatalf("step %d: the table in memory differs from storage (%v, %v)", step, encErr, perr)
		}
		r, err := decodeRecord(disk)
		if err != nil {
			t.Fatal(err)
		}
		st, hashOK, err := restore(r, fx.tables.bounds())
		if err != nil || !hashOK || st.game.Fingerprint() != fp || r.Hash != fp {
			t.Fatalf("step %d: the stored record does not restore: %v, hash ok %t", step, err, hashOK)
		}
		if r.Status == store.StatusFinished && (len(r.Pending) != 0 || len(r.Work) != 0 || r.Point != nil) {
			t.Fatalf("step %d: a finished table keeps pending %v, work %v", step, r.Pending, r.Work)
		}
		p.drain()
		for id, frames := range p.frames {
			for i := max(seen[id], 1); i < len(frames); i++ {
				if frames[i].Revision < frames[i-1].Revision {
					t.Fatalf("step %d: %s's revision went from %d to %d", step, id, frames[i-1].Revision, frames[i].Revision)
				}
			}
			seen[id] = len(frames)
		}
	}
	check(-1)

	for i := 0; i+1 < len(ops); i += 2 {
		op, arg := ops[i]%9, int(ops[i+1])
		ids := slices.Sorted(maps.Keys(p.clients))
		switch op {
		case 0: // a pick, change or confirmation from any client
			pickFrom(ids[arg%len(ids)], arg>>3, "")
		case 1: // an earlier submission again, a closed tab reconnecting to send it
			if len(sent) > 0 {
				s := sent[arg%len(sent)]
				if c := closed[s.id]; c != nil {
					tab(s.id, c.UserID, c.Name, func() { submit(s.id, s.sub) })
				} else {
					submit(s.id, s.sub)
				}
			}
		case 2: // a fresh seq for an old reference
			pickFrom(ids[arg%len(ids)], arg>>3, refs[(arg>>2)%len(refs)])
		case 3: // undo from seat 0, on its latest revision or a stale one
			base := h.latest(tabletest.SeatClient(0)).Revision - arg&1
			switch out := h.undo(int(next(tabletest.SeatClient(0))), base); out {
			case outAccepted, outDuplicate, outStale, outIllegal, outForbidden:
			default:
				t.Fatalf("undo: %s", out)
			}
		case 4: // a restart keeps every client's pick, confirmation and reference
			before := map[tabletest.ClientID]tabletest.Snapshot{}
			for _, id := range ids {
				before[id] = h.snap(id)
			}
			fx.restart()
			h.rejoin()
			for _, id := range ids {
				if got := h.snap(id); got.Ref != before[id].Ref || got.Pick != before[id].Pick || got.Confirm != before[id].Confirm {
					t.Fatalf("%s across a restart: %+v, then %+v", id, before[id], got)
				}
			}
		case 5: // one client detaches and attaches again
			id := ids[arg%len(ids)]
			c := p.clients[id]
			fx.m.Leave(c, p.code)
			if err := p.join(ctx, id, c.UserID, c.Name); err != nil {
				t.Fatal(err)
			}
		case 6: // every required seat confirms, which resolves the point
			sn := h.snap(tabletest.SeatClient(0))
			for _, s := range sn.Required {
				if id := tabletest.SeatClient(s); p.clients[id] != nil {
					pickFrom(id, arg<<1|1, "")
				}
			}
			for k := 0; solo && k < MaxSeats && botAct(arg+k); k++ {
			}
		case 7: // seat 0's second tab, or a bot, acts
			if solo {
				botAct(arg)
			} else {
				pickFrom(tabClient, arg, "")
			}
		case 8: // a new tab of a seat's user opens, picks and closes
			if c := p.clients[tabletest.SeatClient(arg%n)]; c != nil {
				tabs++
				id := tabletest.ClientID(fmt.Sprintf("seat-%d-new-%d", arg%n, tabs))
				tab(id, c.UserID, c.Name, func() { pickFrom(id, arg>>3, "") })
			}
		}
		check(i / 2)
	}
}
