package bot

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"

// ScholarPresetVersion freezes the complete calibrated policy: Age horizon,
// 3/4 rank plus 1/4 win reward (scale 3360), the four rollout preferences,
// key noise 1500, shortlist ordering and the work below. Any change to that
// recipe needs a new version and new validation/held-out evidence. Keep old
// implementations while stored rooms still use them.
const ScholarPresetVersion = 1

const (
	ScholarAttempts      = 2
	ScholarCandidates    = 8
	ScholarSamplerChecks = rules.MaxSamplerChecks
)

// ScholarWork returns a fresh value, so no caller can mutate the frozen row.
func ScholarWork() AgeWork {
	return AgeWork{Attempts: ScholarAttempts, Candidates: ScholarCandidates, SamplerChecks: ScholarSamplerChecks}
}
