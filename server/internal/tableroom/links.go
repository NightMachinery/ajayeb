package tableroom

import (
	"errors"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Member is a connection the lobby hands over at Start, with the claim it
// joined with and the user that claim makes it act as (both "" without
// one).
type Member struct {
	Client *room.Client
	Claim  string
	User   string
}

// link answers link, or with reset relink, in the protocol's order:
// device links must be on, a human must hold the seat, and the sender must
// act as that seat's user. An admin, real or temporary, has no other
// seat's link: a claim lets a device play as the seat's user, seeing its
// hand and pending pick, and nothing would show it, while every seat switch
// is in everyone's log. An admin who needs another seat seats someone
// there, publicly. A reset
// raises the seat's epoch to the next public revision, which revokes every
// claim made for it before, logs the reset for everyone and ends the
// connections that joined with a revoked claim. Either way the sender
// alone receives the seat's current claim, in a linked frame. Every
// refusal carries seq 0.
func (a *actor) link(c *room.Client, seat int, reset bool) string {
	r := &a.st.rec
	op := "link"
	if reset {
		op = "relink"
	}
	cl := a.m.Claims()
	if cl == nil || seat < 0 || seat >= len(r.Seats) || r.Seats[seat].Kind != SeatHuman {
		return a.refuse(c, 0, op, seat, outBad)
	}
	user, target := a.userOf(c), r.Seats[seat].UserID
	if user != target {
		return a.refuse(c, 0, op, seat, outNotAdmin)
	}
	if reset {
		cand := r.clone()
		bumpPublic(&cand)
		cand.Seats[seat].Epoch = cand.Public
		s := seat
		appendLog(&cand, wire.PlatformLogEntry{Kind: "link", At: len(cand.Batches), By: user, Seat: &s})
		if err := a.commit(op, cand, a.st.game); err != nil {
			return a.refuse(c, 0, op, seat, outStorage)
		}
	}
	claim := cl.Claim(a.code, target, a.st.rec.Seats[seat].Epoch)
	if own, ok := a.claims[c]; reset && ok && own.user == target {
		// The sender joined with the claim it has just revoked: it keeps
		// its place under the new one, which it is about to receive.
		a.claims[c] = claimed{claim: claim, user: target}
	}
	c.Out.Push(wire.PlatformLinked{T: "linked", Code: a.code, Seat: seat, Claim: claim})
	if reset {
		gone := a.revokeClaims()
		a.publish(a.newBroadcast(false), nil, nil)
		a.afterRevoke(gone)
	}
	return a.note(op, seat, outAccepted)
}

// revokeClaims ends every connection whose claim no longer belongs to a
// human seat's user and epoch (after a switch, a resignation or a link
// reset): it receives error bad-claim and is detached, like a connection
// that left, so its next request is refused until it joins again. It
// returns the users those connections acted as.
func (a *actor) revokeClaims() []string {
	var gone []string
	for _, c := range slices.Clone(a.conns) {
		cl, ok := a.claims[c]
		if !ok {
			continue
		}
		if u, ok := a.claimHolder(&a.st.rec, cl.claim); ok && u == cl.user {
			continue
		}
		a.conns = slices.DeleteFunc(a.conns, func(x *room.Client) bool { return x == c })
		delete(a.claims, c)
		c.Out.Push(wire.NewError(wire.ErrorCodeBadClaim, "that device link was reset"))
		gone = append(gone, cl.user)
	}
	return gone
}

// afterRevoke settles presence after revokeClaims, once the change is
// published: the roles' away times of users left with no connection, the
// bots and the idle clock.
func (a *actor) afterRevoke(gone []string) {
	if len(gone) == 0 {
		return
	}
	if slices.ContainsFunc(gone, func(u string) bool { return !a.connected(u) }) {
		a.checkRoles()
	}
	a.dropped()
}

// Rematch is a finished table's successor lobby: the table it follows,
// its Start slots in lobby order (each human's to be reserved for them,
// the bots kept), the options and the requester, its creator and only
// admin.
type Rematch struct {
	Sides       string
	Parent      string
	Creator     string
	Slots       []StartSlot
	LobbyOrder  bool
	AllowUndo   bool
	FinalReveal bool
}

// ErrCodeTaken: another room holds the code a rematch reserved.
var ErrCodeTaken = errors.New("tableroom: the reserved successor code is another room's")

// Successors makes a finished table's rematch lobby (package lobbyroom).
// Reserve returns a fresh code for it, which the table writes down first.
// Rematch then makes the lobby under that code, or finds the one an
// earlier attempt for the same parent made, or answers ErrCodeTaken.
type Successors interface {
	Reserve(m *room.Manager) (string, error)
	Rematch(m *room.Manager, code string, r Rematch) error
}

// SetSuccessors gives the tables the lobby kind that makes their rematch
// lobbies. Without one a rematch is refused with bad-request.
func (t *Tables) SetSuccessors(s Successors) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.succ = s
}

func (t *Tables) successors() Successors {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.succ
}

// rematch answers a rematch request from an original participant of a
// finished table with its one successor lobby, in three writes: the
// table reserves a fresh code (Reserved, with the requester as
// Rematcher), the lobby is made under it, recording this table as its
// parent, and the table records it as its Successor. A request after a
// refused write or a crash between them, by any original participant,
// goes on from the reservation, so the table never has two successors;
// one after the last write answers the same code. A new successor's code
// goes to every connection in a created frame, the requester's included;
// a repeated request's to the requester alone. A connection that joined
// with an original human's claim also receives that human's claim in the
// successor, where their slot is reserved. Reserving the code counts
// against the requester's room-creation limits (room.Client.AllowCreate),
// as Duel's rematch does; going on from a reservation does not.
func (a *actor) rematch(c *room.Client) string {
	r := &a.st.rec
	user := a.userOf(c)
	switch {
	case r.Status == store.StatusPlaying:
		return a.refuse(c, 0, "rematch", -1, outBad)
	case !a.original(user):
		return a.refuse(c, 0, "rematch", -1, outForbidden)
	case r.Successor != "":
		c.Out.Push(a.created(c, r.Successor))
		return a.note("rematch", -1, outDuplicate)
	}
	succ := a.t.successors()
	if succ == nil {
		return a.refuse(c, 0, "rematch", -1, outBad)
	}
	if r.Reserved == "" {
		if c.AllowCreate != nil && !c.AllowCreate() {
			return a.refuse(c, 0, "rematch", -1, outRate)
		}
		code, err := succ.Reserve(a.m)
		if err != nil {
			a.t.log.Warn("table rematch failed", "room", a.code, "err", err)
			return a.refuse(c, 0, "rematch", -1, outStorage)
		}
		cand := r.clone()
		cand.Reserved, cand.Rematcher = code, user
		if err := a.commit("rematch", cand, a.st.game); err != nil {
			return a.refuse(c, 0, "rematch", -1, outStorage)
		}
		r = &a.st.rec
	}
	code := r.Reserved
	err := succ.Rematch(a.m, code, Rematch{Parent: a.code, Creator: r.Rematcher, Slots: slices.Clone(r.Start),
		Sides: r.Setup.Sides, LobbyOrder: !r.Setup.RandomSeating, AllowUndo: r.AllowUndo, FinalReveal: !r.Setup.HideAtEnd})
	switch {
	case errors.Is(err, ErrCodeTaken):
		// Another room took the code first: forget it, so the next
		// request reserves another.
		cand := r.clone()
		cand.Reserved, cand.Rematcher = "", ""
		_ = a.commit("rematch", cand, a.st.game)
		a.t.log.Warn("table rematch code taken", "room", a.code)
		return a.refuse(c, 0, "rematch", -1, outStorage)
	case errors.Is(err, room.ErrForbidden):
		return a.refuse(c, 0, "rematch", -1, outForbidden)
	case err != nil:
		a.t.log.Warn("table rematch failed", "room", a.code, "err", err)
		return a.refuse(c, 0, "rematch", -1, outStorage)
	}
	cand := r.clone()
	cand.Successor, cand.Reserved, cand.Rematcher = code, "", ""
	if err := a.commit("rematch", cand, a.st.game); err != nil {
		return a.refuse(c, 0, "rematch", -1, outStorage)
	}
	for _, o := range a.conns {
		o.Out.Push(a.created(o, code))
	}
	return a.note("rematch", -1, outAccepted)
}

// original reports whether u was a human seat at Start.
func (a *actor) original(u string) bool {
	return u != "" && slices.ContainsFunc(a.st.rec.Start, func(s StartSlot) bool { return s.UserID == u })
}

// created is c's created frame for the successor code: with the claim of
// its user's reserved slot there when c joined with an original human's
// claim (epoch 0, the lobby's first).
func (a *actor) created(c *room.Client, code string) wire.Created {
	f := wire.NewCreated(code)
	if cl, ok := a.claims[c]; ok && a.original(cl.user) {
		if claims := a.m.Claims(); claims != nil {
			k := claims.Claim(code, cl.user, 0)
			f.Claim = &k
		}
	}
	return f
}
