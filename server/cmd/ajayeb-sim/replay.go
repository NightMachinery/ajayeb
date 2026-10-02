package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// maxLogBytes bounds a log file. A real game log is about 5 KB.
const maxLogBytes = 16 << 20

// runReplay is the replay subcommand.
func runReplay(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := newFlags("replay", stderr)
	quiet := fs.Bool("q", false, "print only failures")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "ajayeb-sim replay: name at least one log file (or - for standard input)")
		return errUsage
	}
	failed := 0
	for _, name := range fs.Args() {
		line, err := replayFile(name, stdin)
		if err != nil {
			failed++
			fmt.Fprintf(stdout, "%s: FAIL: %v\n", name, err)
			continue
		}
		if !*quiet {
			fmt.Fprintf(stdout, "%s: %s\n", name, line)
		}
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "ajayeb-sim replay: %d of %d logs failed\n", failed, fs.NArg())
		return errFailed
	}
	return nil
}

// replayFile replays one log and describes the result.
func replayFile(name string, stdin io.Reader) (string, error) {
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(io.LimitReader(stdin, maxLogBytes+1))
	} else {
		data, err = readLimited(name)
	}
	if err != nil {
		return "", err
	}
	if len(data) > maxLogBytes {
		return "", fmt.Errorf("larger than %d bytes", maxLogBytes)
	}
	log, err := decodeLog(data)
	if err != nil {
		return "", err
	}
	end, err := engine.Replay(log)
	if err != nil {
		return "", err
	}
	hash := engine.HashString(&end)
	check := "the log has no finalHash"
	if log.FinalHash != "" {
		check = "matches the log's finalHash" // Replay has compared it
	}
	return fmt.Sprintf("ok: %d actions, %s; final hash %s (%s)", len(log.Actions), describeEnd(&end), hash, check), nil
}

// readLimited reads a file of at most maxLogBytes (plus one byte, so that a
// larger file is noticed).
func readLimited(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxLogBytes+1))
}

// decodeLog decodes an ajayeb-log/1 log, either bare or as the "log" field
// of an object (the engine's golden replays wrap their logs that way).
func decodeLog(data []byte) (*engine.Log, error) {
	var probe struct {
		Format string          `json:"format"`
		Log    json.RawMessage `json:"log"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("not a JSON object: %w", err)
	}
	if probe.Format == "" && len(probe.Log) > 0 {
		data = probe.Log
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var log engine.Log
	if err := dec.Decode(&log); err != nil {
		return nil, fmt.Errorf("not an %s log: %w", engine.LogFormat, err)
	}
	if dec.More() {
		return nil, errors.New("trailing data after the log")
	}
	return &log, nil
}

// describeEnd says where a replayed game stopped.
func describeEnd(s *engine.State) string {
	if !s.Over() {
		return fmt.Sprintf("not over (ply %d, age %d, phase %s, seat %d to move)", s.Ply, s.Age, s.Phase, s.ToMove)
	}
	r := &s.Result
	if r.Winner == engine.SharedVictory {
		return fmt.Sprintf("shared victory, %d-%d", r.Scores[0].Total, r.Scores[1].Total)
	}
	return fmt.Sprintf("seat %d wins by %s, %d-%d", r.Winner, r.Kind, r.Scores[0].Total, r.Scores[1].Total)
}
