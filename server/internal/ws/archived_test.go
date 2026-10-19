package ws

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestArchivedJoinMappingAndGuessLimit(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := store.New("ARCWSS", store.ModeOnline, engine.Options{}, 1, wsEpoch)
	f.Status = store.StatusArchived
	f.ArchiveReason = "private arbitrary details"
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rooms, err := room.NewManager(room.Config{Store: st})
	if err != nil {
		t.Fatal(err)
	}
	defer rooms.Close()
	h := newHarness(t, func(cfg *Config) { cfg.Rooms = rooms; cfg.Limits.JoinMissesPerIP = 2 })
	c := h.client()
	for range 2 {
		c.send(wire.NewJoin(f.Code))
		frame := c.expect("error")
		if frame.Code != string(wire.ErrorCodeRoomArchived) {
			t.Fatalf("archive code: %s", frame.raw)
		}
		if frame.Msg != string(store.ArchiveUnavailable) {
			t.Fatalf("unsafe reason: %s", frame.raw)
		}
	}
	c.send(wire.NewJoin(f.Code))
	c.expectReject(0, wire.RejectCodeRate)
}
