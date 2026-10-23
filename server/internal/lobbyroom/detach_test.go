package lobbyroom

import (
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestLeaveThenJoinStaysAttached: a connection that leaves the lobby and
// joins it again on the same socket ends up attached, whichever the actor
// takes first when both wait: the detach the leave queued, or the join.
// (Its spectator's Start was once answered bad-request, as if it had never
// joined.)
func TestLeaveThenJoinStaysAttached(t *testing.T) {
	f := newFixture(t, nil)
	code := f.create(f.conn(user(0), "c0", "P0"), 3, wire.LobbyOptions{})
	k := f.conn(user(1), "c1", "P1")
	f.join(k, code)
	a := f.lobbies.lookup(code)
	for round := range 20 {
		// Hold the actor, so the leave's detach and the join's attach
		// both wait for it.
		held, release := make(chan struct{}), make(chan struct{})
		go func() { _ = a.call(f.ctx(), func() { close(held); <-release }) }()
		<-held
		f.m.Leave(k.c, code)
		joined := make(chan error, 1)
		go func() { joined <- f.m.Join(f.ctx(), k.c, code) }()
		for deadline := time.Now().Add(wait); len(a.inbox) == 0; time.Sleep(time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatal("the join never reached the lobby")
			}
		}
		close(release)
		if err := <-joined; err != nil {
			t.Fatal(err)
		}
		var attached bool
		if err := a.call(f.ctx(), func() { attached = slices.Contains(a.conns, k.c) }); err != nil {
			t.Fatal(err)
		}
		if !attached {
			t.Fatalf("round %d: the connection was detached after it joined again", round)
		}
	}
}
