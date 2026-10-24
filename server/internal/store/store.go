package store

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrRestoreWrite distinguishes a failed restore rewrite from a missing room,
// even when the filesystem reports not-exist for a vanished parent directory.
var ErrRestoreWrite = errors.New("store: restore write failed")

// ErrExists answers Create and CreateGame when a file already holds the
// code: rooms of every kind share one namespace of codes, and a room's
// first write never replaces another room.
var ErrExists = errors.New("store: the room code is taken")

// Store handles the live and archived room directories under <data>.
// The zero value is not valid; use Open or OpenWithClock.
type Store struct {
	dir     string
	codecs  *Codecs
	now     Clock // stamps archive time when Load archives a room
	locksMu sync.Mutex
	locks   map[string]*roomLock
	sweepMu sync.Mutex // serialize cleanup plans, not ordinary room operations
}

// Open prepares <dataDir>/rooms, creating it (and dataDir) if neither exists
// yet. The store reads the wall clock (time.Now) for the one timestamp it
// sets itself, the Updated stamp of a room Load archives.
func Open(dataDir string) (*Store, error) {
	return OpenWithClock(dataDir, nil)
}

// OpenWithClock is Open with an injected clock; nil means time.Now.
func OpenWithClock(dataDir string, now Clock) (*Store, error) {
	return OpenWithCodecs(dataDir, now, nil)
}

// OpenWithCodecs injects the same immutable table used by preflight and retention.
// nil selects the production table (schema-1 Duel only). Load, PeekHeader and
// Save remain typed Duel APIs; other games keep their own restoration paths.
func OpenWithCodecs(dataDir string, now Clock, codecs *Codecs) (*Store, error) {
	if codecs == nil {
		codecs = productionCodecs
	}
	if now == nil {
		now = time.Now
	}
	dir := filepath.Join(dataDir, "rooms")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("store: creating %s: %w", dir, err)
	}
	return &Store{dir: dir, codecs: codecs, now: now, locks: make(map[string]*roomLock)}, nil
}

// Dir returns the rooms directory this Store reads and writes, mainly for
// logging and for tests that want to inspect the filesystem directly.
func (st *Store) Dir() string { return st.dir }

// ArchiveDir returns the diagnostic archive directory, created on first archive.
func (st *Store) ArchiveDir() string { return filepath.Join(filepath.Dir(st.dir), "rooms-archive") }

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
	return st.save(f, false, "")
}

// Create writes f as its code's first file, as Save does, except that the
// check that no file holds the code (live or archived, of any schema or
// game, damaged or not) and the write are one step under the code's lock.
// When a file holds it, Create writes nothing and returns an error
// wrapping ErrExists, so two creators that drew the same code cannot
// overwrite each other's rooms, whatever their kinds.
func (st *Store) Create(f *File) error {
	return st.save(f, true, "")
}

// ReplaceDuel writes the fully assigned Duel game in place of a clean live
// schema-2 lobby of from, atomically under the code's lock. Unlike Save's
// damaged-file repair, Start never replaces missing, damaged or started games.
func (st *Store) ReplaceDuel(from string, f *File) error {
	if from == "" || from == "duel" || f.Status != StatusPlaying {
		return fmt.Errorf("store: replace %s: %w: invalid Duel handoff", f.Code, ErrGame)
	}
	return st.save(f, false, from)
}

func (st *Store) save(f *File, create bool, from string) error {
	if err := checkSchema(f); err != nil {
		return err
	}
	path, err := st.path(f.Code)
	if err != nil {
		return err
	}
	unlock := st.lockRoom(f.Code)
	defer unlock()
	if from != "" {
		if _, err := os.Stat(st.archivePath(f.Code)); err == nil {
			return fmt.Errorf("store: replace %s: %w", f.Code, ErrArchived)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		old, data, err := st.codecs.read(path)
		if err != nil {
			return fmt.Errorf("store: replace %s: %w", f.Code, err)
		}
		schema, game, err := fileKind(data)
		if err != nil || schema != 2 || game != from || old.Retention().Code != f.Code {
			return fmt.Errorf("store: replace %s: %w: not this lobby", f.Code, ErrGame)
		}
		if old.Retention().Status == StatusArchived {
			return fmt.Errorf("store: replace %s: %w", f.Code, ErrArchived)
		}
		if old.Retention().Status != StatusLobby {
			return fmt.Errorf("store: replace %s: %w: not a live lobby", f.Code, ErrGame)
		}
		if _, err := old.Check(); err != nil {
			return fmt.Errorf("store: replace %s: %w", f.Code, err)
		}
	} else if create {
		if err := st.vacant(f.Code, path); err != nil {
			return err
		}
	} else if _, err := os.Stat(st.archivePath(f.Code)); err == nil {
		return fmt.Errorf("store: save %s: %w", f.Code, ErrArchived)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	} else if existing, err := readFile(path); err == nil {
		if existing.Status == StatusArchived {
			return fmt.Errorf("store: save %s: %w", f.Code, ErrArchived)
		}
	} else if foreign, ferr := foreignRoomFile(path); ferr == nil && foreign {
		return fmt.Errorf("store: save %s: %w", f.Code, err)
	}
	if !ValidHintAdmins(f.HintAdmins) {
		return fmt.Errorf("store: save %s: invalid hint admins", f.Code)
	}
	s, err := replay(f)
	if err != nil {
		return fmt.Errorf("store: save %s: %w", f.Code, err)
	}
	f.Hash = engine.HashString(&s)
	if err := writeFile(path, f); err != nil {
		return err
	}
	if f.Status == StatusArchived {
		return st.moveArchive(path, f.Code)
	}
	return nil
}

// Load reads the room at code, replays it and returns both the header and
// the resulting engine.State.
//
// An unsupported schema/game is refused before Duel decoding with an error
// wrapping ErrSchema (and ErrGame for a game/codec), and left exactly as it is:
// this build cannot read it faithfully, so it must
// not rewrite it either (a newer server's fields would be lost).
//
// Within the same RulesVersion, a complete legal replay is canonical. A differing
// hash is refreshed atomically without changing Updated. A version mismatch or
// failed replay is archived durably with a bounded reason and timestamp. Already
// archived rooms stay archived. A failed write is an error, never a successful load.
func (st *Store) Load(code string) (*File, engine.State, error) {
	if err := ValidateCode(code); err != nil {
		return nil, engine.State{}, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	f, data, path, err := st.readRoom(code)
	if err != nil {
		return nil, engine.State{}, fmt.Errorf("store: load %s: %w", code, err)
	}
	if f.Status == StatusArchived {
		if path != st.archivePath(code) {
			if err := st.moveArchive(path, code); err != nil {
				return nil, engine.State{}, fmt.Errorf("%w: moving archive %s: %w", ErrRestoreWrite, code, err)
			}
		}
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
	if err := st.moveArchive(path, f.Code); err != nil {
		return nil, engine.State{}, fmt.Errorf("%w: moving archive %s: %w", ErrRestoreWrite, f.Code, err)
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

// vacant requires code's lock: it answers ErrExists when a file holds
// code in the live folder (path) or the archive.
func (st *Store) vacant(code, path string) error {
	for _, candidate := range []string{path, st.archivePath(code)} {
		if _, err := os.Lstat(candidate); err == nil {
			return fmt.Errorf("store: create %s: %w", code, ErrExists)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("store: create %s: %w", code, err)
		}
	}
	return nil
}

// Exists reports whether a room file exists for code.
func (st *Store) Exists(code string) (bool, error) {
	path, err := st.path(code)
	if err != nil {
		return false, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	for _, candidate := range []string{path, st.archivePath(code)} {
		if _, err := os.Stat(candidate); err == nil {
			return true, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("store: checking %s: %w", code, err)
		}
	}
	return false, nil
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
	if err := ValidateCode(code); err != nil {
		return nil, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	f, _, _, err := st.readRoom(code)
	return f, err
}

// Codes lists live-directory codes only; diagnostic archives do not join the scan.
func (st *Store) Codes() ([]string, error) { return roomCodes(st.dir) }

func roomCodes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: listing %s: %w", dir, err)
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
	unlock := st.lockRoom(code)
	defer unlock()
	for _, candidate := range []string{path, st.archivePath(code)} {
		if err := os.Remove(candidate); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("store: deleting %s: %w", code, err)
		}
	}
	return nil
}
