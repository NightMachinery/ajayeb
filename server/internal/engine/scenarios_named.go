package engine

import (
	"sort"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file is the fixed set of named scenarios: starting positions built
// for a single rules flow, each with a Trigger script of action keys and an
// Expect sentence describing what that script should lead to. They serve
// three callers: this package's own tests (scenarios_named_test.go runs each
// Trigger and checks the state it predicts, in code rather than by parsing
// Expect), the dev endpoint that Playwright and manual testing use to start
// a game from a named position (POST /api/dev/rooms {mode, seed, scenario}),
// and docs/dev/scenarios.md, which lists them for a person.
//
// NamedScenario(name) builds a fresh *Scenario (a scenario value is consumed
// by NewGameFromScenario, so callers never share one), and
// NamedScenarioNames lists every name the dev endpoint accepts, sorted.
//
// Every scenario below gives its seat the wonders, cities, tokens, pawn
// position and structure its flow needs, and nothing else; wonderFill fills
// the rest of a seat's four wonders with the lowest-numbered ones nobody
// named, exactly as enginetest.Builder does, but this file cannot import
// enginetest (enginetest imports this package, so that would cycle).

// NamedScenario returns a fresh copy of the named scenario, or nil if name is
// not one NamedScenarioNames lists.
func NamedScenario(name string) *Scenario {
	for _, e := range namedScenarios {
		if e.name == name {
			return e.build()
		}
	}
	return nil
}

// NamedScenarioNames lists the named scenarios, sorted: the dev endpoint's
// catalog of what "scenario" can name.
func NamedScenarioNames() []string {
	names := make([]string, len(namedScenarios))
	for i, e := range namedScenarios {
		names[i] = e.name
	}
	sort.Strings(names)
	return names
}

// namedScenarios lists every named scenario and the function that builds it.
// A slice, not a map keyed by name: rules-logic files in this package may
// not use maps (TestEngineImports), and this file is not on their short
// exemption list, so it keeps the same rule even though it is scaffolding
// rather than a rule itself.
var namedScenarios = []struct {
	name  string
	build func() *Scenario
}{
	{"mausoleum", scenarioMausoleum},
	{"mausoleum-empty", scenarioMausoleumEmpty},
	{"great-library", scenarioGreatLibrary},
	{"zeus-destroy", scenarioZeusDestroy},
	{"zeus-no-target", scenarioZeusNoTarget},
	{"circus-maximus", scenarioCircusMaximus},
	{"science-pair", scenarioSciencePair},
	{"science-supremacy", scenarioScienceSupremacy},
	{"law-supremacy", scenarioLawSupremacy},
	{"military-supremacy", scenarioMilitarySupremacy},
	{"military-via-wonder", scenarioMilitaryViaWonder},
	{"looting-double", scenarioLootingDouble},
	{"civilian-end", scenarioCivilianEnd},
	{"tie-blue", scenarioTieBlue},
	{"draw", scenarioDraw},
	{"theology-chain", scenarioTheologyChain},
	{"appian-economy", scenarioAppianEconomy},
	{"age-transition-chooser", scenarioAgeTransitionChooser},
	{"extra-turn-lost", scenarioExtraTurnLost},
}

// seatPtr and intPtr fill the Scenario pointer fields from a literal.
func seatPtr(s Seat) *Seat { return &s }
func intPtr(n int) *int    { return &n }

// wonderFill completes seat 0's and seat 1's named wonders (up to four each)
// with the lowest-numbered wonders nobody named, and returns the Wonders
// value that skips the draft. It mirrors enginetest.Builder's wonder filling
// without importing enginetest (see the file comment).
func wonderFill(seat0, seat1 []string) *[2][WondersPerPlayer]string {
	given := [2][]string{seat0, seat1}
	var out [2][WondersPerPlayer]string
	var used [catalog.NumWonders + 1]bool
	for seat := range given {
		for _, slug := range given[seat] {
			if w, ok := catalog.WonderBySlug(slug); ok {
				used[w] = true
			}
		}
	}
	next := catalog.WonderID(1)
	for seat := range given {
		copy(out[seat][:], given[seat])
		for k := len(given[seat]); k < WondersPerPlayer; k++ {
			for int(next) <= catalog.NumWonders && used[next] {
				next++
			}
			used[next] = true
			out[seat][k] = catalog.Wonder(next).Slug
		}
	}
	return &out
}

// scenarioMausoleum: the Mausoleum built with a card in the discard pile.
func scenarioMausoleum() *Scenario {
	return &Scenario{
		Name:       "mausoleum",
		Age:        2,
		Wonders:    wonderFill([]string{"mausoleum"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Discard:    []string{"baths"},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:mausoleum:rostrum", "revive:baths"},
		Expect: "building the Mausoleum (tucking Rostrum) with Baths in the " +
			"discard pile opens the pending rebuild choice; reviving Baths " +
			"builds it for free, and the turn then passes to seat 1.",
	}
}

// scenarioMausoleumEmpty: the Mausoleum built with nothing in the discard.
func scenarioMausoleumEmpty() *Scenario {
	return &Scenario{
		Name:       "mausoleum-empty",
		Age:        2,
		Wonders:    wonderFill([]string{"mausoleum"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:mausoleum:rostrum"},
		Expect: "an empty discard pile skips the Mausoleum's rebuild choice " +
			"(EvEffectSkipped); the wonder is still built and the turn passes.",
	}
}

// scenarioGreatLibrary: the Great Library draws the front of a fixed box
// (Law first) and the drawer keeps it.
func scenarioGreatLibrary() *Scenario {
	return &Scenario{
		Name:        "great-library",
		Age:         2,
		Wonders:     wonderFill([]string{"great-library"}, nil),
		Players:     [2]ScenarioPlayer{{Coins: intPtr(40)}},
		BoardTokens: []string{"agriculture", "architecture", "economy", "masonry", "mathematics"},
		BoxOrder:    []string{"law", "philosophy", "strategy"},
		Accessible:  []string{"rostrum"},
		ToMove:      seatPtr(0),
		Trigger:     []string{"wonder:great-library:rostrum", "token:law"},
		Expect: "building the Great Library draws the front three tokens of " +
			"the box, Law first; keeping Law takes it and passes the turn, " +
			"with Philosophy and Strategy returned to the box.",
	}
}

// scenarioZeusDestroy: the Statue of Zeus destroys the opponent's one brown
// card.
func scenarioZeusDestroy() *Scenario {
	return &Scenario{
		Name:    "zeus-destroy",
		Age:     2,
		Wonders: wonderFill([]string{"statue-of-zeus"}, nil),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(40)},
			{City: []string{"quarry"}},
		},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:statue-of-zeus:rostrum", "destroy:quarry"},
		Expect: "the Statue of Zeus opens the destroy choice because the " +
			"opponent owns a brown card; destroying Quarry sends it to the " +
			"discard pile at once, dropping its production and the trade " +
			"prices it set.",
	}
}

// scenarioZeusNoTarget: the Statue of Zeus with no brown card to destroy.
func scenarioZeusNoTarget() *Scenario {
	return &Scenario{
		Name:       "zeus-no-target",
		Age:        2,
		Wonders:    wonderFill([]string{"statue-of-zeus"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:statue-of-zeus:rostrum"},
		Expect: "the opponent owns no brown card, so the Statue of Zeus's " +
			"destroy choice is skipped (EvEffectSkipped) and the turn ends " +
			"normally.",
	}
}

// scenarioCircusMaximus: Circus Maximus destroys the opponent's one grey
// card.
func scenarioCircusMaximus() *Scenario {
	return &Scenario{
		Name:    "circus-maximus",
		Age:     2,
		Wonders: wonderFill([]string{"circus-maximus"}, nil),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(40)},
			{City: []string{"glassworks"}},
		},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:circus-maximus:rostrum", "destroy:glassworks"},
		Expect: "Circus Maximus opens the destroy choice because the " +
			"opponent owns a grey card; destroying Glassworks sends it to " +
			"the discard pile at once.",
	}
}

// scenarioSciencePair: a second Quill card opens the board token choice
// without changing the distinct-symbol count.
func scenarioSciencePair() *Scenario {
	return &Scenario{
		Name:       "science-pair",
		Age:        2,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40), City: []string{"scriptorium"}}},
		Accessible: []string{"library"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"construct:library"},
		Expect: "the Library pairs Scriptorium's quill (still one distinct " +
			"symbol): the board token choice opens for seat 0, with the " +
			"board untouched.",
	}
}

// scienceFive lists five cards giving five distinct science symbols (quill,
// mortar, set square, wheel, sundial), leaving globe the only one missing.
// It is catalog data (the card definitions themselves), not shared code.
var scienceFive = []string{"scriptorium", "pharmacist", "workshop", "apothecary", "academy"}

// scenarioScienceSupremacy: a sixth distinct symbol from a card wins at once.
func scenarioScienceSupremacy() *Scenario {
	return &Scenario{
		Name:       "science-supremacy",
		Age:        3,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40), City: scienceFive}},
		Accessible: []string{"university"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"construct:university"},
		Expect: "seat 0 already shows five distinct science symbols; " +
			"building University's globe completes a sixth at once: a " +
			"science victory, with no pending choice in between.",
	}
}

// scenarioLawSupremacy: a science pair that does not add a distinct symbol,
// followed by Law's scales completing the sixth.
func scenarioLawSupremacy() *Scenario {
	return &Scenario{
		Name: "law-supremacy",
		Age:  3,
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(40), City: []string{"scriptorium", "pharmacist", "workshop", "apothecary", "study"}},
		},
		BoardTokens: []string{"agriculture", "architecture", "economy", "masonry", "law"},
		Accessible:  []string{"academy"},
		ToMove:      seatPtr(0),
		Trigger:     []string{"construct:academy", "token:law"},
		Expect: "Academy pairs Study's sundial (still five distinct " +
			"symbols) and opens the board token choice; taking Law adds " +
			"the scales as a sixth distinct symbol, winning the game, " +
			"though no card ever showed six.",
	}
}

// scenarioMilitarySupremacy: a two-shield card reaches the capital.
func scenarioMilitarySupremacy() *Scenario {
	return &Scenario{
		Name:       "military-supremacy",
		Age:        2,
		Players:    [2]ScenarioPlayer{{Coins: intPtr(40)}},
		Accessible: []string{"walls"},
		Pawn:       intPtr(7),
		ToMove:     seatPtr(0),
		Trigger:    []string{"construct:walls"},
		Expect: "the pawn stands two spaces from seat 1's capital; Walls' " +
			"two shields reach it at once, for a military victory to seat 0.",
	}
}

// scenarioMilitaryViaWonder: a wonder's shield reaches the capital before its
// own special choice can open.
func scenarioMilitaryViaWonder() *Scenario {
	return &Scenario{
		Name:    "military-via-wonder",
		Age:     2,
		Wonders: wonderFill([]string{"statue-of-zeus"}, nil),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(40)},
			{City: []string{"lumber-yard"}},
		},
		Accessible: []string{"rostrum"},
		Pawn:       intPtr(8),
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:statue-of-zeus:rostrum"},
		Expect: "the pawn stands one space from seat 1's capital; the " +
			"Statue of Zeus's shield reaches it before its destroy choice " +
			"can open, so the game ends by military victory even though " +
			"the opponent owns a brown card to destroy.",
	}
}

// scenarioLootingDouble: four shields in one move take both of a side's
// looting tokens.
func scenarioLootingDouble() *Scenario {
	return &Scenario{
		Name: "looting-double",
		Age:  3,
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(40), Tokens: []string{"strategy"}},
			{Coins: intPtr(20)},
		},
		Accessible: []string{"arsenal"},
		Pawn:       intPtr(2),
		ToMove:     seatPtr(0),
		Trigger:    []string{"construct:arsenal"},
		Expect: "Strategy adds a shield to the Arsenal's three: the four " +
			"shields push the pawn from 2 to 6, taking both of seat 1's " +
			"looting tokens on the way, 2 coins then 5.",
	}
}

// scenarioCivilianEnd: the last Age III card ends the game, decided by coins
// with no tie.
func scenarioCivilianEnd() *Scenario {
	return &Scenario{
		Name: "civilian-end",
		Age:  3,
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(9)},
			{Coins: intPtr(0)},
		},
		OnlyLeft: []string{"academy"},
		ToMove:   seatPtr(0),
		Trigger:  []string{"discard:academy"},
		Expect: "discarding the last Age III card ends the game; seat 0's " +
			"coins decide a civilian victory with no tie.",
	}
}

// scenarioTieBlue: equal totals at game end, broken by blue-card points.
func scenarioTieBlue() *Scenario {
	return &Scenario{
		Name: "tie-blue",
		Age:  3,
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(0), City: []string{"theater"}},
			{Coins: intPtr(9)},
		},
		OnlyLeft: []string{"academy"},
		ToMove:   seatPtr(1),
		Trigger:  []string{"discard:academy"},
		Expect: "seat 1 discards the last Age III card for 2 coins, tying " +
			"the total score at 3 each; seat 0's Theater breaks the tie on " +
			"blue-card points.",
	}
}

// scenarioDraw: equal totals and equal blue points: a shared victory.
func scenarioDraw() *Scenario {
	return &Scenario{
		Name: "draw",
		Age:  3,
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(9)},
			{Coins: intPtr(9)},
		},
		OnlyLeft: []string{"academy"},
		ToMove:   seatPtr(0),
		Trigger:  []string{"discard:academy"},
		Expect: "equal scores and equal blue points (zero, for both) at " +
			"the end of Age III: a shared victory, not a tie-break.",
	}
}

// scenarioTheologyChain: owning Theology grants an extra turn for every
// wonder, even ones with no play-again of their own, so two can be built
// back to back.
func scenarioTheologyChain() *Scenario {
	return &Scenario{
		Name:       "theology-chain",
		Age:        2,
		Wonders:    wonderFill([]string{"pyramids", "colossus"}, nil),
		Players:    [2]ScenarioPlayer{{Coins: intPtr(60), Tokens: []string{"theology"}}},
		Accessible: []string{"rostrum", "statue"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:pyramids:rostrum", "wonder:colossus:statue"},
		Expect: "neither Pyramids nor Colossus grants its own extra turn, " +
			"but Theology grants one for every wonder its owner builds: " +
			"seat 0 builds both without the turn ever passing.",
	}
}

// scenarioAppianEconomy: the opponent's Economy is paid the trade coins of
// the wonder's own cost before the Appian Way's own coin loss applies.
func scenarioAppianEconomy() *Scenario {
	return &Scenario{
		Name:    "appian-economy",
		Age:     2,
		Wonders: wonderFill([]string{"appian-way"}, nil),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(15)},
			{Coins: intPtr(0), Tokens: []string{"economy"}},
		},
		Accessible: []string{"rostrum"},
		ToMove:     seatPtr(0),
		Trigger:    []string{"wonder:appian-way:rostrum"},
		Expect: "seat 0 buys the Appian Way entirely by trade, 10 coins " +
			"that go to the opponent's Economy; the opponent then loses 3 " +
			"to the Appian Way's own effect, ending at 7, never floored to " +
			"0 by paying the loss first.",
	}
}

// scenarioAgeTransitionChooser: the player on the pawn's weaker side chooses
// the next age's starter, not simply the one who took the last card.
func scenarioAgeTransitionChooser() *Scenario {
	return &Scenario{
		Name:     "age-transition-chooser",
		Age:      1,
		OnlyLeft: []string{"theater"},
		Pawn:     intPtr(3),
		ToMove:   seatPtr(0),
		Trigger:  []string{"discard:theater", "starter:0"},
		Expect: "seat 0 takes Age I's last card with the pawn 3 toward " +
			"seat 1's capital; Age II is dealt, and seat 1, the trailing " +
			"side, chooses who starts it, not seat 0 who just moved.",
	}
}

// scenarioExtraTurnLost: an extra turn granted on an age's last card is lost
// when the age ends before it can be spent.
func scenarioExtraTurnLost() *Scenario {
	return &Scenario{
		Name:     "extra-turn-lost",
		Age:      1,
		Wonders:  wonderFill([]string{"sphinx"}, nil),
		Players:  [2]ScenarioPlayer{{Coins: intPtr(40)}},
		OnlyLeft: []string{"theater"},
		ToMove:   seatPtr(0),
		Trigger:  []string{"wonder:sphinx:theater"},
		Expect: "the Sphinx grants an extra turn on Age I's last card, but " +
			"the structure is now empty: the extra turn is lost " +
			"(EvExtraTurnLost) and Age II is dealt before seat 0, the " +
			"centred pawn's last active player, chooses its starter.",
	}
}
