package engine_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Golden replays pin the engine's behaviour end to end: a seed (and, for the
// forced cases, a starting scenario and a script of actions) is played out
// once, and the resulting log and result are compared, byte for byte, with a
// file under testdata/replays. A change to the rules, the catalog, the RNG or
// the setup order that alters any of these games shows up as a diff here,
// even when every other test still passes. Run with -update to write the
// files after a deliberate change:
//
//	go test ./internal/engine/ -run TestGoldenReplays -update
var updateGolden = flag.Bool("update", false, "rewrite testdata/replays from the current engine")

// goldenReplay is the file format: the full log (seed, options, scenario and
// actions, so it replays on its own) plus the result it produced, so a
// regression in scoring or in a victory check shows up even if it somehow
// left the final hash alone.
type goldenReplay struct {
	Log    *engine.Log   `json:"log"`
	Result engine.Result `json:"result"`
}

// goldenCase is one golden replay: name is the file's base name, and build
// plays the game and returns its sealed log and final state.
type goldenCase struct {
	name  string
	build func(t *testing.T) (*engine.Log, engine.State)
}

// goldenCases covers two natural playouts (random and simple policies from a
// plain deal) and one forced scenario per victory kind, plus a shared
// (tied) civilian victory: the rare case that a plain sweep is unlikely to
// hit. Science supremacy, rare under both policies (TestPropertyRandomGames
// logs 0-3 of it in 2000 games), is pinned here rather than required of the
// sweep.
var goldenCases = []goldenCase{
	{"random", goldenRandom},
	{"simple", goldenSimple},
	{"military", goldenMilitary},
	{"science", goldenScience},
	{"civilian", goldenCivilian},
	{"shared", goldenShared},
}

func TestGoldenReplays(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.name, func(t *testing.T) {
			log, s := c.build(t)
			if !s.Over() {
				t.Fatalf("golden %s: the game is not over", c.name)
			}
			if s.Result.Scores != engine.Scores(&s) {
				t.Fatalf("golden %s: result scores %+v, Scores %+v", c.name, s.Result.Scores, engine.Scores(&s))
			}
			// The log must stand on its own: replaying it reproduces the same
			// hash and result as the game that was actually played.
			final, err := engine.Replay(log)
			if err != nil {
				t.Fatalf("golden %s: replay: %v", c.name, err)
			}
			if h, want := engine.Hash(&final), engine.Hash(&s); h != want {
				t.Fatalf("golden %s: replay hash %016x, want %016x", c.name, h, want)
			}
			if final.Result != s.Result {
				t.Fatalf("golden %s: replay result %+v, want %+v", c.name, final.Result, s.Result)
			}
			checkGoldenFile(t, filepath.Join("testdata", "replays", c.name+".json"), &goldenReplay{Log: log, Result: s.Result})
		})
	}
}

// checkGoldenFile compares got, marshaled, against the file at path, or (with
// -update) writes it there.
func checkGoldenFile(t *testing.T, path string, got *goldenReplay) {
	t.Helper()
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to create it)", path, err)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("%s is stale; run go test ./internal/engine/ -run TestGoldenReplays -update to refresh it:\n--- got ---\n%s\n--- on disk ---\n%s",
			path, data, want)
	}
}

// playPolicyGame plays a full game from a plain deal with pols (indexed by
// seat) and returns its sealed log and final state. It fails the test if the
// game does not end within maxPlies actions (maxPlies is defined, and the
// same bound enforced, in property_test.go).
func playPolicyGame(t *testing.T, seed uint64, opts engine.Options, pols [2]enginetest.Policy) (*engine.Log, engine.State) {
	t.Helper()
	s := engine.NewGame(seed, opts)
	log := engine.NewLog(seed, opts, nil)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for n := 0; !s.Over(); n++ {
		if n >= maxPlies {
			t.Fatalf("seed %d: the game is not over after %d actions (phase %s, age %d)", seed, maxPlies, s.Phase, s.Age)
		}
		a := pols[s.ToMove].Choose(&s, engine.LegalActions(&s, buf[:0]))
		log.Record(&s, a)
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("seed %d: legal action %s failed: %v", seed, a, err)
		}
	}
	log.Seal(&s)
	return log, s
}

// playScript plays a scenario by name (built, scripted and sealed), applying
// steps in order and recording each into the scenario's own log. It fails the
// test at the first step that does not parse, is not legal, or breaks an
// invariant.
func playScript(t *testing.T, b *enginetest.Builder, steps ...string) (*engine.Log, engine.State) {
	t.Helper()
	s := b.State()
	log := b.Log()
	for i, step := range steps {
		a, err := enginetest.ParseStep(step)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		log.Record(&s, a)
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("step %d (%s): %v", i, step, err)
		}
		if err := engine.CheckInvariants(&s); err != nil {
			t.Fatalf("after step %d (%s): %v", i, step, err)
		}
	}
	log.Seal(&s)
	return log, s
}

// goldenRandom plays a full RandomPolicy game from a plain deal.
func goldenRandom(t *testing.T) (*engine.Log, engine.State) {
	const seed = 910_001
	pol := enginetest.NewRandomPolicy(seed ^ 0x5eed)
	return playPolicyGame(t, seed, engine.Options{}, [2]enginetest.Policy{pol, pol})
}

// goldenSimple plays a full SimplePolicy game from a plain deal, one seat's
// policy seeded one past the other's so the two play differently.
func goldenSimple(t *testing.T) (*engine.Log, engine.State) {
	const seed = 910_002
	pols := [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)}
	return playPolicyGame(t, seed, engine.Options{}, pols)
}

// goldenMilitary forces a military supremacy on the last card of an age: the
// pawn one space from seat 1's capital, seat 0 to move with a free shield
// (Guard Tower) accessible.
func goldenMilitary(t *testing.T) (*engine.Log, engine.State) {
	b := enginetest.New(t, 910_003).Name("military supremacy on the last card of an age").
		OnlyLeft("guard-tower").Pawn(8).ToMove(0)
	log, s := playScript(t, b, "construct guard-tower")
	if s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
		t.Fatalf("golden military: result %+v, want a military win for seat 0", s.Result)
	}
	return log, s
}

// goldenScience forces scientific supremacy: seat 0 already owns five
// distinct science symbols (set square, wheel, globe, sundial, quill) and
// builds the accessible Pharmacist (mortar) for its sixth.
func goldenScience(t *testing.T) (*engine.Log, engine.State) {
	b := enginetest.New(t, 910_004).Name("scientific supremacy on a sixth distinct symbol").
		Age(1).ToMove(0).Coins(0, 10).
		City(0, "workshop", "apothecary", "university", "academy", "library").
		Accessible("pharmacist")
	log, s := playScript(t, b, "construct pharmacist")
	if s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("golden science: result %+v, want a science win for seat 0", s.Result)
	}
	return log, s
}

// goldenCivilian forces the end of Age III on its last card (Palace,
// discarded for coins) with cities that give seat 1 the higher total despite
// fewer blue points.
func goldenCivilian(t *testing.T) (*engine.Log, engine.State) {
	b := enginetest.New(t, 910_005).Name("civilian scoring on the last card of Age III").
		Age(3).OnlyLeft("palace").FirstPlayer(0).Coins(0, 0).Coins(1, 0).
		City(0, "theater").City(1, "library", "workshop", "apothecary")
	log, s := playScript(t, b, "discard palace")
	if s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != 1 {
		t.Fatalf("golden civilian: result %+v, want a civilian win for seat 1", s.Result)
	}
	return log, s
}

// goldenShared forces a shared civilian victory: both seats tied on total
// points and on blue points.
func goldenShared(t *testing.T) (*engine.Log, engine.State) {
	b := enginetest.New(t, 910_006).Name("a shared civilian victory, tied on total and on blue points").
		Age(3).OnlyLeft("palace").FirstPlayer(0).Coins(0, 0).Coins(1, 0).
		City(0, "theater").City(1, "altar")
	log, s := playScript(t, b, "discard palace")
	if s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != engine.SharedVictory {
		t.Fatalf("golden shared: result %+v, want a shared civilian victory", s.Result)
	}
	return log, s
}
