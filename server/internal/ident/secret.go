package ident

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
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

// ErrSecretNotRegular is returned (wrapped) by LoadSecret when secret.key is
// not a regular file: a directory, a device, a FIFO or a symbolic link. A
// symbolic link is refused rather than followed, so that whoever can write
// to the data directory cannot point the server at another file.
var ErrSecretNotRegular = errors.New("ident: secret.key is not a regular file")

// LoadSecret is LoadSecretLogged with the default logger.
func LoadSecret(dataDir string) (Secret, error) {
	return LoadSecretLogged(dataDir, slog.Default())
}

// LoadSecretLogged reads the secret from <dataDir>/secret.key, a regular
// file with mode 0600 holding secretBytes bytes of key material. On first
// start, when the file does not exist yet, it creates the file with a fresh
// value from crypto/rand.
//
// It refuses a secret.key that is not a regular file (ErrSecretNotRegular).
// One that its group or other users may read or write (a copy restored with
// the wrong mode, say) is tightened to 0600 with a warning on logger, and
// LoadSecretLogged fails if that chmod does, rather than run with a key
// other users can read.
//
// Two processes (or two goroutines in a test) racing to create the file for
// the first time are both safe: the file is written under a temporary name
// in the same directory and then linked into place with os.Link, which
// fails instead of silently overwriting when another racer wins first.
// Either way, every caller reads back and returns whatever content actually
// ended up on disk, so the whole process agrees on one secret.
func LoadSecretLogged(dataDir string, logger *slog.Logger) (Secret, error) {
	if logger == nil {
		logger = slog.Default()
	}
	path := filepath.Join(dataDir, "secret.key")

	if b, err := readSecretFile(path, logger); err == nil {
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

	b, err := readSecretFile(path, logger)
	if err != nil {
		return Secret{}, fmt.Errorf("ident: reading secret file after create: %w", err)
	}
	return Secret{key: b}, nil
}

// chmodFile changes an open file's mode; a test replaces it to make it fail.
var chmodFile = func(f *os.File, mode fs.FileMode) error { return f.Chmod(mode) }

// readSecretFile reads the secret file at path, refusing anything but a
// regular file and tightening a mode that lets other users at it.
func readSecretFile(path string, logger *slog.Logger) ([]byte, error) {
	// Lstat, so a symbolic link is seen as one and refused.
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s has mode %s", ErrSecretNotRegular, path, info.Mode())
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	// The file opened must be the one checked, not one swapped in since.
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s changed while it was opened", ErrSecretNotRegular, path)
	}
	if perm := opened.Mode().Perm(); perm&0o077 != 0 {
		if err := chmodFile(f, 0o600); err != nil {
			return nil, fmt.Errorf("ident: secret file %s has mode %04o and cannot be made 0600: %w", path, perm, err)
		}
		logger.Warn("the secret file was readable or writable by other users; its mode is now 0600",
			"path", path, "mode", fmt.Sprintf("%04o", perm))
	}
	// One byte more than a valid secret is enough to tell a wrong length.
	b, err := io.ReadAll(io.LimitReader(f, secretBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) != secretBytes {
		size := fmt.Sprint(len(b))
		if len(b) > secretBytes {
			size = "more than " + fmt.Sprint(secretBytes)
		}
		return nil, fmt.Errorf("ident: secret file %s holds %s bytes, want %d", path, size, secretBytes)
	}
	return b, nil
}
