package lobbyroom

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestLargestLobbyFitsTheStore builds the largest lobby record the bounds
// allow (every list full, every id and name at the length that encodes to
// the most bytes, every seq the largest, each client id one the wire
// accepts and encoding/json escapes to six bytes a character) and checks
// that a restart accepts it and that it fits under the store's read cap.
// Before, its seq acknowledgements alone took about three times the cap,
// so an admin with puppet users could grow a lobby until the store could
// no longer load it (M16.haftgane-security-review).
func TestLargestLobbyFitsTheStore(t *testing.T) {
	f := newFixture(t, nil)
	k := f.conn(user(0), "tab", "Creator")
	code := f.create(k, MaxSeats, wire.LobbyOptions{Seating: ptr("lobby")})
	r := f.record(code)
	id := func(kind, i int) string { return fmt.Sprintf("%02x%030x", kind, i) }
	name := strings.Repeat("<", maxNameLen-1) + "x"
	r.Revision = math.MaxInt64
	for i := range r.Slots {
		r.Slots[i] = Slot{Kind: SlotHuman, UserID: id(1, i), Name: name, Taken: math.MaxInt64, Epoch: math.MaxInt64}
	}
	since := time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.UTC)
	creator := id(2, 0)
	r.Roles = roles.New(creator)
	for i := 1; len(r.Roles.Admins) < roles.MaxAdmins; i++ {
		by := creator
		r.Roles.Admins = append(r.Roles.Admins, roles.Admin{UserID: id(2, i), PromotedBy: &by})
	}
	r.Roles.Temporary = id(3, 0)
	for _, ad := range r.Roles.Admins {
		r.Roles.Away = append(r.Roles.Away, roles.Away{UserID: ad.UserID, Since: since})
	}
	r.Roles.Away = append(r.Roles.Away, roles.Away{UserID: r.Roles.Temporary, Since: since})
	r.Unseated = nil
	for i := range maxUnseated {
		r.Unseated = append(r.Unseated, id(5, i))
	}
	r.Retired = nil
	for i := range maxRetired {
		r.Retired = append(r.Retired, id(4, i))
	}
	r.RoomLog = nil
	for range maxRoomLog {
		u, s, ctl, role := id(6, 0), MaxSeats-1, "human", "temporary"
		r.RoomLog = append(r.RoomLog, wire.PlatformLogEntry{Kind: "resign", At: math.MaxInt64, By: id(6, 1), UserID: &u, Seat: &s, Control: &ctl, Role: &role})
	}
	r.Parent = "ZZZZZZ"
	var users []string
	for _, s := range r.Slots {
		users = append(users, s.UserID)
	}
	for _, ad := range r.Roles.Admins {
		users = append(users, ad.UserID)
	}
	users = append(append(users, r.Roles.Temporary), r.Retired...)
	if len(users) != maxSeqUsers {
		t.Fatalf("%d users, want maxSeqUsers %d", len(users), maxSeqUsers)
	}
	r.Seqs = nil
	n := 0
	for _, u := range users {
		for range defaultSeqClients {
			client := strings.Repeat("<", 120) + fmt.Sprintf("%08d", n)
			r.Seqs = append(r.Seqs, SeqRecord{User: u, Client: tableroom.ClientKey(client), Seq: math.MaxInt64})
			n++
		}
	}
	if err := validate(&r, defaultSeqClients); err != nil {
		t.Fatalf("the largest lobby is not one a restart accepts: %v", err)
	}
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("the largest lobby record: %d bytes, %.2f of the store's cap of %d", len(data), float64(len(data))/store.MaxFileBytes, store.MaxFileBytes)
	if len(data) > store.MaxFileBytes*3/4 {
		t.Errorf("the largest lobby record takes %d bytes, more than three quarters of the store's cap of %d", len(data), store.MaxFileBytes)
	}
}
