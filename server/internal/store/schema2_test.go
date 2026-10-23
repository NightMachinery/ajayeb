package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// gameRecord is a minimal schema-2 record for the store's game-agnostic
// operations: the common header plus a hash and a field this build does
// not know.
type gameRecord struct {
	Schema        int                 `json:"schema"`
	Game          string              `json:"game"`
	Code          string              `json:"code"`
	Status        store.Status        `json:"status"`
	Updated       time.Time           `json:"updated"`
	ArchivedAt    *time.Time          `json:"archivedAt,omitempty"`
	ArchiveReason store.ArchiveReason `json:"archiveReason,omitempty"`
	Hash          string              `json:"hash"`
}

func (r *gameRecord) Retention() store.RetentionHeader {
	return store.RetentionHeader{Code: r.Code, Status: r.Status, Updated: r.Updated, ArchivedAt: r.ArchivedAt}
}
func (r *gameRecord) Check() (bool, error) { return false, nil }

var gameNow = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func gameStore(t *testing.T) *store.Store {
	t.Helper()
	codecs, err := store.NewCodecs(store.Codec{Game: "fixture", Decode: func(raw []byte) (store.Inspection, error) {
		var r gameRecord
		err := json.Unmarshal(raw, &r)
		return &r, err
	}})
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), func() time.Time { return gameNow }, codecs)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func gameFile(code, status string) []byte {
	return []byte(`{"schema":2,"game":"fixture","code":"` + code + `","status":"` + status +
		`","updated":"2000-01-01T00:00:00Z","hash":"old","future":{"keep":[1,2,3]}}` + "\n")
}

func place(t *testing.T, st *store.Store, rel string, data []byte) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(st.Dir()), rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fields(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLoadGamePlayableWritesNothing(t *testing.T) {
	st := gameStore(t)
	path := place(t, st, "rooms/GAMAAA.json", gameFile("GAMAAA", "playing"))
	got, err := st.LoadGame("GAMAAA", "fixture", func([]byte) (store.Assessment, error) { return store.Assessment{}, nil })
	if err != nil {
		t.Fatal(err)
	}
	disk, _ := os.ReadFile(path)
	if !bytes.Equal(got, gameFile("GAMAAA", "playing")) || !bytes.Equal(disk, got) {
		t.Fatalf("a playable record changed:\n%s", disk)
	}
}

// TestLoadGameRefreshPreserves: a hash-only refresh rewrites the hash and
// keeps every other field, the unknown one and "updated" included.
func TestLoadGameRefreshPreserves(t *testing.T) {
	st := gameStore(t)
	path := place(t, st, "rooms/GAMAAB.json", gameFile("GAMAAB", "playing"))
	got, err := st.LoadGame("GAMAAB", "fixture", func([]byte) (store.Assessment, error) {
		return store.Assessment{Refresh: map[string]any{"hash": "new"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	disk, _ := os.ReadFile(path)
	if !bytes.Equal(disk, got) {
		t.Fatal("the returned bytes are not the refreshed file")
	}
	before, after := fields(t, gameFile("GAMAAB", "playing")), fields(t, disk)
	if string(after["hash"]) != `"new"` || len(after) != len(before) {
		t.Fatalf("refreshed %s", disk)
	}
	for k, v := range before {
		if k != "hash" && !bytes.Equal(bytes.TrimSpace(after[k]), v) && compact(t, after[k]) != compact(t, v) {
			t.Errorf("field %s changed from %s to %s", k, v, after[k])
		}
	}
}

func compact(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestLoadGameArchives: an incompatible record is marked and moved to the
// archive with its reason and time, keeping its unknown fields; the next
// load answers archived without asking the codec again.
func TestLoadGameArchives(t *testing.T) {
	st := gameStore(t)
	live := place(t, st, "rooms/GAMAAC.json", gameFile("GAMAAC", "playing"))
	_, err := st.LoadGame("GAMAAC", "fixture", func([]byte) (store.Assessment, error) {
		return store.Assessment{Archive: store.ArchiveRulesVersion, Cause: errors.New("rules 7")}, nil
	})
	var archived *store.ArchivedError
	if !errors.As(err, &archived) || archived.Reason != store.ArchiveRulesVersion {
		t.Fatalf("err %v", err)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatal("the live file is still in place")
	}
	disk, err := os.ReadFile(filepath.Join(st.ArchiveDir(), "GAMAAC.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := fields(t, disk)
	if string(m["status"]) != `"archived"` || string(m["archiveReason"]) != `"rules-version"` ||
		compact(t, m["future"]) != `{"keep":[1,2,3]}` || string(m["hash"]) != `"old"` {
		t.Fatalf("archived record %s", disk)
	}
	var at time.Time
	if err := json.Unmarshal(m["archivedAt"], &at); err != nil || !at.Equal(gameNow) {
		t.Fatalf("archivedAt %s", m["archivedAt"])
	}
	_, err = st.LoadGame("GAMAAC", "fixture", func([]byte) (store.Assessment, error) {
		t.Fatal("an archived record was assessed")
		return store.Assessment{}, nil
	})
	if !errors.As(err, &archived) || archived.Reason != store.ArchiveRulesVersion {
		t.Fatalf("second load: %v", err)
	}
	again, _ := os.ReadFile(filepath.Join(st.ArchiveDir(), "GAMAAC.json"))
	if !bytes.Equal(again, disk) {
		t.Fatal("loading an archive rewrote it")
	}
	if _, archivedFlag, err := st.PeekGame("GAMAAC", "fixture"); err != nil || !archivedFlag {
		t.Fatalf("PeekGame: %v %v", archivedFlag, err)
	}
}

// TestGameOperationsLeaveUnsupportedFilesUntouched: another game, an
// unregistered game, schema 1 and malformed bytes are refused by every
// operation and stay byte-identical.
func TestGameOperationsLeaveUnsupportedFilesUntouched(t *testing.T) {
	st := gameStore(t)
	cases := map[string][]byte{
		"GAMBAA": []byte(`{"schema":2,"game":"haftgane","rules":1,"code":"GAMBAA","status":"playing","updated":"2000-01-01T00:00:00Z","future":1}`),
		"GAMBAB": []byte(`{"schema":3,"game":"fixture","code":"GAMBAB","status":"playing","updated":"2000-01-01T00:00:00Z"}`),
		"GAMBAC": []byte(`{"schema":1,"code":"GAMBAC","status":"playing","updated":"2000-01-01T00:00:00Z"}`),
		"GAMBAD": []byte(`{"schema":2,"game":"fixture","code":"GAMBAD"`),
	}
	for code, data := range cases {
		path := place(t, st, "rooms/"+code+".json", data)
		if _, err := st.LoadGame(code, "fixture", func([]byte) (store.Assessment, error) {
			t.Fatalf("%s was assessed", code)
			return store.Assessment{}, nil
		}); err == nil {
			t.Errorf("%s loaded", code)
		}
		if _, _, err := st.PeekGame(code, "fixture"); err == nil {
			t.Errorf("%s peeked", code)
		}
		if code != "GAMBAD" {
			if err := st.SaveGame(code, "fixture", gameFile(code, "playing")); err == nil {
				t.Errorf("%s was overwritten", code)
			}
		}
		disk, _ := os.ReadFile(path)
		if !bytes.Equal(disk, data) {
			t.Errorf("%s changed: %s", code, disk)
		}
	}
}

func TestSaveGame(t *testing.T) {
	st := gameStore(t)
	if err := st.SaveGame("GAMCAA", "fixture", gameFile("GAMCAA", "playing")); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveGame("GAMCAA", "fixture", gameFile("GAMCAB", "playing")); err == nil {
		t.Error("a record of another code was saved")
	}
	if err := st.SaveGame("GAMCAA", "fixture", []byte(`{"schema":2,"game":"fixture"`)); err == nil {
		t.Error("undecodable bytes were saved")
	}
	if err := st.SaveGame("GAMCAA", "other", gameFile("GAMCAA", "playing")); err == nil {
		t.Error("a save under another game id succeeded")
	}
	place(t, st, "rooms-archive/GAMCAC.json", gameFile("GAMCAC", "archived"))
	if err := st.SaveGame("GAMCAC", "fixture", gameFile("GAMCAC", "playing")); !errors.Is(err, store.ErrArchived) {
		t.Errorf("saving over an archive: %v", err)
	}
	place(t, st, "rooms/GAMCAD.json", gameFile("GAMCAD", "archived"))
	if err := st.SaveGame("GAMCAD", "fixture", gameFile("GAMCAD", "playing")); !errors.Is(err, store.ErrArchived) {
		t.Errorf("saving over a live-folder archive: %v", err)
	}
	place(t, st, "rooms/GAMCAE.json", []byte("not json"))
	if err := st.SaveGame("GAMCAE", "fixture", gameFile("GAMCAE", "playing")); err != nil {
		t.Errorf("damaged bytes are overwritable: %v", err)
	}
	if err := st.SaveGame("GAMCAF", "fixture", gameFile("GAMCAF", "archived")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(st.ArchiveDir(), "GAMCAF.json")); err != nil {
		t.Errorf("an archived record stayed live: %v", err)
	}
	data, archived, err := st.PeekGame("GAMCAA", "fixture")
	if err != nil || archived || !bytes.Equal(data, gameFile("GAMCAA", "playing")) {
		t.Fatalf("PeekGame: %v %v %s", archived, err, data)
	}
}

// TestReplaceGame: Start writes a game in place of a live lobby record of
// the same code, and of nothing else: not over another game, an archive,
// damaged bytes or a missing file, nor a record of another code.
func TestReplaceGame(t *testing.T) {
	codecs, err := store.NewCodecs(
		store.Codec{Game: "fixture", Decode: func(raw []byte) (store.Inspection, error) {
			var r gameRecord
			err := json.Unmarshal(raw, &r)
			return &r, err
		}},
		store.Codec{Game: "waiting", Decode: func(raw []byte) (store.Inspection, error) {
			var r gameRecord
			err := json.Unmarshal(raw, &r)
			return &r, err
		}})
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(t.TempDir(), func() time.Time { return gameNow }, codecs)
	if err != nil {
		t.Fatal(err)
	}
	waiting := func(code, status string) []byte {
		return []byte(`{"schema":2,"game":"waiting","code":"` + code + `","status":"` + status + `","updated":"2000-01-01T00:00:00Z"}` + "\n")
	}
	if err := st.SaveGame("GAMRAA", "waiting", waiting("GAMRAA", "lobby")); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceGame("GAMRAA", "waiting", "fixture", gameFile("GAMRAB", "playing")); err == nil {
		t.Error("a record of another code replaced the lobby")
	}
	if err := st.ReplaceGame("GAMRAA", "fixture", "fixture", gameFile("GAMRAA", "playing")); err == nil {
		t.Error("a replacement of a game by itself succeeded")
	}
	if err := st.ReplaceGame("GAMRAA", "waiting", "fixture", gameFile("GAMRAA", "playing")); err != nil {
		t.Fatal(err)
	}
	if g, err := st.Game("GAMRAA"); err != nil || g != "fixture" {
		t.Fatalf("after the replacement: %q %v", g, err)
	}
	// The started game is not a lobby: a second Start cannot overwrite it.
	if err := st.ReplaceGame("GAMRAA", "waiting", "fixture", gameFile("GAMRAA", "playing")); !errors.Is(err, store.ErrGame) {
		t.Errorf("replacing a game: %v", err)
	}
	if err := st.ReplaceGame("GAMRAC", "waiting", "fixture", gameFile("GAMRAC", "playing")); err == nil {
		t.Error("a replacement with nothing in place succeeded")
	}
	place(t, st, "rooms/GAMRAD.json", []byte("not json"))
	if err := st.ReplaceGame("GAMRAD", "waiting", "fixture", gameFile("GAMRAD", "playing")); err == nil {
		t.Error("damaged bytes were replaced")
	}
	place(t, st, "rooms/GAMRAE.json", waiting("GAMRAE", "archived"))
	if err := st.ReplaceGame("GAMRAE", "waiting", "fixture", gameFile("GAMRAE", "playing")); !errors.Is(err, store.ErrArchived) {
		t.Errorf("replacing a live-folder archive: %v", err)
	}
	place(t, st, "rooms-archive/GAMRAF.json", waiting("GAMRAF", "archived"))
	if err := st.ReplaceGame("GAMRAF", "waiting", "fixture", gameFile("GAMRAF", "playing")); !errors.Is(err, store.ErrArchived) {
		t.Errorf("replacing an archive: %v", err)
	}
}
