package room

import (
	"context"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file holds a room's hint settings (docs/dev/decisions.md D23): their
// defaults, the game's admin who may change them, the hints frame that
// changes them, and the room log entries that record each change for both
// players. None of it touches the game: the engine, its options and the
// replay never see the settings, and bots never receive hints.

// DefaultHints returns the hint settings a new room of mode starts with
// when its create frame names none: every family on, except the advisor (a
// review of the player's own move) outside games against the bot, and the
// advisor's strength two-stage. It is also what a room file written before
// hint settings existed loads with.
func DefaultHints(mode store.Mode) view.HintSettings {
	return view.HintSettings{
		Prices:          true,
		Economy:         true,
		Science:         true,
		Military:        true,
		Threats:         true,
		Tempo:           true,
		Wonders:         true,
		Tokens:          true,
		Scoring:         true,
		Counting:        true,
		Recap:           true,
		Advisor:         mode == store.ModeAI,
		AdvisorStrength: DefaultAdvisorStrength,
	}
}

// DefaultAdvisorStrength is the advisor strength of a game whose creator
// chose none, and of a room file written before the strength existed.
const DefaultAdvisorStrength = view.AdvisorStrengthTwoStage

// fillHintFields gives a room file the hint fields a file written before
// they existed lacks: the defaults of its mode (and the default advisor
// strength, for settings saved without one), and as its admin the user
// of its first human seat in seat order. That is the creator in a hot-seat
// or bot room and in an online lobby; for an old online room whose two
// seats are taken it is the player of seat 0, the creator's seat unless
// the creator chose otherwise, which the old file did not record. A room
// with no human seat yet (a development room) gets its admin when its
// first player claims a seat (see claim).
func fillHintFields(f *store.File) {
	if f.Hints == nil {
		h := DefaultHints(f.Mode)
		f.Hints = &h
	}
	if !f.Hints.AdvisorStrength.Valid() {
		f.Hints.AdvisorStrength = DefaultAdvisorStrength
	}
	if f.Admin == "" {
		for _, s := range f.Seats {
			if s.Kind == store.SeatKindHuman && s.UserID != "" {
				f.Admin = s.UserID
				break
			}
		}
	}
}

// Hints routes a hints frame to c's room.
func (m *Manager) Hints(ctx context.Context, c *Client, h wire.Hints) error {
	r, err := m.joinedRoom(c, h.Code)
	if err == nil {
		err = r.post(ctx, msgHints{c: c, hints: h})
	}
	return m.requestFailed(c, h.Seq, err)
}

// setHints handles a hints frame, in this order: the sender must be
// attached; a spectator is never the admin; a seq not newer than the last
// one applied for its client id only re-sends the state with an ack, as for
// an act; the sender's user must be the game's admin; the game must not be
// over. Then the settings are replaced (an empty advisor strength keeps
// the game's), the families switched on or off are logged for every
// viewer, and every connection receives a fresh state (the sender's with
// the ack). The room's version does not change: no position changed, so a
// move chosen before the change is still good. A change of the advisor's
// strength is not logged: it changes no hint the opponent sees, only how
// carefully each player's own moves are reviewed from then on, and a
// review already made keeps the strength it was made with.
func (r *Room) setHints(c *Client, m wire.Hints) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	if mem.role == roleSpectator && !r.hintAdmin(mem) {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotAdmin, "only the game's admin may change its hints"))
		return
	}
	ack := &wire.Ack{ClientID: c.ClientID, Seq: m.Seq}
	if r.applied(c, m.Seq) {
		r.sendState(c, mem, ack, false)
		return
	}
	if !r.hintAdmin(mem) {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeNotAdmin, "only the game's admin may change its hints"))
		return
	}
	if r.state.Over() {
		c.Out.Push(wire.NewReject(m.Seq, wire.RejectCodeGameOver, "the game is over; its hints no longer change"))
		return
	}
	prev := *r.file.Hints
	next := m.Hints
	if next.AdvisorStrength == "" {
		next.AdvisorStrength = prev.AdvisorStrength
	}
	r.file.Hints = &next
	r.logHints(mem, prev, next)
	r.record(c, ack)
	r.persist()
	r.broadcast(nil, c, ack, noHistory)
	r.log.Info("hints changed")
}

// logHints records a change of the settings from prev to next in the room
// log. Changes made at the same ply by the same seat fold into one entry
// (the families that differ from the settings before the first of them),
// and an entry whose changes cancel out is dropped, so toggling a switch
// back and forth leaves no trace. A change of nothing logs nothing.
func (r *Room) logHints(mem member, prev, next view.HintSettings) {
	seat := -1
	if mem.role == rolePlayer {
		seat = int(mem.seat)
	}
	at := int(r.state.Ply)
	log := r.file.RoomLog
	if n := len(log); n > 0 && log[n-1].Kind == string(wire.RoomLogHints) && log[n-1].At == at && log[n-1].Seat == seat {
		// Undo the last entry to find the settings before it.
		for _, f := range log[n-1].On {
			wire.SetHint(&prev, wire.HintFamily(f), false)
		}
		for _, f := range log[n-1].Off {
			wire.SetHint(&prev, wire.HintFamily(f), true)
		}
		log = log[:n-1]
	}
	on, off := wire.DiffHints(prev, next)
	if len(on) > 0 || len(off) > 0 {
		log = append(log, store.LogEntry{
			Kind: string(wire.RoomLogHints),
			At:   at,
			Seat: seat,
			On:   familyNames(on),
			Off:  familyNames(off),
		})
	}
	if len(log) > wire.MaxRoomLog {
		log = append([]store.LogEntry{}, log[len(log)-wire.MaxRoomLog:]...)
	}
	r.file.RoomLog = log
}

// clampRoomLog moves the entries of the room log that happened after the
// current ply back to it, after an undo took the game back: they still
// happened, and before the moves that follow now.
func (r *Room) clampRoomLog() {
	at := int(r.state.Ply)
	for i := range r.file.RoomLog {
		if r.file.RoomLog[i].At > at {
			r.file.RoomLog[i].At = at
		}
	}
}

// familyNames turns families into the strings the room file keeps.
func familyNames(fs []wire.HintFamily) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, string(f))
	}
	return out
}

// wireRoomLog is the room log as every state frame carries it: the newest
// wire.MaxRoomLog entries (hint changes, and link resets from identity.go).
func wireRoomLog(log []store.LogEntry) []wire.RoomLogEntry {
	log = log[max(0, len(log)-wire.MaxRoomLog):]
	out := make([]wire.RoomLogEntry, 0, len(log))
	for _, e := range log {
		w := wire.RoomLogEntry{
			Kind: wire.RoomLogKind(e.Kind),
			At:   e.At,
			On:   make([]wire.HintFamily, 0, len(e.On)),
			Off:  make([]wire.HintFamily, 0, len(e.Off)),
		}
		if e.Seat == 0 || e.Seat == 1 {
			seat := e.Seat
			w.Seat = &seat
		}
		if e.Target != nil && (*e.Target == 0 || *e.Target == 1) {
			target := *e.Target
			w.Target = &target
		}
		for _, f := range e.On {
			w.On = append(w.On, wire.HintFamily(f))
		}
		for _, f := range e.Off {
			w.Off = append(w.Off, wire.HintFamily(f))
		}
		out = append(out, w)
	}
	return out
}

// hintAdmin keeps legacy ownership and gives shared-lobby games precisely the
// frozen Start-time set, including promoted spectators, with no role updates.
func (r *Room) hintAdmin(mem member) bool {
	if mem.user == "" {
		return false
	}
	if len(r.file.HintAdmins) > 0 {
		return slices.Contains(r.file.HintAdmins, mem.user)
	}
	return mem.role != roleSpectator && mem.user == r.file.Admin
}
