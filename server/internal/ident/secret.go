package ident

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// secretBytes is the length of the HMAC secret: 256 bits, generous for
// HMAC-SHA256 (whose key is hashed down to the block size anyway if it were
// longer, and padded if shorter).
const secretBytes = 32

// Secret is the server's HMAC key, loaded once at start and then held in
// memory. The zero value is not valid; use LoadSecret.
type Secret struct {
	key []byte
}

// LoadSecret reads the secret from <dataDir>/secret.key, a file with mode
// 0600 holding secretBytes bytes of key material. On first start, when the
// file does not exist yet, LoadSecret creates it with a fresh value from
// crypto/rand.
//
// Two processes (or two goroutines in a test) racing to create the file for
// the first time are both safe: the file is written under a temporary name
// in the same directory and then linked into place with os.Link, which
// fails instead of silently overwriting when another racer wins first.
// Either way, every caller reads back and returns whatever content actually
// ended up on disk, so the whole process agrees on one secret.
func LoadSecret(dataDir string) (Secret, error) {
	path := filepath.Join(dataDir, "secret.key")

	if b, err := readSecretFile(path); err == nil {
		return Secret{key: b}, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Secret{}, err
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return Secret{}, fmt.Errorf("ident: creating %s: %w", dataDir, err)
	}
	fresh := make([]byte, secretBytes)
	if _, err := rand.Read(fresh); err != nil {
		return Secret{}, fmt.Errorf("ident: generating secret: %w", err)
	}

	// os.CreateTemp always creates its file with mode 0600, and os.Link
	// preserves the file it points at (the two names share one inode), so
	// the installed secret.key keeps that mode without a separate Chmod.
	tmp, err := os.CreateTemp(dataDir, ".secret-*.tmp")
	if err != nil {
		return Secret{}, fmt.Errorf("ident: creating temp secret file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(fresh); err != nil {
		_ = tmp.Close()
		return Secret{}, fmt.Errorf("ident: writing temp secret file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return Secret{}, fmt.Errorf("ident: syncing temp secret file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Secret{}, fmt.Errorf("ident: closing temp secret file: %w", err)
	}

	// os.Link fails with ErrExist if path is already there, unlike Rename
	// (which would silently replace it); that is what makes the race safe.
	// A concurrent winner's file is just as good as ours, so either outcome
	// is fine here, and the common tmp file is always removed by the defer.
	if err := os.Link(tmpPath, path); err != nil && !errors.Is(err, fs.ErrExist) {
		return Secret{}, fmt.Errorf("ident: installing secret file: %w", err)
	}

	b, err := readSecretFile(path)
	if err != nil {
		return Secret{}, fmt.Errorf("ident: reading secret file after create: %w", err)
	}
	return Secret{key: b}, nil
}

func readSecretFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) != secretBytes {
		return nil, fmt.Errorf("ident: secret file %s holds %d bytes, want %d", path, len(b), secretBytes)
	}
	return b, nil
}
