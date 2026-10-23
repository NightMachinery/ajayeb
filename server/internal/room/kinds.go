package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Duel is the stable id of the existing schema-1 room kind.
const Duel = "duel"

// Kind binds the room's composition seams to its file codec. Create receives
// the validated creation frame for that kind, rather than a Duel-only payload.
// Check refuses a cold join before Manager reserves a room slot. Load restores
// the concrete actor but does not start it or register it with Manager.
type Kind struct {
	ID     string
	Codec  *store.Codec // nil only for the built-in schema-1 Duel codec
	Create func(*Manager, *Client, wire.ClientMessage) (string, error)
	Check  func(*Manager, string, *Client, *string) error
	Load   func(*Manager, string) (*Handle, error)
}

// Kinds is a closed, immutable composition-time table, including the store
// half of each registration. No request can register or replace an entry.
type Kinds struct {
	entries map[string]Kind
	codecs  *store.Codecs
}

// NewKinds rejects duplicate ids and incomplete or inconsistent registrations.
// The returned codec table is passed to both the store and read-only preflight.
func NewKinds(kinds ...Kind) (*Kinds, error) {
	r := &Kinds{entries: make(map[string]Kind)}
	var codecs []store.Codec
	for _, k := range kinds {
		if _, ok := r.entries[k.ID]; ok {
			return nil, fmt.Errorf("room: duplicate kind %q", k.ID)
		}
		if k.Create == nil || k.Check == nil || k.Load == nil {
			return nil, fmt.Errorf("room: incomplete kind %q", k.ID)
		}
		if k.ID == Duel {
			if k.Codec != nil {
				return nil, errors.New("room: Duel must use the existing schema-1 codec")
			}
		} else {
			if k.Codec == nil || k.Codec.Game != k.ID {
				return nil, fmt.Errorf("room: kind %q needs its matching codec", k.ID)
			}
			codecs = append(codecs, *k.Codec)
		}
		// Copy the codec descriptor: later mutations by the caller cannot change
		// either the factory table or its independently immutable store table.
		if k.Codec != nil {
			copy := *k.Codec
			k.Codec = &copy
		}
		r.entries[k.ID] = k
	}
	if _, ok := r.entries[Duel]; !ok {
		return nil, errors.New("room: schema-1 Duel registration is required")
	}
	var err error
	r.codecs, err = store.NewCodecs(codecs...)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Kinds) Codecs() *store.Codecs { return r.codecs }

// HandleConfig supplies only lifetime and delivery operations. It never
// exposes rules, views, actions or bot observations to Manager.
// Run must return after Cancel, including any work the actor owns. Cancel
// must be idempotent and safe before Run starts. An Evict hook that agrees
// to leave must call Manager.Forget(handle) before returning; a hook refusing
// eviction must leave it registered. Canceling Run alone is not enough, because
// Manager must see the freed slot before its next reservation attempt.
type HandleConfig struct {
	Run     func()
	Cancel  func()
	Attach  func(context.Context, *Client, *string) error
	Detach  func(*Client)
	Deliver func(context.Context, any) error
	Evict   func(context.Context) error
}

// Handle is Manager's opaque lifetime handle. Its bookkeeping is guarded by
// Manager.mu. The legacy Duel adapter also retains its concrete actor for the
// existing Duel-only requests and diagnostic Snapshot API.
type Handle struct {
	owner      *Manager
	kind       string
	code       string
	hooks      HandleConfig
	start      func()
	afterStart func()
	duel       *Room
	evictable  bool
	idleSince  time.Time
	evicting   bool
}

// NewHandle is the constructor seam for separately typed actors. It does not
// start anything. Manager starts the actor under its existing slot bookkeeping.
func (m *Manager) NewHandle(kind, code string, hooks HandleConfig) (*Handle, error) {
	if _, ok := m.cfg.Kinds.entries[kind]; !ok {
		return nil, fmt.Errorf("room: unknown kind %q", kind)
	}
	if store.ValidateCode(code) != nil || hooks.Run == nil || hooks.Cancel == nil || hooks.Attach == nil || hooks.Detach == nil || hooks.Deliver == nil || hooks.Evict == nil {
		return nil, errors.New("room: incomplete lifetime handle")
	}
	h := &Handle{owner: m, kind: kind, code: code, hooks: hooks}
	h.start = func() {
		m.wg.Add(1)
		go func() { defer m.wg.Done(); defer m.Forget(h); hooks.Run() }()
	}
	return h, nil
}

// Context is the lifetime parent of actors constructed by registered kinds.
func (m *Manager) Context() context.Context { return m.ctx }

// PublishIdle updates the same eviction bookkeeping for any registered actor.
func (m *Manager) PublishIdle(h *Handle, idle bool, since time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h.owner == m {
		h.evictable = idle
		h.idleSince = since
	}
}

// Forget removes only this exact actor, never a replacement under the same code.
func (m *Manager) Forget(h *Handle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h.owner == m && m.rooms[h.code] == h {
		delete(m.rooms, h.code)
	}
}

// CreateHandle allocates a globally unique code and a Manager slot before a
// registered kind builds and durably saves its actor. It uses the same bounded
// retries, cancellation and registration as Duel's original create path.
func (m *Manager) CreateHandle(kind string, build func(code string) (*Handle, error)) (string, error) {
	if _, ok := m.cfg.Kinds.entries[kind]; !ok || build == nil {
		return "", ErrInvalidSpec
	}
	if err := m.reserve(); err != nil {
		return "", err
	}
	defer m.unreserve()
	const attempts = 3
	for range attempts {
		code, err := m.cfg.Store.NewCode()
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrStorage, err)
		}
		if m.Loaded(code) {
			continue
		}
		h, err := build(code)
		if err != nil {
			return "", err
		}
		if h == nil || h.owner != m || h.kind != kind || h.code != code {
			if h != nil && h.hooks.Cancel != nil {
				h.hooks.Cancel()
			}
			return "", ErrInvalidSpec
		}
		if err := m.start(h); err != nil {
			h.hooks.Cancel()
			if errors.Is(err, errCodeTaken) {
				continue
			}
			return "", err
		}
		if h.afterStart != nil {
			h.afterStart()
		}
		return code, nil
	}
	return "", fmt.Errorf("%w: no free room code", ErrStorage)
}

// Swap hands old's code to next, a handle of another registered kind for
// the same code: the shared lobby's Start, which writes the started game's
// record in place of the lobby's and passes its connections on. Swap
// registers next in old's place and starts it. old's actor must stop
// afterwards; its Forget is then a no-op, since next holds the code. The
// connections stay joined to the code, so their later requests and
// detaches reach next; a detach already delivered to old is old's to
// forward.
func (m *Manager) Swap(old, next *Handle) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch {
	case m.closed:
		return ErrClosed
	case old == nil || next == nil || old.owner != m || next.owner != m || next.code != old.code ||
		m.rooms[old.code] != old || next.duel != nil || old == next:
		return ErrInvalidSpec
	}
	if _, ok := m.cfg.Kinds.entries[next.kind]; !ok {
		return ErrInvalidSpec
	}
	m.rooms[old.code] = next
	next.start()
	return nil
}

// Claims returns the device-link claims of Config.Claims for registered
// kinds' actors, or nil when device links are off.
func (m *Manager) Claims() Claims { return m.cfg.Claims }

// post delivers an already decoded request to the concrete actor's inbox.
func (h *Handle) post(ctx context.Context, msg any) error { return h.hooks.Deliver(ctx, msg) }
