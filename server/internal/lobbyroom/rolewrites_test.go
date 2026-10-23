package lobbyroom

import (
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestLobbyRolesRefusedWrites: the store refuses each of the
// temporary-admin rule's changes in turn (the grant, a real admin's
// return, the passing on). Until a retry saves the change, the temporary
// role fails closed: the user it would demote or replace may not edit or
// Start, the frames show no temporary admin the lobby would not obey, and
// the user it would make one has no powers either. Each retry makes the
// change once.
func TestLobbyRolesRefusedWrites(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	admin := f.conn(user(0), "c0", "P0")
	code := f.create(admin, 4, wire.LobbyOptions{})
	p1, p2 := f.conn(user(1), "c1", "P1"), f.conn(user(2), "c2", "P2")
	for _, k := range []*conn{p1, p2} {
		f.join(k, code)
		k.lobby("joined", any2)
	}
	seq := map[*conn]int{}
	// options runs an options edit from k and returns its outcome.
	options := func(k *conn) string {
		seq[k]++
		req := edit(code, seq[k], "options")
		req.Options = &wire.LobbyOptions{}
		a := f.lobbies.lookup(code)
		var out string
		if err := a.call(f.ctx(), func() { out = a.edit(k.c, req) }); err != nil {
			t.Fatal(err)
		}
		return out
	}
	temporary := func() string {
		f.settle(code)
		return f.record(code).Roles.Temporary
	}
	shown := func(k *conn, u string) bool {
		f.settle(code)
		st := k.last()
		i := slices.IndexFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.UserID == u })
		return i >= 0 && st.Humans[i].Temporary
	}
	f.m.Leave(admin.c, code)
	f.settle(code)

	// The grant is refused: user-1 has no powers until the retry.
	f.failSaves(code, 0, 1)
	clk.Advance(5 * time.Minute)
	if out := options(p1); out != outNotAdmin || temporary() != "" {
		t.Fatalf("after a refused grant: %s, temporary %q", out, temporary())
	}
	clk.Advance(roles.Retry)
	if temporary() != user(1) || !shown(p2, user(1)) {
		t.Fatalf("the retried grant: %q", temporary())
	}
	if out := options(p1); out != outAccepted {
		t.Fatalf("the temporary admin's edit: %s", out)
	}

	// The creator's return is refused: user-1 loses the powers at once,
	// and no frame calls them temporary.
	f.failSaves(code, 0, 1)
	f.join(admin, code)
	if out := options(p1); out != outNotAdmin {
		t.Fatalf("a demoted temporary admin's edit: %s", out)
	}
	f.send(p1, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 100, FillBots: true})
	if out := p1.reject(100); out != outNotAdmin {
		t.Fatalf("a demoted temporary admin's Start: %s", out)
	}
	if temporary() != user(1) || shown(p2, user(1)) {
		t.Fatal("the frames show a temporary admin the lobby no longer obeys")
	}
	clk.Advance(roles.Retry)
	if temporary() != "" {
		t.Fatalf("the retried demotion: %q", temporary())
	}

	// The passing on is refused, and user-1 comes back before the retry,
	// whose write is refused too: neither user-1 nor user-2 has the powers
	// until a write saves user-2's.
	f.m.Leave(admin.c, code)
	f.settle(code)
	clk.Advance(5 * time.Minute)
	if temporary() != user(1) {
		t.Fatalf("the second grant: %q", temporary())
	}
	f.m.Leave(p1.c, code)
	f.settle(code)
	f.failSaves(code, 0, 2)
	clk.Advance(5 * time.Minute)
	f.join(p1, code)
	if out1, out2 := options(p1), options(p2); out1 != outNotAdmin || out2 != outNotAdmin || temporary() != user(1) {
		t.Fatalf("after a refused passing on: user-1 %s, user-2 %s, temporary %q", out1, out2, temporary())
	}
	clk.Advance(roles.Retry)
	if temporary() != user(2) {
		t.Fatalf("the retried passing on: %q", temporary())
	}
	if out1, out2 := options(p1), options(p2); out1 != outNotAdmin || out2 != outAccepted {
		t.Fatalf("after the passing on: user-1 %s, user-2 %s", out1, out2)
	}
	var log []string
	for _, e := range f.record(code).RoomLog {
		if e.Kind == "role" {
			log = append(log, *e.UserID+"="+*e.Role)
		}
	}
	want := []string{"user-1=temporary", "user-1=human", "user-1=temporary", "user-1=human", "user-2=temporary"}
	if !slices.Equal(log, want) {
		t.Fatalf("role log %v, want %v", log, want)
	}
}
