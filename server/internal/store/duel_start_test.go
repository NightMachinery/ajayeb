package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestSharedLobbyDuelReplacement(t *testing.T) {
	for _, status := range []string{"lobby", "playing", "archived", "missing", "garbage", "other-code", "other-game"} {
		t.Run(status, func(t *testing.T) {
			codecs, err := store.NewCodecs(store.Codec{Game: "waiting", Decode: func(raw []byte) (store.Inspection, error) {
				var r gameRecord
				err := json.Unmarshal(raw, &r)
				return &r, err
			}})
			if err != nil {
				t.Fatal(err)
			}
			st, err := store.OpenWithCodecs(t.TempDir(), nil, codecs)
			if err != nil {
				t.Fatal(err)
			}
			code := "DUSTAA"
			oldCode, oldGame := code, "waiting"
			if status == "other-code" {
				oldCode = "DUSTAB"
			}
			if status == "other-game" {
				oldGame = "unknown"
			}
			oldStatus := status
			if status == "other-code" || status == "other-game" {
				oldStatus = "lobby"
			}
			data := []byte(`{"schema":2,"game":"` + oldGame + `","code":"` + oldCode + `","status":"` + oldStatus + `","updated":"2000-01-01T00:00:00Z"}`)
			if status == "garbage" {
				data = []byte("{")
			}
			path := filepath.Join(st.Dir(), code+".json")
			if status != "missing" {
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			f := store.New(code, store.ModeOnline, engine.Options{}, 11, gameNow)
			f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "a", Name: "A"}, {Kind: store.SeatKindHuman, UserID: "b", Name: "B"}}
			f.Status = store.StatusPlaying
			err = st.ReplaceDuel("waiting", f)
			if status != "lobby" {
				if err == nil {
					t.Fatal("replaced a target that was not a clean live lobby")
				}
				actual, readErr := os.ReadFile(path)
				if status == "missing" {
					if !os.IsNotExist(readErr) {
						t.Fatal("created a missing target")
					}
				} else if readErr != nil || !bytes.Equal(actual, data) {
					t.Fatal("refused target changed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			actual, _, err := st.Load(code)
			if err != nil || actual.Hash == "" || actual.Status != store.StatusPlaying || actual.Seats != f.Seats {
				t.Fatalf("load after replacement: %+v %v", actual, err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := st.ReplaceDuel("waiting", f); err == nil {
				t.Fatal("a second Start replaced Duel")
			}
			now, _ := os.ReadFile(path)
			if !bytes.Equal(saved, now) {
				t.Fatal("second Start changed bytes")
			}
		})
	}
}

func TestSharedLobbyDuelReplacementConcurrent(t *testing.T) {
	st := gameStore(t)
	code := "DUSTAC"
	if err := st.SaveGame(code, "fixture", gameFile(code, "lobby")); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			f := store.New(code, store.ModeOnline, engine.Options{}, uint64(seed), gameNow)
			f.Status = store.StatusPlaying
			out <- st.ReplaceDuel("fixture", f)
		}(i)
	}
	wg.Wait()
	close(out)
	wins := 0
	for err := range out {
		if err == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("concurrent Start winners %d, want 1", wins)
	}
}

// A shared Duel game's frozen hint admins follow the rules Start writes
// them under (store.ValidHintAdmins): no save writes a damaged set, and a
// file holding one (hand-edited or damaged) archives as replay-failed on
// load, which the deployment check reports first.
func TestHintAdminsChecked(t *testing.T) {
	many := make([]string, store.MaxHintAdmins+1)
	for i := range many {
		many[i] = fmt.Sprintf("u%d", i)
	}
	for _, tc := range []struct {
		name   string
		admins []string
	}{{"empty id", []string{"a", ""}}, {"repeated", []string{"a", "b", "a"}}, {"too many", many}, {"too long", []string{strings.Repeat("x", 65)}}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			st, err := store.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			f := store.New("HADMAA", store.ModeOnline, engine.Options{}, 11, gameNow)
			f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "a", Name: "A"}, {Kind: store.SeatKindHuman, UserID: "b", Name: "B"}}
			f.Status = store.StatusPlaying
			f.HintAdmins = []string{"a", "b"}
			if err := st.Save(f); err != nil {
				t.Fatal(err)
			}
			bad := *f
			bad.HintAdmins = tc.admins
			if err := st.Save(&bad); err == nil {
				t.Fatal("saved a damaged hint-admin set")
			}
			path := filepath.Join(st.Dir(), "HADMAA.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			doc["hintAdmins"] = tc.admins
			if raw, err = json.Marshal(doc); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if r := store.CheckDataDir(dir); r.LiveWouldArchive != 1 || r.Playable != 0 {
				t.Fatalf("deployment check: %+v", r)
			}
			_, _, err = st.Load("HADMAA")
			var archived *store.ArchivedError
			if !errors.As(err, &archived) || archived.Reason != store.ArchiveReplayFailed {
				t.Fatalf("damaged hint admins loaded: %v", err)
			}
		})
	}
}
