package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{"match", "extra"},
		{"match", "-nosuchflag"},
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
	for _, s := range []string{"random", "easy", "greedy", "medium", "hard", "greedy:8", "easy:1"} {
		b, err := parseBot(s)
		if err != nil || b.String() != s {
			t.Errorf("parseBot(%q) = %v, %v", s, b, err)
		}
		if b.make(1) == nil {
			t.Errorf("%s: no bot", s)
		}
	}
	for _, s := range []string{"", "Greedy", "greedy:", "greedy:x", "greedy:-1", "hard:2", "mcts"} {
		if _, err := parseBot(s); err == nil {
			t.Errorf("parseBot(%q) accepted", s)
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
