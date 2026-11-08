package lobbyroom

import (
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestLobbySpectatorLinksEndOnlyOnSuccessfulDuelStart(t *testing.T) {
	f := newFixture(t, nil)
	admin, player, watch := f.conn(user(0), "admin", "Admin"), f.conn(user(1), "player", "Player"), f.conn(user(2), "watch", "Watch")
	code := f.create(admin, 2, wire.LobbyOptions{})
	f.join(player, code)
	f.join(watch, code)
	f.send(admin, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(2))})
	admin.acked(1)
	_, seatClaim := f.link(player, code, user(1), true)
	_, watchClaim := f.link(watch, code, user(2), false)
	device, watchDevice := f.conn("device", "device", "Device"), f.conn("watch-device", "watch-device", "Watch device")
	for k, cl := range map[*conn]string{device: seatClaim, watchDevice: watchClaim} {
		if err := f.m.JoinClaim(f.ctx(), k.c, code, &cl); err != nil {
			t.Fatal(err)
		}
	}
	f.replaceFails = true
	f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2})
	if admin.reject(2) != outStorage {
		t.Fatal("expected failed Start")
	}
	if out, _ := f.link(watchDevice, code, user(2), false); out != outAccepted {
		t.Fatal("failed Start detached watching claim")
	}
	f.replaceFails = false
	f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2})
	st := duelFrame(device)
	if len(st.You.Seats) != 1 || st.You.Seats[0] != 1 {
		t.Fatal("seated claim lost at Duel Start")
	}
	sw := duelFrame(watch)
	if !sw.You.Admin || len(sw.You.Seats) != 0 {
		t.Fatal("ordinary spectating admin lost hint powers")
	}
	e := watchDevice.until("ended spectator claim", func(v any) bool { _, ok := v.(wire.Error); return ok }).(wire.Error)
	if e.Code != wire.ErrorCodeBadClaim || e.Msg != "lobby-link-ended" {
		t.Fatalf("wrong explanation: %+v", e)
	}
	fresh := f.conn("fresh", "fresh", "Fresh")
	if err := f.m.JoinClaim(f.ctx(), fresh.c, code, &watchClaim); !errors.Is(err, room.ErrBadClaim) {
		t.Fatal("unseated claim became Duel claim")
	}
	seated := f.conn("seated", "seated", "Seated")
	if err := f.m.JoinClaim(f.ctx(), seated.c, code, &seatClaim); err != nil {
		t.Fatal(err)
	}
}
