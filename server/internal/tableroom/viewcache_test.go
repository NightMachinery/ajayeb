package tableroom

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestEmptyTableDropsItsViews: when its last connection leaves, a table
// drops its cached views and histories. For a finished seven-seat table
// they take about 560 KB of heap against a record of about 15 KB, and a
// table with no connection stays in memory for up to 30 minutes, so before,
// someone who joined and left one table after another could hold that much
// for each of the 1024 rooms in memory (M16.haftgane-security-review). The
// next join rebuilds its role's view, and its frame is the same as before.
func TestEmptyTableDropsItsViews(t *testing.T) {
	h := newTH(t, Config{Pool: ai.NewPool(1)}, soloSpec(7, 47, DefaultBotLevel))
	me, spec := tabletest.SeatClient(0), tabletest.SpectatorClient
	for seq := 1; ; seq++ {
		fr := h.until(me, "my turn or the end", func(st wire.TableState) bool {
			return st.Status == "finished" || (st.Point != nil && slices.Contains(st.Point.Required, 0) && st.Pending == nil)
		})
		if fr.Status == "finished" {
			break
		}
		if seq > 60 {
			t.Fatal("the game did not end")
		}
		h.pickSelf(seq, fr.View.You.Choices[0].Key)
	}
	frames := func() map[tabletest.ClientID]string {
		out := map[tabletest.ClientID]string{}
		h.on(func(a *actor) {
			for _, id := range []tabletest.ClientID{me, spec} {
				a.inbound(h.p.clients[id], wire.NewResync(h.p.code))
			}
		})
		for _, id := range []tabletest.ClientID{me, spec} {
			data, err := json.Marshal(h.latest(id))
			if err != nil {
				t.Fatal(err)
			}
			out[id] = string(data)
		}
		return out
	}
	cached := func() (n int) {
		h.on(func(a *actor) { n = len(a.engineViews) })
		return n
	}
	before := frames()
	if n := cached(); n != 2 {
		t.Fatalf("%d roles cached while the seat and the spectator watch, want 2", n)
	}
	h.leave(me, spec)
	if n := cached(); n != 0 {
		t.Fatalf("the table still caches %d roles' views with no connection", n)
	}
	if err := h.p.join(h.ctx, me, userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, spec, "watcher", "Watcher"); err != nil {
		t.Fatal(err)
	}
	after := frames()
	for _, id := range []tabletest.ClientID{me, spec} {
		if after[id] != before[id] {
			t.Errorf("%s's frame changed across the drop:\n%s\n%s", id, before[id], after[id])
		}
	}
}
