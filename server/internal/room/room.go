package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// errRoomGone is what a request to a room whose actor has exited (it was
// evicted, or the Manager is closing) gets back. The Manager retries a join
// through a fresh load; it never reaches a client.
var errRoomGone = errors.New("room: the room actor has exited")

// role is what a connection is in a room.
type role uint8

const (
	roleSpectator role = iota // watches; online rooms only
	rolePlayer                // holds one seat (online and bot rooms)
	roleHotseat               // holds both seats (hot-seat rooms)
)

// member is one attached connection.
type member struct {
	role role
	seat engine.Seat // for rolePlayer
	user string      // the connection's user id
}

// Room is one game room: an actor goroutine that owns everything below the
// inbox. Other goroutines only send it messages (see the msg types) and
// never touch its fields.
type Room struct {
	code  string
	m     *Manager
	log   *slog.Logger
	inbox chan any
	done  chan struct{} // closed when the actor has exited

	ctx    context.Context // cancelled when the actor exits; bots run under it
	cancel context.CancelFunc

	// Owned by the actor goroutine.
	file       *store.File  // header, seats, action log; written after every change
	state      engine.State // file's seed and options replayed through file's actions
	hist       []histEntry  // by action of file.Actions: its ply, the seat that took it and its events
	version    int          // bumped by every accepted act and undo, stored in the file; acts must carry it as base
	clients    map[*Client]member
	seqs       *seqLRU // the last applied seq of each client id
	dirty      bool    // the last Save failed; retried on the next change and before eviction
	emptySince time.Time
	idleStop   func() bool
	bot        botTask
	botGen     uint64 // the generation of the last search started (see botTask)
	rematch    string // the code of the rematch room, once one exists (stored in the file)
	dropped    int    // stale bot results dropped, for tests and diagnostics
	shownIdle  bool   // what the actor last published as idle (see publishIdle)

	// Guarded by the Manager's mu: what the Manager knows about this room
	// when it must evict one to make room for another (see Manager.reserve).
	evictable bool      // no connection is attached and nothing is left to save
	idleSince time.Time // when the last connection left
	evicting  bool      // a Manager is asking the room to leave memory
}

// The messages a room's inbox carries.
type (
	msgAttach struct {
		c     *Client
		reply chan error
	}
	msgDetach struct{ c *Client }
	msgAct    struct {
		c   *Client
		act wire.Act
	}
	msgUndo struct {
		c    *Client
		undo wire.Undo
	}
	msgResync struct{ c *Client }
	msgHints  struct {
		c     *Client
		hints wire.Hints
	}
	msgRematch struct {
		c     *Client
		reply chan rematchResult
	}
	msgBotResult struct {
		gen     uint64 // the generation of the search that answers
		version int
		action  engine.Action
		err     error
	}
	msgIdle     struct{}
	msgEvict    struct{ reply chan bool }
	msgSnapshot struct{ reply chan Snapshot }
)

type rematchResult struct {
	code string
	err  error
}

// newRoom builds a room around a file, the state it replays to and the
// history of its actions (replayKeys). It does not start the actor (see
// Manager.start). The version continues from the file's, which is never
// below the length of the log.
func newRoom(m *Manager, f *store.File, s engine.State, hist []histEntry) *Room {
	fillHintFields(f)
	ctx, cancel := context.WithCancel(m.ctx)
	r := &Room{
		code:    f.Code,
		m:       m,
		log:     m.cfg.Logger.With("room", f.Code),
		inbox:   make(chan any, m.cfg.InboxSize),
		done:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
		file:    f,
		state:   s,
		hist:    hist,
		version: max(f.Version, len(f.Actions)),
		clients: make(map[*Client]member),
		seqs:    newSeqLRU(m.cfg.SeqClients),
		rematch: f.Rematch,
		// A new room has nobody attached; it is idle from now on, which is
		// what it reports to the Manager (publishIdle), possibly even
		// before its actor starts (createRoom saves it first).
		emptySince: m.cfg.Clock.Now(),
	}
	r.seqs.load(f.LastSeq)
	return r
}

// run is the actor loop. A room starts with no connection, so its bot (if
// any) waits for the first attach.
func (r *Room) run() {
	defer r.m.wg.Done()
	defer r.exit()
	r.armIdle()
	r.publishIdle()
	for {
		select {
		case <-r.ctx.Done():
			return
		case msg := <-r.inbox:
			if r.handle(msg) {
				return
			}
		}
	}
}

// exit stops whatever the room still has running and marks it gone.
func (r *Room) exit() {
	r.cancelBot()
	r.stopIdle()
	if r.dirty {
		r.persist()
	}
	r.cancel()
	close(r.done)
}

// handle processes one message and reports whether the actor must stop.
func (r *Room) handle(msg any) bool {
	switch msg := msg.(type) {
	case msgAttach:
		msg.reply <- r.attach(msg.c)
	case msgDetach:
		r.detach(msg.c)
	case msgAct:
		r.act(msg.c, msg.act)
	case msgUndo:
		r.undo(msg.c, msg.undo)
	case msgResync:
		r.resync(msg.c)
	case msgHints:
		r.setHints(msg.c, msg.hints)
	case msgRematch:
		code, err := r.doRematch(msg.c)
		msg.reply <- rematchResult{code: code, err: err}
	case msgBotResult:
		r.botResult(msg)
	case msgIdle:
		return r.idle()
	case msgEvict:
		evicted := r.evict()
		msg.reply <- evicted
		return evicted
	case msgSnapshot:
		msg.reply <- r.snapshot()
	}
	return false
}

// post delivers msg to the inbox. It fails with errRoomGone once the actor
// has exited, or with ctx's error.
func (r *Room) post(ctx context.Context, msg any) error {
	select {
	case r.inbox <- msg:
		return nil
	case <-r.done:
		return errRoomGone
	case <-ctx.Done():
		return ctx.Err()
	}
}

// request posts the message build makes around a reply channel and waits
// for the reply.
func request[T any](ctx context.Context, r *Room, build func(reply chan T) any) (T, error) {
	var zero T
	reply := make(chan T, 1)
	if err := r.post(ctx, build(reply)); err != nil {
		return zero, err
	}
	select {
	case v := <-reply:
		return v, nil
	case <-r.done:
		select {
		case v := <-reply:
			return v, nil
		default:
			return zero, errRoomGone
		}
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// --- connections -----------------------------------------------------------

// attach admits c, claiming a seat if it may, and sends every connection a
// fresh state (the newcomer its first one, the others the new presence).
func (r *Room) attach(c *Client) error {
	if mem, ok := r.clients[c]; ok {
		r.sendState(c, mem, nil, true)
		return nil
	}
	mem, claimed, err := r.admit(c)
	if err != nil {
		return err
	}
	mem.user = c.UserID
	r.clients[c] = mem
	r.stopIdle()
	r.publishIdle()
	if claimed {
		r.persist()
	}
	r.maybeStartBot()
	// The newcomer's first frame carries the whole history; the others
	// only learn the new presence.
	r.broadcast(nil, nil, nil, func(cl *Client) bool { return cl == c })
	return nil
}

// admit decides what c is in this room, claiming open seats for it where
// the mode allows, and reports whether it claimed any.
//
// Connections are bounded so that spectators can never lock a player out:
// a new spectator is refused once the room has MaxClients connections,
// while a user who holds a seat is refused only once that user alone has
// MaxClients connections here. A room thus holds at most three times
// MaxClients connections (spectators, and each seated user's tabs).
func (r *Room) admit(c *Client) (member, bool, error) {
	var held []engine.Seat
	for i, s := range r.file.Seats {
		if s.Kind == store.SeatKindHuman && s.UserID == c.UserID {
			held = append(held, engine.Seat(i))
		}
	}
	if len(held) > 0 && r.userConns(c.UserID) >= r.m.cfg.MaxClients {
		return member{}, false, ErrRoomFull
	}
	switch len(held) {
	case 2:
		return member{role: roleHotseat}, false, nil
	case 1:
		return member{role: rolePlayer, seat: held[0]}, false, nil
	}
	if closedTo(r.file.Mode, r.file.Seats, c.UserID) {
		return member{}, false, ErrForbidden
	}

	open := r.openSeats()
	switch r.file.Mode {
	case store.ModeOnline:
		if len(open) == 0 {
			if len(r.clients) >= r.m.cfg.MaxClients {
				return member{}, false, ErrRoomFull
			}
			return member{role: roleSpectator}, false, nil
		}
		r.claim(open[0], c, c.Name)
		return member{role: rolePlayer, seat: open[0]}, true, nil
	case store.ModeHotseat:
		r.claim(0, c, c.Name)
		r.claim(1, c, "")
		return member{role: roleHotseat}, true, nil
	default: // store.ModeAI, the only other mode closedTo lets through
		r.claim(open[0], c, c.Name)
		return member{role: rolePlayer, seat: open[0]}, true, nil
	}
}

// closedTo reports whether a room with this mode and these seats refuses
// userID outright: a hot-seat or bot room in which the user holds no seat
// and finds no seat to claim (both seats of a hot-seat room, the human seat
// of a bot room). The Manager also asks it of a room file's header, so that
// such a join is refused without loading the room (see Manager.get).
func closedTo(mode store.Mode, seats [2]store.Seat, userID string) bool {
	open := 0
	for _, s := range seats {
		switch {
		case s.Kind == store.SeatKindHuman && s.UserID == userID:
			return false
		case s.Kind == store.SeatKindOpen:
			open++
		}
	}
	switch mode {
	case store.ModeOnline:
		return false
	case store.ModeHotseat:
		return open != 2
	case store.ModeAI:
		return open != 1
	}
	return true
}

// userConns counts the connections of one user attached to the room.
func (r *Room) userConns(userID string) int {
	n := 0
	for c := range r.clients {
		if c.UserID == userID {
			n++
		}
	}
	return n
}

// openSeats lists the seats nobody holds yet, in seat order.
func (r *Room) openSeats() []engine.Seat {
	var open []engine.Seat
	for i, s := range r.file.Seats {
		if s.Kind == store.SeatKindOpen {
			open = append(open, engine.Seat(i))
		}
	}
	return open
}

// claim gives seat to c's user, keeping a name the room was created with.
// The first user to claim a seat of a room that has no admin yet (a room
// made with open seats) becomes its admin.
func (r *Room) claim(seat engine.Seat, c *Client, name string) {
	if r.file.Admin == "" {
		r.file.Admin = c.UserID
	}
	s := &r.file.Seats[seat]
	if s.Name != "" {
		name = s.Name
	}
	*s = store.Seat{Kind: store.SeatKindHuman, UserID: c.UserID, Name: name}
}

// detach removes c and tells the others about the presence change. The
// last connection to leave starts the idle countdown and cancels a running
// bot search: a bot thinks only while its human is there, so rooms left
// behind cannot keep the bot pool busy. The next attach starts it again.
func (r *Room) detach(c *Client) {
	if _, ok := r.clients[c]; !ok {
		return
	}
	delete(r.clients, c)
	if len(r.clients) == 0 {
		r.cancelBot()
		r.emptySince = r.m.cfg.Clock.Now()
		r.armIdle()
		r.publishIdle()
		return
	}
	r.broadcast(nil, nil, nil, noHistory)
}

// resync sends c a fresh state, with the history.
func (r *Room) resync(c *Client) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	r.sendState(c, mem, nil, true)
}

// --- actions ---------------------------------------------------------------

// act handles an act frame, in this order: the sender must be attached, and
// not as a spectator; a seq not newer than the last one applied for its
// client id only re-sends the state (with an ack, so its pending action
// resolves); the room must be playing; the sender must hold the seat to
// move; base must be the current version (else reject stale, then a fresh
// state); and the key must be in the engine's legal set. Then the engine's
// own action for that key is applied. Only an applied act uses up its seq:
// after a refusal the client may send the same seq again, and it is judged
// afresh. A resign:<seat> key is a forfeit, which takes the turn check's
// place with its own (see resign).
func (r *Room) act(c *Client, m wire.Act) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	if mem.role == roleSpectator {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotYourTurn, "spectators do not play"))
		return
	}
	ack := &wire.Ack{ClientID: c.ClientID, Seq: m.Seq}
	if r.applied(c, m.Seq) {
		r.sendState(c, mem, ack, false)
		return
	}
	if r.state.Over() {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "the game is over"))
		return
	}
	if r.status() != store.StatusPlaying {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotYourTurn, "the room is waiting for its players"))
		return
	}
	if a, ok := resignByKey(&r.state, m.Key); ok {
		r.resign(c, mem, m, a, ack)
		return
	}
	if !r.mayAct(mem, m.Seat) {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotYourTurn, "this connection does not hold the seat to move"))
		return
	}
	if m.Base != r.version {
		r.rejectStale(c, mem, m.Seq)
		return
	}
	a, ok := legalByKey(&r.state, m.Key)
	if !ok {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "that action is not legal now"))
		return
	}
	r.commit(a, c, ack)
}

// resign applies a forfeit, an act whose key is resign:<seat>, after act's
// checks up to the room's status. A player forfeits only their own seat: in
// an online or bot room the seat the connection holds, whoever is to move;
// in a hot-seat room the seat to move, since both players share the device.
// The act's optional seat, when present, must be that same seat. Then base
// must be the current version, as for any act. A forfeit ends the game, so
// a bot still searching is cancelled, and it cannot be undone.
func (r *Room) resign(c *Client, mem member, m wire.Act, a engine.Action, ack *wire.Ack) {
	seat := a.Starter
	var mine bool
	switch mem.role {
	case roleHotseat:
		mine = seat == r.state.ToMove
	case rolePlayer:
		mine = seat == mem.seat
	}
	if !mine || (m.Seat != nil && *m.Seat != int(seat)) {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotYourTurn, "a player may only forfeit their own seat"))
		return
	}
	if m.Base != r.version {
		r.rejectStale(c, mem, m.Seq)
		return
	}
	r.cancelBot()
	r.commit(a, c, ack)
}

// applied reports whether seq is not newer than the last seq applied for
// c's user and client id. Such a request is only acknowledged: it was
// applied already, or (an older seq that was refused) its client has moved
// on. Seqs are recorded only when a request is applied (see record), so a
// refused request does not use up its seq.
func (r *Room) applied(c *Client, seq int) bool {
	last, ok := r.seqs.get(seqKey(c))
	return ok && seq <= last
}

// record notes ack's seq as the last applied seq of the sender's user and
// client id. A bot's move (no sender) records nothing.
func (r *Room) record(sender *Client, ack *wire.Ack) {
	if sender != nil && ack != nil {
		r.seqs.put(seqKey(sender), ack.Seq)
	}
}

// seqKey is the key of a client's seqs: its user id and its client id. The
// client id alone is not enough, because every client picks its own: another
// user who chose (or learned) the same one could otherwise send a huge seq
// and make the room take this client's next requests for old ones. User ids
// are hex (package ident), so the colon keeps keys of different users apart.
func seqKey(c *Client) string { return c.UserID + ":" + c.ClientID }

// mayAct reports whether the member may act for the seat to move. seat is
// the act frame's optional seat, which must then be that seat.
func (r *Room) mayAct(mem member, seat *int) bool {
	toMove := r.state.ToMove
	if seat != nil && *seat != int(toMove) {
		return false
	}
	switch mem.role {
	case roleHotseat:
		return true
	case rolePlayer:
		return mem.seat == toMove
	}
	return false
}

// rejectStale refuses a request chosen against an old version and follows
// up with the current state.
func (r *Room) rejectStale(c *Client, mem member, seq int) {
	c.Out.Push(wire.NewReject(seq, wire.RejectCodeStale, "base is not the current version"))
	r.sendState(c, mem, nil, false)
}

// commit applies a legal action (or a forfeit), records it in the log and
// the history (and the sender's seq) and persists it, starts the bot if it
// is now to move, and sends every connection the new state with the
// action's events (the sender's frame carries ack). The bot starts first so
// that the frames already say it is thinking.
func (r *Room) commit(a engine.Action, sender *Client, ack *wire.Ack) {
	e := histEntry{ply: r.state.Ply, seat: actor(&r.state, a)}
	if err := engine.Apply(&r.state, a, &e.events); err != nil {
		// a came from LegalActions (or resignByKey) for this exact state, so
		// this is a bug.
		r.log.Error("legal action failed to apply", "key", a.Key(), "err", err)
		return
	}
	r.file.Actions = append(r.file.Actions, a.Key())
	r.hist = append(r.hist, e)
	r.version++
	r.record(sender, ack)
	r.persist()
	r.maybeStartBot()
	r.broadcast(e.events, sender, ack, noHistory)
}

// undo takes back the previous human decision in a hot-seat or bot room:
// the log is cut to just before it and the game is replayed from the seed
// (so hidden cards cannot change). In a bot room this also removes the
// bot's answers after it, and a bot still thinking is cancelled; its result
// would be stale anyway. A finished game cannot be undone, so neither can a
// forfeit. Every connection then receives a frame with the shortened
// history.
func (r *Room) undo(c *Client, m wire.Undo) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	if r.file.Mode == store.ModeOnline || mem.role == roleSpectator {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeBadRequest, "undo is only for hot-seat and bot rooms"))
		return
	}
	ack := &wire.Ack{ClientID: c.ClientID, Seq: m.Seq}
	if r.applied(c, m.Seq) {
		r.sendState(c, mem, ack, false)
		return
	}
	if r.state.Over() {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "the game is over"))
		return
	}
	if m.Base != r.version {
		r.rejectStale(c, mem, m.Seq)
		return
	}
	cut := r.lastHumanDecision()
	if cut < 0 {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "there is nothing to undo"))
		return
	}
	keys := append([]string{}, r.file.Actions[:cut]...)
	s, _, err := replayKeys(r.file, keys)
	if err != nil {
		r.log.Error("replay for undo failed", "err", err)
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "the game cannot be replayed"))
		return
	}
	r.cancelBot()
	r.state = s
	r.file.Actions = keys
	r.hist = r.hist[:cut]
	r.clampRoomLog()
	r.version++
	r.record(c, ack)
	r.persist()
	r.maybeStartBot()
	// Every connection has seen the moves taken back, so every frame
	// carries the shortened history.
	r.broadcast(nil, c, ack, allHistory)
}

// lastHumanDecision returns the index in the log of the last action a human
// seat decided, or -1 when there is none.
func (r *Room) lastHumanDecision() int {
	for i := len(r.hist) - 1; i >= 0; i-- {
		if r.file.Seats[r.hist[i].seat].Kind == store.SeatKindHuman {
			return i
		}
	}
	return -1
}

// canUndo reports whether an undo from mem, sent against the current
// version, would be accepted: a hot-seat or bot room, a player, a game that
// is not over, and a human decision to take back.
func (r *Room) canUndo(mem member) bool {
	return r.file.Mode != store.ModeOnline && mem.role != roleSpectator &&
		!r.state.Over() && r.lastHumanDecision() >= 0
}

// doRematch creates (once) a new room with the same mode and seats and a
// fresh deal, and tells every connection its code with a created frame. The
// code is stored with this room, so a later request, also after the room
// was evicted and reloaded, leads to the same rematch room.
func (r *Room) doRematch(c *Client) (string, error) {
	mem, ok := r.clients[c]
	if !ok {
		return "", ErrNotJoined
	}
	if mem.role == roleSpectator {
		return "", ErrForbidden
	}
	if !r.state.Over() {
		return "", ErrNotOver
	}
	if r.rematch == "" {
		// Creating a room counts against the requester's creation limits,
		// or rematches would be a way around them.
		if c.AllowCreate != nil && !c.AllowCreate() {
			return "", ErrRateLimited
		}
		f := store.New("", r.file.Mode, r.file.Options, r.m.cfg.Seed(), r.m.cfg.Clock.Now())
		f.Seats = r.file.Seats
		// The rematch keeps the game's admin and its hint settings.
		f.Admin = r.file.Admin
		hints := *r.file.Hints
		f.Hints = &hints
		code, err := r.m.createRoom(f)
		if err != nil {
			return "", err
		}
		r.rematch = code
		r.persist()
		r.log.Info("rematch", "code", code)
	}
	for cl := range r.clients {
		cl.Out.Push(wire.NewCreated(r.rematch))
	}
	return r.rematch, nil
}

// --- persistence and idleness -----------------------------------------------

// status is the room's lifecycle state, derived from the game and the
// seats.
func (r *Room) status() store.Status {
	if r.state.Over() {
		return store.StatusFinished
	}
	if len(r.openSeats()) > 0 {
		return store.StatusLobby
	}
	return store.StatusPlaying
}

// persist writes the room file. A failure is logged and leaves the room
// dirty: it keeps running from memory, the next change retries, and it is
// not evicted until a write succeeds.
func (r *Room) persist() {
	f := r.file
	f.Status = r.status()
	f.Result = nil
	if r.state.Over() {
		res := r.state.Result
		f.Result = &res
	}
	f.LastSeq = r.seqs.snapshot()
	f.Version = r.version
	f.Rematch = r.rematch
	f.Updated = r.m.cfg.Clock.Now()
	if err := r.m.cfg.Store.Save(f); err != nil {
		r.dirty = true
		r.publishIdle()
		r.log.Error("saving the room failed", "err", err)
		return
	}
	r.dirty = false
	r.publishIdle()
	if got := engine.HashString(&r.state); got != f.Hash {
		r.log.Error("the saved log does not replay to the room's state", "want", got, "file", f.Hash)
	}
}

// armIdle starts the idle countdown: when it runs out the room checks
// whether it may be evicted.
func (r *Room) armIdle() {
	r.stopIdle()
	inbox, done := r.inbox, r.done
	r.idleStop = r.m.cfg.Clock.AfterFunc(r.m.cfg.IdleTimeout, func() {
		select {
		case inbox <- msgIdle{}:
		case <-done:
		}
	})
}

// stopIdle cancels the idle countdown, if one is running.
func (r *Room) stopIdle() {
	if r.idleStop != nil {
		r.idleStop()
		r.idleStop = nil
	}
}

// idle evicts the room if it has had no connection for the idle timeout,
// and reports whether it did. The file is already on disk (a dirty room is
// saved first, and stays if that fails), so the next join reloads it.
func (r *Room) idle() bool {
	if len(r.clients) > 0 {
		return false
	}
	if r.m.cfg.Clock.Now().Sub(r.emptySince) < r.m.cfg.IdleTimeout {
		return false
	}
	if r.dirty {
		r.persist()
		if r.dirty {
			r.armIdle()
			return false
		}
	}
	r.m.remove(r)
	r.log.Info("evicted idle room")
	return true
}

// evict leaves memory at once, if the room is still idle, because the
// Manager needs its slot for another room; it reports whether it did. A
// connection may have attached since the Manager chose this room, and a
// dirty room must stay until a write succeeds.
func (r *Room) evict() bool {
	if len(r.clients) > 0 || r.dirty {
		return false
	}
	r.m.remove(r)
	r.log.Info("evicted to make room for another")
	return true
}

// publishIdle tells the Manager whether the room may be evicted to make room
// for another: no connection attached and nothing left to save. It only
// takes the Manager's lock when that changes.
func (r *Room) publishIdle() {
	idle := len(r.clients) == 0 && !r.dirty
	if idle == r.shownIdle {
		return
	}
	r.shownIdle = idle
	r.m.setIdle(r, idle, r.emptySince)
}

// Snapshot is a copy of a room's state for tests, diagnostics and
// development endpoints. State holds the hidden information too: never send
// it to a client.
type Snapshot struct {
	Code              string
	Mode              store.Mode
	Status            store.Status
	Version           int
	Seats             [2]store.Seat
	Scenario          string // the named scenario the game started from; empty for an ordinary game
	Actions           []string
	History           int // entries in the room's history: one per action of Actions
	State             engine.State
	Clients           int  // attached connections
	Spectators        int  // attached spectator connections
	BotThinking       bool // a bot search is running for the current version
	DroppedBotResults int  // stale bot results the room has dropped
	Rematch           string
	Dirty             bool // the last write failed
	Admin             string
	Hints             view.HintSettings
	RoomLog           []store.LogEntry
}

func (r *Room) snapshot() Snapshot {
	spectators := 0
	for _, mem := range r.clients {
		if mem.role == roleSpectator {
			spectators++
		}
	}
	return Snapshot{
		Code:              r.code,
		Mode:              r.file.Mode,
		Status:            r.status(),
		Version:           r.version,
		Seats:             r.file.Seats,
		Scenario:          r.file.Scenario,
		Actions:           append([]string{}, r.file.Actions...),
		History:           len(r.hist),
		State:             r.state,
		Clients:           len(r.clients),
		Spectators:        spectators,
		BotThinking:       r.thinking() != nil,
		DroppedBotResults: r.dropped,
		Rematch:           r.rematch,
		Dirty:             r.dirty,
		Admin:             r.file.Admin,
		Hints:             *r.file.Hints,
		RoomLog:           append([]store.LogEntry{}, r.file.RoomLog...),
	}
}
