package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	hbot "github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func scTestConfig() scConfig {
	return scConfig{Format: "scholar-calibration/1", Binary: "test", Partition: "development", Epoch: 1, Blocks: 1, Seats: []int{3}, Profiles: []scProfile{{"a0-c4", hbot.AgeWork{Candidates: 4, SamplerChecks: rules.MaxSamplerChecks}}}, Suite: "screen", GOMAXPROCS: 1}
}
func TestScholarCalibrationPlanAndIsolation(t *testing.T) {
	for n := 3; n <= 7; n++ {
		plan := scPlan(n, "all")
		if len(plan) != 2*n+2*n*(n-1) {
			t.Fatal("incomplete offset/reflection plan", n, len(plan))
		}
		var main, inverse, mixed int
		for _, p := range plan {
			a, b := []byte(p.Candidate), []byte(p.Baseline)
			changes := 0
			for seat := range n {
				if a[seat] != b[seat] {
					changes++
					if seat != p.Focal || a[seat] != 'S' || b[seat] != 'G' {
						t.Fatal("unpaired treatment")
					}
				}
			}
			if changes != 1 {
				t.Fatal("not one replacement")
			}
			switch p.Family {
			case "main":
				main++
			case "inverse":
				inverse++
			case "mixed":
				mixed++
			}
		}
		if main != n || inverse != n || mixed != 2*n*(n-1) {
			t.Fatal("coverage")
		}
	}
	c := scTestConfig()
	seen := map[uint64]bool{}
	for _, part := range []string{"development", "validation", "heldout"} {
		c.Partition = part
		h, err := scHeader(c)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 10; i++ {
			seed := haDealSeed(h, 3, i)
			if seen[seed] {
				t.Fatal("partition collision")
			}
			seen[seed] = true
		}
	}
}
func TestScholarCalibrationReplayAndResume(t *testing.T) {
	c := scTestConfig()
	p := c.Profiles[0]
	b, err := scRunBlock(context.Background(), c, p, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = scValidate(c, p, b); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := scBlockPath(dir, p, 3, 0)
	if err = scPublishJSON(path, scEnvelope{b, haDigest(b)}); err != nil {
		t.Fatal(err)
	}
	if err = scPublishJSON(path, scEnvelope{b, haDigest(b)}); !os.IsExist(err) {
		t.Fatal("overwrote completed evidence", err)
	}
	dataCopy, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := scBlockPath(dir, p, 3, 1)
	if err = os.WriteFile(duplicate, dataCopy, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = scReadBlock(duplicate, c, p); err == nil {
		t.Fatal("copied block inflated independent sample count")
	}
	read, err := scReadBlock(path, c, p)
	if err != nil || !reflect.DeepEqual(read, b) {
		t.Fatal("resume", err)
	}
	summary, err := scSummarize(dir, c)
	if err != nil || !summary.Complete || summary.Eligible || summary.GatesPassed || len(summary.Combined) != 1 || summary.Combined[0].IntervalEstimable {
		t.Fatal("pilot mislabeled heldout", summary, err)
	}
	changed := c
	changed.Profiles = []scProfile{{p.ID, hbot.AgeWork{Attempts: 1, Candidates: 4, SamplerChecks: rules.MaxSamplerChecks}}}
	if _, err = scReadBlock(path, changed, changed.Profiles[0]); err == nil {
		t.Fatal("resumed changed work")
	}
	b.Games[0].Batches[0].Picks[0].Key = "sell:invalid"
	bad := filepath.Join(dir, "bad.json")
	if err = scPublishJSON(bad, scEnvelope{b, haDigest(b)}); err != nil {
		t.Fatal(err)
	}
	if _, err = scReadBlock(bad, c, p); err == nil {
		t.Fatal("accepted corrupted replay")
	}
	data, _ := json.Marshal(scEnvelope{b, "wrong"})
	if err = os.WriteFile(bad, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = scReadBlock(bad, c, p); err == nil {
		t.Fatal("accepted wrong digest")
	}
}
func TestScholarCalibrationProfileAndQuantiles(t *testing.T) {
	for _, spec := range []string{"1", "-1x8", "1x0", "65536x8", "1x218", "1x8,1x8"} {
		if _, err := scProfiles(spec, rules.MaxSamplerChecks); err == nil {
			t.Fatal("bad profile", spec)
		}
	}
	q := scQuantiles("test", []int64{4, 1, 2, 3})
	if q.P50 != 2 || q.P95 != 4 || q.P99 != 4 {
		t.Fatal(q)
	}
	if got := scInterval([]float64{0.9}); got.Low != 0 || got.High != 1 {
		t.Fatal("one block has fake precision", got)
	}
}

func TestScholarCalibrationClosedHeldout(t *testing.T) {
	c := scTestConfig()
	c.Partition = "heldout"
	c.Blocks = 100
	c.Seats = []int{3, 4, 5, 6, 7}
	c.Suite = "all"
	c.Frozen = true
	// No heldout seed is executed: the empty ledger models an unfinished shard.
	summary, err := scSummarize(t.TempDir(), c)
	if err != nil || summary.Complete || summary.Eligible || summary.GatesPassed || len(summary.Contrasts) != 0 || len(summary.Combined) != 0 {
		t.Fatal("interim heldout strength exposed", err)
	}
}
