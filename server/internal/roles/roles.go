// Package roles holds the admin roles of D42, shared by the shared lobby
// and the Haftgane table, which carries the lobby's roles on at Start
// (docs/haftgane/platform-architecture.md, "Admin roles", "Temporary
// admin", "Roles across the room's life").
//
// Roles is a plain value that the room records persist. Every operation is
// a pure function of that value, the time the caller passes in and the
// presence the caller reports, so the actors run them inside their
// transactions and the tests drive them with any clock.
//
// The creator is permanent. Other real admins may step down. Real admins
// promote humans permanently and demote their own promotees (the creator
// anyone except itself). Temporary admins may promote temporary admins
// and demote their own temporary promotees. Every temporary role ends
// when a real admin returns. A bounded recency history prefers former
// temporary admins; Candidates gives the fallback a restart-stable order.
// All real admins must have been away for AwayLimit before a temporary
// team is made, and all temporary admins away that long before replacement.
//
// A room saves each change the rule makes. When its store refuses one,
// the room keeps the unsaved roles (Carry starts the next check from
// them) and checks again after Retry, and the temporary role fails
// closed meanwhile: Holds grants it only to a user whom both the saved
// roles and the rule, checked now, name. So a refused write can delay a
// change, but never keeps a demoted or replaced temporary admin's powers.
//
// Away times start when a room sees an admin's last connection drop. An
// admin still connected when the server stopped has no away time, so
// their absence counts from the room's next check after the restart,
// not from the shutdown; one already away keeps their time.
package roles

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// AwayLimit is how long every real admin must have been away before a
// temporary admin is made, and how long a temporary admin may be away
// before the role passes on.
const AwayLimit = 5 * time.Minute

// MaxAdmins caps new real promotions. A legacy creator restored by Upgrade
// can occupy the final authority slot without removing old real grants.
const MaxAdmins = 32

// MaxAuthorities preserves the existing room and Duel handoff bound.
const MaxAuthorities = MaxAdmins + 1

// MaxHistory bounds distinct former temporary admins, oldest first.
const MaxHistory = 64

const maxUserIDBytes = 64

// Retry is how soon a room checks its roles again after its store
// refused a role change.
const Retry = 5 * time.Second

// Log entry roles.
const (
	RoleAdmin     = "admin"
	RoleTemporary = "temporary"
	RoleHuman     = "human"
)

// Errors of the role operations, each a reject code.
var (
	// ErrNotAdmin: the sender may not make this change.
	ErrNotAdmin = errors.New("not-admin")
	// ErrForbidden: nobody may make this change (the creator's demotion,
	// the creator stepping down, too many admins).
	ErrForbidden = errors.New("forbidden")
	// ErrBadRequest: the target is not someone the change applies to.
	ErrBadRequest = errors.New("bad-request")
)

// Admin is one real admin. PromotedBy is nil for the creator.
type Admin struct {
	UserID     string  `json:"userId"`
	PromotedBy *string `json:"promotedBy,omitempty"`
}

// Away is the time a real or temporary admin's last connection left.
type Away struct {
	UserID string    `json:"userId"`
	Since  time.Time `json:"since"`
}

// Roles are a room's roles, persisted with it.
type Roles struct {
	Creator string  `json:"creator"`
	Admins  []Admin `json:"admins"`
	// Temporary is the temporary admin, or "".
	Temporary   string  `json:"temporary"`
	TemporaryBy *string `json:"temporaryBy,omitempty"`
	// Deputies are the temporary admins promoted by other temporary admins.
	Deputies []Admin `json:"deputies"`
	// History is temporary-admin recency, oldest first, one row per user.
	History []string `json:"history"`
	// Declined excludes temporary admins who stepped down until they disconnect.
	Declined []string `json:"declined"`
	// Elections advances only when an automatic grant is persisted.
	Elections uint64 `json:"elections"`
	// Away holds, for each real admin and the temporary admin whose
	// connections are all absent, the time the last one left.
	Away []Away `json:"away"`
}

// New returns the roles of a new room: the creator, if any, is its one
// real admin.
func New(creator string) Roles {
	r := Roles{Creator: creator, Admins: []Admin{}, Deputies: []Admin{}, History: []string{}, Declined: []string{}, Away: []Away{}}
	if creator != "" {
		r.Admins = append(r.Admins, Admin{UserID: creator})
	}
	return r
}

// Clone returns a deep copy.
func (r Roles) Clone() Roles {
	c := r
	if r.TemporaryBy != nil {
		by := *r.TemporaryBy
		c.TemporaryBy = &by
	}
	c.Admins = make([]Admin, 0, len(r.Admins))
	for _, a := range r.Admins {
		if a.PromotedBy != nil {
			by := *a.PromotedBy
			a.PromotedBy = &by
		}
		c.Admins = append(c.Admins, a)
	}
	c.Deputies = cloneAdmins(r.Deputies)
	c.History = append([]string{}, r.History...)
	c.Declined = append([]string{}, r.Declined...)
	c.Away = slices.Clone(r.Away)
	if c.Away == nil {
		c.Away = []Away{}
	}
	return c
}

// Real reports whether u is a real admin.
func (r *Roles) Real(u string) bool { return r.admin(u) >= 0 }

// IsAdmin reports whether u has the admin powers: a real admin or the
// temporary admin.
func (r *Roles) IsAdmin(u string) bool { return u != "" && (r.Real(u) || r.Temp(u)) }

// Temp reports whether u has a temporary role.
func (r *Roles) Temp(u string) bool { return u != "" && (r.Temporary == u || r.deputy(u) >= 0) }

func (r *Roles) deputy(u string) int {
	return slices.IndexFunc(r.Deputies, func(a Admin) bool { return a.UserID == u })
}

func cloneAdmins(as []Admin) []Admin {
	out := make([]Admin, 0, len(as))
	for _, a := range as {
		if a.PromotedBy != nil {
			by := *a.PromotedBy
			a.PromotedBy = &by
		}
		out = append(out, a)
	}
	return out
}

// PromotedBy returns who promoted u, or nil.
func (r *Roles) PromotedBy(u string) *string {
	if i := r.admin(u); i >= 0 && r.Admins[i].PromotedBy != nil {
		by := *r.Admins[i].PromotedBy
		return &by
	}
	if r.Temporary == u && r.TemporaryBy != nil {
		by := *r.TemporaryBy
		return &by
	}
	if i := r.deputy(u); i >= 0 && r.Deputies[i].PromotedBy != nil {
		by := *r.Deputies[i].PromotedBy
		return &by
	}
	return nil
}

func (r *Roles) admin(u string) int {
	return slices.IndexFunc(r.Admins, func(a Admin) bool { return a.UserID == u })
}

func entry(at int, by, user, role string) wire.PlatformLogEntry {
	u, ro := user, role
	return wire.PlatformLogEntry{Kind: "role", At: at, By: by, UserID: &u, Role: &ro}
}

// HasRoom reports whether the admin cap leaves room for one more admin.
func (r *Roles) HasRoom(by ...string) bool {
	if len(r.Admins)+len(r.Deputies)+boolInt(r.Temporary != "") >= MaxAuthorities {
		return false
	}
	return len(by) > 0 && r.Temp(by[0]) || len(r.Admins) < MaxAdmins
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// Promote grants a real role for a real promoter, a temporary role for a
// temporary promoter. The total authority cap also bounds temporary grants.
func (r *Roles) Promote(by, target string, inRoom bool, at int) (wire.PlatformLogEntry, error) {
	switch {
	case !r.IsAdmin(by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	case target == "" || len(target) > maxUserIDBytes || !inRoom || r.IsAdmin(target):
		return wire.PlatformLogEntry{}, ErrBadRequest
	case !r.HasRoom(by):
		return wire.PlatformLogEntry{}, ErrForbidden
	}
	p := by
	a := Admin{UserID: target, PromotedBy: &p}
	if r.Temp(by) {
		r.Deputies = append(r.Deputies, a)
		r.remember(target)
		return entry(at, by, target, RoleTemporary), nil
	}
	r.Admins = append(r.Admins, a)
	return entry(at, by, target, RoleAdmin), nil
}

// Demote enforces direct promotion ancestry. A temporary admin can only
// demote a temporary admin they promoted, never a real admin or the root.
// Demoting a promoter leaves its promotees in place.
func (r *Roles) Demote(by, target string, at int) (wire.PlatformLogEntry, error) {
	switch {
	case !r.IsAdmin(by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	case target == r.Creator && target != "":
		return wire.PlatformLogEntry{}, ErrForbidden
	case !r.IsAdmin(target):
		return wire.PlatformLogEntry{}, ErrBadRequest
	}
	if i := r.deputy(target); i >= 0 {
		if by != r.Creator && (r.Deputies[i].PromotedBy == nil || *r.Deputies[i].PromotedBy != by) {
			return wire.PlatformLogEntry{}, ErrNotAdmin
		}
		r.Deputies = slices.Delete(r.Deputies, i, i+1)
	} else if i := r.admin(target); i >= 0 {
		if !r.Real(by) || by != r.Creator && (r.Admins[i].PromotedBy == nil || *r.Admins[i].PromotedBy != by) {
			return wire.PlatformLogEntry{}, ErrNotAdmin
		}
		r.Admins = slices.Delete(r.Admins, i, i+1)
	} else {
		if by != r.Creator && (r.TemporaryBy == nil || *r.TemporaryBy != by) {
			return wire.PlatformLogEntry{}, ErrNotAdmin
		}
		r.removeRoot()
	}
	r.pruneAway()
	return entry(at, by, target, RoleHuman), nil
}

// StepDown removes an admin's role, except the permanent creator. A
// temporary root hands its place to its oldest remaining deputy. With no
// admins left, the normal temporary election applies on the next check.
func (r *Roles) StepDown(by string, at int) (wire.PlatformLogEntry, error) {
	switch {
	case by == r.Creator && by != "":
		return wire.PlatformLogEntry{}, ErrForbidden
	case !r.IsAdmin(by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	}
	if i := r.admin(by); i >= 0 {
		r.Admins = slices.Delete(r.Admins, i, i+1)
	} else if i := r.deputy(by); i >= 0 {
		r.Deputies = slices.Delete(r.Deputies, i, i+1)
	} else {
		r.removeRoot()
	}
	r.Declined = slices.DeleteFunc(r.Declined, func(u string) bool { return u == by })
	r.Declined = append(r.Declined, by)
	if len(r.Declined) > MaxHistory {
		r.Declined = slices.Delete(r.Declined, 0, len(r.Declined)-MaxHistory)
	}
	r.pruneAway()
	return entry(at, by, by, RoleHuman), nil
}

func (r *Roles) removeRoot() {
	r.Temporary, r.TemporaryBy = "", nil
	if len(r.Deputies) > 0 {
		a := r.Deputies[0]
		r.Temporary, r.TemporaryBy = a.UserID, a.PromotedBy
		r.Deputies = slices.Delete(r.Deputies, 0, 1)
	}
}

func (r *Roles) pruneAway() {
	r.Away = slices.DeleteFunc(r.Away, func(w Away) bool { return !r.IsAdmin(w.UserID) })
}

func (r *Roles) remember(user string) {
	r.History = slices.DeleteFunc(r.History, func(u string) bool { return u == user })
	r.History = append(r.History, user)
	if len(r.History) > MaxHistory {
		r.History = slices.Delete(r.History, 0, len(r.History)-MaxHistory)
	}
}

func (r *Roles) temps() []string {
	out := []string{}
	if r.Temporary != "" {
		out = append(out, r.Temporary)
	}
	for _, a := range r.Deputies {
		out = append(out, a.UserID)
	}
	return out
}

func (r *Roles) clearTemps(at int) []wire.PlatformLogEntry {
	var log []wire.PlatformLogEntry
	for _, u := range r.temps() {
		log = append(log, entry(at, "", u, RoleHuman))
	}
	r.Temporary, r.TemporaryBy, r.Deputies = "", nil, []Admin{}
	r.pruneAway()
	return log
}

// Candidates orders connected humans by a hash of public room code,
// durable election count and user id, independently within the two pools.
// Candidate order and connection arrival order cannot change a replay.
func Candidates(code string, election uint64, seated, spectators []string) []string {
	order := func(pool []string) []string {
		out := append([]string{}, pool...)
		rank := func(u string) [32]byte {
			data := []byte(code + "\x00")
			data = binary.BigEndian.AppendUint64(data, election)
			return sha256.Sum256(append(data, []byte(u)...))
		}
		slices.SortFunc(out, func(a, b string) int { x, y := rank(a), rank(b); return bytes.Compare(x[:], y[:]) })
		return slices.Compact(out)
	}
	return append(order(seated), order(spectators)...)
}

// Check applies the five-minute rule. candidates are hash-ordered seated
// humans then spectators (Candidates); connected former temporary admins
// take priority over both pools, most recent first.
func (r *Roles) Check(now time.Time, connected func(string) bool, candidates []string, at int) ([]wire.PlatformLogEntry, time.Time) {
	var log []wire.PlatformLogEntry
	r.Declined = slices.DeleteFunc(r.Declined, func(u string) bool { return !connected(u) })
	r.sync(now, connected)
	realHere := slices.ContainsFunc(r.Admins, func(a Admin) bool { return connected(a.UserID) })
	if realHere {
		log = r.clearTemps(at)
		r.sync(now, connected)
		return log, time.Time{}
	}
	// A legacy full roster can gain its restored creator as the 33rd real
	// admin. Keep the durable cap: only a real admin can free room for a
	// temporary grant by demotion or step-down.
	if len(r.Admins) >= MaxAuthorities {
		return log, time.Time{}
	}
	var allAway time.Time
	for _, a := range r.Admins {
		if s := r.since(a.UserID); s.After(allAway) {
			allAway = s
		}
	}
	if len(r.Admins) > 0 {
		allAway = allAway.Add(AwayLimit)
	}
	if now.Before(allAway) {
		log = r.clearTemps(at)
		r.sync(now, connected)
		return log, allAway
	}
	var due time.Time
	for _, u := range r.temps() {
		if connected(u) {
			return log, time.Time{}
		}
		if t := r.since(u).Add(AwayLimit); t.After(due) {
			due = t
		}
	}
	if now.Before(due) {
		return log, due
	}
	eligible := func(u string) bool {
		return u != "" && !r.Real(u) && !slices.Contains(r.Declined, u) && connected(u) && slices.Contains(candidates, u)
	}
	selected := ""
	for i := len(r.History) - 1; i >= 0; i-- {
		if eligible(r.History[i]) {
			selected = r.History[i]
			break
		}
	}
	if selected == "" {
		for _, u := range candidates {
			if eligible(u) {
				selected = u
				break
			}
		}
	}
	if selected == "" {
		return log, time.Time{}
	}
	log = r.clearTemps(at)
	r.Temporary = selected
	r.remember(selected)
	r.Elections++
	log = append(log, entry(at, "", selected, RoleTemporary))
	r.sync(now, connected)
	return log, time.Time{}
}

// Carry brings onto r, the saved roles, what ahead (the roles of a later
// check whose write the store refused, or nil) holds that r lacks: its
// temporary admin, unless a later saved change made them a real admin,
// and its away times for admins r has none for. The result is valid
// roles. It returns the log entries of the temporary change.
func (r *Roles) Carry(ahead *Roles, at int) []wire.PlatformLogEntry {
	if ahead == nil {
		return nil
	}
	var log []wire.PlatformLogEntry
	// A refused disconnect can clear an exclusion. Preserve a step-down
	// added since ahead was made, identified by ahead still holding its role.
	r.Declined = slices.DeleteFunc(r.Declined, func(u string) bool { return !slices.Contains(ahead.Declined, u) && !ahead.IsAdmin(u) })
	// Only Check makes ahead, so it cannot change real promotions. Filter
	// out users since made real; carry the whole temporary team and history.
	if ahead.Elections > r.Elections || ahead.Temporary == "" && r.Temporary != "" {
		ts := ahead.temps()
		ts = slices.DeleteFunc(ts, r.Real)
		for _, u := range r.temps() {
			if !slices.Contains(ts, u) {
				log = append(log, entry(at, "", u, RoleHuman))
			}
		}
		for _, u := range ts {
			if !r.Temp(u) {
				log = append(log, entry(at, "", u, RoleTemporary))
			}
		}
		r.Temporary, r.TemporaryBy, r.Deputies = "", nil, []Admin{}
		if len(ts) > 0 {
			r.Temporary = ts[0]
			r.TemporaryBy = ahead.PromotedBy(ts[0])
			for _, a := range ahead.Deputies {
				if a.UserID != r.Temporary && !r.Real(a.UserID) {
					r.Deputies = append(r.Deputies, cloneAdmins([]Admin{a})...)
				}
			}
		}
		r.History = append([]string{}, ahead.History...)
		r.Elections = ahead.Elections
	}
	away := make([]Away, 0, len(r.Away)+len(ahead.Away))
	// ahead records returns as well as departures. Do not resurrect an old
	// absence after a refused return followed by another disconnect.
	for _, w := range r.Away {
		if r.IsAdmin(w.UserID) && !ahead.IsAdmin(w.UserID) {
			away = append(away, w)
		}
	}
	for _, w := range ahead.Away {
		if r.IsAdmin(w.UserID) {
			away = append(away, w)
		}
	}
	r.Away = away
	return log
}

// Holds reports whether u has the admin powers at now: a real admin, or
// the temporary admin of r, the saved roles, whom the rule still keeps
// when checked now from r and ahead (Carry). A temporary change the
// store refused thus takes the powers from the user it demoted or
// replaced at once, and gives them to their successor only once saved.
func (r *Roles) Holds(u string, ahead *Roles, now time.Time, connected func(string) bool, candidates []string) bool {
	if r.Real(u) {
		return true
	}
	if u == "" || !r.Temp(u) {
		return false
	}
	live := r.Clone()
	live.Carry(ahead, 0)
	live.Check(now, connected, candidates, 0)
	return live.Temp(u) && live.Elections == r.Elections
}

// sync keeps one Away row for each real or temporary admin who is not
// connected, stamped when first seen away, and none for anyone else.
func (r *Roles) sync(now time.Time, connected func(string) bool) {
	keep := r.Away[:0:0]
	track := func(u string) {
		if u == "" || connected(u) || slices.ContainsFunc(keep, func(a Away) bool { return a.UserID == u }) {
			return
		}
		since := now
		if i := slices.IndexFunc(r.Away, func(a Away) bool { return a.UserID == u }); i >= 0 {
			since = r.Away[i].Since
		}
		keep = append(keep, Away{UserID: u, Since: since})
	}
	for _, a := range r.Admins {
		track(a.UserID)
	}
	for _, u := range r.temps() {
		track(u)
	}
	r.Away = keep
}

func (r *Roles) since(u string) time.Time {
	if i := slices.IndexFunc(r.Away, func(a Away) bool { return a.UserID == u }); i >= 0 {
		return r.Away[i].Since
	}
	return time.Time{}
}

// Upgrade initializes format-5 role metadata without changing old grants.
// A creator who stepped down under the old policy regains the permanent role.
func (r *Roles) Upgrade() {
	if r.Creator != "" && !r.Real(r.Creator) {
		r.Admins = append([]Admin{{UserID: r.Creator}}, r.Admins...)
		// Legacy step-down allowed a full real-admin list without the creator.
		// Restore every real grant; defer the temporary role if the list is full.
		if len(r.Admins) == MaxAuthorities {
			r.Temporary = ""
		}
	}
	r.Deputies = []Admin{}
	r.History = []string{}
	r.Declined = []string{}
	r.TemporaryBy = nil
	if r.Temporary != "" {
		if r.Real(r.Temporary) {
			r.Temporary = ""
		} else {
			r.remember(r.Temporary)
		}
	}
	r.pruneAway()
}

func (r *Roles) Valid() error {
	bad := errors.New("roles: invalid")
	validID := func(u string) bool { return u != "" && len(u) <= maxUserIDBytes }
	if len(r.Admins)+len(r.Deputies)+boolInt(r.Temporary != "") > MaxAuthorities || len(r.History) > MaxHistory || len(r.Declined) > MaxHistory ||
		r.Creator != "" && (!validID(r.Creator) || !r.Real(r.Creator)) {
		return bad
	}
	for i, a := range r.Admins {
		if !validID(a.UserID) || r.admin(a.UserID) != i || a.PromotedBy != nil && !validID(*a.PromotedBy) || (a.PromotedBy == nil && a.UserID != r.Creator) {
			return bad
		}
	}
	if r.Temporary != "" && (!validID(r.Temporary) || r.Real(r.Temporary)) || r.Temporary == "" && len(r.Deputies) > 0 {
		return bad
	}
	if r.TemporaryBy != nil && (r.Temporary == "" || !validID(*r.TemporaryBy)) {
		return bad
	}
	for i, a := range r.Deputies {
		if !validID(a.UserID) || a.UserID == r.Temporary || r.Real(a.UserID) || r.deputy(a.UserID) != i || a.PromotedBy == nil || !validID(*a.PromotedBy) {
			return bad
		}
	}
	for i, u := range r.Declined {
		if !validID(u) || slices.Index(r.Declined, u) != i {
			return bad
		}
	}
	for i, u := range r.History {
		if !validID(u) || slices.Index(r.History, u) != i {
			return bad
		}
	}
	for i, w := range r.Away {
		if !r.IsAdmin(w.UserID) || slices.IndexFunc(r.Away, func(x Away) bool { return x.UserID == w.UserID }) != i || w.Since.IsZero() {
			return bad
		}
	}
	return nil
}

// Equal compares all durable role metadata, including silent changes.
func (r *Roles) Equal(o *Roles) bool {
	sameAdmin := func(a, b Admin) bool { return a.UserID == b.UserID && sameString(a.PromotedBy, b.PromotedBy) }
	return r.Creator == o.Creator && r.Temporary == o.Temporary && sameString(r.TemporaryBy, o.TemporaryBy) && r.Elections == o.Elections &&
		slices.EqualFunc(r.Admins, o.Admins, sameAdmin) && slices.EqualFunc(r.Deputies, o.Deputies, sameAdmin) && slices.Equal(r.History, o.History) && slices.Equal(r.Declined, o.Declined) &&
		slices.EqualFunc(r.Away, o.Away, func(a, b Away) bool { return a.UserID == b.UserID && a.Since.Equal(b.Since) })
}
func sameString(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
