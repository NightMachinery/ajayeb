package room_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// A room whose file has grown past the store's read cap (planted, or
// corrupted) is a storage failure: refuse without loading it or dropping the
// client's saved room, and leave the file intact for repair.
func TestOversizedRoomFileIsRetryable(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	h.evict(code, a)

	path := filepath.Join(h.st.Dir(), code+".json")
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	padded := append(bytes.Repeat([]byte(" "), store.MaxFileBytes), valid...)
	if err := os.WriteFile(path, padded, 0o600); err != nil {
		t.Fatal(err)
	}

	again := h.client("user-a", "tab-a2", "Ann")
	if e, err := again.joinErr(code); e.Code != wire.ErrorCodeBusy || !errors.Is(err, room.ErrStorage) {
		t.Fatalf("joining the oversized room: %v, %v", e, err)
	}
	if h.m.Loaded(code) {
		t.Fatal("the oversized room was loaded")
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, padded) {
		t.Fatalf("the oversized file changed: %v", err)
	}
	// Once the file is repaired, the same saved room can be opened again.
	if err := os.WriteFile(path, valid, 0o600); err != nil {
		t.Fatal(err)
	}
	if state := again.join(code); state.Code != code {
		t.Fatal("repaired room did not reopen")
	}

}

// seedGateBot reports its seed on started when a search starts, then waits
// for a release (or its context) before it plays the first legal action.
type seedGateBot struct {
	seed    uint64
	started chan<- uint64
	release <-chan struct{}
}

func (b seedGateBot) Choose(ctx context.Context, obs engine.Observation, bu ai.Budget, p func(ai.Progress)) (engine.Action, error) {
	b.started <- b.seed
	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return firstLegalBot{}.Choose(ctx, obs, bu, p)
}

// Eight bot rooms played from one address cannot hold up the bot of a room
// played from another by more than one search: in a pool of one slot, the
// other address's search runs next once the running one ends, ahead of
// the first address's seven waiting searches. A room whose player leaves
// while its search waits takes it out of the queue.
func TestBotSearchesAreFairBetweenAddresses(t *testing.T) {
	pool := ai.NewPoolPerKey(1, 2)
	started := make(chan uint64, 32)
	release := make(chan struct{})
	h := newHarness(t, func(c *room.Config) {
		c.BotPool = pool
		c.NewBot = func(_ store.BotLevel, seed uint64) room.Bot {
			return seedGateBot{seed: seed, started: started, release: release}
		}
	})
	// The bot of a room seeded s first decides at ply 0 with this seed.
	botSeed := func(roomSeed uint64) uint64 { return ai.MixSeed(roomSeed, 0) }
	open := func(user string, roomSeed uint64, addr string, tab int) *tclient {
		t.Helper()
		code, err := h.m.CreateFromSpec(room.Spec{
			Mode:    store.ModeAI,
			Seats:   [2]store.Seat{{Kind: store.SeatKindHuman, UserID: user, Name: "Ann"}, {Kind: store.SeatKindBot, Bot: store.BotHard}},
			Seed:    roomSeed,
			Options: engine.Options{FirstPlayer: engine.FirstPlayerSeat1},
		})
		if err != nil {
			t.Fatalf("CreateFromSpec: %v", err)
		}
		c := h.client(user, fmt.Sprint("tab-", tab), "Ann")
		c.Addr = addr
		c.join(code)
		return c
	}
	nextStart := func() uint64 {
		t.Helper()
		select {
		case s := <-started:
			return s
		case <-time.After(waitLimit):
			t.Fatal("no search started")
		}
		return 0
	}
	waitQueue := func(n int) {
		t.Helper()
		h.waitFor(fmt.Sprintf("%d searches waiting", n), func() bool { return pool.Waiting() == n })
	}

	const addrA, addrB = "192.0.2.1/32", "2001:db8:1:2::/64"
	var roomsA []*tclient
	for i := range 8 {
		roomsA = append(roomsA, open("user-a", uint64(100+i), addrA, i))
	}
	if got := nextStart(); got != botSeed(100) {
		t.Fatalf("the first search is %d, want the first room's", got)
	}
	waitQueue(7)
	b := open("user-b", 200, addrB, 99)
	waitQueue(8)

	// A player of the first address leaves a room whose search waits: the
	// search leaves the queue.
	h.m.Disconnect(roomsA[5].Client)
	waitQueue(7)

	release <- struct{}{} // the running search of the first address ends
	if got := nextStart(); got != botSeed(200) {
		t.Fatalf("after one search the next to run is %d, want the other address's %d", got, botSeed(200))
	}
	release <- struct{}{}
	b.waitState(func(s wire.State) bool { return s.Version == 1 })
	if pool.InUse() != 1 || pool.Waiting() != 5 {
		t.Fatalf("in use %d, waiting %d; want the first address's next search running and 5 waiting", pool.InUse(), pool.Waiting())
	}
}
