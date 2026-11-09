package tableroom

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func cycleEntry(kind, user string, seat, at int) wire.PlatformLogEntry {
	control := "human"
	// Table resignations identify their user by By alone.
	if kind == "resign" {
		return wire.PlatformLogEntry{Kind: kind, By: user, Seat: &seat, Control: &control, At: at}
	}
	return wire.PlatformLogEntry{Kind: kind, By: user, UserID: &user, Seat: &seat, Control: &control, At: at}
}

func cyclePrefix() []wire.PlatformLogEntry {
	// Put both disclosures at the oldest end of an almost-full log.
	copy := cycleEntry("link-copy", "admin", 0, 0)
	target := "player"
	copy.UserID = &target
	seat := cycleEntry("seat", "admin", 1, 0)
	seat.UserID = &target
	log := []wire.PlatformLogEntry{copy, seat}
	for len(log) < maxRoomLog-2 {
		log = append(log, wire.PlatformLogEntry{Kind: "role", By: "admin"})
	}
	return log
}

func TestOwnSeatCyclesKeepDisclosures(t *testing.T) {
	for _, first := range []string{"resign", "seat"} {
		t.Run(first, func(t *testing.T) {
			r := Record{RoomLog: cyclePrefix()}
			prefix := append([]wire.PlatformLogEntry{}, r.RoomLog...)
			second := "seat"
			if first == "seat" {
				second = "resign"
			}
			for at := range 2048 {
				appendLog(&r, cycleEntry(first, "player", 1, at*2), cycleEntry(second, "player", 1, at*2+1))
				if len(r.RoomLog) != maxRoomLog {
					t.Fatalf("cycle %d: %d entries", at, len(r.RoomLog))
				}
			}
			if !reflect.DeepEqual(r.RoomLog[:len(prefix)], prefix) {
				t.Fatal("cycles evicted earlier disclosures")
			}
			want := []wire.PlatformLogEntry{cycleEntry(first, "player", 1, 4094), cycleEntry(second, "player", 1, 4095)}
			if !reflect.DeepEqual(r.RoomLog[len(prefix):], want) {
				t.Fatal("did not keep the latest leave and rejoin with their batch counts")
			}
		})
	}
}

func TestOwnSeatCycleBoundaries(t *testing.T) {
	leave, rejoin := cycleEntry("resign", "player", 1, 1), cycleEntry("seat", "player", 1, 2)
	for _, kind := range []string{"resign", "seat", "role", "link", "link-copy"} {
		t.Run("intervening-"+kind, func(t *testing.T) {
			by := "admin"
			if kind == "link" {
				by = "player"
			}
			barrier := cycleEntry(kind, by, 1, 3)
			target := "player"
			barrier.UserID = &target
			r := Record{RoomLog: []wire.PlatformLogEntry{leave, rejoin}}
			appendLog(&r, barrier, cycleEntry("resign", "player", 1, 4), cycleEntry("seat", "player", 1, 5), cycleEntry("resign", "player", 1, 6))
			want := []wire.PlatformLogEntry{leave, rejoin, barrier, cycleEntry("seat", "player", 1, 5), cycleEntry("resign", "player", 1, 6)}
			if !reflect.DeepEqual(r.RoomLog, want) {
				t.Fatalf("coalesced across %s: %+v", kind, r.RoomLog)
			}
		})
	}
	for _, change := range []string{"seat", "user"} {
		t.Run("different-"+change, func(t *testing.T) {
			user, seat := "player", 2
			if change == "user" {
				user, seat = "other", 1
			}
			r := Record{RoomLog: []wire.PlatformLogEntry{leave, rejoin}}
			appendLog(&r, cycleEntry("resign", user, seat, 3), cycleEntry("seat", user, seat, 4), cycleEntry("resign", user, seat, 5))
			want := []wire.PlatformLogEntry{leave, rejoin, cycleEntry("seat", user, seat, 4), cycleEntry("resign", user, seat, 5)}
			if !reflect.DeepEqual(r.RoomLog, want) {
				t.Fatal("coalesced different users or seats")
			}
		})
	}
}

func TestTableSeatCycleFramesAndRestore(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 479))
	player := tabletest.SeatClient(1)
	h.on(func(a *actor) { a.st.rec.RoomLog = cyclePrefix() })
	for cycle := range 4 {
		seq := cycle*2 + 1
		h.want(player, wire.TableResign{T: "table-resign", Seq: seq, Confirm: true}, outAccepted, "self-leave")
		left := h.latest(player)
		if left.You.Seat != nil || left.You.RejoinSeat == nil || left.View.You != nil {
			t.Fatal("leave frame retained private seat view")
		}
		h.want(player, toHuman(seq+1, 1, userOf(1)), outAccepted, "self-rejoin")
		joined := h.latest(player)
		if joined.You.Seat == nil || *joined.You.Seat != 1 || joined.View.You == nil || joined.History == nil {
			t.Fatal("rejoin did not restore own view and history")
		}
		log := h.sameLogEverywhere(maxRoomLog)
		if !reflect.DeepEqual(log[:maxRoomLog-2], cyclePrefix()) {
			t.Fatal("cycle evicted a disclosure")
		}
	}
	before := h.record()
	h.f.restart()
	h.rejoin()
	if !reflect.DeepEqual(h.latest(player).RoomLog, before.RoomLog) {
		t.Fatal("restart changed coalesced log")
	}
}
