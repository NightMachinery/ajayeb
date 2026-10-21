package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// These are real Store writes, including whitespace and archive paths, not
// JSON round trips. All identities and metadata are synthetic test values.
func TestPlatformSchema1Baseline(t *testing.T) {
	for _, name := range []string{"lobby", "playing", "finished", "archived-playing", "archived-finished", "rules-mismatch", "hash-refresh"} {
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
			data := t.TempDir()
			st, err := store.OpenWithClock(data, func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			f := store.New("BASENX", store.ModeOnline, engine.Options{FirstPlayer: engine.FirstPlayerSeat0}, 11, now.Add(-time.Hour))
			f.Seats = [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "fixture-owner", Name: "Seat 0", Epoch: 2}, {Kind: store.SeatKindOpen}}
			f.Admin = "fixture-owner"
			if name != "lobby" {
				f.Mode = store.ModeHotseat
				f.Seats[1] = store.Seat{Kind: store.SeatKindHuman, UserID: "fixture-owner", Name: "Seat 1", NameNo: 2}
				f.Status = store.StatusPlaying
				f.Actions, _ = playActions(t, f.Seed, f.Options, 12)
				f.Version = 19
				f.LastSeq["fixture-owner:fixture-tab"] = 17
				f.NoUndo = true
				f.Hints = &view.HintSettings{Prices: true, Advisor: false}
				f.RoomLog = []store.LogEntry{{Kind: "hints", At: 3, Seat: -1, On: []string{"prices"}, Off: []string{"advisor"}}}
			}
			if name == "finished" || name == "archived-finished" {
				var state engine.State
				f.Actions, state = playActions(t, f.Seed, f.Options, 200)
				if !state.Over() {
					t.Fatal("fixture policy did not finish")
				}
				f.Version = len(f.Actions) + 2
				f.Status, f.Result, f.Rematch = store.StatusFinished, &state.Result, "REMATC"
			}
			if name == "archived-playing" || name == "archived-finished" {
				f.Status, f.ArchiveReason, f.ArchivedAt = store.StatusArchived, store.ArchiveReplayFailed, &now
			}
			if err := st.Save(f); err != nil {
				t.Fatal(err)
			}
			if name == "rules-mismatch" || name == "hash-refresh" {
				rewriteField(t, st, f.Code, func(raw map[string]any) {
					raw["futureMetadata"] = map[string]any{"keep": true}
					if name == "rules-mismatch" {
						raw["rules"] = engine.RulesVersion + 1
					} else {
						raw["hash"] = "previous-layout"
					}
				})
				_, _, loadErr := st.Load(f.Code)
				if name == "rules-mismatch" && !errors.Is(loadErr, store.ErrArchived) {
					t.Fatalf("archive: %v", loadErr)
				}
				if name == "hash-refresh" && loadErr != nil {
					t.Fatal(loadErr)
				}
			}
			folder := "rooms"
			archived := name == "archived-playing" || name == "archived-finished" || name == "rules-mismatch"
			if archived {
				folder = "rooms-archive"
			}
			relative := filepath.Join(folder, f.Code+".json")
			actual, err := os.ReadFile(filepath.Join(data, relative))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", "platform-schema1", name, relative)
			if os.Getenv("UPDATE_PLATFORM_BASELINES") == "1" {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, actual, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			golden, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, golden) {
				t.Fatalf("%s: schema-1 write bytes changed; regenerate deliberately (docs/testing.md)", path)
			}
			// Load the committed bytes in a separate store, not merely the value
			// just saved. Compatible loads and repeated archive loads never rewrite.
			fresh := t.TempDir()
			target := filepath.Join(fresh, relative)
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, golden, 0o600); err != nil {
				t.Fatal(err)
			}
			reader, err := store.OpenWithClock(fresh, func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			got, state, err := reader.Load(f.Code)
			if archived {
				if !errors.Is(err, store.ErrArchived) {
					t.Fatalf("want archived, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var want store.File
			if err := json.Unmarshal(golden, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, &want) {
				t.Fatal("load changed schema-1 fields")
			}
			if !archived && engine.HashString(&state) != want.Hash {
				t.Fatal("loaded replay hash changed")
			}
			after, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(after, golden) {
				t.Fatal("load rewrote committed bytes", err)
			}
			if archived {
				if _, err := os.Stat(filepath.Join(data, "rooms", f.Code+".json")); !os.IsNotExist(err) {
					t.Fatal("archive remained in live layout")
				}
				if codes, err := reader.Codes(); err != nil || len(codes) != 0 {
					t.Fatal("archive joined live scan", codes, err)
				}
			}
		})
	}
}
