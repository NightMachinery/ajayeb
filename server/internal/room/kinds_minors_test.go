package room

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestRoomKindIdlePublicationBeforeRegistration(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	f := store.New("REGAAA", store.ModeOnline, engine.Options{}, 1, time.Now())
	r := newRoom(m, f, engine.NewGame(1, engine.Options{}), nil)
	h := duelHandle(m, r) // Deliberately build before the first persist publishes idle.
	r.persist()
	if r.dirty {
		t.Fatal("initial persist failed")
	}
	if err := m.start(h); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Snapshot(context.Background(), f.Code); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	idle := h.evictable
	m.mu.Unlock()
	if !idle {
		t.Fatal("pre-registration idle publication was lost")
	}
	m.evict(h)
	<-r.done
	if m.Loaded(f.Code) {
		t.Fatal("initially idle room could not be evicted")
	}
}

func TestRoomKindUnmatchedSelectorIsLogged(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	m, err := NewManager(Config{Store: st, Logger: slog.New(slog.NewTextHandler(&log, nil)), KindOf: func(string) (string, error) { return "unknown-kind", nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.get("REGAAA", nil, nil); !errors.Is(err, ErrStorage) {
		t.Fatalf("unmatched kind: %v", err)
	}
	if !strings.Contains(log.String(), "room=REGAAA") || !strings.Contains(log.String(), "game=unknown-kind") {
		t.Fatalf("missing selector diagnostic: %s", log.String())
	}
}
