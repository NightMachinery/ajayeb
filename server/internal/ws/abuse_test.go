package ws

import (
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// These tests pin the per-address limits against a client that owns many
// addresses (an IPv6 /64) or tries to enumerate, farm or hammer rooms.

func TestLimitKey(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want string
	}{
		{"192.0.2.1", "192.0.2.1/32"},
		{"2001:db8:1:2::1", "2001:db8:1:2::/64"},
		{"2001:db8:1:2:ffff:ffff:ffff:ffff", "2001:db8:1:2::/64"},
		{"2001:db8:1:3::1", "2001:db8:1:3::/64"},
		{"::1", "::/64"},
	} {
		if got := limitKey(netip.MustParseAddr(tc.addr)); got != netip.MustParsePrefix(tc.want) {
			t.Errorf("limitKey(%s) = %v, want %s", tc.addr, got, tc.want)
		}
	}
	if got := limitKey(netip.Addr{}); got.IsValid() {
		t.Errorf("limitKey of the zero address = %v, want the zero prefix", got)
	}
}

// fromProxy is a harness option that trusts the loopback proxy, so a test
// can pick each client's address with forwardedFor.
func fromProxy(cfg *Config) {
	cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
}

func forwardedFor(ip string) http.Header { return http.Header{"X-Forwarded-For": {ip}} }

// TestSocketsPerIPv6Prefix checks that one IPv6 /64 is one client address
// for the socket limit: a host can use every address in its /64, so
// counting single addresses would give it billions of allowances.
func TestSocketsPerIPv6Prefix(t *testing.T) {
	h := newHarness(t, fromProxy)
	for i := range DefaultMaxConnsPerIP {
		h.mustDial(forwardedFor(netip.AddrFrom16([16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 1, 0, 2, 15: byte(i + 1)}).String()))
	}
	h.waitConns(DefaultMaxConnsPerIP)
	if status, code := h.dialStatus(forwardedFor("2001:db8:1:2:dead:beef:0:1")); status != http.StatusTooManyRequests || code != "too-many-connections" {
		t.Fatalf("socket %d from one /64: status %d code %q, want 429 too-many-connections", DefaultMaxConnsPerIP+1, status, code)
	}
	h.mustDial(forwardedFor("2001:db8:1:3::1")) // the next /64 has its own allowance
	h.waitConns(DefaultMaxConnsPerIP + 1)
}

// TestCreateLimitPerIPv6Prefix checks that the creation limit per address
// counts a whole /64 too.
func TestCreateLimitPerIPv6Prefix(t *testing.T) {
	h := newHarness(t, fromProxy, func(cfg *Config) { cfg.Limits.CreatesPerIP = 1 })
	h.clientWith(forwardedFor("2001:db8:1:2::1")).create(wire.ModeHotseat)
	late := h.clientWith(forwardedFor("2001:db8:1:2::2"))
	late.send(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	late.expectReject(0, wire.RejectCodeRate)
	h.clientWith(forwardedFor("2001:db8:1:3::1")).create(wire.ModeHotseat)
}

// TestJoinMissesAreLimited checks the limit on joins of codes that name no
// room, per client address: past it, every join is refused with reject rate
// (also of a room that exists, or guessing could go on), until the address
// earns a miss back.
func TestJoinMissesAreLimited(t *testing.T) {
	const misses = 3
	h := newHarness(t, fromProxy, func(cfg *Config) { cfg.Limits.JoinMissesPerIP = misses })
	owner := h.clientWith(forwardedFor("198.51.100.1"))
	owner.create(wire.ModeOnline)

	guesser := h.clientWith(forwardedFor("198.51.100.2"))
	for _, code := range []string{"ABCDEF", "BCDEFG", "CDEFGH"} {
		guesser.send(wire.NewJoin(code))
		if f := guesser.expect("error"); f.Code != string(wire.ErrorCodeRoomNotFound) {
			t.Fatalf("join of %s: %s", code, f.raw)
		}
	}
	guesser.send(wire.NewJoin(owner.code))
	guesser.expectReject(0, wire.RejectCodeRate)
	// Another connection from the same address is limited too.
	other := h.clientWith(forwardedFor("198.51.100.2"))
	other.send(wire.NewJoin(owner.code))
	other.expectReject(0, wire.RejectCodeRate)

	// Another address is not, and joins that find their room cost nothing.
	friend := h.clientWith(forwardedFor("198.51.100.3"))
	for range misses + 1 {
		friend.join(owner.code)
	}

	h.clock.Advance(time.Hour / misses)
	guesser.join(owner.code)
}

// TestUndoLimitPerIP checks the limit on undo frames per client address:
// each applied undo rewrites the room file, so undo must not be a way to
// make the server write as fast as the frame limit allows.
func TestUndoLimitPerIP(t *testing.T) {
	const undos = 2
	h := newHarness(t, func(cfg *Config) { cfg.Limits.UndosPerIP = undos })
	c := h.client()
	st := c.create(wire.ModeHotseat)
	for range undos {
		c.seq++
		c.send(wire.NewUndo(c.code, c.seq, st.Version))
		c.expectReject(c.seq, wire.RejectCodeIllegal) // nothing to undo, but it counts
	}
	c.seq++
	c.send(wire.NewUndo(c.code, c.seq, st.Version))
	c.expectReject(c.seq, wire.RejectCodeRate)

	h.clock.Advance(time.Minute / undos)
	c.send(wire.NewUndo(c.code, c.seq, st.Version))
	c.expectReject(c.seq, wire.RejectCodeIllegal)
}

// TestRematchCountsAgainstCreateLimit checks that a rematch that creates a
// room counts against the room creation limits, so finished games cannot be
// farmed into new rooms without limit.
func TestRematchCountsAgainstCreateLimit(t *testing.T) {
	h := newHarness(t, func(cfg *Config) { cfg.Limits.CreatesPerUser = 1 })
	code, err := h.rooms.CreateFromSpec(room.Spec{
		Mode:     store.ModeHotseat,
		Seats:    [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}},
		Seed:     42,
		Scenario: "science-supremacy",
	})
	if err != nil {
		t.Fatal(err)
	}
	c := h.client()
	c.create(wire.ModeHotseat) // the user's one create
	st := c.join(code)
	c.act(engine.NamedScenario("science-supremacy").Trigger[0], st.Version)
	if over := c.until("the game over", func(f frame) bool { return f.T == "state" && f.Ack != nil }); over.Room.Status != "finished" {
		t.Fatalf("after the winning move: %s", over.raw)
	}

	c.send(wire.NewRematch(code))
	c.expectReject(0, wire.RejectCodeRate)

	h.clock.Advance(time.Hour)
	c.send(wire.NewRematch(code))
	if created := c.expect("created"); created.Code == code || created.Code == "" {
		t.Fatalf("rematch created %q", created.Code)
	}
}
