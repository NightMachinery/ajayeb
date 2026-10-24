// Package lobbyroom is the shared lobby of D60: its own small room kind,
// "lobby", where a group of two to seven gathers before a game
// (docs/haftgane/platform-architecture.md, "Lobby, admin powers and
// rematch" and "Shared lobby"). The creator takes the first slot and is
// the first real admin; humans joining by link take the open slots, and
// later ones watch. Admins add and remove bots, change the seat count and
// the options, seat spectators, remove humans and open a rematch's
// reservations; anyone may release their own slot. The roles are package
// roles', persisted with the lobby and carried into the game.
//
// Start, by a seated admin, writes the started game's record in place of
// the lobby's under the same code (store.ReplaceGame), builds its actor
// with the lobby's connections handed over and swaps it in
// (room.Manager.Swap), so the next frame each client receives is the
// game's. Three to seven seats start a Haftgane table (package tableroom);
// two start an assigned online Duel game through room.Manager.StartDuel.
// A finished table's rematch, and a finished shared Duel game's, is a new
// lobby made here (Rematch, RematchDuel).
//
// Every change is a transaction, as in the table: clone the record,
// validate the operation, write the candidate atomically, then swap it in
// and publish. A two-seat lobby, Duel's online game, can always be
// created, started and rematched. A lobby of three or more seats, which
// starts a Haftgane table, is behind the same switch as creating tables
// (Config.Creation): creating one, growing a lobby to three or more seats,
// starting one and a table's rematch. Loading a saved one is not.
package lobbyroom

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Defaults for the zero fields of Config.
const (
	defaultSeqClients  = 32
	defaultMaxClients  = 32
	defaultIdleTimeout = 30 * time.Minute
	defaultInboxSize   = 64
)

// Storage is what a lobby needs from package store; *store.Store
// satisfies it.
type Storage interface {
	LoadGame(code, game string, assess func(data []byte) (store.Assessment, error)) ([]byte, error)
	SaveGame(code, game string, data []byte) error
	// CreateGame writes a new lobby's first record, refusing with
	// store.ErrExists a code that any file holds (store.Store.CreateGame).
	CreateGame(code, game string, data []byte) ([]byte, error)
	PeekGame(code, game string) ([]byte, bool, error)
}

// Config holds the lobbies' dependencies and limits.
type Config struct {
	Clock  room.Clock   // default room.SystemClock
	Logger *slog.Logger // default: discard
	// Tables starts the Haftgane games.
	Tables *tableroom.Tables
	// Creation allows lobbies of three or more seats, which start Haftgane
	// tables: lobby-create and a seats edit to three or more, their Start,
	// and a table's rematch. It is off in production until Haftgane ships
	// (-haftgane-tables). Two-seat lobbies, which start Duel's online game,
	// and Duel's shared rematches do not depend on it.
	Creation bool
	// SeqClients bounds, for each user, the clients whose last accepted
	// seq the lobby remembers.
	SeqClients int
	// MaxClients bounds the connections a user without a slot may find in
	// the lobby, and MaxUserClients every user's own connections (error
	// room-full beyond either). Defaults 32 and
	// tableroom.DefaultMaxUserClients.
	MaxClients     int
	MaxUserClients int
	// RequestBurst and RequestInterval are each user's request budget
	// (tableroom.Budget). Defaults tableroom.DefaultRequestBurst and
	// tableroom.DefaultRequestInterval.
	RequestBurst    int
	RequestInterval time.Duration
	IdleTimeout     time.Duration
	InboxSize       int
}

// Lobbies is the lobby kind: its registration, creation and rematches.
// Build one with New, register Kind in the Manager's kinds, then Bind the
// Manager and the store.
type Lobbies struct {
	cfg  Config
	log  *slog.Logger
	mu   sync.Mutex
	m    *room.Manager
	st   Storage
	live map[string]*actor
}

// New returns the lobby kind with cfg's defaults filled in, and makes it
// the tables' maker of rematch lobbies.
func New(cfg Config) *Lobbies {
	if cfg.Clock == nil {
		cfg.Clock = room.SystemClock{}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.SeqClients <= 0 {
		cfg.SeqClients = defaultSeqClients
	}
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = defaultMaxClients
	}
	if cfg.MaxUserClients <= 0 {
		cfg.MaxUserClients = tableroom.DefaultMaxUserClients
	}
	if cfg.RequestBurst <= 0 {
		cfg.RequestBurst = tableroom.DefaultRequestBurst
	}
	if cfg.RequestInterval <= 0 {
		cfg.RequestInterval = tableroom.DefaultRequestInterval
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}
	if cfg.InboxSize <= 0 {
		cfg.InboxSize = defaultInboxSize
	}
	l := &Lobbies{cfg: cfg, log: cfg.Logger, live: map[string]*actor{}}
	if cfg.Tables != nil {
		cfg.Tables.SetSuccessors(l)
	}
	return l
}

// Kind is the room-kind registration.
func (l *Lobbies) Kind() room.Kind {
	return room.Kind{ID: Game, Codec: Codec(), Create: l.create, Check: l.check, Load: l.load}
}

// Bind gives the lobbies their Manager and store (a *store.Store opened
// with this kind's codec).
func (l *Lobbies) Bind(m *room.Manager, st Storage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.m, l.st = m, st
	m.SetDuelSuccessors(l)
}

var errUnbound = errors.New("lobbyroom: no Manager and store bound")

func (l *Lobbies) store() Storage {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.st == nil {
		return unbound{}
	}
	return l.st
}

type unbound struct{}

func (unbound) LoadGame(string, string, func([]byte) (store.Assessment, error)) ([]byte, error) {
	return nil, errUnbound
}
func (unbound) SaveGame(string, string, []byte) error             { return errUnbound }
func (unbound) PeekGame(string, string) ([]byte, bool, error)     { return nil, false, errUnbound }
func (unbound) CreateGame(string, string, []byte) ([]byte, error) { return nil, errUnbound }

// create answers lobby-create: a lobby of the requested seat count and
// options, the creator in its first slot and its one real admin. The
// Manager then joins the creator.
func (l *Lobbies) create(m *room.Manager, c *room.Client, msg wire.ClientMessage) (string, error) {
	req, ok := msg.(wire.LobbyCreate)
	switch {
	case !ok || req.SeatCount < MinSeats || req.SeatCount > MaxSeats || !validOptions(req.Options) || !validName(c.Name):
		return "", room.ErrInvalidSpec
	case !l.allows(req.SeatCount):
		return "", room.ErrForbidden
	}
	r := newRecord(req.SeatCount, req.Options, c.UserID)
	r.Slots[0] = Slot{Kind: SlotHuman, UserID: c.UserID, Name: c.Name}
	return l.save(m, r)
}

// allows reports whether a lobby may have seats seats under Config.Creation.
func (l *Lobbies) allows(seats int) bool {
	return seats <= MinSeats || l.cfg.Creation
}

// maxSeats is the most seats a lobby may have under Config.Creation.
func (l *Lobbies) maxSeats() int {
	if l.cfg.Creation {
		return MaxSeats
	}
	return MinSeats
}

// Reserve returns a fresh code for a finished table's rematch lobby
// (tableroom.Successors), which the table writes down before Rematch
// makes the lobby.
func (l *Lobbies) Reserve(m *room.Manager) (string, error) {
	if !l.cfg.Creation {
		return "", room.ErrForbidden
	}
	return m.FreshCode()
}

// ReserveDuel returns a fresh code for a finished shared Duel game's
// rematch lobby (room.DuelSuccessors), which the game writes down before
// RematchDuel makes the lobby. A two-seat lobby needs no Creation.
func (l *Lobbies) ReserveDuel(m *room.Manager) (string, error) {
	return m.FreshCode()
}

// Rematch makes a finished table's successor lobby under the code the
// table reserved (tableroom.Successors): the old seat count, options and
// Start-time bots, each original human's slot reserved for them, the
// requester its creator and only admin, and the table its parent. When
// an earlier attempt for the same parent already wrote it, Rematch finds
// it and writes nothing; when another room has the code, it answers
// tableroom.ErrCodeTaken. Both are decided by the store's CreateGame,
// from the bytes that hold the code, in the same step as the write, so
// another creator taking the code at any moment can neither be
// overwritten nor overwrite the lobby. It only writes the record: the
// first join loads the lobby, as after a restart.
func (l *Lobbies) Rematch(m *room.Manager, code string, rm tableroom.Rematch) error {
	if !l.cfg.Creation {
		return room.ErrForbidden
	}
	seating := "random"
	if rm.LobbyOrder {
		seating = "lobby"
	}
	sides := rm.Sides
	if sides == "" {
		sides = "day"
	}
	undo, reveal := rm.AllowUndo, rm.FinalReveal
	r := newRecord(len(rm.Slots), wire.LobbyOptions{Sides: &sides, Seating: &seating, AllowUndo: &undo, FinalReveal: &reveal}, rm.Creator)
	for i, s := range rm.Slots {
		if s.UserID != "" {
			r.Slots[i] = Slot{Kind: SlotReserved, UserID: s.UserID, Name: s.Name}
		} else {
			r.Slots[i] = Slot{Kind: SlotBot, Bot: s.Bot}
		}
	}
	r.Code, r.Parent = code, rm.Parent
	return l.saveRematch(m, code, r)
}

// RematchDuel creates the two-seat successor lobby of a shared-lobby Duel
// game under the code ReserveDuel gave, as Rematch does for a table; it
// needs no Creation.
func (l *Lobbies) RematchDuel(m *room.Manager, code string, rm room.DuelRematch) error {
	opts := wire.LobbyOptions{Hints: &rm.Hints}
	switch rm.FirstPlayer {
	case engine.FirstPlayerSeat0:
		v := 0
		opts.FirstPlayer = &v
	case engine.FirstPlayerSeat1:
		v := 1
		opts.FirstPlayer = &v
	}
	r := newRecord(2, opts, rm.Creator)
	for i, s := range rm.Seats {
		if s.Kind == store.SeatKindHuman {
			r.Slots[i] = Slot{Kind: SlotReserved, UserID: s.UserID, Name: s.Name}
		} else {
			r.Slots[i] = Slot{Kind: SlotBot, Bot: string(s.Bot)}
		}
	}
	r.Code, r.Parent = code, rm.Parent
	if err := l.saveRematch(m, code, r); errors.Is(err, tableroom.ErrCodeTaken) {
		return store.ErrExists
	} else {
		return err
	}
}

// saveRematch shares the conditional create and parent-idempotency transaction
// between both games, while their wrappers supply their own setup options.
func (l *Lobbies) saveRematch(m *room.Manager, code string, r Record) error {
	now := l.cfg.Clock.Now().UTC()
	r.Created, r.Updated = now, now
	if r.Parent == "" || validate(&r, l.cfg.SeqClients) != nil {
		return room.ErrInvalidSpec
	}
	successor := func(data []byte) error {
		if old, err := decodeRecord(data); err == nil && old.Parent == r.Parent {
			return nil
		}
		return tableroom.ErrCodeTaken
	}
	st := l.store()
	if m.Loaded(code) {
		// A room in memory has its file, which says whose it is.
		data, _, _ := st.PeekGame(code, Game)
		return successor(data)
	}
	data, err := r.encode()
	var existing []byte
	if err == nil {
		existing, err = st.CreateGame(code, Game, data)
	}
	switch {
	case errors.Is(err, store.ErrExists):
		return successor(existing)
	case err != nil:
		l.log.Warn("lobby rematch failed", "room", code, "err", err)
		return room.ErrStorage
	}
	l.log.Info("lobby created", "room", code, "seats", r.SeatCount, "parent", r.Parent)
	return nil
}

func newRecord(seats int, opts wire.LobbyOptions, creator string) Record {
	r := Record{Schema: Schema, Game: Game, Format: recordFormat, Status: store.StatusLobby,
		SeatCount: seats, Options: cloneOptions(opts), Slots: make([]Slot, seats), Roles: roles.New(creator),
		Unseated: []string{}, Retired: []string{}, Seqs: []SeqRecord{}, RoomLog: []wire.PlatformLogEntry{}}
	for i := range r.Slots {
		r.Slots[i] = Slot{Kind: SlotOpen}
	}
	return r
}

// save gives a new lobby record a fresh code, writes it and starts its
// actor.
func (l *Lobbies) save(m *room.Manager, base Record) (string, error) {
	if err := validate(&base, l.cfg.SeqClients); err != nil {
		return "", room.ErrInvalidSpec
	}
	return m.CreateHandle(Game, func(code string) (*room.Handle, error) {
		r := base.clone()
		r.Code = code
		now := l.cfg.Clock.Now().UTC()
		r.Created, r.Updated = now, now
		data, err := r.encode()
		if err == nil {
			_, err = l.store().CreateGame(code, Game, data)
		}
		if errors.Is(err, store.ErrExists) {
			return nil, err
		}
		if err != nil {
			l.log.Warn("lobby create failed", "room", code, "err", err)
			return nil, room.ErrStorage
		}
		l.log.Info("lobby created", "room", code, "seats", r.SeatCount)
		return l.newActor(m, r)
	})
}

// check refuses a cold join before the Manager reserves a slot. A claim
// is checked by the actor once the lobby is loaded.
func (l *Lobbies) check(_ *room.Manager, code string, _ *room.Client, _ *string) error {
	data, archived, err := l.store().PeekGame(code, Game)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return room.ErrRoomNotFound
	case err != nil:
		l.log.Warn("lobby check failed", "room", code, "err", err)
		return room.ErrStorage
	case archived:
		var reason store.ArchiveReason
		if r, err := decodeRecord(data); err == nil {
			reason = r.ArchiveReason
		}
		return room.ArchivedRoom(reason)
	}
	return nil
}

// load restores a lobby: a record that does not validate is archived as
// replay-failed, as a table's would be.
func (l *Lobbies) load(m *room.Manager, code string) (*room.Handle, error) {
	var rec *Record
	_, err := l.store().LoadGame(code, Game, func(data []byte) (store.Assessment, error) {
		r, err := decodeRecord(data)
		if err != nil {
			return store.Assessment{}, err
		}
		if err := validate(r, l.cfg.SeqClients); err != nil {
			return store.Assessment{Archive: store.ArchiveReplayFailed, Cause: err}, nil
		}
		rec = r
		return store.Assessment{}, nil
	})
	var archived *store.ArchivedError
	switch {
	case errors.As(err, &archived):
		l.log.Warn("lobby archived", "room", code, "reason", string(store.SafeArchiveReason(archived.Reason)))
		return nil, room.ArchivedRoom(archived.Reason)
	case errors.Is(err, fs.ErrNotExist):
		return nil, room.ErrRoomNotFound
	case err != nil || rec == nil:
		l.log.Warn("lobby load failed", "room", code, "err", err)
		return nil, room.ErrStorage
	}
	return l.newActor(m, *rec)
}

func (l *Lobbies) register(a *actor) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.live[a.code] = a
}

func (l *Lobbies) unregister(a *actor) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.live[a.code] == a {
		delete(l.live, a.code)
	}
}

// lookup returns the running actor of code, or nil (tests, diagnostics).
func (l *Lobbies) lookup(code string) *actor {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.live[code]
}
