package tableroom

import (
	"context"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
)

// TestBotSearchCountsUnderThePointsPlayer: a table's bot search holds its
// pool permit under the address of the player whose point it is, the
// lowest required human seat with a connection; failing that, the table
// creator's; failing that, the smallest attached participant's. Before,
// the smallest participant's address always paid, so one player's searches
// could hold another player's permits.
func TestBotSearchCountsUnderThePointsPlayer(t *testing.T) {
	pool := ai.NewPoolPerKey(2, 1)
	work := faults.NewWork(true)
	sp := humans(3, 405)
	sp.Creator = userOf(2)
	h := newTH(t, Config{Pool: pool, BotWork: work.Run}, sp)
	creator := tabletest.SeatClient(2)
	addr := func(u string) string { return "addr-" + u }
	key := func() (k string) {
		h.on(func(a *actor) { k = a.poolKey() })
		return k
	}
	// Hold seat 1's player's permit; seat 0 goes to a bot, whose search must
	// wait for it, although user-0, still watching, has the smallest address.
	ctx, cancel := context.WithTimeout(h.ctx, wait)
	defer cancel()
	if err := pool.AcquireFor(ctx, addr(userOf(1))); err != nil {
		t.Fatal(err)
	}
	h.want(creator, toBot(1, 0, BotEasy), outAccepted, "seat 0 to a bot")
	for pool.Waiting() != 1 {
		select {
		case <-work.Started():
			t.Fatal("the bot searched under another address than the point's player's")
		case <-ctx.Done():
			t.Fatal("the bot's search never waited for the point's player's address")
		case <-time.After(time.Millisecond):
		}
	}
	pool.ReleaseFor(addr(userOf(1)))
	select {
	case <-work.Started():
	case <-ctx.Done():
		t.Fatal("the bot's search did not start once the permit was free")
	}
	if k := key(); k != addr(userOf(1)) {
		t.Fatalf("the key with seat 1's player attached: %s", k)
	}
	h.leave(tabletest.SeatClient(1))
	if k := key(); k != addr(userOf(2)) {
		t.Fatalf("the key with seat 1's player gone: %s", k)
	}
	// No human seat left: the creator, watching, pays, not user-0.
	h.want(creator, toBot(2, 1, BotEasy), outAccepted, "seat 1 to a bot")
	h.want(creator, toBot(3, 2, BotEasy), outAccepted, "seat 2 to a bot")
	if k := key(); k != addr(userOf(2)) {
		t.Fatalf("the key with no human seat: %s", k)
	}
	h.leave(creator)
	if k := key(); k != addr(userOf(0)) {
		t.Fatalf("the key with the creator gone: %s", k)
	}
}
