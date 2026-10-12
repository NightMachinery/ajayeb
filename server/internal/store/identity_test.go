package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// TestIdentityFieldsPersisted checks that the seats' name numbers and link
// epochs and a link reset's room log entry round trip, and that they are
// not part of the game: the hash is the same with and without them.
func TestIdentityFieldsPersisted(t *testing.T) {
	st := newStore(t)
	seats := [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "u-a", Name: "Sara"},
		{Kind: store.SeatKindHuman, UserID: "u-b", Name: "sara"},
	}
	plain := store.New("DXNPMN", store.ModeOnline, engine.Options{}, 1, time.Now())
	plain.Seats = seats
	if err := st.Save(plain); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f := store.New("DXNSVD", store.ModeOnline, engine.Options{}, 1, time.Now())
	f.Seats = seats
	f.Seats[0].Epoch = 3
	f.Seats[1].NameNo = 2
	target := 0
	f.RoomLog = []store.LogEntry{{Kind: "link", At: 0, Seat: 0, On: []string{}, Off: []string{}, Target: &target}}
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if f.Hash != plain.Hash {
		t.Fatalf("hash %s with identity fields, %s without: they must not be part of the game", f.Hash, plain.Hash)
	}
	got, _, err := st.Load("DXNSVD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Seats[0].Epoch != 3 || got.Seats[0].NameNo != 0 || got.Seats[1].NameNo != 2 || got.Seats[1].Epoch != 0 {
		t.Fatalf("Load: seats %+v", got.Seats)
	}
	if len(got.RoomLog) != 1 || got.RoomLog[0].Kind != "link" || got.RoomLog[0].Target == nil || *got.RoomLog[0].Target != 0 {
		t.Fatalf("Load: room log %+v", got.RoomLog)
	}
}

// TestIdentityFieldsAbsentInOldFile checks that a room file written before
// the identity fields existed loads with every seat at number 0 and epoch
// 0, and that a file whose seats have neither leaves the keys out.
func TestIdentityFieldsAbsentInOldFile(t *testing.T) {
	st := newStore(t)
	f := store.New("DXNQKD", store.ModeOnline, engine.Options{}, 1, time.Now())
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "u-a", Name: "Sara"},
		{Kind: store.SeatKindHuman, UserID: "u-b", Name: "Sara"},
	}
	f.RoomLog = []store.LogEntry{{Kind: "hints", At: 0, Seat: 0, On: []string{"advisor"}, Off: []string{}}}
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(st.Dir(), "DXNQKD.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, key := range []string{`"nameNo"`, `"epoch"`, `"target"`} {
		if strings.Contains(string(data), key) {
			t.Fatalf("a file without identity fields has the key %s:\n%s", key, data)
		}
	}
	got, _, err := st.Load("DXNQKD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for i, s := range got.Seats {
		if s.NameNo != 0 || s.Epoch != 0 {
			t.Fatalf("Load: seat %d %+v", i, s)
		}
	}
	if got.RoomLog[0].Target != nil {
		t.Fatalf("Load: a hints entry has a target: %+v", got.RoomLog[0])
	}
}
