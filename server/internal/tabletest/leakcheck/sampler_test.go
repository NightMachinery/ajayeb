package leakcheck_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

func TestSamplerDiagnosticsLeakOracle(t *testing.T) {
	setup := rules.Setup{Seats: 7, Seed: 0x123456abcdef, Options: rules.Options{Sides: rules.SidesDay}}
	g, err := rules.NewGame(setup)
	if err != nil {
		t.Fatal(err)
	}
	truth, err := leakcheck.FromReal(setup, nil)
	if err != nil {
		t.Fatal(err)
	}
	checker := leakcheck.New(truth, leakcheck.Config{})
	for seat := rules.Seat(0); seat < 7; seat++ {
		for _, budget := range []int{0, rules.MaxSamplerChecks} {
			sample, err := bot.SampleRoot(context.Background(), g.Observe(seat), g.Legal(seat), 91, budget)
			if err != nil {
				t.Fatal(err)
			}
			// RootSample's hidden state is unexported. Calibrators export only these
			// counters; neither success nor failure diagnostics carry private facts.
			data, err := json.Marshal(sample.Counters)
			if err != nil {
				t.Fatal(err)
			}
			age, err := bot.ChooseAge(context.Background(), g.Observe(seat), g.Legal(seat), 91, bot.AgeWork{Attempts: 2, Candidates: 4, SamplerChecks: budget})
			if err != nil {
				t.Fatal(err)
			}
			// Calibration exports counters, not raw root keys or simulated cards.
			ageData, err := json.Marshal(age.Counters)
			if err != nil {
				t.Fatal(err)
			}
			if leaks := checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Viewer: int(seat), Data: ageData}); len(leaks) != 0 {
				t.Fatal("Age diagnostics", leaks)
			}
			if leaks := checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Viewer: int(seat), Data: data}); len(leaks) != 0 {
				t.Fatal(leaks)
			}
		}
	}
	bad, _ := json.Marshal(map[string]string{"seed": "123456abcdef"})
	if len(checker.Check(leakcheck.Doc{Kind: leakcheck.Frame, Data: bad})) == 0 {
		t.Fatal("leak oracle negative control did not fail")
	}
}
