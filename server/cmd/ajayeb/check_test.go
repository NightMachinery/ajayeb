package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestCheckRoomsExitsWithoutInitialization(t *testing.T) {
	data := filepath.Join(t.TempDir(), "missing")
	var out, stderr bytes.Buffer
	// An invalid listen address would fail if check mode opened a socket.
	if code := run([]string{"--check-rooms", "--data", data, "--addr", "not-an-address"}, &out, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, &stderr)
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Fatalf("created data/identity: %v", err)
	}
	var report store.CheckReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Format != "ajayeb-room-check/1" {
		t.Fatalf("report: %s %v", &out, err)
	}
}

func TestCheckRoomsExitCodesAndReadOnlyBytes(t *testing.T) {
	data := t.TempDir()
	st, err := store.Open(data)
	if err != nil {
		t.Fatal(err)
	}
	f := store.New("CXTTST", store.ModeOnline, engine.Options{}, 1, time.Now())
	f.Status = store.StatusPlaying
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(st.Dir(), f.Code+".json")
	for _, tc := range []struct{ rules, schema, want int }{{engine.RulesVersion, store.Schema, 0}, {engine.RulesVersion + 1, store.Schema, 3}, {engine.RulesVersion, store.Schema + 1, 1}} {
		f.Rules, f.Schema = tc.rules, tc.schema
		raw, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		if code := run([]string{"--check-rooms", "--data", data}, &out, &stderr); code != tc.want {
			t.Fatalf("exit %d want %d: %s", code, tc.want, &stderr)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, after) {
			t.Fatal("check changed room bytes")
		}
		entries, err := os.ReadDir(data)
		if err != nil || len(entries) != 1 || entries[0].Name() != "rooms" {
			t.Fatal("check initialized identity")
		}
	}
}

func TestCheckRoomsArchiveFolderIsReadOnlyAndNeverLiveRisk(t *testing.T) {
	for _, liveFolder := range []bool{false, true} {
		t.Run(map[bool]string{false: "archive-only", true: "both-folders"}[liveFolder], func(t *testing.T) {
			data := t.TempDir()
			archive := filepath.Join(data, "rooms-archive")
			if err := os.Mkdir(archive, 0o700); err != nil {
				t.Fatal(err)
			}
			f := store.New("ARCHVD", store.ModeOnline, engine.Options{}, 1, time.Now())
			f.Status, f.Rules, f.ArchiveReason = store.StatusArchived, engine.RulesVersion+1, store.ArchiveRulesVersion
			f.Actions = []string{"not-a-replayable-key"}
			raw, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(archive, f.Code+".json")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if liveFolder {
				if err := os.Mkdir(filepath.Join(data, "rooms"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(archive, "stray.json"), []byte("{"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(archive, "FDERXX.json"), 0o700); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := run([]string{"--check-rooms", "--data", data}, &out, &stderr); code != 0 {
				t.Fatalf("exit %d: %s", code, &stderr)
			}
			var r store.CheckReport
			if err := json.Unmarshal(out.Bytes(), &r); err != nil {
				t.Fatal(err)
			}
			if r.Rooms != 1 || r.Archived != 1 || r.Playable != 0 || r.WouldArchive != 0 || r.LiveWouldArchive != 0 || r.Errors != 0 {
				t.Fatalf("archive report: %+v", r)
			}
			if after, err := os.ReadFile(path); err != nil || !bytes.Equal(raw, after) {
				t.Fatal("check changed archive bytes")
			}
			entries, err := os.ReadDir(data)
			want := 1
			if liveFolder {
				want++
			}
			if err != nil || len(entries) != want {
				t.Fatal("check initialized data")
			}
		})
	}
}
