package protocoltest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// decodeServerFrame peeks a server frame's "t" and strictly decodes it into
// the matching real wire type, so every test in this package works with the
// same types the server and the TypeScript client share, instead of a
// parallel hand-rolled struct.
func decodeServerFrame(raw []byte) (any, error) {
	var head struct {
		T string `json:"t"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("peek \"t\": %w", err)
	}
	switch head.T {
	case "welcome":
		var m wire.Welcome
		return m, json.Unmarshal(raw, &m)
	case "created":
		var m wire.Created
		return m, json.Unmarshal(raw, &m)
	case "state":
		var m wire.State
		return m, json.Unmarshal(raw, &m)
	case "reject":
		var m wire.Reject
		return m, json.Unmarshal(raw, &m)
	case "error":
		var m wire.Error
		return m, json.Unmarshal(raw, &m)
	default:
		return nil, fmt.Errorf("unknown server frame type %q", head.T)
	}
}

// rawCapture is every byte a connection has received, for the leak checks
// in redact_test.go, safe for a reader goroutine to write and the test
// goroutine to read concurrently.
type rawCapture struct {
	mu     sync.Mutex
	chunks [][]byte
}

func (c *rawCapture) add(b []byte) {
	cp := make([]byte, len(b))
	copy(cp, b)
	c.mu.Lock()
	c.chunks = append(c.chunks, cp)
	c.mu.Unlock()
}

// since returns the chunks received after the first n chunks, and the new
// total, so a caller can scan only what is new since its last check.
func (c *rawCapture) since(n int) ([][]byte, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n >= len(c.chunks) {
		return nil, len(c.chunks)
	}
	out := make([][]byte, len(c.chunks)-n)
	copy(out, c.chunks[n:])
	return out, len(c.chunks)
}

// tclient is one WebSocket connection, played the way a browser tab would:
// a background goroutine reads and decodes every frame (so the server is
// never kept waiting for a slow consumer, and a ping is answered the way
// coder/websocket answers one while Read is in progress), and the test
// goroutine consumes decoded frames and tracks its own idempotency state
// (seq) and the latest state it has seen, exactly as the web client's
// connection module would.
type tclient struct {
	t        *testing.T
	ws       *websocket.Conn
	token    string
	clientID string
	name     string

	frames chan any // decoded wire.Welcome/Created/State/Reject/Error
	raw    *rawCapture
	done   chan struct{} // closed when the reader goroutine stops
	err    error         // why the reader stopped; valid after done is closed

	seq    int
	code   string     // the room this client last created or joined
	latest wire.State // the most recent state frame seen; touched only by the test goroutine
}

var clientSerial int

// dial opens a socket to srv and starts its reader goroutine. name is a
// label for test failures only.
func dial(t *testing.T, srv *server, name string) *tclient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, srv.wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", name, err)
	}
	conn.SetReadLimit(1 << 20)
	token, err := ident.NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	clientSerial++
	c := &tclient{
		t:        t,
		ws:       conn,
		token:    token,
		clientID: fmt.Sprintf("tab-%d", clientSerial),
		name:     name,
		frames:   make(chan any, 256),
		raw:      &rawCapture{},
		done:     make(chan struct{}),
	}
	go c.read()
	t.Cleanup(func() {
		_ = c.ws.CloseNow()
		<-c.done
	})
	return c
}

func (c *tclient) read() {
	defer close(c.done)
	for {
		_, data, err := c.ws.Read(context.Background())
		if err != nil {
			c.err = err
			return
		}
		c.raw.add(data)
		msg, err := decodeServerFrame(data)
		if err != nil {
			c.err = fmt.Errorf("decode: %w (raw %s)", err, data)
			return
		}
		c.frames <- msg
	}
}

// send marshals and writes one client frame.
func (c *tclient) send(v wire.ClientMessage) {
	c.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		c.t.Fatalf("%s: marshal: %v", c.name, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, b); err != nil {
		c.t.Fatalf("%s: write: %v", c.name, err)
	}
}

// next returns the next decoded frame, failing the test if the socket
// closes or nothing arrives within wait.
func (c *tclient) next() any {
	c.t.Helper()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case f := <-c.frames:
		if st, ok := f.(wire.State); ok {
			c.latest = st
		}
		return f
	case <-c.done:
		select {
		case f := <-c.frames:
			if st, ok := f.(wire.State); ok {
				c.latest = st
			}
			return f
		default:
		}
		c.t.Fatalf("%s: socket closed (%v) while a frame was expected", c.name, c.err)
	case <-timer.C:
		c.t.Fatalf("%s: no frame arrived within %s", c.name, wait)
	}
	return nil
}

// until consumes frames until one satisfies ok, updating c.latest for every
// state frame seen along the way (including the matching one).
func (c *tclient) until(what string, ok func(any) bool) any {
	c.t.Helper()
	deadline := time.Now().Add(wait)
	for {
		if time.Now().After(deadline) {
			c.t.Fatalf("%s: timed out waiting for %s", c.name, what)
		}
		f := c.next()
		if ok(f) {
			return f
		}
	}
}

// untilState waits for a state frame of this client's current room whose
// Version is at least v. A state already seen that satisfies v (c.latest)
// is returned at once, without waiting for a new frame that may never
// come: a caller syncing several connections to a version only one of them
// actually advanced past must not block on the ones already there.
func (c *tclient) untilState(v int) wire.State {
	c.t.Helper()
	if c.latest.Version >= v {
		return c.latest
	}
	f := c.until(fmt.Sprintf("a state at version >= %d", v), func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Version >= v
	})
	return f.(wire.State)
}

// untilAck waits for the state frame that acknowledges seq. A reject of the
// same seq fails the test at once, with the reject's reason, instead of
// leaving untilAck to wait out the full timeout for an ack that a rejected
// request will never receive.
func (c *tclient) untilAck(seq int) wire.State {
	c.t.Helper()
	f := c.until(fmt.Sprintf("the ack for seq %d", seq), func(f any) bool {
		if r, ok := f.(wire.Reject); ok && r.Seq == seq {
			c.t.Fatalf("%s: seq %d was rejected (%s): %s", c.name, seq, r.Code, r.Msg)
		}
		st, ok := f.(wire.State)
		return ok && st.Ack != nil && st.Ack.Seq == seq
	})
	return f.(wire.State)
}

// untilReject waits for a reject of seq, skipping any state frames in
// between (a reject can arrive interleaved with another connection's state
// broadcasts).
func (c *tclient) untilReject(seq int) wire.Reject {
	c.t.Helper()
	f := c.until(fmt.Sprintf("a reject of seq %d", seq), func(f any) bool {
		r, ok := f.(wire.Reject)
		return ok && r.Seq == seq
	})
	return f.(wire.Reject)
}

// hello says hello and checks the welcome.
func (c *tclient) hello() {
	c.t.Helper()
	c.send(wire.NewHello(wire.ProtocolVersion, c.token, c.name, c.clientID))
	f := c.next()
	w, ok := f.(wire.Welcome)
	if !ok || w.V != wire.ProtocolVersion {
		c.t.Fatalf("%s: hello: want welcome v%d, got %#v", c.name, wire.ProtocolVersion, f)
	}
}

// create creates a room and returns its first state frame.
func (c *tclient) create(mode wire.Mode, bot *wire.BotLevel, seat wire.SeatChoice) wire.State {
	c.t.Helper()
	c.send(wire.NewCreate(mode, bot, seat, []string{c.name}))
	f := c.next()
	created, ok := f.(wire.Created)
	if !ok {
		c.t.Fatalf("%s: create: want created, got %#v", c.name, f)
	}
	c.code = created.Code
	st := c.until("the new room's state", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Code == created.Code
	}).(wire.State)
	return st
}

// join joins a room (as a player or, once its seats are taken, a
// spectator) and returns its state.
func (c *tclient) join(code string) wire.State {
	c.t.Helper()
	c.send(wire.NewJoin(code))
	c.code = code
	return c.until("the joined room's state", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Code == code
	}).(wire.State)
}

// nextSeq returns the next sequence number for an act or undo.
func (c *tclient) nextSeq() int {
	c.seq++
	return c.seq
}

// act sends an act against c.latest.Version with a fresh seq and returns
// it. seat is nil outside hot-seat rooms.
func (c *tclient) act(key string, seat *int) int {
	c.t.Helper()
	seq := c.nextSeq()
	c.send(wire.NewAct(c.code, seq, c.latest.Version, key, seat))
	return seq
}

// actAt is act with an explicit base, for the stale-base and
// duplicate-seq tests.
func (c *tclient) actAt(seq, base int, key string, seat *int) {
	c.t.Helper()
	c.send(wire.NewAct(c.code, seq, base, key, seat))
}

// undo sends an undo against c.latest.Version with a fresh seq.
func (c *tclient) undo() int {
	c.t.Helper()
	seq := c.nextSeq()
	c.send(wire.NewUndo(c.code, seq, c.latest.Version))
	return seq
}

// seatInt is a small helper for building Act.Seat (*int) literals inline.
func seatInt(seat int) *int { return &seat }
