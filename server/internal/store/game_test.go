package store_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func TestRoomGameHeaderSelection(t *testing.T) {
	st := newStore(t)
	if _, err := st.Game("../bad"); !errors.Is(err, store.ErrBadCode) {
		t.Fatalf("invalid code: %v", err)
	}
	if _, err := st.Game("REGAAA"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing room: %v", err)
	}
	raw := []byte(`{"seed":{},"schema":1,"code":"REGAAA"}`)
	for _, dir := range []string{st.Dir(), st.ArchiveDir()} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "REGAAA.json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if game, err := st.Game("REGAAA"); err != nil || game != "duel" {
			t.Fatalf("header decoded Duel payload: %q %v", game, err)
		}
		if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, raw) {
			t.Fatalf("selector changed bytes: %v", err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(st.Dir(), "REGAAA.json")
	if err := os.WriteFile(path, bytes.Repeat([]byte(" "), store.MaxFileBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Game("REGAAA"); !errors.Is(err, store.ErrTooLarge) {
		t.Fatalf("uncapped selection: %v", err)
	}
}
