package tableroom

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// clientFramesPath is the recording the web client's table Session and
// Presenter replay (web/src/lib/games/haftgane/session/recorded.test.ts).
const clientFramesPath = "testdata/client-frames.json"

// clientFramesCode stands for the table's random code in the recording.
const clientFramesCode = "TBLAAA"

// clientEntry is one thing a client sent or received, in order. Raw marks
// a request the web Session never makes itself (an illegal key, a
// repeated seq, a pick from a seat it lost); a replay skips it and feeds
// the answer alone.
type clientEntry struct {
	Send json.RawMessage
	Raw  bool
	Recv json.RawMessage
}

// TestRecordClientFrames records what the clients of one table (three
// humans and a spectator) send and receive, in order, through the
// Manager: seat 0's draft, change, confirmation, withdrawal and second
// confirmation; seat 1's illegal key, confirmation, and change on the
// decision that seat 2's confirmation has just closed; seat 0's repeated
// seq; seat 0, the admin, giving seat 1 to a bot; and seat 1's pick from
// the seat it no longer holds. The recording is compared with
// testdata/client-frames.json, or rewritten with UPDATE_CLIENT_FRAMES=1.
func TestRecordClientFrames(t *testing.T) {
	f := newFixture(t, Config{BotWork: blockBots})
	ctx := testContext(t)
	code, err := f.tables.CreateTable(humans(3, 7))
	if err != nil {
		t.Fatal(err)
	}
	order := []string{"seat-0", "seat-1", "seat-2", "watcher"}
	users := map[string]string{}
	clients := map[string]*room.Client{}
	logs := map[string][]clientEntry{}
	latest := map[string]wire.TableState{}
	seqs := map[string]int{}
	picks := map[string]wire.TablePick{}
	take := func() {
		for _, id := range order {
			c := clients[id]
			if c == nil {
				continue
			}
			for _, m := range c.Out.Take() {
				if st, ok := m.(wire.TableState); ok {
					latest[id] = st
				}
				b, err := json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
				logs[id] = append(logs[id], clientEntry{Recv: b})
			}
		}
	}
	barrier := func() {
		a := f.tables.lookup(code)
		if a == nil {
			t.Fatal("the table is not running")
		}
		if err := a.call(ctx, func() {}); err != nil {
			t.Fatal(err)
		}
	}
	// send dispatches msg from id; the answers are taken when settle runs.
	send := func(id string, raw bool, msg wire.ClientMessage) {
		b, err := json.Marshal(msg)
		if err != nil {
			t.Fatal(err)
		}
		logs[id] = append(logs[id], clientEntry{Send: b, Raw: raw})
		if err := f.m.DispatchPlatform(ctx, clients[id], msg); err != nil {
			t.Fatal(err)
		}
	}
	settle := func() {
		barrier()
		take()
	}
	join := func(id, user, name string) {
		c := newClient(user, id, name)
		if err := f.m.Join(ctx, c, code); err != nil {
			t.Fatal(err)
		}
		clients[id], users[id] = c, user
		settle()
	}
	// seatOf is the seat of the client's latest frame, which the Session
	// names in its picks.
	seatOf := func(id string) *int {
		if s := latest[id].You.Seat; s != nil {
			v := *s
			return &v
		}
		return nil
	}
	// pick is a pick as the Session sends it: the next seq, on the point
	// and for the seat of the client's latest frame.
	pick := func(id, key string, confirm bool) wire.TablePick {
		seqs[id]++
		picks[id] = wire.TablePick{T: "table-pick", Code: code, Seq: seqs[id], Point: latest[id].Point.Ref, Key: key, Confirm: confirm, Seat: seatOf(id)}
		return picks[id]
	}
	choice := func(id string, i int) string {
		v := latest[id].View.You
		if v == nil || len(v.Choices) <= i {
			t.Fatalf("%s has no choice %d", id, i)
		}
		return v.Choices[i].Key
	}

	for s := range 3 {
		join(order[s], userOf(s), "P"+strconv.Itoa(s))
	}
	join("watcher", "watcher", "Watcher")

	// Seat 0 drafts, changes, confirms, withdraws and confirms again.
	for _, p := range []struct {
		i       int
		confirm bool
	}{{0, false}, {1, false}, {1, true}, {1, false}, {1, true}} {
		send("seat-0", false, pick("seat-0", choice("seat-0", p.i), p.confirm))
		settle()
	}
	// Seat 1 sends a key it was not offered (the Session never does), then
	// confirms.
	illegal := wire.TablePick{T: "table-pick", Code: code, Seq: seqs["seat-1"] + 1, Point: latest["seat-1"].Point.Ref, Key: "no-such-key"}
	send("seat-1", true, illegal)
	settle()
	send("seat-1", false, pick("seat-1", choice("seat-1", 0), true))
	settle()
	// Seat 2's confirmation resolves the point while seat 1, which has not
	// seen it yet, changes its pick: decision-closed.
	closing := latest["seat-1"]
	send("seat-2", false, pick("seat-2", choice("seat-2", 0), true))
	barrier()
	seqs["seat-1"]++
	send("seat-1", false, wire.TablePick{T: "table-pick", Code: code, Seq: seqs["seat-1"], Point: closing.Point.Ref,
		Key: closing.View.You.Choices[1].Key, Confirm: true, Seat: seatOf("seat-1")})
	settle()
	// Seat 0 repeats its last seq: the answer is its current frame,
	// acknowledging it.
	send("seat-0", true, picks["seat-0"])
	settle()
	// Seat 0, the creator, gives seat 1 to a bot; seat 1's tab now watches,
	// and its pick is refused.
	seqs["seat-0"]++
	bot := BotEasy
	send("seat-0", false, wire.TableSeat{T: "table-seat", Code: code, Seq: seqs["seat-0"], Seat: 1, Control: SeatBot, Bot: &bot})
	settle()
	if latest["seat-1"].You.Seat != nil {
		t.Fatal("seat 1's tab still holds a seat")
	}
	send("seat-1", true, wire.TablePick{T: "table-pick", Code: code, Seq: seqs["seat-1"] + 1, Point: latest["seat-1"].Point.Ref, Key: "no-such-key"})
	settle()

	got := renderClientFrames(t, order, users, logs, code)
	if os.Getenv("UPDATE_CLIENT_FRAMES") == "1" {
		if err := os.MkdirAll(filepath.Dir(clientFramesPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(clientFramesPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(clientFramesPath)
	if err != nil {
		t.Fatalf("%v (record it with UPDATE_CLIENT_FRAMES=1)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("the recorded client frames changed: rerun with UPDATE_CLIENT_FRAMES=1 and review the diff of %s", clientFramesPath)
	}
	// Every outcome the replay relies on is in the recording.
	for _, outcome := range []string{`"code":"illegal"`, `"code":"decision-closed"`, `"code":"forbidden"`, `"events":[{`, `"history":[`} {
		if !bytes.Contains(got, []byte(outcome)) {
			t.Fatalf("the recording has no %s", outcome)
		}
	}
}

// renderClientFrames writes the recording as JSON with one entry per line,
// the table's code replaced by clientFramesCode.
func renderClientFrames(t *testing.T, order []string, users map[string]string, logs map[string][]clientEntry, code string) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("{\"code\":\"" + clientFramesCode + "\",\"clients\":{\n")
	for i, id := range order {
		b.WriteString("\"" + id + "\":{\"user\":\"" + users[id] + "\",\"log\":[\n")
		for j, e := range logs[id] {
			switch {
			case e.Send != nil:
				b.WriteString("{\"send\":")
				b.Write(e.Send)
				if e.Raw {
					b.WriteString(",\"raw\":true")
				}
			default:
				b.WriteString("{\"recv\":")
				b.Write(e.Recv)
			}
			b.WriteString("}")
			if j < len(logs[id])-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("]}")
		if i < len(order)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}}\n")
	out := strings.ReplaceAll(b.String(), code, clientFramesCode)
	var check any
	if err := json.Unmarshal([]byte(out), &check); err != nil {
		t.Fatalf("the recording is not JSON: %v", err)
	}
	return []byte(out)
}
