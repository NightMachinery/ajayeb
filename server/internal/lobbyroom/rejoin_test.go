package lobbyroom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestLobbySelfRejoin(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "admin", "A")
	p := f.conn(user(1), "player", "P")
	w := f.conn(user(2), "watcher", "W")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(p, code)
	f.join(w, code)
	remove := edit(code, 1, "remove-human")
	remove.UserID = ptr(user(1))
	f.send(a, remove)
	a.acked(1)
	st := p.last()
	if st.You.Seat != nil || st.You.RejoinSeat == nil || *st.You.RejoinSeat != 1 {
		t.Fatalf("offer: %+v", st.You)
	}
	if w.last().You.RejoinSeat != nil {
		t.Fatal("another viewer received the offer")
	}
	req := edit(code, 1, "seat")
	req.UserID = ptr(user(1))
	req.Seat = ptr(1)
	f.send(w, req)
	if got := w.reject(1); got != outNotAdmin {
		t.Fatalf("other user: %s", got)
	}
	// Offered availability is rechecked after an admin seats another human.
	seat := edit(code, 2, "seat")
	seat.UserID = ptr(user(2))
	seat.Seat = ptr(1)
	f.send(a, seat)
	a.acked(2)
	f.send(p, req)
	if got := p.reject(1); got != outNotAdmin {
		t.Fatalf("human occupied seat: %s", got)
	}
	remove = edit(code, 3, "remove-human")
	remove.UserID = ptr(user(2))
	f.send(a, remove)
	a.acked(3)
	bot := edit(code, 4, "add-bot")
	bot.Seat = ptr(1)
	bot.Bot = ptr(tableroom.BotEasy)
	f.send(a, bot)
	a.acked(4)
	f.send(p, req)
	if got := p.reject(1); got != outNotAdmin {
		t.Fatalf("bot in lobby seat: %s", got)
	}
	rm := edit(code, 5, "remove-bot")
	rm.Seat = ptr(1)
	f.send(a, rm)
	a.acked(5)
	f.restart()
	p = f.conn(user(1), "reconnect", "P")
	f.join(p, code)
	if p.last().You.RejoinSeat == nil {
		t.Fatal("restart lost former seat")
	}
	f.send(p, req)
	st = p.acked(1)
	if st.You.Seat == nil || *st.You.Seat != 1 || st.You.RejoinSeat != nil {
		t.Fatalf("after rejoin: %+v", st.You)
	}
	last := st.RoomLog[len(st.RoomLog)-1]
	if last.Kind != "seat" || last.By != user(1) || last.UserID == nil || *last.UserID != user(1) || last.Seat == nil || *last.Seat != 1 {
		t.Fatalf("log: %+v", last)
	}
	revision := st.Revision
	f.send(p, req)
	if p.acked(1).Revision != revision {
		t.Fatal("retry repeated rejoin")
	}
}

func TestFormat4LobbyPreservesMetadata(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "River")
	b := f.conn(user(1), "b", "River")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	// A name that leaves and returns must not get initialized from slot order.
	renamed := f.conn(user(0), "rename", "Brook")
	f.join(renamed, code)
	again := f.conn(user(0), "again", "River")
	f.join(again, code)
	before := f.record(code)
	before.Format = 4
	data, err := before.encode()
	if err != nil {
		t.Fatal(err)
	}
	// Remove format-5 additions, as the shipped format-4 writer did.
	var old map[string]json.RawMessage
	if err = json.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	delete(old, "previous")
	var roles map[string]json.RawMessage
	if err = json.Unmarshal(old["roles"], &roles); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"deputies", "temporaryBy", "history", "declined", "elections"} {
		delete(roles, key)
	}
	old["roles"], _ = json.Marshal(roles)
	data, _ = json.Marshal(old)
	after, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = validate(after, defaultSeqClients); err != nil {
		t.Fatal(err)
	}
	if after.Format != recordFormat || !reflect.DeepEqual(before.Names, after.Names) || !reflect.DeepEqual(before.Seqs, after.Seqs) {
		t.Fatal("migration reset names or client keys")
	}
	f.close()
	if err = os.WriteFile(filepath.Join(f.st.Dir(), code+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	f.start()
	b = f.conn(user(1), "load", "River")
	f.join(b, code)
	if got := b.last(); got.Humans[0].NameNo != before.Names.Get(user(0)).Number {
		t.Fatal("loaded suffix changed")
	}
}

// Captured by the unchanged format-4 writer at 0b948662, using fictional
// names and test identities. The creator reclaimed a duplicate name.
func TestFormat4LobbyFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/format-4-roles.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = validate(r, defaultSeqClients); err != nil {
		t.Fatal(err)
	}
	if r.Format != recordFormat || r.Names.Get("user-0").Number != 3 || r.Names.Get("user-1").Number != 2 {
		t.Fatalf("name ledger reset: %+v", r.Names)
	}
}

func TestFormat4RetiredCreator(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.send(a, edit(code, 1, "release"))
	a.acked(1)
	old := f.record(code)
	old.Format = 4
	old.Roles.Admins = slices.DeleteFunc(old.Roles.Admins, func(a roles.Admin) bool { return a.UserID == user(0) })
	old.Retired = append(old.Retired, user(0))
	data, err := old.encode()
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = validate(r, defaultSeqClients); err != nil {
		t.Fatal(err)
	}
	if !r.Roles.Real(user(0)) || slices.Contains(r.Retired, user(0)) {
		t.Fatal("creator was not restored out of retirement")
	}
}
