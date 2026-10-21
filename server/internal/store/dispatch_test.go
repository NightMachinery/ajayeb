package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// This test deliberately uses only the pre-dispatch store API. Run the same
// source against the pinned pre-slice master to prove rollback preservation.
func TestRoomFileOldBinaryPreservesSchema2(t *testing.T) {
	st := newStore(t)
	fixtures := map[string]string{
		"rooms/FTXAAA.json":         `{"schema":2,"game":"haftgane","rules":1,"code":"FTXAAA","status":"playing","updated":"2000-01-01T00:00:00Z","future":{"pending":[1,2,3]}}`,
		"rooms/FTXAAB.json":         `{"schema":2,"game":"haftgane","rules":1,"code":"FTXAAB","status":"finished","updated":"2000-01-01T00:00:00Z","future":{"result":true}}`,
		"rooms/FTXAAC.json":         `{"schema":2,"game":"haftgane","rules":1,"code":"FTXAAC","status":"archived","updated":"2000-01-01T00:00:00Z","future":{"keep":true}}`,
		"rooms-archive/FTXAAD.json": `{"schema":2,"game":"haftgane","rules":1,"code":"FTXAAD","status":"archived","updated":"2000-01-01T00:00:00Z","future":{"keep":true}}`,
	}
	root := filepath.Dir(st.Dir())
	for name, raw := range fixtures {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(raw+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		code := strings.TrimSuffix(filepath.Base(name), ".json")
		if _, _, err := st.Load(code); !errors.Is(err, store.ErrSchema) {
			t.Fatalf("load %s: %v", code, err)
		}
		if _, err := st.PeekHeader(code); !errors.Is(err, store.ErrSchema) {
			t.Fatalf("peek %s: %v", code, err)
		}
	}
	reaper := store.NewReaper(st, func() time.Time { return time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC) })
	reaper.ArchiveMaxFiles, reaper.ArchiveMaxBytes = 0, 0
	if removed, err := reaper.Sweep(); !errors.Is(err, store.ErrSchema) || len(removed) != 0 {
		t.Fatalf("sweep: removed=%v err=%v", removed, err)
	}
	report := store.CheckDataDir(root)
	if report.Rooms != 4 || report.Errors != 4 || report.SchemaErrors != 4 || report.WouldArchive != 0 || report.Playable != 0 || report.Rules != 1 || report.Format != "ajayeb-room-check/1" {
		t.Fatalf("report: %+v", report)
	}
	for name, raw := range fixtures {
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || !bytes.Equal(got, []byte(raw+"\n")) {
			t.Fatalf("changed %s: %v", name, err)
		}
	}
	for _, dir := range []string{"rooms", "rooms-archive"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		want := 3
		if dir == "rooms-archive" {
			want = 1
		}
		if len(entries) != want {
			t.Fatalf("unexpected %s entries: %v", dir, entries)
		}
	}
}

func TestRoomFileDispatchBeforeDuelDecode(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{"unsupported-schema", `{"schema":99,"seed":{},"actions":false}`, "schema"},
		{"empty-production-schema2", `{"schema":2,"game":"haftgane","seed":{},"actions":false}`, "game"},
		{"unknown-schema1-game", `{"schema":1,"game":"foreign","seed":{},"actions":false}`, "game"},
		{"missing-game", `{"schema":2,"seed":{}}`, "game"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newStore(t)
			path := filepath.Join(st.Dir(), "HEADER.json")
			raw := []byte(tc.raw)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			want := store.ErrSchema
			if tc.want == "game" {
				want = store.ErrGame
			}
			if _, _, err := st.Load("HEADER"); !errors.Is(err, want) || strings.Contains(err.Error(), "cannot unmarshal") {
				t.Fatalf("must refuse %s before Duel decode: %v", tc.want, err)
			}
			if _, err := st.PeekHeader("HEADER"); !errors.Is(err, want) {
				t.Fatalf("peek: %v", err)
			}
			if _, _, err := st.ReadLog("HEADER"); !errors.Is(err, want) {
				t.Fatalf("log: %v", err)
			}
			if removed, err := store.NewReaper(st, nil).Sweep(); err == nil || len(removed) != 0 {
				t.Fatalf("sweep: %v %v", removed, err)
			}
			report := store.CheckDataDir(filepath.Dir(st.Dir()))
			if report.Errors != 1 || report.SchemaErrors != 1 || report.WouldArchive != 0 {
				t.Fatalf("report: %+v", report)
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(raw, got) {
				t.Fatalf("file changed: %v", err)
			}
		})
	}
}

// A fake owns its file shape and rules. None of these records can be decoded or
// replayed as Duel, and rules 900 must not be compared with Duel's version.
type dispatchFake struct {
	Code       string       `json:"code"`
	Status     store.Status `json:"status"`
	Updated    time.Time    `json:"updated"`
	ArchivedAt *time.Time   `json:"archivedAt"`
	Verdict    string       `json:"verdict"`
}

func (f *dispatchFake) Retention() store.RetentionHeader {
	return store.RetentionHeader{Code: f.Code, Status: f.Status, Updated: f.Updated, ArchivedAt: f.ArchivedAt}
}
func (f *dispatchFake) Check() (bool, error) {
	if f.Status == store.StatusArchived {
		panic("archived record replayed")
	}
	switch f.Verdict {
	case "rules", "replay":
		return false, fmt.Errorf("fake incompatible %s", f.Verdict)
	case "hash":
		return true, nil
	}
	return false, nil
}
func dispatchCodec(calls *int) store.Codec {
	return store.Codec{Game: "fixture", Decode: func(raw []byte) (store.Inspection, error) {
		(*calls)++
		var f dispatchFake
		err := json.Unmarshal(raw, &f)
		return &f, err
	}}
}

func TestRoomFileCodecRegistration(t *testing.T) {
	calls := 0
	codec := dispatchCodec(&calls)
	for _, codecs := range [][]store.Codec{
		{codec, codec},
		{{Game: "duel", Decode: codec.Decode}},
		{{Game: "Fixture", Decode: codec.Decode}},
		{{Game: strings.Repeat("x", 33), Decode: codec.Decode}},
		{{Game: "fixture"}},
	} {
		if _, err := store.NewCodecs(codecs...); err == nil {
			t.Fatalf("accepted invalid registration: %v", codecs)
		}
	}
	if calls != 0 {
		t.Fatal("registration decoded a file")
	}
	if _, err := store.NewCodecs(codec); err != nil {
		t.Fatal(err)
	}
}

func TestRoomFileCodecPreflightAndRetention(t *testing.T) {
	calls := 0
	codecs, err := store.NewCodecs(dispatchCodec(&calls))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	st, err := store.OpenWithCodecs(t.TempDir(), func() time.Time { return now }, codecs)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(st.Dir())
	fixtures := map[string]string{
		"rooms/FAKEAA.json":         `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAA","status":"playing","updated":"2026-10-21T00:00:00Z","seed":{}}`,
		"rooms/FAKEAB.json":         `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAB","status":"lobby","updated":"2026-10-21T00:00:00Z","verdict":"hash"}`,
		"rooms/FAKEAC.json":         `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAC","status":"playing","updated":"2026-10-21T00:00:00Z","verdict":"rules"}`,
		"rooms/FAKEAD.json":         `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAD","status":"finished","updated":"2000-01-01T00:00:00Z","verdict":"replay"}`,
		"rooms/FAKEAE.json":         `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAE","status":"archived","updated":"2026-10-21T00:00:00Z"}`,
		"rooms-archive/FAKEAF.json": `{"schema":2,"game":"fixture","rules":900,"code":"FAKEAF","status":"archived","updated":"2000-01-01T00:00:00Z"}`,
	}
	for name, raw := range fixtures {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	before := map[string]map[string]string{"rooms": directoryBytes(t, st.Dir()), "rooms-archive": directoryBytes(t, st.ArchiveDir())}
	report := store.CheckDataDirWithCodecs(root, codecs)
	if report.Rooms != 6 || report.Errors != 0 || report.SchemaErrors != 0 || report.Playable != 2 || report.RefreshableHashes != 1 || report.WouldArchive != 2 || report.LiveWouldArchive != 1 || report.Playing != 2 || report.Lobby != 1 || report.Finished != 1 || report.Archived != 2 || report.Rules != engine.RulesVersion || report.Format != "ajayeb-room-check/1" {
		t.Fatalf("report: %+v", report)
	}
	if calls != 6 {
		t.Fatalf("decode calls: %d", calls)
	}
	for dir, want := range before {
		if !reflect.DeepEqual(want, directoryBytes(t, filepath.Join(root, dir))) {
			t.Fatalf("preflight wrote %s", dir)
		}
	}
	if r := store.CheckDataDir(root); r.Errors != 6 || r.SchemaErrors != 6 {
		t.Fatalf("production registry leaked: %+v", r)
	}
	if calls != 6 {
		t.Fatal("production used test codec")
	}
	removed, err := store.NewReaper(st, func() time.Time { return now }).Sweep()
	if err != nil || !reflect.DeepEqual(removed, []string{"FAKEAD", "FAKEAF"}) {
		t.Fatalf("retention: %v %v", removed, err)
	}
	if got, err := os.ReadFile(filepath.Join(st.ArchiveDir(), "FAKEAE.json")); err != nil || string(got) != fixtures["rooms/FAKEAE.json"] {
		t.Fatalf("legacy migration changed bytes: %v", err)
	}
}

func TestRoomFileMalformedHeadersFailClosed(t *testing.T) {
	calls := 0
	codecs, err := store.NewCodecs(dispatchCodec(&calls))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`null`, `[]`, `{}`, `{"schema":null}`, `{"schema":1.5}`, `{"schema":21474836470}`,
		`{"schema":1,"Schema":2,"game":"fixture"}`, `{"schema":2,"game":"fixture","Game":"duel"}`,
		`{"schema":2,"game":null}`, `{"schema":2,"game":""}`, `{"schema":2,"game":42}`,
		`{"schema":2,"game":"` + strings.Repeat("x", 33) + `"}`, `{"schema":2,"game":"../fixture"}`,
		`{"schema":2,"game":"fixture"} {}`, `{"schema":2,"game":"fixture","status":"playing","code":"BADHDR"}`,
		`{"schema":2,"game":"fixture","status":"invalid","code":"BADHDR","updated":"2000-01-01T00:00:00Z"}`,
		`{"schema":2,"game":"fixture","status":"playing","code":"../bad","updated":"2000-01-01T00:00:00Z"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			st, err := store.OpenWithCodecs(t.TempDir(), nil, codecs)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(st.Dir(), "BADHDR.json")
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			reaper := store.NewReaper(st, nil)
			reaper.ArchiveMaxFiles, reaper.ArchiveMaxBytes = 0, 0
			if removed, err := reaper.Sweep(); err == nil || len(removed) != 0 {
				t.Fatalf("malformed file reaped: %v %v", removed, err)
			}
			if r := store.CheckDataDirWithCodecs(filepath.Dir(st.Dir()), codecs); r.Errors != 1 || r.WouldArchive != 0 {
				t.Fatalf("report: %+v", r)
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != raw {
				t.Fatalf("changed malformed file: %v", err)
			}
		})
	}
}

func TestRoomFileSaveRefusesUnsupportedTarget(t *testing.T) {
	st := newStore(t)
	f := store.New("NSAVEX", store.ModeHotseat, engine.Options{}, 1, time.Now())
	for _, raw := range []string{`{"schema":2,"game":"haftgane","seed":{}}`, `{"schema":1,"game":"unknown","seed":{}}`} {
		path := filepath.Join(st.Dir(), f.Code+".json")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if err := st.Save(f); !errors.Is(err, store.ErrSchema) {
			t.Fatalf("Save overwrote unsupported file: %v", err)
		}
		if got, err := os.ReadFile(path); err != nil || string(got) != raw {
			t.Fatalf("Save changed bytes: %v", err)
		}
	}
}

func TestRoomFileCodecSelection(t *testing.T) {
	for _, tc := range []struct {
		schema     int
		game, want string
	}{
		{2, "fixture", "fixture"}, {2, "another", "another"}, {2, "absent", ""}, {1, "fixture", ""}, {99, "fixture", ""},
	} {
		t.Run(fmt.Sprintf("%d-%s", tc.schema, tc.game), func(t *testing.T) {
			calls := []string{}
			var registrations []store.Codec
			// Put the discriminators after the opaque payload. The selected codec must
			// receive the original bytes, not a common record re-encoded by the store.
			raw := fmt.Sprintf(`{"seed":{},"code":"SETECT","status":"playing","updated":"2026-10-21T00:00:00Z","game":%q,"schema":%d}`, tc.game, tc.schema)
			for _, game := range []string{"fixture", "another"} {
				registrations = append(registrations, store.Codec{Game: game, Decode: func(data []byte) (store.Inspection, error) {
					calls = append(calls, game)
					if string(data) != raw {
						t.Fatal("codec received changed bytes")
					}
					var f dispatchFake
					err := json.Unmarshal(data, &f)
					return &f, err
				}})
			}
			codecs, err := store.NewCodecs(registrations...)
			if err != nil {
				t.Fatal(err)
			}
			st := newStore(t)
			if err := os.WriteFile(filepath.Join(st.Dir(), "SETECT.json"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			report := store.CheckDataDirWithCodecs(filepath.Dir(st.Dir()), codecs)
			if tc.want == "" {
				if len(calls) != 0 || report.Errors != 1 || report.SchemaErrors != 1 {
					t.Fatalf("unsupported codec ran: calls=%v report=%+v", calls, report)
				}
			} else if !reflect.DeepEqual(calls, []string{tc.want}) || report.Playable != 1 || report.Errors != 0 {
				t.Fatalf("selection: calls=%v report=%+v", calls, report)
			}
		})
	}
}
