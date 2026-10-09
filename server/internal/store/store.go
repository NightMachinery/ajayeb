package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// Store is a handle on one <data>/rooms directory. The zero value is not
// valid; use Open.
type Store struct {
	dir string
}

// Open prepares <dataDir>/rooms, creating it (and dataDir) if neither exists
// yet.
func Open(dataDir string) (*Store, error) {
	dir := filepath.Join(dataDir, "rooms")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("store: creating %s: %w", dir, err)
	}
	return &Store{dir: dir}, nil
}

// Dir returns the rooms directory this Store reads and writes, mainly for
// logging and for tests that want to inspect the filesystem directly.
func (st *Store) Dir() string { return st.dir }

// path turns a room code into the room file's path, after validating the
// code (ValidateCode): the one checkpoint that keeps an untrusted code from
// ever reaching the filesystem as anything but a plain CodeLength-character
// name.
func (st *Store) path(code string) (string, error) {
	if err := ValidateCode(code); err != nil {
		return "", err
	}
	return filepath.Join(st.dir, code+".json"), nil
}

// Save writes f: it first replays f's seed, scenario, options and actions
// through the engine (exactly as Load would) to compute a fresh f.Hash, then
// writes the file atomically. An action log that does not replay is an
// error, and nothing is written; Save can only produce a file Load will
// later trust.
//
// Callers normally call Save after every accepted action (the room actor's
// "write after every accepted action" rule in docs/architecture.md): replay
// cost is O(actions so far), which stays small for one game of 7 Wonders
// Duel.
func (st *Store) Save(f *File) error {
	path, err := st.path(f.Code)
	if err != nil {
		return err
	}
	s, err := replay(f)
	if err != nil {
		return fmt.Errorf("store: save %s: %w", f.Code, err)
	}
	f.Hash = engine.HashString(&s)
	return writeFile(path, f)
}

// Load reads the room at code, replays it and returns both the header and
// the resulting engine.State.
//
// On a rules-version mismatch, a replay failure or a hash mismatch, Load
// does not return the state: it instead flips the file's Status to
// StatusArchived, writes that back (best effort; a write failure here is
// folded into the returned error but does not change the fact that the room
// is no longer trusted), and returns the archived header alongside a
// descriptive error wrapping ErrRulesVersion or ErrHashMismatch. The file on
// disk is never deleted by Load, so it stays available for a bug report.
func (st *Store) Load(code string) (*File, engine.State, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, engine.State{}, err
	}
	f, err := readFile(path)
	if err != nil {
		return nil, engine.State{}, fmt.Errorf("store: load %s: %w", code, err)
	}

	if f.Rules != engine.RulesVersion {
		loadErr := fmt.Errorf("store: load %s: %w: file plays rules %d, this engine plays %d",
			code, ErrRulesVersion, f.Rules, engine.RulesVersion)
		return st.archive(path, f, loadErr)
	}

	s, err := replay(f)
	if err != nil {
		return st.archive(path, f, fmt.Errorf("store: load %s: %w", code, err))
	}
	if got := engine.HashString(&s); got != f.Hash {
		loadErr := fmt.Errorf("store: load %s: %w: replay hash %s, file says %s", code, ErrHashMismatch, got, f.Hash)
		return st.archive(path, f, loadErr)
	}
	return f, s, nil
}

// archive flips f to StatusArchived, writes it back (best effort) and
// returns it alongside loadErr, as Load's failure paths do.
func (st *Store) archive(path string, f *File, loadErr error) (*File, engine.State, error) {
	archived := *f
	archived.Status = StatusArchived
	if err := writeFile(path, &archived); err != nil {
		loadErr = fmt.Errorf("%w (and archiving it failed: %v)", loadErr, err)
	}
	return &archived, engine.State{}, loadErr
}

// Exists reports whether a room file exists for code.
func (st *Store) Exists(code string) (bool, error) {
	path, err := st.path(code)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("store: checking %s: %w", code, err)
}

// NewCode returns a fresh room code with no existing file, retrying
// GenerateCode on the rare collision. It gives up after a generous number of
// attempts, which only a broken RNG or a near-exhausted code space would
// ever exhaust.
func (st *Store) NewCode() (string, error) {
	const maxAttempts = 50
	for range maxAttempts {
		code, err := GenerateCode()
		if err != nil {
			return "", err
		}
		exists, err := st.Exists(code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("store: could not find a free room code in %d attempts", maxAttempts)
}

// PeekHeader reads code's file with no replay or hash check: just the
// header fields (status, timestamps and so on), fast, for callers that do
// not need a playable state, such as the reaper's sweep or a room listing.
// An untrusted (for example StatusArchived) file is still readable this
// way; only Load vouches for the state matching Hash.
func (st *Store) PeekHeader(code string) (*File, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, err
	}
	f, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("store: peek %s: %w", code, err)
	}
	return f, nil
}

// Codes lists every room code with a file on disk, in no particular order.
func (st *Store) Codes() ([]string, error) {
	entries, err := os.ReadDir(st.dir)
	if err != nil {
		return nil, fmt.Errorf("store: listing %s: %w", st.dir, err)
	}
	codes := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok {
			continue // a leftover temp file (.room-*.tmp) or something else entirely
		}
		if ValidateCode(name) != nil {
			continue
		}
		codes = append(codes, name)
	}
	return codes, nil
}

// Delete removes code's room file. It is not an error for the file to
// already be gone.
func (st *Store) Delete(code string) error {
	path, err := st.path(code)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("store: deleting %s: %w", code, err)
	}
	return nil
}
