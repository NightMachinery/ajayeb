// Package direct is an in-process tabletest.Table over the fake engine: a
// small model of the Haftgane actor's decision-point rules
// (docs/haftgane/platform-architecture.md, "Decision points and
// acceptance" and "Persistence, replay and compatibility"), so the N-seat
// driver, the leak oracle and the failure fixtures run before the actor
// exists. It is not the actor: no lobby, roles, seat switches, undo, frames
// or presence beyond what the fixtures need.
//
// Acceptance checks run in the platform's order: membership; the client's
// last applied seq (a seq not newer is a harmless duplicate); the open point
// reference (else decision-closed); the seat being required, and the key
// being offered to it (else illegal); a seq older than the one that last
// changed the seat's pick from the same client (stale). Only an applied
// submission uses up its seq. The confirmation that completes the required
// set resolves the point at once.
//
// Every change is a transaction: build the candidate, write it durably
// through the Store, then swap it in. A failed write discards the candidate;
// a crash (an error wrapping a crash sentinel the Store chooses) kills the
// table, and the test restores a new one from the stored record.
package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

// DefaultSeqClients is Duel's bound on remembered client seqs.
const DefaultSeqClients = 32

// Store persists the table's record. Save writes atomically, or not at all.
type Store interface {
	Save(ctx context.Context, data []byte) error
}

// Permits is a fair work pool, as ai.Pool is for the actor.
type Permits interface {
	Acquire(ctx context.Context) error
	Release()
}

// Work stands for one bot search: it returns when the search is done, or
// with the context's error when cancelled.
type Work func(ctx context.Context, seat int, point string) error

// Options configure a table.
type Options struct {
	Setup fakeengine.Setup
	// SeqClients bounds the clients whose last applied seq is remembered
	// (default DefaultSeqClients); a small value makes stale seqs reachable.
	SeqClients int
	// Crashed reports whether a Store error means the process died (the
	// table then refuses everything until restored). Nil: never.
	Crashed func(error) bool
	Store   Store
	Logger  *slog.Logger
	// Bots are the seats a bot controls; RunBots picks for them.
	Bots    []int
	Permits Permits
	Work    Work
}

// Errors.
var (
	ErrDead          = errors.New("direct: the table crashed; restore it")
	ErrUnknownClient = errors.New("direct: unknown client")
	ErrNoHuman       = errors.New("direct: no human participant is attached; bots wait")
	ErrRestore       = errors.New("direct: invalid record")
)

// Table is the in-process table. It is not safe for concurrent use: like
// the actor, its caller serializes.
type Table struct {
	opts    Options
	v       value
	seqs    lru
	members map[tabletest.ClientID]int // client -> seat, -1 for a spectator
	present map[tabletest.ClientID]bool
	dead    bool
}

// value is everything a transaction replaces at once.
type value struct {
	game    fakeengine.Game
	batches []tabletest.Batch
	rev     uint64
	ref     string
	slots   [fakeengine.MaxSeats]slot
}

type slot struct {
	key     string
	confirm bool
	seqs    []ClientSeq // per client, the seq that last changed this slot
}

// ClientSeq is one client's seq.
type ClientSeq struct {
	Client tabletest.ClientID `json:"client"`
	Seq    uint64             `json:"seq"`
}

// New creates a table at the start of a game, with one present client per
// seat (tabletest.SeatClient) and a spectator, and writes its first record.
func New(ctx context.Context, opts Options) (*Table, error) {
	g, err := fakeengine.NewGame(opts.Setup)
	if err != nil {
		return nil, err
	}
	t := newTable(opts, value{game: g, batches: []tabletest.Batch{}})
	t.v.ref = refFor(&t.v.game, t.v.rev)
	if err := t.persist(ctx, t.v, t.seqs); err != nil {
		return nil, err
	}
	return t, nil
}

// Factory returns a tabletest.Factory: a fresh table with the prefix
// resolved through the engine, without persisting the prefix's steps.
func Factory(opts Options) tabletest.Factory {
	return func(ctx context.Context, prefix []tabletest.Batch) (tabletest.Table, error) {
		g, err := fakeengine.Replay(opts.Setup, engineBatches(prefix))
		if err != nil {
			return nil, err
		}
		t := newTable(opts, value{game: g, batches: slices.Clone(prefix), rev: uint64(len(prefix))})
		if t.v.batches == nil {
			t.v.batches = []tabletest.Batch{}
		}
		t.v.ref = refFor(&t.v.game, t.v.rev)
		if err := t.persist(ctx, t.v, t.seqs); err != nil {
			return nil, err
		}
		return t, nil
	}
}

func newTable(opts Options, v value) *Table {
	if opts.SeqClients <= 0 {
		opts.SeqClients = DefaultSeqClients
	}
	t := &Table{
		opts:    opts,
		v:       v,
		seqs:    lru{cap: opts.SeqClients},
		members: map[tabletest.ClientID]int{tabletest.SpectatorClient: -1},
		present: map[tabletest.ClientID]bool{tabletest.SpectatorClient: true},
	}
	for s := 0; s < int(opts.Setup.Seats); s++ {
		if !slices.Contains(opts.Bots, s) {
			t.members[tabletest.SeatClient(s)] = s
			t.present[tabletest.SeatClient(s)] = true
		}
	}
	return t
}

// Attach makes c a member for seat (-1 for a spectator) and present: a
// second tab, or a former controller watching.
func (t *Table) Attach(c tabletest.ClientID, seat int) {
	t.members[c] = seat
	t.present[c] = true
}

// Detach marks c absent; it stays a member.
func (t *Table) Detach(c tabletest.ClientID) { t.present[c] = false }

// Game returns the current engine value, for oracles.
func (t *Table) Game() fakeengine.Game { return t.v.game }

// refFor derives the wire point reference from the logical id and the
// durable revision at opening.
func refFor(g *fakeengine.Game, rev uint64) string {
	return pointName(g) + "@" + strconv.FormatUint(rev, 10)
}

// pointName is the open point's logical id, or "over".
func pointName(g *fakeengine.Game) string {
	if id := g.Point().ID; id != "" {
		return id
	}
	return "over"
}

// Snapshot returns what client c sees of the open point.
func (t *Table) Snapshot(_ context.Context, c tabletest.ClientID) (tabletest.Snapshot, error) {
	if t.dead {
		return tabletest.Snapshot{}, ErrDead
	}
	seat, ok := t.members[c]
	if !ok {
		return tabletest.Snapshot{}, ErrUnknownClient
	}
	p := t.v.game.Point()
	sn := tabletest.Snapshot{
		Ref: t.v.ref, Point: p.ID, Required: []int{}, Confirmed: []int{}, Seat: seat, Legal: []string{},
		Over: p.Kind == fakeengine.PointNone,
	}
	for _, s := range p.Required.Seats() {
		sn.Required = append(sn.Required, int(s))
		if t.v.slots[s].confirm {
			sn.Confirmed = append(sn.Confirmed, int(s))
		}
	}
	if seat >= 0 && p.Required.Has(fakeengine.Seat(seat)) {
		for _, ch := range t.v.game.Legal(fakeengine.Seat(seat)) {
			sn.Legal = append(sn.Legal, ch.Key)
		}
		sn.Pick, sn.Confirm = t.v.slots[seat].key, t.v.slots[seat].confirm
	}
	return sn, nil
}

// Submit applies one submission under the platform's acceptance rules.
func (t *Table) Submit(ctx context.Context, c tabletest.ClientID, s tabletest.Submission) (tabletest.Outcome, error) {
	if t.dead {
		return "", ErrDead
	}
	seat, ok := t.members[c]
	if !ok || seat < 0 {
		return t.answer("pick", seat, tabletest.Forbidden), nil
	}
	if last, ok := t.seqs.get(c); ok && s.Seq <= last {
		return t.answer("pick", seat, tabletest.Duplicate), nil
	}
	p := t.v.game.Point()
	if p.Kind == fakeengine.PointNone || s.Ref != t.v.ref {
		return t.answer("pick", seat, tabletest.Closed), nil
	}
	if !p.Required.Has(fakeengine.Seat(seat)) || !t.offered(seat, s.Key) {
		return t.answer("pick", seat, tabletest.Illegal), nil
	}
	if s.Seq < seqOf(t.v.slots[seat].seqs, c) {
		return t.answer("pick", seat, tabletest.Stale), nil
	}
	seqs := t.seqs.clone()
	seqs.put(c, s.Seq)
	return t.transact(ctx, "pick", seat, s.Key, s.Confirm, &ClientSeq{Client: c, Seq: s.Seq}, seqs)
}

func (t *Table) offered(seat int, key string) bool {
	for _, ch := range t.v.game.Legal(fakeengine.Seat(seat)) {
		if ch.Key == key {
			return true
		}
	}
	return false
}

// transact builds the candidate for one accepted pick, resolves the point
// when this confirmation completes the required set, persists the
// candidate, then swaps it in.
func (t *Table) transact(ctx context.Context, op string, seat int, key string, confirm bool, by *ClientSeq, seqs lru) (tabletest.Outcome, error) {
	cand := t.v
	cand.slots[seat] = slot{key: key, confirm: confirm, seqs: slices.Clone(t.v.slots[seat].seqs)}
	if by != nil {
		cand.slots[seat].seqs = setSeq(cand.slots[seat].seqs, *by)
	}
	cand.rev++
	outcome := tabletest.Accepted
	p := cand.game.Point()
	complete := true
	for _, s := range p.Required.Seats() {
		complete = complete && cand.slots[s].confirm
	}
	if complete {
		b := tabletest.Batch{Point: p.ID, Picks: []tabletest.Pick{}}
		for _, s := range p.Required.Seats() {
			b.Picks = append(b.Picks, tabletest.Pick{Seat: int(s), Key: cand.slots[s].key})
		}
		next, _, err := cand.game.Resolve(engineBatches([]tabletest.Batch{b})[0])
		if err != nil {
			return "", err // a key was legal at acceptance, so this is a bug
		}
		cand.game = next
		cand.batches = append(slices.Clone(t.v.batches), b)
		cand.slots = [fakeengine.MaxSeats]slot{}
		cand.ref = refFor(&cand.game, cand.rev)
		outcome = tabletest.Resolved
	}
	if err := t.persist(ctx, cand, seqs); err != nil {
		if t.opts.Crashed != nil && t.opts.Crashed(err) {
			t.dead = true
			return "", err
		}
		return t.answer(op, seat, tabletest.StorageError), nil
	}
	t.v, t.seqs = cand, seqs
	return t.answer(op, seat, outcome), nil
}

// answer logs a refusal or an outcome with safe fields only: the operation,
// the seat, the revision and the outcome code, never a key or a card.
func (t *Table) answer(op string, seat int, o tabletest.Outcome) tabletest.Outcome {
	if t.opts.Logger != nil {
		t.opts.Logger.Info("table", "op", op, "seat", seat, "rev", t.v.rev, "outcome", string(o))
	}
	return o
}

// Canonical returns the complete batches and the rules hash.
func (t *Table) Canonical(context.Context) (tabletest.Canonical, error) {
	if t.dead {
		return tabletest.Canonical{}, ErrDead
	}
	return tabletest.Canonical{Batches: slices.Clone(t.v.batches), Hash: t.v.game.Fingerprint()}, nil
}

// humanAttached reports whether a present client holds a seat. Spectators
// cannot keep bot work alive.
func (t *Table) humanAttached() bool {
	for c, seat := range t.members {
		if seat >= 0 && t.present[c] {
			return true
		}
	}
	return false
}

// RunBots lets each bot seat that the open point requires and that has not
// confirmed pick, one at a time: acquire a permit, do the work, release,
// then submit the first offered key, confirmed. It stops when the point
// resolves. With no human participant present it does nothing and returns
// ErrNoHuman; a cancelled context stops it before the next submission. It
// returns how many bots picked.
func (t *Table) RunBots(ctx context.Context) (int, error) {
	if t.dead {
		return 0, ErrDead
	}
	n := 0
	p := t.v.game.Point()
	for _, s := range p.Required.Seats() {
		seat := int(s)
		if !slices.Contains(t.opts.Bots, seat) || t.v.slots[seat].confirm {
			continue
		}
		if !t.humanAttached() {
			return n, ErrNoHuman
		}
		if t.opts.Permits != nil {
			if err := t.opts.Permits.Acquire(ctx); err != nil {
				return n, err
			}
		}
		var err error
		if t.opts.Work != nil {
			err = t.opts.Work(ctx, seat, p.ID)
		}
		if t.opts.Permits != nil {
			t.opts.Permits.Release()
		}
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			return n, err
		}
		if !t.humanAttached() {
			return n, ErrNoHuman
		}
		o, err := t.transact(ctx, "bot-pick", seat, t.v.game.Legal(s)[0].Key, true, nil, t.seqs.clone())
		if err != nil {
			return n, err
		}
		if o == tabletest.StorageError {
			return n, errors.New("direct: storage error on a bot pick")
		}
		n++
		if o == tabletest.Resolved {
			break
		}
	}
	return n, nil
}

// --- the record ----------------------------------------------------------

// Record is the table's durable state: the setup, the canonical batches and
// rules hash, the revision and open point reference, the private pending
// block and the remembered seqs. It is the harness's own format; the
// actor's schema-2 codec is its own.
type Record struct {
	Seed      uint64            `json:"seed"`
	Seats     uint8             `json:"seats"`
	Seating   uint8             `json:"seating"`
	HideAtEnd bool              `json:"hideAtEnd"`
	Batches   []tabletest.Batch `json:"batches"`
	Hash      string            `json:"hash"`
	Revision  uint64            `json:"revision"`
	Ref       string            `json:"ref"`
	Pending   []PendingSlot     `json:"pending"`
	Seqs      []ClientSeq       `json:"seqs"`
}

// PendingSlot is one required seat's unresolved pick.
type PendingSlot struct {
	Seat      int         `json:"seat"`
	Key       string      `json:"key"`
	Confirmed bool        `json:"confirmed"`
	Seqs      []ClientSeq `json:"seqs"`
}

// Record returns the table's current record.
func (t *Table) Record() Record { return t.record(t.v, t.seqs) }

func (t *Table) record(v value, seqs lru) Record {
	st := v.game.State
	r := Record{
		Seed: st.Seed, Seats: st.Seats, Seating: uint8(st.Opts.Seating), HideAtEnd: st.Opts.HideAtEnd,
		Batches: slices.Clone(v.batches), Hash: v.game.Fingerprint(), Revision: v.rev, Ref: v.ref,
		Pending: []PendingSlot{}, Seqs: slices.Clone(seqs.entries),
	}
	if r.Seqs == nil {
		r.Seqs = []ClientSeq{}
	}
	for s := range v.slots {
		if v.slots[s].key != "" {
			r.Pending = append(r.Pending, PendingSlot{
				Seat: s, Key: v.slots[s].key, Confirmed: v.slots[s].confirm, Seqs: append([]ClientSeq{}, v.slots[s].seqs...),
			})
		}
	}
	return r
}

func (t *Table) persist(ctx context.Context, v value, seqs lru) error {
	if t.opts.Store == nil {
		return nil
	}
	data, err := json.Marshal(t.record(v, seqs))
	if err != nil {
		return err
	}
	return t.opts.Store.Save(ctx, data)
}

// Restore rebuilds a table from a stored record, validating it: the setup,
// every batch by replay, the rules hash, the open point reference, and the
// pending block (each slot a required seat at most once, with an offered
// key, and not every required seat confirmed). Options.Setup is ignored:
// the record holds the setup.
func Restore(opts Options, data []byte) (*Table, error) {
	var r Record
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, errors.Join(ErrRestore, err)
	}
	return RestoreRecord(opts, r)
}

// RestoreRecord is Restore from a decoded record.
func RestoreRecord(opts Options, r Record) (*Table, error) {
	bad := func(msg string) error { return errors.Join(ErrRestore, errors.New(msg)) }
	opts.Setup = fakeengine.Setup{Seed: r.Seed, Seats: r.Seats, Options: fakeengine.Options{
		Seating: fakeengine.SeatingRule(r.Seating), HideAtEnd: r.HideAtEnd,
	}}
	g, err := fakeengine.Replay(opts.Setup, engineBatches(r.Batches))
	if err != nil {
		return nil, errors.Join(ErrRestore, err)
	}
	if g.Fingerprint() != r.Hash {
		return nil, bad("the rules hash does not match the replay")
	}
	if err := g.CheckInvariants(); err != nil {
		return nil, errors.Join(ErrRestore, err)
	}
	p := g.Point()
	id, rev, ok := strings.Cut(r.Ref, "@")
	if !ok || id != pointName(&g) || rev == "" {
		return nil, bad("the point reference " + r.Ref + " is not the open point")
	}
	if n, err := strconv.ParseUint(rev, 10, 64); err != nil || n > r.Revision {
		return nil, bad("the point reference " + r.Ref + " is newer than the revision")
	}
	v := value{game: g, batches: r.Batches, rev: r.Revision, ref: r.Ref}
	if v.batches == nil {
		v.batches = []tabletest.Batch{}
	}
	confirmed := 0
	for _, ps := range r.Pending {
		switch {
		case ps.Seat < 0 || ps.Seat >= fakeengine.MaxSeats || !p.Required.Has(fakeengine.Seat(ps.Seat)):
			return nil, bad("a pending pick for a seat the point does not require")
		case v.slots[ps.Seat].key != "":
			return nil, bad("two pending picks for one seat")
		case ps.Key == "":
			return nil, bad("a pending slot without a key")
		}
		ok := false
		for _, ch := range g.Legal(fakeengine.Seat(ps.Seat)) {
			ok = ok || ch.Key == ps.Key
		}
		if !ok {
			return nil, bad("a pending key the seat is not offered")
		}
		v.slots[ps.Seat] = slot{key: ps.Key, confirm: ps.Confirmed, seqs: slices.Clone(ps.Seqs)}
		if ps.Confirmed {
			confirmed++
		}
	}
	if p.Kind != fakeengine.PointNone && confirmed == p.Required.Count() {
		return nil, bad("every required seat has confirmed, so the point should have resolved")
	}
	t := newTable(opts, v)
	for _, cs := range r.Seqs {
		t.seqs.put(cs.Client, cs.Seq)
	}
	return t, nil
}

func engineBatches(bs []tabletest.Batch) []fakeengine.Batch {
	out := make([]fakeengine.Batch, 0, len(bs))
	for _, b := range bs {
		eb := fakeengine.Batch{Point: b.Point}
		for _, p := range b.Picks {
			eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: fakeengine.Seat(p.Seat), Key: p.Key})
		}
		out = append(out, eb)
	}
	return out
}

func seqOf(seqs []ClientSeq, c tabletest.ClientID) uint64 {
	for _, cs := range seqs {
		if cs.Client == c {
			return cs.Seq
		}
	}
	return 0
}

func setSeq(seqs []ClientSeq, by ClientSeq) []ClientSeq {
	for i := range seqs {
		if seqs[i].Client == by.Client {
			seqs[i].Seq = by.Seq
			return seqs
		}
	}
	return append(seqs, by)
}

// lru remembers the last applied seq of at most cap clients, the oldest
// applied first. Unlike Duel's seqLRU, which lives only in memory, a lookup
// is not a use: the record holds this list, so a duplicate must change
// nothing, and a table restored from its record must forget the same
// clients as one that never stopped.
type lru struct {
	cap     int
	entries []ClientSeq
}

func (l *lru) get(c tabletest.ClientID) (uint64, bool) {
	for _, e := range l.entries {
		if e.Client == c {
			return e.Seq, true
		}
	}
	return 0, false
}

func (l *lru) put(c tabletest.ClientID, seq uint64) {
	for i, e := range l.entries {
		if e.Client == c {
			l.entries = append(l.entries[:i:i], l.entries[i+1:]...)
			break
		}
	}
	l.entries = append(l.entries, ClientSeq{Client: c, Seq: seq})
	if len(l.entries) > l.cap {
		l.entries = l.entries[len(l.entries)-l.cap:]
	}
}

func (l lru) clone() lru {
	return lru{cap: l.cap, entries: slices.Clone(l.entries)}
}
