package room

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestPlatformV1AdmissionBeforeAnySeatClaim(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsupported-production-codec", true: "registered-test-kind"}[registered], func(t *testing.T) {
			var loads, checks, attaches, creates int
			kind := fixtureKind()
			kind.ID = "haftgane"
			kind.Codec.Game = kind.ID
			kind.Create = func(*Manager, *Client, wire.ClientMessage) (string, error) { creates++; return "", ErrInvalidSpec }
			kind.Check = func(*Manager, string, *Client, *string) error { checks++; return nil }
			kind.Load = func(m *Manager, code string) (*Handle, error) {
				loads++
				ctx, cancel := context.WithCancel(m.Context())
				return m.NewHandle(kind.ID, code, HandleConfig{Run: func() { <-ctx.Done() }, Cancel: cancel,
					Attach: func(context.Context, *Client, *string) error { attaches++; return nil }, Detach: func(*Client) {}, Deliver: func(context.Context, any) error { return nil }, Evict: func(context.Context) error { return ErrForbidden }})
			}
			entries := []Kind{DuelKind()}
			if registered {
				entries = append(entries, kind)
			}
			kinds, err := NewKinds(entries...)
			if err != nil {
				t.Fatal(err)
			}
			st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"schema":2,"game":"haftgane","code":"REGAAA","seed":{}}`)
			path := filepath.Join(st.Dir(), "REGAAA.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			m, err := NewManager(Config{Store: st, Kinds: kinds, KindOf: st.Game})
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			old := NewClient("old-user", "old-tab", "Old")
			refused := func() {
				t.Helper()
				if err := m.Join(context.Background(), old, "REGAAA"); !errors.Is(err, ErrProtocol) {
					t.Fatalf("v1 join: %v", err)
				}
				frames := old.Out.Take()
				if len(frames) != 1 || frames[0].(wire.Error).Code != wire.ErrorCodeProtocol || old.Joined() != "" {
					t.Fatalf("v1 refusal: %+v", frames)
				}
			}
			refused()
			if checks != 0 || loads != 0 || attaches != 0 || m.Len() != 0 {
				t.Fatal("cold v1 invite reached admission/load/seat claim")
			}
			if registered {
				v2 := NewClient("new-user", "new-tab", "New")
				v2.ProtocolVersion = 2
				if err := m.Join(context.Background(), v2, "REGAAA"); err != nil {
					t.Fatal(err)
				}
				refused()
				if loads != 1 || checks != 1 || attaches != 1 {
					t.Fatal("warm v1 invite reached attach")
				}
			}
			if _, err := m.createKind(context.Background(), old, kind.ID, wire.LobbyCreate{}); !errors.Is(err, ErrProtocol) || creates != 0 {
				t.Fatalf("v1 create invoked constructor: %v", err)
			}
			if after, err := os.ReadFile(path); err != nil || string(after) != string(raw) {
				t.Fatal("v1 refusal changed persisted bytes")
			}
		})
	}
}

func TestPlatformDispatchByActualKind(t *testing.T) {
	ctx := context.Background()
	var received []Inbound
	var detached int
	kind := fixtureKind()
	kind.ID = "haftgane"
	kind.Codec.Game = kind.ID
	kind.Load = func(m *Manager, code string) (*Handle, error) {
		life, cancel := context.WithCancel(m.Context())
		return m.NewHandle(kind.ID, code, HandleConfig{Run: func() { <-life.Done() }, Cancel: cancel, Attach: func(context.Context, *Client, *string) error { return nil }, Detach: func(*Client) { detached++ }, Deliver: func(_ context.Context, msg any) error {
			inbound, ok := msg.(Inbound)
			if !ok {
				t.Fatalf("untyped inbound: %T", msg)
			}
			received = append(received, inbound)
			return nil
		}, Evict: func(context.Context) error { return ErrForbidden }})
	}
	kinds, err := NewKinds(DuelKind(), kind)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), nil, kinds.Codecs())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st, Kinds: kinds, KindOf: func(string) (string, error) { return kind.ID, nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	c := NewClient("user", "tab", "Name")
	c.ProtocolVersion = 2
	if err := m.Join(ctx, c, "REGAAA"); err != nil {
		t.Fatal(err)
	}
	pick := wire.TablePick{T: "table-pick", Code: "REGAAA", Seq: 1, Point: wire.PointRef{ID: "sides", Revision: 0}, Key: "side:day", Confirm: false}
	if err := m.DispatchPlatform(ctx, c, pick); err != nil {
		t.Fatal(err)
	}
	if err := m.Resync(ctx, c, "REGAAA"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Rematch(ctx, c, "REGAAA"); err != nil {
		t.Fatal(err)
	}
	if err := m.Link(ctx, c, wire.NewLink("REGAAA", 0)); err != nil {
		t.Fatal(err)
	}
	if err := m.DispatchPlatform(ctx, c, wire.PlatformRelink{T: "relink", Code: "REGAAA", Seat: 6}); err != nil {
		t.Fatal(err)
	}
	want := []wire.ClientMessage{pick, wire.NewResync("REGAAA"), wire.NewRematch("REGAAA"), wire.NewLink("REGAAA", 0), wire.PlatformRelink{T: "relink", Code: "REGAAA", Seat: 6}}
	if len(received) != len(want) {
		t.Fatalf("received %d common/platform requests", len(received))
	}
	for i, msg := range received {
		if msg.Client != c || !reflect.DeepEqual(msg.Frame, want[i]) {
			t.Fatalf("lost authenticated sender/frame: %+v", msg)
		}
	}
	if err := m.DispatchPlatform(ctx, c, wire.LobbyStart{T: "lobby-start", Code: "REGAAA", Seq: 2}); !errors.Is(err, ErrInvalidSpec) {
		t.Fatal("wrong family reached actor")
	}
	if err := m.Act(ctx, c, wire.NewAct("REGAAA", 3, 0, "construct:baths", nil)); !errors.Is(err, ErrNotJoined) {
		t.Fatal("Duel act reached table actor")
	}
	if len(received) != len(want) {
		t.Fatal("wrong-family request delivered")
	}
	m.Leave(c, "REGAAA")
	if detached != 1 || c.Joined() != "" {
		t.Fatal("common leave bypassed actual kind")
	}
	if err := m.DispatchPlatform(ctx, c, pick); !errors.Is(err, ErrNotJoined) {
		t.Fatal("unjoined sender delivered")
	}
}

func TestPlatformProductionRegistryRemainsDuelOnly(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	c := NewClient("user", "tab", "Name")
	c.ProtocolVersion = 2
	if _, err := m.CreatePlatform(context.Background(), c, wire.LobbyCreate{T: "lobby-create", SeatCount: 3}); !errors.Is(err, ErrInvalidSpec) || m.Len() != 0 {
		t.Fatal("production lobby unexpectedly reachable")
	}
	code, err := m.Create(context.Background(), c, wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if err != nil {
		t.Fatal(err)
	}
	c.Out.Take()
	if err := m.DispatchPlatform(context.Background(), c, wire.TablePick{Code: code}); !errors.Is(err, ErrInvalidSpec) {
		t.Fatal("Duel accepted a table request")
	}
	if err := m.DispatchPlatform(context.Background(), c, wire.PlatformLink{Code: code, Seat: 6}); !errors.Is(err, ErrInvalidSpec) {
		t.Fatal("Duel accepted seat 6")
	}
	// An actor barrier proves malformed requests never indexed its seat array.
	if _, err := m.Snapshot(context.Background(), code); err != nil {
		t.Fatal(err)
	}
}
