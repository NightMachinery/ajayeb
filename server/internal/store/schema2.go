package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
)

// This file holds the game-agnostic file operations of schema-2 rooms. Each
// game's codec owns its record and its restoration; the store owns the
// per-code lock, the atomic write, the hash-only refresh and the durable
// archive (D28, D29), exactly as for schema-1 Duel. A schema-2 record names
// its common header fields as Duel's do: "code", "status", "updated",
// "archivedAt" and "archiveReason".

// Assessment is a schema-2 codec's verdict on a live record at load. A zero
// Assessment means playable as stored.
type Assessment struct {
	// Refresh holds top-level fields to rewrite in place, such as a hash
	// that a compatible replay recomputed. Every other field is preserved,
	// including fields this build does not know, and so is "updated".
	Refresh map[string]any
	// Archive, when set, archives the record durably with this reason;
	// Cause is the private detail for the error and the server log.
	Archive ArchiveReason
	Cause   error
}

// LoadGame reads code's schema-2 record of game under the code's lock and
// returns its bytes once assess has accepted them, after a refresh if it
// asked for one. An archived record returns an *ArchivedError (a live-folder
// archive is first moved, as Load does); a record assess archives is marked
// and moved durably first. A file of another schema or game, or one its
// codec cannot decode, is refused and left untouched, byte for byte. So is
// a file that the refresh or the archive mark would take over MaxFileBytes:
// the error wraps ErrRestoreWrite and ErrTooLarge.
func (st *Store) LoadGame(code, game string, assess func(data []byte) (Assessment, error)) ([]byte, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	data, path, archived, err := st.readGame(code, path, game)
	if err != nil {
		return nil, fmt.Errorf("store: load %s: %w", code, err)
	}
	if archived {
		if path != st.archivePath(code) {
			if err := st.moveArchive(path, code); err != nil {
				return nil, fmt.Errorf("%w: moving archive %s: %w", ErrRestoreWrite, code, err)
			}
		}
		var meta struct {
			ArchiveReason ArchiveReason `json:"archiveReason"`
		}
		_ = json.Unmarshal(data, &meta)
		return nil, &ArchivedError{Reason: SafeArchiveReason(meta.ArchiveReason)}
	}
	verdict, err := assess(data)
	if err != nil {
		return nil, fmt.Errorf("store: load %s: %w", code, err)
	}
	if verdict.Archive != "" {
		now := st.now()
		marked, err := patchFields(data, map[string]any{"status": StatusArchived, "updated": now,
			"archiveReason": verdict.Archive, "archivedAt": now})
		if err == nil {
			err = writeBytes(path, marked)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: archiving failed: %w (replay: %v)", ErrRestoreWrite, err, verdict.Cause)
		}
		if err := st.moveArchive(path, code); err != nil {
			return nil, fmt.Errorf("%w: moving archive %s: %w", ErrRestoreWrite, code, err)
		}
		return nil, &ArchivedError{Reason: verdict.Archive, Cause: verdict.Cause}
	}
	if len(verdict.Refresh) > 0 {
		refreshed, err := patchFields(data, verdict.Refresh)
		if err == nil {
			err = writeBytes(path, refreshed)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: refresh for %s: %w", ErrRestoreWrite, code, err)
		}
		// No hash values in the log: a schema-2 game's hash covers hidden cards.
		slog.Warn("refreshed room replay hash", "room", code, "game", game)
		data = refreshed
	}
	return data, nil
}

// SaveGame atomically writes code's schema-2 record of game. The bytes must
// decode through the game's registered codec as that code, so the store
// never writes a file it cannot read back. It refuses to overwrite an
// archived room or a cleanly read file of another schema or game, an
// unknown schema included; damaged bytes stay overwritable, as for Duel's
// Save. A record whose status is
// archived is moved to the archive folder after it is written.
func (st *Store) SaveGame(code, game string, data []byte) error {
	return st.saveGame(code, game, data, "")
}

// ReplaceGame atomically writes code's schema-2 record of game in place of
// its live record of another game, from: the shared lobby's Start, which
// writes the started game under the lobby's code. It checks data as
// SaveGame does, and refuses unless a cleanly read, unarchived record of
// from is in place, so a Start can never overwrite a game.
func (st *Store) ReplaceGame(code, from, game string, data []byte) error {
	if from == "" || from == game {
		return fmt.Errorf("store: replace %s: %w: nothing to replace", code, ErrGame)
	}
	return st.saveGame(code, game, data, from)
}

// CreateGame atomically writes data as code's first record of game. It
// checks data as SaveGame does; then, under the code's lock, the check that
// no file holds the code (live or archived, of any schema or game, damaged
// or not) and the write are one step, so two creators that drew the same
// code cannot overwrite each other's rooms, whatever their kinds. When a
// file holds it, CreateGame writes nothing and returns an error wrapping
// ErrExists, with the live file's bytes as read under that lock (nil for
// an archived room or an unreadable file): a caller retrying its own
// creation can tell the room it made from another.
func (st *Store) CreateGame(code, game string, data []byte) ([]byte, error) {
	path, record, err := st.checkGame(code, game, data)
	if err != nil {
		return nil, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	if err := st.vacant(code, path); err != nil {
		if !errors.Is(err, ErrExists) {
			return nil, err
		}
		existing, rerr := readCapped(path)
		if rerr != nil {
			existing = nil
		}
		return existing, err
	}
	if err := writeBytes(path, data); err != nil {
		return nil, err
	}
	if record.Retention().Status == StatusArchived {
		return nil, st.moveArchive(path, code)
	}
	return nil, nil
}

// checkGame validates code and checks that data fits the read cap and
// decodes through game's codec as a record of code, so the store never
// writes a file it cannot read back. A record over the cap is refused with
// ErrTooLarge before anything is read or locked; the file in place stays
// as it was.
func (st *Store) checkGame(code, game string, data []byte) (string, Inspection, error) {
	path, err := st.path(code)
	if err != nil {
		return "", nil, err
	}
	if len(data) > MaxFileBytes {
		return "", nil, fmt.Errorf("store: save %s: %w: %d bytes, at most %d are read", code, ErrTooLarge, len(data), MaxFileBytes)
	}
	record, err := st.codecs.decode(data)
	if err != nil {
		return "", nil, fmt.Errorf("store: save %s: %w", code, err)
	}
	if schema, g, _ := fileKind(data); schema != 2 || g != game || record.Retention().Code != code {
		return "", nil, fmt.Errorf("store: save %s: the record is not a %s room of this code", code, game)
	}
	return path, record, nil
}

func (st *Store) saveGame(code, game string, data []byte, from string) error {
	path, record, err := st.checkGame(code, game, data)
	if err != nil {
		return err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	if _, err := os.Stat(st.archivePath(code)); err == nil {
		return fmt.Errorf("store: save %s: %w", code, ErrArchived)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if existing, err := readCapped(path); err == nil {
		schema, g, kerr := fileKind(existing)
		switch {
		case from != "":
			if kerr != nil || schema != 2 || g != from {
				return fmt.Errorf("store: replace %s: %w: no %s record is in place", code, ErrGame, from)
			}
			old, derr := st.codecs.decode(existing)
			if derr != nil || old.Retention().Code != code {
				return fmt.Errorf("store: replace %s: %w: the %s record does not decode", code, ErrGame, from)
			}
			if old.Retention().Status == StatusArchived {
				return fmt.Errorf("store: replace %s: %w", code, ErrArchived)
			}
		case errors.Is(kerr, ErrSchema):
			// A clean header of a schema this build does not know: a newer
			// server's room, which must not be lost.
			return fmt.Errorf("store: save %s: %w", code, kerr)
		case kerr != nil:
			// Damaged bytes stay overwritable.
		case schema != 2 || g != game:
			return fmt.Errorf("store: save %s: %w: another game's file is in place", code, ErrGame)
		default:
			if old, derr := st.codecs.decode(existing); derr == nil && old.Retention().Status == StatusArchived {
				return fmt.Errorf("store: save %s: %w", code, ErrArchived)
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) || from != "" {
		return fmt.Errorf("store: save %s: %w", code, err)
	}
	if err := writeBytes(path, data); err != nil {
		return err
	}
	if record.Retention().Status == StatusArchived {
		return st.moveArchive(path, code)
	}
	return nil
}

// PeekGame reads code's schema-2 record of game under the lock, with no
// assessment and no write, from the live folder or else the archive. It
// reports whether the record is archived, for admission checks before a
// load.
func (st *Store) PeekGame(code, game string) ([]byte, bool, error) {
	path, err := st.path(code)
	if err != nil {
		return nil, false, err
	}
	unlock := st.lockRoom(code)
	defer unlock()
	data, _, archived, err := st.readGame(code, path, game)
	return data, archived, err
}

// readGame requires a validated code and its lock. It decodes through the
// registered codec, so the header is validated, and refuses another
// schema or game, a mismatched code, and a live record in the archive.
func (st *Store) readGame(code, path, game string) ([]byte, string, bool, error) {
	record, data, err := st.codecs.read(path)
	if errors.Is(err, fs.ErrNotExist) {
		path = st.archivePath(code)
		record, data, err = st.codecs.read(path)
	}
	if err != nil {
		return nil, path, false, err
	}
	if schema, g, _ := fileKind(data); schema != 2 || g != game {
		return nil, path, false, fmt.Errorf("%w: not a %s room", ErrGame, game)
	}
	h := record.Retention()
	archived := h.Status == StatusArchived
	if h.Code != code || (path == st.archivePath(code) && !archived) {
		return nil, path, false, fmt.Errorf("store: invalid room header for %s", code)
	}
	return data, path, archived, nil
}

// patchFields returns data with the given top-level fields replaced, every
// other field preserved, in rewriteFields' encoding.
func patchFields(data []byte, changes map[string]any) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for key, value := range changes {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		raw[key] = encoded
	}
	encoded, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
