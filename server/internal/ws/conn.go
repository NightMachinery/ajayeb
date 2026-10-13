package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// conn is one WebSocket connection. Three kinds of goroutine work on it:
//
//   - the read loop (serve's goroutine, the HTTP handler's) reads frames,
//     applies the frame limit, answers hello and routes every other request
//     to the rooms, one at a time;
//   - the writer takes frames from the outbox and writes them, and is the
//     one that closes the socket once the connection ends;
//   - at most MaxMissedPongs ping goroutines, each waiting for one pong.
//
// The clock's callbacks (the hello deadline and the ping ticks) only take
// mu, change a field and maybe end the connection; they never block.
//
// Ending a connection (end) is a decision recorded once, with the close
// status to send. The writer then flushes what is queued (an error frame
// explaining the end, for example) and closes the socket with that status,
// which also ends the read loop; serve then detaches the client from its
// room and gives back its slot.
type conn struct {
	s   *Server
	ws  *websocket.Conn
	ip  netip.Addr   // the client address, for logs
	key netip.Prefix // the client address as the per-address limits count it (limitKey)
	log *slog.Logger
	out *room.Outbox

	// Owned by the read loop.
	client  *room.Client // set by hello
	frames  bucket       // the frame limit
	rejects int          // frames refused in a row by the frame limit

	ctx        context.Context // for room requests; cancelled when the connection ends
	cancel     context.CancelFunc
	ended      chan struct{} // closed by end
	writerDone chan struct{}
	pings      sync.WaitGroup

	mu         sync.Mutex
	ending     bool
	status     websocket.StatusCode
	reason     string
	abrupt     bool   // close without the closing handshake (the peer is gone)
	greeted    bool   // hello arrived
	clientID   string // from hello; for tests and logs
	unanswered int    // pings sent since the last pong
	stopHello  func() bool
	stopPing   func() bool
}

// newConn starts the hello deadline and the pings of a new connection.
func newConn(s *Server, ws *websocket.Conn, ip netip.Addr, key netip.Prefix) *conn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &conn{
		s:          s,
		ws:         ws,
		ip:         ip,
		key:        key,
		log:        s.log.With("ip", ip),
		out:        room.NewOutbox(),
		ctx:        ctx,
		cancel:     cancel,
		ended:      make(chan struct{}),
		writerDone: make(chan struct{}),
	}
	c.mu.Lock()
	c.stopHello = s.clock.AfterFunc(s.lim.HelloTimeout, c.helloExpired)
	c.armPingLocked()
	c.mu.Unlock()
	return c
}

// serve runs the connection until it closes, then cleans up after it.
func (c *conn) serve() {
	c.log.Debug("socket opened")
	go c.writeLoop()
	c.readLoop()

	c.end(websocket.StatusNormalClosure, "", false)
	<-c.writerDone
	_ = c.ws.CloseNow() // in case the writer's Close gave up
	c.pings.Wait()
	if c.client != nil {
		c.s.cfg.Rooms.Disconnect(c.client)
	}
	c.out.Close()
	c.mu.Lock()
	status, reason := c.status, c.reason
	c.mu.Unlock()
	c.log.Debug("socket closed", "status", int(status), "reason", reason)
}

// end records that the connection is over and why: the writer closes the
// socket with status and reason, or at once without the closing handshake
// when abrupt is set (a peer that stopped answering). Only the first call
// counts.
func (c *conn) end(status websocket.StatusCode, reason string, abrupt bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.endLocked(status, reason, abrupt)
}

func (c *conn) endLocked(status websocket.StatusCode, reason string, abrupt bool) {
	if c.ending {
		return
	}
	c.ending = true
	c.status, c.reason, c.abrupt = status, reason, abrupt
	if c.stopHello != nil {
		c.stopHello()
		c.stopHello = nil
	}
	if c.stopPing != nil {
		c.stopPing()
		c.stopPing = nil
	}
	c.cancel()
	close(c.ended)
}

// isEnding reports whether end has been called.
func (c *conn) isEnding() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ending
}

// --- timers ------------------------------------------------------------------

// helloExpired ends a connection that has not said hello in time.
func (c *conn) helloExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.greeted || c.ending {
		return
	}
	c.endLocked(websocket.StatusPolicyViolation, "hello timeout", false)
}

// armPingLocked schedules the next ping tick. The caller holds mu.
func (c *conn) armPingLocked() {
	c.stopPing = c.s.clock.AfterFunc(c.s.lim.PingInterval, c.pingTick)
}

// pingTick sends a ping, or drops the connection when the last
// MaxMissedPongs pings all went unanswered.
func (c *conn) pingTick() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ending {
		return
	}
	if c.unanswered >= c.s.lim.MaxMissedPongs {
		c.endLocked(websocket.StatusPolicyViolation, "missed pongs", true)
		return
	}
	c.unanswered++
	c.pings.Add(1)
	go c.ping()
	c.armPingLocked()
}

// ping sends one ping and waits for its pong, which clears the count of
// unanswered pings. It gives up when the socket closes, or after a real-time
// safety net longer than the ping ticks that would drop the connection.
func (c *conn) ping() {
	defer c.pings.Done()
	wait := time.Duration(c.s.lim.MaxMissedPongs+1) * c.s.lim.PingInterval
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	if err := c.ws.Ping(ctx); err == nil {
		c.mu.Lock()
		c.unanswered = 0
		c.mu.Unlock()
	}
}

// --- reading -----------------------------------------------------------------

// readLoop reads and handles frames until the socket fails or closes, or
// the connection ends.
func (c *conn) readLoop() {
	for {
		typ, data, err := c.ws.Read(context.Background())
		if err != nil {
			if status := websocket.CloseStatus(err); status != -1 {
				c.log.Debug("peer closed the socket", "status", int(status))
			} else if !c.isEnding() {
				c.log.Debug("reading failed", "err", err)
			}
			return
		}
		if c.isEnding() {
			return
		}
		if typ != websocket.MessageText {
			c.end(websocket.StatusUnsupportedData, "text frames only", false)
			return
		}
		if !c.s.frame.allow(&c.frames, c.s.clock.Now()) {
			c.rejects++
			if c.rejects > c.s.lim.MaxRateRejects {
				c.end(websocket.StatusPolicyViolation, "rate limit", false)
				return
			}
			c.out.Push(wire.NewReject(peekSeq(data), wire.RejectCodeRate, "too many frames; slow down"))
			continue
		}
		c.rejects = 0
		if !c.handle(data) {
			return
		}
	}
}

// handle answers one frame within the frame limit, and reports whether
// the connection goes on.
func (c *conn) handle(data []byte) bool {
	if c.client == nil {
		return c.hello(data)
	}
	msg, err := wire.DecodeClient(data)
	if err != nil {
		c.out.Push(wire.NewReject(peekSeq(data), wire.RejectCodeBadRequest, "malformed frame"))
		return true
	}
	rooms := c.s.cfg.Rooms
	switch m := msg.(type) {
	case wire.Hello:
		c.out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "hello was already received"))
	case wire.Create:
		if !c.s.allowCreate(c.client.UserID, c.key) {
			c.out.Push(wire.NewReject(0, wire.RejectCodeRate, "too many rooms created; try again later"))
			return true
		}
		_, err = rooms.Create(c.ctx, c.client, m)
	case wire.Join:
		if !c.s.allowJoin(c.key) {
			c.out.Push(wire.NewReject(0, wire.RejectCodeRate, "too many joins of unknown rooms; try again later"))
			return true
		}
		err = rooms.JoinClaim(c.ctx, c.client, m.Code, m.Claim)
		// A bad claim counts like an unknown room: both are guesses.
		if errors.Is(err, room.ErrRoomNotFound) || errors.Is(err, room.ErrBadClaim) {
			c.s.joinMissed(c.key)
		}
	case wire.Act:
		err = rooms.Act(c.ctx, c.client, m)
	case wire.Undo:
		if !c.s.allowUndo(c.key) {
			c.out.Push(wire.NewReject(m.Seq, wire.RejectCodeRate, "too many undos; slow down"))
			return true
		}
		err = rooms.Undo(c.ctx, c.client, m)
	case wire.Resync:
		err = rooms.Resync(c.ctx, c.client, m.Code)
	case wire.Leave:
		rooms.Leave(c.client, m.Code)
	case wire.Rematch:
		_, err = rooms.Rematch(c.ctx, c.client, m.Code)
	case wire.Hints:
		err = rooms.Hints(c.ctx, c.client, m)
	case wire.Link:
		err = rooms.Link(c.ctx, c.client, m)
	case wire.Relink:
		err = rooms.Relink(c.ctx, c.client, m)
	}
	switch {
	case err == nil:
	case errors.Is(err, room.ErrClosed):
		c.end(CloseServiceRestart, "server restart", false)
		return false
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return false
	default:
		// The rooms have already told the client.
		c.log.Debug("request refused", "err", err)
	}
	return true
}

// hello answers the first frame: a valid hello in the supported protocol
// version gets a welcome, anything else an error frame and the end of the
// connection (status 1008).
func (c *conn) hello(data []byte) bool {
	if v, ok := peekHelloVersion(data); ok && v != wire.ProtocolVersion {
		return c.refuse(wire.ErrorCodeProtocol, "unsupported protocol version")
	}
	msg, err := wire.DecodeClient(data)
	h, ok := msg.(wire.Hello)
	if err != nil || !ok {
		return c.refuse(wire.ErrorCodeBadHello, "the first frame must be a valid hello")
	}
	userID, err := c.s.cfg.Identity.UserID(h.Token)
	if err != nil {
		return c.refuse(wire.ErrorCodeBadHello, "the first frame must be a valid hello")
	}

	c.mu.Lock()
	if c.ending {
		c.mu.Unlock()
		return false
	}
	c.greeted = true
	c.clientID = h.ClientID
	if c.stopHello != nil {
		c.stopHello()
		c.stopHello = nil
	}
	c.mu.Unlock()

	c.client = &room.Client{
		UserID:   userID,
		ClientID: h.ClientID,
		Name:     h.Name,
		Out:      c.out,
		// A rematch that creates a room counts against the same limits as
		// a create.
		AllowCreate: func() bool { return c.s.allowCreate(userID, c.key) },
		// The bot pool shares its slots fairly between addresses.
		Addr: c.key.String(),
	}
	c.log.Debug("hello", "user", userID)
	c.out.Push(c.s.welcome)
	return true
}

// refuse sends an error frame and ends the connection with status 1008.
func (c *conn) refuse(code wire.ErrorCode, msg string) bool {
	c.out.Push(wire.NewError(code, msg))
	c.end(websocket.StatusPolicyViolation, string(code), false)
	return false
}

// peekSeq returns the "seq" of a frame, for the reject that answers it, or
// 0 when it has none.
func peekSeq(data []byte) int {
	var head struct {
		Seq int `json:"seq"`
	}
	if json.Unmarshal(data, &head) != nil || head.Seq < 0 {
		return 0
	}
	return head.Seq
}

// peekHelloVersion returns the "v" of a frame whose "t" is hello, and
// whether it is one. A hello without "v" reads as version 0.
func peekHelloVersion(data []byte) (int, bool) {
	var head struct {
		T string `json:"t"`
		V int    `json:"v"`
	}
	if json.Unmarshal(data, &head) != nil || head.T != "hello" {
		return 0, false
	}
	return head.V, true
}

// --- writing -----------------------------------------------------------------

// writeLoop writes the outbox's frames until the connection ends, then
// closes the socket.
func (c *conn) writeLoop() {
	defer close(c.writerDone)
	for {
		select {
		case <-c.out.Ready():
			if err := c.flush(); err != nil {
				if !c.isEnding() {
					c.log.Debug("writing failed", "err", err)
				}
				c.end(websocket.StatusInternalError, "", true)
				continue
			}
			if c.out.Overflowed() {
				c.end(websocket.StatusPolicyViolation, "client too slow", false)
			}
		case <-c.ended:
			c.finish()
			return
		}
	}
}

// finish closes the socket the way end decided: after the frames still
// queued (such as an error frame that explains the end), with the closing
// handshake, or at once.
func (c *conn) finish() {
	c.mu.Lock()
	status, reason, abrupt := c.status, c.reason, c.abrupt
	c.mu.Unlock()
	if abrupt {
		_ = c.ws.CloseNow()
		return
	}
	_ = c.flush()
	_ = c.ws.Close(status, reason)
}

// flush writes every queued frame, each within the write timeout.
func (c *conn) flush() error {
	for _, m := range c.out.Take() {
		b, err := json.Marshal(m)
		if err != nil {
			c.log.Error("encoding a frame failed", "err", err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.s.lim.WriteTimeout)
		err = c.ws.Write(ctx, websocket.MessageText, b)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}
