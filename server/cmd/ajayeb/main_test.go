package main

import (
	"bytes"
	"io"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseFlagsDefaults(t *testing.T) {
	opts, err := parseFlags(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.addr != defaultAddr || opts.dataDir != defaultData || opts.botDelay != defaultBotDelay || opts.botSearches != defaultBotSearches {
		t.Errorf("defaults = %+v", opts)
	}
	if opts.dev || opts.devEndpoints || opts.staticDir != "" || opts.seedBase != 0 {
		t.Errorf("unexpected non-zero defaults: %+v", opts)
	}
	if opts.origins == nil || opts.trustProxy == nil {
		t.Error("origins and trustProxy should be empty, not nil")
	}
}

func TestParseFlagsValues(t *testing.T) {
	opts, err := parseFlags([]string{
		"--addr", "127.0.0.1:0",
		"--data", "/tmp/x",
		"--static", "../web/build",
		"--dev", "--dev-endpoints",
		"--origin", "https://Example.org, http://localhost:38952",
		"--origin=https://other.example:8443/",
		"--trust-proxy", "127.0.0.1,10.0.0.0/8",
		"--trust-proxy", "::1",
		"--bot-delay", "0s",
		"--bot-searches", "3",
		"--seed-base", "1000",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	wantOrigins := []string{"https://example.org", "http://localhost:38952", "https://other.example:8443"}
	if !slices.Equal(opts.origins, wantOrigins) {
		t.Errorf("origins = %q, want %q", opts.origins, wantOrigins)
	}
	wantProxies := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("::1/128"),
	}
	if !slices.Equal(opts.trustProxy, wantProxies) {
		t.Errorf("trustProxy = %v, want %v", opts.trustProxy, wantProxies)
	}
	if !opts.dev || !opts.devEndpoints || opts.botDelay != 0 || opts.botSearches != 3 || opts.seedBase != 1000 {
		t.Errorf("opts = %+v", opts)
	}
	if opts.addr != "127.0.0.1:0" || opts.dataDir != "/tmp/x" || opts.staticDir != "../web/build" {
		t.Errorf("opts = %+v", opts)
	}
}

func TestParseFlagsRejects(t *testing.T) {
	for name, args := range map[string][]string{
		"dev endpoints without dev": {"--dev-endpoints"},
		"negative bot delay":        {"--bot-delay", "-1s"},
		"no bot searches":           {"--bot-searches", "0"},
		"empty data":                {"--data", ""},
		"origin without scheme":     {"--origin", "example.org"},
		"origin with path":          {"--origin", "https://example.org/app"},
		"origin with ftp":           {"--origin", "ftp://example.org"},
		"origin with user":          {"--origin", "https://me@example.org"},
		"bad proxy":                 {"--trust-proxy", "not-an-ip"},
		"bad cidr":                  {"--trust-proxy", "10.0.0.0/99"},
		"stray argument":            {"serve"},
		"unknown flag":              {"--nope"},
		"bad seed base":             {"--seed-base", "-1"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseFlags(args, io.Discard); err == nil {
				t.Errorf("parseFlags(%q) succeeded, want an error", args)
			}
		})
	}
}

func TestRunRefusesDevEndpointsWithoutDev(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"--dev-endpoints"}, io.Discard, &stderr); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "-dev-endpoints requires -dev") {
		t.Errorf("stderr = %q, want the refusal reason", stderr.String())
	}
}

func TestRunVersionAndHelp(t *testing.T) {
	var stdout bytes.Buffer
	if code := run([]string{"--version"}, &stdout, io.Discard); code != 0 {
		t.Errorf("--version exit code = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != version {
		t.Errorf("--version printed %q, want %q", got, version)
	}

	var stderr bytes.Buffer
	if code := run([]string{"--help"}, io.Discard, &stderr); code != 0 {
		t.Errorf("--help exit code = %d, want 0", code)
	}
	for _, flagName := range []string{"-addr", "-data", "-static", "-dev-endpoints", "-origin", "-trust-proxy", "-bot-delay", "-bot-searches", "-seed-base"} {
		if !strings.Contains(stderr.String(), flagName) {
			t.Errorf("--help output lacks %s", flagName)
		}
	}
	if code := run([]string{"--nope"}, io.Discard, io.Discard); code != 2 {
		t.Errorf("unknown flag exit code = %d, want 2", code)
	}
}

func TestBotDelayParsesDurations(t *testing.T) {
	opts, err := parseFlags([]string{"--bot-delay", "1.5s"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.botDelay != 1500*time.Millisecond {
		t.Errorf("botDelay = %v, want 1.5s", opts.botDelay)
	}
}
