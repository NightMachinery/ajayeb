package lobbyroom

import (
	"errors"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestRematchClaimsRevoked: a rematch hands each original human a claim
// of epoch 0 for their reserved slot, which holds while the human keeps
// the reservation or the slot it became without a break, into the table
// at Start. Once they lose it (removed, released, the reservation opened,
// the seat count cut), the claim is refused, and it stays refused when
// they are seated again, across a restart and at the table.
func TestRematchClaimsRevoked(t *testing.T) {
	f := newFixture(t, nil)
	parent, err := f.m.FreshCode()
	if err != nil {
		t.Fatal(err)
	}
	code, err := f.lobbies.Reserve(f.m)
	if err != nil {
		t.Fatal(err)
	}
	slots := []tableroom.StartSlot{{UserID: user(0), Name: "P0"}, {UserID: user(1), Name: "P1"}, {UserID: user(2), Name: "P2"},
		{UserID: user(3), Name: "P3"}, {Bot: tableroom.BotEasy}, {UserID: user(4), Name: "P4"}}
	if err := f.lobbies.Rematch(f.m, code, tableroom.Rematch{Parent: parent, Creator: user(0), LobbyOrder: true, Slots: slots}); err != nil {
		t.Fatal(err)
	}
	devices := 0
	// device joins a new connection with u's rematch claim.
	device := func(u string) (*conn, error) {
		devices++
		d := f.conn("device-"+u, "d"+user(devices), "Phone")
		cl := f.secret.Claim(code, u, 0)
		return d, f.m.JoinClaim(f.ctx(), d.c, code, &cl)
	}
	revoked := func(what string, users ...string) {
		t.Helper()
		for _, u := range users {
			if _, err := device(u); !errors.Is(err, room.ErrBadClaim) {
				t.Fatalf("%s: %s's rematch claim: %v", what, u, err)
			}
		}
	}
	// own holds each human's own connection, joined as their user.
	own := map[string]*conn{}
	arrive := func(i int) {
		own[user(i)] = f.conn(user(i), "c"+strconv.Itoa(i), "P"+strconv.Itoa(i))
		f.join(own[user(i)], code)
	}
	for _, i := range []int{0, 1, 3} {
		arrive(i)
	}
	admin := own[user(0)]
	seq := 0
	do := func(k *conn, req wire.LobbyEdit) {
		t.Helper()
		seq++
		req.Seq = seq
		f.send(k, req)
		k.acked(seq)
	}

	// Unbroken: user-0's and user-1's devices take their humans' slots.
	if _, err := device(user(0)); err != nil {
		t.Fatal(err)
	}
	d1, err := device(user(1))
	if err != nil {
		t.Fatal(err)
	}
	// Removed and seated again: the device is ended, the claim stays dead.
	req := edit(code, 0, "remove-human")
	req.UserID = ptr(user(1))
	do(admin, req)
	d1.until("the device's bad-claim", func(v any) bool { e, ok := v.(wire.Error); return ok && e.Code == wire.ErrorCodeBadClaim })
	revoked("removed", user(1))
	req = edit(code, 0, "seat")
	req.Seat, req.UserID = ptr(1), ptr(user(1))
	do(admin, req)
	// Released and seated again.
	rel := edit(code, 1, "release")
	f.send(own[user(3)], rel)
	own[user(3)].acked(1)
	req = edit(code, 0, "seat")
	req.Seat, req.UserID = ptr(3), ptr(user(3))
	do(admin, req)
	// A reservation opened, and its human then joins: an open slot.
	req = edit(code, 0, "open-reservation")
	req.Seat = ptr(2)
	do(admin, req)
	revoked("reservation opened", user(2))
	arrive(2)
	// The seat count cut below a reservation, then raised: its human
	// takes the new open slot.
	req = edit(code, 0, "seats")
	req.SeatCount = ptr(5)
	do(admin, req)
	revoked("seat count cut", user(4))
	req = edit(code, 0, "seats")
	req.SeatCount = ptr(6)
	do(admin, req)
	arrive(4)
	r := f.record(code)
	for slot, u := range map[int]string{0: user(0), 1: user(1), 2: user(2), 3: user(3), 5: user(4)} {
		if s := r.Slots[slot]; s.Kind != SlotHuman || s.UserID != u || (s.Epoch == 0) != (u == user(0)) {
			t.Fatalf("slot %d: %+v", slot, s)
		}
	}
	revoked("seated again", user(1), user(2), user(3), user(4))

	// Across a restart, the same.
	f.restart()
	for _, k := range own {
		f.join(k, code)
	}
	revoked("after a restart", user(1), user(2), user(3), user(4))
	d0, err := device(user(0))
	if err != nil {
		t.Fatalf("the unbroken claim after a restart: %v", err)
	}

	// At the table: the unbroken claim acts as its seat, with its hand;
	// the others are refused.
	f.send(admin, wire.LobbyStart{T: "lobby-start", Code: code, Seq: seq + 1, FillBots: true})
	st := d0.table("the device at the table", func(wire.TableState) bool { return true })
	if st.You.UserID != user(0) || st.You.Seat == nil || *st.You.Seat != 0 || st.View.You == nil {
		t.Fatalf("the unbroken claim at the table: %+v", st.You)
	}
	revoked("at the table", user(1), user(2), user(3), user(4))
	for _, u := range []string{user(1), user(2), user(3), user(4)} {
		if st := own[u].table(u+" at the table", func(wire.TableState) bool { return true }); st.You.Seat == nil {
			t.Fatalf("%s lost their seat", u)
		}
	}
}
