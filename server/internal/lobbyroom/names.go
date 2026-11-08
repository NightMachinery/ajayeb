package lobbyroom

// assignSlotNames initializes a new or legacy record in deterministic slot
// order. Existing assignments are retained when a user merely moves slots.
func (r *Record) assignSlotNames() {
	for _, s := range r.Slots {
		if s.Kind == SlotHuman || s.Kind == SlotReserved {
			r.Names.Assign(s.UserID, s.Name)
		}
	}
}

// rename keeps the stored slot and the shared identity in agreement.
// Device-link connections never rename their claimed identity.
func (r *Record) rename(user, name string) bool {
	changed := r.Names.Assign(user, name)
	for i := range r.Slots {
		if r.Slots[i].UserID == user {
			r.Slots[i].Name = name
		}
	}
	return changed
}
