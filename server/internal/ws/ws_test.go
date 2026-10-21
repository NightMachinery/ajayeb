package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestNewRequiresRoomsAndIdentity(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("New without Rooms succeeded")
	}
	m := &room.Manager{}
	if _, err := New(Config{Rooms: m}); err == nil {
		t.Error("New without Identity succeeded")
	}
}

func TestHelloWelcome(t *testing.T) {
	h := newHarness(t)
	c := h.client() // checks the welcome's version, build and rules
	if got := h.srv.Conns(); got != 1 {
		t.Errorf("open sockets = %d, want 1", got)
	}
	// A second hello is refused, without closing.
	c.send(wire.NewHello(wire.ProtocolVersion, c.token, "Sara", c.clientID))
	c.expectReject(0, wire.RejectCodeBadRequest)
	c.send(wire.NewResync("ZZZZZZ"))
	c.expectReject(0, wire.RejectCodeBadRequest)
}

func TestHelloTimeout(t *testing.T) {
	h := newHarness(t)
	c := h.connect(nil)
	h.waitConns(1)
	sc := h.conns()[0]

	h.clock.Advance(DefaultHelloTimeout - time.Millisecond)
	if sc.isEnding() {
		t.Fatal("the connection ended before the hello timeout")
	}
	h.clock.Advance(time.Millisecond)
	if status := c.closed(); status != websocket.StatusPolicyViolation {
		t.Errorf("close status = %d, want 1008", status)
	}
	h.waitConns(0)
}

func TestHelloInTimeStopsTheDeadline(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	h.clock.Advance(DefaultHelloTimeout + time.Second)
	c.send(wire.NewResync("ZZZZZZ"))
	c.expectReject(0, wire.RejectCodeBadRequest)
	if got := h.srv.Conns(); got != 1 {
		t.Errorf("open sockets = %d, want 1 after the hello deadline passed", got)
	}
}

func TestHelloRefused(t *testing.T) {
	token := strings.Repeat("A", 22)
	for _, tc := range []struct {
		name  string
		frame string
		code  wire.ErrorCode
	}{
		{"not a hello", `{"t":"join","code":"K7QX2M"}`, wire.ErrorCodeBadHello},
		{"not json", `hello`, wire.ErrorCodeBadHello},
		{"bad token", `{"t":"hello","v":1,"token":"short","name":"","clientId":"c1"}`, wire.ErrorCodeBadHello},
		{"no client id", `{"t":"hello","v":1,"token":"` + token + `","name":"","clientId":""}`, wire.ErrorCodeBadHello},
		{"unknown field", `{"t":"hello","v":1,"token":"` + token + `","name":"","clientId":"c1","x":1}`, wire.ErrorCodeBadHello},
		{"newer protocol", `{"t":"hello","v":3,"token":"` + token + `","name":"","clientId":"c1"}`, wire.ErrorCodeProtocol},
		{"no version", `{"t":"hello","token":"` + token + `","name":"","clientId":"c1"}`, wire.ErrorCodeProtocol},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			c := h.connect(nil)
			c.send([]byte(tc.frame))
			e := c.expect("error")
			if e.Code != string(tc.code) {
				t.Errorf("error code = %q, want %q", e.Code, tc.code)
			}
			if status := c.closed(); status != websocket.StatusPolicyViolation {
				t.Errorf("close status = %d, want 1008", status)
			}
			h.waitConns(0)
		})
	}
}

func TestReadLimit(t *testing.T) {
	h := newHarness(t)
	c := h.client()

	// A frame of exactly ReadLimit bytes is read (and refused as malformed:
	// its code is far too long).
	head, tail := `{"t":"resync","code":"`, `"}`
	fits := head + strings.Repeat("A", int(ReadLimit)-len(head)-len(tail)) + tail
	if len(fits) != int(ReadLimit) {
		t.Fatalf("test frame is %d bytes", len(fits))
	}
	c.send([]byte(fits))
	c.expectReject(0, wire.RejectCodeBadRequest)

	// One byte more closes the socket with 1009.
	c.send([]byte(head + strings.Repeat("A", int(ReadLimit)-len(head)-len(tail)+1) + tail))
	if status := c.closed(); status != websocket.StatusMessageTooBig {
		t.Errorf("close status = %d, want 1009", status)
	}
	h.waitConns(0)
}

func TestBinaryFrameCloses(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageBinary, []byte(`{"t":"resync","code":"K7QX2M"}`)); err != nil {
		t.Fatal(err)
	}
	if status := c.closed(); status != websocket.StatusUnsupportedData {
		t.Errorf("close status = %d, want 1003", status)
	}
}

func TestFrameRateLimit(t *testing.T) {
	h := newHarness(t)
	c := h.client() // the hello is the bucket's first frame

	// The clock stands still, so the bucket never refills: 19 more frames
	// fit, and leave frames get no answer.
	for range DefaultFrameBurst - 1 {
		c.send(wire.NewLeave("K7QX2M"))
	}
	for seq := 101; seq <= 103; seq++ {
		c.send(wire.NewAct("K7QX2M", seq, 0, "construct:baths", nil))
	}
	for seq := 101; seq <= 103; seq++ {
		r := c.expect("reject")
		if r.Code != string(wire.RejectCodeRate) || r.Seq != seq {
			t.Fatalf("frame over the limit: got %s, want reject rate for seq %d", r.raw, seq)
		}
	}

	// 100 ms earn one frame back: it is handled (the room says join first).
	h.clock.Advance(100 * time.Millisecond)
	c.send(wire.NewAct("K7QX2M", 104, 0, "construct:baths", nil))
	c.expectReject(104, wire.RejectCodeBadRequest)
	c.send(wire.NewAct("K7QX2M", 105, 0, "construct:baths", nil))
	c.expectReject(105, wire.RejectCodeRate)

	// A full second earns ten.
	h.clock.Advance(time.Second)
	for i := range DefaultFrameRate {
		c.send(wire.NewAct("K7QX2M", 200+i, 0, "construct:baths", nil))
	}
	for i := range DefaultFrameRate {
		c.expectReject(200+i, wire.RejectCodeBadRequest)
	}
	if got := h.srv.Conns(); got != 1 {
		t.Errorf("open sockets = %d, want 1", got)
	}
}

func TestTooManyRateRejectsClose(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	for range DefaultFrameBurst - 1 {
		c.send(wire.NewLeave("K7QX2M"))
	}
	// DefaultMaxRateRejects refused frames in a row are each answered; the
	// next one closes the socket.
	for i := range DefaultMaxRateRejects + 1 {
		c.send(wire.NewAct("K7QX2M", 300+i, 0, "construct:baths", nil))
	}
	for i := range DefaultMaxRateRejects {
		c.expectReject(300+i, wire.RejectCodeRate)
	}
	if status := c.closed(); status != websocket.StatusPolicyViolation {
		t.Errorf("close status = %d, want 1008", status)
	}
	h.waitConns(0)
}

func TestSocketsPerIP(t *testing.T) {
	h := newHarness(t)
	socks := make([]*websocket.Conn, 0, DefaultMaxConnsPerIP)
	for range DefaultMaxConnsPerIP {
		socks = append(socks, h.mustDial(nil))
	}
	h.waitConns(DefaultMaxConnsPerIP)
	if status, code := h.dialStatus(nil); status != http.StatusTooManyRequests || code != "too-many-connections" {
		t.Fatalf("socket %d: status %d code %q, want 429 too-many-connections", DefaultMaxConnsPerIP+1, status, code)
	}
	// Closing one frees its slot.
	_ = socks[0].CloseNow()
	h.waitConns(DefaultMaxConnsPerIP - 1)
	h.mustDial(nil)
	h.waitConns(DefaultMaxConnsPerIP)
}

func TestSocketsPerForwardedIP(t *testing.T) {
	h := newHarness(t, func(cfg *Config) {
		cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	})
	from := func(ip string) http.Header { return http.Header{"X-Forwarded-For": {ip}} }
	for range DefaultMaxConnsPerIP {
		h.mustDial(from("198.51.100.1"))
	}
	h.waitConns(DefaultMaxConnsPerIP)
	if status, _ := h.dialStatus(from("198.51.100.1")); status != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429 for the 17th socket of one forwarded address", status)
	}
	// Another client behind the same proxy has its own allowance, even
	// when it lies about the hops before the proxy.
	h.mustDial(from("198.51.100.1, 198.51.100.2"))
	h.mustDial(from("198.51.100.2"))
	h.waitConns(DefaultMaxConnsPerIP + 2)
}

func TestForwardedForIgnoredFromUntrustedPeer(t *testing.T) {
	h := newHarness(t) // no trusted proxies
	for i := range DefaultMaxConnsPerIP {
		h.mustDial(http.Header{"X-Forwarded-For": {fmt.Sprintf("198.51.100.%d", i+1)}})
	}
	h.waitConns(DefaultMaxConnsPerIP)
	status, _ := h.dialStatus(http.Header{"X-Forwarded-For": {"203.0.113.50"}})
	if status != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429: an untrusted peer's X-Forwarded-For must not count", status)
	}
}

func TestMaxConns(t *testing.T) {
	h := newHarness(t, func(cfg *Config) { cfg.Limits.MaxConns = 3 })
	for range 3 {
		h.mustDial(nil)
	}
	h.waitConns(3)
	if status, code := h.dialStatus(nil); status != http.StatusServiceUnavailable || code != "busy" {
		t.Fatalf("status %d code %q, want 503 busy", status, code)
	}
}

func TestNonUpgradeRequest(t *testing.T) {
	h := newHarness(t)
	resp, err := h.httpc.Get(h.hs.URL + "/ws")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired {
		t.Errorf("status %d, want 426", resp.StatusCode)
	}
	h.srv.mu.Lock()
	total, perIP := h.srv.total, len(h.srv.perIP)
	h.srv.mu.Unlock()
	if total != 0 || perIP != 0 {
		t.Errorf("a refused upgrade kept its slot: total %d, addresses %d", total, perIP)
	}
}

// paced creates a room and then lets the clock earn its frame back, so a
// long run of creates stays inside the frame limit (and earns back far too
// little to matter for the creation limits: one create per minute or more).
func (c *tclient) paced(mode wire.Mode) {
	c.t.Helper()
	c.create(mode)
	c.h.clock.Advance(time.Second / DefaultFrameRate)
}

func TestCreateLimitPerUser(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	for range DefaultCreatesPerUser {
		c.paced(wire.ModeHotseat)
	}
	c.send(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	c.expectReject(0, wire.RejectCodeRate)

	// Another user at the same address has its own allowance.
	h.client().create(wire.ModeHotseat)

	// One create is earned back every three minutes.
	h.clock.Advance(3 * time.Minute)
	c.create(wire.ModeHotseat)
	c.send(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	c.expectReject(0, wire.RejectCodeRate)
}

func TestCreateLimitPerIP(t *testing.T) {
	h := newHarness(t, func(cfg *Config) {
		cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	})
	from := func(ip string) http.Header { return http.Header{"X-Forwarded-For": {ip}} }
	for range DefaultCreatesPerIP / DefaultCreatesPerUser {
		c := h.clientWith(from("198.51.100.1"))
		for range DefaultCreatesPerUser {
			c.paced(wire.ModeHotseat)
		}
	}
	// A fresh user at the same address is refused: the address is spent.
	late := h.clientWith(from("198.51.100.1"))
	late.send(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	late.expectReject(0, wire.RejectCodeRate)
	// Another address is not.
	h.clientWith(from("198.51.100.2")).create(wire.ModeHotseat)

	// The refusal used up nothing of the user's own allowance: once the
	// address earns a create back, that user gets it.
	h.clock.Advance(time.Minute)
	late.create(wire.ModeHotseat)
}

func TestPingKeepsLiveClient(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	sc := h.serverConn(c.clientID)
	for tick := range 4 {
		h.clock.Advance(DefaultPingInterval)
		waitUntil(t, fmt.Sprintf("the pong to ping %d", tick+1), func() bool {
			sc.mu.Lock()
			defer sc.mu.Unlock()
			return sc.unanswered == 0
		})
	}
	if sc.isEnding() {
		t.Fatal("a client that answers pings was dropped")
	}
	c.send(wire.NewResync("ZZZZZZ"))
	c.expectReject(0, wire.RejectCodeBadRequest)
}

func TestPingDropsSilentClient(t *testing.T) {
	h := newHarness(t)
	ws := h.mustDial(nil) // never reads, so never answers a ping
	h.waitConns(1)
	sc := h.conns()[0]
	hello, err := json.Marshal(wire.NewHello(wire.ProtocolVersion, strings.Repeat("A", 22), "", "silent"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	if err := ws.Write(ctx, websocket.MessageText, hello); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the hello", func() bool {
		sc.mu.Lock()
		defer sc.mu.Unlock()
		return sc.greeted
	})

	for range DefaultMaxMissedPongs {
		h.clock.Advance(DefaultPingInterval)
		if sc.isEnding() {
			t.Fatal("dropped before two pings went unanswered")
		}
	}
	h.clock.Advance(DefaultPingInterval)
	if !sc.isEnding() {
		t.Fatal("not dropped at the tick after two unanswered pings")
	}
	h.waitConns(0)
	for {
		if _, _, err := ws.Read(ctx); err != nil {
			break // the welcome and the pings first, then the end of the socket
		}
	}
}

func TestOrigins(t *testing.T) {
	site := "https://ajayeb.example"
	for _, tc := range []struct {
		dev    bool
		origin string
		ok     bool
	}{
		{false, "", true}, // a tool, not a browser
		{false, site, true},
		{false, "HTTPS://AJAYEB.EXAMPLE", true},
		{false, "http://ajayeb.example", false}, // the scheme counts
		{false, "https://evil.example", false},
		{false, "http://localhost:38952", false},
		{false, "http://127.0.0.1:38952", false},
		{true, site, true},
		{true, "http://localhost:38952", true},
		{true, "http://localhost", true},
		{true, "http://127.0.0.1:5173", true},
		{true, "https://evil.example", false},
		{true, "http://localhost.evil.example", false},
	} {
		t.Run(fmt.Sprintf("dev=%v %s", tc.dev, tc.origin), func(t *testing.T) {
			h := newHarness(t, func(cfg *Config) {
				cfg.Origins = []string{site}
				cfg.Dev = tc.dev
			})
			header := http.Header{}
			if tc.origin != "" {
				header.Set("Origin", tc.origin)
			}
			if tc.ok {
				h.mustDial(header)
				return
			}
			if status, _ := h.dialStatus(header); status != http.StatusForbidden {
				t.Errorf("status %d, want 403", status)
			}
		})
	}
}

func TestOwnHostOriginIsAccepted(t *testing.T) {
	h := newHarness(t, func(cfg *Config) { cfg.Origins = []string{"https://ajayeb.example"} })
	h.mustDial(http.Header{"Origin": {h.hs.URL}})
}

func TestPlayHotseatOverTheSocket(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	st := c.create(wire.ModeHotseat)
	if st.Room == nil || st.Room.Mode != "hotseat" || st.View == nil || len(st.View.Choices) == 0 {
		t.Fatalf("first state = %s", st.raw)
	}
	if st.You == nil || len(st.You.Seats) != 2 || !st.You.CanAct {
		t.Fatalf("a hot-seat creator should hold both seats: %s", st.raw)
	}

	// Act: the state that answers carries the ack.
	seq := c.act(st.View.Choices[0].Key, st.Version)
	next := c.until("the acked state", func(f frame) bool { return f.T == "state" && f.Ack != nil })
	if next.Ack.Seq != seq || next.Ack.ClientID != c.clientID || next.Version != st.Version+1 || next.View.Ply != st.View.Ply+1 {
		t.Fatalf("after the act: %s", next.raw)
	}
	// A duplicate seq is only acknowledged.
	c.send(wire.NewAct(c.code, seq, st.Version, st.View.Choices[0].Key, nil))
	dup := c.until("the duplicate's ack", func(f frame) bool { return f.T == "state" && f.Ack != nil })
	if dup.Version != next.Version {
		t.Fatalf("a duplicate act was applied again: %s", dup.raw)
	}
	// A stale base is rejected, then a fresh state follows.
	c.act(next.View.Choices[0].Key, st.Version)
	c.expectReject(c.seq, wire.RejectCodeStale)
	fresh := c.expect("state")
	if fresh.Version != next.Version {
		t.Fatalf("state after stale = %s", fresh.raw)
	}
	// Undo takes the move back.
	c.seq++
	c.send(wire.NewUndo(c.code, c.seq, fresh.Version))
	undone := c.until("the undo", func(f frame) bool { return f.T == "state" && f.Ack != nil && f.Ack.Seq == c.seq })
	if undone.View.Ply != st.View.Ply || undone.Version != fresh.Version+1 {
		t.Fatalf("after undo: %s", undone.raw)
	}
	// Resync sends the same position again.
	c.send(wire.NewResync(c.code))
	again := c.expect("state")
	if again.Version != undone.Version {
		t.Fatalf("resync = %s", again.raw)
	}
	// Leave detaches; joining again brings the seats back.
	c.send(wire.NewLeave(c.code))
	back := c.join(c.code)
	if back.You == nil || len(back.You.Seats) != 2 {
		t.Fatalf("rejoined state = %s", back.raw)
	}
}

func TestOnlineRoomJoinAndSpectate(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	lobby := a.create(wire.ModeOnline)
	if lobby.Room.Status != "lobby" {
		t.Fatalf("an online room starts in the lobby: %s", lobby.raw)
	}
	b := h.client()
	joined := b.join(a.code)
	if joined.Room.Status != "playing" || joined.You == nil || len(joined.You.Seats) != 1 || joined.You.Seats[0] != 1 {
		t.Fatalf("second user's state = %s", joined.raw)
	}
	spectator := h.client()
	watched := spectator.join(a.code)
	if watched.You == nil || len(watched.You.Seats) != 0 || watched.Room.Spectators != 1 {
		t.Fatalf("spectator's state = %s", watched.raw)
	}
	// The seat not to move is told so.
	mover, waiter := a, b
	if joined.View.ToMove != nil && *joined.View.ToMove == 1 {
		mover, waiter = b, a
	}
	waiter.act("draft:pyramids", joined.Version)
	waiter.expectReject(waiter.seq, wire.RejectCodeNotYourTurn)
	cur := mover.until("the playing state", func(f frame) bool {
		return f.T == "state" && f.Room.Status == "playing" && f.View != nil && len(f.View.Choices) > 0
	})
	mover.act(cur.View.Choices[0].Key, cur.Version)
	mover.until("the move", func(f frame) bool { return f.T == "state" && f.Ack != nil && f.Ack.Seq == mover.seq })
	spectator.until("the move as a spectator sees it", func(f frame) bool { return f.T == "state" && f.Version == cur.Version+1 })
}

func TestRematchOverTheSocket(t *testing.T) {
	h := newHarness(t)
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
	st := c.join(code)
	trigger := engine.NamedScenario("science-supremacy").Trigger
	if len(trigger) != 1 {
		t.Fatalf("science-supremacy trigger = %q, want one key", trigger)
	}
	c.act(trigger[0], st.Version)
	over := c.until("the game over", func(f frame) bool { return f.T == "state" && f.Ack != nil })
	if over.Room.Status != "finished" {
		t.Fatalf("after the winning move: %s", over.raw)
	}
	c.send(wire.NewRematch(code))
	created := c.expect("created")
	if created.Code == code || created.Code == "" {
		t.Fatalf("rematch created %q", created.Code)
	}
	fresh := c.join(created.Code)
	if fresh.Room.Status != "playing" || fresh.Version != 0 {
		t.Fatalf("rematch room state = %s", fresh.raw)
	}
}

func TestShutdownClosesWithServiceRestart(t *testing.T) {
	h := newHarness(t)
	a := h.client()
	a.create(wire.ModeHotseat)
	b := h.client()
	pending := h.connect(nil) // still before hello
	h.waitConns(3)

	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	if err := h.srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	for _, c := range []*tclient{a, b, pending} {
		if status := c.closed(); status != CloseServiceRestart {
			t.Errorf("client %s: close status %d, want 1012", c.clientID, status)
		}
	}
	if got := h.srv.Conns(); got != 0 {
		t.Errorf("open sockets after Shutdown = %d", got)
	}
	snap, err := h.rooms.Snapshot(ctx, a.code)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Clients != 0 {
		t.Errorf("the room still has %d clients after Shutdown", snap.Clients)
	}
	if status, code := h.dialStatus(nil); status != http.StatusServiceUnavailable || code != "shutting-down" {
		t.Errorf("dial after Shutdown: %d %q, want 503 shutting-down", status, code)
	}
}

func TestClosedRoomsEndTheConnection(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	h.rooms.Close()
	c.send(wire.NewJoin("K7QX2M"))
	if status := c.closed(); status != CloseServiceRestart {
		t.Errorf("close status %d, want 1012 once the rooms are closing", status)
	}
}

func TestDisconnectDetachesFromTheRoom(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	c.create(wire.ModeHotseat)
	_ = c.ws.Close(websocket.StatusNormalClosure, "bye")
	h.waitConns(0)
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	snap, err := h.rooms.Snapshot(ctx, c.code)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Clients != 0 {
		t.Errorf("room clients = %d after the socket closed, want 0", snap.Clients)
	}
}

func TestTokensNeverLeak(t *testing.T) {
	h := newHarness(t)
	c := h.client()
	st := c.create(wire.ModeHotseat)
	c.act(st.View.Choices[0].Key, st.Version)
	c.until("the move", func(f frame) bool { return f.T == "state" && f.Ack != nil })
	c.send([]byte(`{"t":"act","code":"` + c.token + `"}`))
	c.expectReject(0, wire.RejectCodeBadRequest)
	bad := h.connect(nil)
	bad.send([]byte(`{"t":"hello","v":1,"token":"` + c.token + `x","name":"","clientId":"leak"}`))
	bad.expect("error")
	bad.closed()

	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	h.waitConns(0)
	if strings.Contains(c.raw.String(), c.token) {
		t.Error("a server frame contains the client's token")
	}
	if strings.Contains(h.logs.String(), c.token) {
		t.Errorf("the log contains the client's token:\n%s", h.logs.String())
	}
	if !strings.Contains(h.logs.String(), "socket closed") {
		t.Error("the debug log is empty; the leak check proves nothing")
	}
}
