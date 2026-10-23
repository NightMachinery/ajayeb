package lobbyroom

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestCreationCollisions: rooms of every kind share one namespace of
// codes, and a room's first write never replaces another room's file. A
// competitor (another table's rematch, an ordinary lobby, a table, a Duel
// room) takes the code a rematch reserved, before the rematch writes or
// after it. Before: the rematch answers ErrCodeTaken and the competitor's
// bytes stay as they were. After: the competitor loses the code (a
// rematch answers ErrCodeTaken; the others are made under another code)
// and the rematch's lobby stays as it was.
func TestCreationCollisions(t *testing.T) {
	slots := []tableroom.StartSlot{{UserID: user(0), Name: "P0"}, {Bot: tableroom.BotEasy}, {Bot: tableroom.BotEasy}}
	rematch := func(f *fixture, code, parent string) error {
		return f.lobbies.Rematch(f.m, code, tableroom.Rematch{Parent: parent, Creator: user(0), Slots: slots})
	}
	fresh := func(f *fixture) string {
		code, err := f.m.FreshCode()
		if err != nil {
			t.Fatal(err)
		}
		return code
	}
	// Each competitor makes its room with the Manager drawing code first,
	// and returns the code it got, or "" when it lost.
	competitors := []struct {
		name   string
		create func(f *fixture, code string) string
	}{
		{"another rematch", func(f *fixture, code string) string {
			switch err := rematch(f, code, fresh(f)); {
			case err == nil:
				return code
			case errors.Is(err, tableroom.ErrCodeTaken):
				return ""
			default:
				t.Fatal(err)
				return ""
			}
		}},
		{"a lobby", func(f *fixture, code string) string {
			f.drawNext(code)
			return f.create(f.conn(user(5), "c5", "P5"), 3, wire.LobbyOptions{})
		}},
		{"a table", func(f *fixture, code string) string {
			f.drawNext(code)
			seed := uint64(5)
			got, err := f.tables.CreateTable(tableroom.Spec{Seed: &seed, Creator: user(5), Participants: []tableroom.Participant{
				{UserID: user(5), Name: "P5"}, {UserID: user(6), Name: "P6"}, {UserID: user(7), Name: "P7"}}})
			if err != nil {
				t.Fatal(err)
			}
			return got
		}},
		{"a Duel room", func(f *fixture, code string) string {
			f.drawNext(code)
			got, err := f.m.CreateFromSpec(room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}}})
			if err != nil {
				t.Fatal(err)
			}
			return got
		}},
	}
	for _, c := range competitors {
		for _, rematchFirst := range []bool{false, true} {
			name := c.name + " first"
			if rematchFirst {
				name = "the rematch before " + c.name
			}
			t.Run(name, func(t *testing.T) {
				f := newFixture(t, nil)
				code, parent := fresh(f), fresh(f)
				file := func() []byte {
					data, err := os.ReadFile(filepath.Join(f.st.Dir(), code+".json"))
					if err != nil {
						t.Fatal(err)
					}
					return data
				}
				if !rematchFirst {
					if got := c.create(f, code); got != code {
						t.Fatalf("the competitor got %q, want %s", got, code)
					}
					before := file()
					if err := rematch(f, code, parent); !errors.Is(err, tableroom.ErrCodeTaken) {
						t.Fatalf("the rematch: %v", err)
					}
					if !bytes.Equal(file(), before) {
						t.Fatal("the competitor's room was rewritten")
					}
					return
				}
				if err := rematch(f, code, parent); err != nil {
					t.Fatal(err)
				}
				before := file()
				if got := c.create(f, code); got == code {
					t.Fatalf("the competitor took the rematch's code")
				}
				if !bytes.Equal(file(), before) || f.record(code).Parent != parent {
					t.Fatal("the rematch's lobby was rewritten")
				}
				// The rematch, asked again, finds its own lobby.
				if err := rematch(f, code, parent); err != nil || !bytes.Equal(file(), before) {
					t.Fatalf("the rematch again: %v", err)
				}
			})
		}
	}
}

// drawNext queues the Manager's next NewCode answers.
func (f *fixture) drawNext(codes ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes = codes
}

// competingStore writes a competitor's lobby under code at the moment the
// rematch makes its own: after any check the rematch made, immediately
// before its write. An unconditional write to code (SaveGame), which a
// creation must never use, is counted.
type competingStore struct {
	failingStore
	code     string
	victim   []byte
	inserted bool
	saves    int
}

func (s *competingStore) SaveGame(code, game string, data []byte) error {
	if code == s.code {
		s.saves++
	}
	return s.failingStore.SaveGame(code, game, data)
}

func (s *competingStore) CreateGame(code, game string, data []byte) ([]byte, error) {
	if code == s.code && !s.inserted {
		s.inserted = true
		if err := s.Store.SaveGame(code, Game, s.victim); err != nil {
			return nil, err
		}
	}
	return s.failingStore.CreateGame(code, game, data)
}

// TestRematchCollisionAtTheWrite: another creator's lobby lands under the
// reserved code at the moment of the rematch's write. The rematch answers
// ErrCodeTaken and the competitor's bytes stay as they were. (The review's
// probe planted the competitor inside SaveGame, which the rematch called
// then; it now makes its lobby with the conditional CreateGame alone.)
func TestRematchCollisionAtTheWrite(t *testing.T) {
	f := newFixture(t, nil)
	code, err := f.m.FreshCode()
	if err != nil {
		t.Fatal(err)
	}
	parent, err := f.m.FreshCode()
	if err != nil {
		t.Fatal(err)
	}
	r := newRecord(3, wire.LobbyOptions{}, "competing-creator")
	r.Code = code
	r.Created = time.Now().UTC()
	r.Updated = r.Created
	r.Slots[0] = Slot{Kind: SlotHuman, UserID: "competing-creator", Name: "Other"}
	victim, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	st := &competingStore{failingStore: failingStore{Store: f.st, f: f}, code: code, victim: victim}
	f.lobbies.Bind(f.m, st)
	err = f.lobbies.Rematch(f.m, code, tableroom.Rematch{Parent: parent, Creator: user(0), Slots: []tableroom.StartSlot{{UserID: user(0), Name: "P0"}, {Bot: tableroom.BotEasy}, {Bot: tableroom.BotEasy}}})
	after, _, rerr := f.st.PeekGame(code, Game)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if !errors.Is(err, tableroom.ErrCodeTaken) || !bytes.Equal(after, victim) || !st.inserted || st.saves != 0 {
		t.Fatalf("the rematch: %v; the competitor's lobby kept: %v; planted %v; unconditional writes %d", err, bytes.Equal(after, victim), st.inserted, st.saves)
	}
}
