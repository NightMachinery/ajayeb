package store

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrRestoreWrite distinguishes a failed restore rewrite from a missing room,
// even when the filesystem reports not-exist for a vanished parent directory.
var ErrRestoreWrite = errors.New("store: restore write failed")

// Store is a handle on one <data>/rooms directory. The zero value is not
// valid; use Open or OpenWithClock.
type Store struct {
	dir string
	now Clock // stamps archive time when Load archives a room
}

// Open prepares <dataDir>/rooms, creating it (and dataDir) if neither exists
// yet. The store reads the wall clock (time.Now) for the one timestamp it
// sets itself, the Updated stamp of a room Load archives.
func Open(dataDir string) (*Store, error) {
	return OpenWithClock(dataDir, nil)
}

// OpenWithClock is Open with an injected clock; nil means time.Now.
func OpenWithClock(dataDir string, now Clock) (*Store, error) {
	if now == nil {
		now = time.Now
	}
	dir := filepath.Join(dataDir, "rooms")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("store: creating %s: %w", dir, err)
	}
	return &Store{dir: dir, now: now}, nil
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
// A file of another Schema is refused with an error wrapping ErrSchema and
// left exactly as it is: this build cannot read it faithfully, so it must
// not rewrite it either (a newer server's fields would be lost).
//
// Within the same RulesVersion, a complete legal replay is canonical. A differing
// hash is refreshed atomically without changing Updated. A version mismatch or
// failed replay is archived durably with a bounded reason and timestamp. Already
// archived rooms stay archived. A failed write is an error, never a successful load.
func (st *Store) Load(code string) (*File, engine.State, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, engine.State{}, err
	}
	f, data, err := readFileData(path)
	if err != nil {
		return nil, engine.State{}, fmt.Errorf("store: load %s: %w", code, err)
	}
	if err := checkSchema(f); err != nil {
		return nil, engine.State{}, fmt.Errorf("store: load %s: %w", code, err)
	}

	if f.Status == StatusArchived {
		return f, engine.State{}, &ArchivedError{Reason: SafeArchiveReason(f.ArchiveReason)}
	}
	s, reason, err := assess(f)
	if err != nil {
		return st.archive(path, f, data, reason, err)
	}
	if got := engine.HashString(&s); got != f.Hash {
		old := f.Hash
		refreshed := *f
		refreshed.Hash = got
		if err := rewriteFields(path, data, map[string]any{"hash": got}); err != nil {
			return nil, engine.State{}, fmt.Errorf("%w: refresh hash for %s: %w", ErrRestoreWrite, code, err)
		}
		slog.Warn("refreshed room replay hash", "room", code, "oldHash", old, "newHash", got)
		f = &refreshed
	}
	return f, s, nil
}

func (st *Store) archive(path string, f *File, data []byte, reason ArchiveReason, cause error) (*File, engine.State, error) {
	archived := *f
	archived.Status = StatusArchived
	archived.Updated = st.now()
	archived.ArchivedAt = &archived.Updated
	archived.ArchiveReason = reason
	if err := rewriteFields(path, data, map[string]any{"status": archived.Status, "updated": archived.Updated,
		"archiveReason": reason, "archivedAt": archived.ArchivedAt}); err != nil {
		return nil, engine.State{}, fmt.Errorf("%w: archiving failed: %w (replay: %v)", ErrRestoreWrite, err, cause)
	}
	return &archived, engine.State{}, &ArchivedError{Reason: reason, Cause: cause}
}

// checkSchema refuses a file whose Schema is not the one this build writes.
func checkSchema(f *File) error {
	if f.Schema != Schema {
		return fmt.Errorf("%w: the file has schema %d, this server reads %d", ErrSchema, f.Schema, Schema)
	}
	return nil
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
// way; only Load vouches for the state matching Hash. A file of another
// Schema is refused (ErrSchema), as Load refuses it, because its header
// fields may not mean what this build thinks.
func (st *Store) PeekHeader(code string) (*File, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, err
	}
	f, err := readFile(path)
	if err == nil {
		err = checkSchema(f)
	}
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
