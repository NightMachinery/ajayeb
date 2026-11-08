package tableroom

import (
	"context"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

func TestRoomWorkQueueHoldsNoSearchPermit(t *testing.T) {
	pool, gate := ai.NewPool(2), ai.NewPool(1)
	if err := gate.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := newTH(t, Config{Pool: pool, RoomWork: gate}, soloSpec(3, 913, BotNormal))
	waitFor(t, "room-work waiter", func() bool { return gate.Waiting() == 1 })
	if pool.InUse() != 0 || pool.Waiting() != 0 {
		t.Fatal("queued table reserved a shared search permit")
	}
	// A Duel search can acquire and release while the table is queued.
	if err := pool.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	pool.Release()
	h.leave(tabletest.SeatClient(0))
	waitFor(t, "cancelled room-work waiter", func() bool { return gate.Waiting() == 0 })
	gate.Release()
	h.f.close()
	if gate.InUse() != 0 || pool.InUse() != 0 || pool.Waiting() != 0 {
		t.Fatal("admission permit leak")
	}
}

func TestRoomWorkReleasedBeforeObserverAndResultDelivery(t *testing.T) {
	pool, gate := ai.NewPool(2), ai.NewPool(1)
	reports := make(chan bool, 4)
	h := newTH(t, Config{Pool: pool, RoomWork: gate, BotFinished: func(BotReport) {
		// The result is sent to the actor only after this callback returns,
		// so the next seat cannot have acquired either permit yet.
		reports <- gate.InUse() == 0 && pool.InUse() == 0
	}}, soloSpec(3, 913, BotEasy))
	for range 2 {
		select {
		case free := <-reports:
			if !free {
				t.Fatal("search permit held through observer/result delivery")
			}
		case <-h.ctx.Done():
			t.Fatal("bot report did not arrive")
		}
	}
	h.f.close()
}
