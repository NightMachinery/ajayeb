package tableroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
)

// BotPreset freezes the released algorithm and every deterministic work bound.
// New tables offer Apprentice and Scholar. Scholar uses the calibrated Age row
// in full for every seat. Legacy set 2 retains its Greedy normal/hard policies.
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
	Candidates    int    `json:"candidates"`
	SamplerChecks int    `json:"samplerChecks"`
}

// policyVersion names the room's preset set and changes whenever its mapping
// or work changes. Version 1 mapped easy to Random and normal to Apprentice;
// version 2 mapped easy to Apprentice and normal/hard to Greedy. Keep set 2
// supported verbatim: restore, switches and new points use the table's rows.
const policyVersion = "haftgane-bot-3"

// Keep historical versions literal, rather than silently adopting new defaults.
// TestBaselinePresetsMatchSupportedWork fails if Choose stops accepting these
// rows: a version bump must retain dispatch for stored baseline policies.
func legacyPresets() []BotPreset {
	out := []BotPreset{}
	for _, pair := range []struct{ level, algorithm string }{{BotEasy, "apprentice"}, {BotNormal, "greedy"}, {BotHard, "greedy"}} {
		out = append(out, BotPreset{Policy: "haftgane-bot-2", PolicyVersion: 1, Preset: 1, Level: pair.level, Algorithm: pair.algorithm, Allowance: 1, MaxChoices: 217, MaxProbes: 12, MaxQuotes: 2616})
	}
	return out
}
func releasedPresets() []BotPreset {
	out := legacyPresets()[:2]
	for i := range out {
		out[i].Policy = policyVersion
	}
	p := &out[1]
	p.Algorithm, p.Preset = "scholar", bot.ScholarPresetVersion
	p.Allowance, p.Candidates, p.SamplerChecks = bot.ScholarAttempts, bot.ScholarCandidates, bot.ScholarSamplerChecks
	return out
}
func supportedPresets(ps []BotPreset) bool {
	return slices.Equal(ps, releasedPresets()) || slices.Equal(ps, legacyPresets())
}
func (p BotPreset) ageWork() bot.AgeWork {
	return bot.AgeWork{Attempts: uint16(p.Allowance), Candidates: uint16(p.Candidates), SamplerChecks: p.SamplerChecks}
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
