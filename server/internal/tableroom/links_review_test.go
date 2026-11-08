package tableroom

import (
	"errors"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestSpectatorLinkSurvivesTableSeating(t *testing.T) {
	for _, self := range []bool{true, false} {
		name := "admin-seats"
		if self {
			name = "phone-rejoins"
		}
		t.Run(name, func(t *testing.T) {
			h := newTH(t, Config{BotWork: blockBots}, humans(3, 111))
			admin, player := tabletest.SeatClient(0), tabletest.SeatClient(1)
			_, seatClaim := h.userLink(player, userOf(1), false)
			h.want(admin, toBot(1, 1, BotEasy), outAccepted, "release seat")
			_, old := h.userLink(player, userOf(1), false)
			_, current := h.userLink(player, userOf(1), true)
			if err := h.joinClaim("phone", "phone", current); err != nil {
				t.Fatal(err)
			}
			by, seq := admin, 2
			if self {
				by, seq = "phone", 1
			}
			h.want(by, toHuman(seq, 1, userOf(1)), outAccepted, "seat linked user")
			st := h.latest("phone")
			if h.revoked("phone") || st.You.Seat == nil || *st.You.Seat != 1 || st.View.You == nil {
				t.Fatal("linked phone lost its seat")
			}
			_, after := h.userLink("phone", userOf(1), false)
			if after != current {
				t.Fatal("seating changed the current user claim")
			}
			for _, dead := range []string{seatClaim, old} {
				if err := h.joinClaim("revoked", "revoked", dead); !errors.Is(err, room.ErrBadClaim) {
					t.Fatal("seating revived old claim")
				}
			}
			sn := h.snap("phone")
			h.expect(h.send("phone", uint64(seq+1), sn.Ref, sn.Legal[0], false), tabletest.Accepted, "phone acts in seat")
		})
	}
}

func TestTemporaryAdminCannotLinkRealAdmins(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC))
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 112))
	creator, temp, real, watch := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SpectatorClient
	h.want(creator, roleOp(1, "promote", userOf(2)), outAccepted, "promote real admin")
	h.on(func(a *actor) { a.st.rec.Roles.History = []string{userOf(1)} })
	h.leave(creator, real)
	clk.Advance(5 * time.Minute)
	h.until(temp, "temporary admin", func(st wire.TableState) bool { return human(t, st, userOf(1)).Temporary })
	for _, target := range []string{userOf(0), userOf(2)} {
		for _, reset := range []bool{false, true} {
			if out, claim := h.userLink(temp, target, reset); out != outNotAdmin || claim != "" {
				t.Fatal("temporary admin read real admin link")
			}
		}
		hint := human(t, h.latest(temp), target)
		if hint.Linkable != nil {
			t.Fatal("forbidden target still offered to temporary admin")
		}
		// The broadcast metadata is shared: filtering one viewer must not change others.
		if hint := human(t, h.latest(watch), target); hint.Linkable == nil || !*hint.Linkable {
			t.Fatal("temporary viewer mutated shared metadata")
		}
	}
	if out, _ := h.userLink(temp, userOf(1), false); out != outAccepted {
		t.Fatal("temporary admin lost own link")
	}
	h.want(temp, roleOp(1, "promote", "watcher"), outAccepted, "temporary admin promotes watcher")
	if out, _ := h.userLink(watch, userOf(1), false); out != outAccepted {
		t.Fatal("temporary deputy cannot link another temporary human")
	}
	// When a real admin returns, ordinary real-admin recovery works again.
	h.joinAs(creator, userOf(0))
	if out, _ := h.userLink(temp, userOf(2), false); out != outNotAdmin {
		t.Fatal("expired temporary authority still links")
	}
	if out, _ := h.userLink(creator, userOf(2), false); out != outAccepted {
		t.Fatal("real admin cannot recover real admin")
	}
}

func TestConsecutiveTableSelfResetsKeepDisclosure(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 113))
	admin, player := tabletest.SeatClient(0), tabletest.SeatClient(1)
	if out, _ := h.userLink(admin, userOf(1), false); out != outAccepted {
		t.Fatal(out)
	}
	h.on(func(a *actor) {
		copy := a.st.rec.RoomLog[len(a.st.rec.RoomLog)-1]
		a.st.rec.RoomLog = make([]wire.PlatformLogEntry, maxRoomLog)
		for i := range a.st.rec.RoomLog {
			a.st.rec.RoomLog[i] = copy
		}
	})
	_, first := h.userLink(player, userOf(1), true)
	for range 3 {
		if out, next := h.userLink(player, userOf(1), true); out != outAccepted || next == first {
			t.Fatal("reset did not advance epoch")
		}
	}
	r := h.record()
	copies := 0
	for _, entry := range r.RoomLog {
		if entry.Kind == "link-copy" {
			copies++
		}
	}
	if len(r.RoomLog) != maxRoomLog || copies != maxRoomLog-1 {
		t.Fatal("self-resets evicted disclosure entries")
	}
	if out, _ := h.userLink(admin, userOf(1), false); out != outAccepted {
		t.Fatal(out)
	}
	if out, _ := h.userLink(admin, userOf(1), false); out != outAccepted {
		t.Fatal(out)
	}
	r = h.record()
	for _, entry := range r.RoomLog[len(r.RoomLog)-2:] {
		if entry.Kind != "link-copy" {
			t.Fatal("copies were coalesced")
		}
	}
}
