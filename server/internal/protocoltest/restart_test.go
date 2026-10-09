package protocoltest

import (
	"math/rand/v2"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestRestartPersistence stops a server mid-game and starts a new one on
// the same data directory, the way a redeploy does (docs/protocol.md,
// infra design section 2's "Persistence"): the same user, identified by
// the same token against the same secret.key, picks the game back up and
// finishes it, and an independent replay of the room's seed and action log
// afterward gives the same hash as the live, finished state.
func TestRestartPersistence(t *testing.T) {
	dir := t.TempDir()
	rng := rand.New(rand.NewPCG(41, 42))

	srv1 := newServer(t, dir, 9_900_000_000_001)
	c := dial(t, srv1, "Human")
	c.hello()
	token, clientID := c.token, c.clientID
	bot := wire.BotLevelEasy
	c.create(wire.ModeAI, &bot, wire.SeatNumber(0))
	code := c.code

	for played := 0; played < 3; {
		if c.latest.You.CanAct {
			key := pickChoice(rng, c.latest.View.Choices)
			seq := c.act(key, nil)
			c.untilAck(seq)
			played++
		} else {
			c.untilState(c.latest.Version + 1)
		}
	}
	versionBeforeRestart := c.latest.Version

	srv1.close() // the first server's rooms are already on disk after every accepted action

	srv2 := newServer(t, dir, 9_900_100_000_001) // same data dir; a different seed base matters only if it ever created a new room, which this test does not do
	r := dial(t, srv2, "Human-resumed")
	r.token, r.clientID = token, clientID
	r.hello()
	resumed := r.join(code)
	// >=, not ==: the bot's reply to the last human move runs in its own
	// goroutine and may finish (and persist) in the window between this
	// test reading its own ack and calling srv1.close(), landing one
	// version ahead of what the client observed. The room is never behind
	// what the client last saw.
	if resumed.Version < versionBeforeRestart {
		t.Fatalf("restart: resumed at version %d, want at least %d", resumed.Version, versionBeforeRestart)
	}

	for !over(r) {
		if r.latest.You.CanAct {
			key := pickChoice(rng, r.latest.View.Choices)
			seq := r.act(key, nil)
			r.untilAck(seq)
		} else {
			r.untilState(r.latest.Version + 1)
		}
	}
	if r.latest.View.Result == nil {
		t.Fatalf("restart: game ended without a result")
	}
	checkReplay(t, srv2, code)
}
