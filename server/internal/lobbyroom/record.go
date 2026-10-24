package lobbyroom

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Game is the lobby kind's stable id: the room kind, the codec's game and
// the frames' game.
const Game = "lobby"

// Schema is the record's schema, dispatched by the store's bounded header.
const Schema = 2

// recordFormat is the format this build writes. decodeRecord upgrades
// format 1 (before slots had claim epochs) and format 2 (before clients
// were remembered by tableroom.ClientKey) and refuses any other, so a
// newer build's lobby is left as it is.
const recordFormat = 3

// Seat counts: two seats start Duel (M18.shared-lobby), three to seven
// Haftgane.
const (
	MinSeats = 2
	MaxSeats = tableroom.MaxSeats
)

// Slot kinds. A reserved slot waits for one user (a rematch's original
// human): it is not filled, and only its user takes it, by joining.
const (
	SlotOpen     = "open"
	SlotReserved = "reserved"
	SlotHuman    = "human"
	SlotBot      = "bot"
)

// Bounds of the lists that grow with the lobby's life.
const (
	maxRoomLog  = 512
	maxRetired  = 32
	maxUnseated = 64
	maxNameLen  = 64
	// maxSeqUsers bounds the users with remembered seqs: the humans in
	// slots, the real admins, the temporary admin and the retired users.
	maxSeqUsers = MaxSeats + roles.MaxAdmins + 1 + maxRetired
)

// Record is a lobby's durable record, the whole schema-2 room file.
type Record struct {
	Schema        int                 `json:"schema"`
	Game          string              `json:"game"`
	Format        int                 `json:"format"`
	Code          string              `json:"code"`
	Status        store.Status        `json:"status"`
	Created       time.Time           `json:"created"`
	Updated       time.Time           `json:"updated"`
	ArchivedAt    *time.Time          `json:"archivedAt,omitempty"`
	ArchiveReason store.ArchiveReason `json:"archiveReason,omitempty"`

	SeatCount int               `json:"seatCount"`
	Options   wire.LobbyOptions `json:"options"`
	// Slots are the lobby's slots in lobby order, SeatCount of them.
	Slots []Slot `json:"slots"`
	// Roles are the lobby's roles (package roles), carried into the game.
	Roles roles.Roles `json:"roles"`
	// Unseated are the users who released their slot or were removed from
	// one, oldest first and at most maxUnseated: joining again does not
	// put them back in an open slot; an admin may.
	Unseated []string `json:"unseated"`
	// Retired are the users who lost all authority while the lobby
	// remembered their seqs (as for a table).
	Retired []string `json:"retired"`
	// Revision advances on every change; every change is public.
	Revision int         `json:"revision"`
	Seqs     []SeqRecord `json:"seqs"`
	// RoomLog is the public log of role changes, at most maxRoomLog.
	RoomLog []wire.PlatformLogEntry `json:"roomLog"`
	// Parent is the finished table whose rematch made this lobby, or "".
	// A retried rematch finds its lobby by it (Lobbies.Rematch).
	Parent string `json:"parent"`
}

// Slot is one lobby slot. Taken is the revision when its human took it,
// which ranks the candidates for the temporary admin. Epoch is the claim
// epoch of its user's device links: 0 for a rematch's reservation and the
// human who takes it (the claim a rematch hands out, carried into the
// table at Start), and the revision of the change for every other
// seating, so a claim revoked when its user lost a slot stays revoked
// when they are seated again. The creator's first slot is epoch 0 too:
// nothing hands out a claim for it.
type Slot struct {
	Kind   string `json:"kind"`
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Bot    string `json:"bot"`
	Taken  int    `json:"taken"`
	Epoch  int    `json:"epoch"`
}

// SeqRecord is one client's last accepted seq.
type SeqRecord struct {
	User   string `json:"user"`
	Client string `json:"client"`
	Seq    int    `json:"seq"`
}

func (r *Record) clone() Record {
	c := *r
	if r.ArchivedAt != nil {
		at := *r.ArchivedAt
		c.ArchivedAt = &at
	}
	c.Options = cloneOptions(r.Options)
	c.Slots = slices.Clone(r.Slots)
	c.Roles = r.Roles.Clone()
	c.Unseated = slices.Clone(r.Unseated)
	c.Retired = slices.Clone(r.Retired)
	c.Seqs = slices.Clone(r.Seqs)
	c.RoomLog = slices.Clone(r.RoomLog)
	return c
}

func cloneOptions(o wire.LobbyOptions) wire.LobbyOptions {
	c := wire.LobbyOptions{}
	if o.Hints != nil {
		h := *o.Hints
		c.Hints = &h
	}
	if o.Sides != nil {
		v := *o.Sides
		c.Sides = &v
	}
	if o.Seating != nil {
		v := *o.Seating
		c.Seating = &v
	}
	if o.FirstPlayer != nil {
		v := *o.FirstPlayer
		c.FirstPlayer = &v
	}
	if o.AllowUndo != nil {
		v := *o.AllowUndo
		c.AllowUndo = &v
	}
	if o.FinalReveal != nil {
		v := *o.FinalReveal
		c.FinalReveal = &v
	}
	return c
}

// encode returns the record's file bytes, with every slice a JSON array.
func (r *Record) encode() ([]byte, error) {
	c := r.clone()
	for _, p := range []*[]string{&c.Unseated, &c.Retired} {
		if *p == nil {
			*p = []string{}
		}
	}
	if c.Seqs == nil {
		c.Seqs = []SeqRecord{}
	}
	if c.RoomLog == nil {
		c.RoomLog = []wire.PlatformLogEntry{}
	}
	data, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// decodeRecord decodes a record and validates its header. Unknown fields
// are ignored, as for the other room files; format 1 is upgraded and any
// other format refused.
func decodeRecord(data []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("lobbyroom: decoding the record: %w", err)
	}
	if r.Format == 1 || r.Format == 2 {
		upgrade(&r)
	}
	switch {
	case r.Schema != Schema || r.Game != Game:
		return nil, fmt.Errorf("%w: not a %s schema-%d record", store.ErrGame, Game, Schema)
	case r.Format != recordFormat:
		return nil, fmt.Errorf("%w: lobby record format %d is not this build's", store.ErrGame, r.Format)
	case store.ValidateCode(r.Code) != nil || r.Updated.IsZero():
		return nil, errors.New("lobbyroom: invalid record header")
	case r.Status != store.StatusLobby && r.Status != store.StatusArchived:
		return nil, errors.New("lobbyroom: invalid record status")
	}
	return &r, nil
}

// upgrade reads a format-1 or format-2 record. Format 1's slots had no
// claim epochs: a reservation keeps epoch 0, and a human the epoch of the
// revision they took their slot at, so a rematch claim from before fails
// closed rather than outliving a removal. Both kept each client's id as
// sent, so the seq acknowledgements are re-keyed by tableroom.ClientKey,
// once: format 3 marks the keys, which ClientKey must not see again.
func upgrade(r *Record) {
	if r.Format == 1 {
		for i := range r.Slots {
			if r.Slots[i].Kind == SlotHuman {
				r.Slots[i].Epoch = r.Slots[i].Taken
			}
		}
	}
	for i := range r.Seqs {
		r.Seqs[i].Client = tableroom.ClientKey(r.Seqs[i].Client)
	}
	r.Format = recordFormat
}

type inspection struct{ r *Record }

func (i inspection) Retention() store.RetentionHeader {
	return store.RetentionHeader{Code: i.r.Code, Status: i.r.Status, Updated: i.r.Updated, ArchivedAt: i.r.ArchivedAt}
}

// Check is the preflight's verdict: a lobby has no game to replay, so it
// is compatible when it validates.
func (i inspection) Check() (bool, error) {
	if err := validate(i.r, defaultSeqClients); err != nil {
		return false, err
	}
	return false, nil
}

// Codec is the lobby kind's store codec.
func Codec() *store.Codec {
	return &store.Codec{Game: Game, Decode: func(data []byte) (store.Inspection, error) {
		r, err := decodeRecord(data)
		if err != nil {
			return nil, err
		}
		return inspection{r: r}, nil
	}}
}

var errInvalid = errors.New("lobbyroom: invalid record")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalid, fmt.Sprintf(format, args...))
}

// validate checks everything a write keeps and a restart relies on: the
// seat count and slots, the options, the roles, the bounded lists and the
// seq acknowledgements, each user's at most acks clients.
func validate(r *Record, acks int) error {
	if r.SeatCount < MinSeats || r.SeatCount > MaxSeats || len(r.Slots) != r.SeatCount {
		return invalid("seat count %d with %d slots", r.SeatCount, len(r.Slots))
	}
	if r.Revision < 0 {
		return invalid("a negative revision")
	}
	users := map[string]bool{}
	for i, s := range r.Slots {
		switch s.Kind {
		case SlotOpen:
			if s.UserID != "" || s.Name != "" || s.Bot != "" {
				return invalid("open slot %d", i)
			}
		case SlotReserved, SlotHuman:
			if s.UserID == "" || len(s.UserID) > maxNameLen || users[s.UserID] || !validName(s.Name) || s.Bot != "" {
				return invalid("slot %d", i)
			}
			users[s.UserID] = true
		case SlotBot:
			if !validLevel(s.Bot) || s.UserID != "" || s.Name != "" {
				return invalid("bot slot %d", i)
			}
		default:
			return invalid("slot %d's kind", i)
		}
		if s.Taken < 0 || s.Taken > r.Revision || s.Kind != SlotHuman && s.Taken != 0 {
			return invalid("when slot %d was taken", i)
		}
		if s.Epoch < 0 || s.Epoch > r.Revision || s.Kind != SlotHuman && s.Epoch != 0 {
			return invalid("slot %d's claim epoch", i)
		}
	}
	if !validOptions(r.Options) {
		return invalid("the options")
	}
	if r.Parent != "" && (store.ValidateCode(r.Parent) != nil || r.Parent == r.Code) {
		return invalid("the parent")
	}
	if r.Roles.Valid() != nil {
		return invalid("the roles")
	}
	if len(r.Unseated) > maxUnseated || len(r.Retired) > maxRetired || len(r.RoomLog) > maxRoomLog {
		return invalid("a list past its bound")
	}
	for i, u := range r.Unseated {
		if u == "" || slices.Index(r.Unseated, u) != i {
			return invalid("the unseated users")
		}
	}
	for i, u := range r.Retired {
		if u == "" || slices.Index(r.Retired, u) != i || hasAuthority(r, u) {
			return invalid("the retired users")
		}
	}
	if !validSeqs(r.Seqs) || !seqsBounded(r.Seqs, acks, maxSeqUsers) {
		return invalid("the seq acknowledgements")
	}
	for _, x := range r.Seqs {
		if !hasAuthority(r, x.User) && !slices.Contains(r.Retired, x.User) {
			return invalid("the seq acknowledgements")
		}
	}
	return nil
}

func validName(name string) bool {
	return name != "" && len(name) <= maxNameLen && utf8.ValidString(name) && strings.TrimSpace(name) == name
}

func validLevel(level string) bool {
	switch level {
	case tableroom.BotEasy, tableroom.BotNormal, tableroom.BotHard:
		return true
	}
	return false
}

func validOptions(o wire.LobbyOptions) bool {
	if o.Hints != nil && o.Hints.AdvisorStrength != "" && !o.Hints.AdvisorStrength.Valid() {
		return false
	}
	if o.Sides != nil && *o.Sides != "choose" && *o.Sides != "day" && *o.Sides != "random" {
		return false
	}
	if o.Seating != nil && *o.Seating != "random" && *o.Seating != "lobby" {
		return false
	}
	return o.FirstPlayer == nil || *o.FirstPlayer == 0 || *o.FirstPlayer == 1
}

// hasAuthority reports whether u may act in the lobby: a human in a slot
// or an admin (real or temporary).
func hasAuthority(r *Record, u string) bool {
	return u != "" && (slotOf(r, u, SlotHuman) >= 0 || r.Roles.IsAdmin(u))
}

// slotOf returns the slot of kind that u holds, or -1.
func slotOf(r *Record, u, kind string) int {
	return slices.IndexFunc(r.Slots, func(s Slot) bool { return s.Kind == kind && s.UserID == u })
}

func validSeqs(xs []SeqRecord) bool {
	type key struct{ u, c string }
	seen := map[key]bool{}
	for _, x := range xs {
		k := key{x.User, x.Client}
		if x.Seq < 0 || x.Client == "" || x.User == "" || seen[k] {
			return false
		}
		seen[k] = true
	}
	return true
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

// seqLast returns the client's remembered seq and whether it is there; a
// lookup never reorders the list.
func seqLast(xs []SeqRecord, user, client string) (int, bool) {
	for _, x := range xs {
		if x.User == user && x.Client == client {
			return x.Seq, true
		}
	}
	return 0, false
}

// seqPut records the client's last accepted seq, moving it to the end;
// beyond capacity it forgets the same user's client whose last accepted
// seq is the oldest, so no user's activity changes another user's
// answers (the table's rule, tableroom's seqPut).
func seqPut(xs *[]SeqRecord, k SeqRecord, capacity int) {
	*xs = slices.DeleteFunc(*xs, func(x SeqRecord) bool { return x.User == k.User && x.Client == k.Client })
	*xs = append(*xs, k)
	n := 0
	for _, x := range *xs {
		if x.User == k.User {
			n++
		}
	}
	if n > capacity {
		i := slices.IndexFunc(*xs, func(x SeqRecord) bool { return x.User == k.User })
		*xs = slices.Delete(*xs, i, i+1)
	}
}

// retire keeps the retired list after a change from before to after, as
// the table does: users who lost all authority join it in user order,
// users who regained it leave it, and beyond maxRetired the oldest is
// forgotten with its seqs.
func retire(before, after *Record) {
	after.Retired = slices.DeleteFunc(after.Retired, func(u string) bool { return hasAuthority(after, u) })
	var lost []string
	for _, u := range authorities(before) {
		if !hasAuthority(after, u) && !slices.Contains(after.Retired, u) && !slices.Contains(lost, u) {
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

func authorities(r *Record) []string {
	var out []string
	for _, s := range r.Slots {
		if s.Kind == SlotHuman {
			out = append(out, s.UserID)
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

// unseat records that u left a slot by releasing it or being removed.
func unseat(r *Record, u string) {
	r.Unseated = slices.DeleteFunc(r.Unseated, func(x string) bool { return x == u })
	r.Unseated = append(r.Unseated, u)
	if over := len(r.Unseated) - maxUnseated; over > 0 {
		r.Unseated = slices.Delete(r.Unseated, 0, over)
	}
}

func appendLog(r *Record, entries ...wire.PlatformLogEntry) {
	r.RoomLog = append(r.RoomLog, entries...)
	if over := len(r.RoomLog) - maxRoomLog; over > 0 {
		r.RoomLog = slices.Delete(r.RoomLog, 0, over)
	}
}
