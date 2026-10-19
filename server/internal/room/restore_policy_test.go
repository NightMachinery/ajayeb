package room_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func rewritePolicyRoom(t *testing.T, st *store.Store, code string, edit func(map[string]any)) {
	t.Helper()
	path := filepath.Join(st.Dir(), code+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	edit(raw)
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOnDemandRestoreRefreshesHashAndAcceptsNextAction(t *testing.T) {
	h := newHarness(t)
	a := h.client("owner", "tab1", "Player")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Player", "Rival"}))
	a.act(st.View.Choices[0].Key, st.Version, seatPtr(*st.View.ToMove))
	h.snapshot(code)
	st = a.current()
	wantVersion := st.Version
	h.evict(code, a)
	before, err := h.st.PeekHeader(code)
	if err != nil {
		t.Fatal(err)
	}
	rewritePolicyRoom(t, h.st, code, func(raw map[string]any) { raw["hash"] = "old-derived-layout" })
	b := h.client("owner", "tab2", "Player")
	restored := b.join(code)
	if restored.Version != wantVersion || len(restored.View.Choices) == 0 {
		t.Fatal("room not playable after restore")
	}
	hdr, err := h.st.PeekHeader(code)
	if err != nil || hdr.Hash != before.Hash || !hdr.Updated.Equal(before.Updated) {
		t.Fatalf("refresh: %+v %v", hdr, err)
	}
	b.act(restored.View.Choices[0].Key, restored.Version, seatPtr(*restored.View.ToMove))
	h.snapshot(code)
	if next := b.current(); next.Version <= wantVersion {
		t.Fatal("next action not accepted")
	}
}

func TestArchivedRestoreHasDistinctTypedSafeError(t *testing.T) {
	for _, reason := range []store.ArchiveReason{store.ArchiveRulesVersion, store.ArchiveReplayFailed} {
		t.Run(string(reason), func(t *testing.T) {
			h := newHarness(t)
			f := store.New("ARCTST", store.ModeOnline, engine.Options{}, 1, roomEpoch)
			f.Status = store.StatusPlaying
			if err := h.st.Save(f); err != nil {
				t.Fatal(err)
			}
			rewritePolicyRoom(t, h.st, f.Code, func(raw map[string]any) {
				if reason == store.ArchiveRulesVersion {
					raw["rules"] = engine.RulesVersion + 1
				} else {
					raw["actions"] = []string{"private-invalid-key"}
				}
			})
			c := h.client("visitor", "tab", "Player")
			for range 2 {
				frame, err := c.joinErr(f.Code)
				if !errors.Is(err, room.ErrRoomArchived) || frame.Code != wire.ErrorCodeRoomArchived || frame.Msg != string(reason) {
					t.Fatalf("join: %+v %v", frame, err)
				}
			}
			if h.m.Loaded(f.Code) {
				t.Fatal("archived room loaded")
			}
			frame, err := c.joinErr("ABSENT")
			if !errors.Is(err, room.ErrRoomNotFound) || frame.Code != wire.ErrorCodeRoomNotFound {
				t.Fatalf("missing: %+v %v", frame, err)
			}
		})
	}
}

// restoreGate proves that both joins have passed their initial lookup while
// the first disk load is still pending. The second must then reuse the actor,
// never start another rewriting Load from the old snapshot.
type restoreGate struct {
	room.Storage
	loads        atomic.Int32
	headers      atomic.Int32
	firstLoad    chan struct{}
	secondHeader chan struct{}
	release      chan struct{}
}

func (g *restoreGate) PeekHeader(code string) (*store.File, error) {
	f, err := g.Storage.PeekHeader(code)
	if g.headers.Add(1) == 2 {
		close(g.secondHeader)
	}
	return f, err
}
func (g *restoreGate) Load(code string) (*store.File, engine.State, error) {
	if g.loads.Add(1) == 1 {
		close(g.firstLoad)
		<-g.release
	}
	return g.Storage.Load(code)
}

func TestConcurrentRestoreRefreshesOnlyOnceBeforeActorStarts(t *testing.T) {
	var gate *restoreGate
	h := newHarness(t, func(cfg *room.Config) {
		gate = &restoreGate{Storage: cfg.Store, firstLoad: make(chan struct{}), secondHeader: make(chan struct{}), release: make(chan struct{})}
		cfg.Store = gate
	})
	f := store.New("RACETX", store.ModeHotseat, engine.Options{}, 1, roomEpoch)
	f.Status = store.StatusPlaying
	f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "owner"}, {Kind: store.SeatKindHuman, UserID: "owner"}}
	if err := h.st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewritePolicyRoom(t, h.st, f.Code, func(raw map[string]any) { raw["hash"] = "old-layout" })
	a, b := h.client("owner", "one", "Player"), h.client("owner", "two", "Player")
	done := make(chan error, 2)
	ctx, cancel := context.WithTimeout(context.Background(), waitLimit)
	defer cancel()
	go func() { done <- h.m.Join(ctx, a.Client, f.Code) }()
	select {
	case <-gate.firstLoad:
	case <-time.After(waitLimit):
		t.Fatal("first load not reached")
	}
	go func() { done <- h.m.Join(ctx, b.Client, f.Code) }()
	select {
	case <-gate.secondHeader:
	case <-time.After(waitLimit):
		close(gate.release)
		t.Fatal("second header not reached")
	}
	close(gate.release)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if n := gate.loads.Load(); n != 1 {
		t.Fatalf("disk loaded %d times; late refresh could overwrite newer actions", n)
	}
}

func TestRestoreWriteFailuresAreRetryableAndKeepRoomFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("write permission failure requires unprivileged user")
	}
	for _, field := range []string{"hash", "rules"} {
		t.Run(field, func(t *testing.T) {
			h := newHarness(t)
			f := store.New("RETRYX", store.ModeOnline, engine.Options{}, 1, roomEpoch)
			f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "owner"}, {Kind: store.SeatKindOpen}}
			if err := h.st.Save(f); err != nil {
				t.Fatal(err)
			}
			rewritePolicyRoom(t, h.st, f.Code, func(raw map[string]any) {
				if field == "hash" {
					raw[field] = "old-layout"
				} else {
					raw[field] = engine.RulesVersion + 1
				}
			})
			path := filepath.Join(h.st.Dir(), f.Code+".json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(h.st.Dir(), 0o500); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chmod(h.st.Dir(), 0o700) }()
			c := h.client("owner", "tab", "Player")
			frame, err := c.joinErr(f.Code)
			if !errors.Is(err, room.ErrStorage) || frame.Code != wire.ErrorCodeBusy {
				t.Fatalf("storage refusal: %+v %v", frame, err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed write changed room file")
			}
			if h.m.Loaded(f.Code) {
				t.Fatal("failed write loaded room")
			}
			if err := os.Chmod(h.st.Dir(), 0o700); err != nil {
				t.Fatal(err)
			}
			if field == "hash" {
				restored := c.join(f.Code)
				if restored.Room.Status != wire.RoomStatusLobby {
					t.Fatalf("restored status %s", restored.Room.Status)
				}
			} else {
				frame, err := c.joinErr(f.Code)
				if !errors.Is(err, room.ErrRoomArchived) || frame.Code != wire.ErrorCodeRoomArchived {
					t.Fatalf("durable archive after retry: %+v %v", frame, err)
				}
			}
		})
	}
}

// A rewrite can report not-exist after its file was read (for example, when
// the parent directory disappears). That remains a temporary write failure.
type restoreWriteFailure struct{ room.Storage }

func (s restoreWriteFailure) Load(string) (*store.File, engine.State, error) {
	return nil, engine.State{}, errors.Join(store.ErrRestoreWrite, os.ErrNotExist)
}
func TestRestoreWriteMissingDirectoryIsNotMissingRoom(t *testing.T) {
	h := newHarness(t, func(cfg *room.Config) { cfg.Store = restoreWriteFailure{cfg.Store} })
	f := store.New("RETRYX", store.ModeOnline, engine.Options{}, 1, roomEpoch)
	if err := h.st.Save(f); err != nil {
		t.Fatal(err)
	}
	c := h.client("owner", "tab", "Player")
	frame, err := c.joinErr(f.Code)
	if !errors.Is(err, room.ErrStorage) || frame.Code != wire.ErrorCodeBusy {
		t.Fatalf("restore-write not-exist: %+v %v", frame, err)
	}
}
