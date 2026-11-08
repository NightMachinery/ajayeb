package tableroom

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
)

// A changed golden means bumping ScholarPresetVersion or bot.PolicyVersion
// (as appropriate) and keeping the old implementation for stored rooms. Do
// not merely regenerate this file after retuning a feature or the seed mixer.
// These development fixtures pin actual keys and counters, not strength.
func TestFrozenBotDecisionGolden(t *testing.T) {
	if RaceEnabled {
		t.Skip("normal suite pins deterministic policy outputs; race tests cover actual Scholar workers")
	}
	type fixture struct {
		Seats int    `json:"seats"`
		Seed  uint64 `json:"seed"`
		Sides string `json:"sides"`
		Turn  int    `json:"turn"`
		Seat  int    `json:"seat"`
	}
	type row struct {
		Fixture     fixture         `json:"fixture"`
		Point       string          `json:"point"`
		Preset      BotPreset       `json:"preset"`
		Entropy     uint64          `json:"entropy"`
		Key         string          `json:"key"`
		Counters    bot.Counters    `json:"counters"`
		AgeCounters bot.AgeCounters `json:"ageCounters"`
	}
	rows := []row{}
	for _, f := range []fixture{
		{3, 1971, "day", 1, 1}, {3, 1971, "day", 3, 1},
		{7, 814, "day", 1, 5}, {7, 814, "day", 3, 5},
		{3, 941, "choose", 0, 2},
	} {
		su, err := recordSetup(SetupRecord{Seats: f.Seats, Seed: f.Seed, Sides: f.Sides})
		if err != nil {
			t.Fatal(err)
		}
		g, err := newGame(su)
		if err != nil {
			t.Fatal(err)
		}
		// Reach turn three without using any policy under test. Every seat takes
		// the engine's first offered action at each preceding simultaneous point.
		for f.Turn > 1 && int(g.State.Turn) < f.Turn {
			b := batch{Point: g.Point().ID, Picks: []pick{}}
			for _, seat := range g.Point().Required.Seats() {
				b.Picks = append(b.Picks, pick{Seat: seat, Key: g.Legal(seat)[0].Key()})
			}
			g, _, err = g.Resolve(b)
			if err != nil {
				t.Fatal(err)
			}
		}
		if int(g.State.Turn) != f.Turn {
			t.Fatal("wrong golden point", g.Point())
		}
		o, cs := g.Observe(seatNum(f.Seat)), g.Legal(seatNum(f.Seat))
		recipes := []BotPreset{releasedPresets()[1], legacyPresets()[1], legacyPresets()[0]}
		// Both tape domains pin the seed mixer independently of the release
		// mapping: set-2 Scholar and set-3 Greedy are tape fixtures, not room levels.
		for _, family := range []string{"haftgane-bot-2", "haftgane-bot-3"} {
			for _, preset := range recipes {
				preset.Policy = family
				entropy := botTape(f.Seed, g.Point().ID, f.Seat, o.Fingerprint(), preset)
				d, err := runBot(context.Background(), botInput{row: WorkRow{Seat: f.Seat, Point: g.Point().ID, BotPreset: preset}, observation: o, offered: cs, entropy: entropy})
				if err != nil || !d.Complete || d.fallback != "" {
					t.Fatal("golden policy incomplete", f, preset, d, err)
				}
				rows = append(rows, row{f, g.Point().ID, preset, entropy, d.Key, d.Counters, d.ageCounters})
			}
		}
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join("testdata", "frozen-bot-decisions.golden.json")
	if os.Getenv("UPDATE_HAFTGANE_BOT_GOLDEN") == "1" {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, data) {
		t.Fatal("frozen bot decisions changed; version the changed policy and preserve stored-room behavior before updating UPDATE_HAFTGANE_BOT_GOLDEN=1")
	}
	t.Logf("%d frozen decision rows (ten full Scholar calls)", len(rows))
}
