package lobbyroom

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func (f *fixture) link(k *conn, code, target string, reset bool) (string, string) {
	f.t.Helper()
	a := f.lobbies.lookup(code)
	var out string
	if err := a.call(f.ctx(), func() { out = a.link(k.c, -1, &target, reset) }); err != nil {
		f.t.Fatal(err)
	}
	if out != outAccepted {
		return out, ""
	}
	linked := k.until("linked", func(v any) bool { _, ok := v.(wire.PlatformLinked); return ok }).(wire.PlatformLinked)
	if linked.UserID == nil || *linked.UserID != target || linked.Seat != -1 {
		f.t.Fatalf("linked: %+v", linked)
	}
	return out, linked.Claim
}

func TestLobbyDeviceLinks(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	admin, p1, p2, spectator := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "p1", "One"), f.conn(user(2), "p2", "Two"), f.conn(user(3), "spectator", "Watcher")
	code := f.create(admin, 3, wire.LobbyOptions{})
	for _, k := range []*conn{p1, p2, spectator} {
		f.join(k, code)
	}
	if out, _ := f.link(spectator, code, user(3), false); out != outBad {
		t.Fatal("plain spectator got a device link")
	}
	if out, _ := f.link(spectator, code, user(1), false); out != outNotAdmin {
		t.Fatal("spectator got another link")
	}
	if out, _ := f.link(p1, code, user(2), false); out != outNotAdmin {
		t.Fatal("non-admin got another link")
	}
	_, own := f.link(p1, code, user(1), false)
	_, copied := f.link(admin, code, user(1), false)
	if own != copied {
		t.Fatal("copy changed claim")
	}
	for _, k := range []*conn{admin, p1, p2, spectator} {
		f.settle(code)
		st := k.last()
		e := st.RoomLog[len(st.RoomLog)-1]
		if e.Kind != "link-copy" || e.By != user(0) || e.UserID == nil || *e.UserID != user(1) {
			t.Fatalf("public log: %+v", e)
		}
		b, _ := json.Marshal(st)
		if strings.Contains(string(b), copied) {
			t.Fatal("claim in public snapshot")
		}
	}
	device := f.conn("phone", "phone", "Phone")
	if err := f.m.JoinClaim(f.ctx(), device.c, code, &own); err != nil {
		t.Fatal(err)
	}
	_, next := f.link(p1, code, user(1), true)
	if next == own {
		t.Fatal("reset did not change claim")
	}
	device.until("revoked", func(v any) bool { e, ok := v.(wire.Error); return ok && e.Code == wire.ErrorCodeBadClaim })
	f.send(device, edit(code, 1, "release"))
	if device.reject(1) != outBad {
		t.Fatal("revoked session remained attached")
	}
	// A claimed connection may rotate itself without being detached.
	fresh := f.conn("fresh", "fresh", "Fresh")
	if err := f.m.JoinClaim(f.ctx(), fresh.c, code, &next); err != nil {
		t.Fatal(err)
	}
	_, newest := f.link(fresh, code, user(1), true)
	st := fresh.last()
	if st.You.UserID != user(1) || st.You.Seat == nil {
		t.Fatal("reset detached its requester")
	}
	other := f.create(f.conn(user(9), "other", "Other"), 3, wire.LobbyOptions{})
	wrong := f.conn("wrong", "wrong", "Wrong")
	if err := f.m.JoinClaim(f.ctx(), wrong.c, other, &newest); !errors.Is(err, room.ErrBadClaim) {
		t.Fatal("claim crossed rooms")
	}
	// A refused durable disclosure sends neither a claim nor a public event.
	before := f.record(code)
	f.failSaves(code, 0, 1)
	if out, claim := f.link(admin, code, user(2), false); out != outStorage || claim != "" {
		t.Fatal("failed write disclosed claim")
	}
	after := f.record(code)
	if after.Revision != before.Revision || !reflect.DeepEqual(after.RoomLog, before.RoomLog) {
		t.Fatal("failed write changed state")
	}
	// Temporary admins have the same recovery power, and lose it on real return.
	a := f.lobbies.lookup(code)
	if err := a.call(f.ctx(), func() { a.rec.Roles.History = []string{user(1)} }); err != nil {
		t.Fatal(err)
	}
	f.m.Leave(admin.c, code)
	f.settle(code)
	clk.Advance(5 * time.Minute)
	f.settle(code)
	if out, _ := f.link(p1, code, user(2), false); out != outAccepted {
		t.Fatalf("temporary admin: %s", out)
	}
	f.join(admin, code)
	if out, _ := f.link(p1, code, user(2), false); out != outNotAdmin {
		t.Fatal("returned admin did not revoke temporary power")
	}
}

func TestLobbyLinksCarryThroughStart(t *testing.T) {
	for _, order := range []string{"lobby", "random"} {
		t.Run(order, func(t *testing.T) {
			f := newFixture(t, nil)
			admin, player, third, watcher := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player"), f.conn(user(2), "third", "Third"), f.conn(user(3), "watcher", "Watcher")
			code := f.create(admin, 3, wire.LobbyOptions{Seating: &order})
			for _, k := range []*conn{player, third, watcher} {
				f.join(k, code)
			}
			f.send(admin, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(3))})
			admin.acked(1)
			_, old := f.link(player, code, user(1), false)
			_, current := f.link(player, code, user(1), true)
			_, watchClaim := f.link(watcher, code, user(3), false)
			// Exercise the actual new user-targeted dispatch too.
			f.send(admin, wire.PlatformLink{T: "link", Code: code, Seat: -1, UserID: ptr(user(1))})
			admin.until("linked dispatch", func(v any) bool { _, ok := v.(wire.PlatformLinked); return ok })
			device := f.conn("device", "device", "Device")
			if err := f.m.JoinClaim(f.ctx(), device.c, code, &current); err != nil {
				t.Fatal(err)
			}
			f.restart()
			f.join(admin, code)
			f.join(player, code)
			f.join(third, code)
			if err := f.m.JoinClaim(f.ctx(), device.c, code, &current); err != nil {
				t.Fatal(err)
			}
			watchDevice := f.conn("watch-device", "watch-device", "Watch device")
			if err := f.m.JoinClaim(f.ctx(), watchDevice.c, code, &watchClaim); err != nil {
				t.Fatal(err)
			}
			f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2, FillBots: true})
			st := device.table("device hand after Start", func(wire.TableState) bool { return true })
			if st.You.UserID != user(1) || st.You.Seat == nil || st.View.You == nil {
				t.Fatalf("handoff: %+v", st.You)
			}
			if len(st.RoomLog) != 3 || st.RoomLog[2].Kind != "link-copy" {
				t.Fatalf("lost log at Start: %+v", st.RoomLog)
			}
			watchFrame := watchDevice.table("watching admin after Start", func(wire.TableState) bool { return true })
			if watchFrame.You.UserID != user(3) || watchFrame.You.Seat != nil {
				t.Fatal("watcher identity lost at Start")
			}
			for _, cl := range []string{current, watchClaim} {
				d := f.conn("after-"+cl, "after-"+cl, "After")
				if err := f.m.JoinClaim(f.ctx(), d.c, code, &cl); err != nil {
					t.Fatalf("copied link after Start: %v", err)
				}
			}
			d := f.conn("old", "old", "Old")
			if err := f.m.JoinClaim(f.ctx(), d.c, code, &old); !errors.Is(err, room.ErrBadClaim) {
				t.Fatal("revoked lobby claim revived after Start")
			}
		})
	}
}

func TestFormat5LobbyLinksLoad(t *testing.T) {
	f := newFixture(t, nil)
	admin := f.conn(user(0), "admin", "Admin")
	code := f.create(admin, 3, wire.LobbyOptions{})
	r := f.record(code)
	r.Format, r.Links = 5, nil
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.Format != 6 || !reflect.DeepEqual(upgraded.Slots, r.Slots) || !reflect.DeepEqual(upgraded.Names, r.Names) {
		t.Fatal("format 5 metadata changed")
	}
}

func TestWatchingDeviceDoesNotTakeNewLobbySlot(t *testing.T) {
	f := newFixture(t, nil)
	admin, player, watch := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player"), f.conn(user(2), "watch", "Watch")
	code := f.create(admin, 2, wire.LobbyOptions{})
	f.join(player, code)
	f.join(watch, code)
	f.send(admin, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(2))})
	admin.acked(1)
	_, claim := f.link(watch, code, user(2), false)
	expand := edit(code, 2, "seats")
	expand.SeatCount = ptr(3)
	f.send(admin, expand)
	admin.acked(2)
	device := f.conn("device", "device", "Different name")
	if err := f.m.JoinClaim(f.ctx(), device.c, code, &claim); err != nil {
		t.Fatal(err)
	}
	st := device.last()
	if st.You.UserID != user(2) || st.You.Seat != nil || f.record(code).Names.Get(user(2)).Name != "Watch" {
		t.Fatal("spectator device took a slot or renamed its identity")
	}
	if out, got := f.link(device, code, user(2), false); out != outAccepted || got != claim {
		t.Fatal("spectator join changed its claim")
	}
}

func TestLinkLogAfterLobbyShrinks(t *testing.T) {
	f := newFixture(t, nil)
	admin := f.conn(user(0), "admin", "Admin")
	code := f.create(admin, 4, wire.LobbyOptions{})
	for i := 1; i < 4; i++ {
		f.join(f.conn(user(i), "p"+user(i), "Player"), code)
	}
	f.link(admin, code, user(3), false)
	shrink := edit(code, 1, "seats")
	shrink.SeatCount = ptr(3)
	f.send(admin, shrink)
	admin.acked(1)
	f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2, FillBots: true})
	st := admin.table("smaller table", func(wire.TableState) bool { return true })
	for _, e := range st.RoomLog {
		if e.Seat != nil && (*e.Seat < 0 || *e.Seat >= 3) {
			t.Fatalf("invalid historical seat: %+v", e)
		}
	}
}
