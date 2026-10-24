package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The tests of the read cap (MaxFileBytes).

// A room file larger than MaxFileBytes is refused by every reader with
// ErrTooLarge, and left exactly as it is (Load does not archive it).
func TestOversizedFileIsRefused(t *testing.T) {
	st := newStore(t)
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	keys, _ := playActions(t, 3, engine.Options{}, 4)
	f := store.New("K7QX2M", store.ModeHotseat, engine.Options{}, 3, now)
	f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "u"}, {Kind: store.SeatKindHuman, UserID: "u"}}
	f.Actions = keys
	f.Status = store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(st.Dir(), "K7QX2M.json")
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Still valid JSON for the same room: only white space was added.
	padded := append(bytes.Repeat([]byte(" "), store.MaxFileBytes), valid...)
	if err := os.WriteFile(path, padded, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := st.Load("K7QX2M"); !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("Load = %v, want ErrTooLarge", err)
	}
	if _, err := st.PeekHeader("K7QX2M"); !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("PeekHeader = %v, want ErrTooLarge", err)
	}
	if f, _, err := st.ReadLog("K7QX2M"); f != nil || !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("ReadLog = %v, %v, want no file and ErrTooLarge", f, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, padded) {
		t.Error("the oversized file was rewritten")
	}

	// The same content at exactly the cap is still read.
	exact := append(bytes.Repeat([]byte(" "), store.MaxFileBytes-len(valid)), valid...)
	if err := os.WriteFile(path, exact, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Load("K7QX2M"); err != nil {
		t.Errorf("Load at exactly MaxFileBytes: %v", err)
	}
}

// The largest room file the server can write stays far below the cap: a
// whole game, the most client seqs a room remembers, a full room log and
// names of the longest length.
func TestLargestFileFitsTheCap(t *testing.T) {
	st := newStore(t)
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	keys, s := playActions(t, 5, engine.Options{}, engine.MaxGameActions)
	if !s.Over() {
		t.Fatalf("the game did not end in %d actions", engine.MaxGameActions)
	}
	f := store.New("K7QX2N", store.ModeOnline, engine.Options{}, 5, now)
	name := strings.Repeat("م", 64) // 64 two-byte characters
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: strings.Repeat("a", 32), Name: name, NameNo: 2, Epoch: 9},
		{Kind: store.SeatKindHuman, UserID: strings.Repeat("b", 32), Name: name, NameNo: 3, Epoch: 9},
	}
	f.Actions = keys
	f.Status = store.StatusFinished
	plain := sizeAfterSave(t, st, f)
	t.Logf("a finished game's file: %d bytes", plain)

	for i := range 32 {
		f.LastSeq[fmt.Sprintf("%s:%s%02d", strings.Repeat("c", 32), strings.Repeat("d", 62), i)] = 1 << 30
	}
	f.Admin = strings.Repeat("a", 32)
	f.Hints = &view.HintSettings{}
	families := make([]string, 16)
	for i := range families {
		families[i] = fmt.Sprintf("family-%02d-%s", i, strings.Repeat("x", 12))
	}
	for i := range 64 {
		target := 1
		f.RoomLog = append(f.RoomLog, store.LogEntry{Kind: "hints", At: i, Seat: 0, On: families, Off: families, Target: &target})
	}
	f.Rematch = "K7QX2P"
	full := sizeAfterSave(t, st, f)
	t.Logf("the largest file: %d bytes, the cap %d", full, store.MaxFileBytes)
	if full > store.MaxFileBytes/4 {
		t.Errorf("the largest file is %d bytes, more than a quarter of the cap %d", full, store.MaxFileBytes)
	}
	if _, _, err := st.Load("K7QX2N"); err != nil {
		t.Errorf("Load of the largest file: %v", err)
	}
}

func sizeAfterSave(t *testing.T, st *store.Store, f *store.File) int {
	t.Helper()
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(st.Dir(), f.Code+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return int(info.Size())
}

// The store never writes a room file it would refuse to read: SaveGame,
// CreateGame and ReplaceGame refuse a record larger than MaxFileBytes with
// ErrTooLarge and leave the file as it was. Before, such a write landed,
// and the room it belonged to could no longer be saved over or loaded
// (M16.haftgane-security-review).
func TestOversizedRecordIsNotWritten(t *testing.T) {
	st := gameStore(t)
	big := func(code string) []byte {
		return []byte(`{"schema":2,"game":"fixture","code":"` + code + `","status":"playing","updated":"2000-01-01T00:00:00Z","hash":"old","pad":"` +
			strings.Repeat("x", store.MaxFileBytes) + `"}` + "\n")
	}
	if err := st.SaveGame("GAMBAA", "fixture", gameFile("GAMBAA", "playing")); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveGame("GAMBAA", "fixture", big("GAMBAA")); !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("SaveGame of an oversized record: %v", err)
	}
	if data, _, err := st.PeekGame("GAMBAA", "fixture"); err != nil || !bytes.Equal(data, gameFile("GAMBAA", "playing")) {
		t.Errorf("the record in place after a refused save: %v %.80s", err, data)
	}
	if _, err := st.CreateGame("GAMBAB", "fixture", big("GAMBAB")); !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("CreateGame of an oversized record: %v", err)
	}
	if _, _, err := st.PeekGame("GAMBAB", "fixture"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused create left a file: %v", err)
	}
	if err := st.ReplaceGame("GAMBAA", "other", "fixture", big("GAMBAA")); !errors.Is(err, store.ErrTooLarge) {
		t.Errorf("ReplaceGame of an oversized record: %v", err)
	}
	entries, err := os.ReadDir(st.Dir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("a refused write left %s", e.Name())
		}
	}
}

// TestLoadTimeWritesKeepTheCap: a load's hash refresh or archive mark that
// would take a record over MaxFileBytes is not written. LoadGame answers an
// error wrapping ErrRestoreWrite and ErrTooLarge, the record stays as it
// was, byte for byte and readable, and no temporary file is left. Before,
// a record exactly at the cap came out a little larger, and the store then
// refused to read it (M16.haftgane-security-review, from its re-check).
func TestLoadTimeWritesKeepTheCap(t *testing.T) {
	for _, archive := range []bool{false, true} {
		t.Run(map[bool]string{false: "refresh", true: "archive"}[archive], func(t *testing.T) {
			st := gameStore(t)
			var m map[string]any
			if err := json.Unmarshal(gameFile("CAPAAA", "playing"), &m); err != nil {
				t.Fatal(err)
			}
			m["pad"] = ""
			raw, _ := json.Marshal(m)
			m["pad"] = strings.Repeat("x", store.MaxFileBytes-len(raw))
			raw, _ = json.Marshal(m)
			if len(raw) != store.MaxFileBytes {
				t.Fatalf("the record takes %d bytes, want the cap", len(raw))
			}
			if err := st.SaveGame("CAPAAA", "fixture", raw); err != nil {
				t.Fatal(err)
			}
			_, err := st.LoadGame("CAPAAA", "fixture", func([]byte) (store.Assessment, error) {
				if archive {
					return store.Assessment{Archive: store.ArchiveReplayFailed}, nil
				}
				return store.Assessment{Refresh: map[string]any{"hash": strings.Repeat("a", 64)}}, nil
			})
			if !errors.Is(err, store.ErrRestoreWrite) || !errors.Is(err, store.ErrTooLarge) {
				t.Fatalf("LoadGame: %v, want ErrRestoreWrite and ErrTooLarge", err)
			}
			got, _, err := st.PeekGame("CAPAAA", "fixture")
			if err != nil || !bytes.Equal(got, raw) {
				t.Fatalf("the record after the refused write: %d bytes, %v", len(got), err)
			}
			entries, err := os.ReadDir(st.Dir())
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".tmp") {
					t.Fatalf("a temporary file was left: %s", e.Name())
				}
			}
		})
	}
}
