package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// TestHintFieldsPersisted checks that the admin, the hint settings and the
// room log round trip, and that they are not part of the game: the hash is
// the same with and without them.
func TestHintFieldsPersisted(t *testing.T) {
	st := newStore(t)
	keys, _ := playActions(t, 1, engine.Options{}, 3)
	plain := store.New("HNTPAN", store.ModeOnline, engine.Options{}, 1, time.Now())
	plain.Actions = keys
	if err := st.Save(plain); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f := store.New("HNTSVD", store.ModeOnline, engine.Options{}, 1, time.Now())
	f.Actions = keys
	f.Admin = "u-admin"
	f.Hints = &view.HintSettings{Prices: true, Science: true, Advisor: true}
	f.RoomLog = []store.LogEntry{{Kind: "hints", At: 2, Seat: 1, On: []string{"advisor"}, Off: []string{"economy"}}}
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if f.Hash != plain.Hash {
		t.Fatalf("hash %s with hint settings, %s without: they must not be part of the game", f.Hash, plain.Hash)
	}
	got, _, err := st.Load("HNTSVD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Admin != "u-admin" || got.Hints == nil || *got.Hints != *f.Hints {
		t.Fatalf("Load: admin %q, hints %+v", got.Admin, got.Hints)
	}
	if len(got.RoomLog) != 1 {
		t.Fatalf("Load: room log %+v", got.RoomLog)
	}
	e := got.RoomLog[0]
	if e.Kind != "hints" || e.At != 2 || e.Seat != 1 || strings.Join(e.On, ",") != "advisor" || strings.Join(e.Off, ",") != "economy" {
		t.Fatalf("Load: room log entry %+v", e)
	}
}

// TestHintFieldsAbsentInOldFile checks that a file written before the hint
// fields existed loads, with no admin, no settings and no room log (package
// room fills in the defaults), and that a new file without them leaves the
// keys out.
func TestHintFieldsAbsentInOldFile(t *testing.T) {
	st := newStore(t)
	f := store.New("HNTXYD", store.ModeHotseat, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(st.Dir(), "HNTXYD.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, key := range []string{`"admin"`, `"hints"`, `"roomLog"`} {
		if strings.Contains(string(data), key) {
			t.Fatalf("a file without hint settings has the key %s:\n%s", key, data)
		}
	}
	got, _, err := st.Load("HNTXYD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Admin != "" || got.Hints != nil || len(got.RoomLog) != 0 {
		t.Fatalf("Load: admin %q, hints %+v, room log %+v", got.Admin, got.Hints, got.RoomLog)
	}
}
