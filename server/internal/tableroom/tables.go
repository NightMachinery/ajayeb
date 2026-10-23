// Package tableroom is Haftgane's room actor: one goroutine per table that
// owns its record, accepts simultaneous picks, resolves complete batches,
// projects per-viewer frames, runs its bots and persists every change
// before anyone hears of it (docs/haftgane/platform-architecture.md). It is
// a separate actor beside Duel's, registered as the "haftgane" room kind
// through package room's lifetime handle; Duel's room, codec and frames are
// untouched.
//
// The engine sits behind one seam, engine.go: the fake engine until
// M16.haftgane-engine-contract binds the real one. The kind's codec and
// load path are always registered; creating a table is behind a flag that
// is off in production (Config.Creation), and only the Go API and the
// development endpoint create one until the lobby's Start calls it.
//
// Every change is a transaction: clone the record, validate the operation,
// write the candidate atomically, then swap it in and publish frames. A
// failed write discards the candidate and answers a private storage
// refusal. Nothing is published, a reveal included, before it is durable.
package tableroom

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Defaults for the zero fields of Config.
const (
	DefaultSeqClients  = 32
	DefaultPickClients = 32
	DefaultMaxClients  = 32
	DefaultIdleTimeout = 30 * time.Minute
	DefaultInboxSize   = 64
	// maxNameBytes bounds a stored display name, as hello does.
	maxNameBytes = 64
)

// Storage is what a table needs from package store; *store.Store
// satisfies it. SaveGame writes atomically or not at all.
type Storage interface {
	LoadGame(code, game string, assess func(data []byte) (store.Assessment, error)) ([]byte, error)
	SaveGame(code, game string, data []byte) error
	PeekGame(code, game string) ([]byte, bool, error)
}

// Config holds the tables' dependencies and limits.
type Config struct {
	Clock  room.Clock   // default room.SystemClock
	Logger *slog.Logger // default: discard
	// Pool bounds bot searches across every room (ai.Pool, keyed by the
	// smallest attached human participant's address). Nil: no bound.
	Pool *ai.Pool
	// Seed returns a new table's seed. Default: crypto/rand.
	Seed func() uint64
	// Creation allows creating tables. It is off in production until the
	// lobby ships (flag -haftgane-tables); loading saved tables does not
	// depend on it.
	Creation bool
	// SeqClients bounds, for each user, the clients whose last accepted
	// seq the table remembers.
	SeqClients int
	// PickClients bounds, for each user, the clients whose floor a pending
	// pick keeps (the seq that last changed it from that client). A new
	// client past it is refused with rate until the point resolves, since
	// forgetting a floor would let an old seq overwrite a newer pick.
	PickClients int
	MaxClients  int
	IdleTimeout time.Duration
	InboxSize   int
	// BotWork, when set, runs inside each bot search, holding its pool
	// permit: the tests' stand-in for search time and cancellation.
	BotWork func(ctx context.Context, seat int, point string) error
}

// Errors of the creation API.
var (
	// ErrCreationOff: table creation is switched off.
	ErrCreationOff = errors.New("tableroom: table creation is off")
	// ErrSpec: the seat list or the options cannot make a table.
	ErrSpec = errors.New("tableroom: invalid table spec")
)

// Tables is the table kind: its registration, its creation API and the
// log download. Build one with New, register Kind in the Manager's kinds
// (its codec goes into the store's), then Bind the Manager and the store.
type Tables struct {
	cfg  Config
	log  *slog.Logger
	mu   sync.Mutex
	m    *room.Manager
	st   Storage
	live map[string]*actor
}

// New returns the table kind with cfg's defaults filled in.
func New(cfg Config) *Tables {
	if cfg.Clock == nil {
		cfg.Clock = room.SystemClock{}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.Seed == nil {
		cfg.Seed = cryptoSeed
	}
	if cfg.SeqClients <= 0 {
		cfg.SeqClients = DefaultSeqClients
	}
	if cfg.PickClients <= 0 {
		cfg.PickClients = DefaultPickClients
	}
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = DefaultMaxClients
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.InboxSize <= 0 {
		cfg.InboxSize = DefaultInboxSize
	}
	return &Tables{cfg: cfg, log: cfg.Logger, live: map[string]*actor{}}
}

func cryptoSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(b[:])
}

// Kind is the room-kind registration: the codec, the cold-join check and
// the load. A creation frame from a client is refused: no lobby creates
// tables yet.
func (t *Tables) Kind() room.Kind {
	return room.Kind{
		ID:    Game,
		Codec: Codec(),
		Create: func(*room.Manager, *room.Client, wire.ClientMessage) (string, error) {
			return "", room.ErrInvalidSpec
		},
		Check: t.check,
		Load:  t.load,
	}
}

// Bind gives the tables the Manager they create rooms in and the store
// they keep them in (a *store.Store opened with this kind's codec).
func (t *Tables) Bind(m *room.Manager, st Storage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.m, t.st = m, st
}

func (t *Tables) manager() *room.Manager {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.m
}

// errUnbound is a table operation before Bind.
var errUnbound = errors.New("tableroom: no Manager and store bound")

func (t *Tables) store() Storage {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.st == nil {
		return unbound{}
	}
	return t.st
}

// unbound is the store of unbound Tables: it fails every operation.
type unbound struct{}

func (unbound) LoadGame(string, string, func([]byte) (store.Assessment, error)) ([]byte, error) {
	return nil, errUnbound
}
func (unbound) SaveGame(string, string, []byte) error         { return errUnbound }
func (unbound) PeekGame(string, string) ([]byte, bool, error) { return nil, false, errUnbound }

// check refuses a cold join before the Manager reserves a slot.
func (t *Tables) check(_ *room.Manager, code string, _ *room.Client, claim *string) error {
	if claim != nil {
		return room.ErrBadClaim
	}
	data, archived, err := t.store().PeekGame(code, Game)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return room.ErrRoomNotFound
	case err != nil:
		t.log.Warn("table check failed", "room", code, "err", err)
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

// bounds are the seq bounds every write keeps and restore checks.
func (t *Tables) bounds() seqBounds {
	return seqBounds{acks: t.cfg.SeqClients, floors: t.cfg.PickClients}
}

// load restores a table: the store's assessment validates the record,
// refreshes a stale hash or archives it durably.
func (t *Tables) load(m *room.Manager, code string) (*room.Handle, error) {
	var st *state
	_, err := t.store().LoadGame(code, Game, func(data []byte) (store.Assessment, error) {
		v, s, err := assess(data, t.bounds())
		st = s
		return v, err
	})
	var archived *store.ArchivedError
	switch {
	case errors.As(err, &archived):
		// The cause stays out of the log: it can name the record's fields.
		t.log.Warn("table archived", "room", code, "reason", string(store.SafeArchiveReason(archived.Reason)))
		return nil, room.ArchivedRoom(archived.Reason)
	case errors.Is(err, fs.ErrNotExist):
		return nil, room.ErrRoomNotFound
	case err != nil || st == nil:
		t.log.Warn("table load failed", "room", code, "err", err)
		return nil, room.ErrStorage
	}
	return t.newActor(m, st, nil)
}

// Participant is one lobby slot of a new table: a human (UserID and
// Name) or a bot (Bot, its level).
type Participant struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Bot    string `json:"bot"`
}

// Spec is a new table: its participants in lobby order and its options.
// Roles, when set, are the lobby's roles carried on at Start; otherwise
// the creator (if any) is the one admin. Prefix, for tests and development
// only, starts the table with these complete batches already resolved.
type Spec struct {
	Participants  []Participant `json:"participants"`
	RandomSeating bool          `json:"randomSeating"`
	HideAtEnd     bool          `json:"hideAtEnd"`
	AllowUndo     bool          `json:"allowUndo"`
	Creator       string        `json:"creator"`
	Roles         *roles.Roles  `json:"roles,omitempty"`
	Seed          *uint64       `json:"seed,omitempty"`
	Prefix        []BatchRecord `json:"prefix"`
}

// CreateTable starts a filled table from a seat list and returns its code:
// the tests' and the development endpoint's direct path, behind
// Config.Creation. The lobby's Start builds the same record (StartTable).
func (t *Tables) CreateTable(spec Spec) (string, error) {
	if !t.cfg.Creation {
		return "", ErrCreationOff
	}
	m := t.manager()
	if m == nil {
		return "", errUnbound
	}
	base, g, err := t.newRecord(spec)
	if err != nil {
		return "", err
	}
	return m.CreateHandle(Game, func(code string) (*room.Handle, error) {
		rec := base.clone()
		rec.Code = code
		now := t.cfg.Clock.Now().UTC()
		rec.Created, rec.Updated = now, now
		data, err := rec.encode()
		if err == nil {
			err = t.store().SaveGame(code, Game, data)
		}
		if err != nil {
			t.log.Warn("table create failed", "room", code, "err", err)
			return nil, room.ErrStorage
		}
		t.log.Info("table created", "room", code, "seats", len(rec.Seats))
		return t.newActor(m, &state{rec: rec, game: g}, nil)
	})
}

// newRecord builds and validates a new table's record (without its code
// and times) and its engine value.
func (t *Tables) newRecord(spec Spec) (Record, game, error) {
	n := len(spec.Participants)
	if n < MinSeats || n > MaxSeats || len(spec.Prefix) > maxBatches {
		return Record{}, game{}, ErrSpec
	}
	users := map[string]bool{}
	humans := 0
	for _, p := range spec.Participants {
		switch {
		case p.Bot != "":
			if _, ok := allowance(p.Bot); !ok || p.UserID != "" {
				return Record{}, game{}, ErrSpec
			}
		default:
			humans++
			if p.UserID == "" || users[p.UserID] || len(p.UserID) > 64 || !validName(p.Name) {
				return Record{}, game{}, ErrSpec
			}
			users[p.UserID] = true
		}
	}
	rs := roles.New(spec.Creator)
	if spec.Roles != nil {
		rs = spec.Roles.Clone()
	}
	seed := t.cfg.Seed()
	if spec.Seed != nil {
		seed = *spec.Seed
	}
	su := makeSetup(seed, n, spec.RandomSeating, spec.HideAtEnd)
	g, err := replay(su, engineBatches(spec.Prefix))
	if err != nil {
		return Record{}, game{}, ErrSpec
	}
	order := seatOrder(su.Seed, su.Seats, su.Options.Seating)
	base := Record{
		Schema: Schema, Game: Game, Rules: RulesVersion,
		Setup:      SetupRecord{Seed: seed, Seats: n, RandomSeating: spec.RandomSeating, HideAtEnd: spec.HideAtEnd},
		Assignment: make([]int, n), Original: make([]string, n), Seats: make([]SeatRecord, n),
		Roles: rs, Previous: []PreviousRecord{}, Retired: []string{},
		StartingHumans: humans, AllowUndo: spec.AllowUndo,
		Batches: make([]BatchRecord, 0, len(spec.Prefix)), Hash: g.Fingerprint(),
		Public: len(spec.Prefix) + 1, Edits: make([]int, n),
		Pending: []PendingSlot{}, Seqs: []SeqRecord{}, Work: []WorkRow{}, RoomLog: []wire.PlatformLogEntry{},
		Status: store.StatusPlaying,
	}
	for _, b := range spec.Prefix {
		base.Batches = append(base.Batches, BatchRecord{Point: b.Point, Picks: append([]PickRecord{}, b.Picks...)})
	}
	for s := range n {
		p := spec.Participants[order[s]]
		base.Assignment[s] = int(order[s])
		sr := SeatRecord{Seat: s, Kind: SeatHuman, UserID: p.UserID, Name: p.Name}
		if p.Bot != "" {
			sr = SeatRecord{Seat: s, Kind: SeatBot, Bot: p.Bot}
		} else {
			base.Original[s] = p.UserID
		}
		base.Seats[s] = sr
	}
	openPoint(&base, &g)
	if err := validate(&base, &g, t.bounds()); err != nil {
		t.log.Error("table create invalid", "err", err)
		return Record{}, game{}, ErrSpec
	}
	return base, g, nil
}

func validName(name string) bool {
	return name != "" && len(name) <= maxNameBytes && utf8.ValidString(name) && strings.TrimSpace(name) == name
}

// openPoint installs g's open point in the record: a fresh wire reference
// at the current public revision and a work row for each required bot
// seat, or the finished status once the game is over.
func openPoint(r *Record, g *game) {
	p := g.Point()
	r.Work = []WorkRow{}
	if pointOver(p) {
		r.Point, r.Status = nil, store.StatusFinished
		return
	}
	r.Point, r.Status = &wire.PointRef{ID: p.ID, Revision: r.Public}, store.StatusPlaying
	for _, s := range p.Required.Seats() {
		if sr := r.Seats[s]; sr.Kind == SeatBot {
			a, _ := allowance(sr.Bot)
			r.Work = append(r.Work, WorkRow{Seat: int(s), Point: p.ID, Policy: policyVersion, Level: sr.Bot, Allowance: a})
		}
	}
}

func (t *Tables) register(a *actor) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.live[a.code] = a
}

func (t *Tables) unregister(a *actor) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.live[a.code] == a {
		delete(t.live, a.code)
	}
}

// lookup returns the running actor of code, or nil (tests, diagnostics).
func (t *Tables) lookup(code string) *actor {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.live[code]
}
