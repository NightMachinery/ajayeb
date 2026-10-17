package tutorial

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file pins the derived tutorial object (Derive) for FirstGame in a
// golden file: at every position of the script, one past its end, after a
// deviation at each guided turn (and up to the player's next move), at the
// Age II starter choice, once Age II is in play, and after a forfeit. Run
//
//	go test ./internal/tutorial/ -run TestDeriveGolden -update
//
// after a deliberate change to the script or the derivation.

var updateGolden = flag.Bool("update", false, "rewrite testdata/derive.golden.json")

// game replays keys from the script's seed and returns the state and the
// seat that took each key.
func game(t *testing.T, sc *Script, keys []string) (engine.State, []engine.Seat) {
	t.Helper()
	s := engine.NewGame(sc.Seed, sc.Options)
	seats := make([]engine.Seat, 0, len(keys))
	for i, key := range keys {
		a, err := engine.ParseKey(key)
		if err != nil {
			t.Fatalf("key %d %q: %v", i, key, err)
		}
		seat := s.ToMove
		if a.Type == engine.ActResign {
			seat = a.Starter
		}
		if err := engine.IsLegal(&s, a); err != nil {
			t.Fatalf("key %d %q: %v", i, key, err)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("key %d %q: %v", i, key, err)
		}
		seats = append(seats, seat)
	}
	return s, seats
}

// scriptKeys returns the first p keys of the script.
func scriptKeys(sc *Script, p int) []string {
	keys := make([]string, 0, p)
	for _, e := range sc.Entries[:p] {
		keys = append(keys, e.Key)
	}
	return keys
}

// firstLegalOther returns the first legal key of s other than not.
func firstLegalOther(t *testing.T, s *engine.State, not string) string {
	t.Helper()
	for _, a := range engine.LegalActions(s, nil) {
		if k := a.Key(); k != not {
			return k
		}
	}
	t.Fatalf("no legal key other than %q", not)
	return ""
}

// goldenCase is one derived object of the golden file.
type goldenCase struct {
	Case     string             `json:"case"`
	Ply      int                `json:"ply"`
	Tutorial wire.TutorialState `json:"tutorial"`
}

func TestDeriveGolden(t *testing.T) {
	sc := &FirstGame
	n := len(sc.Entries)
	var cases []goldenCase
	add := func(name string, keys []string) wire.TutorialState {
		s, seats := game(t, sc, keys)
		ts := Derive(sc, keys, seats, &s)
		cases = append(cases, goldenCase{Case: name, Ply: len(keys), Tutorial: ts})
		return ts
	}

	// Every position of the script, and one past it.
	for p := 0; p <= n; p++ {
		add(fmt.Sprintf("on script at %d", p), scriptKeys(sc, p))
	}
	end, _ := game(t, sc, scriptKeys(sc, n))
	past := append(scriptKeys(sc, n), firstLegalOther(t, &end, ""))
	add("one past the script", past)

	// A deviation at each guided turn: right after it (the bot to move),
	// after the bot's reply (the player to move, leftPath still shown), and
	// after the player's next move (leftPath gone).
	for i, e := range sc.Entries {
		if e.Seat != sc.Human {
			continue
		}
		keys := scriptKeys(sc, i)
		s, _ := game(t, sc, keys)
		keys = append(keys, firstLegalOther(t, &s, e.Key))
		add(fmt.Sprintf("left at %d", i), keys)
		for _, label := range []string{"left at %d, next move", "left at %d, the player moved again"} {
			s, _ = game(t, sc, keys)
			if s.Over() {
				break
			}
			keys = append(keys, firstLegalOther(t, &s, ""))
			add(fmt.Sprintf(label, i), keys)
		}
	}

	// Playing on from the end of the script to the Age II starter choice
	// (free) and to Age II in play (complete).
	keys := scriptKeys(sc, n)
	sawStarter := false
	for {
		s, _ := game(t, sc, keys)
		if s.Age >= 2 && s.Phase == engine.PhaseChooseStarter && !sawStarter {
			sawStarter = true
			add("Age II starter choice", keys)
		}
		if s.Age >= 2 && s.Phase == engine.PhasePlay {
			add("Age II in play", keys)
			break
		}
		if s.Over() || len(keys) > 200 {
			t.Fatalf("the game did not reach Age II in play (over %v, %d keys)", s.Over(), len(keys))
		}
		keys = append(keys, firstLegalOther(t, &s, ""))
	}

	// A forfeit in the middle of the script.
	add("forfeit", append(scriptKeys(sc, 9), engine.ResignAction(sc.Human).Key()))

	got, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "derive.golden.json")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to create it)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is stale; run go test ./internal/tutorial/ -run TestDeriveGolden -update after a deliberate change:\n%s", path, got)
	}
}

// TestDeriveRules checks the derivation's rules directly, beside the golden
// file: the guided turns, the outro, leftPath, the statuses, undo and the
// never-nil lessons.
func TestDeriveRules(t *testing.T) {
	sc := &FirstGame
	n := len(sc.Entries)
	turns := sc.Turns()
	if turns != 10 {
		t.Errorf("FirstGame has %d guided turns, want 10 (3 draft picks, 7 Age I turns)", turns)
	}
	turn := 0
	for p := 0; p <= n; p++ {
		keys := scriptKeys(sc, p)
		s, seats := game(t, sc, keys)
		ts := Derive(sc, keys, seats, &s)
		if ts.Lessons == nil {
			t.Fatalf("at %d: nil lessons", p)
		}
		if ts.Left || ts.Script != sc.Name {
			t.Errorf("at %d: left %v, script %q", p, ts.Left, ts.Script)
		}
		if p == n {
			if ts.Status != wire.TutorialStatusFree || ts.Action != nil || ts.Turn != 0 || ts.Turns != 0 || !reflect.DeepEqual(ts.Lessons, sc.Outro) {
				t.Errorf("at the end: %+v, want free with the outro", ts)
			}
			continue
		}
		e := sc.Entries[p]
		if ts.Status != wire.TutorialStatusGuided || ts.Turns != turns {
			t.Errorf("at %d: status %s, turns %d", p, ts.Status, ts.Turns)
		}
		if e.Seat == sc.Human {
			turn++
			if ts.Action == nil || ts.Action.Key != e.Key || !reflect.DeepEqual(ts.Lessons, e.Lessons) {
				t.Errorf("at %d: action %+v lessons %v, want the entry's", p, ts.Action, ts.Lessons)
			}
			if ts.Turn != turn {
				t.Errorf("at %d: turn %d, want %d", p, ts.Turn, turn)
			}
		} else if ts.Action != nil || len(ts.Lessons) != 0 {
			t.Errorf("at %d (the bot's entry): action %+v lessons %v", p, ts.Action, ts.Lessons)
		}
		// BotKey names exactly the bot's entries; GuidedMove every
		// script move once played.
		key, ok := BotKey(sc, keys)
		if ok != (e.Seat != sc.Human) || (ok && key != e.Key) {
			t.Errorf("at %d: BotKey = %q, %v", p, key, ok)
		}
		played := append(slices.Clone(keys), e.Key)
		if !GuidedMove(sc, played, p) {
			t.Errorf("at %d: GuidedMove is false for the script's own move", p)
		}
	}
	if _, ok := BotKey(sc, scriptKeys(sc, n)); ok {
		t.Error("BotKey past the script's end")
	}

	// A deviation: no bot key, no guided move, leftPath; undoing it brings
	// the guided turn back unchanged.
	i := 0
	for sc.Entries[i].Seat != sc.Human || i == 0 {
		i++
	}
	keys := scriptKeys(sc, i)
	s, seats := game(t, sc, keys)
	before := Derive(sc, keys, seats, &s)
	off := append(slices.Clone(keys), firstLegalOther(t, &s, sc.Entries[i].Key))
	s2, seats2 := game(t, sc, off)
	ts := Derive(sc, off, seats2, &s2)
	if !ts.Left || ts.Status != wire.TutorialStatusFree || len(ts.Lessons) != 1 || ts.Lessons[0].ID != wire.TutorialLessonLeftPath || ts.Action != nil {
		t.Errorf("after a deviation at %d: %+v", i, ts)
	}
	if _, ok := BotKey(sc, off); ok {
		t.Error("BotKey after a deviation")
	}
	if GuidedMove(sc, off, i) {
		t.Error("GuidedMove for the deviating move")
	}
	if !GuidedMove(sc, off, i-1) {
		t.Error("GuidedMove false for a script move before the deviation")
	}
	if GuidedMove(sc, off, -1) || GuidedMove(sc, off, len(off)) {
		t.Error("GuidedMove out of range")
	}
	s3, seats3 := game(t, sc, keys)
	if again := Derive(sc, keys, seats3, &s3); !reflect.DeepEqual(again, before) {
		t.Errorf("after undo: %+v, want %+v", again, before)
	}
	// A short seats slice never shows leftPath by mistake.
	off2 := append(slices.Clone(off), firstLegalOther(t, &s2, ""))
	s4, _ := game(t, sc, off2)
	if ts := Derive(sc, off2, nil, &s4); len(ts.Lessons) != 0 {
		t.Errorf("with no seats: %+v", ts)
	}
}

// TestDeriveCopies checks that Derive's lessons and action are copies:
// changing them leaves the script alone.
func TestDeriveCopies(t *testing.T) {
	sc := &FirstGame
	s, seats := game(t, sc, nil)
	ts := Derive(sc, nil, seats, &s)
	ts.Lessons[0].ID = "changed"
	ts.Action.Key = "changed"
	if sc.Entries[0].Lessons[0].ID == "changed" || sc.Entries[0].Action.Key == "changed" {
		t.Fatal("Derive returned the script's own lessons or action")
	}
}
