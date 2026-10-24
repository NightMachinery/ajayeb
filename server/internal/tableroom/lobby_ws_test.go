package tableroom_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// untilLobby reads until a lobby-state frame satisfies ok.
func (c *client) untilLobby(ctx context.Context, what string, ok func(wire.LobbyState) bool) wire.LobbyState {
	c.t.Helper()
	others := len(c.other)
	for {
		if c.next(ctx) == "lobby-state" && ok(c.lobby[len(c.lobby)-1]) {
			return c.lobby[len(c.lobby)-1]
		}
		if len(c.other) > others {
			c.t.Fatalf("waiting for %s: %s", what, c.other[len(c.other)-1])
		}
	}
}

// untilOther reads until a frame of type typ that is not a state arrives,
// and decodes it into v.
func (c *client) untilOther(ctx context.Context, typ string, v any) {
	c.t.Helper()
	for {
		others := len(c.other)
		c.next(ctx)
		if len(c.other) == others {
			continue
		}
		data := c.other[len(c.other)-1]
		var head struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal([]byte(data), &head); err != nil || head.T != typ {
			c.t.Fatalf("waiting for %s: %s", typ, data)
		}
		if err := json.Unmarshal([]byte(data), v); err != nil {
			c.t.Fatal(err)
		}
		return
	}
}

// stateNow is the latest table-state frame if it satisfies ok, or else the
// next one that does.
func (c *client) stateNow(ctx context.Context, what string, ok func(wire.TableState) bool) wire.TableState {
	c.t.Helper()
	if len(c.states) > 0 && ok(c.states[len(c.states)-1]) {
		return c.states[len(c.states)-1]
	}
	return c.untilState(ctx, what, ok)
}

func (c *client) nextSeq() int {
	c.seq++
	return c.seq
}

// acked reads until the frame acknowledging seq.
func (c *client) acked(ctx context.Context, what string, seq int) wire.TableState {
	c.t.Helper()
	return c.untilState(ctx, what, func(st wire.TableState) bool {
		return st.Ack != nil && st.Ack.ClientID == c.clientID && st.Ack.Seq == seq
	})
}

func admitted(user string) func(wire.LobbyState) bool {
	return func(st wire.LobbyState) bool { return st.You.UserID == user }
}

func humanIn(hs []wire.PlatformHuman, user string) (wire.PlatformHuman, bool) {
	i := slices.IndexFunc(hs, func(h wire.PlatformHuman) bool { return h.UserID == user })
	if i < 0 {
		return wire.PlatformHuman{}, false
	}
	return hs[i], true
}

// seatModel is the test's own model of who sits where and who is an
// admin, from the frame whose room log first has Log entries on.
type seatModel struct {
	Log   int
	Seat  map[string]int
	Admin map[string]bool
}

func (m seatModel) next(log int, change func(*seatModel)) seatModel {
	n := seatModel{Log: log, Seat: maps.Clone(m.Seat), Admin: maps.Clone(m.Admin)}
	change(&n)
	return n
}

// viewerAt is the viewer of a frame with a room log of length log, sent
// to a connection acting as user.
func viewerAt(models []seatModel, log int, user, clientID string) tableroom.FrameViewer {
	m := models[0]
	for _, x := range models {
		if x.Log <= log {
			m = x
		}
	}
	v := tableroom.FrameViewer{Seat: leakcheck.Spectator, Client: clientID, Admin: m.Admin[user]}
	if s, ok := m.Seat[user]; ok {
		v.Seat = s
	}
	return v
}

// TestWebSocketLobbyToTable: at 3 to 7 seats, over the real stack, with a
// spectator and two device links. A lobby made by lobby-create fills by
// joins and add-bot, a lobby role change reaches everyone, and Start makes
// every connection's next frame the table's under the same code, the
// sender's included. In the table an admin switches a seat to a bot, which
// revokes that seat's device link on its connection, seats the spectator
// there and demotes the user promoted in the lobby, and the last human
// seat resigns. Every switch and role change reaches every connection's
// log, the previous-seat hint only admins, and every frame passes the leak
// oracle against its viewer as the test's model says it was at that frame's
// log length, under both end policies: hidden at the end for odd seat
// counts, where the log download is refused and no final frame reveals,
// and revealed for even ones, where every final frame does. The finished
// table's rematch reaches every connection, a device linked to the
// creator's seat follows with its new claim into the reservation of the
// creator's user, and the rematch, with the original humans back in their
// reserved slots, starts again. The race build plays 4 and 7 seats only.
func TestWebSocketLobbyToTable(t *testing.T) {
	s := newStack(t, t.TempDir())
	seats := tableroom.SweepSeats()
	if tableroom.RaceEnabled {
		// Four seats and seven: an even and an odd count, so both end
		// policies, and the largest table (SweepSeats has two odd ends).
		seats = []int{tableroom.MinSeats + 1, tableroom.MaxSeats}
	}
	for _, n := range seats {
		lobbyToTable(wsContext(t, 5*time.Minute), t, s, n)
	}
}

func lobbyToTable(ctx context.Context, t *testing.T, s *stack, n int) {
	hide := n%2 == 1
	label := fmt.Sprintf("%d seats, hide=%t", n, hide)
	tokens := newTokens(t, n+2)
	seating, reveal, bot, last := "lobby", !hide, tableroom.BotEasy, n-1

	// The lobby: the creator, n-2 more humans by join, a bot in the last
	// slot, then the spectator, for whom no slot is left.
	creator := s.dial(ctx, tokens[0], "seat-0", "P0")
	creator.send(ctx, wire.LobbyCreate{T: "lobby-create", SeatCount: n, Options: wire.LobbyOptions{Seating: &seating, FinalReveal: &reveal}})
	var made wire.Created
	creator.untilOther(ctx, "created", &made)
	code := made.Code
	if st := creator.untilLobby(ctx, "the new lobby", admitted(creator.user)); st.Code != code || st.You.Seat == nil || *st.You.Seat != 0 {
		t.Fatalf("%s: the new lobby: %+v", label, st)
	}
	humans := []*client{creator}
	for i := 1; i < last; i++ {
		c := s.dial(ctx, tokens[i], fmt.Sprintf("seat-%d", i), fmt.Sprintf("P%d", i))
		c.send(ctx, wire.NewJoin(code))
		if st := c.untilLobby(ctx, "the join", admitted(c.user)); st.You.Seat == nil || *st.You.Seat != i {
			t.Fatalf("%s: P%d joined as %+v", label, i, st.You)
		}
		humans = append(humans, c)
	}
	seq := creator.nextSeq()
	creator.send(ctx, wire.LobbyEdit{T: "lobby-edit", Code: code, Seq: seq, Op: "add-bot", Seat: &last, Bot: &bot})
	creator.untilLobby(ctx, "the bot", func(st wire.LobbyState) bool { return st.Ack != nil && st.Ack.Seq == seq })
	watcher := s.dial(ctx, tokens[last], "spectator", "Watcher")
	watcher.send(ctx, wire.NewJoin(code))
	if st := watcher.untilLobby(ctx, "the watcher's join", admitted(watcher.user)); st.You.Seat != nil || st.CanStart {
		t.Fatalf("%s: the watcher joined as %+v, canStart %t", label, st.You, st.CanStart)
	}
	u1 := humans[1].user
	creator.send(ctx, wire.LobbyRole{T: "lobby-role", Code: code, Seq: creator.nextSeq(), Op: "promote", UserID: &u1})
	everyone := append(slices.Clone(humans), watcher)
	for _, c := range everyone {
		st := c.untilLobby(ctx, "the promotion", func(st wire.LobbyState) bool { h, _ := humanIn(st.Humans, u1); return h.Admin })
		if h, _ := humanIn(st.Humans, u1); h.PromotedBy == nil || *h.PromotedBy != creator.user || len(st.RoomLog) != 1 || st.RoomLog[0].Kind != "role" {
			t.Fatalf("%s: %s's lobby after the promotion: %+v, %+v", label, c.clientID, h, st.RoomLog)
		}
	}
	if !creator.lobby[len(creator.lobby)-1].CanStart {
		t.Fatalf("%s: the creator cannot start a full lobby", label)
	}

	// Start: every connection's next frame is the table's.
	lobbyFrames := len(creator.lobby)
	creator.send(ctx, wire.LobbyStart{T: "lobby-start", Code: code, Seq: creator.nextSeq()})
	var first wire.TableState
	for i, c := range everyone {
		fr := c.untilState(ctx, "the table's first frame", func(wire.TableState) bool { return true })
		want := &i
		if c == watcher {
			want = nil
		}
		if fr.Code != code || (fr.You.Seat == nil) != (want == nil) || want != nil && *fr.You.Seat != *want || fr.Seats[last].Kind != "bot" || fr.FinalReveal == hide {
			t.Fatalf("%s: %s's first table frame: %+v, seats %+v", label, c.clientID, fr.You, fr.Seats)
		}
		first = fr
	}
	if len(creator.lobby) != lobbyFrames {
		t.Fatal("the start's sender got a lobby frame after it")
	}
	for _, c := range everyone {
		for _, st := range c.lobby {
			if st.You.PreviousSeats != nil {
				t.Fatalf("%s: a previous-seat hint in a lobby frame", label)
			}
		}
	}

	// Device links: one for the creator's seat, one for seat 1.
	device := func(owner *client, seat int, token, id string) *client {
		owner.send(ctx, wire.PlatformLink{T: "link", Code: code, Seat: seat})
		var l wire.PlatformLinked
		owner.untilOther(ctx, "linked", &l)
		d := s.dial(ctx, token, id, "Device")
		d.send(ctx, wire.Join{T: "join", Code: code, Claim: &l.Claim})
		if fr := d.untilState(ctx, "the device's join", func(st wire.TableState) bool { return st.History != nil }); fr.You.UserID != owner.user || fr.You.Seat == nil || *fr.You.Seat != seat {
			t.Fatalf("%s: %s joined as %+v", label, id, fr.You)
		}
		return d
	}
	dev0 := device(creator, 0, tokens[n], "device-0")
	dev1 := device(humans[1], 1, tokens[n+1], "device-1")
	actsAs := map[*client]string{dev0: creator.user, dev1: u1}
	for _, c := range everyone {
		actsAs[c] = c.user
	}

	// The table's changes, each with the model from its frames on.
	models := []seatModel{{Log: len(first.RoomLog), Seat: map[string]int{}, Admin: map[string]bool{creator.user: true, u1: true}}}
	for i, c := range humans {
		models[0].Seat[c.user] = i
	}
	change := func(c *client, f func(seq int) any, model func(*seatModel)) wire.TableState {
		seq := c.nextSeq()
		c.send(ctx, f(seq))
		fr := c.acked(ctx, "a change's acknowledgement", seq)
		models = append(models, models[len(models)-1].next(len(fr.RoomLog), model))
		return fr
	}
	fr := change(creator, func(seq int) any {
		return wire.TableSeat{T: "table-seat", Code: code, Seq: seq, Seat: 1, Control: "bot", Bot: &bot}
	}, func(m *seatModel) { delete(m.Seat, u1) })
	var e wire.Error
	dev1.untilOther(ctx, "error", &e)
	if e.Code != wire.ErrorCodeBadClaim {
		t.Fatalf("%s: the switch's revocation: %+v", label, e)
	}
	if ps := fr.You.PreviousSeats; ps == nil || !slices.Contains(*ps, wire.PreviousSeat{UserID: u1, Seat: 1}) {
		t.Fatalf("%s: the admin's hint after the switch: %v", label, ps)
	}
	change(creator, func(seq int) any {
		return wire.TableSeat{T: "table-seat", Code: code, Seq: seq, Seat: 1, Control: "human", UserID: &watcher.user}
	}, func(m *seatModel) { m.Seat[watcher.user] = 1 })
	change(creator, func(seq int) any {
		return wire.TableRole{T: "table-role", Code: code, Seq: seq, Op: "demote", UserID: &u1}
	}, func(m *seatModel) { delete(m.Admin, u1) })
	resigner := watcher
	if n > 3 {
		resigner = humans[n-2]
	}
	change(resigner, func(seq int) any {
		return wire.TableResign{T: "table-resign", Code: code, Seq: seq, Confirm: true}
	}, func(m *seatModel) { delete(m.Seat, resigner.user) })
	final := models[len(models)-1]
	live := append(slices.Clone(everyone), dev0)
	var log []wire.PlatformLogEntry
	for _, c := range live {
		st := c.stateNow(ctx, "the last change", func(st wire.TableState) bool { return len(st.RoomLog) == final.Log })
		if log == nil {
			log = st.RoomLog
		}
		if !slices.EqualFunc(st.RoomLog, log, func(x, y wire.PlatformLogEntry) bool {
			a, _ := json.Marshal(x)
			b, _ := json.Marshal(y)
			return string(a) == string(b)
		}) {
			t.Fatalf("%s: %s's room log differs: %+v", label, c.clientID, st.RoomLog)
		}
	}
	var kinds []string
	for _, e := range log[models[0].Log:] {
		kinds = append(kinds, e.Kind)
	}
	if !slices.Equal(kinds, []string{"seat", "seat", "role", "resign"}) {
		t.Fatalf("%s: the table's log entries: %v", label, kinds)
	}

	// Play to the end: the humans still seated pick, the bots play theirs.
	w := &wsTable{s: s, code: code, clients: map[tabletest.ClientID]*client{}, acked: map[tabletest.ClientID]uint64{}}
	for _, c := range everyone {
		if seat, ok := final.Seat[c.user]; ok {
			w.clients[tabletest.SeatClient(seat)] = c
		}
	}
	w.clients[tabletest.SpectatorClient] = humans[1]
	playOut(ctx, t, w)

	// The end, under its policy.
	_, status, _ := s.download(ctx, code)
	if want := map[bool]int{false: http.StatusOK, true: http.StatusForbidden}[hide]; status != want {
		t.Fatalf("%s: the log download answered %d, want %d", label, status, want)
	}
	for _, c := range live {
		c.send(ctx, wire.NewResync(code))
		st := c.untilState(ctx, "the final resync", func(st wire.TableState) bool { return st.History != nil })
		if st.Status != "finished" || st.FinalReveal == hide || (st.View.Reveal == nil) != hide || !hide && len(st.View.Reveal.Tucked) != n {
			t.Fatalf("%s: %s's final frame: status %s, finalReveal %t, reveal %+v", label, c.clientID, st.Status, st.FinalReveal, st.View.Reveal)
		}
	}
	data, _, err := s.st.PeekGame(code, tableroom.Game)
	if err != nil {
		t.Fatal(err)
	}
	checker := leakcheck.New(tableroom.RecordTruth(t, data), tableroom.FrameConfig)
	hinted := 0
	for _, c := range append(live, dev1) {
		var frames []wire.TableState
		var raws [][]byte
		var viewers []tableroom.FrameViewer
		for i, st := range c.states {
			if st.Code != code {
				continue
			}
			frames, raws = append(frames, st), append(raws, c.raw[i])
			viewers = append(viewers, viewerAt(models, len(st.RoomLog), actsAs[c], c.clientID))
			if ps := st.You.PreviousSeats; ps != nil && len(*ps) > 0 {
				hinted++
			}
		}
		where := label + " " + c.clientID
		if leaks := tableroom.ViewerLeaks(t, checker, where, viewers, frames, raws, nil); len(leaks) != 0 {
			t.Fatalf("%s: %v", where, leaks[:min(5, len(leaks))])
		}
	}
	if hinted == 0 {
		t.Fatalf("%s: no frame carried a previous-seat hint", label)
	}

	// The rematch: asked by an original human, now a spectator; every
	// connection gets its code, the creator's device with its new claim.
	humans[1].send(ctx, wire.NewRematch(code))
	created := map[*client]wire.Created{}
	for _, c := range live {
		var cr wire.Created
		c.untilOther(ctx, "created", &cr)
		if cr.Code == code || (cr.Claim != nil) != (c == dev0) {
			t.Fatalf("%s: %s's created frame: %+v", label, c.clientID, cr)
		}
		created[c] = cr
	}
	next := created[humans[1]].Code
	creator.send(ctx, wire.NewRematch(code))
	var again wire.Created
	if creator.untilOther(ctx, "created", &again); again.Code != next {
		t.Fatalf("%s: a second rematch made %s, not %s", label, again.Code, next)
	}
	join := func(c *client, claim *string) wire.LobbyState {
		c.send(ctx, wire.Join{T: "join", Code: next, Claim: claim})
		return c.untilLobby(ctx, "the rematch's join", func(st wire.LobbyState) bool { return st.Code == next && st.You.UserID == actsAs[c] })
	}
	st := join(humans[1], nil)
	if h, _ := humanIn(st.Humans, u1); !h.Creator || !h.Admin || st.You.Seat == nil || *st.You.Seat != 1 || st.Seats[last].Kind != "bot" || st.Seats[0].Kind != "reserved" {
		t.Fatalf("%s: the rematch lobby: you %+v, seats %+v", label, st.You, st.Seats)
	}
	if st := join(dev0, created[dev0].Claim); st.You.Seat == nil || *st.You.Seat != 0 {
		t.Fatalf("%s: the creator's device in the rematch: %+v", label, st.You)
	}
	for i := 2; i < last; i++ {
		join(humans[i], nil)
	}
	if st := join(watcher, nil); st.You.Seat != nil {
		t.Fatalf("%s: the watcher took a reserved slot: %+v", label, st.You)
	}
	humans[1].send(ctx, wire.LobbyStart{T: "lobby-start", Code: next, Seq: humans[1].nextSeq()})
	for _, c := range append([]*client{dev0, watcher}, humans[1:last]...) {
		fr := c.untilState(ctx, "the rematch's first frame", func(st wire.TableState) bool { return st.Code == next })
		if seat, ok := models[0].Seat[actsAs[c]]; ok != (fr.You.Seat != nil) || ok && *fr.You.Seat != seat {
			t.Fatalf("%s: %s in the rematch's table: %+v", label, c.clientID, fr.You)
		}
	}
	for _, c := range append(live, dev1) {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}
}

// playOut plays a table to the end: whenever a required seat with a
// client has not confirmed, it confirms its first offered key; otherwise
// it waits for a bot's move.
func playOut(ctx context.Context, t *testing.T, w *wsTable) {
	t.Helper()
	watch := w.clients[tabletest.SpectatorClient]
	for step := 0; ; step++ {
		if step > 400 {
			t.Fatal("the game did not end")
		}
		sn, _ := w.Snapshot(ctx, tabletest.SpectatorClient)
		if sn.Over {
			return
		}
		moved := false
		for _, seat := range sn.Required {
			id := tabletest.SeatClient(seat)
			c, ok := w.clients[id]
			if !ok || slices.Contains(sn.Confirmed, seat) {
				continue
			}
			mine, _ := w.Snapshot(ctx, id)
			if mine.Ref != sn.Ref || len(mine.Legal) == 0 {
				moved = true
				break
			}
			o, err := w.Submit(ctx, id, tabletest.Submission{Seq: uint64(c.nextSeq()), Ref: mine.Ref, Key: mine.Legal[0], Confirm: true})
			if err != nil || o != tabletest.Accepted && o != tabletest.Resolved && o != tabletest.Closed {
				t.Fatalf("point %s, seat %d: %s (%v)", sn.Point, seat, o, err)
			}
			moved = true
			break
		}
		if !moved {
			confirmed := len(sn.Confirmed)
			watch.untilState(ctx, "a bot's move", func(st wire.TableState) bool {
				return st.Point == nil || tableroom.RefString(st.Point.Ref) != sn.Ref || len(st.Point.Confirmed) > confirmed
			})
		}
	}
}
