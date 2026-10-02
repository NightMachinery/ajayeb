package enginetest_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

func TestParseStep(t *testing.T) {
	cases := []struct {
		step string
		want engine.Action
	}{
		{"construct:baths", engine.ConstructAction(23)},
		{"build baths", engine.ConstructAction(23)},
		{"construct baths", engine.ConstructAction(23)},
		{"discard baths", engine.DiscardAction(23)},
		{"sell baths", engine.DiscardAction(23)},
		{"wonder mausoleum stable", engine.WonderAction(catalog.WonderMausoleum, 9)},
		{"wonder:mausoleum:stable", engine.WonderAction(catalog.WonderMausoleum, 9)},
		{"rebuild baths", engine.ReviveAction(23)},
		{"revive baths", engine.ReviveAction(23)},
		{"destroy quarry", engine.DestroyAction(5)},
		{"token law", engine.TokenAction(catalog.TokenLaw)},
		{"take law", engine.TokenAction(catalog.TokenLaw)},
		{"starter 1", engine.StarterAction(1)},
		{"start 0", engine.StarterAction(0)},
		{"draft sphinx", engine.DraftAction(catalog.WonderSphinx)},
		{"  draft   sphinx ", engine.DraftAction(catalog.WonderSphinx)},
	}
	for _, c := range cases {
		got, err := enginetest.ParseStep(c.step)
		if err != nil || got != c.want {
			t.Errorf("ParseStep(%q) = %v, %v; want %v", c.step, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "fly baths", "build the-baths", "build", "wonder mausoleum", "starter 2", "Build baths", "construct:baths extra"} {
		if a, err := enginetest.ParseStep(bad); err == nil {
			t.Errorf("ParseStep(%q) = %v, want an error", bad, a)
		}
	}
}

func TestDoAndMustFail(t *testing.T) {
	s := enginetest.New(t, 1).FirstPlayer(0).Accessible("tavern", "theater").State()
	ev := enginetest.Do(t, &s, "discard tavern", "build theater")
	if len(ev) < 2 || ev[0].Kind != engine.EvDiscard || ev[1].Kind != engine.EvBuild {
		t.Fatalf("events %+v", ev)
	}
	if !s.Players[1].City.Has(21) || s.ToMove != 0 {
		t.Fatalf("theater was not built by seat 1: %+v", s.Players[1].City)
	}
	if r := enginetest.MustFail(t, &s, "token law"); r != engine.ReasonWrongPhase {
		t.Fatalf("reason %s", r)
	}
	if enginetest.IsLegal(&s, "discard tavern") || enginetest.IsLegal(&s, "nonsense") {
		t.Fatal("IsLegal accepts a taken card or nonsense")
	}
	keys := enginetest.Keys(&s)
	if len(keys) == 0 || !slices.Contains(keys, "discard:"+s.Slots[firstAccessible(&s)].Card.String()) {
		t.Fatalf("keys %v", keys)
	}
}

func firstAccessible(s *engine.State) int {
	acc := engine.Accessible(s)
	for i := range engine.SlotsPerAge {
		if acc&(1<<i) != 0 {
			return i
		}
	}
	return -1
}

func TestBuilderFillsWonders(t *testing.T) {
	s := enginetest.New(t, 2).Wonders(1, "sphinx", "pyramids").State()
	var got [2][]catalog.WonderID
	for seat := range s.Players {
		for _, ws := range s.Players[seat].Wonders {
			got[seat] = append(got[seat], ws.ID)
		}
	}
	// Seat 0 gets the four lowest wonders nobody was given; seat 1 its two
	// and the next two.
	want := [2][]catalog.WonderID{
		{catalog.WonderColossus, catalog.WonderGreatLighthouse, catalog.WonderTempleOfArtemis, catalog.WonderMausoleum},
		{catalog.WonderSphinx, catalog.WonderPyramids, catalog.WonderGreatLibrary, catalog.WonderPiraeus},
	}
	if !slices.Equal(got[0], want[0]) || !slices.Equal(got[1], want[1]) {
		t.Fatalf("wonders %v, want %v", got, want)
	}
	sc := enginetest.New(t, 2).Name("x").Coins(0, 3).Scenario()
	if sc.Name != "x" || *sc.Players[0].Coins != 3 || sc.Wonders != nil {
		t.Fatalf("scenario %+v", sc)
	}
}

func TestPlayoutPolicies(t *testing.T) {
	for seed := uint64(1); seed <= 10; seed++ {
		for _, pols := range [][2]enginetest.Policy{
			{enginetest.NewRandomPolicy(seed), enginetest.NewRandomPolicy(seed + 100)},
			{enginetest.NewSimplePolicy(seed), enginetest.NewRandomPolicy(seed)},
			{&enginetest.SimplePolicy{RNG: engine.NewRNG(seed), Explore: 1000}, enginetest.NewSimplePolicy(seed + 7)},
		} {
			s := engine.NewGame(seed, engine.Options{})
			n := 0
			err := enginetest.Playout(&s, pols, 200, func(a engine.Action, ev []engine.Event) error {
				n++
				return engine.CheckInvariants(&s)
			})
			if err != nil || !s.Over() || n != int(s.Ply) {
				t.Fatalf("seed %d: %v (over %v, %d actions)", seed, err, s.Over(), n)
			}
		}
	}
	s := engine.NewGame(3, engine.Options{})
	pol := enginetest.NewRandomPolicy(3)
	if err := enginetest.Playout(&s, [2]enginetest.Policy{pol, pol}, 10, nil); err == nil || s.Ply != 10 {
		t.Fatalf("a 10-action limit: %v at ply %d", err, s.Ply)
	}
}

// TestSimplePolicyBuilds checks that the simple policy prefers building to
// discarding.
func TestSimplePolicyBuilds(t *testing.T) {
	builds, discards := 0, 0
	for seed := uint64(1); seed <= 20; seed++ {
		s := engine.NewGame(seed, engine.Options{})
		pol := enginetest.NewSimplePolicy(seed)
		err := enginetest.Playout(&s, [2]enginetest.Policy{pol, pol}, 200, func(a engine.Action, _ []engine.Event) error {
			switch a.Type {
			case engine.ActConstructBuilding, engine.ActConstructWonder:
				builds++
			case engine.ActDiscardBuilding:
				discards++
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if builds <= discards {
		t.Fatalf("%d builds and %d discards", builds, discards)
	}
}
