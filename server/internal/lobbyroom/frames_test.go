package lobbyroom

import (
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room"
)

// isAdmin calls Now exactly once when it evaluates Roles.Holds for a
// temporary admin. Frame building has no other clock reads, so counting
// them measures authority checks without adding production hooks.
type holdsClock struct {
	room.Clock
	calls int
	at    time.Time
}

func (c *holdsClock) Now() time.Time { c.calls++; return c.at }

func TestPublishComputesRoleMetadataOnce(t *testing.T) {
	clk := &holdsClock{at: time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)}
	a := &actor{l: New(Config{Creation: true, Clock: clk})}
	a.rec = Record{Code: "ABCD", SeatCount: 3, Slots: []Slot{{Kind: SlotHuman, UserID: user(0)}, {Kind: SlotOpen}, {Kind: SlotOpen}}, Roles: roles.New(user(0))}
	a.rec.Roles.Away = []roles.Away{{UserID: user(0), Since: clk.at.Add(-6 * time.Minute)}}
	a.rec.Roles.Temporary = user(1)
	for i := 1; i <= 32; i++ {
		a.conns = append(a.conns, room.NewClient(user(i), user(i), "Player"))
		if i > 1 {
			a.rec.Roles.Deputies = append(a.rec.Roles.Deputies, roles.Admin{UserID: user(i), PromotedBy: ptr(user(1))})
		}
	}
	for _, except := range []bool{false, true} {
		clk.calls = 0
		viewers := len(a.conns)
		if except {
			a.broadcastExcept(a.conns[0])
			viewers--
		} else {
			a.publish(nil, nil)
		}
		// One Holds per temporary admin in meta(), plus two per viewer:
		// viewer permissions and canStart. No connections x admins term.
		want := 32 + 2*viewers
		if clk.calls != want {
			t.Fatalf("except=%v: %d Holds calls, want %d", except, clk.calls, want)
		}
		for i, c := range a.conns {
			frames := c.Out.Take()
			if except && i == 0 {
				if len(frames) != 0 {
					t.Fatal("excepted viewer received a frame")
				}
				continue
			}
			if len(frames) != 1 {
				t.Fatalf("viewer %d: %d frames", i, len(frames))
			}
		}
	}
}
