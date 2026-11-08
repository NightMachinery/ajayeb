package bot

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func TestScholarPresetRecipe(t *testing.T) {
	// A v1 recipe change must fail loudly, even when rules bounds or another
	// policy's constants change. A new version requires fresh strength evidence.
	if ScholarPresetVersion != 1 || ScholarWork() != (AgeWork{Attempts: 2, Candidates: 8, SamplerChecks: 1048576}) || AgeRewardScale != 3360 || numRolloutPreferences != 4 || ApprenticeTemperature != 1500 || rules.MaxSamplerChecks != 1048576 {
		t.Fatal("Scholar v1 recipe changed without a versioned migration")
	}
}
