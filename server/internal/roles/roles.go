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
// The real admins are the creator and every human an admin promoted. Any
// real admin may promote any human in the room; an admin may demote the
// admins they promoted, the creator anyone, and nobody the creator. Any
// real admin may step down. Only once every real admin has been
// disconnected for AwayLimit does a temporary admin take the role, chosen
// by the caller's candidate order; a real admin's return demotes them at
// once, and a temporary admin away for AwayLimit passes the role on. A
// temporary admin has the seat powers, the lobby's edits and Start, but
// cannot promote or demote. With no real admin at all (the last one
// stepped down, or a table made without a creator), the rule applies at
// once.
//
// A room saves each change the rule makes. When its store refuses one,
// the room keeps the unsaved roles (Carry starts the next check from
// them) and checks again after Retry, and the temporary role fails
// closed meanwhile: Holds grants it only to a user whom both the saved
// roles and the rule, checked now, name. So a refused write can delay a
// change, but never keeps a demoted or replaced temporary admin's powers.
package roles

import (
	"errors"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// AwayLimit is how long every real admin must have been away before a
// temporary admin is made, and how long a temporary admin may be away
// before the role passes on.
const AwayLimit = 5 * time.Minute

// MaxAdmins bounds the real admins of one room.
const MaxAdmins = 32

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
	// a temporary admin stepping down, too many admins).
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
	Temporary string `json:"temporary"`
	// Away holds, for each real admin and the temporary admin whose
	// connections are all absent, the time the last one left.
	Away []Away `json:"away"`
}

// New returns the roles of a new room: the creator, if any, is its one
// real admin.
func New(creator string) Roles {
	r := Roles{Creator: creator, Admins: []Admin{}, Away: []Away{}}
	if creator != "" {
		r.Admins = append(r.Admins, Admin{UserID: creator})
	}
	return r
}

// Clone returns a deep copy.
func (r Roles) Clone() Roles {
	c := r
	c.Admins = make([]Admin, 0, len(r.Admins))
	for _, a := range r.Admins {
		if a.PromotedBy != nil {
			by := *a.PromotedBy
			a.PromotedBy = &by
		}
		c.Admins = append(c.Admins, a)
	}
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
func (r *Roles) IsAdmin(u string) bool { return u != "" && (r.Real(u) || r.Temporary == u) }

// PromotedBy returns who promoted u, or nil.
func (r *Roles) PromotedBy(u string) *string {
	if i := r.admin(u); i >= 0 && r.Admins[i].PromotedBy != nil {
		by := *r.Admins[i].PromotedBy
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

// Promote makes target a real admin, promoted by by. Only a real admin may
// promote, and only a human in the room (seated or spectating).
func (r *Roles) Promote(by, target string, inRoom bool, at int) (wire.PlatformLogEntry, error) {
	switch {
	case !r.Real(by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	case target == "" || !inRoom || r.Real(target):
		return wire.PlatformLogEntry{}, ErrBadRequest
	case len(r.Admins) >= MaxAdmins:
		return wire.PlatformLogEntry{}, ErrForbidden
	}
	p := by
	r.Admins = append(r.Admins, Admin{UserID: target, PromotedBy: &p})
	if r.Temporary == target {
		r.Temporary = ""
	}
	return entry(at, by, target, RoleAdmin), nil
}

// Demote removes target's real admin role. The creator may demote anyone
// else; another admin only those they promoted. The admins target promoted
// stay admins.
func (r *Roles) Demote(by, target string, at int) (wire.PlatformLogEntry, error) {
	i := r.admin(target)
	switch {
	case !r.Real(by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	case i < 0:
		return wire.PlatformLogEntry{}, ErrBadRequest
	case target == r.Creator:
		return wire.PlatformLogEntry{}, ErrForbidden
	case by != r.Creator && (r.Admins[i].PromotedBy == nil || *r.Admins[i].PromotedBy != by):
		return wire.PlatformLogEntry{}, ErrNotAdmin
	}
	r.Admins = slices.Delete(r.Admins, i, i+1)
	return entry(at, by, target, RoleHuman), nil
}

// StepDown removes by's own real admin role, the creator's included. A
// temporary admin cannot step down: the role passes on when they leave.
func (r *Roles) StepDown(by string, at int) (wire.PlatformLogEntry, error) {
	i := r.admin(by)
	switch {
	case i >= 0:
	case r.Temporary == by && by != "":
		return wire.PlatformLogEntry{}, ErrForbidden
	default:
		return wire.PlatformLogEntry{}, ErrNotAdmin
	}
	r.Admins = slices.Delete(r.Admins, i, i+1)
	return entry(at, by, by, RoleHuman), nil
}

// Check brings the roles up to date at now: it records when each real or
// temporary admin went away (connected reports presence), demotes the
// temporary admin once a real admin is connected, and makes or passes on
// the temporary role under the five-minute rule. candidates are the
// connected humans in the order the rule prefers them: seated humans by
// how long they have held their seat, then spectators by how long they
// have been connected. It returns the log entries of the changes and the
// next time the rule could change something by the clock alone (zero:
// never, until presence or roles change).
func (r *Roles) Check(now time.Time, connected func(string) bool, candidates []string, at int) ([]wire.PlatformLogEntry, time.Time) {
	var log []wire.PlatformLogEntry
	r.sync(now, connected)
	realHere := slices.ContainsFunc(r.Admins, func(a Admin) bool { return connected(a.UserID) })
	if realHere {
		if r.Temporary != "" {
			log = append(log, entry(at, "", r.Temporary, RoleHuman))
			r.Temporary = ""
			r.sync(now, connected)
		}
		return log, time.Time{}
	}
	// Every real admin is away: the role waits until the last of them has
	// been away for AwayLimit.
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
		if r.Temporary != "" {
			log = append(log, entry(at, "", r.Temporary, RoleHuman))
			r.Temporary = ""
			r.sync(now, connected)
		}
		return log, allAway
	}
	var tempDue time.Time
	if r.Temporary != "" && !connected(r.Temporary) {
		tempDue = r.since(r.Temporary).Add(AwayLimit)
	}
	if r.Temporary != "" && (connected(r.Temporary) || now.Before(tempDue)) {
		return log, tempDue
	}
	for _, c := range candidates {
		if c == "" || c == r.Temporary || !connected(c) || r.Real(c) {
			continue
		}
		if r.Temporary != "" {
			log = append(log, entry(at, "", r.Temporary, RoleHuman))
		}
		r.Temporary = c
		log = append(log, entry(at, "", c, RoleTemporary))
		r.sync(now, connected)
		return log, time.Time{}
	}
	// Nobody to pass the role to: it stays, and the next connection checks
	// again.
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
	t := ahead.Temporary
	if r.Real(t) {
		t = ""
	}
	if t != r.Temporary {
		if r.Temporary != "" {
			log = append(log, entry(at, "", r.Temporary, RoleHuman))
		}
		if t != "" {
			log = append(log, entry(at, "", t, RoleTemporary))
		}
		r.Temporary = t
	}
	away := make([]Away, 0, len(r.Away)+len(ahead.Away))
	for _, w := range append(slices.Clone(r.Away), ahead.Away...) {
		if r.IsAdmin(w.UserID) && !slices.ContainsFunc(away, func(x Away) bool { return x.UserID == w.UserID }) {
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
	if u == "" || r.Temporary != u {
		return false
	}
	live := r.Clone()
	live.Carry(ahead, 0)
	live.Check(now, connected, candidates, 0)
	return live.Temporary == u
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
	track(r.Temporary)
	r.Away = keep
}

func (r *Roles) since(u string) time.Time {
	if i := slices.IndexFunc(r.Away, func(a Away) bool { return a.UserID == u }); i >= 0 {
		return r.Away[i].Since
	}
	return time.Time{}
}

// Valid checks roles read from a record: nonempty, distinct real admins
// within MaxAdmins, each promoter named, a temporary admin who is not also
// real, and Away rows only for real or temporary admins, once each.
func (r *Roles) Valid() error {
	bad := errors.New("roles: invalid")
	if len(r.Admins) > MaxAdmins {
		return bad
	}
	for i, a := range r.Admins {
		if a.UserID == "" || r.admin(a.UserID) != i || (a.PromotedBy != nil && *a.PromotedBy == "") ||
			(a.PromotedBy == nil && a.UserID != r.Creator) {
			return bad
		}
	}
	if r.Temporary != "" && r.Real(r.Temporary) {
		return bad
	}
	for i, w := range r.Away {
		if !r.IsAdmin(w.UserID) || slices.IndexFunc(r.Away, func(x Away) bool { return x.UserID == w.UserID }) != i || w.Since.IsZero() {
			return bad
		}
	}
	return nil
}
