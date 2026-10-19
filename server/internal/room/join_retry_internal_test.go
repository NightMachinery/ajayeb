package room

import (
	"context"
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestJoinExhaustedEvictionRetriesAreTemporary(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	// Hold the exact eviction window: lookup sees an actor whose done channel
	// has closed but whose registry entry has not yet been removed. Every
	// attempt must retry, without needing timing or a running goroutine.
	done := make(chan struct{})
	close(done)
	m.rooms["RACEXX"] = &Room{code: "RACEXX", done: done, inbox: make(chan any)}
	c := NewClient("player", "tab", "Player")
	if err := m.Join(context.Background(), c, "RACEXX"); !errors.Is(err, ErrStorage) {
		t.Fatalf("exhausted retries: %v", err)
	}
	frames := c.Out.Take()
	if len(frames) != 1 {
		t.Fatalf("frames: %+v", frames)
	}
	if got, ok := frames[0].(wire.Error); !ok || got.Code != wire.ErrorCodeBusy {
		t.Fatalf("must keep Home room retryable: %+v", frames[0])
	}
	if c.Joined() != "" {
		t.Fatal("failed join recorded membership")
	}
}
