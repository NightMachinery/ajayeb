package tableroom

import (
	"errors"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// linkOf runs link (reset: relink) from client id and returns its
// outcome and, when accepted, the claim of the linked frame.
func (h *th) linkOf(id tabletest.ClientID, seat int, reset bool) (string, string) {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	var out string
	if err := a.call(h.ctx, func() { out = a.link(h.p.clients[id], seat, reset) }); err != nil {
		h.t.Fatal(err)
	}
	h.p.drain()
	if out != outAccepted {
		return out, ""
	}
	others := h.p.other[id]
	l, ok := others[len(others)-1].(wire.PlatformLinked)
	if !ok || l.Seat != seat || l.Code != h.p.code {
		h.t.Fatalf("no linked frame: %+v", others)
	}
	return out, l.Claim
}

// joinClaim joins a new client as user u with claim.
func (h *th) joinClaim(id tabletest.ClientID, u, claim string) error {
	c := newClient(u, string(id), "Device")
	if err := h.f.m.JoinClaim(h.ctx, c, h.p.code, &claim); err != nil {
		return err
	}
	h.p.clients[id] = c
	h.p.drain()
	return nil
}

// revoked reports whether id received error bad-claim.
func (h *th) revoked(id tabletest.ClientID) bool {
	h.p.drain()
	return slices.ContainsFunc(h.p.other[id], func(m wire.ServerMessage) bool {
		e, ok := m.(wire.Error)
		return ok && e.Code == wire.ErrorCodeBadClaim
	})
}

// TestDeviceLinks: a seat's user or an admin gets the seat's claim; a
// connection joining with it acts as that seat's user, admin powers
// included; a relink revokes it (the connection gets bad-claim and is
// detached, except the sender, which keeps its place under the new
// claim) and is logged for everyone; a seat switch revokes the replaced
// user's claims; a claim of another room or a revoked one is refused.
func TestDeviceLinks(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 53))
	admin, c1, spec := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SpectatorClient
	if out, _ := h.linkOf(c1, 2, false); out != outNotAdmin {
		t.Fatalf("another seat's link: %s", out)
	}
	if out, _ := h.linkOf(spec, 0, false); out != outNotAdmin {
		t.Fatalf("a spectator's link: %s", out)
	}
	if out, _ := h.linkOf(admin, 7, false); out != outBad {
		t.Fatalf("a link of no seat: %s", out)
	}
	rev := h.latest(spec).Revision
	_, claim1 := h.linkOf(c1, 1, false)
	if claim1 != h.f.secret.Claim(h.p.code, userOf(1), 0) || h.latest(spec).Revision != rev {
		t.Fatal("a link changed the table or gave the wrong claim")
	}
	if _, claim := h.linkOf(admin, 1, false); claim != claim1 {
		t.Fatal("the admin got another claim")
	}
	if err := h.joinClaim("device-1", "phone", "not-a-claim"); !errors.Is(err, room.ErrBadClaim) {
		t.Fatalf("a bad claim: %v", err)
	}
	if err := h.joinClaim("device-1", "phone", claim1); err != nil {
		t.Fatal(err)
	}
	fr := h.latest("device-1")
	if fr.You.UserID != userOf(1) || fr.You.Seat == nil || *fr.You.Seat != 1 || fr.View.You == nil {
		t.Fatalf("the device's frame: %+v", fr.You)
	}
	sn := h.snap("device-1")
	h.expect(h.send("device-1", 1, sn.Ref, sn.Legal[0], false), tabletest.Accepted, "the device's draft")
	if pd := h.latest(c1).Pending; pd == nil || pd.Key != sn.Legal[0] {
		t.Fatal("the seat's own tab does not see the device's draft")
	}

	// The creator's device acts as the admin.
	_, claim0 := h.linkOf(admin, 0, false)
	if err := h.joinClaim("device-0", "laptop", claim0); err != nil {
		t.Fatal(err)
	}
	if out := h.do("device-0", roleOp(1, "promote", userOf(2))); out != outAccepted {
		t.Fatalf("the admin's device promotes: %s", out)
	}

	// A relink by the seat's own tab revokes the device.
	_, claim1b := h.linkOf(c1, 1, true)
	if claim1b == claim1 || !h.revoked("device-1") {
		t.Fatal("the relink did not revoke the device")
	}
	if out := h.send("device-1", 2, sn.Ref, sn.Legal[0], true); out != tabletest.Forbidden {
		t.Fatalf("the revoked device's pick: %s", out)
	}
	if err := h.joinClaim("device-1b", "phone", claim1); !errors.Is(err, room.ErrBadClaim) {
		t.Fatalf("the revoked claim: %v", err)
	}
	delete(h.p.clients, "device-1")
	log := h.sameLogEverywhere(2)
	if e := log[1]; e.Kind != "link" || e.By != userOf(1) || *e.Seat != 1 {
		t.Fatalf("the relink's entry: %+v", e)
	}
	// A device relinking its own seat keeps its place under the new claim.
	if err := h.joinClaim("device-1c", "phone", claim1b); err != nil {
		t.Fatal(err)
	}
	if _, claim := h.linkOf("device-1c", 1, true); claim == claim1b || h.revoked("device-1c") || h.latest("device-1c").You.UserID != userOf(1) {
		t.Fatal("the relinking device lost its place")
	}
	// A switch of the seat to a bot revokes the device's claim too.
	h.want(admin, toBot(1, 1, BotEasy), outAccepted, "seat 1 to a bot")
	if !h.revoked("device-1c") {
		t.Fatal("the switch did not revoke the device")
	}
	if h.revoked("device-0") || h.revoked(c1) {
		t.Fatal("a connection without a revoked claim was ended")
	}
	if out, _ := h.linkOf(admin, 1, false); out != outBad {
		t.Fatalf("a bot seat's link: %s", out)
	}
}

// TestRematchRefusals: no rematch while the game runs, and none without
// a lobby kind to make it.
func TestRematchRefusals(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 59))
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	var out string
	if err := a.call(h.ctx, func() { out = a.rematch(h.p.clients[tabletest.SeatClient(0)]) }); err != nil {
		t.Fatal(err)
	}
	if out != outBad {
		t.Fatalf("a rematch while playing: %s", out)
	}
}

// TestRematchCreationLimit: on a finished table, a spectator's rematch is
// forbidden; an original human's counts against their room-creation
// limit, so a refused one makes nothing; an allowed one makes the
// successor once and sends its code to every connection; a repeat answers
// the same code to its sender without asking the limit again.
func TestRematchCreationLimit(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 61))
	stub := &stubSuccessors{}
	h.f.succ = stub
	h.f.tables.SetSuccessors(stub)
	if _, err := tabletest.Play(h.ctx, h.p, tabletest.Config{Seats: 3, PickSeed: 61, ArrivalSeed: 62}); err != nil {
		t.Fatal(err)
	}
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	rematch := func(id tabletest.ClientID) string {
		var out string
		if err := a.call(h.ctx, func() { out = a.rematch(h.p.clients[id]) }); err != nil {
			t.Fatal(err)
		}
		h.p.drain()
		return out
	}
	seat0, seat1 := h.p.clients[tabletest.SeatClient(0)], h.p.clients[tabletest.SeatClient(1)]
	if out := rematch(tabletest.SpectatorClient); out != outForbidden {
		t.Fatalf("the spectator's rematch: %s", out)
	}
	asked := 0
	allow := false
	seat0.AllowCreate = func() bool { asked++; return allow }
	if out := rematch(tabletest.SeatClient(0)); out != outRate || len(stub.made) != 0 {
		t.Fatalf("a rematch past the limit: %s, %d made", out, len(stub.made))
	}
	allow = true
	if out := rematch(tabletest.SeatClient(0)); out != outAccepted || len(stub.made) != 1 || stub.made[0].Creator != userOf(0) {
		t.Fatalf("the rematch: %s, %+v", out, stub.made)
	}
	for id := range h.p.clients {
		others := h.p.other[id]
		if cr, ok := others[len(others)-1].(wire.Created); !ok || cr.Code != "RMATCH" || cr.Claim != nil {
			t.Fatalf("%s's created frame: %+v", id, others)
		}
	}
	seat1.AllowCreate = func() bool { asked++; return false }
	if out := rematch(tabletest.SeatClient(1)); out != outDuplicate || asked != 2 || len(stub.made) != 1 {
		t.Fatalf("a repeated rematch: %s, the limit asked %d times, %d made", out, asked, len(stub.made))
	}
}
