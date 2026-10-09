package protocoltest

import (
	"math/rand/v2"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestReconnectMidGame drops a connection mid-game and reconnects with the
// same identity (the token and clientId a browser tab would keep in
// storage): the reconnected client picks the game back up at the current
// version, and resending the last act's seq (as a client would if it never
// learned whether its first send got through) does not apply it twice.
func TestReconnectMidGame(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_800_000_000_001)
	rng := rand.New(rand.NewPCG(31, 32))

	c := dial(t, srv, "Human")
	c.hello()
	bot := wire.BotLevelEasy
	c.create(wire.ModeAI, &bot, wire.SeatNumber(0))
	code, token, clientID := c.code, c.token, c.clientID

	// Play a couple of human turns normally first.
	for played := 0; played < 2; {
		if c.latest.You.CanAct {
			key := pickChoice(rng, c.latest.View.Choices)
			seq := c.act(key, nil)
			c.untilAck(seq)
			played++
		} else {
			c.untilState(c.latest.Version + 1)
		}
	}
	for !c.latest.You.CanAct {
		c.untilState(c.latest.Version + 1)
	}

	// One more action, acknowledged normally, then the socket drops before
	// the tab would have read anything else (a closed laptop lid, a lost
	// WiFi signal).
	key := pickChoice(rng, c.latest.View.Choices)
	seq := c.act(key, nil)
	settled := c.untilAck(seq)
	_ = c.ws.CloseNow()

	r := dial(t, srv, "Human-reconnected")
	r.token, r.clientID = token, clientID
	r.hello()
	rejoined := r.join(code)
	if rejoined.Version < settled.Version {
		t.Fatalf("reconnect: rejoined at version %d, want at least %d (the room keeps playing while a tab is away)", rejoined.Version, settled.Version)
	}

	// Resend the already-applied seq: a browser that reconnects before
	// confirming a send would retry it with the same seq, and it must not
	// apply a second time.
	r.actAt(seq, rejoined.Version, key, nil)
	after := r.untilAck(seq)
	if after.Version != rejoined.Version {
		t.Fatalf("reconnect: resending an applied seq changed the version: got %d, want unchanged %d", after.Version, rejoined.Version)
	}

	// The game is still playable from the reconnected connection.
	for !over(r) {
		if r.latest.You.CanAct {
			k := pickChoice(rng, r.latest.View.Choices)
			s := r.act(k, nil)
			r.untilAck(s)
		} else {
			r.untilState(r.latest.Version + 1)
		}
	}
	if r.latest.View.Result == nil {
		t.Fatalf("reconnect: game ended without a result")
	}
}
