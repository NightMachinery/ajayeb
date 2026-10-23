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
// two will start Duel once M18.shared-lobby adds Duel's create path, and
// are refused until then. A finished table's rematch is a new lobby made
// here (Rematch).
//
// Every change is a transaction, as in the table: clone the record,
// validate the operation, write the candidate atomically, then swap it in
// and publish. Creating a lobby is behind the same switch as creating
// tables (Config.Creation); loading a saved one is not.
package lobbyroom

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"sync"
	"time"

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
	PeekGame(code, game string) ([]byte, bool, error)
}

// Config holds the lobbies' dependencies and limits.
type Config struct {
	Clock  room.Clock   // default room.SystemClock
	Logger *slog.Logger // default: discard
	// Tables starts the Haftgane games.
	Tables *tableroom.Tables
	// Creation allows creating lobbies (lobby-create and rematches). It is
	// off in production until the lobby's client ships (-haftgane-tables).
	Creation bool
	// SeqClients bounds, for each user, the clients whose last accepted
	// seq the lobby remembers.
	SeqClients  int
	MaxClients  int
	IdleTimeout time.Duration
	InboxSize   int
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
func (unbound) SaveGame(string, string, []byte) error         { return errUnbound }
func (unbound) PeekGame(string, string) ([]byte, bool, error) { return nil, false, errUnbound }

// create answers lobby-create: a lobby of the requested seat count and
// options, the creator in its first slot and its one real admin. The
// Manager then joins the creator.
func (l *Lobbies) create(m *room.Manager, c *room.Client, msg wire.ClientMessage) (string, error) {
	req, ok := msg.(wire.LobbyCreate)
	switch {
	case !l.cfg.Creation:
		return "", room.ErrForbidden
	case !ok || req.SeatCount < MinSeats || req.SeatCount > MaxSeats || !validOptions(req.Options) || !validName(c.Name):
		return "", room.ErrInvalidSpec
	}
	r := newRecord(req.SeatCount, req.Options, c.UserID)
	r.Slots[0] = Slot{Kind: SlotHuman, UserID: c.UserID, Name: c.Name}
	return l.save(m, r)
}

// Rematch makes a finished table's successor lobby (tableroom.Successors):
// the old seat count, options and Start-time bots, each original human's
// slot reserved for them, and the requester its creator and only admin.
func (l *Lobbies) Rematch(m *room.Manager, rm tableroom.Rematch) (string, error) {
	if !l.cfg.Creation {
		return "", room.ErrForbidden
	}
	seating := "random"
	if rm.LobbyOrder {
		seating = "lobby"
	}
	undo, reveal := rm.AllowUndo, rm.FinalReveal
	r := newRecord(len(rm.Slots), wire.LobbyOptions{Seating: &seating, AllowUndo: &undo, FinalReveal: &reveal}, rm.Creator)
	for i, s := range rm.Slots {
		if s.UserID != "" {
			r.Slots[i] = Slot{Kind: SlotReserved, UserID: s.UserID, Name: s.Name}
		} else {
			r.Slots[i] = Slot{Kind: SlotBot, Bot: s.Bot}
		}
	}
	return l.save(m, r)
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
			err = l.store().SaveGame(code, Game, data)
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
