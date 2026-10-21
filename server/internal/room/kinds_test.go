package room

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type kindInspection struct{ code string }

func (f kindInspection) Retention() store.RetentionHeader {
	return store.RetentionHeader{Code: f.code, Status: store.StatusPlaying, Updated: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)}
}
func (f kindInspection) Check() (bool, error) { return false, nil }
func fixtureKind() Kind {
	return Kind{ID: "fixture", Codec: &store.Codec{Game: "fixture", Decode: func(data []byte) (store.Inspection, error) {
		var raw struct {
			Code string `json:"code"`
		}
		err := json.Unmarshal(data, &raw)
		return kindInspection{raw.Code}, err
	}}, Create: func(*Manager, *Client, wire.ClientMessage) (string, error) { return "", ErrInvalidSpec }, Check: func(*Manager, string, *Client, *string) error { return nil }, Load: func(*Manager, string) (*Handle, error) { return nil, ErrStorage }}
}

func TestRoomKindsRegistration(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kinds []Kind
	}{
		{"missing-duel", []Kind{fixtureKind()}},
		{"duplicate-duel", []Kind{DuelKind(), DuelKind()}},
		{"duplicate-game", []Kind{DuelKind(), fixtureKind(), fixtureKind()}},
		{"missing-create", []Kind{func() Kind { k := DuelKind(); k.Create = nil; return k }()}},
		{"missing-check", []Kind{func() Kind { k := DuelKind(); k.Check = nil; return k }()}},
		{"missing-load", []Kind{func() Kind { k := DuelKind(); k.Load = nil; return k }()}},
		{"replace-duel-codec", []Kind{func() Kind { k := DuelKind(); k.Codec = fixtureKind().Codec; return k }()}},
		{"missing-codec", []Kind{DuelKind(), func() Kind { k := fixtureKind(); k.Codec = nil; return k }()}},
		{"different-codec-id", []Kind{DuelKind(), func() Kind { k := fixtureKind(); k.Codec.Game = "another"; return k }()}},
		{"invalid-id", []Kind{DuelKind(), func() Kind { k := fixtureKind(); k.ID = "Invalid"; k.Codec.Game = k.ID; return k }()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewKinds(tc.kinds...); err == nil {
				t.Fatal("accepted invalid registration")
			}
		})
	}
}

func TestRoomKindsCodecBinding(t *testing.T) {
	kind := fixtureKind()
	var decoded atomic.Int32
	original := kind.Codec.Decode
	kind.Codec.Decode = func(data []byte) (store.Inspection, error) { decoded.Add(1); return original(data) }
	kinds, err := NewKinds(DuelKind(), kind)
	if err != nil {
		t.Fatal(err)
	}
	kind.Codec.Game = "mutated"
	kind.Codec.Decode = nil
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"seed":{},"schema":2,"game":"fixture","code":"REGAAA"}`)
	path := filepath.Join(st.Dir(), "REGAAA.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := st.Game("REGAAA"); err != nil || got != "fixture" || decoded.Load() != 0 {
		t.Fatalf("bounded selection: %q %v calls=%d", got, err, decoded.Load())
	}
	if report := store.CheckDataDirWithCodecs(filepath.Dir(st.Dir()), kinds.Codecs()); report.Playable != 1 || report.Errors != 0 || decoded.Load() != 1 {
		t.Fatalf("bound codec: %+v calls=%d", report, decoded.Load())
	}
	if report := store.CheckDataDir(filepath.Dir(st.Dir())); report.Errors != 1 {
		t.Fatalf("test codec leaked to production: %+v", report)
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != string(raw) {
		t.Fatalf("changed file: %v", err)
	}
	for _, raw := range []string{`{"schema":1,"game":"fixture"}`, `{"schema":2,"game":"duel"}`, `{"schema":2,"game":"unknown"}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := st.Game("REGAAA"); !errors.Is(err, store.ErrGame) {
			t.Fatalf("selected an unsupported schema/game tuple: %v", err)
		}
	}
	if decoded.Load() != 1 {
		t.Fatal("header selection called a game decoder")
	}

}

func TestRoomKindsDuelDispatch(t *testing.T) {
	ctx := context.Background()
	var creates, checks, loads, attaches, detaches, deliveries, evictions atomic.Int32
	kind := DuelKind()
	create, check, load := kind.Create, kind.Check, kind.Load
	kind.Create = func(m *Manager, c *Client, req wire.ClientMessage) (string, error) {
		creates.Add(1)
		return create(m, c, req)
	}
	kind.Check = func(m *Manager, code string, c *Client, claim *string) error {
		checks.Add(1)
		return check(m, code, c, claim)
	}
	kind.Load = func(m *Manager, code string) (*Handle, error) {
		loads.Add(1)
		h, err := load(m, code)
		if err != nil {
			return nil, err
		}
		attach, detach, deliver, evict := h.hooks.Attach, h.hooks.Detach, h.hooks.Deliver, h.hooks.Evict
		h.hooks.Attach = func(ctx context.Context, c *Client, claim *string) error {
			attaches.Add(1)
			return attach(ctx, c, claim)
		}
		h.hooks.Detach = func(c *Client) { detaches.Add(1); detach(c) }
		h.hooks.Deliver = func(ctx context.Context, msg any) error { deliveries.Add(1); return deliver(ctx, msg) }
		h.hooks.Evict = func(ctx context.Context) error { evictions.Add(1); return evict(ctx) }
		return h, nil
	}
	kinds, err := NewKinds(kind)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st, Kinds: kinds, KindOf: st.Game, MaxRooms: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	c := NewClient("owner", "first", "Player")
	code, err := m.Create(ctx, c, wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	if err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 1 || c.Joined() != code {
		t.Fatal("create factory was bypassed")
	}
	m.Disconnect(c)
	if _, err := m.Snapshot(ctx, code); err != nil {
		t.Fatal(err)
	}
	old, _ := m.lookup(code)
	m.evict(old)
	<-old.duel.done
	if m.Loaded(code) {
		t.Fatal("eviction left registry entry")
	}
	again := NewClient("owner", "second", "Player")
	if err := m.Join(ctx, again, code); err != nil {
		t.Fatal(err)
	}
	if checks.Load() != 1 || loads.Load() != 1 || attaches.Load() != 1 {
		t.Fatalf("cold dispatch: check=%d load=%d attach=%d", checks.Load(), loads.Load(), attaches.Load())
	}
	if err := m.Resync(ctx, again, code); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Snapshot(ctx, code); err != nil {
		t.Fatal(err)
	}
	if deliveries.Load() != 1 {
		t.Fatal("delivery adapter was bypassed")
	}
	m.Disconnect(again)
	if _, err := m.Snapshot(ctx, code); err != nil {
		t.Fatal(err)
	}
	if detaches.Load() != 1 {
		t.Fatal("detach adapter was bypassed")
	}
	other := NewClient("other", "third", "Other")
	if _, err := m.Create(ctx, other, wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil)); err != nil {
		t.Fatal(err)
	}
	if evictions.Load() != 1 || m.Loaded(code) || m.Len() != 1 {
		t.Fatal("shared capacity bookkeeping did not evict the idle Duel room")
	}
}

func TestRoomKindsAdmissionBeforeReserve(t *testing.T) {
	var loads atomic.Int32
	kind := fixtureKind()
	claim := "claim-sentinel"
	kind.Check = func(_ *Manager, _ string, c *Client, got *string) error {
		if c.UserID != "outsider" || got == nil || *got != claim {
			t.Error("identity/claim changed at seam")
		}
		return ErrForbidden
	}
	kind.Load = func(*Manager, string) (*Handle, error) { loads.Add(1); return nil, ErrStorage }
	kinds, err := NewKinds(DuelKind(), kind)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st, Kinds: kinds, KindOf: st.Game, MaxRooms: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	owner := NewClient("owner", "owner-tab", "Owner")
	active, err := m.Create(context.Background(), owner, wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	if err != nil {
		t.Fatal(err)
	}
	for _, game := range []string{"fixture", "unregistered"} {
		raw := []byte(`{"schema":2,"game":"` + game + `","code":"REGAAA","seed":{}}`)
		path := filepath.Join(st.Dir(), "REGAAA.json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		c := NewClient("outsider", "outsider-tab", "Outsider")
		c.ProtocolVersion = wire.PlatformProtocolVersion
		want := ErrForbidden
		if game == "unregistered" {
			want = ErrStorage
		}
		if err := m.JoinClaim(context.Background(), c, "REGAAA", &claim); !errors.Is(err, want) {
			t.Fatalf("cold refusal: %v want %v", err, want)
		}
		if loads.Load() != 0 || m.Len() != 1 || !m.Loaded(active) || m.pending != 0 {
			t.Fatal("refused join consumed a slot or loaded a game")
		}
		if after, err := os.ReadFile(path); err != nil || string(after) != string(raw) {
			t.Fatal("unsupported file changed")
		}
	}
}

type kindCodeStore struct {
	Storage
	codes []string
	next  atomic.Int32
}

func (s *kindCodeStore) NewCode() (string, error) {
	i := int(s.next.Add(1)) - 1
	return s.codes[i], nil
}

func TestRoomKindsGlobalCodesAndHandleLifetime(t *testing.T) {
	kinds, err := NewKinds(DuelKind(), fixtureKind())
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	source := &kindCodeStore{Storage: st, codes: []string{"REGAAA", "REGAAA", "REGAAB"}}
	m, err := NewManager(Config{Store: source, Kinds: kinds, KindOf: st.Game})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	code, err := m.CreateFromSpec(Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}}})
	if err != nil || code != "REGAAA" {
		t.Fatalf("Duel create: %s %v", code, err)
	}
	var first *Handle
	firstDone := make(chan struct{})
	build := func(code string) (*Handle, error) {
		m.mu.Lock()
		reserved := m.pending
		m.mu.Unlock()
		if reserved != 1 {
			t.Error("factory ran without reservation")
		}
		ctx, cancel := context.WithCancel(m.Context())
		hooks := HandleConfig{Run: func() { defer close(firstDone); <-ctx.Done() }, Cancel: cancel,
			Attach: func(context.Context, *Client, *string) error { return nil }, Detach: func(*Client) {}, Deliver: func(context.Context, any) error { return nil }, Evict: func(context.Context) error { return nil }}
		var err error
		first, err = m.NewHandle("fixture", code, hooks)
		return first, err
	}
	code, err = m.CreateHandle("fixture", build)
	if err != nil || code != "REGAAB" || source.next.Load() != 3 || m.Len() != 2 {
		t.Fatalf("global codes: %s %v calls=%d rooms=%d", code, err, source.next.Load(), m.Len())
	}
	m.Forget(first)
	replacementDone := make(chan struct{})
	ctx, cancel := context.WithCancel(m.Context())
	second, err := m.NewHandle("fixture", code, HandleConfig{Run: func() { defer close(replacementDone); <-ctx.Done() }, Cancel: cancel,
		Attach: func(context.Context, *Client, *string) error { return nil }, Detach: func(*Client) {}, Deliver: func(context.Context, any) error { return nil }, Evict: func(context.Context) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.start(second); err != nil {
		t.Fatal(err)
	}
	first.hooks.Cancel()
	<-firstDone
	m.Forget(first)
	m.PublishIdle(first, true, time.Now())
	if got, _ := m.lookup(code); got != second || got.evictable {
		t.Fatal("stale actor changed its replacement")
	}
	m.Close()
	select {
	case <-replacementDone:
	default:
		t.Fatal("Close returned before the registered actor stopped")
	}
	if m.Len() != 0 {
		t.Fatal("Close left actor registered")
	}
}

func TestRoomKindsForeignLoadUsesRegisteredHandle(t *testing.T) {
	var loads, attaches, detaches, evicts atomic.Int32
	var loaded *Handle
	kind := fixtureKind()
	kind.Load = func(m *Manager, code string) (*Handle, error) {
		loads.Add(1)
		ctx, cancel := context.WithCancel(m.Context())
		h, err := m.NewHandle("fixture", code, HandleConfig{
			Run: func() { <-ctx.Done() }, Cancel: cancel,
			Attach: func(_ context.Context, c *Client, claim *string) error {
				if c.UserID != "owner" || claim != nil {
					t.Error("attach identity changed")
				}
				attaches.Add(1)
				return nil
			},
			Detach: func(*Client) { detaches.Add(1) }, Deliver: func(_ context.Context, message any) error {
				inbound, ok := message.(Inbound)
				if !ok {
					t.Error("non-platform delivery to foreign actor")
				} else if _, ok := inbound.Frame.(wire.Resync); !ok {
					t.Error("unexpected common request")
				}
				return nil
			},
			Evict: func(context.Context) error { evicts.Add(1); m.Forget(loaded); cancel(); return nil },
		})
		loaded = h
		return h, err
	}
	kinds, err := NewKinds(DuelKind(), kind)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"schema":2,"game":"fixture","code":"REGAAA","seed":{}}`)
	path := filepath.Join(st.Dir(), "REGAAA.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st, Kinds: kinds, KindOf: st.Game, MaxRooms: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	a, b := NewClient("owner", "a", "A"), NewClient("owner", "b", "B")
	a.ProtocolVersion, b.ProtocolVersion = wire.PlatformProtocolVersion, wire.PlatformProtocolVersion
	for _, c := range []*Client{a, b} {
		if err := m.Join(context.Background(), c, "REGAAA"); err != nil {
			t.Fatal(err)
		}
	}
	if loads.Load() != 1 || attaches.Load() != 2 || m.Len() != 1 {
		t.Fatal("cold kind loader or warm handle bypassed")
	}
	if err := m.Resync(context.Background(), a, "REGAAA"); err != nil {
		t.Fatalf("common resync did not reach foreign kind: %v", err)
	}
	if _, err := m.Snapshot(context.Background(), "REGAAA"); !errors.Is(err, ErrInvalidSpec) {
		t.Fatalf("foreign state exposed as Duel snapshot: %v", err)
	}
	m.Disconnect(a)
	m.Disconnect(b)
	m.PublishIdle(loaded, true, time.Now())
	if _, err := m.Create(context.Background(), NewClient("new-owner", "c", "C"), wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil)); err != nil {
		t.Fatal(err)
	}
	if detaches.Load() != 2 || evicts.Load() != 1 || m.Loaded("REGAAA") || m.Len() != 1 {
		t.Fatal("foreign lifetime did not use shared bookkeeping")
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != string(raw) {
		t.Fatal("foreign fixture was rewritten")
	}
}
