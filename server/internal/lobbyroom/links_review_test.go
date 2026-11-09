package lobbyroom

import (
	"errors"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestSpectatorLinkSurvivesLobbySeating(t *testing.T) {
	for _, self := range []bool{true, false} {
		name := "admin-seats"
		if self {
			name = "phone-rejoins"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, nil)
			admin, player := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player")
			code := f.create(admin, 3, wire.LobbyOptions{})
			f.join(player, code)
			_, seatClaim := f.link(player, code, user(1), false)
			remove := edit(code, 1, "remove-human")
			remove.UserID = ptr(user(1))
			f.send(admin, remove)
			admin.acked(1)
			_, old := f.link(player, code, user(1), false)
			_, current := f.link(player, code, user(1), true)
			phone := f.conn("phone", "phone", "Phone")
			if err := f.m.JoinClaim(f.ctx(), phone.c, code, &current); err != nil {
				t.Fatal(err)
			}
			by, seq := admin, 2
			if self {
				by, seq = phone, 1
			}
			req := edit(code, seq, "seat")
			req.UserID = ptr(user(1))
			req.Seat = ptr(1)
			f.send(by, req)
			by.acked(seq)
			// Fetching a link succeeds only while the phone remains attached.
			f.send(phone, wire.PlatformLink{T: "link", Code: code, Seat: -1, UserID: ptr(user(1))})
			linked := phone.until("still linked after seating", func(v any) bool { _, ok := v.(wire.PlatformLinked); return ok }).(wire.PlatformLinked)
			if linked.Claim != current {
				t.Fatal("seating changed current claim")
			}
			if f.record(code).Slots[1].Epoch == 0 {
				t.Fatal("unexpected epoch rollback")
			}
			for _, dead := range []string{seatClaim, old} {
				d := f.conn("revoked", "revoked", "Revoked")
				if err := f.m.JoinClaim(f.ctx(), d.c, code, &dead); !errors.Is(err, room.ErrBadClaim) {
					t.Fatal("seating revived reset link")
				}
			}
			req = edit(code, seq+1, "release")
			f.send(phone, req)
			phone.until("release from linked seat", func(v any) bool { e, ok := v.(wire.Error); return ok && e.Code == wire.ErrorCodeBadClaim })
			if r := f.record(code); r.Slots[1].Kind != SlotOpen {
				t.Fatal("linked phone could not act in its seat")
			}
		})
	}
}

func TestTemporaryLobbyAdminCannotLinkRealAdmins(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	creator, temp, real, watch := f.conn(user(0), "creator", "Creator"), f.conn(user(1), "temp", "Temp"), f.conn(user(2), "real", "Real"), f.conn(user(3), "watch", "Watch")
	code := f.create(creator, 3, wire.LobbyOptions{})
	for _, k := range []*conn{temp, real, watch} {
		f.join(k, code)
	}
	f.send(creator, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(2))})
	creator.acked(1)
	a := f.lobbies.lookup(code)
	if err := a.call(f.ctx(), func() { a.rec.Roles.History = []string{user(1)} }); err != nil {
		t.Fatal(err)
	}
	f.m.Leave(creator.c, code)
	f.m.Leave(real.c, code)
	f.settle(code)
	clk.Advance(5 * time.Minute)
	f.settle(code)
	for _, target := range []string{user(0), user(2)} {
		for _, reset := range []bool{false, true} {
			if out, claim := f.link(temp, code, target, reset); out != outNotAdmin || claim != "" {
				t.Fatal("temporary admin got real-admin link")
			}
		}
	}
	// Read a fresh per-viewer frame, without consuming an unrelated queued frame.
	var shown, other wire.LobbyState
	if err := a.call(f.ctx(), func() {
		b := a.newBroadcast()
		shown = b.frame(temp.c, nil)
		other = b.frame(watch.c, nil)
		// Reading the restricted viewer again must not alter the frame already queued.
		_ = b.frame(temp.c, nil)
	}); err != nil {
		t.Fatal(err)
	}
	for _, h := range shown.Humans {
		if (h.UserID == user(0) || h.UserID == user(2)) && h.Linkable != nil {
			t.Fatal("blocked link button still offered")
		}
	}
	for _, h := range other.Humans {
		if h.UserID == user(0) && h.Linkable == nil {
			t.Fatal("viewer hint affected other viewers")
		}
	}
	if out, _ := f.link(temp, code, user(1), false); out != outAccepted {
		t.Fatal("temporary admin lost own link")
	}
	f.send(temp, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(3))})
	temp.acked(1)
	if out, _ := f.link(watch, code, user(1), false); out != outAccepted {
		t.Fatal("temporary deputy lost recovery for temporary user")
	}
	f.join(creator, code)
	if out, _ := f.link(temp, code, user(2), false); out != outNotAdmin {
		t.Fatal("stale temporary authority survived real return")
	}
	if out, _ := f.link(creator, code, user(2), false); out != outAccepted {
		t.Fatal("real admin recovery was blocked")
	}
}

func TestConsecutiveLobbySelfResetsKeepDisclosure(t *testing.T) {
	f := newFixture(t, nil)
	admin, player := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player")
	code := f.create(admin, 3, wire.LobbyOptions{})
	f.join(player, code)
	f.link(admin, code, user(1), false)
	a := f.lobbies.lookup(code)
	if err := a.call(f.ctx(), func() {
		entry := a.rec.RoomLog[len(a.rec.RoomLog)-1]
		a.rec.RoomLog = make([]wire.PlatformLogEntry, maxRoomLog)
		for i := range a.rec.RoomLog {
			a.rec.RoomLog[i] = entry
		}
	}); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		if out, _ := f.link(player, code, user(1), true); out != outAccepted {
			t.Fatal(out)
		}
	}
	r := f.record(code)
	copies := 0
	for _, entry := range r.RoomLog {
		if entry.Kind == "link-copy" {
			copies++
		}
	}
	if len(r.RoomLog) != maxRoomLog || copies != maxRoomLog-1 {
		t.Fatal("self-resets evicted disclosure entries")
	}
	for range 2 {
		if out, _ := f.link(admin, code, user(1), false); out != outAccepted {
			t.Fatal(out)
		}
	}
	r = f.record(code)
	for _, entry := range r.RoomLog[len(r.RoomLog)-2:] {
		if entry.Kind != "link-copy" {
			t.Fatal("copies coalesced")
		}
	}
}

func TestSpectatorLinkSurvivesLobbySlotTakenOnJoin(t *testing.T) {
	f := newFixture(t, nil)
	admin, player, watch := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player"), f.conn(user(2), "watch", "Watch")
	code := f.create(admin, 2, wire.LobbyOptions{})
	f.join(player, code)
	f.join(watch, code)
	f.send(admin, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(2))})
	admin.acked(1)
	_, old := f.link(watch, code, user(2), false)
	_, current := f.link(watch, code, user(2), true)
	f.m.Leave(watch.c, code)
	f.settle(code)
	expand := edit(code, 2, "seats")
	expand.SeatCount = ptr(3)
	f.send(admin, expand)
	admin.acked(2)
	f.join(watch, code)
	st := watch.last()
	if st.You.Seat == nil || *st.You.Seat != 2 {
		t.Fatal("native user did not take open slot")
	}
	device := f.conn("device", "device", "Phone")
	if err := f.m.JoinClaim(f.ctx(), device.c, code, &current); err != nil {
		t.Fatal("taking slot revoked current spectator link")
	}
	if st := device.last(); st.You.Seat == nil || *st.You.Seat != 2 {
		t.Fatal("phone lost seat after native join")
	}
	dead := f.conn("dead", "dead", "Old")
	if err := f.m.JoinClaim(f.ctx(), dead.c, code, &old); !errors.Is(err, room.ErrBadClaim) {
		t.Fatal("native join revived reset link")
	}
}
