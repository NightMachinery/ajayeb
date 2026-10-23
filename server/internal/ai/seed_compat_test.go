package ai_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
)

// Pin actual policy keys before extracting MixSeed. The inputs include the
// mixed policy seed; the same fixture must pass after the shared wrapper lands.
func TestSeedMixerBotGoldens(t *testing.T) {
	type row struct {
		Seed     uint64 `json:"seed"`
		Position int    `json:"position"`
		Bot      string `json:"bot"`
		Key      string `json:"key"`
	}
	rows := []row{}
	for seed := uint64(101); seed < 105; seed++ {
		for i, state := range aiPositions(t, seed) {
			if state.Over() || i%11 != 0 {
				continue
			}
			policySeed := ai.MixSeed(seed, uint64(state.ToMove))
			bots := []struct {
				name string
				bot  ai.Bot
			}{{"random", ai.NewRandom(policySeed)}, {"greedy", ai.NewGreedy(policySeed, ai.GreedyConfig{})}, {"easy", ai.NewGreedy(policySeed, ai.GreedyConfig{Temperature: ai.EasyTemperature})}}
			for _, b := range bots {
				a := aiChoose(t, b.bot, &state, aiFastGreedy)
				rows = append(rows, row{seed, i, b.name, a.Key()})
			}
		}
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join("testdata", "seed-mixer-bots.golden.json")
	if os.Getenv("UPDATE_DUEL_SEED_COMPAT_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, data) {
		t.Fatal("Duel policy keys changed across the seed-mixer seam")
	}
	t.Logf("%d pre-extraction bot-key golden rows", len(rows))
}
