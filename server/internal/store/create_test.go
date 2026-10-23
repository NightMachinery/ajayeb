package store_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

func duelFile(code, name string) *store.File {
	f := store.New(code, store.ModeHotseat, engine.Options{}, 1, gameNow)
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "deadbeef", Name: name},
		{Kind: store.SeatKindHuman, UserID: "deadbeef", Name: name},
	}
	f.Status = store.StatusPlaying
	return f
}

// held places a file of each kind that can hold a code: a live record of
// the game, another schema-2 game's, a Duel room, damaged bytes, a newer
// schema, and an archived room. It returns the codes, each with the path
// of its file.
func held(t *testing.T, st *store.Store) map[string]string {
	t.Helper()
	files := map[string]string{
		"CRTAAA": place(t, st, "rooms/CRTAAA.json", gameFile("CRTAAA", "playing")),
		"CRTAAB": place(t, st, "rooms/CRTAAB.json", []byte(`{"schema":2,"game":"other","code":"CRTAAB","status":"playing"}`+"\n")),
		"CRTAAD": place(t, st, "rooms/CRTAAD.json", []byte("not json")),
		"CRTAAE": place(t, st, "rooms/CRTAAE.json", []byte(`{"schema":9,"code":"CRTAAE"}`+"\n")),
		"CRTAAF": place(t, st, "rooms-archive/CRTAAF.json", gameFile("CRTAAF", "archived")),
	}
	if err := st.Save(duelFile("CRTAAC", "Sara")); err != nil {
		t.Fatal(err)
	}
	files["CRTAAC"] = filepath.Join(st.Dir(), "CRTAAC.json")
	return files
}

func readAll(t *testing.T, paths map[string]string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for code, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		out[code] = data
	}
	return out
}

// TestCreateGame: a room's first record is written only to a free code.
// Whatever file holds the code, CreateGame writes nothing, answers
// ErrExists and returns the live bytes it found, under the same lock.
func TestCreateGame(t *testing.T) {
	st := gameStore(t)
	files := held(t, st)
	before := readAll(t, files)
	for code := range files {
		existing, err := st.CreateGame(code, "fixture", gameFile(code, "playing"))
		if !errors.Is(err, store.ErrExists) {
			t.Errorf("%s: %v", code, err)
		}
		want := before[code]
		if code == "CRTAAF" {
			want = nil
		}
		if !bytes.Equal(existing, want) {
			t.Errorf("%s: the bytes found: %q", code, existing)
		}
	}
	if after := readAll(t, files); !equalFiles(after, before) {
		t.Error("a held code's file changed")
	}
	if _, err := st.CreateGame("CRTABA", "fixture", gameFile("CRTABB", "playing")); err == nil || errors.Is(err, store.ErrExists) {
		t.Errorf("a record of another code: %v", err)
	}
	if existing, err := st.CreateGame("CRTABA", "fixture", gameFile("CRTABA", "playing")); err != nil || existing != nil {
		t.Fatalf("a free code: %v %q", err, existing)
	}
	if data, _, err := st.PeekGame("CRTABA", "fixture"); err != nil || !bytes.Equal(data, gameFile("CRTABA", "playing")) {
		t.Fatalf("the created record: %v %s", err, data)
	}
	if _, err := st.CreateGame("CRTABA", "fixture", gameFile("CRTABA", "lobby")); !errors.Is(err, store.ErrExists) {
		t.Fatalf("a second creation: %v", err)
	}
}

// TestCreate: Duel's first save is refused for every held code, and
// writes as Save does for a free one.
func TestCreate(t *testing.T) {
	st := gameStore(t)
	files := held(t, st)
	before := readAll(t, files)
	for code := range files {
		if err := st.Create(duelFile(code, "Omid")); !errors.Is(err, store.ErrExists) {
			t.Errorf("%s: %v", code, err)
		}
	}
	if after := readAll(t, files); !equalFiles(after, before) {
		t.Error("a held code's file changed")
	}
	f := duelFile("CRTABC", "Omid")
	if err := st.Create(f); err != nil || f.Hash == "" {
		t.Fatalf("a free code: %v (hash %q)", err, f.Hash)
	}
	if got, _, err := st.Load("CRTABC"); err != nil || got.Seats[0].Name != "Omid" {
		t.Fatalf("the created room: %v", err)
	}
}

// TestCreateRace: creators of every kind racing for one code (schema-2
// records of two games and Duel rooms): exactly one wins, and the file is
// the winner's.
func TestCreateRace(t *testing.T) {
	for round := range 20 {
		st := gameStore(t)
		code := "CRTRA" + string("ABCDEFGHJKMNPQRSTUVWXYZ"[round])
		const n = 12
		var wg sync.WaitGroup
		errs := make([]error, n)
		for i := range n {
			wg.Go(func() {
				switch i % 3 {
				case 0:
					_, errs[i] = st.CreateGame(code, "fixture", markedFile(code, "lobby", i))
				case 1:
					_, errs[i] = st.CreateGame(code, "fixture", markedFile(code, "playing", i))
				default:
					errs[i] = st.Create(duelFile(code, "P"+strconv.Itoa(i)))
				}
			})
		}
		wg.Wait()
		winner := -1
		for i, err := range errs {
			switch {
			case err == nil && winner >= 0:
				t.Fatalf("round %d: creators %d and %d both won", round, winner, i)
			case err == nil:
				winner = i
			case !errors.Is(err, store.ErrExists):
				t.Fatalf("round %d: creator %d: %v", round, i, err)
			}
		}
		if winner < 0 {
			t.Fatalf("round %d: nobody won", round)
		}
		data, err := os.ReadFile(filepath.Join(st.Dir(), code+".json"))
		if err != nil {
			t.Fatal(err)
		}
		mark := []byte(`"creator":` + strconv.Itoa(winner) + `}`)
		if winner%3 == 2 {
			mark = []byte(`"P` + strconv.Itoa(winner) + `"`)
		}
		if !bytes.Contains(data, mark) {
			t.Fatalf("round %d: the file is not creator %d's: %s", round, winner, data)
		}
	}
}

// markedFile is a fixture record that names its creator in a field the
// codec keeps without reading.
func markedFile(code, status string, creator int) []byte {
	return []byte(`{"schema":2,"game":"fixture","code":"` + code + `","status":"` + status +
		`","updated":"2000-01-01T00:00:00Z","hash":"old","creator":` + strconv.Itoa(creator) + `}` + "\n")
}

func equalFiles(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !bytes.Equal(v, b[k]) {
			return false
		}
	}
	return true
}
