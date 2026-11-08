package lobbyroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/namenum"
)

// assignSlotNames initializes a new or legacy record in deterministic slot
// order. Existing assignments are retained when a user merely moves slots.
func (r *Record) assignSlotNames(present ...namenum.Assignment) {
	for i, s := range r.Slots {
		if s.Kind == SlotHuman || s.Kind == SlotReserved {
			r.Names.AssignBounded(s.UserID, s.Name, maxNameCounters, present...)
			r.Slots[i].Name = r.Names.Get(s.UserID).Name
		}
	}
}

// rename keeps the stored slot and the shared identity in agreement.
// Device-link connections never rename their claimed identity.
func (r *Record) rename(user, name string, present []namenum.Assignment) bool {
	changed := r.Names.AssignBounded(user, name, maxNameCounters, present...)
	for i := range r.Slots {
		if r.Slots[i].UserID == user {
			r.Slots[i].Name = r.Names.Get(user).Name
		}
	}
	return changed
}

// Remember only the already bounded population used by authority/seq history.
// Previous seats are separately bounded and retained for rematches.
const maxNameUsers = maxSeqUsers
const maxNameCounters = 2 * maxNameUsers

func (r *Record) remembersName(user string) bool {
	return hasAuthority(r, user) || slices.Contains(r.Retired, user) || slotOf(r, user, SlotReserved) >= 0
}

func (r *Record) syncNames(present []namenum.Assignment) {
	r.Names.Retain(r.remembersName)
	for _, a := range present {
		if r.remembersName(a.UserID) && r.Names.Get(a.UserID).UserID == "" {
			r.Names.Import(a, maxNameCounters)
		}
	}
	r.assignSlotNames(present...)
}

func (a *actor) assignment(user string) namenum.Assignment {
	if x := a.rec.Names.Get(user); x.UserID != "" {
		return x
	}
	for _, x := range a.visitors {
		if x.UserID == user {
			return x
		}
	}
	return namenum.Assignment{}
}

func (a *actor) visit(user, name string) bool {
	name = namenum.CleanName(name)
	if name == "" {
		return false
	}
	a.visitors = slices.DeleteFunc(a.visitors, func(x namenum.Assignment) bool { return !a.connected(x.UserID) })
	old := a.assignment(user)
	if old.UserID != "" && namenum.Key(old.Name) == namenum.Key(name) {
		if old.Name == name {
			return false
		}
		for i := range a.visitors {
			if a.visitors[i].UserID == user {
				a.visitors[i].Name = name
			}
		}
		return true
	}
	a.visitors = slices.DeleteFunc(a.visitors, func(x namenum.Assignment) bool { return x.UserID == user })
	a.visitors = append(a.visitors, a.rec.Names.Preview(user, name, a.visitors))
	return true
}

// Moving beyond the retained history must not renumber someone still watching.
// Promote/demote the in-memory identity only after the record write succeeds.
func (a *actor) retainVisitors(before, after *Record) {
	a.visitors = slices.DeleteFunc(a.visitors, func(x namenum.Assignment) bool {
		return !a.connected(x.UserID) || after.Names.Get(x.UserID).UserID != ""
	})
	for _, x := range before.Names.Users {
		if a.connected(x.UserID) && after.Names.Get(x.UserID).UserID == "" {
			a.visitors = slices.DeleteFunc(a.visitors, func(v namenum.Assignment) bool { return v.UserID == x.UserID })
			a.visitors = append(a.visitors, x)
		}
	}
}
