package tableroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// BotPreset freezes the released algorithm and all deterministic work bounds.
// Each room code plays the policy of its persona where one exists: easy
// (Apprentice) is Apprentice. Until the Scholar policy exists, normal
// (Scholar) and hard (Master) both play Greedy as a stand-in. Random stays a
// simulator baseline (D46). Every policy completes one pass.
type BotPreset struct {
	Policy        string `json:"policy"`
	PolicyVersion int    `json:"policyVersion"`
	Preset        int    `json:"preset"`
	Level         string `json:"level"`
	Algorithm     string `json:"algorithm"`
	Allowance     int    `json:"allowance"`
	MaxChoices    int    `json:"maxChoices"`
	MaxProbes     int    `json:"maxProbes"`
	MaxQuotes     int    `json:"maxQuotes"`
}

// policyVersion names the room's preset set. It changes whenever
// releasedPresets does, so a work row written under another set is refused
// by name. Version 1 mapped easy to Random and normal to Apprentice.
const policyVersion = "haftgane-bot-2"

func releasedPresets() []BotPreset {
	out := []BotPreset{}
	for _, pair := range []struct {
		level     string
		algorithm bot.Level
	}{{BotEasy, bot.Apprentice}, {BotNormal, bot.Greedy}, {BotHard, bot.Greedy}} {
		out = append(out, BotPreset{Policy: policyVersion, PolicyVersion: bot.PolicyVersion, Preset: bot.PresetVersion, Level: pair.level, Algorithm: pair.algorithm.String(), Allowance: int(bot.DefaultWork().Iterations), MaxChoices: rules.MaxTurnChoices, MaxProbes: bot.MaxProbes, MaxQuotes: bot.MaxQuoteCalls})
	}
	return out
}
func frozenPreset(r *Record, level string) (BotPreset, bool) {
	i := slices.IndexFunc(r.BotPresets, func(p BotPreset) bool { return p.Level == level })
	if i < 0 {
		return BotPreset{}, false
	}
	return r.BotPresets[i], true
}

// workRow is the seat's work row at point, from the table's frozen preset for
// the seat's level. It reports false when the table froze no such level.
func workRow(r *Record, point string, seat int) (WorkRow, bool) {
	preset, ok := frozenPreset(r, r.Seats[seat].Bot)
	return WorkRow{Seat: seat, Point: point, BotPreset: preset}, ok
}
func (p BotPreset) work() bot.Work {
	return bot.Work{PolicyVersion: uint16(p.PolicyVersion), PresetVersion: uint16(p.Preset), Iterations: uint16(p.Allowance)}
}
func allowance(level string) (int, bool) {
	for _, p := range releasedPresets() {
		if p.Level == level {
			return p.Allowance, true
		}
	}
	return 0, false
}

// BotFallback is a bounded diagnostic, never an observation or a search cache.
// The accepted key remains only in the normal pending/batch record.
type BotFallback struct {
	At     int    `json:"at"`
	Point  string `json:"point"`
	Seat   int    `json:"seat"`
	Reason string `json:"reason"`
}
