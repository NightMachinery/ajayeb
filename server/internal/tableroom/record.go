package tableroom

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Game is the table kind's stable id: the room kind, the codec's game and
// the frames' game.
const Game = "haftgane"

// Schema is the record's schema, dispatched by the store's bounded header.
const Schema = 2

// Record is a table's durable record, the whole schema-2 room file
// (docs/haftgane/platform-architecture.md, "Persistence, replay and
// compatibility"). It holds no pending hash, engine checkpoint, bot result
// or task generation: the engine value is rebuilt by replaying Batches from
// Setup, and restore validates all of it.
type Record struct {
	Schema int    `json:"schema"`
	Game   string `json:"game"`
	Rules  int    `json:"rules"`
	// Format is the record format (recordFormat); decodeRecord upgrades an
	// older one explicitly and refuses a newer one.
	Format        int                 `json:"format"`
	Code          string              `json:"code"`
	Status        store.Status        `json:"status"`
	Created       time.Time           `json:"created"`
	Updated       time.Time           `json:"updated"`
	ArchivedAt    *time.Time          `json:"archivedAt,omitempty"`
	ArchiveReason store.ArchiveReason `json:"archiveReason,omitempty"`

	// Setup is the validated setup: seed, seat count and options.
	Setup SetupRecord `json:"setup"`
	// Assignment is the fixed initial assignment: seat s holds lobby slot
	// Assignment[s]. Restore recomputes it from the setup.
	Assignment []int `json:"assignment"`
	// Original is each seat's original participant: the human who held it
	// at Start, or "" for a starting bot. Later switches never change it.
	Original []string `json:"original"`
	// Seats are the current controllers, one per seat.
	Seats []SeatRecord `json:"seats"`
	// Start holds the lobby's slots at Start, in lobby order: each human's
	// user and name, or a bot's level. A rematch's lobby reserves the
	// humans' slots and keeps the bots.
	Start []StartSlot `json:"start"`
	// Successor is the code of the rematch's lobby, once one was made.
	Successor string `json:"successor"`
	// Reserved is the code a rematch wrote down for its lobby before
	// making it, and Rematcher the original human who asked, its creator.
	// A retry after a refused write or a crash, by anyone who played,
	// makes or finds that same lobby, so a table has one successor. Both
	// are empty once Successor is set.
	Reserved  string `json:"reserved"`
	Rematcher string `json:"rematcher"`
	// Roles are the table's roles (package roles), carried on from the
	// lobby at Start.
	Roles roles.Roles `json:"roles"`
	// Previous holds the seat each former player held last, oldest first
	// and at most maxPrevious: the admins' previous-seat hint.
	Previous []PreviousRecord `json:"previous"`
	// Retired are the users who lost all authority (no seat, no admin
	// role) while the table remembered their seqs, in the order they lost
	// it. Their seqs stay until more than maxRetired users follow them, so
	// a retry after a switch is still a duplicate; the order depends only
	// on public switches and role changes.
	Retired []string `json:"retired"`
	// StartingHumans and AllowUndo are the starting policy inputs, fixed at
	// Start; UndoOff can only become true afterwards.
	StartingHumans int  `json:"startingHumans"`
	AllowUndo      bool `json:"allowUndo"`
	UndoOff        bool `json:"undoOff"`

	// Batches are the canonical complete batches; Hash is the rules hash
	// of their replay.
	Batches []BatchRecord `json:"batches"`
	Hash    string        `json:"hash"`

	// Public is the public revision: it advances on every change everyone
	// may see. Edits counts each seat's private edits since the last
	// public change, which resets them. A viewer's frame revision is
	// Public<<editBits plus its own seat's Edits (revisionOf), so a private
	// edit advances only its owner's frames, and a seat switch (a public
	// change) never takes a viewer's revision backwards.
	Public int   `json:"public"`
	Edits  []int `json:"edits"`
	// Point is the open point's wire reference (nil once the game is
	// over): the logical id and the public revision at opening.
	Point *wire.PointRef `json:"point,omitempty"`
	// Pending is the private pending block: each required seat's current
	// pick and confirmed flag, with the seq per client that last set it.
	Pending []PendingSlot `json:"pending"`

	// Seqs are the seq acknowledgements, oldest accepted first: at most
	// Config.SeqClients clients for each user, so one user's clients never
	// push out another's.
	Seqs []SeqRecord `json:"seqs"`
	// Work is the bot work ledger of the open point.
	Work         []WorkRow     `json:"work"`
	BotPresets   []BotPreset   `json:"botPresets"`
	BotFallbacks []BotFallback `json:"botFallbacks"`
	// RoomLog is the public log of seat switches, resignations, role
	// changes and device-link resets, oldest first and at most maxRoomLog.
	RoomLog []wire.PlatformLogEntry `json:"roomLog"`
}

// Bounds of the record's lists that grow with switches and role changes.
const (
	maxPrevious = 64
	maxRetired  = 32
	maxRoomLog  = 512
	// maxSlotUsers bounds the users whose floors one pending pick keeps:
	// each user who controlled the seat during the point and changed it.
	maxSlotUsers = 16
	// editBits is the room a frame revision leaves for a seat's own edits
	// between two public changes; a seat past maxEdits is refused with
	// rate until the next public change.
	editBits = 20
	maxEdits = 1<<editBits - 1
)

// PreviousRecord is the seat a former player held last.
type PreviousRecord struct {
	UserID string `json:"userId"`
	Seat   int    `json:"seat"`
}

// revisionOf is the frame revision of a viewer at seat (-1: no seat).
func revisionOf(r *Record, seat int) int {
	rev := r.Public << editBits
	if seat >= 0 {
		rev += r.Edits[seat]
	}
	return rev
}

// bumpPublic records a public change: the public revision advances and
// every seat's private edit count starts again.
func bumpPublic(r *Record) {
	r.Public++
	for s := range r.Edits {
		r.Edits[s] = 0
	}
}

// appendLog adds entries to the room log, forgetting its oldest beyond
// maxRoomLog.
func appendLog(r *Record, entries ...wire.PlatformLogEntry) {
	r.RoomLog = append(r.RoomLog, entries...)
	if over := len(r.RoomLog) - maxRoomLog; over > 0 {
		r.RoomLog = slices.Delete(r.RoomLog, 0, over)
	}
}

// SetupRecord is the stored setup.
type SetupRecord struct {
	Seed          uint64 `json:"seed"`
	Seats         int    `json:"seats"`
	RandomSeating bool   `json:"randomSeating"`
	HideAtEnd     bool   `json:"hideAtEnd"`
	Sides         string `json:"sides"`
}

// Seat controller kinds. A table has no open seats: the lobby's Start
// fills every one.
const (
	SeatHuman = "human"
	SeatBot   = "bot"
)

// SeatRecord is one seat's current controller: a human or a bot. Taken
// is the public revision when its human took it (0 at Start), which ranks
// the candidates for the temporary admin. Epoch is the epoch of the
// seat's device-link claims (package ident): the public revision of the
// last time a human took the seat or its link was reset, 0 at Start. Two
// claims for one user never share an epoch, so a reset or a switch
// revokes every claim made before it. (Records of the dev path before the
// lobby may hold open seats; restore refuses them.)
type SeatRecord struct {
	Seat   int    `json:"seat"`
	Kind   string `json:"kind"`
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Bot    string `json:"bot"`
	Taken  int    `json:"taken"`
	Epoch  int    `json:"epoch"`
}

// StartSlot is one lobby slot at Start: a human (UserID and Name) or a
// bot (Bot, its level).
type StartSlot struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Bot    string `json:"bot"`
}

// recordFormat is the format this build writes. Format 0 (no field) is
// the room actor's development record from before the lobby, which
// upgrade reads explicitly.
// Format 2 freezes algorithm/preset/bounds for future switches and points.
// Older records without supported frozen presets fail closed on load.
const recordFormat = 2

// BatchRecord is one complete batch: the logical point id and one key per
// required seat, sorted by seat.
type BatchRecord struct {
	Point string       `json:"point"`
	Picks []PickRecord `json:"picks"`
}

// PickRecord is one seat's key.
type PickRecord struct {
	Seat int    `json:"seat"`
	Key  string `json:"key"`
}

// PendingSlot is one required seat's accepted, unresolved pick.
type PendingSlot struct {
	Seat      int         `json:"seat"`
	Key       string      `json:"key"`
	Confirmed bool        `json:"confirmed"`
	Seqs      []SeqRecord `json:"seqs"`
}

// SeqRecord is one client's seq: a user's tab.
type SeqRecord struct {
	User   string `json:"user"`
	Client string `json:"client"`
	Seq    int    `json:"seq"`
}

// WorkRow is one bot's work at the open point, persisted before its work
// starts and frozen through restart: the policy version, the level and
// the fixed deterministic allowance.
type WorkRow struct {
	Seat  int    `json:"seat"`
	Point string `json:"point"`
	BotPreset
}

// The lowest internal level is the resignation default; an admin explicitly
// selects the level for a switch. Both use the table's frozen preset row.
const (
	BotEasy         = "easy"
	BotNormal       = "normal"
	BotHard         = "hard"
	DefaultBotLevel = BotEasy
)

// clone returns a deep copy, so a transaction never shares a slice with
// the committed record.
func (r *Record) clone() Record {
	c := *r
	if r.ArchivedAt != nil {
		at := *r.ArchivedAt
		c.ArchivedAt = &at
	}
	c.Assignment = slices.Clone(r.Assignment)
	c.Original = slices.Clone(r.Original)
	c.Seats = slices.Clone(r.Seats)
	c.Start = slices.Clone(r.Start)
	c.Roles = r.Roles.Clone()
	c.Previous = slices.Clone(r.Previous)
	c.Retired = slices.Clone(r.Retired)
	c.Batches = make([]BatchRecord, 0, len(r.Batches))
	for _, b := range r.Batches {
		c.Batches = append(c.Batches, BatchRecord{Point: b.Point, Picks: slices.Clone(b.Picks)})
	}
	c.Edits = slices.Clone(r.Edits)
	if r.Point != nil {
		p := *r.Point
		c.Point = &p
	}
	c.Pending = make([]PendingSlot, 0, len(r.Pending))
	for _, p := range r.Pending {
		p.Seqs = slices.Clone(p.Seqs)
		c.Pending = append(c.Pending, p)
	}
	c.Seqs = slices.Clone(r.Seqs)
	c.Work = slices.Clone(r.Work)
	c.BotPresets = slices.Clone(r.BotPresets)
	c.BotFallbacks = slices.Clone(r.BotFallbacks)
	c.RoomLog = slices.Clone(r.RoomLog)
	return c
}

// encode returns the record's file bytes, with every slice a JSON array.
func (r *Record) encode() ([]byte, error) {
	c := r.clone()
	for i := range c.Pending {
		if c.Pending[i].Seqs == nil {
			c.Pending[i].Seqs = []SeqRecord{}
		}
	}
	if c.Previous == nil {
		c.Previous = []PreviousRecord{}
	}
	if c.Retired == nil {
		c.Retired = []string{}
	}
	if c.BotPresets == nil {
		c.BotPresets = []BotPreset{}
	}
	if c.BotFallbacks == nil {
		c.BotFallbacks = []BotFallback{}
	}
	if c.Start == nil {
		c.Start = []StartSlot{}
	}
	data, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// decodeRecord decodes a record and validates its header. Unknown fields
// are ignored, as for Duel's file, so an additive field of a newer build
// of the same rules does not make the room unreadable; but a newer record
// format, whose fields this build would misread, is refused and left as
// it is, and an older one is upgraded explicitly.
func decodeRecord(data []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("tableroom: decoding the record: %w", err)
	}
	if r.Schema != Schema || r.Game != Game {
		return nil, fmt.Errorf("%w: not a %s schema-%d record", store.ErrGame, Game, Schema)
	}
	switch {
	case r.Format > recordFormat || r.Format < 0:
		return nil, fmt.Errorf("%w: record format %d is not this build's", store.ErrGame, r.Format)
	case r.Format < recordFormat:
		upgrade(&r)
	}
	if store.ValidateCode(r.Code) != nil || r.Updated.IsZero() {
		return nil, errors.New("tableroom: invalid record header")
	}
	switch r.Status {
	case store.StatusPlaying, store.StatusFinished, store.StatusArchived:
	default:
		return nil, errors.New("tableroom: invalid record status")
	}
	return &r, nil
}

// upgrade preserves format-1 metadata and reads a format-0 development record
// from before the lobby: it had no previous seats, retired users, away
// times, epochs or Start slots, and no switches had happened, so each
// seat's Start slot is its original human or its current bot. A seat
// still open stays open, which restore refuses. No bot presets are invented;
// records without supported frozen configuration fail validation.
func upgrade(r *Record) {
	if r.Format == 1 {
		r.Format = recordFormat
		return
	}
	r.Format = recordFormat
	r.Previous, r.Retired = []PreviousRecord{}, []string{}
	r.Roles = r.Roles.Clone()
	if len(r.Assignment) != len(r.Seats) || len(r.Original) != len(r.Seats) {
		return // validate refuses it
	}
	r.Start = make([]StartSlot, len(r.Seats))
	for s, sr := range r.Seats {
		slot := r.Assignment[s]
		if slot < 0 || slot >= len(r.Start) {
			return
		}
		if r.Original[s] != "" {
			r.Start[slot] = StartSlot{UserID: r.Original[s], Name: sr.Name}
		} else {
			r.Start[slot] = StartSlot{Bot: sr.Bot}
		}
	}
}

// inspection is the codec's read-only view of a record for the store's
// retention and preflight.
type inspection struct{ r *Record }

func (i inspection) Retention() store.RetentionHeader {
	return store.RetentionHeader{Code: i.r.Code, Status: i.r.Status, Updated: i.r.Updated, ArchivedAt: i.r.ArchivedAt}
}

// Check reports a hash-only mismatch after a compatible legal replay; an
// error means the record would archive (another rules version, or an
// invalid replay or pending block).
func (i inspection) Check() (bool, error) {
	if i.r.Rules != RulesVersion {
		return false, fmt.Errorf("%w: rules %d, engine %d", store.ErrRulesVersion, i.r.Rules, RulesVersion)
	}
	_, hashOK, err := restore(i.r, defaultBounds)
	if err != nil {
		return false, err
	}
	return !hashOK, nil
}

// Codec is the table kind's store codec.
func Codec() *store.Codec {
	return &store.Codec{Game: Game, Decode: func(data []byte) (store.Inspection, error) {
		r, err := decodeRecord(data)
		if err != nil {
			return nil, err
		}
		return inspection{r: r}, nil
	}}
}

// assess is the store's load-time verdict on a live record: another rules
// version archives as rules-version, an invalid replay or pending block as
// replay-failed, and a hash-only mismatch refreshes the hash in place.
func assess(data []byte, b seqBounds) (store.Assessment, *state, error) {
	r, err := decodeRecord(data)
	if err != nil {
		return store.Assessment{}, nil, err
	}
	if r.Rules != RulesVersion {
		return store.Assessment{Archive: store.ArchiveRulesVersion, Cause: fmt.Errorf("rules %d, engine %d", r.Rules, RulesVersion)}, nil, nil
	}
	st, hashOK, err := restore(r, b)
	if err != nil {
		return store.Assessment{Archive: store.ArchiveReplayFailed, Cause: err}, nil, nil
	}
	if !hashOK {
		st.rec.Hash = st.game.Fingerprint()
		return store.Assessment{Refresh: map[string]any{"hash": st.rec.Hash}}, st, nil
	}
	return store.Assessment{}, st, nil
}

// errInvalid is a record restore refuses. Its messages name fields and
// seats, never keys, cards, seeds or hashes.
var errInvalid = errors.New("tableroom: invalid record")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalid, fmt.Sprintf(format, args...))
}

// seqBounds are the per-user bounds of a record's seq acknowledgements
// and of each pending pick's floors. Every write keeps them and restore
// checks them, with the same values (Config.SeqClients and
// Config.PickClients).
type seqBounds struct{ acks, floors int }

// defaultBounds are the bounds of a table with the default Config.
var defaultBounds = seqBounds{acks: DefaultSeqClients, floors: DefaultPickClients}

// restore rebuilds the engine value from a record and validates
// everything the design lists: the setup and assignment, every complete
// batch's point, required seats and key membership (by replay), the
// controllers, the current point reference, each pending slot, the seq
// acknowledgements and the work ledger. It reports whether the stored hash
// matches the replay. Pending picks are never applied to the engine.
func restore(r *Record, b seqBounds) (*state, bool, error) {
	n := r.Setup.Seats
	if n < MinSeats || n > MaxSeats {
		return nil, false, invalid("seat count %d", n)
	}
	if len(r.Batches) > maxBatches {
		return nil, false, invalid("%d batches", len(r.Batches))
	}
	su, err := recordSetup(r.Setup)
	if err != nil {
		return nil, false, err
	}
	g, err := replay(su, engineBatches(r.Batches))
	if err != nil {
		return nil, false, invalid("the batches do not replay")
	}
	if err := g.CheckInvariants(); err != nil {
		return nil, false, invalid("the replayed game breaks an invariant")
	}
	if !validSeqs(r.Seqs) {
		return nil, false, invalid("the seq acknowledgements")
	}
	st := &state{rec: r.clone(), game: g}
	st.rec.Seqs = trimSeqs(st.rec.Seqs, b.acks)
	if err := validate(&st.rec, &g, b); err != nil {
		return nil, false, err
	}
	return st, g.Fingerprint() == r.Hash, nil
}

// validate checks a record against its engine value: everything restore
// checks but the replay itself. A restart runs it on the replayed value
// and every write on the value its transaction produced, so nothing is
// saved that would not restore.
func validate(r *Record, g *game, b seqBounds) error {
	n := r.Setup.Seats
	if n < MinSeats || n > MaxSeats {
		return invalid("seat count %d", n)
	}
	su, err := recordSetup(r.Setup)
	if err != nil {
		return err
	}
	order := seatOrder(su.Seed, su.Seats, su.Options.Seating)
	if len(r.Assignment) != n || len(r.Original) != n || len(r.Seats) != n || len(r.Edits) != n {
		return invalid("per-seat fields do not hold %d seats", n)
	}
	for s := range n {
		if r.Assignment[s] != int(order[s]) {
			return invalid("the assignment of seat %d", s)
		}
		if r.Edits[s] < 0 || r.Edits[s] > maxEdits {
			return invalid("the edit count of seat %d", s)
		}
		if t, e := r.Seats[s].Taken, r.Seats[s].Epoch; t < 0 || t > r.Public || e < 0 || e > r.Public {
			return invalid("when seat %d was taken or linked", s)
		}
	}
	if err := checkSeats(r); err != nil {
		return err
	}
	if err := checkStart(r); err != nil {
		return err
	}
	p := g.Point()
	switch {
	case pointOver(p) != (r.Status == store.StatusFinished):
		return invalid("the status does not match the game")
	case r.Public < 0 || r.StartingHumans < 0 || r.StartingHumans > n:
		return invalid("a negative revision or a bad starting count")
	case pointOver(p) && (r.Point != nil || len(r.Pending) > 0 || len(r.Work) > 0):
		return invalid("a point, pending pick or work row after the end")
	case !pointOver(p) && (r.Point == nil || r.Point.ID != p.ID || r.Point.Revision < 0 || r.Point.Revision > r.Public):
		return invalid("the point reference is not the open point")
	}
	confirmed := 0
	var seen [MaxSeats]bool
	for _, ps := range r.Pending {
		switch {
		case ps.Seat < 0 || ps.Seat >= n || !p.Required.Has(seatNum(ps.Seat)):
			return invalid("a pending pick for a seat the point does not require")
		case seen[ps.Seat]:
			return invalid("two pending picks for seat %d", ps.Seat)
		case !slices.Contains(legalKeys(g, ps.Seat), ps.Key):
			return invalid("a pending key seat %d is not offered", ps.Seat)
		case !validSeqs(ps.Seqs) || !seqsBounded(ps.Seqs, b.floors, maxSlotUsers):
			return invalid("the seqs of seat %d's pending pick", ps.Seat)
		}
		seen[ps.Seat] = true
		if ps.Confirmed {
			confirmed++
		}
	}
	if !pointOver(p) && confirmed == p.Required.Count() {
		return invalid("every required seat has confirmed, so the point should have resolved")
	}
	if err := checkAuthority(r, b); err != nil {
		return err
	}
	return checkWork(r, g)
}

// checkAuthority checks what grows with switches and role changes: the
// roles, the previous seats, the retired users, the room log, and that
// only a user with authority or a retired one has seqs remembered, each at
// most b.acks clients.
func checkAuthority(r *Record, b seqBounds) error {
	if r.Roles.Valid() != nil {
		return invalid("the roles")
	}
	if len(r.Previous) > maxPrevious || len(r.Retired) > maxRetired || len(r.RoomLog) > maxRoomLog {
		return invalid("a list past its bound")
	}
	for i, p := range r.Previous {
		if p.UserID == "" || p.Seat < 0 || p.Seat >= len(r.Seats) ||
			slices.IndexFunc(r.Previous, func(x PreviousRecord) bool { return x.UserID == p.UserID }) != i {
			return invalid("the previous seats")
		}
	}
	for i, u := range r.Retired {
		if u == "" || slices.Index(r.Retired, u) != i || hasAuthority(r, u) {
			return invalid("the retired users")
		}
	}
	if !validSeqs(r.Seqs) || !seqsBounded(r.Seqs, b.acks, maxSeqUsers) {
		return invalid("the seq acknowledgements")
	}
	for _, x := range r.Seqs {
		if !hasAuthority(r, x.User) && !slices.Contains(r.Retired, x.User) {
			return invalid("the seq acknowledgements")
		}
	}
	return nil
}

// maxSeqUsers bounds the users with remembered seqs: every seated human,
// every real admin, the temporary admin and the retired users.
const maxSeqUsers = MaxSeats + roles.MaxAdmins + 1 + maxRetired

// hasAuthority reports whether u may act at the table: a seated human or
// an admin (real or temporary).
func hasAuthority(r *Record, u string) bool {
	return u != "" && (slices.ContainsFunc(r.Seats, func(s SeatRecord) bool { return s.Kind == SeatHuman && s.UserID == u }) ||
		r.Roles.IsAdmin(u))
}

// seqsBounded reports whether xs holds at most capacity clients of each
// user, and at most users users.
func seqsBounded(xs []SeqRecord, capacity, users int) bool {
	per := map[string]int{}
	for _, x := range xs {
		per[x.User]++
		if per[x.User] > capacity {
			return false
		}
	}
	return len(per) <= users
}

func checkSeats(r *Record) error {
	users := map[string]bool{}
	for s, sr := range r.Seats {
		if sr.Seat != s {
			return invalid("seat %d is out of order", s)
		}
		switch sr.Kind {
		case SeatHuman:
			if sr.UserID == "" || sr.Bot != "" || users[sr.UserID] {
				return invalid("human seat %d", s)
			}
			users[sr.UserID] = true
		case SeatBot:
			if _, ok := frozenPreset(r, sr.Bot); !ok || sr.UserID != "" {
				return invalid("bot seat %d", s)
			}
		default:
			return invalid("seat %d's kind", s)
		}
	}
	return nil
}

// checkStart requires the Start slots to match the original seats: each
// human seat's slot names its original human, each starting bot's slot a
// level, and a successor only once the game is over.
func checkStart(r *Record) error {
	if r.Format != recordFormat || len(r.Start) != len(r.Seats) {
		return invalid("the record format or the Start slots")
	}
	humans := 0
	for s := range r.Seats {
		slot := r.Start[r.Assignment[s]]
		switch {
		case r.Original[s] != "":
			if slot.UserID != r.Original[s] || slot.Bot != "" || !validName(slot.Name) {
				return invalid("the Start slot of seat %d", s)
			}
			humans++
		default:
			if _, ok := frozenPreset(r, slot.Bot); !ok || slot.UserID != "" || slot.Name != "" {
				return invalid("the Start slot of seat %d", s)
			}
		}
	}
	switch {
	case humans != r.StartingHumans:
		return invalid("the starting human count")
	case r.Successor != "" && (r.Status == store.StatusPlaying || store.ValidateCode(r.Successor) != nil || r.Successor == r.Code):
		return invalid("the successor")
	case r.Reserved == "" && r.Rematcher != "",
		r.Reserved != "" && (r.Status == store.StatusPlaying || r.Successor != "" || store.ValidateCode(r.Reserved) != nil ||
			r.Reserved == r.Code || !slices.ContainsFunc(r.Start, func(s StartSlot) bool { return s.UserID != "" && s.UserID == r.Rematcher })):
		return invalid("the reserved successor")
	}
	return nil
}

// checkWork requires exactly one current-policy row for each required bot
// seat of the open point, and none for anything else.
func checkWork(r *Record, g *game) error {
	if !slices.Equal(r.BotPresets, releasedPresets()) {
		return invalid("unsupported frozen bot presets")
	}
	p := g.Point()
	if len(r.BotFallbacks) > maxRoomLog {
		return invalid("too many bot diagnostics")
	}
	for _, f := range r.BotFallbacks {
		if f.At < 0 || f.At > len(r.Batches) || f.Point == "" || f.Seat < 0 || f.Seat >= len(r.Seats) || f.Reason != "watchdog" {
			return invalid("a bot diagnostic")
		}
		if f.At < len(r.Batches) {
			if f.Point != r.Batches[f.At].Point || !slices.ContainsFunc(r.Batches[f.At].Picks, func(p PickRecord) bool { return p.Seat == f.Seat }) {
				return invalid("a bot diagnostic for another batch")
			}
		} else if pointOver(p) || f.Point != p.ID || !p.Required.Has(seatNum(f.Seat)) {
			return invalid("a bot diagnostic for another point")
		}
	}
	var rows [MaxSeats]bool
	for _, w := range r.Work {
		switch {
		case w.Seat < 0 || w.Seat >= len(r.Seats) || r.Seats[w.Seat].Kind != SeatBot:
			return invalid("a work row for a seat no bot controls")
		case rows[w.Seat] || w.Point != p.ID || !p.Required.Has(seatNum(w.Seat)):
			return invalid("a work row for seat %d that is not for the open point", w.Seat)
		case w.Policy != policyVersion:
			return invalid("work row of seat %d has an unsupported policy", w.Seat)
		}
		if preset, ok := frozenPreset(r, r.Seats[w.Seat].Bot); !ok || w.BotPreset != preset {
			return invalid("work row of seat %d has a bad level or allowance", w.Seat)
		}
		rows[w.Seat] = true
	}
	for _, s := range p.Required.Seats() {
		if r.Seats[s].Kind == SeatBot && !rows[s] {
			return invalid("required bot seat %d has no work row", s)
		}
	}
	return nil
}

// trimSeqs keeps each user's newest capacity clients, in order. Only a
// record written under a larger capacity has more; forgetting them is
// safe, because each pending pick still keeps its own floors.
func trimSeqs(xs []SeqRecord, capacity int) []SeqRecord {
	kept := map[string]int{}
	out := make([]SeqRecord, 0, len(xs))
	for i := len(xs) - 1; i >= 0; i-- {
		if kept[xs[i].User] < capacity {
			kept[xs[i].User]++
			out = append(out, xs[i])
		}
	}
	slices.Reverse(out)
	return out
}

func validSeqs(xs []SeqRecord) bool {
	type key struct{ u, c string }
	seen := map[key]bool{}
	for _, x := range xs {
		k := key{x.User, x.Client}
		if x.Seq < 0 || x.Client == "" || seen[k] {
			return false
		}
		seen[k] = true
	}
	return true
}

func engineBatches(bs []BatchRecord) []batch {
	out := make([]batch, 0, len(bs))
	for _, b := range bs {
		eb := batch{Point: b.Point, Picks: make([]pick, 0, len(b.Picks))}
		for _, p := range b.Picks {
			if p.Seat < 0 || p.Seat >= MaxSeats {
				eb.Picks = append(eb.Picks, pick{Seat: MaxSeats, Key: p.Key})
				continue
			}
			eb.Picks = append(eb.Picks, pick{Seat: seatNum(p.Seat), Key: p.Key})
		}
		out = append(out, eb)
	}
	return out
}
