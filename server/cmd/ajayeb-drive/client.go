package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// frameWait bounds how long client waits for one frame. It is a safety net
// against a hung server, not a timing assumption: every wait returns the
// moment the frame it is looking for arrives.
const frameWait = 30 * time.Second

// minSendInterval paces this connection's outgoing frames comfortably under
// internal/ws's default per-connection limit (10 frames/s, burst 20): a
// full game is dozens of actions sent back to back, which a human clicking
// never does but a driver happily would, and tripping the rate limit
// repeatedly in a row closes the socket outright (DefaultMaxRateRejects).
// act retries a stray "rate" reject too, as a second line of defense.
const minSendInterval = 110 * time.Millisecond

// client is one WebSocket connection to a running ajayeb server, playing
// the protocol the way a browser tab would: hello once, then create or
// join, then act (and, in hot-seat and ai rooms, undo) against the version
// it last saw. Unlike internal/protocoltest's tclient (which fails the
// test on anything unexpected), every method here returns an error, since
// this is a command-line tool driving a real, separately started binary,
// not a Go test.
type client struct {
	ws       *websocket.Conn
	token    string
	clientID string
	name     string

	seq    int
	code   string
	latest wire.State

	frames   chan any
	errc     chan error // the reader goroutine's reason for stopping, sent once
	lastSend time.Time  // for pacing; zero until the first send
}

// decodeServerFrame peeks a server frame's "t" and strictly decodes it into
// the matching real wire type, the type the server and the TypeScript
// client actually share.
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

// dial opens a socket to wsURL and says hello, deriving a fresh identity
// (a random token, so every connection this process makes is a distinct
// user; playOnline dials twice for exactly that reason). xff, when
// non-empty, is sent as X-Forwarded-For: harmless against a server that
// does not trust this process as a proxy, and the way --concurrency gets
// past internal/ws's per-address socket cap (16 by default) against a
// server started with --trust-proxy naming this process's own address, for
// a true many-games-at-once memory sweep from one driver.
func dial(ctx context.Context, wsURL, name, xff string) (*client, error) {
	var opts *websocket.DialOptions
	if xff != "" {
		opts = &websocket.DialOptions{HTTPHeader: map[string][]string{"X-Forwarded-For": {xff}}}
	}
	conn, _, err := websocket.Dial(ctx, wsURL, opts)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", wsURL, err)
	}
	conn.SetReadLimit(1 << 20)
	token, err := ident.NewToken()
	if err != nil {
		_ = conn.CloseNow()
		return nil, fmt.Errorf("new token: %w", err)
	}
	c := &client{
		ws:       conn,
		token:    token,
		clientID: fmt.Sprintf("drive-%s-%d", name, time.Now().UnixNano()),
		name:     name,
		frames:   make(chan any, 256),
		errc:     make(chan error, 1),
	}
	go c.read()
	if err := c.hello(ctx); err != nil {
		c.close()
		return nil, err
	}
	return c, nil
}

func (c *client) read() {
	for {
		_, data, err := c.ws.Read(context.Background())
		if err != nil {
			c.errc <- err
			return
		}
		msg, err := decodeServerFrame(data)
		if err != nil {
			c.errc <- fmt.Errorf("decode: %w (raw %s)", err, data)
			return
		}
		c.frames <- msg
	}
}

// close drops the connection. Safe to call once the reader has already
// stopped on its own (a server-initiated close).
func (c *client) close() { _ = c.ws.CloseNow() }

// pace blocks until it is safe to send another frame on this connection
// without risking the per-connection rate limit, then records the send.
func (c *client) pace(ctx context.Context) error {
	if !c.lastSend.IsZero() {
		if wait := minSendInterval - time.Since(c.lastSend); wait > 0 {
			t := time.NewTimer(wait)
			defer t.Stop()
			select {
			case <-t.C:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	c.lastSend = time.Now()
	return nil
}

func (c *client) send(ctx context.Context, v wire.ClientMessage) error {
	if err := c.pace(ctx); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := c.ws.Write(ctx, websocket.MessageText, b); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// next returns the next decoded frame.
func (c *client) next(ctx context.Context) (any, error) {
	wctx, cancel := context.WithTimeout(ctx, frameWait)
	defer cancel()
	select {
	case f := <-c.frames:
		if st, ok := f.(wire.State); ok {
			c.latest = st
		}
		return f, nil
	case err := <-c.errc:
		// A frame already queued ahead of the close is still delivered.
		select {
		case f := <-c.frames:
			if st, ok := f.(wire.State); ok {
				c.latest = st
			}
			return f, nil
		default:
		}
		return nil, fmt.Errorf("%s: connection closed: %w", c.name, err)
	case <-wctx.Done():
		return nil, fmt.Errorf("%s: no frame arrived within %s", c.name, frameWait)
	}
}

// until consumes frames until ok reports true for one, or an error (a
// protocol error, an unexpected reject, a closed connection, a timeout).
func (c *client) until(ctx context.Context, ok func(any) (bool, error)) (any, error) {
	for {
		f, err := c.next(ctx)
		if err != nil {
			return nil, err
		}
		done, err := ok(f)
		if err != nil {
			return nil, err
		}
		if done {
			return f, nil
		}
	}
}

// untilState waits for a state at version >= v, or returns an error if a
// reject arrives first (any reject at this level is a protocol error: this
// driver only ever sends base/seq pairs and keys its own bookkeeping says
// are legal, so a reject means the server and the driver disagree, which is
// exactly what --games is meant to catch).
func (c *client) untilState(ctx context.Context, v int) (wire.State, error) {
	if c.latest.Version >= v {
		return c.latest, nil
	}
	f, err := c.until(ctx, func(f any) (bool, error) {
		if r, ok := f.(wire.Reject); ok {
			return false, fmt.Errorf("%s: unexpected reject (%s): %s", c.name, r.Code, r.Msg)
		}
		if e, ok := f.(wire.Error); ok {
			return false, fmt.Errorf("%s: server error (%s): %s", c.name, e.Code, e.Msg)
		}
		st, ok := f.(wire.State)
		return ok && st.Version >= v, nil
	})
	if err != nil {
		return wire.State{}, err
	}
	return f.(wire.State), nil
}

// rejectError is a reject frame, kept as a typed error so a caller (act's
// retry loop) can tell a transient "rate" reject apart from a real protocol
// disagreement (stale, illegal, not-your-turn), which --games exists to
// catch.
type rejectError struct {
	seq  int
	code wire.RejectCode
	msg  string
}

func (e *rejectError) Error() string {
	return fmt.Sprintf("seq %d rejected (%s): %s", e.seq, e.code, e.msg)
}

// untilAck waits for the state that acknowledges seq.
func (c *client) untilAck(ctx context.Context, seq int) (wire.State, error) {
	f, err := c.until(ctx, func(f any) (bool, error) {
		if r, ok := f.(wire.Reject); ok && r.Seq == seq {
			return false, &rejectError{seq: seq, code: r.Code, msg: r.Msg}
		}
		if e, ok := f.(wire.Error); ok {
			return false, fmt.Errorf("%s: server error (%s): %s", c.name, e.Code, e.Msg)
		}
		st, ok := f.(wire.State)
		return ok && st.Ack != nil && st.Ack.Seq == seq, nil
	})
	if err != nil {
		return wire.State{}, err
	}
	return f.(wire.State), nil
}

func (c *client) hello(ctx context.Context) error {
	if err := c.send(ctx, wire.NewHello(wire.ProtocolVersion, c.token, c.name, c.clientID)); err != nil {
		return err
	}
	f, err := c.next(ctx)
	if err != nil {
		return err
	}
	w, ok := f.(wire.Welcome)
	if !ok {
		return fmt.Errorf("%s: hello: want welcome, got %#v", c.name, f)
	}
	if w.V != wire.ProtocolVersion {
		return fmt.Errorf("%s: server speaks protocol v%d, this driver speaks v%d", c.name, w.V, wire.ProtocolVersion)
	}
	return nil
}

func (c *client) create(ctx context.Context, mode wire.Mode, bot *wire.BotLevel, seat wire.SeatChoice) (wire.State, error) {
	if err := c.send(ctx, wire.NewCreate(mode, bot, seat, []string{c.name})); err != nil {
		return wire.State{}, err
	}
	f, err := c.next(ctx)
	if err != nil {
		return wire.State{}, err
	}
	created, ok := f.(wire.Created)
	if !ok {
		return wire.State{}, fmt.Errorf("%s: create: want created, got %#v", c.name, f)
	}
	c.code = created.Code
	st, err := c.until(ctx, func(f any) (bool, error) {
		st, ok := f.(wire.State)
		return ok && st.Code == created.Code, nil
	})
	if err != nil {
		return wire.State{}, err
	}
	return st.(wire.State), nil
}

func (c *client) join(ctx context.Context, code string) (wire.State, error) {
	if err := c.send(ctx, wire.NewJoin(code)); err != nil {
		return wire.State{}, err
	}
	c.code = code
	st, err := c.until(ctx, func(f any) (bool, error) {
		st, ok := f.(wire.State)
		return ok && st.Code == code, nil
	})
	if err != nil {
		return wire.State{}, err
	}
	return st.(wire.State), nil
}

func (c *client) nextSeq() int {
	c.seq++
	return c.seq
}

// act sends an act against c.latest.Version and waits for its ack. seat is
// nil outside hot-seat rooms.
// maxRateRetries bounds how many times act resends a rate-limited action
// before giving up: pacing (minSendInterval) should make this never
// trigger in practice, so a long run of them is itself worth failing on
// rather than retrying forever.
const maxRateRetries = 20

// act sends an act against c.latest.Version and waits for its ack,
// transparently resending (with a fresh seq, the same base) if the server
// answers "reject rate": a well-behaved client slows down and tries again,
// it does not treat its own request rate as a protocol bug.
func (c *client) act(ctx context.Context, key string, seat *int) (wire.State, error) {
	for attempt := 0; ; attempt++ {
		seq := c.nextSeq()
		if err := c.send(ctx, wire.NewAct(c.code, seq, c.latest.Version, key, seat)); err != nil {
			return wire.State{}, err
		}
		st, err := c.untilAck(ctx, seq)
		if err == nil {
			return st, nil
		}
		var rej *rejectError
		if !errors.As(err, &rej) || rej.code != wire.RejectCodeRate || attempt >= maxRateRetries {
			return wire.State{}, fmt.Errorf("%s: %w", c.name, err)
		}
	}
}
