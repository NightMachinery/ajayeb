package tableroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/namenum"
)

// assignSeatNames initializes a legacy table once in its current seat order.
func (r *Record) assignSeatNames() {
	for i, s := range r.Seats {
		if s.Kind == SeatHuman {
			r.Names.AssignBounded(s.UserID, s.Name, maxNameCounters)
			r.Seats[i].Name = r.Names.Get(s.UserID).Name
		}
	}
}

func (r *Record) rename(user, name string, present []namenum.Assignment) bool {
	changed := r.Names.AssignBounded(user, name, maxNameCounters, present...)
	for i := range r.Seats {
		if r.Seats[i].Kind == SeatHuman && r.Seats[i].UserID == user {
			r.Seats[i].Name = r.Names.Get(user).Name
		}
	}
	return changed
}

// Remember only the already bounded population used by authority/seq history.
// Previous seats are separately bounded and retained for rematches.
const maxNameUsers = maxSeqUsers + maxPrevious
const maxNameCounters = 2 * maxNameUsers

func (r *Record) remembersName(user string) bool {
	return hasAuthority(r, user) || slices.Contains(r.Retired, user) || slices.ContainsFunc(r.Previous, func(p PreviousRecord) bool { return p.UserID == user })
}

func (r *Record) syncNames(present []namenum.Assignment) {
	r.Names.Retain(r.remembersName)
	for _, a := range present {
		if r.remembersName(a.UserID) && r.Names.Get(a.UserID).UserID == "" {
			r.Names.Import(a, maxNameCounters)
		}
	}
	r.assignSeatNames()
}

func (a *actor) assignment(user string) namenum.Assignment {
	if x := a.st.rec.Names.Get(user); x.UserID != "" {
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
	a.visitors = append(a.visitors, a.st.rec.Names.Preview(user, name, a.visitors))
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
