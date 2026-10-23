package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestParseTableFlags(t *testing.T) {
	o, err := parseTableFlags(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(o.seats, []int{3, 4, 5, 6, 7}) || o.games != 2 || o.schedules != 8 || o.seqClients != 1 || !o.leaks || !o.tabs || o.race {
		t.Errorf("defaults = %+v", o)
	}
	for _, bad := range [][]string{{"-seats", "2"}, {"-seats", "3,8"}, {"-seats", "x"}, {"-games", "0"}, {"-drafts", "101"}, {"extra"}} {
		if _, err := parseTableFlags(bad, io.Discard); err == nil {
			t.Errorf("%v: no error", bad)
		}
	}
	if _, err := parseTableFlags([]string{"-help"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-help: %v", err)
	}
}

func table(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := runTable(t.Context(), args, &out, &errs)
	return code, out.String(), errs.String()
}

// TestRunTable plays games at three and six seats under several schedules
// with the leak oracle on, and requires one ok line per game, the same
// output for the same seed, duplicate seqs met, and stale seqs met once
// seats race (a stale seq needs two submissions from the forgotten tab).
func TestRunTable(t *testing.T) {
	base := []string{"-seats", "3,6", "-games", "2", "-schedules", "4", "-seed", "40"}
	for _, c := range []struct {
		extra []string
		met   string
	}{{nil, " duplicate="}, {[]string{"-race"}, " stale="}} {
		args := append(slices.Clone(base), c.extra...)
		code, out, errs := table(t, args...)
		if code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", args, code, out, errs)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 4 {
			t.Fatalf("%v: want 4 lines, got:\n%s", args, out)
		}
		for _, l := range lines {
			if !strings.HasSuffix(l, "status=ok") || !strings.Contains(l, " leaks=0") || !strings.Contains(l, " schedules=4 ") {
				t.Errorf("%v: line %q", args, l)
			}
		}
		if !strings.Contains(out, c.met) {
			t.Errorf("%v: no%s met:\n%s", args, strings.TrimSuffix(c.met, "="), out)
		}
		if _, again, _ := table(t, args...); again != out {
			t.Errorf("%v: the same seed gave other output:\n%s\n%s", args, out, again)
		}
	}
}

// TestRunTableExhaust replays the first point at four seats, each changing
// its pick once, under all 8!/2^4 = 2520 arrival orders, and a racing game.
func TestRunTableExhaust(t *testing.T) {
	code, out, errs := table(t, "-seats", "4", "-games", "1", "-schedules", "2", "-drafts", "100", "-resends", "0", "-exhaust", "3000")
	if code != 0 || !strings.Contains(out, " orders=2520 batches=1 ") {
		t.Fatalf("exit %d\n%s%s", code, out, errs)
	}
	code, out, errs = table(t, "-seats", "5", "-games", "1", "-schedules", "3", "-race")
	if code != 0 {
		t.Fatalf("race: exit %d\n%s%s", code, out, errs)
	}
	code, _, errs = table(t, "-seats", "3", "-games", "1", "-exhaust", "10")
	if code != 1 || !strings.Contains(errs, "more than 10 arrival orders") {
		t.Fatalf("an order limit: exit %d, %s", code, errs)
	}
}
