package rules

import (
	"fmt"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Scenario is an explicit position builder for tests/development, not a view.
// Start with ScenarioFromGame, edit its value fields to move existing copies,
// then call NewGameFromScenario. Hands/deals/box/packets are sorted and caches
// rebuilt; no card, income or past choice is invented to repair an invalid
// position. Record describes the history the scenario chooses to retain.
type Scenario struct {
	State  State  `json:"state"`
	Record Record `json:"record"`
}

func ScenarioFromGame(g Game) *Scenario { return &Scenario{State: g.State, Record: g.Record} }

func NewGameFromScenario(seed uint64, opts Options, sc *Scenario) (Game, error) {
	if sc == nil {
		return Game{}, fmt.Errorf("rules: nil scenario")
	}
	if err := opts.Validate(); err != nil {
		return Game{}, err
	}
	g := Game{State: sc.State, Record: sc.Record}
	g.State.Seed = seed
	g.State.Opts = opts
	if g.State.Seats < MinSeats || g.State.Seats > MaxSeats {
		return Game{}, fmt.Errorf("rules: invalid scenario seats")
	}
	for s := uint8(0); s < g.State.Seats; s++ {
		city, err := cityCaches(g.State.Cities[s])
		if err != nil {
			return Game{}, err
		}
		g.State.Cities[s] = city
	}
	for s := range g.State.Hands {
		canonicalIDs(g.State.Hands[s][:])
		for age := 0; age < NumAges; age++ {
			canonicalIDs(g.State.Deals[age][s][:])
			canonicalIDs(g.Record.Packets[age][s][:])
		}
	}
	canonicalIDs(g.State.BoxGuilds[:])
	if err := g.CheckInvariants(); err != nil {
		return Game{}, fmt.Errorf("rules: scenario: %w", err)
	}
	return g, nil
}

// canonicalIDs keeps zero padding after the sorted multiset, not before it.
func canonicalIDs(ids []catalog.DesignID) {
	slices.SortFunc(ids, func(a, b catalog.DesignID) int {
		if a == b {
			return 0
		}
		if a == 0 {
			return 1
		}
		if b == 0 {
			return -1
		}
		return int(a) - int(b)
	})
}
