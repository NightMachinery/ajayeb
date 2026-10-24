package lobbyroom

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// finished starts a three-seat table from a lobby of two humans (user-0
// the creator, user-1) and a bot, in lobby order, and plays it to its end:
// each human confirms its first choice whenever its seat is required.
func (f *fixture) finished() (string, *conn, *conn) {
	f.t.Helper()
	k0, k1 := f.conn(user(0), "c0", "P0"), f.conn(user(1), "c1", "P1")
	code := f.create(k0, 3, wire.LobbyOptions{Seating: ptr("lobby")})
	f.join(k1, code)
	k1.lobby("user-1 in the lobby", any2)
	f.send(k0, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
	ks := []*conn{k0, k1}
	cur := map[*conn]wire.TableState{}
	for _, k := range ks {
		cur[k] = k.table("the table", func(wire.TableState) bool { return true })
	}
	seq := 0
	deadline := time.Now().Add(wait)
	for cur[k0].Status != "finished" {
		if time.Now().After(deadline) {
			f.t.Fatal("the table did not finish")
		}
		for _, k := range ks {
			k.drain()
			for _, m := range k.frames {
				if st, ok := m.(wire.TableState); ok {
					cur[k] = st
				}
			}
			k.frames = nil
		}
		acted := false
		for _, k := range ks {
			st := cur[k]
			if st.Status != "playing" || st.Point == nil || st.You.Seat == nil || st.View.You == nil || len(st.View.You.Choices) == 0 {
				continue
			}
			s, ref := *st.You.Seat, st.Point.Ref
			if !slices.Contains(st.Point.Required, s) || slices.Contains(st.Point.Confirmed, s) {
				continue
			}
			seq++
			f.send(k, wire.TablePick{T: "table-pick", Code: code, Seq: seq, Point: ref, Key: st.View.You.Choices[0].Key, Confirm: true})
			cur[k] = k.table("the pick's answer", func(n wire.TableState) bool {
				return n.Status == "finished" || n.Point == nil || n.Point.Ref != ref || slices.Contains(n.Point.Confirmed, s)
			})
			acted = true
		}
		if !acted && cur[k0].Status != "finished" {
			// Shown only if the wait fails: what each player saw then.
			for i, k := range ks {
				f.t.Logf("player %d before the wait: %s", i, tableSummary(cur[k]))
			}
			cur[k0] = k0.table("the next frame", func(wire.TableState) bool { return true })
		}
	}
	return code, k0, k1
}

// tableSummary is the part of a table frame that says who must act.
func tableSummary(st wire.TableState) string {
	out := fmt.Sprintf("status=%s batch=%d", st.Status, st.Batch)
	if st.Point != nil {
		out += fmt.Sprintf(" point=%s required=%v confirmed=%v", st.Point.Kind, st.Point.Required, st.Point.Confirmed)
	}
	if st.You.Seat != nil {
		out += fmt.Sprintf(" seat=%d", *st.You.Seat)
	}
	if st.View.You != nil {
		out += fmt.Sprintf(" choices=%d", len(st.View.You.Choices))
	}
	return out
}

// tableRecord reads the table record of code from the store.
func (f *fixture) tableRecord(code string) tableroom.Record {
	f.t.Helper()
	data, _, err := f.st.PeekGame(code, tableroom.Game)
	if err != nil {
		f.t.Fatal(err)
	}
	var r tableroom.Record
	if err := json.Unmarshal(data, &r); err != nil {
		f.t.Fatal(err)
	}
	return r
}

// successors lists the lobby records whose parent is code.
func (f *fixture) successors(code string) []string {
	f.t.Helper()
	paths, err := filepath.Glob(filepath.Join(f.st.Dir(), "*.json"))
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			f.t.Fatal(err)
		}
		if r, err := decodeRecord(data); err == nil && r.Parent == code {
			out = append(out, r.Code)
		}
	}
	return out
}

// TestRematchOneSuccessor: a finished table's rematch writes three times
// (the table reserves a code, the lobby is made under it, the table
// records it). However those writes fail, and whoever of the original
// humans asks again after a restart, the table has one successor lobby,
// made for the first requester.
func TestRematchOneSuccessor(t *testing.T) {
	storage := func(v any) bool { r, ok := v.(wire.TableReject); return ok && r.Code == outStorage }
	created := func(v any) bool { _, ok := v.(wire.Created); return ok }
	for _, tc := range []struct {
		name string
		// key and pass arrange the failure: the failSaves key (the table's
		// code, or the lobby game for the lobby's write) and the writes it
		// lets through first.
		key  func(code string) string
		pass int
		// reserved: whether the table holds a reservation after it.
		reserved bool
		made     int
	}{
		{name: "the reservation", key: func(code string) string { return code }},
		{name: "the lobby", key: func(string) string { return Game }, reserved: true},
		{name: "the successor", key: func(code string) string { return code }, pass: 1, reserved: true, made: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, nil)
			code, k0, k1 := f.finished()
			f.failSaves(tc.key(code), tc.pass, 1)
			if _, err := f.m.Rematch(f.ctx(), k0.c, code); err != nil {
				t.Fatal(err)
			}
			k0.until("the storage refusal", storage)
			src := f.tableRecord(code)
			if (src.Reserved != "") != tc.reserved || src.Successor != "" || len(f.successors(code)) != tc.made {
				t.Fatalf("after the refused write: reserved %q, successor %q, %d made", src.Reserved, src.Successor, len(f.successors(code)))
			}
			if tc.reserved && src.Rematcher != user(0) {
				t.Fatalf("the rematcher: %q", src.Rematcher)
			}
			// A crash between the writes is a restart after the refusal.
			f.restart()
			f.join(k1, code)
			if _, err := f.m.Rematch(f.ctx(), k1.c, code); err != nil {
				t.Fatal(err)
			}
			next := k1.until("the created frame", created).(wire.Created)
			made := f.successors(code)
			if len(made) != 1 || made[0] != next.Code || (tc.reserved && next.Code != src.Reserved) {
				t.Fatalf("successors %v, created %s, reserved %s", made, next.Code, src.Reserved)
			}
			// The creator is who reserved the code: user-0 when the first
			// request did, user-1 when only the retry could.
			creator := user(1)
			if tc.reserved {
				creator = user(0)
			}
			if r := f.record(next.Code); r.Roles.Creator != creator {
				t.Fatalf("the successor's creator: %q, want %q", r.Roles.Creator, creator)
			}
			if src := f.tableRecord(code); src.Successor != next.Code || src.Reserved != "" || src.Rematcher != "" {
				t.Fatalf("the table after the rematch: %q %q %q", src.Successor, src.Reserved, src.Rematcher)
			}
			// Asked again by the first requester: the same code, nothing made.
			f.join(k0, code)
			if _, err := f.m.Rematch(f.ctx(), k0.c, code); err != nil {
				t.Fatal(err)
			}
			if again := k0.until("the created frame again", created).(wire.Created); again.Code != next.Code || len(f.successors(code)) != 1 {
				t.Fatalf("again: %s, successors %v", again.Code, f.successors(code))
			}
		})
	}
}

// TestRematchCodeTaken: Rematch refuses a reserved code another room
// holds (a lobby of no parent or another, here), and writes nothing.
func TestRematchCodeTaken(t *testing.T) {
	f := newFixture(t, nil)
	k := f.conn(user(0), "c0", "P0")
	code := f.create(k, 3, wire.LobbyOptions{})
	parent, err := f.m.FreshCode()
	if err != nil {
		t.Fatal(err)
	}
	before := f.record(code)
	rm := tableroom.Rematch{Parent: parent, Creator: user(0), Slots: []tableroom.StartSlot{{UserID: user(0), Name: "P0"}, {Bot: tableroom.BotEasy}, {Bot: tableroom.BotEasy}}}
	if err := f.lobbies.Rematch(f.m, code, rm); !errors.Is(err, tableroom.ErrCodeTaken) {
		t.Fatalf("a taken code: %v", err)
	}
	if after := f.record(code); after.Parent != "" || after.Revision != before.Revision {
		t.Fatalf("the lobby was rewritten: %+v", after)
	}
}
