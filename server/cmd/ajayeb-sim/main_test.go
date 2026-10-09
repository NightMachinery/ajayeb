package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// simRun runs the command and returns its exit status and outputs.
func simRun(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

// simResultLines drops the timing lines of a match report, which vary from
// run to run.
func simResultLines(out string) string {
	var keep []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "moves:") || strings.HasPrefix(l, "games last") {
			continue
		}
		keep = append(keep, l)
	}
	return strings.Join(keep, "\n")
}

func TestUsage(t *testing.T) {
	if code, _, _ := simRun(t, ""); code != 2 {
		t.Errorf("no arguments: exit %d", code)
	}
	if code, out, _ := simRun(t, "", "-help"); code != 0 || !strings.Contains(out, "match") {
		t.Errorf("-help: exit %d, %q", code, out)
	}
	if code, _, _ := simRun(t, "", "frobnicate"); code != 2 {
		t.Errorf("unknown subcommand: exit %d", code)
	}
	for _, args := range [][]string{
		{"match", "-a", "nobody"},
		{"match", "-b", "random:3"},
		{"match", "-a", "greedy:0"},
		{"match", "-games", "0"},
		{"match", "-p", "0"},
		{"match", "-scale", "0"},
		{"match", "-scale", "NaN"},
		{"match", "-shard", "2/2"},
		{"match", "-shard", "x"},
		{"match", "-min", "65"},
		{"match", "-wa", "no-such-weights.json"},
		{"match", "extra"},
		{"match", "-nosuchflag"},
		{"tally"},
		{"tally", "-min", "2", "x.json"},
		{"levels", "-moves", "0"},
		{"levels", "-scale", "-1"},
		{"levels", "extra"},
		{"weights", "extra"},
		{"bench", "-games", "1"},
		{"replay"},
	} {
		if code, _, _ := simRun(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := simRun(t, "", "match", "-help"); code != 0 {
		t.Errorf("match -help: exit %d", code)
	}
}

func TestParseBot(t *testing.T) {
	for _, s := range []string{"random", "easy", "greedy", "medium", "hard", "greedy:8", "easy:1", "medium:5000", "hard:2"} {
		b, err := parseBot(s)
		if err != nil || b.String() != s {
			t.Errorf("parseBot(%q) = %v, %v", s, b, err)
		}
		if b.make(1, nil) == nil {
			t.Errorf("%s: no bot", s)
		}
	}
	for _, s := range []string{"", "Greedy", "greedy:", "greedy:x", "greedy:-1", "greedy:1001", "random:3", "hard:0", "mcts"} {
		if _, err := parseBot(s); err == nil {
			t.Errorf("parseBot(%q) accepted", s)
		}
	}
}

// TestBotBudgets: medium and hard search their levels' fixed budgets, or N
// iterations, scaled; -clock gives them the clock budgets; Greedy's budget
// is its determinizations, whatever the scale.
func TestBotBudgets(t *testing.T) {
	spec := func(s string) botSpec {
		t.Helper()
		b, err := parseBot(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	cases := []struct {
		bot   string
		clock bool
		scale float64
		want  ai.Budget
	}{
		{"random", false, 0.5, ai.Budget{}},
		{"greedy", false, 0.5, ai.Budget{}},
		{"easy:4", true, 1, ai.Budget{MaxIterations: 4}},
		{"medium", false, 1, ai.Medium.FixedBudget()},
		{"hard", false, 1, ai.Hard.FixedBudget()},
		{"medium", true, 1, ai.Medium.Budget()},
		{"hard", true, 0.1, ai.Hard.Budget()},
		{"medium", false, 0.25, ai.Budget{MaxIterations: ai.MediumIterations / 4, Workers: ai.MediumWorkers}},
		{"hard:1000", false, 0.5, ai.Budget{MaxIterations: 500, Workers: ai.HardWorkers}},
		{"medium:1", false, 0.01, ai.Budget{MaxIterations: 1, Workers: ai.MediumWorkers}},
	}
	for _, c := range cases {
		if got := spec(c.bot).budget(c.clock, c.scale); got != c.want {
			t.Errorf("%s (clock %v, scale %v): budget %+v, want %+v", c.bot, c.clock, c.scale, got, c.want)
		}
	}
}

// TestMatchIsDeterministic: a match's results do not depend on -p, and the
// report has the expected shape.
func TestMatchIsDeterministic(t *testing.T) {
	args := []string{"match", "-a", "greedy:1", "-b", "random", "-games", "5", "-seed", "3", "-v"}
	code1, out1, err1 := simRun(t, "", append(args, "-p", "1")...)
	code2, out2, err2 := simRun(t, "", append(args, "-p", "3")...)
	if code1 != 0 || code2 != 0 {
		t.Fatalf("exit %d, %d: %s %s", code1, code2, err1, err2)
	}
	if simResultLines(out1) != simResultLines(out2) {
		t.Fatalf("-p 1 and -p 3 differ:\n%s\n---\n%s", out1, out2)
	}
	for _, want := range []string{"6 games (3 deals", "greedy:1 scores", "95% interval", "game 5:", "moves: greedy:1"} {
		if !strings.Contains(out1, want) {
			t.Errorf("the report lacks %q:\n%s", want, out1)
		}
	}
	t.Log(out1)
}

// TestMatchLogsReplay: the logs a match writes replay cleanly, and a
// tampered log fails.
func TestMatchLogsReplay(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := simRun(t, "", "match", "-a", "easy:1", "-b", "greedy:1", "-games", "2", "-logs", dir); code != 0 {
		t.Fatalf("match: exit %d: %s", code, errOut)
	}
	files, err := filepath.Glob(filepath.Join(dir, "game-*.json"))
	if err != nil || len(files) != 2 {
		t.Fatalf("logs: %v, %v", files, err)
	}
	code, out, errOut := simRun(t, "", append([]string{"replay"}, files...)...)
	if code != 0 || strings.Count(out, ": ok: ") != 2 || !strings.Contains(out, "matches the log's finalHash") {
		t.Fatalf("replay: exit %d:\n%s%s", code, out, errOut)
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var log engine.Log
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatal(err)
	}
	log.FinalHash = "0000000000000000"
	bad, _ := json.Marshal(&log)
	if code, out, _ := simRun(t, string(bad), "replay", "-"); code != 1 || !strings.Contains(out, "FAIL") {
		t.Fatalf("a wrong hash: exit %d, %s", code, out)
	}
	log.FinalHash = ""
	log.Actions = log.Actions[:len(log.Actions)-1]
	cut, _ := json.Marshal(&log)
	if code, out, _ := simRun(t, string(cut), "replay", "-"); code != 0 || !strings.Contains(out, "not over") || !strings.Contains(out, "no finalHash") {
		t.Fatalf("a cut log: exit %d, %s", code, out)
	}
	log.Actions[3].Seat ^= 1
	wrong, _ := json.Marshal(&log)
	if code, _, _ := simRun(t, string(wrong), "replay", "-"); code != 1 {
		t.Fatalf("a wrong seat: exit %d", code)
	}
	if code, _, _ := simRun(t, "{", "replay", "-"); code != 1 {
		t.Fatalf("broken JSON: exit %d", code)
	}
}

// TestReplayGoldens: the engine's golden replays (a log inside an object)
// replay to their recorded hashes.
func TestReplayGoldens(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "internal", "engine", "testdata", "replays", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no golden replays: %v", err)
	}
	code, out, errOut := simRun(t, "", append([]string{"replay"}, files...)...)
	if code != 0 || strings.Count(out, ": ok: ") != len(files) {
		t.Fatalf("exit %d:\n%s%s", code, out, errOut)
	}
}

func TestBenchRuns(t *testing.T) {
	code, out, errOut := simRun(t, "", "bench", "-games", "2", "-positions", "4", "-playouts", "10", "-p", "1")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	for _, want := range []string{"random playout", "Determinize", "greedy ", "match greedy vs random", "match easy vs random", "bench done"} {
		if !strings.Contains(out, want) {
			t.Errorf("the bench output lacks %q:\n%s", want, out)
		}
	}
}

func TestWilson(t *testing.T) {
	lo, hi := wilson(0.5, 100)
	if lo < 0.40 || lo > 0.41 || hi < 0.59 || hi > 0.60 {
		t.Errorf("wilson(0.5, 100) = %v, %v", lo, hi)
	}
	if lo, hi := wilson(1, 100); hi != 1 || lo < 0.96 || lo > 0.97 {
		t.Errorf("wilson(1, 100) = %v, %v", lo, hi)
	}
	if lo, hi := wilson(0, 0); lo != 0 || hi != 1 {
		t.Errorf("wilson(0, 0) = %v, %v", lo, hi)
	}
}

// TestShardsTallyToTheWholeMatch: the shards of a match, merged by tally,
// give the same report as the whole match; tally refuses a duplicate game
// or another match, and -min checks the score and the missing games.
func TestShardsTallyToTheWholeMatch(t *testing.T) {
	dir := t.TempDir()
	args := []string{"match", "-a", "greedy:1", "-b", "random", "-games", "10", "-seed", "5", "-p", "2"}
	whole := filepath.Join(dir, "whole.json")
	code, wholeOut, errOut := simRun(t, "", append(args, "-json", whole, "-min", "0.5")...)
	if code != 0 || !strings.Contains(wholeOut, "PASS: greedy:1 scored") {
		t.Fatalf("whole match: exit %d:\n%s%s", code, wholeOut, errOut)
	}
	var shards []string
	for k := range 3 {
		path := filepath.Join(dir, fmt.Sprintf("shard-%d.json", k))
		if code, out, errOut := simRun(t, "", append(args, "-shard", fmt.Sprintf("%d/3", k), "-json", path)...); code != 0 {
			t.Fatalf("shard %d: exit %d:\n%s%s", k, code, out, errOut)
		}
		shards = append(shards, path)
	}
	code, tallyOut, errOut := simRun(t, "", append([]string{"tally", "-min", "0.5"}, shards...)...)
	if code != 0 {
		t.Fatalf("tally: exit %d:\n%s%s", code, tallyOut, errOut)
	}
	if got, want := simResultLines(tallyOut), simResultLines(wholeOut); got != want {
		t.Errorf("the shards tally to\n%s\nbut the whole match gives\n%s", got, want)
	}
	if code, out, _ := simRun(t, "", "tally", "-min", "0.5", shards[0], shards[2]); code != 1 || !strings.Contains(out, "FAIL: only") {
		t.Errorf("two of three shards with -min: exit %d:\n%s", code, out)
	}
	if code, out, _ := simRun(t, "", "tally", shards[0], shards[2]); code != 0 || !strings.Contains(out, "missing: 4 of the match's 10 games") {
		t.Errorf("two of three shards: exit %d:\n%s", code, out)
	}
	if code, out, _ := simRun(t, "", "match", "-a", "random", "-b", "greedy:1", "-games", "4", "-min", "0.5"); code != 1 || !strings.Contains(out, "FAIL: random scored") {
		t.Errorf("random against greedy with -min 0.5: exit %d:\n%s", code, out)
	}
	if code, _, errOut := simRun(t, "", "tally", whole, shards[1]); code != 1 || !strings.Contains(errOut, "also in") {
		t.Errorf("a game twice: exit %d: %s", code, errOut)
	}
	other := filepath.Join(dir, "other.json")
	if code, _, _ := simRun(t, "", "match", "-a", "greedy:1", "-b", "random", "-games", "10", "-seed", "6", "-shard", "1/3", "-json", other); code != 0 {
		t.Fatal("the other match failed")
	}
	if code, _, errOut := simRun(t, "", "tally", shards[0], other); code != 1 || !strings.Contains(errOut, "not the same match") {
		t.Errorf("two matches: exit %d: %s", code, errOut)
	}
}

// TestWeightsRoundTrip: weights prints the defaults, -wa reads them back
// (with fields left out keeping their defaults), and a bot with other
// weights plays otherwise.
func TestWeightsRoundTrip(t *testing.T) {
	code, out, _ := simRun(t, "", "weights")
	if code != 0 {
		t.Fatalf("weights: exit %d", code)
	}
	var w ai.Weights
	if err := json.Unmarshal([]byte(out), &w); err != nil || w != ai.DefaultWeights() {
		t.Fatalf("weights printed %s (%v)", out, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "w.json")
	if err := os.WriteFile(path, []byte(`{"Win": 100, "Coin": [3, 3, 3, 3]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readWeights(path)
	want := ai.DefaultWeights()
	want.Coin = [4]float64{3, 3, 3, 3}
	if err != nil || got == nil || *got != want {
		t.Fatalf("readWeights: %+v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte(`{"Coins": 1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readWeights(path); err == nil {
		t.Error("an unknown field was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"Coin": [3, 3, 3, 3]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"match", "-a", "greedy:1", "-b", "greedy:1", "-games", "6", "-seed", "2", "-v"}
	_, plain, _ := simRun(t, "", args...)
	code, tuned, errOut := simRun(t, "", append(args, "-wa", path)...)
	if code != 0 || !strings.Contains(tuned, "greedy:1 (A) uses evaluation weights other than the defaults") {
		t.Fatalf("-wa: exit %d:\n%s%s", code, tuned, errOut)
	}
	if simResultLines(plain) == simResultLines(strings.Replace(tuned, "greedy:1 (A) uses evaluation weights other than the defaults\n", "", 1)) {
		t.Error("the weights changed no game")
	}
}

// TestLevelsRuns: levels measures both tree-search levels (here on a
// hundredth of their fixed budgets, to stay quick).
func TestLevelsRuns(t *testing.T) {
	code, out, errOut := simRun(t, "", "levels", "-games", "1", "-moves", "2", "-fixed", "-scale", "0.01")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	medium := fmt.Sprintf("medium (%d iterations on 1 worker)", ai.MediumIterations/100)
	hard := fmt.Sprintf("hard (%d iterations on %d workers)", ai.HardIterations/100, ai.HardWorkers)
	for _, want := range []string{"fixed budgets on 2 positions", "easy (3 determinizations)", medium, hard, "iterations per move: p10", "time per move", "search trees", "heap held"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
}
