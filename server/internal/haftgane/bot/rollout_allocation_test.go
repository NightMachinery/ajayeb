package bot

import (
	"context"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Preserve the previous string-based selector as an independent selection
// reference. It shares the established public Rank API, not the new selector.
func scholarStringPolicy(t testing.TB, o rules.Observation, offered []rules.Choice, pref rolloutPreference, entropy uint64) rules.Action {
	t.Helper()
	var buf [rules.MaxTurnChoices]Ranked
	ranked, _, err := Rank(context.Background(), o, offered, buf[:0])
	if err != nil {
		t.Fatal(err)
	}
	var chosen rules.Action
	var best int64
	bestKey := ""
	for _, x := range ranked {
		var choice rules.Choice
		for _, c := range offered {
			if c.Key() == x.Key {
				choice = c
				break
			}
		}
		value := x.Value
		switch pref {
		case economicPreference:
			value += x.Features.Economy + x.Features.Liquidity + x.Features.Affordability + x.Features.Chains
		case sciencePreference:
			value += x.Features.Science
			if sciencePlacement(choice) {
				value += x.Features.Points
			}
		case militaryPreference:
			value += x.Features.Military
		}
		r := rng.NewRNG2(entropy, rng.Digest([]byte(x.Key)))
		value += int64(r.IntN(2*ApprenticeTemperature+1) - ApprenticeTemperature)
		if bestKey == "" || value > best || value == best && x.Key < bestKey {
			chosen = choice.Action
			best = value
			bestKey = x.Key
		}
	}
	return chosen
}
func TestScholarPolicyStorageIdentity(t *testing.T) {
	calls := 0
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 3; seed++ {
			history := ageHistory(t, rules.Setup{Seats: n, Seed: 0x7363687363726174 + seed, Options: rules.Options{Sides: rules.SideRule(seed)}}, seed+81, true)
			for _, g := range history {
				for seat := rules.Seat(0); seat < rules.Seat(n); seat++ {
					if !g.Point().Required.Has(seat) {
						continue
					}
					o, cs := g.Observe(seat), g.Legal(seat)
					for pref := balancedPreference; pref < numRolloutPreferences; pref++ {
						entropy := uint64(g.State.Points)*17 + uint64(seat)
						want := scholarStringPolicy(t, o, cs, pref, entropy)
						got, err := rolloutPolicy(context.Background(), o, cs, pref, entropy)
						if err != nil || got != want {
							t.Fatal("storage changed selection", n, seed, g.Point(), seat, pref, err)
						}
						for i, j := 0, len(cs)-1; i < j; i, j = i+1, j-1 {
							cs[i], cs[j] = cs[j], cs[i]
						}
						reverse, err := rolloutPolicy(context.Background(), o, cs, pref, entropy)
						if err != nil || reverse != want {
							t.Fatal("offer order changed selection", err)
						}
						calls++
					}
				}
			}
		}
	}
	t.Log("identical string and buffered policy selections", calls)
	g := ageGame(t, 7, 814)
	o, cs := g.Observe(0), g.Legal(0)
	if n := testing.AllocsPerRun(20, func() {
		_, err := rolloutPolicy(context.Background(), o, cs, balancedPreference, 91)
		if err != nil {
			panic(err)
		}
	}); n != 0 {
		t.Fatal("rollout policy allocates", n)
	}
}

func TestScholarDecisionStorageIdentity(t *testing.T) {
	legacy := func(ctx context.Context, o rules.Observation, cs []rules.Choice, pref rolloutPreference, entropy uint64) (rules.Action, error) {
		if err := ctx.Err(); err != nil {
			return rules.Action{}, err
		}
		return scholarStringPolicy(t, o, cs, pref, entropy), nil
	}
	positions := []rules.Game{ageTail(t)}
	for _, g := range scholarLaterPositions(t) {
		if g.State.Age == 2 {
			positions = append(positions, g)
		}
	}
	for _, g := range positions {
		seat := rules.Seat(0)
		for !g.Point().Required.Has(seat) {
			seat++
		}
		o, cs := g.Observe(seat), g.Legal(seat)
		work := ScholarWork()
		want, err := chooseAge(context.Background(), o, cs, 81, work, sampledState, legacy)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ChooseAge(context.Background(), o, cs, 81, work)
		if err != nil || got != want {
			t.Fatal("storage changed complete decision", g.Point(), seat, err)
		}
	}
}
