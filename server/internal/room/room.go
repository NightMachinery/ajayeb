package room

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
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
	version    int          // bumped by every accepted act and undo; acts must carry it as base
	clients    map[*Client]member
	seqs       *seqLRU
	dirty      bool // the last Save failed; retried on the next change and before eviction
	emptySince time.Time
	idleStop   func() bool
	bot        botTask
	rematch    string // the code of the rematch room, once one exists
	dropped    int    // stale bot results dropped, for tests and diagnostics
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
	msgResync  struct{ c *Client }
	msgRematch struct {
		c     *Client
		reply chan rematchResult
	}
	msgBotResult struct {
		version int
		action  engine.Action
		err     error
	}
	msgIdle     struct{}
	msgSnapshot struct{ reply chan Snapshot }
)

type rematchResult struct {
	code string
	err  error
}

// newRoom builds a room around a file and the state it replays to. It does
// not start the actor (see Manager.startRoom).
func newRoom(m *Manager, f *store.File, s engine.State) *Room {
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
		version: len(f.Actions),
		clients: make(map[*Client]member),
		seqs:    newSeqLRU(m.cfg.SeqClients),
	}
	for id, seq := range f.LastSeq {
		r.seqs.put(id, seq)
	}
	return r
}

// run is the actor loop.
func (r *Room) run() {
	defer r.m.wg.Done()
	defer r.exit()
	r.emptySince = r.m.cfg.Clock.Now()
	r.armIdle()
	r.maybeStartBot()
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
	case msgRematch:
		code, err := r.doRematch(msg.c)
		msg.reply <- rematchResult{code: code, err: err}
	case msgBotResult:
		r.botResult(msg)
	case msgIdle:
		return r.idle()
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
		r.sendState(c, mem, nil)
		return nil
	}
	if len(r.clients) >= r.m.cfg.MaxClients {
		return ErrRoomFull
	}
	mem, claimed, err := r.admit(c)
	if err != nil {
		return err
	}
	r.clients[c] = mem
	r.stopIdle()
	if claimed {
		r.persist()
		r.maybeStartBot()
	}
	r.broadcast(nil, nil, nil)
	return nil
}

// admit decides what c is in this room, claiming open seats for it where
// the mode allows, and reports whether it claimed any.
func (r *Room) admit(c *Client) (member, bool, error) {
	var held []engine.Seat
	for i, s := range r.file.Seats {
		if s.Kind == store.SeatKindHuman && s.UserID == c.UserID {
			held = append(held, engine.Seat(i))
		}
	}
	switch len(held) {
	case 2:
		return member{role: roleHotseat}, false, nil
	case 1:
		return member{role: rolePlayer, seat: held[0]}, false, nil
	}

	open := r.openSeats()
	switch r.file.Mode {
	case store.ModeOnline:
		if len(open) == 0 {
			return member{role: roleSpectator}, false, nil
		}
		r.claim(open[0], c, c.Name)
		return member{role: rolePlayer, seat: open[0]}, true, nil
	case store.ModeHotseat:
		if len(open) != 2 {
			return member{}, false, ErrForbidden
		}
		r.claim(0, c, c.Name)
		r.claim(1, c, "")
		return member{role: roleHotseat}, true, nil
	case store.ModeAI:
		if len(open) != 1 {
			return member{}, false, ErrForbidden
		}
		r.claim(open[0], c, c.Name)
		return member{role: rolePlayer, seat: open[0]}, true, nil
	}
	return member{}, false, ErrForbidden
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
func (r *Room) claim(seat engine.Seat, c *Client, name string) {
	s := &r.file.Seats[seat]
	if s.Name != "" {
		name = s.Name
	}
	*s = store.Seat{Kind: store.SeatKindHuman, UserID: c.UserID, Name: name}
}

// detach removes c and tells the others about the presence change. The
// last connection to leave starts the idle countdown.
func (r *Room) detach(c *Client) {
	if _, ok := r.clients[c]; !ok {
		return
	}
	delete(r.clients, c)
	if len(r.clients) == 0 {
		r.emptySince = r.m.cfg.Clock.Now()
		r.armIdle()
		return
	}
	r.broadcast(nil, nil, nil)
}

// resync sends c a fresh state.
func (r *Room) resync(c *Client) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	r.sendState(c, mem, nil)
}

// --- actions ---------------------------------------------------------------

// act handles an act frame, in this order: the sender must be attached; a
// seq it already used only re-sends the state (with an ack, so its pending
// action resolves); the room must be playing; the sender must hold the seat
// to move; base must be the current version (else reject stale, then a
// fresh state); and the key must be in the engine's legal set. Then the
// engine's own action for that key is applied.
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
	if r.duplicate(c, m.Seq) {
		r.sendState(c, mem, ack)
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

// duplicate reports whether seq is not newer than the last seq seen from
// c's client id, and records it otherwise.
func (r *Room) duplicate(c *Client, seq int) bool {
	s := uint64(seq) // wire.DecodeClient rejects a negative seq
	if last, ok := r.seqs.get(c.ClientID); ok && s <= last {
		return true
	}
	r.seqs.put(c.ClientID, s)
	return false
}

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
	r.sendState(c, mem, nil)
}

// commit applies a legal action, records and persists it, sends every
// connection the new state with the action's events (the sender's frame
// carries ack), and starts the bot if it is now to move.
func (r *Room) commit(a engine.Action, sender *Client, ack *wire.Ack) {
	var ev []engine.Event
	if err := engine.Apply(&r.state, a, &ev); err != nil {
		// a came from LegalActions for this exact state, so this is a bug.
		r.log.Error("legal action failed to apply", "key", a.Key(), "err", err)
		return
	}
	r.file.Actions = append(r.file.Actions, a.Key())
	r.version++
	r.persist()
	r.broadcast(ev, sender, ack)
	r.maybeStartBot()
}

// undo takes back the previous human decision in a hot-seat or bot room:
// the log is cut to just before it and the game is replayed from the seed
// (so hidden cards cannot change). In a bot room this also removes the
// bot's answers after it, and a bot still thinking is cancelled; its result
// would be stale anyway. A finished game cannot be undone.
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
	if r.duplicate(c, m.Seq) {
		r.sendState(c, mem, ack)
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
	_, deciders, err := replayKeys(r.file, r.file.Actions)
	if err != nil {
		r.log.Error("replay for undo failed", "err", err)
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeIllegal, "the game cannot be replayed"))
		return
	}
	cut := -1
	for i := len(deciders) - 1; i >= 0; i-- {
		if r.file.Seats[deciders[i]].Kind == store.SeatKindHuman {
			cut = i
			break
		}
	}
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
	r.version++
	r.persist()
	r.broadcast(nil, c, ack)
	r.maybeStartBot()
}

// doRematch creates (once) a new room with the same mode and seats and a
// fresh deal, and tells every connection its code with a created frame.
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
		f := store.New("", r.file.Mode, r.file.Options, r.m.cfg.Seed(), r.m.cfg.Clock.Now())
		f.Seats = r.file.Seats
		code, err := r.m.createRoom(f)
		if err != nil {
			return "", err
		}
		r.rematch = code
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
	f.Updated = r.m.cfg.Clock.Now()
	if err := r.m.cfg.Store.Save(f); err != nil {
		r.dirty = true
		r.log.Error("saving the room failed", "err", err)
		return
	}
	r.dirty = false
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

// Snapshot is a copy of a room's state for tests, diagnostics and
// development endpoints. State holds the hidden information too: never send
// it to a client.
type Snapshot struct {
	Code              string
	Mode              store.Mode
	Status            store.Status
	Version           int
	Seats             [2]store.Seat
	Actions           []string
	State             engine.State
	Clients           int  // attached connections
	Spectators        int  // attached spectator connections
	BotThinking       bool // a bot search is running for the current version
	DroppedBotResults int  // stale bot results the room has dropped
	Rematch           string
	Dirty             bool // the last write failed
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
		Actions:           append([]string{}, r.file.Actions...),
		State:             r.state,
		Clients:           len(r.clients),
		Spectators:        spectators,
		BotThinking:       r.bot.cancel != nil && r.bot.version == r.version,
		DroppedBotResults: r.dropped,
		Rematch:           r.rematch,
		Dirty:             r.dirty,
	}
}
