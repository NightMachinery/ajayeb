package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestArchiveMovesOutOfLiveScanAndKeepsMetadata(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	st, err := store.OpenWithClock(t.TempDir(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	f := store.New("ARCXYZ", store.ModeOnline, engine.Options{}, 2, now.Add(-time.Hour))
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, f.Code, func(raw map[string]any) {
		raw["rules"] = engine.RulesVersion + 1
		raw["futureMetadata"] = map[string]any{"keep": true}
	})
	got, _, err := st.Load(f.Code)
	if !errors.Is(err, store.ErrArchived) || got.ArchiveReason != store.ArchiveRulesVersion || !got.ArchivedAt.Equal(now) {
		t.Fatalf("archive: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(st.Dir(), f.Code+".json")); !os.IsNotExist(err) {
		t.Fatalf("live file remains: %v", err)
	}
	if codes, err := st.Codes(); err != nil || len(codes) != 0 {
		t.Fatalf("live scan: %v, %v", codes, err)
	}
	if exists, err := st.Exists(f.Code); err != nil || !exists {
		t.Fatalf("archived code not reserved: %v, %v", exists, err)
	}
	path := filepath.Join(st.ArchiveDir(), f.Code+".json")
	before, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(before, []byte("futureMetadata")) {
		t.Fatalf("lost metadata: %s, %v", before, err)
	}
	now = now.Add(24 * time.Hour)
	if _, _, err := st.Load(f.Code); !errors.Is(err, store.ErrArchived) {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("repeat load rewrote archive")
	}
	if err := st.Save(f); !errors.Is(err, store.ErrArchived) {
		t.Fatalf("resurrected archive: %v", err)
	}
	if err := st.Delete(f.Code); err != nil {
		t.Fatal(err)
	}
	if exists, _ := st.Exists(f.Code); exists {
		t.Fatal("Delete left archive")
	}
}

func TestLegacyArchiveMigrationPreservesBytesAndAge(t *testing.T) {
	for _, viaLoad := range []bool{false, true} {
		t.Run(map[bool]string{false: "sweep", true: "load"}[viaLoad], func(t *testing.T) {
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			st := newStore(t)
			f := store.New("XEGACY", store.ModeOnline, engine.Options{}, 2, now.Add(-29*24*time.Hour))
			if err := st.Save(f); err != nil {
				t.Fatal(err)
			}
			rewriteField(t, st, f.Code, func(raw map[string]any) {
				raw["status"] = store.StatusArchived
				raw["rules"] = engine.RulesVersion + 1
				raw["futureMetadata"] = "unchanged"
			})
			live := filepath.Join(st.Dir(), f.Code+".json")
			before, err := os.ReadFile(live)
			if err != nil {
				t.Fatal(err)
			}
			reaper := store.NewReaper(st, func() time.Time { return now })
			if viaLoad {
				if _, _, err := st.Load(f.Code); !errors.Is(err, store.ErrArchived) {
					t.Fatal(err)
				}
			} else if removed, err := reaper.Sweep(); err != nil || len(removed) != 0 {
				t.Fatalf("migration: %v, %v", removed, err)
			}
			after, err := os.ReadFile(filepath.Join(st.ArchiveDir(), f.Code+".json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("migration changed bytes: %v", err)
			}
			if _, err := os.Stat(live); !os.IsNotExist(err) {
				t.Fatal("migration left live file")
			}
			now = now.Add(24 * time.Hour)
			if removed, err := reaper.Sweep(); err != nil || len(removed) != 0 {
				t.Fatalf("exact boundary: %v, %v", removed, err)
			}
			now = now.Add(time.Second)
			if removed, err := reaper.Sweep(); err != nil || !reflect.DeepEqual(removed, []string{f.Code}) {
				t.Fatalf("expired legacy: %v, %v", removed, err)
			}
		})
	}
}

func TestArchiveRetentionCapsOldestFirst(t *testing.T) {
	for _, cap := range []string{"age", "count", "bytes", "both"} {
		t.Run(cap, func(t *testing.T) {
			st := newStore(t)
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			codes := []string{"CAPXXA", "CAPXXB", "CAPXXC"}
			var sizes []int64
			for i, code := range codes {
				// Updated is deliberately newer: expiry/order must use ArchivedAt.
				f := store.New(code, store.ModeOnline, engine.Options{}, 2, now)
				at := now.Add(-time.Duration(3-i) * 24 * time.Hour)
				f.Status, f.ArchivedAt = store.StatusArchived, &at
				if err := st.Save(f); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(filepath.Join(st.ArchiveDir(), code+".json"))
				if err != nil {
					t.Fatal(err)
				}
				sizes = append(sizes, info.Size())
			}
			reaper := store.NewReaper(st, func() time.Time { return now })
			if reaper.ArchiveAfter != 30*24*time.Hour || reaper.ArchiveMaxFiles != 1000 || reaper.ArchiveMaxBytes != 64<<20 {
				t.Fatal("wrong archive defaults")
			}
			switch cap {
			case "age":
				reaper.ArchiveAfter = 2 * 24 * time.Hour
			case "count":
				reaper.ArchiveMaxFiles = 2
			case "bytes":
				reaper.ArchiveMaxBytes = sizes[1] + sizes[2]
			case "both":
				reaper.ArchiveMaxFiles = 2
				reaper.ArchiveMaxBytes = sizes[2]
			}
			want := codes[:1]
			if cap == "both" {
				want = codes[:2]
			}
			removed, err := reaper.Sweep()
			if err != nil || !reflect.DeepEqual(removed, want) {
				t.Fatalf("removed %v, %v; want %v", removed, err, want)
			}
			if again, err := reaper.Sweep(); err != nil || len(again) != 0 {
				t.Fatalf("repeat sweep: %v, %v", again, err)
			}
		})
	}
}

func TestArchiveMoveFailureIsRetryableAndDoesNotOverwrite(t *testing.T) {
	st := newStore(t)
	f := store.New("MVERRX", store.ModeOnline, engine.Options{}, 2, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, f.Code, func(raw map[string]any) { raw["rules"] = engine.RulesVersion + 1 })
	// An existing, different diagnostic record must never be overwritten.
	if err := os.MkdirAll(st.ArchiveDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(st.ArchiveDir(), f.Code+".json")
	foreign := []byte("preserve diagnostic record")
	if err := os.WriteFile(archive, foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _, err := st.Load(f.Code); got != nil || !errors.Is(err, store.ErrRestoreWrite) {
		t.Fatalf("move succeeded: %+v, %v", got, err)
	}
	before, err := os.ReadFile(filepath.Join(st.Dir(), f.Code+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var archived store.File
	if err := json.Unmarshal(before, &archived); err != nil || archived.Status != store.StatusArchived {
		t.Fatalf("failed move lost marker: %v", err)
	}
	if after, err := os.ReadFile(archive); err != nil || !bytes.Equal(foreign, after) {
		t.Fatal("overwrote archive")
	}
	if err := st.Save(f); !errors.Is(err, store.ErrArchived) {
		t.Fatalf("reopened pending archive: %v", err)
	}
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Load(f.Code); !errors.Is(err, store.ErrArchived) {
		t.Fatalf("retry: %v", err)
	}
	if after, err := os.ReadFile(archive); err != nil || !bytes.Equal(before, after) {
		t.Fatal("retry changed archive time/bytes")
	}
}

func TestReaperPreservesForeignArchiveFiles(t *testing.T) {
	st := newStore(t)
	if err := os.MkdirAll(st.ArchiveDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"stray.json": []byte("{"), "BRKXXX.json": []byte("{"), "SCHEMA.json": []byte(`{"schema":999,"code":"SCHEMA","status":"archived"}`), "FRESHX.json": []byte(`{"schema":1,"code":"FRESHX","status":"playing"}`)}
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(st.ArchiveDir(), name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reaper := store.NewReaper(st, time.Now)
	reaper.ArchiveMaxFiles, reaper.ArchiveMaxBytes = 0, 0
	if removed, err := reaper.Sweep(); !errors.Is(err, store.ErrSchema) || len(removed) != 0 {
		t.Fatalf("foreign preservation/report: %v, %v", removed, err)
	}
	for name, before := range files {
		if after, err := os.ReadFile(filepath.Join(st.ArchiveDir(), name)); err != nil || !bytes.Equal(before, after) {
			t.Fatalf("changed %s", name)
		}
	}
	if _, _, err := st.Load("FRESHX"); err == nil {
		t.Fatal("played misplaced archive-folder file")
	}
}

func TestConcurrentArchiveLoadsAndSweeps(t *testing.T) {
	st := newStore(t)
	f := store.New("RACEXX", store.ModeOnline, engine.Options{}, 2, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, f.Code, func(raw map[string]any) { raw["rules"] = engine.RulesVersion + 1 })
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				if _, _, err := st.Load(f.Code); !errors.Is(err, store.ErrArchived) {
					t.Errorf("load: %v", err)
				}
			} else {
				if _, err := store.NewReaper(st, time.Now).Sweep(); err != nil {
					t.Errorf("sweep: %v", err)
				}
			}
		}()
	}
	wg.Wait()
	if codes, err := st.Codes(); err != nil || len(codes) != 0 {
		t.Fatalf("live scan after concurrent migration: %v, %v", codes, err)
	}
}

func TestArchiveInterruptedMoveReconcilesDuplicateLinks(t *testing.T) {
	st := newStore(t)
	f := store.New("DUPXXX", store.ModeOnline, engine.Options{}, 2, time.Now())
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, f.Code, func(raw map[string]any) { raw["status"] = store.StatusArchived })
	if err := os.MkdirAll(st.ArchiveDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(st.Dir(), f.Code+".json")
	archive := filepath.Join(st.ArchiveDir(), f.Code+".json")
	before, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate interruption after linking the durable record, before unlink.
	if err := os.Link(live, archive); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.Load(f.Code); !errors.Is(err, store.ErrArchived) {
		t.Fatal(err)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatal("duplicate live link remains")
	}
	if after, err := os.ReadFile(archive); err != nil || !bytes.Equal(before, after) {
		t.Fatal("interrupted move changed diagnostic record")
	}
}

// Promote the reviewer's collision probe to an assertion-based regression.
func TestSweepContinuesPastStuckArchive(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	clock := now.Add(-40 * 24 * time.Hour)
	st, err := store.OpenWithClock(t.TempDir(), func() time.Time { return clock })
	if err != nil {
		t.Fatal(err)
	}
	old := store.New("BBBARC", store.ModeOnline, engine.Options{}, 2, clock)
	if err := st.Save(old); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, old.Code, func(raw map[string]any) { raw["rules"] = engine.RulesVersion + 1 })
	if _, _, err := st.Load(old.Code); !errors.Is(err, store.ErrArchived) {
		t.Fatal(err)
	}
	idle := store.New("ZZZDDD", store.ModeOnline, engine.Options{}, 3, now.Add(-20*24*time.Hour))
	idle.Status = store.StatusPlaying
	if err := st.Save(idle); err != nil {
		t.Fatal(err)
	}
	stuck := store.New("AAACXX", store.ModeOnline, engine.Options{}, 4, now)
	if err := st.Save(stuck); err != nil {
		t.Fatal(err)
	}
	rewriteField(t, st, stuck.Code, func(raw map[string]any) { raw["status"] = store.StatusArchived })
	livePath := filepath.Join(st.Dir(), stuck.Code+".json")
	before, err := os.ReadFile(livePath)
	if err != nil {
		t.Fatal(err)
	}
	foreign := []byte(`{"schema":1}`)
	foreignPath := filepath.Join(st.ArchiveDir(), stuck.Code+".json")
	if err := os.WriteFile(foreignPath, foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	clock = now
	reaper := store.NewReaper(st, func() time.Time { return clock })
	removed, err := reaper.Sweep()
	if err == nil || !strings.Contains(err.Error(), stuck.Code) {
		t.Fatalf("missing collision error: %v", err)
	}
	if !reflect.DeepEqual(removed, []string{idle.Code, old.Code}) {
		t.Fatalf("skipped healthy cleanup: %v", removed)
	}
	for _, path := range []string{filepath.Join(st.Dir(), idle.Code+".json"), filepath.Join(st.ArchiveDir(), old.Code+".json")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("did not expire %s: %v", path, err)
		}
	}
	if !strings.Contains(logs.String(), "room=AAACXX") || !strings.Contains(logs.String(), "archive already exists") {
		t.Fatalf("missing actionable collision log: %s", &logs)
	}
	if after, err := os.ReadFile(livePath); err != nil || !bytes.Equal(before, after) {
		t.Fatal("changed stuck live record")
	}
	if after, err := os.ReadFile(foreignPath); err != nil || !bytes.Equal(foreign, after) {
		t.Fatal("changed foreign archive")
	}
	clock = clock.Add(time.Hour)
	if removed, err := reaper.Sweep(); err == nil || len(removed) != 0 {
		t.Fatalf("repeat sweep: %v, %v", removed, err)
	}
}

func TestSweepArchiveCapsContinueAfterReadFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission failure requires unprivileged user")
	}
	st := newStore(t)
	for _, code := range []string{"AAABAD", "BBBARC", "CCCARC"} {
		f := store.New(code, store.ModeOnline, engine.Options{}, 2, time.Now())
		f.Status = store.StatusArchived
		if err := st.Save(f); err != nil {
			t.Fatal(err)
		}
	}
	badPath := filepath.Join(st.ArchiveDir(), "AAABAD.json")
	before, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badPath, 0); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(badPath, 0o600) }()
	reaper := store.NewReaper(st, time.Now)
	reaper.ArchiveMaxFiles = 0
	removed, err := reaper.Sweep()
	if !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), "AAABAD") || !reflect.DeepEqual(removed, []string{"BBBARC", "CCCARC"}) {
		t.Fatalf("read failure blocked cap: %v, %v", removed, err)
	}
	if err := os.Chmod(badPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if after, err := os.ReadFile(badPath); err != nil || !bytes.Equal(before, after) {
		t.Fatal("modified unreadable archive")
	}
}

func TestSweepLiveListingFailureStillExpiresArchives(t *testing.T) {
	st := newStore(t)
	f := store.New("BBBARC", store.ModeOnline, engine.Options{}, 2, time.Now().Add(-40*24*time.Hour))
	f.Status = store.StatusArchived
	if err := st.Save(f); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(st.Dir()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(st.Dir(), []byte("blocked directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	removed, err := store.NewReaper(st, time.Now).Sweep()
	if err == nil || !strings.Contains(err.Error(), "list live rooms") || !reflect.DeepEqual(removed, []string{f.Code}) {
		t.Fatalf("listing failure blocked archive expiry: %v, %v", removed, err)
	}
}
