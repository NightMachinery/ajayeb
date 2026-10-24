package lobbyroom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func duelFrame(k *conn) wire.State {
	return k.until("Duel state", func(m any) bool { _, ok := m.(wire.State); return ok }).(wire.State)
}

func TestSharedLobbyDuelStart(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	watch := f.conn(user(2), "w", "Watcher")
	code := f.create(a, 2, wire.LobbyOptions{FirstPlayer: ptr(1)})
	f.join(b, code)
	f.join(watch, code)
	if !a.lobby("both Duel seats filled", any2).CanStart {
		t.Fatal("two filled seats must be startable by their seated admin")
	}
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	sa, sb, sw := duelFrame(a), duelFrame(b), duelFrame(watch)
	if sa.Code != code || sa.Room.Status != wire.RoomStatusPlaying || !slices.Equal(sa.You.Seats, []int{0}) || !slices.Equal(sb.You.Seats, []int{1}) || len(sw.You.Seats) != 0 {
		t.Fatalf("handoff changed seats or code: %+v %+v %+v", sa.You, sb.You, sw.You)
	}
	saved, _, err := f.st.Load(code)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Options.FirstPlayer != engine.FirstPlayerSeat1 {
		t.Fatalf("firstPlayer was not mapped: %v", saved.Options)
	}
	if saved.Mode != store.ModeOnline || saved.Status != store.StatusPlaying || saved.Seats[0].UserID != user(0) || saved.Seats[1].UserID != user(1) {
		t.Fatalf("saved handoff: %+v", saved)
	}
	if got, _ := f.st.Game(code); got != room.Duel {
		t.Fatalf("not Duel: %s", got)
	}
	// Neither stale lobby edits nor retries of Start may mutate the new game.
	path := filepath.Join(f.dir, "rooms", code+".json")
	before, _ := os.ReadFile(path)
	_ = f.m.DispatchPlatform(context.Background(), a.c, wire.LobbyEdit{T: "lobby-edit", Code: code, Seq: 2, Op: "release"})
	if got := a.reject(2); got != "bad-request" {
		t.Fatalf("late lobby request: %s", got)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("late lobby request changed Duel")
	}
	// Two tabs of the same identity and a restart retain both assignments.
	f.m.Leave(a.c, code)
	f.restart()
	a2 := f.conn(user(0), "a2", "A")
	a3 := f.conn(user(0), "a3", "A")
	b2 := f.conn(user(1), "b2", "B")
	for _, k := range []*conn{a2, a3, b2} {
		f.join(k, code)
	}
	if !slices.Equal(duelFrame(a2).You.Seats, []int{0}) || !slices.Equal(duelFrame(a3).You.Seats, []int{0}) || !slices.Equal(duelFrame(b2).You.Seats, []int{1}) {
		t.Fatal("reconnect changed an assignment")
	}
}

func TestSharedLobbyDuelAssignedFileMatchesLegacy(t *testing.T) {
	f := newFixture(t, roomtest.NewFakeClock(time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)))
	f.close()
	f.duelSeed = func() uint64 { return 11 }
	f.start()
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	sa, sb := duelFrame(a), duelFrame(b)
	actual, _, err := f.st.Load(code)
	if err != nil {
		t.Fatal(err)
	}
	// Replay the pre-slice online create path with the exact same seed, clock,
	// identities and names, and stop after the friend's ordinary seat claim.
	oldA := f.conn(user(0), "old-a", "A")
	oldB := f.conn(user(1), "old-b", "B")
	oldCode, err := f.m.Create(context.Background(), oldA.c, wire.Create{T: "create", Mode: wire.ModeOnline, Seat: wire.SeatNumber(0)})
	if err != nil {
		t.Fatal(err)
	}
	f.join(oldB, oldCode)
	if err := f.m.Resync(context.Background(), oldA.c, oldCode); err != nil {
		t.Fatal(err)
	}
	oldSa := oldA.until("legacy full playing frame", func(m any) bool {
		v, ok := m.(wire.State)
		return ok && v.Room.Status == wire.RoomStatusPlaying && v.History != nil
	}).(wire.State)
	oldSb := duelFrame(oldB)
	expected, _, err := f.st.Load(oldCode)
	if err != nil {
		t.Fatal(err)
	}
	// A lobby join's claim epoch must survive the handoff. Ordinary online
	// creation starts that seat at epoch zero; all other file fields match.
	if expected.Seats[1].Epoch != 0 || actual.Seats[1].Epoch != 1 {
		t.Fatal("claim epoch provenance changed")
	}
	for i := range expected.Seats {
		expected.Seats[i].Epoch = actual.Seats[i].Epoch
	}
	expected.Code = actual.Code
	if !slices.Equal(actual.HintAdmins, []string{user(0)}) {
		t.Fatalf("frozen admins: %v", actual.HintAdmins)
	}
	actual.HintAdmins = nil
	ea, _ := json.Marshal(actual)
	eb, _ := json.Marshal(expected)
	if !bytes.Equal(ea, eb) {
		t.Fatalf("new assigned Duel differs from legacy online create:\nactual %s\nlegacy %s", ea, eb)
	}
	// Also compare raw room JSON against the existing direct constructor,
	// given the SAME assigned seats including their carried claim epochs.
	directCode, err := f.m.CreateFromSpec(room.Spec{Mode: store.ModeOnline, Seats: actual.Seats, Seed: 11, Options: actual.Options})
	if err != nil {
		t.Fatal(err)
	}
	direct, _, err := f.st.Load(directCode)
	if err != nil {
		t.Fatal(err)
	}
	direct.Code = actual.Code
	raw, _ := json.Marshal(direct)
	if !bytes.Equal(ea, raw) {
		t.Fatalf("assigned constructor changed room bytes:\nactual %s\nlegacy %s", ea, raw)
	}
	for i, pair := range [][2]wire.State{{sa, oldSa}, {sb, oldSb}} {
		if pair[0].You.LinkAdmin == nil || *pair[0].You.LinkAdmin || pair[1].You.LinkAdmin != nil {
			t.Fatal("link capability compatibility changed")
		}
		pair[0].You.LinkAdmin = nil // the only new frame field, deliberately absent in legacy frames
		pair[1].Code = pair[0].Code
		got, _ := json.Marshal(pair[0])
		want, _ := json.Marshal(pair[1])
		if !bytes.Equal(got, want) {
			t.Fatalf("seat %d initial frames differ:\nactual %s\nlegacy %s", i, got, want)
		}
	}
}

func TestSharedLobbyDuelBotAndSpectator(t *testing.T) {
	f := newFixture(t, nil)
	f.close()
	f.duelSeed = func() uint64 { return 11 }
	levels := make(chan store.BotLevel, 32)
	f.duelBot = func(level store.BotLevel, seed uint64) room.Bot {
		levels <- level
		return ai.NewGreedy(seed, ai.GreedyConfig{})
	}
	f.start()
	a := f.conn(user(0), "a", "A")
	watch := f.conn(user(9), "w", "Watcher")
	code := f.create(a, 2, wire.LobbyOptions{FirstPlayer: ptr(1)})
	ab := edit(code, 1, "add-bot")
	ab.Seat = ptr(1)
	ab.Bot = ptr("easy")
	f.send(a, ab)
	a.acked(1)
	// Two-seat bot controls replace a bot's level without releasing its slot.
	ab = edit(code, 2, "add-bot")
	ab.Seat = ptr(1)
	ab.Bot = ptr("hard")
	f.send(a, ab)
	a.acked(2)
	f.join(watch, code)
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 3})
	st := duelFrame(a)
	spectator := duelFrame(watch)
	if st.Room.Mode != wire.ModeOnline || st.Room.AllowUndo || st.Room.Seats[1].Kind != wire.SeatKindBot || st.Room.Seats[1].Bot == nil || *st.Room.Seats[1].Bot != wire.BotLevelHard {
		t.Fatalf("online bot setup: %+v", st.Room)
	}
	if len(spectator.You.Seats) != 0 || spectator.You.CanAct {
		t.Fatal("lobby spectator gained a seat")
	}
	select {
	case level := <-levels:
		if level != store.BotHard {
			t.Fatalf("wrong Duel level: %s", level)
		}
	case <-time.After(wait):
		t.Fatal("assigned online bot did not start")
	}
	a.until("bot move", func(m any) bool { v, ok := m.(wire.State); return ok && v.Version > 0 })
}

func TestSharedLobbyDuelFailedStartPreservesLobby(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	before, err := os.ReadFile(filepath.Join(f.dir, "rooms", code+".json"))
	if err != nil {
		t.Fatal(err)
	}
	f.replaceFails = true
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	if got := a.reject(1); got != outStorage {
		t.Fatalf("failed Start: %s", got)
	}
	after, _ := os.ReadFile(filepath.Join(f.dir, "rooms", code+".json"))
	if !bytes.Equal(before, after) {
		t.Fatal("failed Start changed durable lobby")
	}
	if game, _ := f.st.Game(code); game != Game {
		t.Fatal("failed Start swapped actors")
	}
	f.replaceFails = false
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	duelFrame(a)
	duelFrame(b)
}

func TestSharedLobbyDuelCarriesClaimEpoch(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	claim := f.secret.Claim(code, user(1), 1)
	dev := f.conn("other-device", "d", "Phone")
	if err := f.m.JoinClaim(f.ctx(), dev.c, code, &claim); err != nil {
		t.Fatal(err)
	}
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	if !slices.Equal(duelFrame(dev).You.Seats, []int{1}) {
		t.Fatal("active claim lost its seat during Start")
	}
	f.m.Leave(dev.c, code)
	f.restart()
	dev2 := f.conn("other-device", "d2", "Phone")
	if err := f.m.JoinClaim(f.ctx(), dev2.c, code, &claim); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(duelFrame(dev2).You.Seats, []int{1}) {
		t.Fatal("carried claim lost its seat after restart")
	}
	old := f.secret.Claim(code, user(1), 0)
	bad := f.conn("old-device", "bad", "Phone")
	if err := f.m.JoinClaim(f.ctx(), bad.c, code, &old); err != room.ErrBadClaim {
		t.Fatalf("stale claim revived: %v", err)
	}
}

func duelAck(k *conn, seq int) wire.State {
	return k.until("Duel ack", func(m any) bool { v, ok := m.(wire.State); return ok && v.Ack != nil && v.Ack.Seq == seq }).(wire.State)
}
func duelReject(k *conn, seq int) wire.Reject {
	return k.until("Duel refusal", func(m any) bool { v, ok := m.(wire.Reject); return ok && v.Seq == seq }).(wire.Reject)
}
func TestSharedLobbyDuelFrozenHintAdmins(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	watch := f.conn(user(2), "w", "Watcher")
	hints := room.DefaultHints(store.ModeOnline)
	hints.Counting = false
	code := f.create(a, 2, wire.LobbyOptions{Hints: &hints})
	f.join(b, code)
	f.join(watch, code)
	for i, id := range []string{user(1), user(2)} {
		f.send(a, wire.LobbyRole{T: "lobby-role", Code: code, Seq: i + 1, Op: "promote", UserID: &id})
		a.acked(i + 1)
	}
	f.send(a, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 3, Op: "step-down"})
	a.acked(3)
	f.send(b, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	sa, sb, sw := duelFrame(a), duelFrame(b), duelFrame(watch)
	if sa.You.Admin || !sb.You.Admin || !sw.You.Admin || len(sw.You.Seats) != 0 || sb.Room.Hints != hints {
		t.Fatal("Start did not freeze the effective seated and spectator admins and hint preferences")
	}
	for _, st := range []wire.State{sa, sb, sw} {
		if st.You.LinkAdmin == nil || *st.You.LinkAdmin {
			t.Fatal("shared hint admins gained device-link powers")
		}
	}
	hints.Prices = false
	if err := f.m.Hints(f.ctx(), watch.c, wire.NewHints(code, 10, hints)); err != nil {
		t.Fatal(err)
	}
	if got := duelAck(watch, 10); got.Room.Hints != hints {
		t.Fatal("spectator admin could not change hints")
	}
	if err := f.m.Hints(f.ctx(), a.c, wire.NewHints(code, 11, hints)); err != nil {
		t.Fatal(err)
	}
	if got := duelReject(a, 11); got.Code != wire.RejectCodeNotAdmin {
		t.Fatal("former creator retained hint authority")
	}
	if err := f.m.Relink(f.ctx(), b.c, wire.NewRelink(code, 0)); err != nil {
		t.Fatal(err)
	}
	if got := duelReject(b, 0); got.Code != wire.RejectCodeNotAdmin {
		t.Fatal("promoted hint admin reset the other seat's link")
	}
	// Persistence does not re-run lobby roles or regrant the creator's powers.
	f.restart()
	again := f.conn(user(2), "w2", "Watcher")
	f.join(again, code)
	st := duelFrame(again)
	if !st.You.Admin || st.Room.Hints != hints || len(st.You.Seats) != 0 {
		t.Fatal("frozen spectator hint admin did not survive restart")
	}
	saved, _, err := f.st.Load(code)
	if err != nil || !slices.Equal(saved.HintAdmins, []string{user(1), user(2)}) {
		t.Fatalf("frozen set after reload: %v %v", saved, err)
	}
}

func TestSharedLobbyDuelRematchReservesHumansAndResetsRoles(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	watch := f.conn(user(2), "w", "Watcher")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	f.join(watch, code)
	id := user(1)
	f.send(a, wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: &id})
	a.acked(1)
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2})
	duelFrame(a)
	duelFrame(b)
	duelFrame(watch)
	players := []*conn{a, b}
	for seq := 100; seq < 500; seq++ {
		snap, err := f.m.Snapshot(f.ctx(), code)
		if err != nil {
			t.Fatal(err)
		}
		if snap.Status == store.StatusFinished {
			break
		}
		keys := engine.LegalActions(&snap.State, nil)
		if len(keys) == 0 {
			t.Fatal("a playing game has no legal action")
		}
		k := players[int(snap.State.ToMove)]
		if err := f.m.Act(f.ctx(), k.c, wire.NewAct(code, seq, snap.Version, keys[0].Key(), nil)); err != nil {
			t.Fatal(err)
		}
		duelAck(k, seq)
	}
	snap, err := f.m.Snapshot(f.ctx(), code)
	if err != nil || snap.Status != store.StatusFinished {
		t.Fatal("Duel did not finish")
	}
	if _, err := f.m.Rematch(f.ctx(), watch.c, code); err != room.ErrForbidden {
		t.Fatalf("spectator rematch: %v", err)
	}
	// Simulate another writer taking a code after reservation: its bytes
	// must survive, and the parent must durably remember the next free code.
	competitor := store.New("TAKEXX", store.ModeOnline, engine.Options{}, 99, time.Now())
	if err := f.st.Save(competitor); err != nil {
		t.Fatal(err)
	}
	competitorPath := filepath.Join(f.dir, "rooms", "TAKEXX.json")
	competitorBytes, _ := os.ReadFile(competitorPath)
	f.mu.Lock()
	f.codes = []string{"TAKEXX", "NEXTXX"}
	f.mu.Unlock()
	next, err := f.m.Rematch(f.ctx(), b.c, code)
	if next != "NEXTXX" {
		t.Fatalf("collision retry chose %s", next)
	}
	competitorAfter, _ := os.ReadFile(competitorPath)
	if !bytes.Equal(competitorBytes, competitorAfter) {
		t.Fatal("rematch overwrote the competing room")
	}

	if err != nil {
		t.Fatal(err)
	}
	again, err := f.m.Rematch(f.ctx(), a.c, code)
	if err != nil || again != next {
		t.Fatalf("successor not idempotent: %s %s %v", next, again, err)
	}
	if game, _ := f.st.Game(next); game != Game {
		t.Fatalf("Duel rematch did not create a lobby: %s", game)
	}
	f.join(b, next)
	st := b.lobby("rematch lobby", any2)
	if st.SeatCount != 2 || st.CanStart || st.Seats[0].Kind != SlotReserved || st.Seats[1].Kind != SlotHuman {
		t.Fatalf("rematch reservations: %+v", st)
	}
	var r Record
	if err := f.lobbies.lookup(next).call(f.ctx(), func() { r = f.lobbies.lookup(next).rec.clone() }); err != nil {
		t.Fatal(err)
	}
	if r.Roles.Creator != user(1) || len(r.Roles.Admins) != 1 || r.Roles.Admins[0].UserID != user(1) {
		t.Fatalf("rematch roles: %+v", r.Roles)
	}
	f.restart()
	a2 := f.conn(user(0), "a2", "A")
	f.join(a2, code)
	third, err := f.m.Rematch(f.ctx(), a2.c, code)
	if err != nil || third != next {
		t.Fatalf("restart created another successor: %s %v", third, err)
	}
}

// finishDuel plays a two-human shared Duel game to its end, each turn's
// player taking their first legal action.
func finishDuel(t *testing.T, f *fixture, code string, players []*conn) {
	t.Helper()
	for seq := 100; seq < 600; seq++ {
		snap, err := f.m.Snapshot(f.ctx(), code)
		if err != nil {
			t.Fatal(err)
		}
		if snap.Status == store.StatusFinished {
			return
		}
		keys := engine.LegalActions(&snap.State, nil)
		k := players[int(snap.State.ToMove)]
		if err := f.m.Act(f.ctx(), k.c, wire.NewAct(code, seq, snap.Version, keys[0].Key(), nil)); err != nil {
			t.Fatal(err)
		}
		duelAck(k, seq)
	}
	t.Fatal("Duel did not finish")
}

// TestSharedLobbyDuelWithoutTableCreation: a production server runs
// without -haftgane-tables (Config.Creation off). Home's online Duel still
// creates its two-seat lobby, starts it and rematches it, and the frames
// say two seats is the most; three or more are refused, whether created,
// edited to or started from a lobby made while creation was on.
func TestSharedLobbyDuelWithoutTableCreation(t *testing.T) {
	f := newFixture(t, nil)
	early := f.conn(user(0), "e", "A")
	big, err := f.m.CreatePlatform(f.ctx(), early.c, wire.LobbyCreate{T: "lobby-create", SeatCount: 4, Options: wire.LobbyOptions{}})
	if err != nil {
		t.Fatal(err)
	}
	if st := early.lobby("the creator's frame", any2); st.MaxSeats != MaxSeats {
		t.Fatalf("with creation on, maxSeats %d", st.MaxSeats)
	}
	f.close()
	f.noCreation = true
	f.start()
	a := f.conn(user(0), "a", "A")
	b := f.conn(user(1), "b", "B")
	if _, err := f.m.CreatePlatform(f.ctx(), a.c, wire.LobbyCreate{T: "lobby-create", SeatCount: 4, Options: wire.LobbyOptions{}}); !errors.Is(err, room.ErrForbidden) {
		t.Fatalf("four-seat lobby-create without creation: %v", err)
	}
	code := f.create(a, 2, wire.LobbyOptions{})
	f.join(b, code)
	st := a.lobby("both seated", func(st wire.LobbyState) bool { return st.CanStart && slotOf2(st, user(1)) })
	if st.MaxSeats != MinSeats {
		t.Fatalf("without creation, maxSeats %d", st.MaxSeats)
	}
	seats := edit(code, 1, "seats")
	seats.SeatCount = ptr(4)
	f.send(a, seats)
	if got := a.reject(1); got != outForbidden {
		t.Fatalf("seats edit to 4 without creation: %s", got)
	}
	if r := f.record(code); r.SeatCount != 2 {
		t.Fatalf("refused edit changed the lobby: %d seats", r.SeatCount)
	}
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2})
	duelFrame(a)
	duelFrame(b)
	finishDuel(t, f, code, []*conn{a, b})
	next, err := f.m.Rematch(f.ctx(), b.c, code)
	if err != nil {
		t.Fatalf("Duel rematch without creation: %v", err)
	}
	if game, _ := f.st.Game(next); game != Game {
		t.Fatalf("Duel rematch made %q, not a lobby", game)
	}
	f.join(a, next)
	f.join(b, next)
	b.lobby("both seated in the successor", func(st wire.LobbyState) bool { return st.CanStart })
	f.send(b, wire.LobbyStart{T: "lobby-start", Code: next, Seq: 1})
	duelFrame(a)
	duelFrame(b)
	// The four-seat lobby made earlier loads, but nobody can start it.
	f.join(a, big)
	st = a.lobby("the earlier four-seat lobby", func(st wire.LobbyState) bool { return st.Code == big })
	if st.CanStart || st.MaxSeats != MinSeats || st.SeatCount != 4 {
		t.Fatalf("earlier four-seat lobby: canStart %v maxSeats %d seats %d", st.CanStart, st.MaxSeats, st.SeatCount)
	}
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: big, Seq: 10, FillBots: true})
	if got := a.reject(10); got != outForbidden {
		t.Fatalf("four-seat Start without creation: %s", got)
	}
	if game, _ := f.st.Game(big); game != Game {
		t.Fatalf("refused Start replaced the lobby with %q", game)
	}
}

func slotOf2(st wire.LobbyState, u string) bool {
	return slices.ContainsFunc(st.Seats, func(s wire.PlatformSeat) bool { return s.Kind == SlotHuman && s.UserID != nil && *s.UserID == u })
}
