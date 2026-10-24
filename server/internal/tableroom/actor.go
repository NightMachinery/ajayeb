package tableroom

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
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
	viewHash    string
	engineViews map[int]*roleView
	t           *Tables
	m           *room.Manager
	h           *room.Handle
	code        string
	st          *state
	conns       []*room.Client
	// claims holds, for each connection that joined with a device link,
	// the claim and the user it acts as (package ident).
	claims map[*room.Client]claimed
	// shown is the viewer role (seat, or -1) of the last frame each
	// connection got. A frame for another role carries the whole history,
	// since the connection's events so far were redacted for its old role.
	shown map[*room.Client]int
	// budget is the users' request budget, and reached the most batches
	// the game has had since this actor started: a commit past it opens
	// a point not reached before, which fills every budget.
	budget  *Budget
	reached int

	inbox  chan any
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	idleStop  func() bool
	rolesStop func() bool
	// rolesAhead holds the roles of the last check whose write the store
	// refused, until a write saves them (roles.Carry, roles.Holds).
	rolesAhead *roles.Roles
	emptySince time.Time
	shownIdle  bool
	shownSince time.Time

	botGen    uint64
	botBusy   bool
	botCancel context.CancelFunc
	botWG     sync.WaitGroup
	// botBusy marks a valid task; botRunning also covers cancelled cleanup.
	botRunning bool
	botRunID   uint64
	botSeat    int
	botPoint   string
}

// claimed is a connection's device-link claim and the user it acts as.
type claimed struct{ claim, user string }

type msgAttach struct {
	c     *room.Client
	claim *string
	reply chan error
}

type msgDetach struct{ c *room.Client }

type msgEvict struct{ reply chan bool }

type msgIdle struct{}

// msgRoles is the roles' clock: the temporary-admin rule's next deadline.
type msgRoles struct{}

// msgCall runs fn on the actor goroutine (tests and diagnostics).
type msgCall struct {
	fn    func()
	reply chan struct{}
}

type msgBotResult struct {
	owner    *actor
	gen      uint64
	point    string
	seat     int
	key      string
	err      error
	fallback string
}

// newActor builds a table's actor. members are connections the lobby
// hands over at Start: the actor sends each its first frame when it runs.
func (t *Tables) newActor(m *room.Manager, st *state, members []Member) (*room.Handle, error) {
	a := &actor{t: t, m: m, code: st.rec.Code, st: st, claims: map[*room.Client]claimed{}, shown: map[*room.Client]int{},
		budget: NewBudget(t.cfg.RequestBurst, t.cfg.RequestInterval), reached: len(st.rec.Batches),
		inbox: make(chan any, t.cfg.InboxSize), done: make(chan struct{})}
	for _, mb := range members {
		a.conns = append(a.conns, mb.Client)
		if mb.Claim != "" {
			a.claims[mb.Client] = claimed{claim: mb.Claim, user: mb.User}
		}
	}
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
	defer a.disarmRoles()
	a.emptySince = a.now()
	if len(a.conns) > 0 {
		// Handed over by the lobby's Start: everyone's first table frame,
		// but for a claim that names no seat (none should).
		a.revokeClaims()
		b := a.newBroadcast(false)
		b.history = true
		a.publish(b, nil, nil)
		a.checkRoles()
		a.kickBots()
	}
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
	case msgRoles:
		a.rolesStop = nil
		a.checkRoles()
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

// userOf is the user a connection acts as: the user of the seat whose
// device link it joined with, or its own.
func (a *actor) userOf(c *room.Client) string {
	if cl, ok := a.claims[c]; ok {
		return cl.user
	}
	return c.UserID
}

// claimHolder returns the user of the human seat claim was made for.
// Every seat is checked, in constant time each, whatever the first says.
// Without device links nothing is a claim.
func (a *actor) claimHolder(r *Record, claim string) (string, bool) {
	cl := a.m.Claims()
	if cl == nil {
		return "", false
	}
	user := ""
	for _, sr := range r.Seats {
		if sr.Kind == SeatHuman && cl.VerifyClaim(claim, a.code, sr.UserID, sr.Epoch) && user == "" {
			user = sr.UserID
		}
	}
	return user, user != ""
}

// seatOf returns the seat the user controls, or -1.
func (a *actor) seatOf(userID string) int { return seatIn(&a.st.rec, userID) }

func seatIn(r *Record, userID string) int {
	for s, sr := range r.Seats {
		if sr.Kind == SeatHuman && sr.UserID == userID {
			return s
		}
	}
	return -1
}

func (a *actor) connected(userID string) bool {
	return slices.ContainsFunc(a.conns, func(c *room.Client) bool { return a.userOf(c) == userID })
}

// nameOf is a connected user's display name.
func (a *actor) nameOf(userID string) string {
	for _, c := range a.conns {
		if a.userOf(c) == userID {
			return c.Name
		}
	}
	return ""
}

// inRoom reports whether u is a human in the room: seated, or connected.
func (a *actor) inRoom(u string) bool { return a.seatOf(u) >= 0 || a.connected(u) }

// participant reports whether u keeps bots running: a user controlling a
// seat, or an original participant watching (after resigning, say).
// Unrelated spectators keep no bot running.
func (a *actor) participant(u string) bool {
	return a.seatOf(u) >= 0 || slices.Contains(a.st.rec.Original, u)
}

// humanAttached reports whether a participant has a connection.
func (a *actor) humanAttached() bool {
	return slices.ContainsFunc(a.conns, func(c *room.Client) bool { return a.participant(a.userOf(c)) })
}

// attach adds a connection. A table has no open seats to claim: a user
// without a seat watches until an admin seats them. A connection with a
// seat's claim acts as that seat's user. A connection already attached
// under another claim (or none) is detached and admitted afresh, and
// stays detached if that fails.
func (a *actor) attach(c *room.Client, claim *string) error {
	already := slices.Contains(a.conns, c)
	if cl, ok := a.claims[c]; already && (ok != (claim != nil) || ok && cl.claim != *claim) {
		a.detach(c)
		already = false
	}
	user := c.UserID
	if claim != nil {
		u, ok := a.claimHolder(&a.st.rec, *claim)
		if !ok {
			return room.ErrBadClaim
		}
		user = u
	}
	wasConnected := a.connected(user)
	if !already {
		// Every user holds at most MaxUserClients connections, and a user
		// without a seat finds room only while the table has fewer than
		// MaxClients, so watchers cannot lock a player out.
		n := 0
		for _, o := range a.conns {
			if a.userOf(o) == user {
				n++
			}
		}
		if n >= a.t.cfg.MaxUserClients || a.seatOf(user) < 0 && len(a.conns) >= a.t.cfg.MaxClients {
			return room.ErrRoomFull
		}
		a.conns = append(a.conns, c)
		if claim != nil {
			a.claims[c] = claimed{claim: *claim, user: user}
		}
	}
	b := a.newBroadcast(false)
	c.Out.Push(b.frame(c, nil, true))
	if !wasConnected {
		a.broadcastExcept(b, c)
		a.checkRoles()
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
	user := a.userOf(c)
	a.conns = slices.Delete(a.conns, i, i+1)
	delete(a.claims, c)
	delete(a.shown, c)
	if !a.connected(user) {
		a.broadcastExcept(a.newBroadcast(false), nil)
		a.checkRoles()
	}
	a.dropped()
}

// dropped settles the bots and the idle clock after connections left.
func (a *actor) dropped() {
	if !a.humanAttached() {
		a.stopBots()
	}
	if len(a.conns) == 0 {
		a.emptySince = a.now()
		// No one watches, so the views and histories cached for this
		// position go too: for a finished seven-seat table about 560 KB
		// of heap against a record of about 15 KB, kept for up to the idle
		// timeout. The next join rebuilds its role's.
		a.engineViews = nil
	}
	a.armIdle()
	a.publishIdle()
}

// --- roles ------------------------------------------------------------------

// checkRoles applies the temporary-admin rule now (package roles), from
// the saved roles and any the store refused before, and persists what it
// changed: a role change publicly, a new away time silently. It arms the
// roles' clock for the rule's next deadline, or for a retry when the
// store refuses the write. A finished table changes no roles.
func (a *actor) checkRoles() {
	if a.st.rec.Status != store.StatusPlaying {
		a.rolesAhead = nil
		a.disarmRoles()
		return
	}
	cand := a.st.rec.clone()
	at := len(cand.Batches)
	log := cand.Roles.Carry(a.rolesAhead, at)
	more, next := cand.Roles.Check(a.now(), a.connected, a.candidates(&cand), at)
	log = append(log, more...)
	if len(log) == 0 && sameAway(cand.Roles.Away, a.st.rec.Roles.Away) {
		a.rolesAhead = nil
		a.armRoles(next)
		return
	}
	if len(log) > 0 {
		appendLog(&cand, log...)
		bumpPublic(&cand)
		retire(&a.st.rec, &cand)
	}
	if err := a.commit("roles", cand, a.st.game); err != nil {
		ahead := cand.Roles.Clone()
		a.rolesAhead = &ahead
		if retry := a.now().Add(roles.Retry); next.IsZero() || retry.Before(next) {
			next = retry
		}
		a.armRoles(next)
		return
	}
	a.rolesAhead = nil
	if len(log) > 0 {
		a.publish(a.newBroadcast(false), nil, nil)
		a.stopBots()
		a.kickBots()
	}
	a.armRoles(next)
}

// isAdmin reports whether u has the admin powers now (roles.Holds): a
// temporary admin whose demotion or replacement the store refused has
// none, nor does one whose grant it refused. Frames ask it for every
// connection and human, so only the temporary admin pays for the check.
func (a *actor) isAdmin(u string) bool {
	r := &a.st.rec
	if r.Roles.Real(u) || u == "" || r.Roles.Temporary != u {
		return r.Roles.Real(u)
	}
	return r.Roles.Holds(u, a.rolesAhead, a.now(), a.connected, a.candidates(r))
}

func sameAway(x, y []roles.Away) bool {
	return slices.EqualFunc(x, y, func(p, q roles.Away) bool { return p.UserID == q.UserID && p.Since.Equal(q.Since) })
}

// candidates are the connected humans in the temporary-admin rule's
// order: seated humans by how long they have held their seat (seat order
// breaks ties), then spectators by how long they have been connected.
func (a *actor) candidates(r *Record) []string {
	var seated []SeatRecord
	for _, sr := range r.Seats {
		if sr.Kind == SeatHuman && a.connected(sr.UserID) {
			seated = append(seated, sr)
		}
	}
	slices.SortStableFunc(seated, func(x, y SeatRecord) int { return cmp.Compare(x.Taken, y.Taken) })
	out := []string{}
	for _, sr := range seated {
		out = append(out, sr.UserID)
	}
	for _, c := range a.conns {
		if u := a.userOf(c); seatIn(r, u) < 0 && !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	return out
}

// armRoles sets the roles' clock to at (zero: no deadline).
func (a *actor) armRoles(at time.Time) {
	a.disarmRoles()
	if at.IsZero() {
		return
	}
	a.rolesStop = a.t.cfg.Clock.AfterFunc(max(at.Sub(a.now()), 0), func() {
		select {
		case a.inbox <- msgRoles{}:
		case <-a.done:
		}
	})
}

func (a *actor) disarmRoles() {
	if a.rolesStop != nil {
		a.rolesStop()
		a.rolesStop = nil
	}
}

// retire keeps the retired list after a control change from before to
// after: users who lost all authority join it (in user order, so the
// list follows public changes only), users who regained authority leave
// it, and beyond maxRetired the oldest is forgotten with its seqs.
func retire(before, after *Record) {
	after.Retired = slices.DeleteFunc(after.Retired, func(u string) bool { return hasAuthority(after, u) })
	var lost []string
	for _, u := range authorities(before) {
		if !hasAuthority(after, u) && !slices.Contains(after.Retired, u) {
			lost = append(lost, u)
		}
	}
	slices.Sort(lost)
	after.Retired = append(after.Retired, lost...)
	for len(after.Retired) > maxRetired {
		u := after.Retired[0]
		after.Retired = slices.Delete(after.Retired, 0, 1)
		after.Seqs = slices.DeleteFunc(after.Seqs, func(x SeqRecord) bool { return x.User == u })
	}
}

// authorities are the users with authority at the table.
func authorities(r *Record) []string {
	var out []string
	for _, sr := range r.Seats {
		if sr.Kind == SeatHuman {
			out = append(out, sr.UserID)
		}
	}
	for _, ad := range r.Roles.Admins {
		out = append(out, ad.UserID)
	}
	if r.Roles.Temporary != "" {
		out = append(out, r.Roles.Temporary)
	}
	return out
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
	outNotAdmin  = "not-admin"
	outRate      = "rate"
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
	case wire.TableSeat:
		a.switchSeat(c, f)
	case wire.TableRole:
		a.role(c, f)
	case wire.TableResign:
		a.resign(c, f)
	case wire.PlatformLink:
		a.link(c, f.Seat, false)
	case wire.PlatformRelink:
		a.link(c, f.Seat, true)
	case wire.Link:
		a.link(c, f.Seat, false)
	case wire.Relink:
		a.link(c, f.Seat, true)
	case wire.Rematch:
		a.rematch(c)
	case wire.Resync:
		c.Out.Push(a.newBroadcast(false).frame(c, nil, true))
	default:
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

// duplicate answers a repeated seq with the submitter's current frame,
// acknowledging it; nothing changes.
func (a *actor) duplicate(c *room.Client, op string, seat int, seq int) string {
	c.Out.Push(a.newBroadcast(false).frame(c, &wire.PlatformAck{ClientID: c.ClientID, Seq: seq}, false))
	return a.note(op, seat, outDuplicate)
}

// namesSeat reports whether a request that may name the seat it was made
// for (a pick, an undo, a resignation) fits the sender's current seat. One
// sent before an admin moved the sender, and arriving after, names the old
// seat and is refused with forbidden rather than acting on the new one. A
// request naming no seat acts on the current seat, as older clients'
// requests always did.
func namesSeat(named *int, seat int) bool { return named == nil || *named == seat }

func (a *actor) seqKey(c *room.Client) SeqRecord {
	return SeqRecord{User: a.userOf(c), Client: ClientKey(c.ClientID)}
}

// spend charges a request that passed every other check, and is about to
// change the table, to its sender's user (Budget); false means refuse it
// with rate. Duplicates and refusals never reach it, nor do bots.
func (a *actor) spend(c *room.Client) bool { return a.budget.Spend(a.userOf(c), a.now()) }

// submit applies one {seq, point, key, confirm} submission, checking in
// the design's order: membership (the sender controls a seat, the one the
// request names if it names one), the client's seq acknowledgement, the
// open point reference, the seat being required and the key offered, and
// the slot's floor for this client: a seq older than the one that last
// changed the slot from it, or a new client past the slot's bound (rate).
func (a *actor) submit(c *room.Client, p wire.TablePick) string {
	r := &a.st.rec
	seat := a.seatOf(a.userOf(c))
	if seat < 0 || !namesSeat(p.Seat, seat) {
		return a.refuse(c, p.Seq, "pick", seat, outForbidden)
	}
	key := a.seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && p.Seq <= last {
		return a.duplicate(c, "pick", seat, p.Seq)
	}
	pt := a.st.game.Point()
	if pointOver(pt) || r.Point == nil || p.Point != *r.Point {
		return a.refuse(c, p.Seq, "pick", seat, outClosed)
	}
	if !pt.Required.Has(seatNum(seat)) || !slices.Contains(legalKeys(&a.st.game, seat), p.Key) {
		return a.refuse(c, p.Seq, "pick", seat, outIllegal)
	}
	// The acknowledgements may have forgotten this client, but the slot
	// keeps the seq that last changed it from each client: that same seq
	// is acknowledged again, an older one is stale. A floor is looked up
	// by presence, since seq 0 is valid. The slot keeps a bounded number
	// of each user's floors: past it a new client is refused, because
	// dropping a floor would let a forgotten old seq overwrite this pick.
	// It keeps the floors of every user who controlled the seat during the
	// point, up to maxSlotUsers of them, so a switch never reopens one.
	if slot := pendingOf(r, seat); slot != nil {
		floor, ok := seqLast(slot.Seqs, key)
		users := seqUsers(slot.Seqs, key.User)
		switch {
		case ok && p.Seq == floor:
			return a.duplicate(c, "pick", seat, p.Seq)
		case ok && p.Seq < floor:
			return a.refuse(c, p.Seq, "pick", seat, outStale)
		case !ok && users >= a.t.cfg.PickClients, users == 0 && slotUsers(slot.Seqs) >= maxSlotUsers:
			return a.refuse(c, p.Seq, "pick", seat, outRate)
		}
	}
	cand := r.clone()
	key.Seq = p.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	return a.transact(c, "pick", seat, cand, p.Key, p.Confirm, &key, p.Seq)
}

// transact applies an accepted pick to cand (applyPick). A confirmation
// flip or a resolution is a public change; any other change is the seat's
// private edit, refused with rate past maxEdits between public changes.
// Then it persists, swaps and publishes. c is nil for a bot.
func (a *actor) transact(c *room.Client, op string, seat int, cand Record, key string, confirm bool, by *SeqRecord, seq int) string {
	fail := func(outcome string) string {
		if c != nil {
			return a.refuse(c, seq, op, seat, outcome)
		}
		return a.note(op, seat, outcome)
	}
	g, resolved, public, err := a.applyPick(&cand, a.st.game, seat, key, confirm, by)
	switch {
	case err != nil:
		return fail(outStorage)
	case public:
		bumpPublic(&cand)
	case cand.Edits[seat] >= maxEdits:
		return fail(outRate)
	default:
		cand.Edits[seat]++
	}
	if resolved {
		openPoint(&cand, &g)
	}
	if c != nil && !a.spend(c) {
		return fail(outRate)
	}
	if err := a.commit(op, cand, g); err != nil {
		return fail(outStorage)
	}
	var ack *wire.PlatformAck
	if c != nil {
		ack = &wire.PlatformAck{ClientID: c.ClientID, Seq: seq}
	}
	b := a.newBroadcast(resolved)
	if public {
		a.publish(b, c, ack)
	} else {
		a.publishSeat(b, seat, c, ack)
	}
	if resolved {
		a.stopBots()
	}
	a.kickBots()
	outcome := outAccepted
	if resolved {
		outcome = outResolved
	}
	return a.note(op, seat, outcome)
}

// applyPick sets seat's pending pick on cand: its key, its confirmation
// and, from a client, that client's floor. The confirmation that completes
// the required set appends one sorted batch and resolves it. It returns
// the engine value after the pick, whether it resolved, and whether the
// change is public (a confirmation flip or a resolution); the caller
// advances the revisions and opens the next point.
func (a *actor) applyPick(cand *Record, g game, seat int, key string, confirm bool, by *SeqRecord) (game, bool, bool, error) {
	slot := pendingOf(cand, seat)
	wasConfirmed := slot != nil && slot.Confirmed
	if slot == nil {
		cand.Pending = append(cand.Pending, PendingSlot{Seat: seat, Seqs: []SeqRecord{}})
		slices.SortFunc(cand.Pending, func(x, y PendingSlot) int { return x.Seat - y.Seat })
		slot = pendingOf(cand, seat)
	}
	slot.Key, slot.Confirmed = key, confirm
	if by != nil {
		slot.Seqs = seqSet(slot.Seqs, *by)
	}
	pt := g.Point()
	for _, s := range pt.Required.Seats() {
		if ps := pendingOf(cand, int(s)); ps == nil || !ps.Confirmed {
			return g, false, wasConfirmed != confirm, nil
		}
	}
	b := BatchRecord{Point: pt.ID, Picks: make([]PickRecord, 0, pt.Required.Count())}
	for _, s := range pt.Required.Seats() {
		b.Picks = append(b.Picks, PickRecord{Seat: int(s), Key: pendingOf(cand, int(s)).Key})
	}
	next, _, err := g.Resolve(engineBatches([]BatchRecord{b})[0])
	if err != nil {
		// Every key was offered when it was accepted, so this is a bug;
		// refuse rather than persist a record that cannot replay.
		a.t.log.Error("table resolve failed", "room", a.code, "seat", seat)
		return g, false, false, err
	}
	cand.Batches = append(cand.Batches, b)
	cand.Pending = []PendingSlot{}
	cand.Hash = next.Fingerprint()
	return next, true, true, nil
}

// --- seat switches, roles and resignation -----------------------------------

// switchSeat handles table-seat: an admin gives a seat to a bot (at the
// level the request names, which the table must have frozen) or seats a
// spectator in it. A human replaced
// becomes a spectator. The seat's pending pick stays: a bot taking the
// seat confirms it, a human may still change it.
func (a *actor) switchSeat(c *room.Client, f wire.TableSeat) string {
	r := &a.st.rec
	user := a.userOf(c)
	key := a.seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && f.Seq <= last {
		return a.duplicate(c, "seat", f.Seat, f.Seq)
	}
	switch {
	case !a.isAdmin(user):
		return a.refuse(c, f.Seq, "seat", f.Seat, outNotAdmin)
	case r.Status != store.StatusPlaying:
		return a.refuse(c, f.Seq, "seat", f.Seat, outForbidden)
	case f.Seat < 0 || f.Seat >= len(r.Seats):
		return a.refuse(c, f.Seq, "seat", -1, outBad)
	}
	cur := r.Seats[f.Seat]
	cand := r.clone()
	seat, control := f.Seat, f.Control
	entry := wire.PlatformLogEntry{Kind: "seat", At: len(r.Batches), By: user, Seat: &seat, Control: &control}
	switch {
	case f.Control == SeatBot && f.Bot != nil && f.UserID == nil:
		if _, ok := frozenPreset(r, *f.Bot); !ok || (cur.Kind == SeatBot && cur.Bot == *f.Bot) {
			return a.refuse(c, f.Seq, "seat", f.Seat, outBad)
		}
		cand.Seats[f.Seat] = SeatRecord{Seat: f.Seat, Kind: SeatBot, Bot: *f.Bot}
	case f.Control == SeatHuman && f.UserID != nil && f.Bot == nil:
		target := *f.UserID
		if a.seatOf(target) >= 0 || !a.connected(target) {
			return a.refuse(c, f.Seq, "seat", f.Seat, outBad)
		}
		cand.Seats[f.Seat] = SeatRecord{Seat: f.Seat, Kind: SeatHuman, UserID: target, Name: a.nameOf(target),
			Taken: cand.Public + 1, Epoch: cand.Public + 1}
		entry.UserID = &target
		secondHuman(&cand, target)
	default:
		return a.refuse(c, f.Seq, "seat", f.Seat, outBad)
	}
	if cur.Kind == SeatHuman {
		rememberSeat(&cand, cur.UserID, f.Seat)
	}
	key.Seq = f.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	appendLog(&cand, entry)
	return a.control(c, "seat", f.Seat, cand, f.Seq)
}

// secondHuman turns undo off for good once a human other than the
// original sole human takes a seat.
func secondHuman(r *Record, user string) {
	if r.StartingHumans != 1 {
		return
	}
	for _, o := range r.Original {
		if o != "" && o != user {
			r.UndoOff = true
		}
	}
}

// rememberSeat records the seat a player is leaving, for the admins'
// previous-seat hint.
func rememberSeat(r *Record, user string, seat int) {
	r.Previous = slices.DeleteFunc(r.Previous, func(p PreviousRecord) bool { return p.UserID == user })
	r.Previous = append(r.Previous, PreviousRecord{UserID: user, Seat: seat})
	if over := len(r.Previous) - maxPrevious; over > 0 {
		r.Previous = slices.Delete(r.Previous, 0, over)
	}
}

// role handles table-role: promote, demote or step down (package roles).
func (a *actor) role(c *room.Client, f wire.TableRole) string {
	r := &a.st.rec
	user := a.userOf(c)
	key := a.seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && f.Seq <= last {
		return a.duplicate(c, "role", -1, f.Seq)
	}
	if r.Status != store.StatusPlaying {
		return a.refuse(c, f.Seq, "role", -1, outForbidden)
	}
	cand := r.clone()
	at := len(r.Batches)
	var entry wire.PlatformLogEntry
	var err error
	switch {
	case f.Op == "promote" && f.UserID != nil:
		entry, err = cand.Roles.Promote(user, *f.UserID, a.inRoom(*f.UserID), at)
	case f.Op == "demote" && f.UserID != nil:
		entry, err = cand.Roles.Demote(user, *f.UserID, at)
	case f.Op == "step-down" && f.UserID == nil:
		entry, err = cand.Roles.StepDown(user, at)
	default:
		err = roles.ErrBadRequest
	}
	if err != nil {
		return a.refuse(c, f.Seq, "role", -1, roleOutcome(err))
	}
	key.Seq = f.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	appendLog(&cand, entry)
	return a.control(c, "role", -1, cand, f.Seq)
}

func roleOutcome(err error) string {
	switch {
	case errors.Is(err, roles.ErrNotAdmin):
		return outNotAdmin
	case errors.Is(err, roles.ErrForbidden):
		return outForbidden
	}
	return outBad
}

// resign handles table-resign: the sender's seat goes to a bot at the
// lowest released level, which confirms its pending pick. Nobody wins by
// it, and the game goes on.
func (a *actor) resign(c *room.Client, f wire.TableResign) string {
	r := &a.st.rec
	user := a.userOf(c)
	key := a.seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && f.Seq <= last {
		return a.duplicate(c, "resign", -1, f.Seq)
	}
	seat := a.seatOf(user)
	switch {
	case !f.Confirm:
		return a.refuse(c, f.Seq, "resign", seat, outBad)
	case seat < 0 || !namesSeat(f.Seat, seat) || r.Status != store.StatusPlaying:
		return a.refuse(c, f.Seq, "resign", seat, outForbidden)
	}
	cand := r.clone()
	cand.Seats[seat] = SeatRecord{Seat: seat, Kind: SeatBot, Bot: DefaultBotLevel}
	rememberSeat(&cand, user, seat)
	control := SeatBot
	appendLog(&cand, wire.PlatformLogEntry{Kind: "resign", At: len(r.Batches), By: user, Seat: &seat, Control: &control})
	key.Seq = f.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	return a.control(c, "resign", seat, cand, f.Seq)
}

// control finishes a public change of who controls a seat or holds a
// role. A bot that took seat with an unconfirmed pick confirms it, which
// can resolve the point. The open point's work rows follow the seats, the
// roles are brought up to date (a new admin demotes the temporary one;
// the last real admin stepping down makes one at once), and users who lost
// all authority retire. Then it persists, publishes to everyone and starts
// the bot job again. seat is -1 for a role change.
func (a *actor) control(c *room.Client, op string, seat int, cand Record, seq int) string {
	g := a.st.game
	resolved := false
	if seat >= 0 && cand.Seats[seat].Kind == SeatBot {
		if ps := pendingOf(&cand, seat); ps != nil && !ps.Confirmed {
			var err error
			if g, resolved, _, err = a.applyPick(&cand, g, seat, ps.Key, true, nil); err != nil {
				return a.refuse(c, seq, op, seat, outStorage)
			}
		}
	}
	bumpPublic(&cand)
	if resolved {
		openPoint(&cand, &g)
	} else {
		syncWork(&cand, &g)
	}
	at := len(cand.Batches)
	log := cand.Roles.Carry(a.rolesAhead, at)
	more, next := cand.Roles.Check(a.now(), a.connected, a.candidates(&cand), at)
	appendLog(&cand, append(log, more...)...)
	retire(&a.st.rec, &cand)
	if !a.spend(c) {
		return a.refuse(c, seq, op, seat, outRate)
	}
	if err := a.commit(op, cand, g); err != nil {
		return a.refuse(c, seq, op, seat, outStorage)
	}
	a.rolesAhead = nil
	a.armRoles(next)
	gone := a.revokeClaims()
	a.publish(a.newBroadcast(resolved), c, &wire.PlatformAck{ClientID: c.ClientID, Seq: seq})
	a.afterRevoke(gone)
	if resolved || seat >= 0 && seat == a.botSeat {
		a.stopBots()
	}
	a.kickBots()
	outcome := outAccepted
	if resolved {
		outcome = outResolved
	}
	return a.note(op, seat, outcome)
}

// syncWork makes the open point's work rows follow the seats: one for each
// required bot seat (a bot new to the seat, or at a new level, gets its
// level's fixed allowance, persisted before its work starts), none for a
// seat a human now controls. A level the table never froze gets no row, so
// the commit refuses it.
func syncWork(r *Record, g *game) {
	p := g.Point()
	out := []WorkRow{}
	if !pointOver(p) {
		for _, s := range p.Required.Seats() {
			sr := r.Seats[s]
			if sr.Kind != SeatBot {
				continue
			}
			if i := slices.IndexFunc(r.Work, func(w WorkRow) bool { return w.Seat == int(s) && w.Level == sr.Bot }); i >= 0 {
				out = append(out, r.Work[i])
				continue
			}
			if row, ok := workRow(r, p.ID, int(s)); ok {
				out = append(out, row)
			}
		}
	}
	r.Work = out
}

// commit writes cand durably, then swaps it in with its engine value. A
// failed write leaves the committed state as it was.
func (a *actor) commit(op string, cand Record, g game) error {
	cand.Updated = a.now()
	// The candidate must restore after a restart: the same checks, on the
	// engine value this transaction produced instead of a replay.
	if err := validate(&cand, &g, a.t.bounds()); err != nil {
		a.t.log.Error("table candidate invalid", "room", a.code, "op", op, "err", err)
		return err
	}
	data, err := cand.encode()
	if err == nil {
		err = a.t.store().SaveGame(a.code, Game, data)
	}
	if err != nil {
		a.t.log.Warn("table write failed", "room", a.code, "op", op, "err", err)
		return err
	}
	a.st = &state{rec: cand, game: g}
	// A point the game had not reached before is a new decision for every
	// seat: every budget is full again. Reaching a point again after an
	// undo is not new.
	if n := len(cand.Batches); n > a.reached {
		a.reached = n
		a.budget.Refill()
	}
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
	user := a.userOf(c)
	seat := a.seatOf(user)
	if seat < 0 || !namesSeat(u.Seat, seat) {
		return a.refuse(c, u.Seq, "undo", seat, outForbidden)
	}
	key := a.seqKey(c)
	if last, ok := seqLast(r.Seqs, key); ok && u.Seq <= last {
		return a.duplicate(c, "undo", seat, u.Seq)
	}
	if !a.undoAllowed(user, seat) {
		return a.refuse(c, u.Seq, "undo", seat, outForbidden)
	}
	if u.Base != revisionOf(r, seat) {
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
	if !a.spend(c) {
		return a.refuse(c, u.Seq, "undo", seat, outRate)
	}
	cand := r.clone()
	key.Seq = u.Seq
	seqPut(&cand.Seqs, key, a.t.cfg.SeqClients)
	cand.Batches = cand.Batches[:cut]
	cand.BotFallbacks = slices.DeleteFunc(cand.BotFallbacks, func(f BotFallback) bool { return f.At >= cut })
	cand.Pending = []PendingSlot{}
	bumpPublic(&cand)
	su, err := recordSetup(cand.Setup)
	if err != nil {
		return a.refuse(c, u.Seq, "undo", seat, outStorage)
	}
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

func seqSet(xs []SeqRecord, k SeqRecord) []SeqRecord {
	for i := range xs {
		if sameClient(xs[i], k) {
			xs[i].Seq = k.Seq
			return xs
		}
	}
	return append(xs, k)
}

// seqLast returns the client's remembered seq and whether it is there at
// all (seq 0 is a valid seq). A lookup never reorders the list: a
// duplicate changes nothing, in memory or on disk, so a table restarted
// from its record forgets the same clients as one that never stopped.
func seqLast(xs []SeqRecord, k SeqRecord) (int, bool) {
	for _, x := range xs {
		if sameClient(x, k) {
			return x.Seq, true
		}
	}
	return 0, false
}

// seqPut records k as the client's last accepted seq, moving the client to
// the end. Beyond capacity it forgets the client of the same user whose
// last accepted seq is the oldest: each user has a partition of their own,
// so no user's activity changes the answers another user gets.
func seqPut(xs *[]SeqRecord, k SeqRecord, capacity int) {
	for i, x := range *xs {
		if sameClient(x, k) {
			*xs = slices.Delete(*xs, i, i+1)
			break
		}
	}
	*xs = append(*xs, k)
	if seqUsers(*xs, k.User) > capacity {
		i := slices.IndexFunc(*xs, func(x SeqRecord) bool { return x.User == k.User })
		*xs = slices.Delete(*xs, i, i+1)
	}
}

// slotUsers counts the users with a floor in xs.
func slotUsers(xs []SeqRecord) int {
	var users []string
	for _, x := range xs {
		if !slices.Contains(users, x.User) {
			users = append(users, x.User)
		}
	}
	return len(users)
}

// seqUsers counts the user's clients in xs.
func seqUsers(xs []SeqRecord, user string) int {
	n := 0
	for _, x := range xs {
		if x.User == user {
			n++
		}
	}
	return n
}
