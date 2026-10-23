package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func TestCityCachesCanonicalProducers(t *testing.T) {
	// Insert backwards: public ids, not acquisition order, identify producers.
	c := City{Board: catalog.BoardAlexandria, Side: catalog.Night, Stages: 2, Coins: 17}
	for _, id := range []catalog.DesignID{45, 44, 31, 24, 23, 22, 21, 11, 10, 9, 6, 4, 3, 1} {
		c.Buildings.Add(id)
	}
	got, err := cityCaches(c)
	if err != nil {
		t.Fatal(err)
	}
	brown := [MaxForSaleChoice]catalog.ResMask{
		catalog.Clay.Mask() | catalog.Ore.Mask(),
		catalog.Stone.Mask() | catalog.Clay.Mask(),
		catalog.Wood.Mask() | catalog.Ore.Mask(),
		catalog.Stone.Mask() | catalog.Ore.Mask(),
		catalog.Wood.Mask() | catalog.Stone.Mask(),
		catalog.Wood.Mask() | catalog.Clay.Mask(),
	}
	var own [MaxChoice]catalog.ResMask
	copy(own[:], brown[:])
	copy(own[6:], []catalog.ResMask{catalog.Raw, catalog.Manufactured, catalog.Raw, catalog.Manufactured})
	if got.NChoice != 10 || got.NChoiceForSale != 6 || got.Choice != own || got.ChoiceForSale != brown {
		t.Fatalf("producer order: own %v sale %v", got.Choice, got.ChoiceForSale)
	}
	fixed := [catalog.NumResources]uint8{catalog.Wood: 2, catalog.Glass: 2}
	if got.Produce != fixed || got.ForSale != fixed || got.Discount != ([2]catalog.ResMask{catalog.AllResources, catalog.AllResources}) {
		t.Fatalf("production/discount: %+v", got)
	}
	if got.Coins != 17 || got.Colors[catalog.Brown] != 7 || got.Colors[catalog.Grey] != 1 || got.Colors[catalog.Yellow] != 6 {
		t.Fatal("cache rebuild replayed income or miscounted colours")
	}
	if got.Chains != 1<<catalog.ChainForum|1<<catalog.ChainCaravansery|1<<catalog.ChainLighthouse|1<<catalog.ChainHaven {
		t.Fatalf("chain bits %x", got.Chains)
	}
	again, err := cityCaches(got)
	if err != nil || again != got {
		t.Fatal("cache rebuild not idempotent", err)
	}
}

func TestCityCachesAbilities(t *testing.T) {
	for _, tt := range []struct {
		board                             catalog.BoardID
		side                              catalog.Side
		stages                            uint8
		abilities, shields, scienceChoice uint8
	}{
		{catalog.BoardBabylon, catalog.Night, 2, AbilityPlayLastCard, 0, 2},
		{catalog.BoardBabylon, catalog.Day, 2, 0, 0, 2},
		{catalog.BoardOlympia, catalog.Day, 2, AbilityFreeFirstOfColour, 0, 1},
		{catalog.BoardOlympia, catalog.Night, 2, AbilityFreeFirstTurn | AbilityFreeLastTurn, 0, 1},
		{catalog.BoardRhodos, catalog.Night, 2, 0, 2, 1},
		{catalog.BoardRhodos, catalog.Day, 2, 0, 2, 1},
	} {
		c := City{Board: tt.board, Side: tt.side, Stages: tt.stages, Coins: 9}
		// Three different printed science symbols and the science-choice guild.
		for _, id := range []catalog.DesignID{14, 25, 26, 27, 35, 76} {
			c.Buildings.Add(id)
		}
		got, err := cityCaches(c)
		if err != nil {
			t.Fatal(err)
		}
		if got.Abilities != tt.abilities || got.Shields != tt.shields+3 || got.ScienceChoice != tt.scienceChoice || got.Science != ([catalog.NumSymbols]uint8{catalog.Compass: 1, catalog.Tablet: 1, catalog.Gear: 1}) || got.Coins != 9 {
			t.Fatalf("%s/%s: %+v", tt.board, tt.side, got)
		}
	}
	// An income stage must never run while reconstructing a cache.
	c, err := cityCaches(City{Board: catalog.BoardEphesos, Side: catalog.Night, Stages: 3, Coins: 3})
	if err != nil || c.Coins != 3 {
		t.Fatal("stage income replayed", err)
	}
}

func TestCityCachesBounds(t *testing.T) {
	for _, c := range []City{{}, {Board: 255}, {Board: 1, Side: 255}, {Board: 1, Stages: 1}, {Board: catalog.BoardBabylon, Side: catalog.Night, Stages: 3}} {
		if _, err := cityCaches(c); err == nil {
			t.Fatal("invalid city accepted", c)
		}
	}
	for _, tt := range []struct {
		city   City
		effect catalog.Effect
	}{
		{City{NChoice: MaxChoice}, catalog.ProduceFamily(catalog.Raw)},
		{City{NChoiceForSale: MaxForSaleChoice}, catalog.ProduceChoice(catalog.Wood, catalog.Clay)},
		{City{Shields: 255}, catalog.Shields(1)},
		{City{Produce: [catalog.NumResources]uint8{255}}, catalog.Produce(catalog.Wood, 1)},
		{City{}, catalog.FreeBuild(catalog.NumFreeConditions)},
	} {
		c := tt.city
		if err := cacheEffect(&c, tt.effect); err == nil {
			t.Fatal("cache overflow/invalid condition accepted")
		}
	}
}
