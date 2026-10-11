package store_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestReadLogReplays(t *testing.T) {
	st := newStore(t)
	keys, final := playActions(t, 11, engine.Options{}, 400)
	if !final.Over() {
		t.Fatalf("the first-legal playout of seed 11 should end within 400 plies")
	}
	f := store.New("GREADX", store.ModeHotseat, engine.Options{}, 11, time.Now())
	f.Actions = keys
	f.Status = store.StatusFinished
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	hdr, l, err := st.ReadLog("GREADX")
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if hdr.Status != store.StatusFinished {
		t.Errorf("status %q, want finished", hdr.Status)
	}
	if l.Format != engine.LogFormat || l.Rules != engine.RulesVersion || l.Seed != 11 || l.Scenario != nil {
		t.Errorf("log header %+v", l)
	}
	if len(l.Actions) != len(keys) {
		t.Fatalf("%d log entries, want %d", len(l.Actions), len(keys))
	}
	if l.FinalHash != engine.HashString(&final) {
		t.Errorf("finalHash %s, want %s", l.FinalHash, engine.HashString(&final))
	}

	// The log survives a JSON round trip and replays to the same state.
	data, err := json.Marshal(l)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back engine.Log
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	s, err := engine.Replay(&back)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if engine.HashString(&s) != engine.HashString(&final) {
		t.Errorf("replayed hash differs")
	}
}

func TestReadLogScenario(t *testing.T) {
	st := newStore(t)
	f := store.New("GSCENX", store.ModeHotseat, engine.Options{}, 7, time.Now())
	f.Scenario = "military-supremacy"
	f.Actions = []string{"construct:walls"}
	f.Status = store.StatusFinished
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, l, err := st.ReadLog("GSCENX")
	if err != nil {
		t.Fatalf("ReadLog: %v", err)
	}
	if l.Scenario == nil {
		t.Fatalf("a scenario room's log should carry its scenario")
	}
	s, err := engine.Replay(l)
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if !s.Over() {
		t.Errorf("the replay should end the game")
	}
}

func TestReadLogErrors(t *testing.T) {
	st := newStore(t)
	if _, _, err := st.ReadLog("bad"); !errors.Is(err, store.ErrBadCode) {
		t.Errorf("bad code: %v, want ErrBadCode", err)
	}
	if _, _, err := st.ReadLog("NNNNNN"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing room: %v, want fs.ErrNotExist", err)
	}

	f := store.New("GTAMPX", store.ModeAI, engine.Options{}, 1, time.Now())
	f.Actions, _ = playActions(t, 1, engine.Options{}, 3)
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	rewriteField(t, st, "GTAMPX", func(raw map[string]any) { raw["hash"] = "0000000000000000" })
	before, err := os.ReadFile(filepath.Join(st.Dir(), "GTAMPX.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.ReadLog("GTAMPX"); !errors.Is(err, store.ErrHashMismatch) {
		t.Errorf("tampered room: %v, want ErrHashMismatch", err)
	}
	after, err := os.ReadFile(filepath.Join(st.Dir(), "GTAMPX.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("ReadLog rewrote the file; it must only read")
	}

	rewriteField(t, st, "GTAMPX", func(raw map[string]any) { raw["rules"] = float64(engine.RulesVersion + 1) })
	if _, _, err := st.ReadLog("GTAMPX"); !errors.Is(err, store.ErrRulesVersion) {
		t.Errorf("other rules: %v, want ErrRulesVersion", err)
	}
}
