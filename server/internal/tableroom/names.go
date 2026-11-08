package tableroom

// assignSeatNames initializes a legacy table once in its current seat order.
func (r *Record) assignSeatNames() {
	for _, s := range r.Seats {
		if s.Kind == SeatHuman {
			r.Names.Assign(s.UserID, s.Name)
		}
	}
}

func (r *Record) rename(user, name string) bool {
	changed := r.Names.Assign(user, name)
	for i := range r.Seats {
		if r.Seats[i].Kind == SeatHuman && r.Seats[i].UserID == user {
			r.Seats[i].Name = name
		}
	}
	return changed
}
