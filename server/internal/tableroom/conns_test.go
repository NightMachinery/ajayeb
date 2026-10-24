package tableroom

import (
	"errors"
	"fmt"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

// TestNinthConnectionIsRefused: a user holds at most 8 connections to a
// table, seated or watching; the ninth is refused with room-full, and
// another user still gets in. Before, a seated user could hold 32.
func TestNinthConnectionIsRefused(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 404))
	for _, who := range []struct {
		user  string
		first tabletest.ClientID
	}{{userOf(1), tabletest.SeatClient(1)}, {"watcher", tabletest.SpectatorClient}} {
		for i := 1; i < DefaultMaxUserClients; i++ {
			h.joinAs(tabletest.ClientID(fmt.Sprintf("%s-tab-%d", who.user, i)), who.user)
		}
		err := h.p.join(h.ctx, tabletest.ClientID(who.user+"-tab-9"), who.user, who.user)
		if !errors.Is(err, room.ErrRoomFull) {
			t.Fatalf("%s's ninth connection: %v", who.user, err)
		}
	}
	h.joinAs("other", "another-watcher")
}
