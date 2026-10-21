package room

import "github.com/NightMachinery/ajayeb/server/internal/wire"

// dropOtherFamily replaces a lobby-to-game snapshot under the same code. The
// shared room budget stays bounded, and data never merges across game families.
// Caller holds mu. Duel's existing typed merge remains unchanged.
func (o *Outbox) dropOtherFamily(code, family string) {
	for i, message := range o.queue {
		var oldCode, oldFamily string
		switch old := message.(type) {
		case wire.State:
			oldCode, oldFamily = old.Code, "duel"
		case wire.TableState:
			oldCode, oldFamily = old.Code, "haftgane"
		case wire.LobbyState:
			oldCode, oldFamily = old.Code, "lobby"
		}
		if oldCode == code && oldFamily != "" && oldFamily != family {
			o.queue = append(o.queue[:i], o.queue[i+1:]...)
			o.states--
			return
		}
	}
}

// tableContinues uses completed batches, never the pending point or private
// pick. Undo reduces Batch while advancing Revision, so it drops stale events.
func tableContinues(prev, next wire.TableState) bool {
	return next.Revision >= prev.Revision && next.Batch >= prev.Batch
}

func (o *Outbox) pushTableState(st wire.TableState) {
	replaced := false
	for i, message := range o.queue {
		old, ok := message.(wire.TableState)
		if !ok || old.Code != st.Code {
			continue
		}
		switch {
		case st.History != nil:
		case old.History != nil && tableContinues(old, st):
			st.History = old.History
			fallthrough
		case len(old.Events) > 0 && tableContinues(old, st):
			merged := make([]wire.TableEvent, 0, len(old.Events)+len(st.Events))
			merged = append(merged, old.Events...)
			merged = append(merged, st.Events...)
			if len(merged) > MaxMergedEvents {
				merged = merged[len(merged)-MaxMergedEvents:]
			}
			st.Events = merged
		}
		if st.Ack == nil {
			st.Ack = old.Ack
		}
		o.queue = append(o.queue[:i], o.queue[i+1:]...)
		replaced = true
		break
	}
	if !replaced {
		if o.states >= MaxPendingStates {
			o.overflow = true
			return
		}
		o.states++
	}
	o.queue = append(o.queue, st)
}

// Lobby snapshots contain their complete public log. Only an unacknowledged
// request's ack needs carrying forward; there are no rules events to merge.
func (o *Outbox) pushLobbyState(st wire.LobbyState) {
	replaced := false
	for i, message := range o.queue {
		old, ok := message.(wire.LobbyState)
		if !ok || old.Code != st.Code {
			continue
		}
		if st.Ack == nil {
			st.Ack = old.Ack
		}
		o.queue = append(o.queue[:i], o.queue[i+1:]...)
		replaced = true
		break
	}
	if !replaced {
		if o.states >= MaxPendingStates {
			o.overflow = true
			return
		}
		o.states++
	}
	o.queue = append(o.queue, st)
}
