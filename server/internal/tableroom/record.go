package tableroom

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

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
	Schema        int                 `json:"schema"`
	Game          string              `json:"game"`
	Rules         int                 `json:"rules"`
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
	// Original is each seat's original participant: the first human to
	// control it, or "" while it has had none (a starting bot, or an open
	// seat not yet claimed).
	Original []string `json:"original"`
	// Seats are the current controllers, one per seat.
	Seats []SeatRecord `json:"seats"`
	// Roles are the table's roles. This slice has only the creator, who is
	// the one admin; M16.haftgane-lobby-and-roles adds the rest of D42.
	Roles Roles `json:"roles"`
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
	// may see. Edits counts each seat's private edits; a viewer's frame
	// revision is Public plus its own seat's Edits, so a private edit
	// advances only its owner's frames.
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
	Work []WorkRow `json:"work"`
	// RoomLog is the public log of seat switches and role changes (empty
	// until M16.haftgane-lobby-and-roles).
	RoomLog []wire.PlatformLogEntry `json:"roomLog"`
}

// SetupRecord is the stored setup.
type SetupRecord struct {
	Seed          uint64 `json:"seed"`
	Seats         int    `json:"seats"`
	RandomSeating bool   `json:"randomSeating"`
	HideAtEnd     bool   `json:"hideAtEnd"`
}

// Seat controller kinds.
const (
	SeatOpen  = "open"
	SeatHuman = "human"
	SeatBot   = "bot"
)

// SeatRecord is one seat's current controller. An open seat is a human
// seat no user has claimed yet; the next user to join without a seat
// claims it, as an online Duel room's open seats are claimed today.
type SeatRecord struct {
	Seat   int    `json:"seat"`
	Kind   string `json:"kind"`
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Bot    string `json:"bot"`
}

// Roles are the table's roles.
type Roles struct {
	Creator string        `json:"creator"`
	Admins  []AdminRecord `json:"admins"`
}

// AdminRecord is one admin. PromotedBy is nil for the creator.
type AdminRecord struct {
	UserID     string  `json:"userId"`
	PromotedBy *string `json:"promotedBy,omitempty"`
}

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
	Seat      int    `json:"seat"`
	Point     string `json:"point"`
	Policy    string `json:"policy"`
	Level     string `json:"level"`
	Allowance int    `json:"allowance"`
}

// Bot levels. The lowest released level is the default for a bot seat.
const (
	BotEasy   = "easy"
	BotNormal = "normal"
	BotHard   = "hard"
	// DefaultBotLevel is the lowest released level.
	DefaultBotLevel = BotEasy
)

// allowance is a level's fixed deterministic work allowance.
func allowance(level string) (int, bool) {
	switch level {
	case BotEasy:
		return 1, true
	case BotNormal:
		return 2, true
	case BotHard:
		return 4, true
	}
	return 0, false
}

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
	c.Roles.Admins = make([]AdminRecord, 0, len(r.Roles.Admins))
	for _, a := range r.Roles.Admins {
		if a.PromotedBy != nil {
			by := *a.PromotedBy
			a.PromotedBy = &by
		}
		c.Roles.Admins = append(c.Roles.Admins, a)
	}
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
	c.RoomLog = slices.Clone(r.RoomLog)
	return c
}

// encode returns the record's file bytes, with every slice a JSON array.
func (r *Record) encode() ([]byte, error) {
	c := r.clone()
	for _, p := range []*[]PendingSlot{&c.Pending} {
		for i := range *p {
			if (*p)[i].Seqs == nil {
				(*p)[i].Seqs = []SeqRecord{}
			}
		}
	}
	data, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// decodeRecord decodes a record and validates its header. Unknown fields
// are ignored, as for Duel's file, so an additive field of a newer build
// of the same rules does not make the room unreadable.
func decodeRecord(data []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("tableroom: decoding the record: %w", err)
	}
	if r.Schema != Schema || r.Game != Game {
		return nil, fmt.Errorf("%w: not a %s schema-%d record", store.ErrGame, Game, Schema)
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
	su := makeSetup(r.Setup.Seed, n, r.Setup.RandomSeating, r.Setup.HideAtEnd)
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
	su := makeSetup(r.Setup.Seed, n, r.Setup.RandomSeating, r.Setup.HideAtEnd)
	order := seatOrder(su.Seed, su.Seats, su.Options.Seating)
	if len(r.Assignment) != n || len(r.Original) != n || len(r.Seats) != n || len(r.Edits) != n {
		return invalid("per-seat fields do not hold %d seats", n)
	}
	for s := range n {
		if r.Assignment[s] != int(order[s]) {
			return invalid("the assignment of seat %d", s)
		}
		if r.Edits[s] < 0 {
			return invalid("the edit count of seat %d", s)
		}
	}
	if err := checkSeats(r); err != nil {
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
		case !validSeqs(ps.Seqs) || !seqsBounded(ps.Seqs, b.floors, n):
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
	if !validSeqs(r.Seqs) || !seqsBounded(r.Seqs, b.acks, n) {
		return invalid("the seq acknowledgements")
	}
	return checkWork(r, g)
}

// seqsBounded reports whether xs holds at most capacity clients of each
// user, and no more users than there are seats (only a seated user's
// requests are accepted).
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
		case SeatOpen:
			if sr.UserID != "" || sr.Bot != "" {
				return invalid("open seat %d has a controller", s)
			}
		case SeatHuman:
			if sr.UserID == "" || sr.Bot != "" || users[sr.UserID] {
				return invalid("human seat %d", s)
			}
			users[sr.UserID] = true
		case SeatBot:
			if _, ok := allowance(sr.Bot); !ok || sr.UserID != "" {
				return invalid("bot seat %d", s)
			}
		default:
			return invalid("seat %d's kind", s)
		}
	}
	admins := map[string]bool{}
	for _, a := range r.Roles.Admins {
		if a.UserID == "" || admins[a.UserID] {
			return invalid("the admins")
		}
		admins[a.UserID] = true
	}
	return nil
}

// checkWork requires exactly one current-policy row for each required bot
// seat of the open point, and none for anything else.
func checkWork(r *Record, g *game) error {
	p := g.Point()
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
		if a, ok := allowance(w.Level); !ok || a != w.Allowance {
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
