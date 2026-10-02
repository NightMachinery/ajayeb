package engine_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// recordGame plays a seeded random game from s to the end (or for at most
// plies actions) and returns its sealed log and the hash after every action.
func recordGame(t *testing.T, s engine.State, log *engine.Log, seed uint64, plies int) (*engine.Log, []uint64) {
	t.Helper()
	pol := enginetest.NewRandomPolicy(seed)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	hashes := []uint64{engine.Hash(&s)}
	for n := 0; !s.Over() && n < plies; n++ {
		a := pol.Choose(&s, engine.LegalActions(&s, buf[:0]))
		log.Record(&s, a)
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, engine.Hash(&s))
	}
	log.Seal(&s)
	return log, hashes
}

func TestLogFormat(t *testing.T) {
	opts := engine.Options{FirstPlayer: engine.FirstPlayerSeat1, CenterPawn: engine.LastActiveStarts}
	seed := uint64(1)<<63 + 12345 // beyond JavaScript's safe integers
	log, _ := recordGame(t, engine.NewGame(seed, opts), engine.NewLog(seed, opts, nil), 3, 8)
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		`"format":"ajayeb-log/1"`, `"rules":1`, fmt.Sprintf(`"seed":"%d"`, seed),
		`"options":{"firstPlayer":"seat1","centerPawn":"lastActiveStarts","hideAtEnd":false}`,
		`"scenario":null`, `{"ply":0,"seat":1,"action":"draft:`, `"finalHash":"` + log.FinalHash + `"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the log lacks %s:\n%s", want, text)
		}
	}
	var back engine.Log
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Seed != seed || back.Options != opts || len(back.Actions) != len(log.Actions) || back.FinalHash != log.FinalHash {
		t.Fatalf("round trip: %+v", back)
	}
	for i := range back.Actions {
		if back.Actions[i] != log.Actions[i] {
			t.Fatalf("action %d: %+v, want %+v", i, back.Actions[i], log.Actions[i])
		}
	}
	// Keys are strict on the way in.
	bad := strings.Replace(text, `"action":"draft:`, `"action":"draft:the-`, 1)
	if err := json.Unmarshal([]byte(bad), &back); err == nil {
		t.Fatal("a log with a bad key was accepted")
	}
}

func TestReplayReproducesEveryPly(t *testing.T) {
	for seed := uint64(1); seed <= 20; seed++ {
		log, hashes := recordGame(t, engine.NewGame(seed, engine.Options{}), engine.NewLog(seed, engine.Options{}, nil), seed, 1000)
		steps := 0
		final, err := engine.ReplaySteps(log, func(i int, s *engine.State, ev []engine.Event) error {
			if engine.Hash(s) != hashes[i+1] || len(ev) == 0 || int(s.Ply) != i+1 {
				t.Fatalf("seed %d: replay step %d differs", seed, i)
			}
			steps++
			return nil
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if steps != len(log.Actions) || !final.Over() {
			t.Fatalf("seed %d: %d steps of %d, over %v", seed, steps, len(log.Actions), final.Over())
		}
		if again, err := engine.Replay(log); err != nil || again != final {
			t.Fatalf("seed %d: Replay differs from ReplaySteps: %v", seed, err)
		}
	}
}

func TestReplayScenarioLog(t *testing.T) {
	b := enginetest.New(t, 31).Age(2).City(0, "sawmill").Tokens(1, "law").Pawn(1).OnlyLeft("brewery", "temple", "forum")
	log, hashes := recordGame(t, b.State(), b.Log(), 31, 1000)
	if log.Scenario == nil || log.Scenario.Age != 2 {
		t.Fatal("the log does not carry the scenario")
	}
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}
	var back engine.Log
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	final, err := engine.Replay(&back)
	if err != nil {
		t.Fatal(err)
	}
	if engine.Hash(&final) != hashes[len(hashes)-1] {
		t.Fatal("the scenario replay ends elsewhere")
	}
}

func TestReplayRejects(t *testing.T) {
	good, _ := recordGame(t, engine.NewGame(5, engine.Options{}), engine.NewLog(5, engine.Options{}, nil), 5, 1000)
	clone := func() *engine.Log {
		l := *good
		l.Actions = append([]engine.LogEntry(nil), good.Actions...)
		return &l
	}
	cases := map[string]func(l *engine.Log){
		"format":     func(l *engine.Log) { l.Format = "ajayeb-log/0" },
		"rules":      func(l *engine.Log) { l.Rules = engine.RulesVersion + 1 },
		"seat":       func(l *engine.Log) { l.Actions[3].Seat ^= 1 },
		"ply":        func(l *engine.Log) { l.Actions[4].Ply++ },
		"illegal":    func(l *engine.Log) { l.Actions[10].Action = engine.StarterAction(0) },
		"final hash": func(l *engine.Log) { l.FinalHash = "0000000000000000" },
		"seed":       func(l *engine.Log) { l.Seed++ },
		"scenario":   func(l *engine.Log) { l.Scenario = &engine.Scenario{Age: 9} },
	}
	for name, mutate := range cases {
		l := clone()
		mutate(l)
		if _, err := engine.Replay(l); err == nil {
			t.Errorf("%s: the bad log replayed", name)
		} else if name == "final hash" && !errors.Is(err, engine.ErrHashMismatch) {
			t.Errorf("final hash: %v is not ErrHashMismatch", err)
		}
	}
	stop := errors.New("stop")
	if _, err := engine.ReplaySteps(clone(), func(i int, _ *engine.State, _ []engine.Event) error {
		if i == 7 {
			return stop
		}
		return nil
	}); !errors.Is(err, stop) {
		t.Errorf("the step error was not returned: %v", err)
	}
	if _, err := engine.Replay(clone()); err != nil {
		t.Fatalf("the good log fails: %v", err)
	}
}

// TestHashGolden pins the hash of a new game and of a fixed position. They
// change when the state layout, the setup or the catalog changes, which
// invalidates stored games.
func TestHashGolden(t *testing.T) {
	s := engine.NewGame(1, engine.Options{})
	if got, want := engine.HashString(&s), "1f140210c9fd8b05"; got != want {
		t.Errorf("NewGame(1) hash %s, want %s", got, want)
	}
	enginetest.Do(t, &s, "draft appian-way", "draft sphinx", "draft mausoleum", "draft great-library", "draft hanging-gardens", "draft temple-of-artemis")
	if got, want := engine.HashString(&s), "f5a9046ca057ba3c"; got != want {
		t.Errorf("after the draft: hash %s, want %s", got, want)
	}
}

func TestHashSensitivity(t *testing.T) {
	base := enginetest.New(t, 2).Age(2).State()
	h := engine.Hash(&base)
	if engine.Hash(&base) != h {
		t.Fatal("Hash is not stable")
	}
	if got := engine.HashString(&base); got != fmt.Sprintf("%016x", h) {
		t.Fatalf("HashString %s for %016x", got, h)
	}
	mutations := []func(s *engine.State){
		func(s *engine.State) { s.Seed++ },
		func(s *engine.State) { s.Ply++ },
		func(s *engine.State) { s.Opts.HideAtEnd = true },
		func(s *engine.State) { s.Slots[7].FaceUp = !s.Slots[7].FaceUp },
		func(s *engine.State) { s.Slots[0].Card, s.Slots[1].Card = s.Slots[1].Card, s.Slots[0].Card },
		func(s *engine.State) { s.Players[1].Coins++ },
		func(s *engine.State) { s.Players[0].Sci[3]++ },
		func(s *engine.State) { s.BoxTokens[3], s.BoxTokens[4] = s.BoxTokens[4], s.BoxTokens[3] },
		func(s *engine.State) { s.Removed[2][0], s.Removed[2][1] = s.Removed[2][1], s.Removed[2][0] },
		func(s *engine.State) { s.Result.Winner = 1 },
		func(s *engine.State) { s.Library[2] = 1 },
		func(s *engine.State) { s.ExtraTurn = true },
	}
	for i, m := range mutations {
		s := base
		m(&s)
		if engine.Hash(&s) == h {
			t.Errorf("mutation %d does not change the hash", i)
		}
	}
}
