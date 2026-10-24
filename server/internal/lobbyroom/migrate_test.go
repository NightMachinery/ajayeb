package lobbyroom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestOldLobbyClientIdsAreRekeyedOnLoad: a format-2 lobby kept a long
// client id as sent; on load it is re-keyed by tableroom.ClientKey, so the
// client's accepted seq is still remembered and an older one only
// acknowledged. Before, the raw id was never found again, and an older
// seq removed a bot (M16.haftgane-security-review, from its re-check).
func TestOldLobbyClientIdsAreRekeyedOnLoad(t *testing.T) {
	f := newFixture(t, nil)
	id := strings.Repeat("a", 80)
	k := f.conn(user(0), id, "Creator")
	code := f.create(k, 3, wire.LobbyOptions{})
	ab := edit(code, 8, "add-bot")
	ab.Seat, ab.Bot = ptr(2), ptr(tableroom.BotHard)
	f.send(k, ab)
	k.acked(8)
	r := f.record(code)
	key, n := tableroom.ClientKey(id), 0
	for i := range r.Seqs {
		if r.Seqs[i].Client == key {
			r.Seqs[i].Client = id
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the record held the client %d times", n)
	}
	r.Format = 2
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.st.Dir(), code+".json")
	f.close()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	f.start()
	k = f.conn(user(0), id, "Creator")
	f.join(k, code)
	k.lobby("the creator's frame", any2)
	rm := edit(code, 7, "remove-bot")
	rm.Seat = ptr(2)
	f.send(k, rm)
	if st := k.acked(7); kinds(st)[2] != SlotBot {
		t.Fatalf("an older seq was applied after the format-2 load: %v", kinds(st))
	}
}
