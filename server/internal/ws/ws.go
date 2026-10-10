package ws

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// CloseServiceRestart is the close status sent to every open socket when the
// server shuts down. Status 1012 (service restart) tells clients to reconnect
// with backoff, so a redeploy loses no game.
const CloseServiceRestart = websocket.StatusServiceRestart

// ReadLimit is the largest client frame the server accepts, in bytes. A
// larger frame closes the connection with status 1009 (message too big).
const ReadLimit int64 = wire.MaxClientFrameBytes

// The defaults of the zero fields of Limits.
const (
	// DefaultHelloTimeout is how long a new connection may take to send
	// its hello frame.
	DefaultHelloTimeout = 10 * time.Second
	// DefaultFrameRate and DefaultFrameBurst are each connection's frame
	// limit: a token bucket of 20 frames that refills at 10 per second.
	DefaultFrameRate  = 10
	DefaultFrameBurst = 20
	// DefaultMaxRateRejects is how many frames in a row a connection may
	// send over its frame limit (each answered with reject rate) before the
	// next one closes it with status 1008.
	DefaultMaxRateRejects = 20
	// DefaultMaxConns bounds the open sockets of the whole server.
	DefaultMaxConns = 1024
	// DefaultMaxConnsPerIP bounds the open sockets of one client address
	// (an IPv4 address, or an IPv6 /64; see limitKey).
	DefaultMaxConnsPerIP = 16
	// DefaultCreatesPerUser and DefaultCreatesPerIP bound room creation:
	// token buckets that hold that many creates and refill at that many
	// per hour.
	DefaultCreatesPerUser = 20
	DefaultCreatesPerIP   = 60
	// DefaultJoinMissesPerIP bounds the joins of one client address that
	// name no room: a token bucket that holds that many and refills at that
	// many per hour. Past it every join of the address is refused, so one
	// address cannot guess room codes at the frame rate (many addresses
	// still can; see the package doc).
	DefaultJoinMissesPerIP = 30
	// DefaultUndosPerIP bounds the undo frames of one client address: a
	// token bucket that holds that many and refills at that many per
	// minute. Every applied undo replays and rewrites a room file.
	DefaultUndosPerIP = 60
	// DefaultLimiterKeys bounds the user ids and the addresses each keyed
	// limiter remembers.
	DefaultLimiterKeys = 4096
	// DefaultPingInterval is the time between two pings of a connection.
	DefaultPingInterval = 25 * time.Second
	// DefaultMaxMissedPongs is how many pings in a row may go unanswered:
	// at the next ping time after that, the connection is dropped.
	DefaultMaxMissedPongs = 2
	// DefaultWriteTimeout bounds the writing of one frame to the socket.
	DefaultWriteTimeout = 10 * time.Second
)

// Rooms is what the transport needs from the rooms; *room.Manager
// satisfies it. Every method reports its outcome to the client as frames in
// the client's outbox; the error is only for the transport (see
// room.Manager). Calls for one client never overlap: a connection's read
// loop makes them one after another, and Disconnect comes last.
type Rooms interface {
	Create(ctx context.Context, c *room.Client, req wire.Create) (string, error)
	Join(ctx context.Context, c *room.Client, code string) error
	Leave(c *room.Client, code string)
	Disconnect(c *room.Client)
	Act(ctx context.Context, c *room.Client, act wire.Act) error
	Undo(ctx context.Context, c *room.Client, undo wire.Undo) error
	Resync(ctx context.Context, c *room.Client, code string) error
	Rematch(ctx context.Context, c *room.Client, code string) (string, error)
}

// Identity derives a user id from a client token; ident.Secret satisfies
// it. The token itself goes no further than this call.
type Identity interface {
	UserID(token string) (string, error)
}

// Limits holds the abuse limits. A zero field takes the default of the
// same name.
type Limits struct {
	HelloTimeout    time.Duration
	FrameRate       int // frames per second
	FrameBurst      int
	MaxRateRejects  int
	MaxConns        int
	MaxConnsPerIP   int
	CreatesPerUser  int // per hour
	CreatesPerIP    int // per hour
	JoinMissesPerIP int // per hour
	UndosPerIP      int // per minute
	LimiterKeys     int
	PingInterval    time.Duration
	MaxMissedPongs  int
	WriteTimeout    time.Duration
}

func (l Limits) withDefaults() Limits {
	def := func(v *int, d int) {
		if *v <= 0 {
			*v = d
		}
	}
	defDur := func(v *time.Duration, d time.Duration) {
		if *v <= 0 {
			*v = d
		}
	}
	defDur(&l.HelloTimeout, DefaultHelloTimeout)
	def(&l.FrameRate, DefaultFrameRate)
	def(&l.FrameBurst, DefaultFrameBurst)
	def(&l.MaxRateRejects, DefaultMaxRateRejects)
	def(&l.MaxConns, DefaultMaxConns)
	def(&l.MaxConnsPerIP, DefaultMaxConnsPerIP)
	def(&l.CreatesPerUser, DefaultCreatesPerUser)
	def(&l.CreatesPerIP, DefaultCreatesPerIP)
	def(&l.JoinMissesPerIP, DefaultJoinMissesPerIP)
	def(&l.UndosPerIP, DefaultUndosPerIP)
	def(&l.LimiterKeys, DefaultLimiterKeys)
	defDur(&l.PingInterval, DefaultPingInterval)
	def(&l.MaxMissedPongs, DefaultMaxMissedPongs)
	defDur(&l.WriteTimeout, DefaultWriteTimeout)
	return l
}

// Config holds a Server's dependencies and settings.
type Config struct {
	// Rooms receives the routed requests. Required.
	Rooms Rooms
	// Identity derives user ids from tokens. Required.
	Identity Identity
	// Version is the build version the welcome frame reports.
	Version string

	// Origins are the site origins (scheme://host[:port]) allowed to open
	// a socket besides the server's own host.
	Origins []string
	// Dev also allows loopback origins on any port (the Vite dev server).
	Dev bool
	// TrustedProxies are the peers whose X-Forwarded-For is believed.
	TrustedProxies []netip.Prefix

	// Clock drives the hello deadline, the pings and the rate limits.
	// Default room.SystemClock.
	Clock room.Clock
	// Logger receives the transport's logs. Default: discard. Tokens never
	// reach it.
	Logger *slog.Logger
	// Limits are the abuse limits.
	Limits Limits
}

// Server serves the /ws endpoint. Make one with New; it is an http.Handler.
type Server struct {
	cfg     Config
	lim     Limits
	clock   room.Clock
	log     *slog.Logger
	accept  websocket.AcceptOptions
	frame   rate
	welcome wire.Welcome
	limitMu sync.Mutex                  // guards the keyed limiters below
	byUser  *keyedLimiter[string]       // creates per user id
	byIP    *keyedLimiter[netip.Prefix] // creates per client address
	misses  *keyedLimiter[netip.Prefix] // joins that named no room, per client address
	undos   *keyedLimiter[netip.Prefix] // undo frames per client address

	mu     sync.Mutex
	conns  map[*conn]struct{}
	perIP  map[netip.Prefix]int
	total  int // sockets reserved, including those still upgrading
	closed bool
	wg     sync.WaitGroup // one per reserved socket
}

// New returns a Server.
func New(cfg Config) (*Server, error) {
	if cfg.Rooms == nil {
		return nil, errors.New("ws: Config.Rooms is required")
	}
	if cfg.Identity == nil {
		return nil, errors.New("ws: Config.Identity is required")
	}
	if cfg.Clock == nil {
		cfg.Clock = room.SystemClock{}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	lim := cfg.Limits.withDefaults()
	s := &Server{
		cfg:   cfg,
		lim:   lim,
		clock: cfg.Clock,
		log:   cfg.Logger,
		accept: websocket.AcceptOptions{
			OriginPatterns: originPatterns(cfg.Origins, cfg.Dev),
		},
		frame:   newRate(lim.FrameRate, time.Second, lim.FrameBurst),
		welcome: wire.NewWelcome(wire.ProtocolVersion, wire.ServerInfo{Version: cfg.Version, Rules: engine.RulesVersion}),
		byUser:  newKeyedLimiter[string](newRate(lim.CreatesPerUser, time.Hour, lim.CreatesPerUser), lim.LimiterKeys),
		byIP:    newKeyedLimiter[netip.Prefix](newRate(lim.CreatesPerIP, time.Hour, lim.CreatesPerIP), lim.LimiterKeys),
		misses:  newKeyedLimiter[netip.Prefix](newRate(lim.JoinMissesPerIP, time.Hour, lim.JoinMissesPerIP), lim.LimiterKeys),
		undos:   newKeyedLimiter[netip.Prefix](newRate(lim.UndosPerIP, time.Minute, lim.UndosPerIP), lim.LimiterKeys),
		conns:   make(map[*conn]struct{}),
		perIP:   make(map[netip.Prefix]int),
	}
	return s, nil
}

// ServeHTTP upgrades a request to a WebSocket and serves it until it
// closes. A client address that already has its share of sockets gets 429,
// and a full or stopping server 503, before the upgrade; Accept answers a
// bad origin with 403 and a request that is not a WebSocket handshake with
// 426.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r, s.cfg.TrustedProxies)
	key := limitKey(ip)
	if status, code := s.reserve(key); status != 0 {
		writeError(w, status, code)
		return
	}
	defer s.wg.Done()

	ws, err := websocket.Accept(w, r, &s.accept)
	if err != nil {
		s.release(key)
		s.log.Debug("websocket upgrade refused", "ip", ip, "err", err)
		return
	}
	ws.SetReadLimit(ReadLimit)

	c := newConn(s, ws, ip, key)
	s.register(c)
	c.serve()
	s.unregister(c)
}

// reserve counts a socket from the client address key against the limits,
// or returns the HTTP status and error code that refuse it.
func (s *Server) reserve(key netip.Prefix) (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.closed:
		return http.StatusServiceUnavailable, "shutting-down"
	case s.total >= s.lim.MaxConns:
		return http.StatusServiceUnavailable, "busy"
	case s.perIP[key] >= s.lim.MaxConnsPerIP:
		return http.StatusTooManyRequests, "too-many-connections"
	}
	s.total++
	s.perIP[key]++
	s.wg.Add(1)
	return 0, ""
}

// release gives back a socket reserved for the client address key.
func (s *Server) release(key netip.Prefix) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releaseLocked(key)
}

func (s *Server) releaseLocked(key netip.Prefix) {
	s.total--
	if s.perIP[key]--; s.perIP[key] <= 0 {
		delete(s.perIP, key)
	}
}

// register adds an upgraded connection to the set Shutdown closes. A
// connection that arrives after Shutdown began is closed at once.
func (s *Server) register(c *conn) {
	s.mu.Lock()
	s.conns[c] = struct{}{}
	closed := s.closed
	s.mu.Unlock()
	if closed {
		c.end(CloseServiceRestart, "server restart", false)
	}
}

// unregister removes a finished connection and gives back its socket.
func (s *Server) unregister(c *conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, c)
	s.releaseLocked(c.key)
}

// Shutdown refuses new sockets, closes every open one with status 1012
// (service restart) and waits for their connections to finish, which
// detaches them from their rooms. If ctx ends first, the remaining sockets
// are closed without a handshake and ctx's error is returned once their
// connections have finished.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	conns := make([]*conn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		c.end(CloseServiceRestart, "server restart", false)
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
	}
	s.mu.Lock()
	for c := range s.conns {
		_ = c.ws.CloseNow()
	}
	s.mu.Unlock()
	<-done
	return ctx.Err()
}

// Conns returns the number of open sockets.
func (s *Server) Conns() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// allowCreate reports whether a user at the client address key may create
// a room now, and counts the creation if so. Both the user's and the
// address's limit must allow it; a refusal uses up neither.
func (s *Server) allowCreate(userID string, key netip.Prefix) bool {
	now := s.clock.Now()
	s.limitMu.Lock()
	defer s.limitMu.Unlock()
	if !s.byUser.check(userID, now) || !s.byIP.check(key, now) {
		return false
	}
	s.byUser.take(userID, now)
	s.byIP.take(key, now)
	return true
}

// allowJoin reports whether the client address key may join a room now: it
// may until its joins have named too many rooms that do not exist (see
// joinMissed).
func (s *Server) allowJoin(key netip.Prefix) bool {
	now := s.clock.Now()
	s.limitMu.Lock()
	defer s.limitMu.Unlock()
	return !s.misses.limited(key, now)
}

// joinMissed counts a join of the client address key that named no room.
func (s *Server) joinMissed(key netip.Prefix) {
	now := s.clock.Now()
	s.limitMu.Lock()
	defer s.limitMu.Unlock()
	s.misses.record(key, now)
}

// allowUndo reports whether the client address key may send an undo now,
// and counts it if so.
func (s *Server) allowUndo(key netip.Prefix) bool {
	now := s.clock.Now()
	s.limitMu.Lock()
	defer s.limitMu.Unlock()
	if s.undos.limited(key, now) {
		return false
	}
	s.undos.record(key, now)
	return true
}

// httpError is the JSON body of a refused upgrade, in the shape of the
// HTTP API's errors.
type httpError struct {
	OK   bool   `json:"ok"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func writeError(w http.ResponseWriter, status int, code string) {
	b, _ := json.Marshal(httpError{Code: code, Msg: http.StatusText(status)})
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	if status == http.StatusServiceUnavailable || status == http.StatusTooManyRequests {
		h.Set("Retry-After", "5")
	}
	w.WriteHeader(status)
	_, _ = w.Write(append(b, '\n'))
}
