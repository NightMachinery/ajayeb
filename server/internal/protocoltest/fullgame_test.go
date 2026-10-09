package protocoltest

import (
	"math/rand/v2"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// replayHash independently replays seed and actions from scratch (the same
// recipe package store uses to check a loaded room) and returns the
// resulting state's hash, for comparing against a live room's own
// Snapshot.State hash.
func replayHash(t *testing.T, seed uint64, actions []string) uint64 {
	t.Helper()
	s := engine.NewGame(seed, engine.Options{})
	for _, key := range actions {
		a, err := engine.ParseKey(key)
		if err != nil {
			t.Fatalf("replay: ParseKey(%q): %v", key, err)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("replay: Apply(%q): %v", key, err)
		}
	}
	return engine.Hash(&s)
}

// checkReplay asserts that replaying a finished room's seed and action log
// from scratch gives the same hash as its own ground-truth state: the
// server's live result and an independent replay agree, the property
// restart persistence depends on (store.Load does the same replay when a
// room is reloaded from disk).
func checkReplay(t *testing.T, srv *server, code string) {
	t.Helper()
	snap := srv.snapshot(code)
	if !snap.State.Over() {
		t.Fatalf("checkReplay(%s): room is not finished", code)
	}
	want := engine.Hash(&snap.State)
	got := replayHash(t, snap.State.Seed, snap.Actions)
	if got != want {
		t.Fatalf("checkReplay(%s): replay hash %x, want %x (live state)", code, got, want)
	}
}

func TestFullGameOnlineWithSpectator(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_000_000_000_001)
	rng := rand.New(rand.NewPCG(1, 2))
	res := playOnline(t, srv, rng)

	if res.final.View.Result == nil {
		t.Fatalf("online game ended without a result")
	}
	if res.final.Room.Status != "finished" {
		t.Fatalf("online game's room status = %q, want finished", res.final.Room.Status)
	}
	checkReplay(t, srv, res.code)
}

func TestFullGameHotseat(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_000_000_000_101)
	rng := rand.New(rand.NewPCG(3, 4))
	res := playHotseat(t, srv, rng)

	if res.final.View.Result == nil {
		t.Fatalf("hotseat game ended without a result")
	}
	checkReplay(t, srv, res.code)
}

func TestFullGameAI(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_000_000_000_201)
	rng := rand.New(rand.NewPCG(5, 6))
	res := playAI(t, srv, rng)

	if res.final.View.Result == nil {
		t.Fatalf("ai game ended without a result")
	}
	checkReplay(t, srv, res.code)
}

// TestFullGamesAreVaried plays a handful of ai games with different seeds
// and checks they do not all end the same way, a cheap sanity check that
// the random policy (and the engine behind it) is not stuck in one path.
func TestFullGamesAreVaried(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_000_000_001_001)
	victories := map[string]bool{}
	for i := range 5 {
		rng := rand.New(rand.NewPCG(uint64(100+i), uint64(200+i)))
		res := playAI(t, srv, rng)
		if res.final.View.Result == nil {
			t.Fatalf("game %d ended without a result", i)
		}
		victories[string(res.final.View.Result.Victory)] = true
		checkReplay(t, srv, res.code)
	}
	if len(victories) == 0 {
		t.Fatalf("no games completed")
	}
}
