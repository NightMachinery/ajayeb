package tableroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// BotPreset freezes the released algorithm and all deterministic work bounds.
// Room codes are internal aliases until the public Scholar release: easy is
// Random, normal is Apprentice, hard is Greedy. Every policy completes one pass.
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

const policyVersion = "haftgane-bot-1"

func releasedPresets() []BotPreset {
	out := []BotPreset{}
	for _, pair := range []struct {
		level     string
		algorithm bot.Level
	}{{BotEasy, bot.Random}, {BotNormal, bot.Apprentice}, {BotHard, bot.Greedy}} {
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
func workRow(r *Record, point string, seat int) WorkRow {
	preset, _ := frozenPreset(r, r.Seats[seat].Bot)
	return WorkRow{Seat: seat, Point: point, BotPreset: preset}
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
