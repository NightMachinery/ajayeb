package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
func readFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
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
