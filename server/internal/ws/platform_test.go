package ws

import (
	"encoding/json"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func clientAtVersion(h *harness, version int) *tclient {
	h.t.Helper()
	c := h.connect(nil)
	c.send(wire.NewHello(version, c.token, "Sara", c.clientID))
	welcome := c.expect("welcome")
	if welcome.V != version || welcome.Server == nil || welcome.Server.Version != "test-1" {
		h.t.Fatalf("negotiation: %s", welcome.raw)
	}
	return c
}

func TestPlatformDuelPlaysOnBothVersions(t *testing.T) {
	for _, version := range []int{wire.ProtocolVersion, wire.PlatformProtocolVersion} {
		t.Run(map[int]string{1: "v1", 2: "v2"}[version], func(t *testing.T) {
			h := newHarness(t, func(cfg *Config) { cfg.Limits.FrameBurst = 1000 })
			c := clientAtVersion(h, version)
			st := c.create(wire.ModeHotseat)
			// Request rejection, action, duplicate, undo and reconnect all retain Duel
			// frame families and fields; then play a complete game with server offers.
			c.act("missing", st.Version)
			c.expectReject(c.seq, wire.RejectCodeIllegal)
			c.send(wire.NewResync(c.code))
			st = c.until("resync", func(f frame) bool { return f.T == "state" })
			for move := 0; st.Room.Status != "finished"; move++ {
				if move > 200 || st.View == nil || len(st.View.Choices) == 0 {
					t.Fatalf("game stuck: %s", st.raw)
				}
				seq := c.act(st.View.Choices[0].Key, st.Version)
				st = c.until("ack", func(f frame) bool { return f.T == "state" && f.Ack != nil && f.Ack.Seq == seq })
				if move == 0 {
					c.send(wire.NewAct(c.code, seq, 0, "missing", nil))
					duplicate := c.until("duplicate", func(f frame) bool { return f.T == "state" && f.Ack != nil && f.Ack.Seq == seq })
					if duplicate.Version != st.Version {
						t.Fatal("duplicate applied twice")
					}
					c.seq++
					c.send(wire.NewUndo(c.code, c.seq, st.Version))
					st = c.until("undo", func(f frame) bool { return f.T == "state" && f.Ack != nil && f.Ack.Seq == c.seq })
					if st.View.Ply != 0 {
						t.Fatal("undo did not restore initial position")
					}
					// A new authenticated connection uses the original token in hello.
					reconnect := h.connect(nil)
					reconnect.token = c.token
					reconnect.send(wire.NewHello(version, reconnect.token, "Sara", reconnect.clientID))
					reconnect.expect("welcome")
					again := reconnect.join(c.code)
					if again.Version != st.Version || again.View.Ply != st.View.Ply {
						t.Fatal("reconnect changed position")
					}
					reconnect.send(wire.NewLeave(c.code))
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(st.raw, &fields); err != nil {
					t.Fatal(err)
				}
				if _, exists := fields["game"]; exists {
					t.Fatal("Duel gained game discriminator")
				}
			}
		})
	}
}

func TestPlatformFamiliesAtTransportBoundary(t *testing.T) {
	h := newHarness(t)
	old := clientAtVersion(h, 1)
	old.send([]byte(`{"t":"lobby-create","seatCount":3,"options":{}}`))
	if got := old.expect("error"); got.Code != "protocol" {
		t.Fatalf("v1 create: %s", got.raw)
	}
	old.send([]byte(`{"t":"table-pick","code":"REGAAA","seq":1,"point":{"id":"sides","revision":0},"key":"side:day","confirm":false}`))
	if got := old.expect("error"); got.Code != "protocol" {
		t.Fatalf("v1 table: %s", got.raw)
	}
	c := clientAtVersion(h, 2)
	c.send([]byte(`{"t":"table-pick","code":"REGAAA","seq":9,"point":{"id":"sides","revision":0},"key":"x"}`))
	if got := c.expect("table-reject"); got.Code != "bad-request" || got.Seq != 9 {
		t.Fatalf("strict table decode: %s", got.raw)
	}
	c.send([]byte(`{"t":"lobby-start","code":"REGAAA","seq":10,"fillBots":false,"extra":true}`))
	if got := c.expect("lobby-reject"); got.Code != "bad-request" || got.Seq != 10 {
		t.Fatalf("strict lobby decode: %s", got.raw)
	}
	c.send([]byte(`{"t":"lobby-create","seatCount":3,"options":{}}`))
	if got := c.expect("error"); got.Code != "forbidden" {
		t.Fatalf("unregistered production lobby: %s", got.raw)
	}
	if h.rooms.Len() != 0 {
		t.Fatal("platform request created production actor")
	}
}
