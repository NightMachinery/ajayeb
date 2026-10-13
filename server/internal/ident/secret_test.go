package ident

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The tests of the secret file checks in secret.go.

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes")
	}
}

// writeSecret writes a valid secret file with the given mode, whatever the
// umask.
func writeSecret(t *testing.T, dir string, mode fs.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, "secret.key")
	if err := os.WriteFile(path, bytes.Repeat([]byte{7}, secretBytes), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func bufferLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func TestSecretFileWithOpenModeIsTightened(t *testing.T) {
	skipOnWindows(t)
	for _, mode := range []fs.FileMode{0o644, 0o640, 0o604, 0o660, 0o666, 0o602} {
		dir := t.TempDir()
		path := writeSecret(t, dir, mode)
		logger, buf := bufferLogger()
		s, err := LoadSecretLogged(dir, logger)
		if err != nil {
			t.Fatalf("mode %04o: %v", mode, err)
		}
		if !bytes.Equal(s.key, bytes.Repeat([]byte{7}, secretBytes)) {
			t.Fatalf("mode %04o: the key was not read back", mode)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("mode %04o became %04o, want 0600", mode, got)
		}
		if !strings.Contains(buf.String(), "level=WARN") || !strings.Contains(buf.String(), fmt.Sprintf("mode=%04o", mode)) {
			t.Errorf("mode %04o: no warning logged: %q", mode, buf.String())
		}
	}
}

func TestSecretFileWithPrivateModeIsLeftAlone(t *testing.T) {
	skipOnWindows(t)
	for _, mode := range []fs.FileMode{0o600, 0o400} {
		dir := t.TempDir()
		path := writeSecret(t, dir, mode)
		logger, buf := bufferLogger()
		if _, err := LoadSecretLogged(dir, logger); err != nil {
			t.Fatalf("mode %04o: %v", mode, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != mode {
			t.Errorf("mode %04o became %04o", mode, got)
		}
		if buf.Len() != 0 {
			t.Errorf("mode %04o logged %q", mode, buf.String())
		}
	}
}

// A secret file other users can read, whose mode cannot be fixed, stops the
// server from starting.
func TestSecretFileChmodFailureRefuses(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	writeSecret(t, dir, 0o644)
	boom := errors.New("chmod refused")
	old := chmodFile
	chmodFile = func(*os.File, fs.FileMode) error { return boom }
	t.Cleanup(func() { chmodFile = old })
	logger, _ := bufferLogger()
	if _, err := LoadSecretLogged(dir, logger); !errors.Is(err, boom) {
		t.Fatalf("LoadSecretLogged = %v, want the chmod error", err)
	}
}

func TestSecretFileMustBeRegular(t *testing.T) {
	skipOnWindows(t)
	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "secret.key"), 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSecret(dir); !errors.Is(err, ErrSecretNotRegular) {
			t.Fatalf("LoadSecret = %v, want ErrSecretNotRegular", err)
		}
	})
	t.Run("symbolic link", func(t *testing.T) {
		dir := t.TempDir()
		target := writeSecret(t, t.TempDir(), 0o600)
		if err := os.Symlink(target, filepath.Join(dir, "secret.key")); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSecret(dir); !errors.Is(err, ErrSecretNotRegular) {
			t.Fatalf("LoadSecret = %v, want ErrSecretNotRegular", err)
		}
	})
	t.Run("dangling symbolic link", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Symlink(filepath.Join(dir, "nowhere"), filepath.Join(dir, "secret.key")); err != nil {
			t.Fatal(err)
		}
		// Not "missing, so create one": the link stays, and is refused.
		if _, err := LoadSecret(dir); !errors.Is(err, ErrSecretNotRegular) {
			t.Fatalf("LoadSecret = %v, want ErrSecretNotRegular", err)
		}
	})
}

func TestSecretFileWrongLength(t *testing.T) {
	for _, n := range []int{0, secretBytes - 1, secretBytes + 1, 1 << 20} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "secret.key"), make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSecret(dir); err == nil || !strings.Contains(err.Error(), "bytes, want 32") {
			t.Errorf("%d bytes: LoadSecret = %v, want a length error", n, err)
		}
	}
}
