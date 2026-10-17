package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// TestNoUndoPersisted checks that a room created with undo off keeps that
// through a save and a load, and that the flag is not part of the game: the
// hash is the same with and without it.
func TestNoUndoPersisted(t *testing.T) {
	st := newStore(t)
	keys, _ := playActions(t, 3, engine.Options{}, 4)
	plain := store.New("UNDXPA", store.ModeHotseat, engine.Options{}, 3, time.Now())
	plain.Actions = keys
	if err := st.Save(plain); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f := store.New("UNDXFF", store.ModeHotseat, engine.Options{}, 3, time.Now())
	f.Actions = keys
	f.NoUndo = true
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if f.Hash != plain.Hash {
		t.Fatalf("hash %s with undo off, %s with it on: the flag must not be part of the game", f.Hash, plain.Hash)
	}
	got, _, err := st.Load("UNDXFF")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.NoUndo {
		t.Fatalf("Load: NoUndo false, want true")
	}
}

// TestNoUndoAbsentInOldFile checks that a file without the field (one
// written before it existed, or one of a room that allows undo) leaves the
// key out and loads with undo allowed.
func TestNoUndoAbsentInOldFile(t *testing.T) {
	st := newStore(t)
	f := store.New("UNDXYD", store.ModeAI, engine.Options{}, 1, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(st.Dir(), "UNDXYD.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), `"noUndo"`) {
		t.Fatalf("a room that allows undo has the key noUndo:\n%s", data)
	}
	got, _, err := st.Load("UNDXYD")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.NoUndo {
		t.Fatalf("Load: NoUndo true for a file without the field")
	}
}

// TestTutorialPersisted checks that a tutorial room's script name survives a
// save and a load, stays out of the file of every other room, and is not
// part of the game: the hash is the same with and without it.
func TestTutorialPersisted(t *testing.T) {
	st := newStore(t)
	keys, _ := playActions(t, 5, engine.Options{FirstPlayer: engine.FirstPlayerSeat0}, 4)
	plain := store.New("TUTXPA", store.ModeAI, engine.Options{FirstPlayer: engine.FirstPlayerSeat0}, 5, time.Now())
	plain.Actions = keys
	if err := st.Save(plain); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(st.Dir(), "TUTXPA.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), `"tutorial"`) {
		t.Fatalf("a room that is no tutorial has the key tutorial:\n%s", data)
	}

	f := store.New("TUTXFF", store.ModeAI, engine.Options{FirstPlayer: engine.FirstPlayerSeat0}, 5, time.Now())
	f.Actions = keys
	f.Tutorial = "first-game"
	if err := st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if f.Hash != plain.Hash {
		t.Fatalf("hash %s with a tutorial, %s without: the name must not be part of the game", f.Hash, plain.Hash)
	}
	got, _, err := st.Load("TUTXFF")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Tutorial != "first-game" {
		t.Fatalf("Load: Tutorial %q, want first-game", got.Tutorial)
	}
}
