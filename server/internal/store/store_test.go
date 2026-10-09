package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// playActions plays up to n plies of seed from a plain deal, always taking
// the first legal action LegalActions offers (a deterministic, if not very
// clever, policy), and returns the keys applied plus the final state. It
// stops early if the game ends first.
func playActions(t *testing.T, seed uint64, opts engine.Options, n int) ([]string, engine.State) {
	t.Helper()
	s := engine.NewGame(seed, opts)
	keys := make([]string, 0, n)
	for range n {
		if s.Over() {
			break
		}
		legal := engine.LegalActions(&s, nil)
		if len(legal) == 0 {
			t.Fatalf("seed %d: no legal actions in a non-final state (phase %s)", seed, s.Phase)
		}
		a := legal[0]
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("seed %d: legal action %s: %v", seed, a, err)
		}
		keys = append(keys, a.Key())
	}
	return keys, s
}

func TestGenerateCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		code, err := store.GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
		if err := store.ValidateCode(code); err != nil {
			t.Fatalf("GenerateCode produced %q, which ValidateCode rejects: %v", code, err)
		}
		if code != strings.ToUpper(code) {
			t.Fatalf("GenerateCode produced %q, not all upper case", code)
		}
		seen[code] = true
	}
	if len(seen) < 190 {
		t.Fatalf("200 codes collapsed to only %d distinct values", len(seen))
	}
}

func TestValidateCodeRejects(t *testing.T) {
	cases := []string{
		"",
		"ABCDE",    // too short
		"ABCDEFG",  // too long
		"abcdef",   // lower case
		"AB CDEF",  // space, and too long
		"../../AB", // path traversal shape
		"AB/CDEF",
		"AB.json",
		"A0CDEF", // '0' is excluded from the alphabet
		"A1CDEF", // '1' is excluded
		"AICDEF", // 'I' is excluded
		"ALCDEF", // 'L' is excluded
		"AOCDEF", // 'O' is excluded
	}
	for _, code := range cases {
		if err := store.ValidateCode(code); err == nil {
			t.Errorf("ValidateCode(%q): want error, got nil", code)
		}
	}
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return st
}

func TestSaveLoadRoundTrip(t *testing.T) {
	st := newStore(t)
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)

	keys, want := playActions(t, 1, engine.Options{}, 6)
	f := store.New("K7QX2M", store.ModeHotseat, engine.Options{}, 1, now)
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "deadbeef", Name: "Sara"},
		{Kind: store.SeatKindHuman, UserID: "deadbeef", Name: "Sara"},
	}
	f.Actions = keys
	f.Status = store.StatusPlaying

	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if f.Hash == "" {
		t.Fatalf("Save left Hash empty")
	}

	got, s, err := st.Load("K7QX2M")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Status != store.StatusPlaying {
		t.Fatalf("Load status %q, want %q", got.Status, store.StatusPlaying)
	}
	if got.Hash != f.Hash {
		t.Fatalf("Load hash %q, Save computed %q", got.Hash, f.Hash)
	}
	if engine.Hash(&s) != engine.Hash(&want) {
		t.Fatalf("Load replayed a different state than the one actually played")
	}

	// No stray temp files.
	entries, err := os.ReadDir(st.Dir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "K7QX2M.json" {
			t.Errorf("unexpected entry in the rooms dir: %s", e.Name())
		}
	}
}

func TestSaveRejectsBadCode(t *testing.T) {
	st := newStore(t)
	f := store.New("bad", store.ModeHotseat, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err == nil {
		t.Fatalf("Save with a malformed code: want error, got nil")
	}
	entries, err := os.ReadDir(st.Dir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Save with a bad code touched the filesystem: %v", entries)
	}
}

func TestSaveRejectsBrokenActionLog(t *testing.T) {
	st := newStore(t)
	f := store.New("BADKEY", store.ModeHotseat, engine.Options{}, 1, time.Now())
	f.Actions = []string{"construct:not-a-real-card"}
	if err := st.Save(f); err == nil {
		t.Fatalf("Save with an unparseable action key: want error, got nil")
	}
	if exists, _ := st.Exists("BADKEY"); exists {
		t.Fatalf("Save left a file behind after a replay failure")
	}
}

func TestLoadMissingRoom(t *testing.T) {
	st := newStore(t)
	if _, _, err := st.Load("ZZZZZZ"); err == nil {
		t.Fatalf("Load of a room that was never saved: want error, got nil")
	}
}

func TestLoadArchivesRulesVersionMismatch(t *testing.T) {
	st := newStore(t)
	f := store.New("RUFESV", store.ModeAI, engine.Options{}, 1, time.Now())
	f.Status = store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Tamper with the file on disk as if it had been written by an older
	// (or newer) engine.
	rewriteField(t, st, "RUFESV", func(raw map[string]any) {
		raw["rules"] = float64(engine.RulesVersion + 1)
	})

	got, _, err := st.Load("RUFESV")
	if err == nil {
		t.Fatalf("Load of a rules-version mismatch: want error, got nil")
	}
	if got == nil || got.Status != store.StatusArchived {
		t.Fatalf("Load did not archive the room: got %+v", got)
	}

	// The file is kept, archived, not deleted.
	again, err := st.PeekHeader("RUFESV")
	if err != nil {
		t.Fatalf("PeekHeader after archiving: %v", err)
	}
	if again.Status != store.StatusArchived {
		t.Fatalf("the file on disk is not archived: status %q", again.Status)
	}
}

func TestLoadArchivesBrokenReplay(t *testing.T) {
	st := newStore(t)
	f := store.New("BRAKEN", store.ModeAI, engine.Options{}, 1, time.Now())
	f.Status = store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rewriteField(t, st, "BRAKEN", func(raw map[string]any) {
		raw["actions"] = []any{"construct:bogus-card-id"}
	})

	_, _, err := st.Load("BRAKEN")
	if err == nil {
		t.Fatalf("Load of a file with an unparseable action: want error, got nil")
	}
	again, err := st.PeekHeader("BRAKEN")
	if err != nil {
		t.Fatalf("PeekHeader: %v", err)
	}
	if again.Status != store.StatusArchived {
		t.Fatalf("status %q, want archived", again.Status)
	}
}

func TestLoadArchivesHashMismatch(t *testing.T) {
	st := newStore(t)
	f := store.New("TAMPER", store.ModeAI, engine.Options{}, 1, time.Now())
	f.Status = store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rewriteField(t, st, "TAMPER", func(raw map[string]any) {
		raw["hash"] = "0000000000000000"
	})

	_, _, err := st.Load("TAMPER")
	if err == nil {
		t.Fatalf("Load with a tampered hash: want error, got nil")
	}
	again, err := st.PeekHeader("TAMPER")
	if err != nil {
		t.Fatalf("PeekHeader: %v", err)
	}
	if again.Status != store.StatusArchived {
		t.Fatalf("status %q, want archived", again.Status)
	}
}

func TestNewCodeIsUnique(t *testing.T) {
	st := newStore(t)
	first, err := st.NewCode()
	if err != nil {
		t.Fatalf("NewCode: %v", err)
	}
	f := store.New(first, store.ModeAI, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	second, err := st.NewCode()
	if err != nil {
		t.Fatalf("NewCode: %v", err)
	}
	if second == first {
		t.Fatalf("NewCode returned %q twice in a row with a file already on disk", first)
	}
}

func TestFileJSONShape(t *testing.T) {
	st := newStore(t)
	f := store.New("SHAPE9", store.ModeOnline, engine.Options{}, 42, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(st.Dir(), "SHAPE9.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{
		"schema", "rules", "code", "mode", "created", "updated",
		"seats", "options", "seed", "actions", "lastSeq", "status", "hash",
	} {
		if _, ok := raw[key]; !ok {
			t.Errorf("room file is missing key %q", key)
		}
	}
	for _, key := range []string{"scenario", "result"} {
		if _, ok := raw[key]; ok {
			t.Errorf("room file has key %q with no value for it (omitempty should have dropped it)", key)
		}
	}
	if seedStr, ok := raw["seed"].(string); !ok || seedStr != "42" {
		t.Errorf("seed = %#v, want the decimal string \"42\"", raw["seed"])
	}
	if actions, ok := raw["actions"].([]any); !ok || actions == nil {
		t.Errorf("actions = %#v, want a (possibly empty) array, never null", raw["actions"])
	}
	if lastSeq, ok := raw["lastSeq"].(map[string]any); !ok || lastSeq == nil {
		t.Errorf("lastSeq = %#v, want a (possibly empty) object, never null", raw["lastSeq"])
	}
}

func TestScenarioRoomReplays(t *testing.T) {
	st := newStore(t)
	f := store.New("SCENAR", store.ModeHotseat, engine.Options{}, 7, time.Now())
	f.Scenario = "military-supremacy"
	f.Actions = []string{"construct:walls"}
	f.Status = store.StatusFinished
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, s, err := st.Load("SCENAR")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !s.Over() {
		t.Fatalf("military-supremacy's trigger should end the game")
	}
	if s.Result.Kind != engine.VictoryMilitary {
		t.Fatalf("victory kind %v, want military", s.Result.Kind)
	}
}

func TestReaperSweep(t *testing.T) {
	st := newStore(t)
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		code    string
		status  store.Status
		age     time.Duration
		removed bool
	}{
		{"FRESHA", store.StatusPlaying, time.Hour, false},
		{"CASEA2", store.StatusLobby, 20 * 24 * time.Hour, true},
		{"CASEB2", store.StatusPlaying, 10 * 24 * time.Hour, false}, // under IdleAfter
		{"CASEC2", store.StatusFinished, 8 * 24 * time.Hour, true},
		{"CASED2", store.StatusFinished, 3 * 24 * time.Hour, false}, // under FinishedAfter
		{"CASEE2", store.StatusArchived, 10 * 24 * time.Hour, true},
	}
	for _, c := range cases {
		// store.New stamps Created and Updated from the time it is given,
		// and Save never touches Updated itself (the room package, not
		// store, owns the injected clock), so this pins each case's age.
		f := store.New(c.code, store.ModeAI, engine.Options{}, 1, now.Add(-c.age))
		f.Status = c.status
		if err := st.Save(f); err != nil {
			t.Fatalf("Save %s: %v", c.code, err)
		}
	}

	reaper := store.NewReaper(st, func() time.Time { return now })
	removed, err := reaper.Sweep()
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	gotRemoved := map[string]bool{}
	for _, code := range removed {
		gotRemoved[code] = true
	}
	for _, c := range cases {
		if gotRemoved[c.code] != c.removed {
			t.Errorf("%s: removed = %v, want %v", c.code, gotRemoved[c.code], c.removed)
		}
		exists, err := st.Exists(c.code)
		if err != nil {
			t.Fatalf("Exists(%s): %v", c.code, err)
		}
		if exists == c.removed {
			t.Errorf("%s: Exists = %v after Sweep, want %v", c.code, exists, !c.removed)
		}
	}
}

func TestConcurrentDifferentRooms(t *testing.T) {
	st := newStore(t)
	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			code, err := st.NewCode()
			if err != nil {
				errs[i] = err
				return
			}
			keys, _ := playActions(t, uint64(i+1), engine.Options{}, 3)
			f := store.New(code, store.ModeAI, engine.Options{}, uint64(i+1), time.Now())
			f.Actions = keys
			if err := st.Save(f); err != nil {
				errs[i] = err
				return
			}
			if _, _, err := st.Load(code); err != nil {
				errs[i] = err
			}
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
}

func TestNewReaperDefaultClock(t *testing.T) {
	st := newStore(t)
	reaper := store.NewReaper(st, nil) // nil clock means time.Now
	if _, err := reaper.Sweep(); err != nil {
		t.Fatalf("Sweep with the default clock: %v", err)
	}
}

func TestPeekExistsDeleteRejectBadCode(t *testing.T) {
	st := newStore(t)
	if _, err := st.PeekHeader("../etc"); err == nil {
		t.Errorf("PeekHeader with a bad code: want error, got nil")
	}
	if _, err := st.Exists("not valid"); err == nil {
		t.Errorf("Exists with a bad code: want error, got nil")
	}
	if err := st.Delete("x"); err == nil {
		t.Errorf("Delete with a bad code: want error, got nil")
	}
	if _, _, err := st.Load("../../x"); err == nil {
		t.Errorf("Load with a bad code: want error, got nil")
	}
}

func TestDeleteMissingIsNotError(t *testing.T) {
	st := newStore(t)
	if err := st.Delete("MSNG92"); err != nil {
		t.Fatalf("Delete of a room that never existed: %v", err)
	}
}

func TestCodesIgnoresJunk(t *testing.T) {
	st := newStore(t)
	f := store.New("ACTVE2", store.ModeAI, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// A leftover temp file (as an interrupted write would leave) and a
	// completely unrelated file must not show up as room codes.
	if err := os.WriteFile(filepath.Join(st.Dir(), ".room-abc.tmp"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(st.Dir(), "README.txt"), []byte("hi"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	codes, err := st.Codes()
	if err != nil {
		t.Fatalf("Codes: %v", err)
	}
	if len(codes) != 1 || codes[0] != "ACTVE2" {
		t.Fatalf("Codes() = %v, want [REALON]", codes)
	}
}

func TestSweepSkipsUnreadableHeader(t *testing.T) {
	st := newStore(t)
	good := store.New("ENDED2", store.ModeAI, engine.Options{}, 1, time.Now().Add(-30*24*time.Hour))
	good.Status = store.StatusFinished
	if err := st.Save(good); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// A file that looks like a room (valid code, .json suffix) but holds
	// garbage: Sweep must skip it rather than fail the whole pass, and must
	// still remove the genuinely eligible room alongside it.
	if err := os.WriteFile(filepath.Join(st.Dir(), "GARBD2.json"), []byte("not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	reaper := store.NewReaper(st, time.Now)
	removed, err := reaper.Sweep()
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	found := false
	for _, c := range removed {
		if c == "ENDED2" {
			found = true
		}
		if c == "GARBD2" {
			t.Fatalf("Sweep removed the unreadable file instead of just skipping it")
		}
	}
	if !found {
		t.Fatalf("Sweep did not remove the eligible finished room; removed = %v", removed)
	}
}

// rewriteField reads code's room file, decodes it loosely into a
// map[string]any, lets mutate edit it, and writes the result straight back
// (bypassing Save, which would recompute a correct Hash) -- simulating a
// file that was corrupted or written by a different engine version.
func rewriteField(t *testing.T, st *store.Store, code string, mutate func(raw map[string]any)) {
	t.Helper()
	path := filepath.Join(st.Dir(), code+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	mutate(raw)
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
