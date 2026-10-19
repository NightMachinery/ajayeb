package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

func TestRefreshPreservesRoomMetadata(t *testing.T) {
	st := newStore(t)
	f := store.New("METADT", store.ModeHotseat, engine.Options{}, 27, time.Now().Add(-72*time.Hour).UTC().Round(0))
	f.Actions, _ = playActions(t, f.Seed, f.Options, 9)
	f.Version = 21
	f.Status = store.StatusPlaying
	f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "u-owner", Name: "Player", Epoch: 4, NameNo: 2}, {Kind: store.SeatKindHuman, UserID: "u-owner"}}
	f.LastSeq["u-owner:tab"] = 18
	f.Admin, f.Rematch, f.Tutorial = "u-owner", "REMATC", "first-game"
	f.NoUndo = true
	f.Hints = &view.HintSettings{Prices: true, Advisor: true}
	f.RoomLog = []store.LogEntry{{Kind: "hints", At: 3, On: []string{"prices"}, Off: []string{}}}
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	want := *f
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	rewriteField(t, st, f.Code, func(raw map[string]any) {
		raw["hash"] = "previous-layout"
		raw["additiveMetadata"] = map[string]any{"keep": true}
	})
	got, _, err := st.Load(f.Code)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(logs.Bytes(), []byte("oldHash=previous-layout")) || !bytes.Contains(logs.Bytes(), []byte("newHash="+f.Hash)) || !bytes.Contains(logs.Bytes(), []byte("level=WARN")) {
		t.Fatalf("missing hash refresh warning: %s", &logs)
	}
	rawBytes, err := os.ReadFile(filepath.Join(st.Dir(), f.Code+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(rawBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	if string(metadata["additiveMetadata"]) == "" {
		t.Fatal("lost additive room metadata")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("metadata changed:\ngot %+v\nwant %+v", got, want)
	}
}

func TestArchiveReasonDurableAndBounded(t *testing.T) {
	for _, key := range []string{"construct:bogus-card-id", "starter:01", "construct:baths", "resign:2", "resign:00"} {
		t.Run(key, func(t *testing.T) {
			st := newStore(t)
			f := store.New("ARCXYZ", store.ModeOnline, engine.Options{}, 1, time.Now())
			if err := st.Save(f); err != nil {
				t.Fatal(err)
			}
			rewriteField(t, st, f.Code, func(raw map[string]any) { raw["actions"] = []string{key} })
			got, _, err := st.Load(f.Code)
			var archived *store.ArchivedError
			if !errors.Is(err, store.ErrArchived) || !errors.As(err, &archived) || archived.Reason != store.ArchiveReplayFailed || got.ArchivedAt == nil {
				t.Fatalf("archive: %+v %v", got, err)
			}
			path := filepath.Join(st.ArchiveDir(), f.Code+".json")
			before, _ := os.ReadFile(path)
			again, _, err := st.Load(f.Code)
			after, _ := os.ReadFile(path)
			if !errors.Is(err, store.ErrArchived) || again.ArchiveReason != got.ArchiveReason || !bytes.Equal(before, after) {
				t.Fatal("archive changed on repeated load")
			}
		})
	}
	if store.SafeArchiveReason("private raw error") != store.ArchiveUnavailable {
		t.Fatal("unsafe reason")
	}
}

func TestExistingArchivedRoomNeverReopens(t *testing.T) {
	st := newStore(t)
	f := store.New("PRVARC", store.ModeOnline, engine.Options{}, 2, time.Now())
	f.Status = store.StatusArchived // a clean replay, with no legacy reason
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(st.ArchiveDir(), f.Code+".json")
	before, _ := os.ReadFile(path)
	got, _, err := st.Load(f.Code)
	after, _ := os.ReadFile(path)
	if !errors.Is(err, store.ErrArchived) || got.Status != store.StatusArchived || !bytes.Equal(before, after) {
		t.Fatalf("legacy archive reopened: %+v %v", got, err)
	}
}

func TestRefreshAndArchiveWriteFailuresAreNotSuccess(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission failure requires unprivileged user")
	}
	for _, field := range []string{"hash", "rules"} {
		t.Run(field, func(t *testing.T) {
			st := newStore(t)
			f := store.New("WRFAYX", store.ModeOnline, engine.Options{}, 2, time.Now())
			if err := st.Save(f); err != nil {
				t.Fatal(err)
			}
			rewriteField(t, st, f.Code, func(raw map[string]any) {
				if field == "hash" {
					raw[field] = "old"
				} else {
					raw[field] = engine.RulesVersion + 1
				}
			})
			before, _ := os.ReadFile(filepath.Join(st.Dir(), f.Code+".json"))
			if err := os.Chmod(st.Dir(), 0o500); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chmod(st.Dir(), 0o700) }()
			got, _, err := st.Load(f.Code)
			after, _ := os.ReadFile(filepath.Join(st.Dir(), f.Code+".json"))
			if !errors.Is(err, store.ErrRestoreWrite) || got != nil || errors.Is(err, store.ErrArchived) || !bytes.Equal(before, after) {
				t.Fatalf("write failure: %+v %v", got, err)
			}
		})
	}
}

func TestCheckDataDirReadOnlyCountsAndErrors(t *testing.T) {
	st := newStore(t)
	for _, code := range []string{"GDDXXX", "PRVHSH", "BADRUX", "BADKEY", "PRVARC", "SCHEMA"} {
		f := store.New(code, store.ModeOnline, engine.Options{}, 1, time.Now())
		f.Status = store.StatusPlaying
		if code == "PRVARC" {
			f.Status = store.StatusArchived
		}
		if err := st.Save(f); err != nil {
			t.Fatal(err)
		}
	}
	rewriteField(t, st, "PRVHSH", func(raw map[string]any) { raw["hash"] = "old" })
	rewriteField(t, st, "BADRUX", func(raw map[string]any) { raw["rules"] = engine.RulesVersion + 1 })
	rewriteField(t, st, "BADKEY", func(raw map[string]any) { raw["actions"] = []string{"starter:01"} })
	rewriteField(t, st, "SCHEMA", func(raw map[string]any) { raw["schema"] = store.Schema + 1; raw["newerField"] = "keep" })
	before := directoryBytes(t, st.Dir())
	r := store.CheckDataDir(filepath.Dir(st.Dir()))
	if r.Rooms != 6 || r.Playable != 2 || r.Playing != 4 || r.Archived != 1 || r.WouldArchive != 2 || r.LiveWouldArchive != 2 || r.RefreshableHashes != 1 || r.SchemaErrors != 1 || r.Errors != 1 {
		t.Fatalf("report: %+v", r)
	}
	if !reflect.DeepEqual(before, directoryBytes(t, st.Dir())) {
		t.Fatal("check wrote files")
	}
	if err := os.WriteFile(filepath.Join(st.Dir(), "BRKXXX.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := store.CheckDataDir(filepath.Dir(st.Dir())); r.Errors != 2 {
		t.Fatalf("malformed JSON: %+v", r)
	}
}

func directoryBytes(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	return out
}

func TestCheckFreshDataCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created")
	if r := store.CheckDataDir(path); r.Rooms != 0 || r.Errors != 0 {
		t.Fatalf("report: %+v", r)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created data dir: %v", err)
	}
	path = filepath.Join(t.TempDir(), "not-directory")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := store.CheckDataDir(path); r.Errors != 1 {
		t.Fatalf("failed open: %+v", r)
	}
}

// Keep the strict replay download contract independent of restore compatibility.
func TestDownloadHashRemainsStrict(t *testing.T) {
	st := newStore(t)
	f := store.New("STRYCT", store.ModeOnline, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, f.Code, func(raw map[string]any) { raw["hash"] = "old" })
	_, _, err := st.ReadLog(f.Code)
	if !errors.Is(err, store.ErrHashMismatch) {
		t.Fatalf("strict download: %v", err)
	}
	// JSON itself remains decodable for the candidate checker.
	b, _ := os.ReadFile(filepath.Join(st.Dir(), f.Code+".json"))
	if !json.Valid(b) {
		t.Fatal("invalid JSON")
	}
}

func TestCheckUnreadableDirectoryFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires unprivileged user")
	}
	st := newStore(t)
	if err := os.Chmod(st.Dir(), 0); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(st.Dir(), 0o700) }()
	if r := store.CheckDataDir(filepath.Dir(st.Dir())); r.Errors != 1 {
		t.Fatalf("unreadable directory: %+v", r)
	}
}

func TestCanonicalResignationRemainsReplayable(t *testing.T) {
	st := newStore(t)
	f := store.New("CANKEY", store.ModeOnline, engine.Options{}, 1, time.Now())
	f.Actions = []string{"resign:0"}
	if err := st.Save(f); err != nil {
		t.Fatalf("canonical room resignation: %v", err)
	}
	if _, s, err := st.Load(f.Code); err != nil || !s.Over() {
		t.Fatalf("canonical resignation replay: %v", err)
	}
}

func TestCheckIgnoresNonRoomEntriesLikeCodes(t *testing.T) {
	st := newStore(t)
	for _, name := range []string{"notes.json", "invalid-code.json", "README.txt"} {
		if err := os.WriteFile(filepath.Join(st.Dir(), name), []byte("not JSON"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"ABCDEF.json", "notes-dir.json"} {
		if err := os.Mkdir(filepath.Join(st.Dir(), name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	f := store.New("RDMXXX", store.ModeOnline, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	codes, err := st.Codes()
	if err != nil || len(codes) != 1 || codes[0] != f.Code {
		t.Fatalf("Codes: %v %v", codes, err)
	}
	r := store.CheckDataDir(filepath.Dir(st.Dir()))
	if r.Errors != 0 || r.Rooms != len(codes) || r.Playable != 1 {
		t.Fatalf("stray entries blocked check: %+v", r)
	}
}
