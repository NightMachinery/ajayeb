package store

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

func TestArchivePruningContinuesAfterRemoveFailureWithConservativeTotals(t *testing.T) {
	for _, cap := range []string{"age", "count", "bytes"} {
		t.Run(cap, func(t *testing.T) {
			st, err := Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			var entries []archiveEntry
			var total int64
			for i, code := range []string{"AAACXX", "BBBARC", "CCCARC"} {
				f := New(code, ModeOnline, engine.Options{}, 2, now.Add(-time.Duration(3-i)*time.Hour))
				f.Status = StatusArchived
				if err := st.Save(f); err != nil {
					t.Fatal(err)
				}
				entry, ok, err := NewReaper(st, nil).archiveEntry(code)
				if err != nil || !ok {
					t.Fatalf("archive setup: %v, %v", ok, err)
				}
				entries = append(entries, entry)
				total += entry.size
			}
			reaper := NewReaper(st, nil)
			switch cap {
			case "age":
				reaper.ArchiveAfter = 0
			case "count":
				reaper.ArchiveMaxFiles = 1
			case "bytes":
				reaper.ArchiveMaxBytes = entries[0].size
			}
			failure := errors.New("injected remove failure")
			removed, err := reaper.pruneEntries(now, entries, total, func(path string) (bool, error) {
				if filepath.Base(path) == "AAACXX.json" {
					return false, failure
				}
				return removeRoomFile(path)
			})
			if !errors.Is(err, failure) || !reflect.DeepEqual(removed, []string{"BBBARC", "CCCARC"}) {
				t.Fatalf("removal failure blocked pruning or released its budget: %v, %v", removed, err)
			}
			if _, err := os.Stat(st.archivePath("AAACXX")); err != nil {
				t.Fatalf("failed deletion lost record: %v", err)
			}
			for _, code := range removed {
				if _, err := os.Stat(st.archivePath(code)); !os.IsNotExist(err) {
					t.Fatalf("not removed: %s, %v", code, err)
				}
			}
		})
	}
}
