package tableroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// broadcast builds one publication's metadata. Engine views and history are
// immutable for the engine fingerprint and cached by seat (plus spectator),
// at most eight entries. Pending picks, acks, roles and presence are never
// cached. A resolve or undo changes the fingerprint and discards all entries.
type broadcast struct {
	a        *actor
	resolved bool // this change resolved a batch: frames carry its events
	history  bool // every frame carries its viewer's whole history (undo)
	seats    []wire.PlatformSeat
	humans   []wire.PlatformHuman
	previous *[]wire.PreviousSeat
	views    map[int]*roleView // by seat; -1 is the spectator role
}

type roleView struct {
	proj    projection
	history *[]wire.TableHistoryEntry
}

func (a *actor) newBroadcast(resolved bool) *broadcast {
	if a.engineViews == nil || a.viewHash != a.st.rec.Hash {
		a.viewHash = a.st.rec.Hash
		a.engineViews = map[int]*roleView{}
	}
	b := &broadcast{a: a, resolved: resolved, views: a.engineViews}
	b.seats, b.humans = a.meta()
	return b
}

// role is the viewer role of a connection: its user's seat, or -1.
func (b *broadcast) role(c *room.Client) int { return b.a.seatOf(b.a.userOf(c)) }

func (b *broadcast) view(role int, withHistory bool) *roleView {
	v := b.views[role]
	g := &b.a.st.game
	vw := spectatorViewer()
	if role >= 0 {
		vw = seatViewer(role)
	}
	if v == nil {
		v = &roleView{proj: project(g, vw)}
		b.views[role] = v
	}
	if withHistory && v.history == nil {
		h := history(g, vw, b.a.t.log, b.a.code)
		v.history = &h
	}
	return v
}

// frame is c's whole whitelist snapshot. ack is set only on the
// submitter's own connection; withHistory adds the viewer's full history
// (join, resync, undo), as does a change of the connection's viewer role
// (a switch, a resignation).
func (b *broadcast) frame(c *room.Client, ack *wire.PlatformAck, withHistory bool) wire.TableState {
	a := b.a
	r := &a.st.rec
	role := b.role(c)
	if last, ok := a.shown[c]; ok && last != role {
		withHistory = true
	}
	a.shown[c] = role
	withHistory = withHistory || b.history
	v := b.view(role, withHistory)
	st := wire.NewTableState(a.code)
	st.Revision = revisionOf(r, role)
	st.Batch = len(r.Batches)
	if r.Status == store.StatusFinished {
		st.Status = "finished"
	}
	st.Seats, st.Humans = b.seats, b.humans
	user := a.userOf(c)
	st.You = wire.PlatformYou{UserID: user}
	if role >= 0 {
		seat := role
		st.You.Seat = &seat
	}
	if a.isAdmin(user) {
		st.You.PreviousSeats = b.hints()
	}
	if role < 0 && r.Status == store.StatusPlaying {
		st.You.FormerSeat = formerSeat(r, user)
	}
	if r.Roles.Real(user) {
		room := r.Roles.HasRoom()
		st.You.CanPromote = &room
	}
	st.StartingHumanCount, st.AllowUndo, st.UndoOff, st.FinalReveal = r.StartingHumans, r.AllowUndo, r.UndoOff, !r.Setup.HideAtEnd
	for _, p := range r.BotPresets {
		st.BotLevels = append(st.BotLevels, p.Level)
	}
	if p := a.st.game.Point(); !pointOver(p) && r.Point != nil {
		tp := &wire.TablePoint{Ref: *r.Point, Kind: pointKind(p), Required: []int{}, Confirmed: []int{}}
		for _, s := range p.Required.Seats() {
			tp.Required = append(tp.Required, int(s))
			if ps := pendingOf(r, int(s)); ps != nil && ps.Confirmed {
				tp.Confirmed = append(tp.Confirmed, int(s))
			}
		}
		st.Point = tp
		if ps := pendingOf(r, role); role >= 0 && ps != nil {
			st.Pending = &wire.TablePending{Key: ps.Key, Confirm: ps.Confirmed}
		}
	}
	st.Ack = ack
	if b.resolved {
		st.Events = v.proj.events
	}
	if withHistory {
		st.History = v.history
	}
	st.RoomLog = append([]wire.PlatformLogEntry{}, r.RoomLog...)
	st.View = v.proj.view
	return st
}

// hints is the admins' previous-seat hint: for each connected spectator
// who held a seat earlier in this game, the seat they held last.
func (b *broadcast) hints() *[]wire.PreviousSeat {
	if b.previous == nil {
		a := b.a
		h := []wire.PreviousSeat{}
		for _, p := range a.st.rec.Previous {
			if a.seatOf(p.UserID) < 0 && a.connected(p.UserID) {
				h = append(h, wire.PreviousSeat{UserID: p.UserID, Seat: p.Seat})
			}
		}
		b.previous = &h
	}
	h := slices.Clone(*b.previous)
	return &h
}

// formerSeat is the seat user held last, for that user's own frame only, so
// a former player watching the game sees which seat was theirs on any tab.
func formerSeat(r *Record, user string) *int {
	i := slices.IndexFunc(r.Previous, func(p PreviousRecord) bool { return p.UserID == user })
	if i < 0 {
		return nil
	}
	seat := r.Previous[i].Seat
	return &seat
}

// meta is the public room metadata: seats with their controllers, and the
// humans (seated ones in seat order, then connected spectators in the
// order they arrived) with presence and roles.
func (a *actor) meta() ([]wire.PlatformSeat, []wire.PlatformHuman) {
	r := &a.st.rec
	seats := make([]wire.PlatformSeat, 0, len(r.Seats))
	humans := []wire.PlatformHuman{}
	add := func(userID, name string) {
		if slices.ContainsFunc(humans, func(h wire.PlatformHuman) bool { return h.UserID == userID }) {
			return
		}
		h := wire.PlatformHuman{UserID: userID, Name: name, Connected: a.connected(userID), Creator: userID == r.Roles.Creator,
			Admin: r.Roles.Real(userID), Temporary: r.Roles.Temporary == userID && a.isAdmin(userID), PromotedBy: r.Roles.PromotedBy(userID)}
		same := 0
		for _, o := range humans {
			if o.Name == name {
				same++
			}
		}
		if same > 0 {
			h.NameNo = same + 1
		}
		humans = append(humans, h)
	}
	for _, sr := range r.Seats {
		ps := wire.PlatformSeat{Seat: sr.Seat, Kind: sr.Kind}
		switch sr.Kind {
		case SeatHuman:
			id := sr.UserID
			ps.UserID = &id
			add(sr.UserID, sr.Name)
		case SeatBot:
			level := sr.Bot
			ps.Bot = &level
		}
		seats = append(seats, ps)
	}
	for _, c := range a.conns {
		add(a.userOf(c), c.Name)
	}
	return seats, humans
}

// publish sends every connection its frame; ack goes to the submitter only.
func (a *actor) publish(b *broadcast, submitter *room.Client, ack *wire.PlatformAck) {
	for _, c := range a.conns {
		if c == submitter {
			c.Out.Push(b.frame(c, ack, false))
		} else {
			c.Out.Push(b.frame(c, nil, false))
		}
	}
}

// publishSeat sends a private edit's frames to the seat's own connections
// only: nobody else's frame changes, not even its revision.
func (a *actor) publishSeat(b *broadcast, seat int, submitter *room.Client, ack *wire.PlatformAck) {
	for _, c := range a.conns {
		if a.seatOf(a.userOf(c)) != seat {
			continue
		}
		if c == submitter {
			c.Out.Push(b.frame(c, ack, false))
		} else {
			c.Out.Push(b.frame(c, nil, false))
		}
	}
}

// broadcastExcept sends a presence or claim change to every connection
// but except.
func (a *actor) broadcastExcept(b *broadcast, except *room.Client) {
	for _, c := range a.conns {
		if c != except {
			c.Out.Push(b.frame(c, nil, false))
		}
	}
}
