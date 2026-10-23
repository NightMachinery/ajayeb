package tableroom

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// state is a committed value: the record and the engine value its batches
// replay to. A transaction builds a new state and swaps it in whole.
type state struct {
	rec  Record
	game game
}

// actor runs one table. Everything below is touched only on its goroutine.
type actor struct {
	t     *Tables
	m     *room.Manager
	h     *room.Handle
	code  string
	st    *state
	conns []*room.Client

	inbox  chan any
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	idleStop   func() bool
	emptySince time.Time
	shownIdle  bool
	shownSince time.Time

	botGen    uint64
	botBusy   bool
	botCancel context.CancelFunc
	botWG     sync.WaitGroup
}

type msgAttach struct {
	c     *room.Client
	claim *string
	reply chan error
}

type msgDetach struct{ c *room.Client }

type msgEvict struct{ reply chan bool }

type msgIdle struct{}

// msgCall runs fn on the actor goroutine (tests and diagnostics).
type msgCall struct {
	fn    func()
	reply chan struct{}
}

type msgBotResult struct {
	gen   uint64
	point string
	seat  int
	key   string
	err   error
}

func (t *Tables) newActor(m *room.Manager, st *state) (*room.Handle, error) {
	a := &actor{t: t, m: m, code: st.rec.Code, st: st, inbox: make(chan any, t.cfg.InboxSize), done: make(chan struct{})}
	a.ctx, a.cancel = context.WithCancel(m.Context())
	h, err := m.NewHandle(Game, a.code, room.HandleConfig{
		Run: a.run, Cancel: a.cancel, Attach: a.attachHook, Detach: a.detachHook, Deliver: a.post, Evict: a.evictHook,
	})
	if err != nil {
		a.cancel()
		return nil, err
	}
	a.h = h
	t.register(a)
	return h, nil
}

func (a *actor) run() {
	defer a.t.unregister(a)
	defer close(a.done)
	defer a.botWG.Wait()
	defer a.stopBots()
	a.emptySince = a.now()
	a.armIdle()
	a.publishIdle()
	for {
		select {
		case <-a.ctx.Done():
			a.disarmIdle()
			return
		case msg := <-a.inbox:
			if a.handle(msg) {
				a.disarmIdle()
				return
			}
		}
	}
}

// handle processes one message; true means the actor exits.
func (a *actor) handle(msg any) bool {
	switch msg := msg.(type) {
	case msgAttach:
		msg.reply <- a.attach(msg.c, msg.claim)
	case msgDetach:
		a.detach(msg.c)
	case room.Inbound:
		a.inbound(msg.Client, msg.Frame)
	case msgBotResult:
		a.botResult(msg)
	case msgCall:
		msg.fn()
		close(msg.reply)
	case msgIdle:
		if len(a.conns) == 0 && a.now().Sub(a.emptySince) >= a.t.cfg.IdleTimeout {
			a.m.Forget(a.h)
			return true
		}
		a.idleStop = nil
		a.armIdle()
	case msgEvict:
		if len(a.conns) == 0 {
			a.m.Forget(a.h)
			msg.reply <- true
			return true
		}
		msg.reply <- false
	}
	return false
}

func (a *actor) now() time.Time { return a.t.cfg.Clock.Now().UTC() }

// post delivers msg to the inbox; Manager's Deliver hook.
func (a *actor) post(ctx context.Context, msg any) error {
	select {
	case a.inbox <- msg:
		return nil
	case <-a.done:
		return room.ErrRoomGone
	case <-a.ctx.Done():
		// Cancelled, perhaps before Run ever started.
		return room.ErrRoomGone
	case <-ctx.Done():
		return ctx.Err()
	}
}

// request posts build's message and waits for its reply.
func request[T any](ctx context.Context, a *actor, build func(reply chan T) any) (T, error) {
	var zero T
	reply := make(chan T, 1)
	if err := a.post(ctx, build(reply)); err != nil {
		return zero, err
	}
	select {
	case v := <-reply:
		return v, nil
	case <-a.done:
		select {
		case v := <-reply:
			return v, nil
		default:
			return zero, room.ErrRoomGone
		}
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// call runs fn on the actor goroutine and waits for it.
func (a *actor) call(ctx context.Context, fn func()) error {
	_, err := request(ctx, a, func(reply chan struct{}) any { return msgCall{fn: fn, reply: reply} })
	return err
}

func (a *actor) attachHook(ctx context.Context, c *room.Client, claim *string) error {
	err, rerr := request(ctx, a, func(reply chan error) any { return msgAttach{c: c, claim: claim, reply: reply} })
	if rerr != nil {
		return rerr
	}
	return err
}

func (a *actor) detachHook(c *room.Client) {
	select {
	case a.inbox <- msgDetach{c: c}:
	case <-a.done:
	}
}

func (a *actor) evictHook(ctx context.Context) error {
	_, err := request(ctx, a, func(reply chan bool) any { return msgEvict{reply: reply} })
	return err
}

// --- membership and presence ----------------------------------------------

// seatOf returns the seat the user controls, or -1.
func (a *actor) seatOf(userID string) int {
	for s, sr := range a.st.rec.Seats {
		if sr.Kind == SeatHuman && sr.UserID == userID {
			return s
		}
	}
	return -1
}

func (a *actor) connected(userID string) bool {
	return slices.ContainsFunc(a.conns, func(c *room.Client) bool { return c.UserID == userID })
}

// humanAttached reports whether a human participant (a user controlling a
// seat) has a connection. Spectators alone keep no bot running.
func (a *actor) humanAttached() bool {
	return slices.ContainsFunc(a.conns, func(c *room.Client) bool { return a.seatOf(c.UserID) >= 0 })
}

func (a *actor) attach(c *room.Client, claim *string) error {
	if claim != nil {
		return room.ErrBadClaim
	}
	already := slices.Contains(a.conns, c)
	wasConnected := a.connected(c.UserID)
	seat := a.seatOf(c.UserID)
	if !already {
		if seat >= 0 {
			n := 0
			for _, o := range a.conns {
				if o.UserID == c.UserID {
					n++
				}
			}
			if n >= a.t.cfg.MaxClients {
				return room.ErrRoomFull
			}
		} else if len(a.conns) >= a.t.cfg.MaxClients {
			return room.ErrRoomFull
		}
	}
	claimed := false
	if seat < 0 && a.st.rec.Status == store.StatusPlaying && validName(c.Name) {
		for s, sr := range a.st.rec.Seats {
			if sr.Kind != SeatOpen {
				continue
			}
			cand := a.st.rec.clone()
			cand.Seats[s] = SeatRecord{Seat: s, Kind: SeatHuman, UserID: c.UserID, Name: c.Name}
			if cand.Original[s] == "" {
				cand.Original[s] = c.UserID
			}
			cand.Public++
			if err := a.commit("claim", cand, a.st.game); err != nil {
				return room.ErrStorage
			}
			claimed = true
			break
		}
	}
	if !already {
		a.conns = append(a.conns, c)
	}
	b := a.newBroadcast(false)
	c.Out.Push(b.frame(c, nil, true))
	if claimed || !wasConnected {
		a.broadcastExcept(b, c)
	}
	a.armIdle()
	a.publishIdle()
	a.kickBots()
	return nil
}

func (a *actor) detach(c *room.Client) {
	i := slices.Index(a.conns, c)
	if i < 0 {
		return
	}
	a.conns = slices.Delete(a.conns, i, i+1)
	if !a.connected(c.UserID) {
		a.broadcastExcept(a.newBroadcast(false), nil)
	}
	if !a.humanAttached() {
		a.stopBots()
	}
	if len(a.conns) == 0 {
		a.emptySince = a.now()
	}
	a.armIdle()
	a.publishIdle()
}

// armIdle starts the idle countdown while nobody is connected.
func (a *actor) armIdle() {
	if len(a.conns) > 0 {
		a.disarmIdle()
		return
	}
	if a.idleStop != nil {
		return
	}
	a.idleStop = a.t.cfg.Clock.AfterFunc(a.t.cfg.IdleTimeout, func() {
		select {
		case a.inbox <- msgIdle{}:
		case <-a.done:
		}
	})
}

func (a *actor) disarmIdle() {
	if a.idleStop != nil {
		a.idleStop()
		a.idleStop = nil
	}
}

// publishIdle tells the Manager whether the table may be evicted: every
// change is already durable, so an unconnected table always may.
func (a *actor) publishIdle() {
	idle := len(a.conns) == 0
	if idle == a.shownIdle && a.emptySince.Equal(a.shownSince) {
		return
	}
	a.shownIdle, a.shownSince = idle, a.emptySince
	a.m.PublishIdle(a.h, idle, a.emptySince)
}

// --- requests -----------------------------------------------------------

// Outcomes of a submission, as the table answers and logs them.
const (
	outAccepted  = "accepted"
	outResolved  = "resolved"
	outDuplicate = "duplicate"
	outStale     = "stale"
	outIllegal   = "illegal"
	outClosed    = "decision-closed"
	outForbidden = "forbidden"
	outStorage   = "storage"
	outBad       = "bad-request"
)

func (a *actor) inbound(c *room.Client, f wire.ClientMessage) {
	if !slices.Contains(a.conns, c) {
		c.Out.Push(wire.NewTableReject(seqOfFrame(f), outBad))
		return
	}
	switch f := f.(type) {
	case wire.TablePick:
		a.submit(c, f)
	case wire.TableUndo:
		a.undo(c, f)
	case wire.Resync:
		c.Out.Push(a.newBroadcast(false).frame(c, nil, true))
	default:
		// Seat switches, roles, resignation, rematch and device links are
		// M16.haftgane-lobby-and-roles's.
		c.Out.Push(wire.NewTableReject(seqOfFrame(f), outBad))
	}
}

func seqOfFrame(f wire.ClientMessage) int {
	_, _, seq, _ := wire.PlatformTarget(f)
	return seq
}

// note logs an operation's safe fields only: the operation, the seat, the
// public revision and the outcome, never a key, a card, a seed or a hash.
func (a *actor) note(op string, seat int, outcome string) string {
	a.t.log.Info("table", "room", a.code, "op", op, "seat", seat, "rev", a.st.rec.Public, "outcome", outcome)
	return outcome
}

// refuse answers a private reject to the submitter only.
func (a *actor) refuse(c *room.Client, seq int, op string, seat int, outcome string) string {
	c.Out.Push(wire.NewTableReject(seq, outcome))
	return a.note(op, seat, outcome)
}

func seqKey(c *room.Client) SeqRecord { return SeqRecord{User: c.UserID, Client: c.ClientID} }

// submit applies one {seq, point, key, confirm} submission, checking in
// the design's order: membership, the client's seq acknowledgement, the
// open point reference, the seat being required and the key offered, and
// a seq older than the one that last changed the slot from this client.
func (a *actor) submit(c *room.Client, p wire.TablePick) string {
	r := &a.st.rec
	seat := a.seatOf(c.UserID)
	if seat < 0 {
		return a.refuse(c, p.Seq, "pick", seat, outForbidden)
	}
	key := seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && p.Seq <= last {
		c.Out.Push(a.newBroadcast(false).frame(c, &wire.PlatformAck{ClientID: c.ClientID, Seq: p.Seq}, false))
		return a.note("pick", seat, outDuplicate)
	}
	pt := a.st.game.Point()
	if pointOver(pt) || r.Point == nil || p.Point != *r.Point {
		return a.refuse(c, p.Seq, "pick", seat, outClosed)
	}
	if !pt.Required.Has(seatNum(seat)) || !slices.Contains(legalKeys(&a.st.game, seat), p.Key) {
		return a.refuse(c, p.Seq, "pick", seat, outIllegal)
	}
	if slot := pendingOf(r, seat); slot != nil && p.Seq < seqIn(slot.Seqs, key) {
		return a.refuse(c, p.Seq, "pick", seat, outStale)
	}
	cand := r.clone()
	key.Seq = p.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	return a.transact(c, "pick", seat, cand, p.Key, p.Confirm, &key, p.Seq)
}

// transact applies an accepted pick to cand: the slot's key and confirmed
// flag, and the confirmation that completes the required set appends one
// sorted batch, resolves it and opens the next point in the same write.
// A confirmation flip or a resolution advances the public revision; any
// other change is the seat's private edit. Then it persists, swaps and
// publishes. c is nil for a bot.
func (a *actor) transact(c *room.Client, op string, seat int, cand Record, key string, confirm bool, by *SeqRecord, seq int) string {
	slot := pendingOf(&cand, seat)
	wasConfirmed := slot != nil && slot.Confirmed
	if slot == nil {
		cand.Pending = append(cand.Pending, PendingSlot{Seat: seat, Seqs: []SeqRecord{}})
		slices.SortFunc(cand.Pending, func(x, y PendingSlot) int { return x.Seat - y.Seat })
		slot = pendingOf(&cand, seat)
	}
	slot.Key, slot.Confirmed = key, confirm
	if by != nil {
		slot.Seqs = seqSet(slot.Seqs, *by)
	}
	g := a.st.game
	pt := g.Point()
	complete := true
	for _, s := range pt.Required.Seats() {
		ps := pendingOf(&cand, int(s))
		complete = complete && ps != nil && ps.Confirmed
	}
	outcome := outAccepted
	public := wasConfirmed != confirm
	if complete {
		b := BatchRecord{Point: pt.ID, Picks: make([]PickRecord, 0, pt.Required.Count())}
		for _, s := range pt.Required.Seats() {
			b.Picks = append(b.Picks, PickRecord{Seat: int(s), Key: pendingOf(&cand, int(s)).Key})
		}
		next, _, err := g.Resolve(engineBatches([]BatchRecord{b})[0])
		if err != nil {
			// Every key was offered when it was accepted, so this is a bug;
			// refuse rather than persist a record that cannot replay.
			a.t.log.Error("table resolve failed", "room", a.code, "op", op, "seat", seat)
			if c != nil {
				return a.refuse(c, seq, op, seat, outStorage)
			}
			return a.note(op, seat, outStorage)
		}
		g = next
		cand.Batches = append(cand.Batches, b)
		cand.Pending = []PendingSlot{}
		cand.Hash = g.Fingerprint()
		public = true
		outcome = outResolved
	}
	if public {
		cand.Public++
	} else {
		cand.Edits[seat]++
	}
	if complete {
		openPoint(&cand, &g)
	}
	if err := a.commit(op, cand, g); err != nil {
		if c != nil {
			return a.refuse(c, seq, op, seat, outStorage)
		}
		return a.note(op, seat, outStorage)
	}
	var ack *wire.PlatformAck
	if c != nil {
		ack = &wire.PlatformAck{ClientID: c.ClientID, Seq: seq}
	}
	b := a.newBroadcast(complete)
	if public {
		a.publish(b, c, ack)
	} else {
		a.publishSeat(b, seat, c, ack)
	}
	if complete {
		a.stopBots()
	}
	a.kickBots()
	return a.note(op, seat, outcome)
}

// commit writes cand durably, then swaps it in with its engine value. A
// failed write leaves the committed state as it was.
func (a *actor) commit(op string, cand Record, g game) error {
	cand.Updated = a.now()
	data, err := cand.encode()
	if err == nil {
		err = a.t.store().SaveGame(a.code, Game, data)
	}
	if err != nil {
		a.t.log.Warn("table write failed", "room", a.code, "op", op, "err", err)
		return err
	}
	a.st = &state{rec: cand, game: g}
	return nil
}

// undoAllowed: only the original sole human, while still controlling
// their seat, with undo allowed at Start and not switched off since, and
// only while the game runs.
func (a *actor) undoAllowed(userID string, seat int) bool {
	r := &a.st.rec
	return r.AllowUndo && r.StartingHumans == 1 && !r.UndoOff && r.Status == store.StatusPlaying &&
		seat >= 0 && r.Original[seat] == userID && r.Seats[seat].UserID == userID
}

// undo cuts before the latest complete batch holding the human's
// decision, clears the pending block and reopens that point under a fresh
// reference. Base must be the requester's latest frame revision.
func (a *actor) undo(c *room.Client, u wire.TableUndo) string {
	r := &a.st.rec
	seat := a.seatOf(c.UserID)
	if seat < 0 {
		return a.refuse(c, u.Seq, "undo", seat, outForbidden)
	}
	key := seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && u.Seq <= last {
		c.Out.Push(a.newBroadcast(false).frame(c, &wire.PlatformAck{ClientID: c.ClientID, Seq: u.Seq}, false))
		return a.note("undo", seat, outDuplicate)
	}
	if !a.undoAllowed(c.UserID, seat) {
		return a.refuse(c, u.Seq, "undo", seat, outForbidden)
	}
	if u.Base != r.Public+r.Edits[seat] {
		return a.refuse(c, u.Seq, "undo", seat, outStale)
	}
	cut := -1
	for i := len(r.Batches) - 1; i >= 0 && cut < 0; i-- {
		if slices.ContainsFunc(r.Batches[i].Picks, func(p PickRecord) bool { return p.Seat == seat }) {
			cut = i
		}
	}
	if cut < 0 {
		return a.refuse(c, u.Seq, "undo", seat, outIllegal)
	}
	cand := r.clone()
	key.Seq = u.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	cand.Batches = cand.Batches[:cut]
	cand.Pending = []PendingSlot{}
	cand.Public++
	su := makeSetup(cand.Setup.Seed, cand.Setup.Seats, cand.Setup.RandomSeating, cand.Setup.HideAtEnd)
	g, err := replay(su, engineBatches(cand.Batches))
	if err != nil {
		a.t.log.Error("table undo replay failed", "room", a.code, "seat", seat)
		return a.refuse(c, u.Seq, "undo", seat, outStorage)
	}
	cand.Hash = g.Fingerprint()
	openPoint(&cand, &g)
	if err := a.commit("undo", cand, g); err != nil {
		return a.refuse(c, u.Seq, "undo", seat, outStorage)
	}
	a.stopBots()
	b := a.newBroadcast(false)
	b.history = true
	a.publish(b, c, &wire.PlatformAck{ClientID: c.ClientID, Seq: u.Seq})
	a.kickBots()
	return a.note("undo", seat, outAccepted)
}

// --- the pending block and seq acknowledgements ----------------------------

func pendingOf(r *Record, seat int) *PendingSlot {
	for i := range r.Pending {
		if r.Pending[i].Seat == seat {
			return &r.Pending[i]
		}
	}
	return nil
}

func sameClient(a, b SeqRecord) bool { return a.User == b.User && a.Client == b.Client }

func seqIn(xs []SeqRecord, k SeqRecord) int {
	for _, x := range xs {
		if sameClient(x, k) {
			return x.Seq
		}
	}
	return 0
}

func seqSet(xs []SeqRecord, k SeqRecord) []SeqRecord {
	for i := range xs {
		if sameClient(xs[i], k) {
			xs[i].Seq = k.Seq
			return xs
		}
	}
	return append(xs, k)
}

// seqLast returns the client's last applied seq. A lookup never reorders
// the list: a duplicate changes nothing, in memory or on disk, so a table
// restarted from its record forgets the same clients as one that never
// stopped.
func seqLast(xs []SeqRecord, k SeqRecord) (int, bool) {
	for _, x := range xs {
		if sameClient(x, k) {
			return x.Seq, true
		}
	}
	return 0, false
}

// seqPut records k as the client's last applied seq, moving the client to
// the end, and beyond capacity forgets the clients whose last applied seq
// is the oldest.
func seqPut(xs *[]SeqRecord, k SeqRecord, capacity int) {
	for i, x := range *xs {
		if sameClient(x, k) {
			*xs = slices.Delete(*xs, i, i+1)
			break
		}
	}
	*xs = append(*xs, k)
	if len(*xs) > capacity {
		*xs = slices.Clone((*xs)[len(*xs)-capacity:])
	}
}
