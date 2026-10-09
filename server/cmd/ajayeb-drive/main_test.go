package main

import (
	"io"
	"os"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestParseFlagsDefaults(t *testing.T) {
	_, err := parseFlags(nil, io.Discard)
	if err == nil {
		t.Fatal("want an error: -origin and -mode are required")
	}

	opts, err := parseFlags([]string{"--origin", "http://127.0.0.1:38951", "--mode", "ai"}, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.origin != "http://127.0.0.1:38951" || opts.mode != "ai" {
		t.Errorf("opts = %+v", opts)
	}
	if opts.games != 1 || opts.concurrency != 1 || opts.seedBase != 0 || opts.bot != wire.BotLevelEasy {
		t.Errorf("defaults = %+v", opts)
	}
}

func TestParseFlagsValues(t *testing.T) {
	opts, err := parseFlags([]string{
		"--origin", "http://127.0.0.1:38951",
		"--mode", "online",
		"--games", "20",
		"--seed", "12345",
		"--concurrency", "20",
		"--bot", "hard",
		"--game-timeout", "5s",
	}, io.Discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.games != 20 || opts.seedBase != 12345 || opts.concurrency != 20 || opts.bot != wire.BotLevelHard {
		t.Errorf("opts = %+v", opts)
	}
}

func TestParseFlagsRejectsBadMode(t *testing.T) {
	if _, err := parseFlags([]string{"--origin", "http://x", "--mode", "nonsense"}, io.Discard); err == nil {
		t.Fatal("want an error for an unknown mode")
	}
}

func TestParseFlagsRejectsBadBot(t *testing.T) {
	if _, err := parseFlags([]string{"--origin", "http://x", "--mode", "ai", "--bot", "nonsense"}, io.Discard); err == nil {
		t.Fatal("want an error for an unknown bot level")
	}
}

func TestWsURLFrom(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:38951":  "ws://127.0.0.1:38951/ws",
		"https://example.org":     "wss://example.org/ws",
		"http://127.0.0.1:38951/": "ws://127.0.0.1:38951/ws",
	}
	for origin, want := range cases {
		got, err := wsURLFrom(origin)
		if err != nil {
			t.Fatalf("wsURLFrom(%q): %v", origin, err)
		}
		if got != want {
			t.Errorf("wsURLFrom(%q) = %q, want %q", origin, got, want)
		}
	}
	if _, err := wsURLFrom("ftp://example.org"); err == nil {
		t.Fatal("want an error for an unsupported scheme")
	}
}

func TestRunUsageError(t *testing.T) {
	if code := run(nil, io.Discard, io.Discard); code != 2 {
		t.Errorf("run with no flags = %d, want 2", code)
	}
}

func TestParseFlagsServerPID(t *testing.T) {
	opts, err := parseFlags([]string{"--origin", "http://x", "--mode", "ai", "--server-pid", "4242"}, io.Discard)
	if err != nil || opts.serverPID != 4242 {
		t.Fatalf("parseFlags = %+v, %v; want serverPID 4242", opts, err)
	}
	if _, err := parseFlags([]string{"--origin", "http://x", "--mode", "ai", "--server-pid", "-1"}, io.Discard); err == nil {
		t.Fatal("want an error for a negative -server-pid")
	}
}

func TestReadRSSOfThisProcess(t *testing.T) {
	kib, err := readRSS(t.Context(), os.Getpid())
	if err != nil {
		t.Fatalf("readRSS: %v", err)
	}
	if kib <= 0 {
		t.Fatalf("readRSS = %d KiB, want a positive size", kib)
	}
}
