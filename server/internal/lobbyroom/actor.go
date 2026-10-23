package lobbyroom

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// actor runs one lobby. Everything below mu is touched only on its
// goroutine.
type actor struct {
	l    *Lobbies
	m    *room.Manager
	h    *room.Handle
	code string

	// mu guards the detaches the Manager delivers and, after Start, the
	// table's detach hook they go to instead: a detach is never lost
	// between the lobby and the game it started.
	mu       sync.Mutex
	detaches []*room.Client
	next     func(*room.Client)
	wake     chan struct{}

	rec    Record
	conns  []*room.Client
	claims map[*room.Client]claimed

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
}

// claimed is a connection's device-link claim and the user it acts as.
type claimed struct{ claim, user string }

type msgAttach struct {
	c     *room.Client
	claim *string
	reply chan error
}

type msgEvict struct{ reply chan bool }

type msgIdle struct{}

type msgRoles struct{}

// msgCall runs fn on the actor goroutine (tests and diagnostics).
type msgCall struct {
	fn    func()
	reply chan struct{}
}

func (l *Lobbies) newActor(m *room.Manager, r Record) (*room.Handle, error) {
	a := &actor{l: l, m: m, code: r.Code, rec: r, claims: map[*room.Client]claimed{},
		wake: make(chan struct{}, 1), inbox: make(chan any, l.cfg.InboxSize), done: make(chan struct{})}
	a.ctx, a.cancel = context.WithCancel(m.Context())
	h, err := m.NewHandle(Game, a.code, room.HandleConfig{
		Run: a.run, Cancel: a.cancel, Attach: a.attachHook, Detach: a.detachHook, Deliver: a.post, Evict: a.evictHook,
	})
	if err != nil {
		a.cancel()
		return nil, err
	}
	a.h = h
	l.register(a)
	return h, nil
}

func (a *actor) run() {
	defer a.l.unregister(a)
	defer close(a.done)
	defer a.disarmRoles()
	a.emptySince = a.now()
	a.armIdle()
	a.publishIdle()
	for {
		select {
		case <-a.ctx.Done():
			a.disarmIdle()
			return
		case <-a.wake:
			a.takeDetaches()
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
	case room.Inbound:
		return a.inbound(msg.Client, msg.Frame)
	case msgRoles:
		a.rolesStop = nil
		a.checkRoles()
	case msgCall:
		msg.fn()
		close(msg.reply)
	case msgIdle:
		if len(a.conns) == 0 && a.now().Sub(a.emptySince) >= a.l.cfg.IdleTimeout {
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

func (a *actor) now() time.Time { return a.l.cfg.Clock.Now().UTC() }

func (a *actor) post(ctx context.Context, msg any) error {
	select {
	case a.inbox <- msg:
		return nil
	case <-a.done:
		return room.ErrRoomGone
	case <-a.ctx.Done():
		return room.ErrRoomGone
	case <-ctx.Done():
		return ctx.Err()
	}
}

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

// detachHook queues a detach for the actor, or, once Start handed the
// connections to a table, passes it to the table.
func (a *actor) detachHook(c *room.Client) {
	a.mu.Lock()
	next := a.next
	if next == nil {
		a.detaches = append(a.detaches, c)
	}
	a.mu.Unlock()
	if next != nil {
		next(c)
		return
	}
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *actor) takeDetaches() {
	a.mu.Lock()
	cs := a.detaches
	a.detaches = nil
	a.mu.Unlock()
	for _, c := range cs {
		a.detach(c)
	}
}

func (a *actor) evictHook(ctx context.Context) error {
	_, err := request(ctx, a, func(reply chan bool) any { return msgEvict{reply: reply} })
	return err
}

// --- membership and presence ----------------------------------------------

// userOf is the user a connection acts as: the user of the slot whose
// claim it joined with, or its own.
func (a *actor) userOf(c *room.Client) string {
	if cl, ok := a.claims[c]; ok {
		return cl.user
	}
	return c.UserID
}

// claimHolder returns the user of the human or reserved slot claim was
// made for: a lobby's claims are of epoch 0, the one a rematch hands out
// and Start carries into the table. Every slot is checked, in constant
// time each.
func (a *actor) claimHolder(claim string) (string, bool) {
	cl := a.m.Claims()
	if cl == nil {
		return "", false
	}
	user := ""
	for _, s := range a.rec.Slots {
		if (s.Kind == SlotHuman || s.Kind == SlotReserved) && cl.VerifyClaim(claim, a.code, s.UserID, 0) && user == "" {
			user = s.UserID
		}
	}
	return user, user != ""
}

func (a *actor) connected(u string) bool {
	return slices.ContainsFunc(a.conns, func(c *room.Client) bool { return a.userOf(c) == u })
}

func (a *actor) nameOf(u string) string {
	for _, c := range a.conns {
		if a.userOf(c) == u {
			return c.Name
		}
	}
	return ""
}

// attach adds a connection. Its user takes the slot reserved for them,
// or, unless they left a slot before, the first open slot; otherwise they
// watch. A connection already attached under another claim (or none) is
// detached and admitted afresh.
func (a *actor) attach(c *room.Client, claim *string) error {
	already := slices.Contains(a.conns, c)
	if cl, ok := a.claims[c]; already && (ok != (claim != nil) || ok && cl.claim != *claim) {
		a.detach(c)
		already = false
	}
	user := c.UserID
	if claim != nil {
		u, ok := a.claimHolder(*claim)
		if !ok {
			return room.ErrBadClaim
		}
		user = u
	}
	wasConnected := a.connected(user)
	if !already {
		if len(a.conns) >= a.l.cfg.MaxClients {
			return room.ErrRoomFull
		}
		a.conns = append(a.conns, c)
		if claim != nil {
			a.claims[c] = claimed{claim: *claim, user: user}
		}
	}
	if !wasConnected && a.takeSlot(user, c.Name, claim != nil) {
		a.publish(nil, nil)
	} else {
		c.Out.Push(a.frame(c, nil))
		if !wasConnected {
			a.broadcastExcept(c)
		}
	}
	if !wasConnected {
		a.checkRoles()
	}
	a.armIdle()
	a.publishIdle()
	return nil
}

// takeSlot seats a user who just connected: in the slot reserved for
// them, or the first open one unless they left a slot before. A claim
// never renames its slot. It reports whether it committed a change.
func (a *actor) takeSlot(user, name string, claimed bool) bool {
	r := &a.rec
	i := slotOf(r, user, SlotReserved)
	switch {
	case i >= 0:
		if claimed || !validName(name) {
			name = r.Slots[i].Name
		}
	case slotOf(r, user, SlotHuman) >= 0 || slices.Contains(r.Unseated, user) || !validName(name):
		return false
	default:
		i = slices.IndexFunc(r.Slots, func(s Slot) bool { return s.Kind == SlotOpen })
		if i < 0 {
			return false
		}
	}
	cand := r.clone()
	cand.Revision++
	cand.Slots[i] = Slot{Kind: SlotHuman, UserID: user, Name: name, Taken: cand.Revision}
	retire(r, &cand)
	return a.commit("join", cand) == nil
}

func (a *actor) detach(c *room.Client) {
	i := slices.Index(a.conns, c)
	if i < 0 {
		return
	}
	user := a.userOf(c)
	a.conns = slices.Delete(a.conns, i, i+1)
	delete(a.claims, c)
	if !a.connected(user) {
		a.broadcastExcept(nil)
		a.checkRoles()
	}
	if len(a.conns) == 0 {
		a.emptySince = a.now()
	}
	a.armIdle()
	a.publishIdle()
}

// revokeClaims ends every connection whose claim no longer names a slot's
// user (after a removal or a release): error bad-claim, and detached. It
// returns whether a user was left with no connection.
func (a *actor) revokeClaims() bool {
	away := false
	for _, c := range slices.Clone(a.conns) {
		cl, ok := a.claims[c]
		if !ok {
			continue
		}
		if u, ok := a.claimHolder(cl.claim); ok && u == cl.user {
			continue
		}
		a.conns = slices.DeleteFunc(a.conns, func(x *room.Client) bool { return x == c })
		delete(a.claims, c)
		c.Out.Push(wire.NewError(wire.ErrorCodeBadClaim, "that device link no longer names a seat here"))
		away = away || !a.connected(cl.user)
	}
	if len(a.conns) == 0 {
		a.emptySince = a.now()
	}
	return away
}

// --- roles ------------------------------------------------------------------

// checkRoles applies the temporary-admin rule now, from the saved roles
// and any the store refused before, and persists what it changed,
// publicly for a role change and silently for an away time. When the
// store refuses the write, it checks again after roles.Retry.
func (a *actor) checkRoles() {
	cand := a.rec.clone()
	log := cand.Roles.Carry(a.rolesAhead, 0)
	more, next := cand.Roles.Check(a.now(), a.connected, a.candidates(&cand), 0)
	log = append(log, more...)
	if len(log) == 0 && sameAway(cand.Roles.Away, a.rec.Roles.Away) {
		a.rolesAhead = nil
		a.armRoles(next)
		return
	}
	if len(log) > 0 {
		appendLog(&cand, log...)
		cand.Revision++
		retire(&a.rec, &cand)
	}
	if err := a.commit("roles", cand); err != nil {
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
		a.publish(nil, nil)
	}
	a.armRoles(next)
}

// isAdmin reports whether u has the admin powers now (roles.Holds): a
// temporary admin whose demotion or replacement the store refused has
// none, nor does one whose grant it refused. Frames ask it for every
// connection and human, so only the temporary admin pays for the check.
func (a *actor) isAdmin(u string) bool {
	r := &a.rec
	if r.Roles.Real(u) || u == "" || r.Roles.Temporary != u {
		return r.Roles.Real(u)
	}
	return r.Roles.Holds(u, a.rolesAhead, a.now(), a.connected, a.candidates(r))
}

func sameAway(x, y []roles.Away) bool {
	return slices.EqualFunc(x, y, func(p, q roles.Away) bool { return p.UserID == q.UserID && p.Since.Equal(q.Since) })
}

// candidates are the connected humans in the temporary-admin rule's
// order: those in slots by how long they have held them (slot order
// breaks ties), then the others by how long they have been connected.
func (a *actor) candidates(r *Record) []string {
	type held struct {
		user  string
		taken int
	}
	var seated []held
	for _, s := range r.Slots {
		if s.Kind == SlotHuman && a.connected(s.UserID) {
			seated = append(seated, held{s.UserID, s.Taken})
		}
	}
	slices.SortStableFunc(seated, func(x, y held) int { return cmp.Compare(x.taken, y.taken) })
	out := []string{}
	for _, s := range seated {
		out = append(out, s.user)
	}
	for _, c := range a.conns {
		if u := a.userOf(c); !slices.Contains(out, u) {
			out = append(out, u)
		}
	}
	return out
}

func (a *actor) armRoles(at time.Time) {
	a.disarmRoles()
	if at.IsZero() {
		return
	}
	a.rolesStop = a.l.cfg.Clock.AfterFunc(max(at.Sub(a.now()), 0), func() {
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

func (a *actor) armIdle() {
	if len(a.conns) > 0 {
		a.disarmIdle()
		return
	}
	if a.idleStop != nil {
		return
	}
	a.idleStop = a.l.cfg.Clock.AfterFunc(a.l.cfg.IdleTimeout, func() {
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

func (a *actor) publishIdle() {
	idle := len(a.conns) == 0
	if idle == a.shownIdle && a.emptySince.Equal(a.shownSince) {
		return
	}
	a.shownIdle, a.shownSince = idle, a.emptySince
	a.m.PublishIdle(a.h, idle, a.emptySince)
}

// commit writes cand durably, then swaps it in. A failed write leaves the
// committed record as it was.
func (a *actor) commit(op string, cand Record) error {
	cand.Updated = a.now()
	if err := validate(&cand, a.l.cfg.SeqClients); err != nil {
		a.l.log.Error("lobby candidate invalid", "room", a.code, "op", op, "err", err)
		return err
	}
	data, err := cand.encode()
	if err == nil {
		err = a.l.store().SaveGame(a.code, Game, data)
	}
	if err != nil {
		a.l.log.Warn("lobby write failed", "room", a.code, "op", op, "err", err)
		return err
	}
	a.rec = cand
	return nil
}

// --- requests ---------------------------------------------------------------

// Outcomes, as the lobby answers and logs them.
const (
	outAccepted  = "accepted"
	outDuplicate = "duplicate"
	outForbidden = "forbidden"
	outNotAdmin  = "not-admin"
	outStorage   = "storage"
	outBad       = "bad-request"
	outStarted   = "started"
)

// inbound handles one request; true means Start handed the lobby over and
// the actor exits.
func (a *actor) inbound(c *room.Client, f wire.ClientMessage) bool {
	if !slices.Contains(a.conns, c) {
		c.Out.Push(wire.NewLobbyReject(seqOf(f), outBad))
		return false
	}
	switch f := f.(type) {
	case wire.LobbyEdit:
		a.edit(c, f)
	case wire.LobbyRole:
		a.role(c, f)
	case wire.LobbyStart:
		return a.start(c, f)
	case wire.Resync:
		c.Out.Push(a.frame(c, nil))
	default:
		// No device links in a lobby (a rematch hands out its reserved
		// slots' claims), and no rematch of a lobby.
		c.Out.Push(wire.NewLobbyReject(seqOf(f), outBad))
	}
	return false
}

func seqOf(f wire.ClientMessage) int {
	_, _, seq, _ := wire.PlatformTarget(f)
	return seq
}

func (a *actor) note(op, outcome string) string {
	a.l.log.Info("lobby", "room", a.code, "op", op, "rev", a.rec.Revision, "outcome", outcome)
	return outcome
}

func (a *actor) refuse(c *room.Client, seq int, op, outcome string) string {
	c.Out.Push(wire.NewLobbyReject(seq, outcome))
	return a.note(op, outcome)
}

// seen answers a seq at or below the client's remembered one with its
// current frame acknowledging it; nothing changes.
func (a *actor) seen(c *room.Client, seq int) bool {
	last, ok := seqLast(a.rec.Seqs, a.userOf(c), c.ClientID)
	if ok && seq <= last {
		c.Out.Push(a.frame(c, &wire.PlatformAck{ClientID: c.ClientID, Seq: seq}))
		a.note("seq", outDuplicate)
		return true
	}
	return false
}

// apply finishes an accepted request: it records the seq, advances the
// revision, brings the roles up to date, retires users who lost all
// authority, persists and publishes to everyone, the sender with an ack.
func (a *actor) apply(c *room.Client, op string, seq int, cand Record) string {
	seqPut(&cand.Seqs, SeqRecord{User: a.userOf(c), Client: c.ClientID, Seq: seq}, a.l.cfg.SeqClients)
	cand.Revision++
	log := cand.Roles.Carry(a.rolesAhead, 0)
	more, next := cand.Roles.Check(a.now(), a.connected, a.candidates(&cand), 0)
	appendLog(&cand, append(log, more...)...)
	retire(&a.rec, &cand)
	if err := a.commit(op, cand); err != nil {
		return a.refuse(c, seq, op, outStorage)
	}
	a.rolesAhead = nil
	a.armRoles(next)
	away := a.revokeClaims()
	a.publish(c, &wire.PlatformAck{ClientID: c.ClientID, Seq: seq})
	if away {
		a.checkRoles()
	}
	a.armIdle()
	a.publishIdle()
	return a.note(op, outAccepted)
}

// edit handles lobby-edit. Every operation but release is an admin's.
func (a *actor) edit(c *room.Client, f wire.LobbyEdit) string {
	if a.seen(c, f.Seq) {
		return outDuplicate
	}
	r := &a.rec
	user := a.userOf(c)
	if f.Op != "release" && !a.isAdmin(user) {
		return a.refuse(c, f.Seq, f.Op, outNotAdmin)
	}
	cand := r.clone()
	slot := func() (int, bool) {
		if f.Seat == nil || *f.Seat < 0 || *f.Seat >= r.SeatCount {
			return 0, false
		}
		return *f.Seat, true
	}
	bad := func() string { return a.refuse(c, f.Seq, f.Op, outBad) }
	switch f.Op {
	case "seats":
		if f.SeatCount == nil || *f.SeatCount < MinSeats || *f.SeatCount > MaxSeats || *f.SeatCount == r.SeatCount {
			return bad()
		}
		n := *f.SeatCount
		for _, s := range cand.Slots[min(n, len(cand.Slots)):] {
			if s.Kind == SlotHuman {
				unseat(&cand, s.UserID)
			}
		}
		if n < len(cand.Slots) {
			cand.Slots = cand.Slots[:n]
		}
		for len(cand.Slots) < n {
			cand.Slots = append(cand.Slots, Slot{Kind: SlotOpen})
		}
		cand.SeatCount = n
	case "options":
		if f.Options == nil || !validOptions(*f.Options) {
			return bad()
		}
		cand.Options = cloneOptions(*f.Options)
	case "add-bot":
		i, ok := slot()
		if !ok || f.Bot == nil || !validLevel(*f.Bot) || r.Slots[i].Kind != SlotOpen {
			return bad()
		}
		cand.Slots[i] = Slot{Kind: SlotBot, Bot: *f.Bot}
	case "remove-bot":
		i, ok := slot()
		if !ok || r.Slots[i].Kind != SlotBot {
			return bad()
		}
		cand.Slots[i] = Slot{Kind: SlotOpen}
	case "seat":
		i, ok := slot()
		if !ok || f.UserID == nil || r.Slots[i].Kind != SlotOpen || !a.connected(*f.UserID) ||
			slotOf(r, *f.UserID, SlotHuman) >= 0 || slotOf(r, *f.UserID, SlotReserved) >= 0 {
			return bad()
		}
		cand.Slots[i] = Slot{Kind: SlotHuman, UserID: *f.UserID, Name: a.nameOf(*f.UserID), Taken: cand.Revision + 1}
		cand.Unseated = slices.DeleteFunc(cand.Unseated, func(u string) bool { return u == *f.UserID })
	case "remove-human":
		if f.UserID == nil {
			return bad()
		}
		i := slotOf(r, *f.UserID, SlotHuman)
		if i < 0 {
			return bad()
		}
		cand.Slots[i] = Slot{Kind: SlotOpen}
		unseat(&cand, *f.UserID)
	case "release":
		i := slotOf(r, user, SlotHuman)
		if i < 0 {
			return a.refuse(c, f.Seq, f.Op, outForbidden)
		}
		cand.Slots[i] = Slot{Kind: SlotOpen}
		unseat(&cand, user)
	case "open-reservation":
		i, ok := slot()
		if !ok || r.Slots[i].Kind != SlotReserved {
			return bad()
		}
		cand.Slots[i] = Slot{Kind: SlotOpen}
	default:
		return bad()
	}
	return a.apply(c, f.Op, f.Seq, cand)
}

// role handles lobby-role, under the table's role policy (package roles).
func (a *actor) role(c *room.Client, f wire.LobbyRole) string {
	if a.seen(c, f.Seq) {
		return outDuplicate
	}
	cand := a.rec.clone()
	user := a.userOf(c)
	var entry wire.PlatformLogEntry
	var err error
	switch {
	case f.Op == "promote" && f.UserID != nil:
		inRoom := a.connected(*f.UserID) || slotOf(&a.rec, *f.UserID, SlotHuman) >= 0
		entry, err = cand.Roles.Promote(user, *f.UserID, inRoom, 0)
	case f.Op == "demote" && f.UserID != nil:
		entry, err = cand.Roles.Demote(user, *f.UserID, 0)
	case f.Op == "step-down" && f.UserID == nil:
		entry, err = cand.Roles.StepDown(user, 0)
	default:
		err = roles.ErrBadRequest
	}
	switch {
	case errors.Is(err, roles.ErrNotAdmin):
		return a.refuse(c, f.Seq, "role", outNotAdmin)
	case errors.Is(err, roles.ErrForbidden):
		return a.refuse(c, f.Seq, "role", outForbidden)
	case err != nil:
		return a.refuse(c, f.Seq, "role", outBad)
	}
	appendLog(&cand, entry)
	return a.apply(c, "role", f.Seq, cand)
}

// canStart reports whether u could Start now: a seated admin (real or
// temporary), three or more seats (two start Duel from M18.shared-lobby)
// and no reservation still waiting.
func (a *actor) canStart(u string) bool {
	r := &a.rec
	return a.isAdmin(u) && slotOf(r, u, SlotHuman) >= 0 && r.SeatCount >= tableroom.MinSeats &&
		!slices.ContainsFunc(r.Slots, func(s Slot) bool { return s.Kind == SlotReserved })
}

// start handles lobby-start. With every slot filled (fillBots fills the
// open ones with bots of the lowest level; a reservation must be opened
// first), it starts the game under this code: the table's record replaces
// the lobby's, the connections go over with their claims, and the table's
// actor takes the code. Each connection's next frame is the table's; the
// sender's carries no ack. Then this actor stops: a detach the Manager
// delivers to it later goes to the table.
func (a *actor) start(c *room.Client, f wire.LobbyStart) bool {
	if a.seen(c, f.Seq) {
		return false
	}
	r := &a.rec
	user := a.userOf(c)
	switch {
	case !a.isAdmin(user):
		a.refuse(c, f.Seq, "start", outNotAdmin)
		return false
	case !a.canStart(user) || !f.FillBots && slices.ContainsFunc(r.Slots, func(s Slot) bool { return s.Kind == SlotOpen }):
		a.refuse(c, f.Seq, "start", outForbidden)
		return false
	case a.l.cfg.Tables == nil:
		a.refuse(c, f.Seq, "start", outBad)
		return false
	}
	spec := tableroom.Spec{RandomSeating: r.Options.Seating == nil || *r.Options.Seating == "random",
		HideAtEnd: r.Options.FinalReveal != nil && !*r.Options.FinalReveal,
		AllowUndo: r.Options.AllowUndo == nil || *r.Options.AllowUndo, Creator: r.Roles.Creator}
	// The table starts from the roles as the rule last made them, saved
	// or not: the table's record, written at Start, saves them.
	rs := r.Roles.Clone()
	rs.Carry(a.rolesAhead, 0)
	spec.Roles = &rs
	for _, s := range r.Slots {
		switch s.Kind {
		case SlotHuman:
			spec.Participants = append(spec.Participants, tableroom.Participant{UserID: s.UserID, Name: s.Name})
		case SlotBot:
			spec.Participants = append(spec.Participants, tableroom.Participant{Bot: s.Bot})
		default:
			spec.Participants = append(spec.Participants, tableroom.Participant{Bot: tableroom.DefaultBotLevel})
		}
	}
	// Under mu, no detach can slip between the connections handed over
	// and the table's taking the later ones.
	a.mu.Lock()
	pending := a.detaches
	a.detaches = nil
	for _, d := range pending {
		a.conns = slices.DeleteFunc(a.conns, func(x *room.Client) bool { return x == d })
		delete(a.claims, d)
	}
	if !slices.Contains(a.conns, c) {
		a.mu.Unlock()
		return false
	}
	members := make([]tableroom.Member, 0, len(a.conns))
	for _, o := range a.conns {
		mb := tableroom.Member{Client: o}
		if cl, ok := a.claims[o]; ok {
			mb.Claim, mb.User = cl.claim, cl.user
		}
		members = append(members, mb)
	}
	started, err := a.l.cfg.Tables.StartTable(a.m, a.code, Game, spec, members)
	if err != nil {
		a.mu.Unlock()
		a.refuse(c, f.Seq, "start", outStorage)
		return false
	}
	a.next = started.Detach
	a.mu.Unlock()
	if err := a.m.Swap(a.h, started.Handle); err != nil {
		// The record is the table's now: the next join loads it.
		a.l.log.Error("lobby swap failed", "room", a.code, "err", err)
		started.Abort()
		a.m.Forget(a.h)
	}
	a.drain()
	a.note("start", outStarted)
	return true
}

// drain answers what is still queued once the lobby is gone: an attach
// is told the room is gone, so the Manager retries it on the table, and a
// lobby request is refused as forbidden.
func (a *actor) drain() {
	for {
		select {
		case msg := <-a.inbox:
			switch msg := msg.(type) {
			case msgAttach:
				msg.reply <- room.ErrRoomGone
			case room.Inbound:
				msg.Client.Out.Push(wire.NewLobbyReject(seqOf(msg.Frame), outForbidden))
			case msgCall:
				msg.fn()
				close(msg.reply)
			case msgEvict:
				msg.reply <- false
			}
		default:
			return
		}
	}
}
