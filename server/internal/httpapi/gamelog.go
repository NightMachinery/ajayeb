package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// This file serves the replay download, GET /api/rooms/{code}/log: the game
// log (ajayeb-log/1, see docs/protocol.md "The game log") of a room whose
// game has ended. The log holds the seed, which determines every hidden
// card, so it is served only when the end of the game has revealed all of
// them anyway: never while the game runs, and never for a game whose
// hideAtEnd option keeps them hidden after the end. Anyone who knows the
// room code may watch the room, so the code is all a download needs.

// GameLogs reads a room's game log (a *store.Store). ReadLog returns the
// room's header whenever the file could be read, also when its replay
// failed, and an error wrapping store.ErrBadCode or fs.ErrNotExist when
// there is no such room.
type GameLogs interface {
	ReadLog(code string) (*store.File, *engine.Log, error)
}

// The log download limit, per client address: a bucket of logBurst
// requests that refills one request every logInterval. Every request
// counts, a miss included, so the endpoint is no faster a way to guess room
// codes than a join (30 misses an hour). logCapacity bounds the addresses
// remembered.
const (
	logBurst    = 10
	logInterval = 2 * time.Minute
	logCapacity = 4096
)

type logAPI struct {
	logs    GameLogs
	trusted []netip.Prefix
	now     func() time.Time

	mu      sync.Mutex
	buckets map[netip.Prefix]time.Time // the GCRA theoretical arrival time of each address
}

func newLogAPI(logs GameLogs, trusted []netip.Prefix, now func() time.Time) *logAPI {
	if now == nil {
		now = time.Now
	}
	return &logAPI{logs: logs, trusted: trusted, now: now, buckets: make(map[netip.Prefix]time.Time)}
}

// allow takes one request from key's bucket, the GCRA form of a token
// bucket (as in package ws): a request is allowed while the bucket's
// theoretical arrival time is at most (logBurst-1) intervals ahead of now.
// When logCapacity addresses are remembered, full buckets are forgotten
// first, and a newcomer is refused while every remembered bucket is still
// refilling, so a flood of addresses cannot reset anyone's limit.
func (a *logAPI) allow(key netip.Prefix) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := a.now()
	tat, known := a.buckets[key]
	if !known && len(a.buckets) >= logCapacity {
		for k, t := range a.buckets {
			if !t.After(now) {
				delete(a.buckets, k)
			}
		}
		if len(a.buckets) >= logCapacity {
			return false
		}
	}
	if tat.Before(now) {
		tat = now
	}
	if tat.Sub(now) > (logBurst-1)*logInterval {
		return false
	}
	a.buckets[key] = tat.Add(logInterval)
	return true
}

func (a *logAPI) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "method-not-allowed", "use GET")
		return
	}
	if !a.allow(limitKey(clientIP(r, a.trusted))) {
		w.Header().Set("Retry-After", "120")
		writeError(w, http.StatusTooManyRequests, "rate", "too many log requests from this address")
		return
	}
	code := strings.ToUpper(r.PathValue("code"))
	f, l, err := a.logs.ReadLog(code)
	if f == nil {
		if errors.Is(err, store.ErrBadCode) || errors.Is(err, fs.ErrNotExist) {
			writeError(w, http.StatusNotFound, "not-found", "no such room")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "the room could not be read")
		return
	}
	switch f.Status {
	case store.StatusFinished:
	case store.StatusArchived:
		writeError(w, http.StatusGone, "archived", "the room was archived and its log is not trusted")
		return
	default:
		writeError(w, http.StatusConflict, "game-running", "the game has not ended")
		return
	}
	if f.Options.HideAtEnd {
		writeError(w, http.StatusForbidden, "log-hidden", "this game keeps its hidden cards hidden after the end")
		return
	}
	if err != nil || l == nil {
		writeError(w, http.StatusInternalServerError, "internal", "the room's log does not replay")
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="ajayeb-`+f.Code+`.json"`)
	writeJSON(w, http.StatusOK, l)
}

// clientIP returns the address a request comes from, for the per-address
// limit, by the rule package ws follows (docs/protocol.md "The client
// address"): the TCP peer, unless the peer is a trusted proxy, in which case
// X-Forwarded-For is read from the right and the rightmost entry that is not
// itself a trusted proxy is the client. A malformed entry stops the walk.
func clientIP(r *http.Request, trusted []netip.Prefix) netip.Addr {
	ip := parseHostAddr(r.RemoteAddr)
	if !ip.IsValid() || !isTrusted(ip, trusted) {
		return ip
	}
	var hops []string
	for _, v := range r.Header.Values("X-Forwarded-For") {
		for hop := range strings.SplitSeq(v, ",") {
			hops = append(hops, strings.TrimSpace(hop))
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		h := parseHostAddr(hops[i])
		if !h.IsValid() {
			break
		}
		ip = h
		if !isTrusted(h, trusted) {
			break
		}
	}
	return ip
}

// limitKey counts an IPv4 address on its own and an IPv6 address by its
// /64, as package ws does. The zero address gives the zero prefix.
func limitKey(a netip.Addr) netip.Prefix {
	bits := 64
	if a.Is4() {
		bits = 32
	}
	p, err := a.Prefix(bits)
	if err != nil {
		return netip.Prefix{}
	}
	return p
}

// parseHostAddr parses an address with or without a port, unmapping
// IPv4-in-IPv6; the zero Addr when s is neither.
func parseHostAddr(s string) netip.Addr {
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap()
	}
	a, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(s, "["), "]"))
	if err != nil {
		return netip.Addr{}
	}
	return a.Unmap()
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}
