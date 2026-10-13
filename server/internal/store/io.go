package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// ErrTooLarge is wrapped by Load, PeekHeader and ReadLog when a room file is
// larger than MaxFileBytes. No file this server writes comes near it, so
// such a file was not written by it (or not by it alone); it is refused
// before it is read, so a planted file cannot make the server read or
// decode an unbounded amount of data. The file is left as it is.
var ErrTooLarge = errors.New("store: room file too large")

// MaxFileBytes is the largest room file the store reads. It is derived from
// the bounds on what a room file holds, and made four times as large as
// their sum, so no legitimate file comes near it:
//
//   - the action log: at most engine.MaxGameActions keys (and never more
//     than wire.MaxHistory, the history a state frame can carry), each a
//     short canonical key (maxStoredKeyBytes with its indentation);
//   - the room log: at most wire.MaxRoomLog entries (maxStoredLogEntryBytes
//     each);
//   - everything else (seats with names of at most 64 characters, the
//     options, the per-client seqs of at most 32 clients, the hint settings,
//     the result), which fits in maxStoredHeaderBytes.
//
// That makes it 704 KiB. A finished game's file takes a few KiB, and the
// largest file this server can write under 100 KiB (limits_test.go).
const MaxFileBytes = 4 * (maxStoredKeyBytes*max(engine.MaxGameActions, wire.MaxHistory) +
	maxStoredLogEntryBytes*wire.MaxRoomLog + maxStoredHeaderBytes)

const (
	maxStoredKeyBytes      = 96
	maxStoredLogEntryBytes = 1 << 10
	maxStoredHeaderBytes   = 64 << 10
)

// writeFile atomically replaces the file at path with f's JSON encoding: a
// temporary file in the same directory is written, synced and renamed over
// the old path (os.CreateTemp, Write, Sync, Rename), so a crash leaves
// either the old file or the new one, never a torn one. It does not touch
// f.Hash or validate anything; callers that need that call replay
// themselves first (see Save and Load).
func writeFile(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("store: encoding %s: %w", path, err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".room-*.tmp")
	if err != nil {
		return fmt.Errorf("store: creating temp file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("store: writing %s: %w", tmpPath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("store: syncing %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("store: closing %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("store: installing %s: %w", path, err)
	}
	removeTmp = false
	return nil
}

// readFile reads and decodes the file at path, with no replay or hash
// check. See PeekHeader (which uses it directly, for callers that only need
// the header) and Load (which uses it and then verifies the result).
//
// A file larger than MaxFileBytes is refused with an error wrapping
// ErrTooLarge, before any of it is read.
func readFile(path string) (*File, error) {
	data, err := readCapped(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("store: decoding %s: %w", path, err)
	}
	// Every action bumped the version once, so a file written before
	// Version was stored (or with a smaller one) reads as the action count.
	f.Version = max(f.Version, len(f.Actions))
	return &f, nil
}

// readCapped reads the file at path, refusing one larger than MaxFileBytes
// (ErrTooLarge). The size is checked before reading, and the read itself is
// capped too, in case the file grows in between.
func readCapped(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxFileBytes {
		return nil, fmt.Errorf("%w: %s holds %d bytes, at most %d are read", ErrTooLarge, path, info.Size(), MaxFileBytes)
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileBytes {
		return nil, fmt.Errorf("%w: %s grew past %d bytes while it was read", ErrTooLarge, path, MaxFileBytes)
	}
	return data, nil
}
