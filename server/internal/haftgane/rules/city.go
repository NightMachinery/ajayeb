package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// cityCaches is the sole derived-cache writer. Symbols are indexed by board
// start, then ascending building DesignID/effect index, then ascending stage
// number/effect index. ForSale uses the same order with nonsaleable symbols
// omitted. The view can reproduce this order from the public city to identify
// a PaymentUnit's producer. Income and scoring are not replayed by this helper.
func cityCaches(c City) (City, error) {
	board, ok := catalog.BoardByID(c.Board)
	if !ok {
		return City{}, fmt.Errorf("rules: invalid board %d", c.Board)
	}
	if c.Side > catalog.Night || c.Stages > MaxStages || (c.Side == catalog.NoSide && c.Stages != 0) || (c.Side != catalog.NoSide && c.Stages > board.Sides[c.Side-1].NumStages) {
		return City{}, fmt.Errorf("rules: invalid side/stage count")
	}
	out := City{Board: c.Board, Side: c.Side, Stages: c.Stages, Tucked: c.Tucked, Buildings: c.Buildings, Coins: c.Coins, Victories: c.Victories, Defeats: c.Defeats, Kept: c.Kept}
	out.Produce[board.Start] = 1
	out.ForSale[board.Start] = 1
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if !c.Buildings.Has(id) {
			continue
		}
		d, exists := catalog.DesignByID(id)
		if !exists {
			return City{}, fmt.Errorf("rules: unregistered building")
		}
		if err := cacheAdd(&out.Colors[d.Color], 1); err != nil {
			return City{}, err
		}
		out.Chains |= catalog.T.ChainOutBits[id]
		for _, e := range d.Effects {
			if err := cacheEffect(&out, e); err != nil {
				return City{}, err
			}
		}
	}
	if c.Side != catalog.NoSide {
		for stage := uint8(0); stage < c.Stages; stage++ {
			for _, e := range board.Sides[c.Side-1].Stages[stage].Effects {
				if err := cacheEffect(&out, e); err != nil {
					return City{}, err
				}
			}
		}
	}
	return out, nil
}

func cacheAdd(dst *uint8, n uint8) error {
	if uint16(*dst)+uint16(n) > 255 {
		return fmt.Errorf("rules: city cache overflow")
	}
	*dst += n
	return nil
}
func cacheEffect(c *City, e catalog.Effect) error {
	if e.Kind.Timings()&catalog.Continuous == 0 {
		return nil
	}
	switch e.Kind {
	case catalog.EffProduce:
		if err := cacheAdd(&c.Produce[e.Res], e.N); err != nil {
			return err
		}
		if e.ForSale {
			return cacheAdd(&c.ForSale[e.Res], e.N)
		}
	case catalog.EffProduceChoice:
		if c.NChoice == MaxChoice {
			return fmt.Errorf("rules: too many own choice producers")
		}
		c.Choice[c.NChoice] = e.Mask
		c.NChoice++
		if e.ForSale {
			if c.NChoiceForSale == MaxForSaleChoice {
				return fmt.Errorf("rules: too many saleable choice producers")
			}
			c.ChoiceForSale[c.NChoiceForSale] = e.Mask
			c.NChoiceForSale++
		}
	case catalog.EffTradeDiscount:
		if e.Scope&catalog.Left != 0 {
			c.Discount[0] |= e.Mask
		}
		if e.Scope&catalog.Right != 0 {
			c.Discount[1] |= e.Mask
		}
	case catalog.EffShields:
		return cacheAdd(&c.Shields, e.N)
	case catalog.EffScience:
		return cacheAdd(&c.Science[e.Symbol], 1)
	case catalog.EffScienceChoice:
		return cacheAdd(&c.ScienceChoice, 1)
	case catalog.EffPlayLastCard:
		c.Abilities |= AbilityPlayLastCard
	case catalog.EffFreeBuild:
		switch e.When {
		case catalog.FirstOfColour:
			c.Abilities |= AbilityFreeFirstOfColour
		case catalog.FirstTurnOfAge:
			c.Abilities |= AbilityFreeFirstTurn
		case catalog.LastTurnOfAge:
			c.Abilities |= AbilityFreeLastTurn
		default:
			return fmt.Errorf("rules: unknown free-build condition")
		}
	default:
		return fmt.Errorf("rules: unhandled continuous effect %s", e.Kind)
	}
	return nil
}
