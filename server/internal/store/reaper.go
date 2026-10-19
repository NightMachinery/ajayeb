package store

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Clock returns the current time. Production code passes time.Now; tests
// use a fixed or stepped value without sleeping.
type Clock func() time.Time

// Reaper expires live rooms and bounds diagnostic archives. The zero value
// is not valid; use NewReaper. Configure limits before the first Sweep.
type Reaper struct {
	st  *Store
	now Clock

	FinishedAfter   time.Duration
	IdleAfter       time.Duration
	ArchiveAfter    time.Duration
	ArchiveMaxFiles int
	ArchiveMaxBytes int64
}

// NewReaper uses 7 days for finished rooms, 14 days for idle rooms and
// 30 days for archives, capped at 1000 files or 64 MiB. nil means time.Now.
func NewReaper(st *Store, now Clock) *Reaper {
	if now == nil {
		now = time.Now
	}
	return &Reaper{
		st: st, now: now,
		FinishedAfter:   7 * 24 * time.Hour,
		IdleAfter:       14 * 24 * time.Hour,
		ArchiveAfter:    30 * 24 * time.Hour,
		ArchiveMaxFiles: 1000,
		ArchiveMaxBytes: 64 << 20,
	}
}

// Sweep runs at startup and hourly. It migrates legacy archives without
// changing their bytes, expires live rooms by Updated, then prunes archives
// by ArchivedAt (Updated for legacy records), oldest first until both caps
// hold. Repeated reads do not extend retention. Malformed/unknown-schema
// records and unrelated files are preserved, outside the managed caps.
// It reads headers only and never replays a game. Failures are logged and
// joined in the returned error; one bad file never stops other cleanup.
func (r *Reaper) Sweep() ([]string, error) {
	r.st.sweepMu.Lock()
	defer r.st.sweepMu.Unlock()
	codes, err := r.st.Codes()
	var failures error
	if err != nil {
		failures = reapError("", "list live rooms", err)
	}
	now := r.now()
	var removed []string
	for _, code := range codes {
		deleted, err := r.sweepLive(code, now)
		if err != nil {
			failures = errors.Join(failures, reapError(code, "expire or migrate live room", err))
		}
		if deleted {
			removed = append(removed, code)
		}
	}
	archived, err := r.pruneArchives(now)
	return append(removed, archived...), errors.Join(failures, err)
}

func (r *Reaper) sweepLive(code string, now time.Time) (bool, error) {
	unlock := r.st.lockRoom(code)
	defer unlock()
	path := filepath.Join(r.st.dir, code+".json")
	f, err := readFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := checkSchema(f); err != nil {
		return false, err
	}
	if f.Code != code {
		return false, fmt.Errorf("store: mismatched room header for %s", code)
	}
	if f.Status == StatusArchived {
		return false, r.st.moveArchive(path, code)
	}
	age := now.Sub(f.Updated)
	eligible := (f.Status == StatusFinished && age > r.FinishedAfter) ||
		((f.Status == StatusLobby || f.Status == StatusPlaying) && age > r.IdleAfter)
	if !eligible {
		return false, nil
	}
	return removeRoomFile(path)
}

type archiveEntry struct {
	code string
	at   time.Time
	size int64
}

func (r *Reaper) pruneArchives(now time.Time) ([]string, error) {
	codes, err := roomCodes(r.st.ArchiveDir())
	if err != nil {
		return nil, reapError("", "list archived rooms", err)
	}
	var failures error
	var entries []archiveEntry
	var total int64
	for _, code := range codes {
		entry, ok, err := r.archiveEntry(code)
		if err != nil {
			failures = errors.Join(failures, reapError(code, "read archived room", err))
			continue
		}
		if ok {
			entries = append(entries, entry)
			total += entry.size
		}
	}
	removed, err := r.pruneEntries(now, entries, total, removeRoomFile)
	return removed, errors.Join(failures, err)
}

// Keep scanning oldest-first after a failed removal, counting that file's
// bytes until an actual deletion (or confirmed absence) releases the space.
func (r *Reaper) pruneEntries(now time.Time, entries []archiveEntry, total int64, remove func(string) (bool, error)) ([]string, error) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].at.Equal(entries[j].at) {
			return entries[i].code < entries[j].code
		}
		return entries[i].at.Before(entries[j].at)
	})
	count := len(entries)
	var failures error
	var removed []string
	for _, entry := range entries {
		if now.Sub(entry.at) <= r.ArchiveAfter && count <= r.ArchiveMaxFiles && total <= r.ArchiveMaxBytes {
			break
		}
		unlock := r.st.lockRoom(entry.code)
		deleted, err := remove(r.st.archivePath(entry.code))
		unlock()
		if err != nil {
			failures = errors.Join(failures, reapError(entry.code, "remove archived room", err))
			continue
		}
		count--
		total -= entry.size
		if deleted {
			removed = append(removed, entry.code)
		}
	}
	return removed, failures
}

func (r *Reaper) archiveEntry(code string) (archiveEntry, bool, error) {
	unlock := r.st.lockRoom(code)
	defer unlock()
	path := r.st.archivePath(code)
	f, data, err := readFileData(path)
	if errors.Is(err, fs.ErrNotExist) {
		return archiveEntry{}, false, nil
	}
	if err != nil {
		return archiveEntry{}, false, err
	}
	if err := checkSchema(f); err != nil {
		return archiveEntry{}, false, err
	}
	if f.Code != code || f.Status != StatusArchived {
		return archiveEntry{}, false, fmt.Errorf("store: invalid archive header for %s", code)
	}
	at := f.Updated
	if f.ArchivedAt != nil {
		at = *f.ArchivedAt
	}
	return archiveEntry{code: code, at: at, size: int64(len(data))}, true, nil
}

func reapError(code, operation string, err error) error {
	slog.Warn("room cleanup failed", "room", code, "operation", operation, "err", err)
	return fmt.Errorf("store: reaper %s %s: %w", operation, code, err)
}

func removeRoomFile(path string) (bool, error) {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}
