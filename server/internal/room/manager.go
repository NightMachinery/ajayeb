package room

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Manager is the registry of the rooms in memory: it creates rooms, loads
// them from the store on demand, routes each client request to its room's
// actor, and stops every room on Close. Its methods are safe for
// concurrent use, except that calls for one Client must not overlap (see
// Client).
//
// Every request's outcome reaches the client as frames in its outbox:
// created and state frames on success, a reject or an error frame on
// failure. The error a method returns is for the transport's logging and
// for deciding whether to keep the connection; the client has already been
// told.
type Manager struct {
	cfg    Config
	log    *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup // room actors and bot searches

	mu      sync.Mutex
	rooms   map[string]*Room
	pending int // rooms being created or loaded, counted against MaxRooms
	closed  bool
}

// NewManager returns a Manager. cfg.Store is required.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Store == nil {
		return nil, errors.New("room: Config.Store is required")
	}
	cfg = cfg.withDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cfg:    cfg,
		log:    cfg.Logger,
		ctx:    ctx,
		cancel: cancel,
		rooms:  make(map[string]*Room),
	}, nil
}

// Close stops every room and bot search and waits for them. Rooms are
// already on disk (a room whose last write failed tries once more). After
// Close every method fails with ErrClosed.
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	m.cancel()
	m.wg.Wait()
	m.mu.Lock()
	clear(m.rooms)
	m.mu.Unlock()
}

// Len returns the number of rooms in memory.
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rooms)
}

// Loaded reports whether the room is in memory.
func (m *Manager) Loaded(code string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rooms[code] != nil
}

// --- requests ----------------------------------------------------------------

// Create makes a room for req, owned by c, and joins c to it: c receives a
// created frame and then its first state. In an online room c takes the
// seat req chooses (the other stays open until someone joins); in a
// hot-seat room c holds both seats; in a bot room c takes the chosen seat
// (seat 0 unless req says otherwise) and the bot the other.
func (m *Manager) Create(ctx context.Context, c *Client, req wire.Create) (string, error) {
	f, err := m.newFile(c, req)
	if err == nil {
		var code string
		code, err = m.createRoom(f)
		if err == nil {
			c.Out.Push(wire.NewCreated(code))
			return code, m.Join(ctx, c, code)
		}
	}
	c.Out.Push(wire.NewError(errorCode(err), errorMsg(err)))
	return "", err
}

// newFile builds the room file a create frame asks for.
func (m *Manager) newFile(c *Client, req wire.Create) (*store.File, error) {
	name := func(i int, fallback string) string {
		if i < len(req.Names) && req.Names[i] != "" {
			return req.Names[i]
		}
		return fallback
	}
	seat, ok := req.Seat.Seat()
	if !ok {
		seat = int(m.cfg.Seed() & 1)
	}
	if seat != 0 && seat != 1 {
		return nil, fmt.Errorf("%w: seat %d", ErrInvalidSpec, seat)
	}
	human := func(name string) store.Seat {
		return store.Seat{Kind: store.SeatKindHuman, UserID: c.UserID, Name: name}
	}

	f := store.New("", store.Mode(req.Mode), engine.Options{}, m.cfg.Seed(), m.cfg.Clock.Now())
	// The creator is the game's admin, whatever seat they took, and the
	// game starts from the hint settings they sent (or the defaults).
	f.Admin = c.UserID
	if req.Hints != nil {
		h := *req.Hints
		f.Hints = &h
	}
	switch req.Mode {
	case wire.ModeOnline:
		f.Seats[seat] = human(name(seat, c.Name))
		f.Seats[1-seat] = store.Seat{Kind: store.SeatKindOpen}
	case wire.ModeHotseat:
		f.Seats[0] = human(name(0, c.Name))
		f.Seats[1] = human(name(1, ""))
	case wire.ModeAI:
		if req.Bot == nil {
			return nil, fmt.Errorf("%w: a bot room needs a bot level", ErrInvalidSpec)
		}
		f.Seats[seat] = human(name(0, c.Name))
		f.Seats[1-seat] = store.Seat{Kind: store.SeatKindBot, Bot: store.BotLevel(*req.Bot)}
	default:
		return nil, fmt.Errorf("%w: unknown mode %q", ErrInvalidSpec, req.Mode)
	}
	return f, nil
}

// Spec describes a room made directly (by a development endpoint, a tool
// or a test) rather than through a create frame.
type Spec struct {
	Mode store.Mode
	// Seats are the seats as the room starts. An open seat is claimed by
	// the first user to join: in a hot-seat room both seats must be open or
	// held by one user, and the first joiner takes both.
	Seats   [2]store.Seat
	Seed    uint64
	Options engine.Options
	// Scenario names a named scenario (engine.NamedScenario) to start
	// from; empty for an ordinary deal.
	Scenario string
	// Actions are applied first, each checked against the legal set.
	Actions []string
	// Hints are the game's hint settings; nil means the defaults of the
	// mode. The room's admin is the first user to claim a seat.
	Hints *view.HintSettings
}

// CreateFromSpec makes a room from spec and returns its code. Nobody is
// attached; the room is evicted after the idle timeout unless someone
// joins.
func (m *Manager) CreateFromSpec(spec Spec) (string, error) {
	f := store.New("", spec.Mode, spec.Options, spec.Seed, m.cfg.Clock.Now())
	f.Seats = spec.Seats
	f.Scenario = spec.Scenario
	f.Actions = append(f.Actions, spec.Actions...)
	if spec.Hints != nil {
		h := *spec.Hints
		f.Hints = &h
	}
	return m.createRoom(f)
}

// createRoom checks f's seats and game, gives it a fresh code, saves it and
// starts its actor.
func (m *Manager) createRoom(f *store.File) (string, error) {
	if err := checkSeats(f.Mode, f.Seats); err != nil {
		return "", err
	}
	s, hist, err := replayKeys(f, f.Actions)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}
	f.Version = max(f.Version, len(f.Actions))
	if err := m.reserve(); err != nil {
		return "", err
	}
	defer m.unreserve()

	const attempts = 3
	for range attempts {
		code, err := m.cfg.Store.NewCode()
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrStorage, err)
		}
		if m.Loaded(code) {
			continue
		}
		f.Code = code
		r := newRoom(m, f, s, hist)
		r.persist()
		if r.dirty {
			r.cancel()
			return "", fmt.Errorf("%w: saving the new room failed", ErrStorage)
		}
		mode := f.Mode // f belongs to the actor once it starts
		if err := m.start(r); err != nil {
			r.cancel()
			if errors.Is(err, errCodeTaken) {
				continue
			}
			return "", err
		}
		r.log.Info("created room", "mode", mode)
		return code, nil
	}
	return "", fmt.Errorf("%w: no free room code", ErrStorage)
}

var errCodeTaken = errors.New("room: code already in memory")

// checkSeats checks that seats fit mode: an online room has two human or
// open seats held by different users; a hot-seat room has both seats open
// or both held by one user; a bot room has one bot seat of a known level
// and one human or open seat.
func checkSeats(mode store.Mode, seats [2]store.Seat) error {
	bots, open := 0, 0
	for _, s := range seats {
		switch s.Kind {
		case store.SeatKindOpen:
			open++
		case store.SeatKindBot:
			bots++
			switch s.Bot {
			case store.BotEasy, store.BotNormal, store.BotHard:
			default:
				return fmt.Errorf("%w: unknown bot level %q", ErrInvalidSpec, s.Bot)
			}
		case store.SeatKindHuman:
			if s.UserID == "" {
				return fmt.Errorf("%w: a human seat without a user", ErrInvalidSpec)
			}
		default:
			return fmt.Errorf("%w: unknown seat kind %q", ErrInvalidSpec, s.Kind)
		}
	}
	sameUser := seats[0].Kind == store.SeatKindHuman && seats[1].Kind == store.SeatKindHuman &&
		seats[0].UserID == seats[1].UserID
	var ok bool
	switch mode {
	case store.ModeOnline:
		ok = bots == 0 && !sameUser
	case store.ModeHotseat:
		ok = bots == 0 && (open == 2 || sameUser)
	case store.ModeAI:
		ok = bots == 1
	default:
		return fmt.Errorf("%w: unknown mode %q", ErrInvalidSpec, mode)
	}
	if !ok {
		return fmt.Errorf("%w: seats %v do not fit a %s room", ErrInvalidSpec, seats, mode)
	}
	return nil
}

// reserve counts a room about to be created or loaded against MaxRooms.
// When the Manager is full, it first evicts the room that has been idle
// longest (no connection attached, nothing left to save: it is on disk and
// comes back on the next join), so rooms nobody uses can never lock anyone
// out. Only when every room in memory has a connection is the answer
// ErrTooManyRooms.
func (m *Manager) reserve() error {
	// Each round either reserves, gives up, or evicts one room. A room may
	// refuse (a connection attached after it was chosen), so the rounds are
	// bounded rather than looping on a busy server.
	const rounds = 8
	for range rounds {
		victim, err := m.reserveOrChoose()
		if err != nil || victim == nil {
			return err
		}
		m.evict(victim)
	}
	return ErrTooManyRooms
}

// reserveOrChoose reserves a slot if one is free. Otherwise it returns the
// room idle longest, marked as being evicted so that no other caller picks
// it too, or ErrTooManyRooms when no room is idle.
func (m *Manager) reserveOrChoose() (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	if len(m.rooms)+m.pending < m.cfg.MaxRooms {
		m.pending++
		return nil, nil
	}
	var victim *Room
	for _, r := range m.rooms {
		if !r.evictable || r.evicting {
			continue
		}
		if victim == nil || r.idleSince.Before(victim.idleSince) ||
			(r.idleSince.Equal(victim.idleSince) && r.code < victim.code) {
			victim = r
		}
	}
	if victim == nil {
		return nil, ErrTooManyRooms
	}
	victim.evicting = true
	return victim, nil
}

// evict asks a room chosen by reserveOrChoose to leave memory, and waits
// until it has answered (it leaves only if it is still idle) or is gone.
func (m *Manager) evict(r *Room) {
	_, _ = request(m.ctx, r, func(reply chan bool) any { return msgEvict{reply: reply} })
	m.mu.Lock()
	r.evicting = false
	m.mu.Unlock()
}

// setIdle records what a room's actor publishes about itself: whether it
// may be evicted, and since when nobody is attached.
func (m *Manager) setIdle(r *Room, idle bool, since time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.evictable = idle
	r.idleSince = since
}

func (m *Manager) unreserve() {
	m.mu.Lock()
	m.pending--
	m.mu.Unlock()
}

// start registers r and starts its actor, unless its code is taken.
func (m *Manager) start(r *Room) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	if m.rooms[r.code] != nil {
		return errCodeTaken
	}
	m.rooms[r.code] = r
	m.wg.Add(1)
	go r.run()
	return nil
}

// remove unregisters r, if it is still the room registered for its code.
func (m *Manager) remove(r *Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rooms[r.code] == r {
		delete(m.rooms, r.code)
	}
}

// lookup returns the room in memory for code, or nil.
func (m *Manager) lookup(code string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, ErrClosed
	}
	return m.rooms[code], nil
}

// get returns the room for code, loading it from the store if it is not in
// memory. Before a load, check (when set) sees the room file's header and
// may refuse with its own error; the room is then not loaded at all, so a
// join the room would refuse costs a read of the file, never a slot among
// the rooms in memory.
func (m *Manager) get(code string, check func(*store.File) error) (*Room, error) {
	if store.ValidateCode(code) != nil {
		return nil, ErrRoomNotFound
	}
	if r, err := m.lookup(code); r != nil || err != nil {
		return r, err
	}
	if check != nil {
		hdr, err := m.cfg.Store.PeekHeader(code)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				m.log.Warn("reading a room's header failed", "room", code, "err", err)
			}
			return nil, ErrRoomNotFound
		}
		if hdr.Status == store.StatusArchived {
			return nil, ErrRoomNotFound
		}
		if err := check(hdr); err != nil {
			return nil, err
		}
	}
	if err := m.reserve(); err != nil {
		return nil, err
	}
	defer m.unreserve()
	f, s, err := m.cfg.Store.Load(code)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			m.log.Warn("loading a room failed", "room", code, "err", err)
		}
		return nil, ErrRoomNotFound
	}
	if f.Status == store.StatusArchived {
		return nil, ErrRoomNotFound
	}
	// The store has checked that the log replays to its hash; the room also
	// needs each action's seat and events, for undo and the history.
	_, hist, err := replayKeys(f, f.Actions)
	if err != nil {
		m.log.Error("a stored room does not replay through the legal actions", "room", code, "err", err)
		return nil, ErrRoomNotFound
	}
	r := newRoom(m, f, s, hist)
	actions := len(f.Actions) // f belongs to the actor once it starts
	switch err := m.start(r); {
	case errors.Is(err, errCodeTaken):
		r.cancel()
		return m.lookup(code) // someone else loaded it meanwhile
	case err != nil:
		r.cancel()
		return nil, err
	}
	r.log.Info("loaded room", "actions", actions)
	return r, nil
}

// Join attaches c to the room with this code, loading it if needed, and
// sends c its state. A user who holds a seat gets it back; in an online
// room a new user takes an open seat, or watches as a spectator when both
// are taken. A hot-seat or bot room admits only its owner. Joining another
// room first leaves the one c was in.
func (m *Manager) Join(ctx context.Context, c *Client, code string) error {
	return m.JoinClaim(ctx, c, code, nil)
}

// JoinClaim is Join with a join frame's optional claim (a device link,
// package ident): with a valid claim c acts in this room, and only here, as
// the user of the seat the claim was made for. A claim that is no current
// claim of the room's seats fails with ErrBadClaim (error frame bad-claim).
func (m *Manager) JoinClaim(ctx context.Context, c *Client, code string, claim *string) error {
	if c.joined != "" && c.joined != code {
		m.Leave(c, c.joined)
	}
	err := m.join(ctx, c, code, claim)
	if err != nil {
		if !errors.Is(err, ErrClosed) && !isContextErr(err) {
			c.Out.Push(wire.NewError(errorCode(err), errorMsg(err)))
		}
		return err
	}
	c.joined = code
	return nil
}

func (m *Manager) join(ctx context.Context, c *Client, code string, claim *string) error {
	// A room this user could never enter, or a claim that is not one of the
	// room's, is refused from its file's header without loading it (see
	// get); a room in memory decides in admit.
	check := func(f *store.File) error {
		user := c.UserID
		if claim != nil {
			u, ok := claimHolder(m.cfg.Claims, code, f.Seats, *claim)
			if !ok {
				return ErrBadClaim
			}
			user = u
		}
		if closedTo(f.Mode, f.Seats, user) {
			return ErrForbidden
		}
		return nil
	}
	// A room can exit (idle eviction) between the lookup and the attach;
	// the retry then loads it again from disk.
	const attempts = 3
	for range attempts {
		r, err := m.get(code, check)
		if err != nil {
			return err
		}
		if r == nil {
			continue
		}
		res, err := request(ctx, r, func(reply chan error) any { return msgAttach{c: c, claim: claim, reply: reply} })
		if errors.Is(err, errRoomGone) {
			continue
		}
		if err != nil {
			// The caller stopped waiting, but the attach may already be in
			// the inbox. Nothing would ever detach it (the client does not
			// record the room), so the room would report a ghost as present
			// and never go idle. A detach queued behind it takes it back; for
			// a client that never got in, it does nothing.
			_ = r.post(context.Background(), msgDetach{c: c})
			return err
		}
		return res
	}
	return ErrRoomNotFound
}

// Leave detaches c from the room it joined; the seats stay with their
// users, so they can come back. code must be the room c joined.
func (m *Manager) Leave(c *Client, code string) {
	if c.joined == "" || c.joined != code {
		return
	}
	c.joined = ""
	if r, _ := m.lookup(code); r != nil {
		_ = r.post(context.Background(), msgDetach{c: c})
	}
}

// Disconnect detaches c from whatever room it joined; the transport calls
// it when the connection closes.
func (m *Manager) Disconnect(c *Client) { m.Leave(c, c.joined) }

// joinedRoom returns the room c joined, if its code is code.
func (m *Manager) joinedRoom(c *Client, code string) (*Room, error) {
	if c.joined == "" || c.joined != code {
		return nil, ErrNotJoined
	}
	r, err := m.lookup(code)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrNotJoined
	}
	return r, nil
}

// Act routes an act frame to c's room.
func (m *Manager) Act(ctx context.Context, c *Client, act wire.Act) error {
	r, err := m.joinedRoom(c, act.Code)
	if err == nil {
		err = r.post(ctx, msgAct{c: c, act: act})
	}
	return m.requestFailed(c, act.Seq, err)
}

// Undo routes an undo frame to c's room.
func (m *Manager) Undo(ctx context.Context, c *Client, undo wire.Undo) error {
	r, err := m.joinedRoom(c, undo.Code)
	if err == nil {
		err = r.post(ctx, msgUndo{c: c, undo: undo})
	}
	return m.requestFailed(c, undo.Seq, err)
}

// Resync asks c's room for a fresh state frame.
func (m *Manager) Resync(ctx context.Context, c *Client, code string) error {
	r, err := m.joinedRoom(c, code)
	if err == nil {
		err = r.post(ctx, msgResync{c: c})
	}
	return m.requestFailed(c, 0, err)
}

// Rematch asks c's finished room for a rematch: a new room with the same
// mode and seats and a fresh deal. Every connection of the old room
// receives a created frame with the new code (the same code for every
// request); the clients then join it.
func (m *Manager) Rematch(ctx context.Context, c *Client, code string) (string, error) {
	r, err := m.joinedRoom(c, code)
	if err != nil {
		return "", m.requestFailed(c, 0, err)
	}
	res, err := request(ctx, r, func(reply chan rematchResult) any { return msgRematch{c: c, reply: reply} })
	if err == nil {
		err = res.err
	}
	switch {
	case err == nil:
		return res.code, nil
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrTooManyRooms), errors.Is(err, ErrStorage), errors.Is(err, ErrInvalidSpec):
		c.Out.Push(wire.NewError(errorCode(err), errorMsg(err)))
		return "", err
	case errors.Is(err, ErrRateLimited):
		c.Out.Push(wire.NewReject(0, wire.RejectCodeRate, "too many rooms created; try again later"))
		return "", err
	default:
		return "", m.requestFailed(c, 0, err)
	}
}

// requestFailed tells c why a request could not reach its room, and passes
// err on.
func (m *Manager) requestFailed(c *Client, seq int, err error) error {
	switch {
	case err == nil, errors.Is(err, ErrClosed), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	case errors.Is(err, ErrNotJoined), errors.Is(err, errRoomGone):
		c.Out.Push(wire.NewReject(seq, wire.RejectCodeBadRequest, "join the room first"))
	case errors.Is(err, ErrNotOver):
		c.Out.Push(wire.NewReject(seq, wire.RejectCodeBadRequest, "the game is not over"))
	default:
		c.Out.Push(wire.NewReject(seq, wire.RejectCodeBadRequest, "the request failed"))
	}
	return err
}

// Snapshot returns a copy of a room's state, loading the room if needed.
// It holds hidden information: for tests, diagnostics and development
// endpoints only, never for a client.
func (m *Manager) Snapshot(ctx context.Context, code string) (Snapshot, error) {
	for range 3 {
		r, err := m.get(code, nil)
		if err != nil {
			return Snapshot{}, err
		}
		if r == nil {
			continue
		}
		snap, err := request(ctx, r, func(reply chan Snapshot) any { return msgSnapshot{reply: reply} })
		if errors.Is(err, errRoomGone) {
			continue
		}
		return snap, err
	}
	return Snapshot{}, ErrRoomNotFound
}

// isContextErr reports whether err means the caller stopped waiting (its
// connection closed), so there is nobody left to tell.
func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
