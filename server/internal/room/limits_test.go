package room_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// A room whose file has grown past the store's read cap (planted, or
// corrupted) is not found: the join is refused cleanly, the room is not
// loaded, and the file is left for a look.
func TestOversizedRoomFileIsNotFound(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	h.evict(code, a)

	path := filepath.Join(h.st.Dir(), code+".json")
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	padded := append(bytes.Repeat([]byte(" "), store.MaxFileBytes), valid...)
	if err := os.WriteFile(path, padded, 0o600); err != nil {
		t.Fatal(err)
	}

	again := h.client("user-a", "tab-a2", "Ann")
	if e, err := again.joinErr(code); e.Code != wire.ErrorCodeRoomNotFound || !errors.Is(err, room.ErrRoomNotFound) {
		t.Fatalf("joining the oversized room: %v, %v", e, err)
	}
	if h.m.Loaded(code) {
		t.Fatal("the oversized room was loaded")
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, padded) {
		t.Fatalf("the oversized file changed: %v", err)
	}
}
