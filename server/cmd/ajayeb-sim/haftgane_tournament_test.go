package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	hbot "github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func TestHATournamentSyntheticRewards(t *testing.T) {
	for _, tc := range []struct {
		scores, coins  []int16
		focal          int
		rank, win, mid float64
	}{
		{[]int16{10, 10, 9}, []int16{3, 3, 100}, 0, .75, .5, 1.5},
		{[]int16{10, 10, 10}, []int16{2, 1, 3}, 0, .5, 0, 2},
		{[]int16{10, 10, 10}, []int16{3, 3, 3}, 0, .5, 1.0 / 3, 2},
		{[]int16{10, 9, 8}, []int16{0, 100, 200}, 0, 1, 1, 1},
		{[]int16{10, 9, 8}, []int16{0, 100, 200}, 2, 0, 0, 3},
	} {
		got := haScoreOutcome(tc.scores, tc.coins, tc.focal)
		if got.Rank != tc.rank || got.Win != tc.win || got.Midrank != tc.mid {
			t.Fatal("score/coin/tie utility", got, tc)
		}
	}
	ci := haInterval([]float64{.4, .4, .4, .4})
	if ci.Mean != .4 || ci.Low != .4 || ci.High != .4 {
		t.Fatal("constant blocks gained artificial uncertainty", ci)
	}
	if haInterval([]float64{0, 1}) != haInterval([]float64{0, 1}) {
		t.Fatal("bootstrap is not deterministic")
	}
}
func TestHATournamentFixedDealsAndRotations(t *testing.T) {
	h, err := haNewHeader("development", 3, []int{3, 4, 5, 6, 7}, "all")
	if err != nil {
		t.Fatal(err)
	}
	for n := 3; n <= 7; n++ {
		plan := haPlan(n, "all")
		if len(plan) != 2*n*n+5*n {
			t.Fatal("missing treatment", n, len(plan))
		}
		ids := map[string]bool{}
		for _, c := range plan {
			if ids[c.ID] {
				t.Fatal("duplicate treatment", c.ID)
			}
			ids[c.ID] = true
			for s := 0; s < n; s++ {
				if s != c.Focal && c.Candidate[s] != c.Baseline[s] {
					t.Fatal("paired background changed", c)
				}
			}
		}
		for focal := 0; focal < n; focal++ {
			for offset := 0; offset < n-1; offset++ {
				for _, reflected := range []bool{false, true} {
					found := false
					for _, c := range plan {
						if c.Name == "mixed-greedy-apprentice" && c.Focal == focal && c.Offset == offset && c.Reflected == reflected {
							found = true
						}
					}
					if !found {
						t.Fatal("missing mixed rotation/reflection", n, focal, offset, reflected)
					}
				}
			}
		}
		for block := 0; block < 3; block++ {
			g, err := haInitial(h, n, block)
			if err != nil {
				t.Fatal(err)
			}
			again, err := haInitial(h, n, block)
			if err != nil || g != again {
				t.Fatal("deal is not fixed", err)
			}
			if g.Point().Kind != rules.PointTurn {
				t.Fatal("policy can alter materialized sides")
			}
			for s := 0; s < n; s++ {
				if block == 0 && g.State.Cities[s].Side != catalog.Day || block == 1 && g.State.Cities[s].Side != catalog.Night {
					t.Fatal("side strata", block, s)
				}
			}
			val, _ := haNewHeader("validation", 3, h.Seats, "all")
			if haPolicyEntropy(h, n, block, g.Point(), 0) != haPolicyEntropy(val, n, block, g.Point(), 0) {
				t.Fatal("deal root entered policy entropy")
			}
		}
	}
	seen := map[uint64]bool{}
	for _, partition := range []string{"development", "validation", "release"} {
		h, _ := haNewHeader(partition, 100, []int{3, 4, 5, 6, 7}, "all")
		for n := 3; n <= 7; n++ {
			for b := 0; b < 100; b++ {
				seed := haDealSeed(h, n, b)
				if seen[seed] {
					t.Fatal("seed partitions overlap")
				}
				seen[seed] = true
			}
		}
	}
}
func haTestBlock(t *testing.T) (haHeader, haBlock) {
	t.Helper()
	h, err := haNewHeader("development", 1, []int{3}, "all")
	if err != nil {
		t.Fatal(err)
	}
	b, err := haRunBlock(context.Background(), h, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := haValidateBlock(h, b); err != nil {
		t.Fatal(err)
	}
	return h, b
}
func haCloneTestBlock(t *testing.T, b haBlock) haBlock {
	t.Helper()
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var out haBlock
	if err = json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestHATournamentReplayAndCompleteBlocks(t *testing.T) {
	h, b := haTestBlock(t)
	again, err := haRunBlock(context.Background(), h, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Only active time may differ. Keys, outcomes, work, deals and treatment
	// references must survive restart and equivalent-lineup caching unchanged.
	for _, block := range []*haBlock{&b, &again} {
		for gi := range block.Games {
			for _, d := range block.Games[gi].Diagnostics {
				d[2] = 0
			}
		}
	}
	if !reflect.DeepEqual(b, again) {
		t.Fatal("repeat changed semantic results")
	}
	for _, tc := range []struct {
		name, want string
		edit       func(*haBlock)
	}{
		{"missing rotation", "comparison", func(b *haBlock) { b.Comparisons = b.Comparisons[1:] }},
		{"duplicate rotation", "comparison", func(b *haBlock) { b.Comparisons[0] = b.Comparisons[1] }},
		{"missing game", "game", func(b *haBlock) { b.Games = b.Games[1:] }},
		{"duplicate game", "game", func(b *haBlock) { b.Games = append(b.Games, b.Games[0]) }},
		{"wrong lineup", "lineup", func(b *haBlock) { b.Games[0].Lineup[0] = hbot.NumLevels }},
		{"wrong score", "result", func(b *haBlock) { b.Games[0].Result.Scores[0].Total++ }},
		{"incomplete work", "diagnostic", func(b *haBlock) { b.Games[0].Diagnostics[0][3] = 0 }},
		{"partial diagnostics", "diagnostic", func(b *haBlock) { b.Games[0].Diagnostics = b.Games[0].Diagnostics[1:] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := haCloneTestBlock(t, b)
			tc.edit(&bad)
			err := haValidateBlock(h, bad)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal("incomplete/corrupt block accepted", err)
			}
		})
	}
}
func TestHATournamentShardsAndCLI(t *testing.T) {
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "0.jsonl"), filepath.Join(dir, "1.jsonl.gz")}
	for k, path := range paths {
		shard := "0/2"
		if k == 1 {
			shard = "1/2"
		}
		code, _, errOut := simRun(t, "", "tournament", "-game", "haftgane", "-blocks", "2", "-seats", "3", "-suite", "release", "-shard", shard, "-out", path)
		if code != 0 {
			t.Fatal("shard", k, code, errOut)
		}
	}
	a, err := haMerge(paths)
	if err != nil {
		t.Fatal(err)
	}
	b, err := haMerge([]string{paths[1], paths[0]})
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("merge order changed report", err)
	}
	for _, tc := range []struct {
		paths []string
		want  string
	}{
		{paths[:1], "missing"},
		{[]string{paths[0], paths[0], paths[1]}, "duplicate"},
	} {
		if _, err := haMerge(tc.paths); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatal("bad shard set accepted", tc, err)
		}
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	cut := strings.LastIndex(strings.TrimSpace(string(data)), "\n")
	truncated := filepath.Join(dir, "truncated.jsonl")
	if err := os.WriteFile(truncated, data[:cut+1], 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := haMerge([]string{truncated, paths[1]}); err == nil || !strings.Contains(err.Error(), "footer") {
		t.Fatal("truncated stream accepted", err)
	}
	for _, args := range [][]string{
		{"tournament"}, {"tournament", "-game", "duel"}, {"tournament", "-game", "haftgane", "-blocks", "0"},
		{"tournament", "-game", "haftgane", "-seats", "3,3"}, {"tournament", "-game", "haftgane", "-partition", "training"},
		{"tournament-tally", "-game", "haftgane"},
	} {
		if code, _, _ := simRun(t, "", args...); code != 2 {
			t.Fatal("usage", args, code)
		}
	}
}
