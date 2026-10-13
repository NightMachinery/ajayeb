package ws

import (
	"context"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// addrRooms records the address of each client that creates a room.
type addrRooms struct {
	*room.Manager
	addrs chan string
}

func (a addrRooms) Create(ctx context.Context, c *room.Client, req wire.Create) (string, error) {
	a.addrs <- c.Addr
	return a.Manager.Create(ctx, c, req)
}

// The rooms see each client's address as the per-address limits count it,
// which the bot pool shares its slots by.
func TestClientCarriesItsAddress(t *testing.T) {
	addrs := make(chan string, 1)
	h := newHarness(t, func(c *Config) { c.Rooms = addrRooms{Manager: c.Rooms.(*room.Manager), addrs: addrs} })
	h.client().create(wire.ModeHotseat)
	select {
	case got := <-addrs:
		if got != "127.0.0.1/32" {
			t.Fatalf("the client's address is %q, want 127.0.0.1/32", got)
		}
	case <-time.After(wsWait):
		t.Fatal("no room was created")
	}
}
