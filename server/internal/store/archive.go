package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type roomLock struct {
	mu   sync.Mutex
	refs int
}

// Keep moves, restores, saves and reaping of one code mutually exclusive.
// Reference counting bounds the lock map without serializing different rooms.
func (st *Store) lockRoom(code string) func() {
	st.locksMu.Lock()
	l := st.locks[code]
	if l == nil {
		l = &roomLock{}
		st.locks[code] = l
	}
	l.refs++
	st.locksMu.Unlock()
	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		st.locksMu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(st.locks, code)
		}
		st.locksMu.Unlock()
	}
}

func (st *Store) archivePath(code string) string {
	return filepath.Join(st.ArchiveDir(), code+".json")
}

// readRoom requires a validated code and its lock. Archive-folder contents can
// never become playable, even if a misplaced file has a live status.
func (st *Store) readRoom(code string) (*File, []byte, string, error) {
	path := filepath.Join(st.dir, code+".json")
	f, data, err := readFileData(path)
	if errors.Is(err, fs.ErrNotExist) {
		path = st.archivePath(code)
		f, data, err = readFileData(path)
	}
	if err == nil {
		err = checkSchema(f)
	}
	if err == nil && (f.Code != code || (path == st.archivePath(code) && f.Status != StatusArchived)) {
		err = fmt.Errorf("store: invalid room header for %s", code)
	}
	return f, data, path, err
}

// The archive marker must already be durable at source. Link before unlink
// avoids overwriting an existing diagnostic record and leaves a recoverable
// duplicate if interrupted. Both directories must be on the same filesystem.
func (st *Store) moveArchive(source, code string) error {
	if err := os.MkdirAll(st.ArchiveDir(), 0o700); err != nil {
		return err
	}
	// A newly created archive directory must itself survive a crash before
	// the source link is removed, not just the file entry inside it.
	if err := syncDir(filepath.Dir(st.ArchiveDir())); err != nil {
		return err
	}
	target := st.archivePath(code)
	if err := os.Link(source, target); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		a, aerr := os.Stat(source)
		b, berr := os.Stat(target)
		if aerr != nil || berr != nil || !os.SameFile(a, b) {
			return fmt.Errorf("store: archive already exists for %s: %w", code, err)
		}
	}
	// Persist the new directory entry before removing the old one.
	if err := syncDir(st.ArchiveDir()); err != nil {
		return err
	}
	if err := os.Remove(source); err != nil {
		return err
	}
	return syncDir(st.dir)
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}
