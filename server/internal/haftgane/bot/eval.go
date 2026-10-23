package bot

import (
	"context"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

type evaluation struct {
	o          rules.Observation
	own        rules.PublicFacts
	neighbours [2]rules.PublicFacts
	stats      rules.Statistics
	probes     [MaxProbes]catalog.Cost
	numProbes  int
	before     Features
}

func remaining(c *rules.Current) int {
	if c.Phase == rules.PhaseOver {
		return 0
	}
	if c.Phase == rules.PhaseSides {
		return 18
	}
	return max(0, (3-int(c.Age))*6+6-min(int(c.Turn), 6))
}
func prepare(o rules.Observation, counters *Counters) (evaluation, bool) {
	e := evaluation{o: o, stats: o.Statistics()}
	var ok bool
	e.own, ok = o.Now.Facts(o.Now.Seat)
	if !ok {
		return e, false
	}
	for i, s := range [2]rules.Seat{o.Now.Seat.Left(o.Now.Seats), o.Now.Seat.Right(o.Now.Seats)} {
		e.neighbours[i], ok = o.Now.Facts(s)
		if !ok {
			return e, false
		}
	}
	add := func(cost catalog.Cost) {
		for _, c := range e.probes[:e.numProbes] {
			if c == cost {
				return
			}
		}
		if e.numProbes < MaxProbes {
			e.probes[e.numProbes] = cost
			e.numProbes++
		}
	}
	for _, id := range o.Now.Hand {
		if d, ok := catalog.DesignByID(id); ok {
			add(d.Cost)
		}
	}
	city := o.Now.Cities[o.Now.Seat]
	board, _ := catalog.BoardByID(city.Board)
	if city.Side != catalog.NoSide && city.Stages < board.Sides[city.Side-1].NumStages {
		add(board.Sides[city.Side-1].Stages[city.Stages].Cost)
	}
	// Public catalog probes, not a look at any actual future deal.
	for _, id := range []catalog.DesignID{38, 48, 57, 71} {
		d, _ := catalog.DesignByID(id)
		add(d.Cost)
	}
	e.before = e.snapshot(o.Now, e.own, counters)
	return e, true
}

// Rank does exactly one pass over the supplied bounded set. It returns only
// completed rankings, sorted by descending value then canonical key. Named
// features are estimates in milli-VP, except Points, which uses engine facts.
func Rank(ctx context.Context, o rules.Observation, offered []rules.Choice, dst []Ranked) ([]Ranked, Counters, error) {
	counts := Counters{}
	if len(offered) == 0 || len(offered) > rules.MaxTurnChoices || !o.Now.Seat.Valid(o.Now.Seats) {
		return dst, counts, ErrInput
	}
	if err := ctx.Err(); err != nil {
		return dst, counts, err
	}
	e, ok := prepare(o, &counts)
	if !ok {
		return dst, counts, ErrInput
	}
	start := len(dst)
	for _, c := range offered {
		if err := ctx.Err(); err != nil {
			return dst[:start], counts, err
		}
		if !c.Valid() {
			return dst[:start], counts, ErrInput
		}
		after, facts, ok := o.Preview(c)
		counts.Previews++
		if !ok {
			return dst[:start], counts, ErrInput
		}
		f := subtract(e.snapshot(after, facts, &counts), e.before)
		f.Points = int64(facts.Score.Total-e.own.Score.Total) * 1000
		f.Passing = e.denial(c)
		if c.Kind == rules.ActStage {
			city := after.Cities[after.Seat]
			board, _ := catalog.BoardByID(city.Board)
			for _, effect := range board.Sides[city.Side-1].Stages[city.Stages-1].Effects {
				if effect.Kind == catalog.EffBuildFromDiscard {
					f.Wonder += min(int64(5500), int64(o.Now.Pile)*250+2000)
				}
			}
		}
		for side, paid := range [2]uint8{c.Left, c.Right} {
			// Same own payment can enrich a different rival. A public leader's
			// liquidity is worth more; no hidden hand or human-targeting bonus.
			premium := int64(160)
			if e.neighbours[side].Score.Total > e.own.Score.Total {
				premium += min(int64(250), int64(e.neighbours[side].Score.Total-e.own.Score.Total)*12)
			}
			f.Externality -= int64(paid) * premium
		}
		dst = append(dst, Ranked{Key: c.Key(), Value: f.Total(), Features: f})
		counts.Ranked++
	}
	for i := start + 1; i < len(dst); i++ {
		for j := i; j > start && (dst[j].Value > dst[j-1].Value || dst[j].Value == dst[j-1].Value && dst[j].Key < dst[j-1].Key); j-- {
			dst[j], dst[j-1] = dst[j-1], dst[j]
		}
	}
	counts.Iterations = 1
	return dst, counts, nil
}
func subtract(a, b Features) Features {
	return Features{Points: a.Points - b.Points, Military: a.Military - b.Military, Science: a.Science - b.Science, Economy: a.Economy - b.Economy, Liquidity: a.Liquidity - b.Liquidity, Chains: a.Chains - b.Chains, Wonder: a.Wonder - b.Wonder, Affordability: a.Affordability - b.Affordability}
}
func (e *evaluation) snapshot(c rules.Current, f rules.PublicFacts, counts *Counters) Features {
	out := Features{}
	remain := remaining(&c)
	me := c.Seat
	city := c.Cities[me]
	if c.Phase != rules.PhaseSides && c.Phase != rules.PhaseOver {
		for i, conflict := range f.Conflict {
			value := int64(conflict) * 1000
			if conflict > 0 {
				margin := int(f.Shields) - int(e.neighbours[i].Shields)
				certainty := min(1000, 650+min(margin, 3)*90+int(c.Turn)*15)
				value = value * int64(certainty) / 1000
				for age := int(c.Age) + 1; age <= 3; age++ {
					value += int64(2*age-1) * 1000 * int64(min(margin, 3)) * 80 / 3000
				}
			}
			out.Military += value
		}
	}
	if remain == 0 {
		// A new sixth-turn Babylon stage still grants the current Age's extra card.
		if c.Phase == rules.PhaseTurn && c.Turn == 6 && f.Abilities&rules.AbilityPlayLastCard != 0 {
			out.Wonder = 4000
		}
		return out
	}
	// Only unearned science growth, beyond the exact current green score.
	science := int(f.ScienceChoices)
	for symbol := catalog.Compass; symbol <= catalog.Tablet; symbol++ {
		science += int(f.Science[symbol])
	}
	out.Science = int64(science) * 1800 * int64(3-max(1, int(c.Age))) / 2
	// Each physical choice is one flexible unit, never all mask resources.
	for r, n := range f.Produce {
		weight := int64(1600)
		if catalog.Resource(r) >= catalog.Glass {
			weight = 2200
		}
		for unit := 0; unit < int(n); unit++ {
			out.Economy += weight / int64(1+unit*2)
		}
	}
	for _, mask := range f.Choices[:f.NumChoices] {
		var usefulness int64
		for r := catalog.Resource(0); r < catalog.NumResources; r++ {
			if mask.Has(r) {
				usefulness += 2400 / int64(1+f.Produce[r])
			}
		}
		out.Economy += usefulness / int64(mask.Count())
	}
	for _, mask := range f.Discount {
		for r := catalog.Resource(0); r < catalog.NumResources; r++ {
			if mask.Has(r) {
				out.Economy += 400 / int64(1+f.Produce[r])
			}
		}
	}
	out.Economy = out.Economy * int64(remain) / 18
	for coin := 0; coin < min(int(city.Coins), 9); coin++ {
		out.Liquidity += int64(500-coin*50) * int64(remain) / 18
	}
	for chain := catalog.Chain(1); int(chain) <= catalog.NumChains; chain++ {
		if f.Chains&chain.Bit() == 0 {
			continue
		}
		id := catalog.T.ChainTarget[chain]
		d, ok := catalog.DesignByID(id)
		if !ok || d.Age < c.Age || catalog.CopiesAt(id, c.Seats) == 0 || !city.Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
			continue
		}
		available := d.Age > c.Age
		if !available {
			for _, h := range c.Hand {
				if h == id {
					available = true
				}
			}
			if e.stats.Returning[id].AtMost > 0 {
				available = true
			}
		}
		if available {
			out.Chains += 900
		}
	}
	out.Wonder = e.wonder(c, f)
	// Hypothetical ample-purse public quotes measure structural affordability,
	// not root legality or same-round funds. The root choices remain authoritative.
	quoted := c
	quoted.Cities[me].Coins = 30000
	for _, cost := range e.probes[:e.numProbes] {
		q := quoted.Quote(me, cost)
		counts.Quotes++
		value := int64(-6000)
		if q.Affordable {
			value = -int64(q.Total) * 700
		}
		out.Affordability += value
	}
	if e.numProbes > 0 {
		out.Affordability = out.Affordability * int64(remain) / int64(18*e.numProbes*3)
	}
	return out
}
func (e *evaluation) wonder(c rules.Current, f rules.PublicFacts) int64 {
	city := c.Cities[c.Seat]
	if city.Side == catalog.NoSide {
		return 0
	}
	board, _ := catalog.BoardByID(city.Board)
	side := board.Sides[city.Side-1]
	remain := remaining(&c)
	age := max(1, int(c.Age))
	var value int64
	if f.Abilities&rules.AbilityPlayLastCard != 0 {
		value += int64(4-age) * 4000
	}
	if f.Abilities&rules.AbilityFreeFirstOfColour != 0 {
		missing := 0
		for color := catalog.Brown; color < catalog.NumColors; color++ {
			if city.Buildings.And(catalog.T.ByColor[color]).Empty() {
				missing++
			}
		}
		value += int64(min(missing, remain)) * 650
	}
	if f.Abilities&rules.AbilityFreeFirstTurn != 0 {
		value += int64(3-age) * 1800
	}
	if f.Abilities&rules.AbilityFreeLastTurn != 0 {
		value += int64(4-age) * 1800
	}
	// Future stages are discounted opportunities, never all booked at full value.
	discounts := [4]int64{550, 250, 100, 40}
	for i := int(city.Stages); i < int(side.NumStages); i++ {
		distance := i - int(city.Stages)
		var estimate int64
		for _, effect := range side.Stages[i].Effects {
			switch effect.Kind {
			case catalog.EffPoints:
				estimate += int64(effect.N) * 1000
			case catalog.EffCoins:
				estimate += int64(effect.N) * 400
			case catalog.EffProduceChoice:
				estimate += 2200 * int64(remain) / 18
			case catalog.EffScienceChoice:
				estimate += 4000
			case catalog.EffShields:
				estimate += 2000
			case catalog.EffPlayLastCard:
				estimate += int64(4-age) * 4000
			case catalog.EffBuildFromDiscard:
				estimate += min(int64(5500), int64(c.Pile)*250+2000)
			case catalog.EffFreeBuild:
				estimate += int64(remain) * 350
			}
		}
		// Only still-unbuilt opportunities enter this estimate.
		value += estimate * discounts[distance] / 1000
	}
	return value
}
func (e *evaluation) denial(c rules.Choice) int64 {
	if c.Kind == rules.ActSide || c.Kind == rules.ActBuildDiscarded {
		return 0
	}
	d, ok := catalog.DesignByID(c.Design)
	if !ok {
		return 0
	}
	receiver := e.o.Now.Seat.Left(e.o.Now.Seats)
	if e.o.Now.Age == 2 {
		receiver = e.o.Now.Seat.Right(e.o.Now.Seats)
	}
	city := e.o.Now.Cities[receiver]
	if !city.Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
		return 0
	}
	// Bounded public-specialist denial. Every action with this carrier withholds
	// the same card. Returning packets at small tables reduce its permanence.
	value := int64(80)
	if d.Color == catalog.Green {
		value += int64(city.Buildings.And(catalog.T.ByColor[catalog.Green]).Count()) * 90
	}
	if d.Color == catalog.Red {
		value += int64(city.Buildings.And(catalog.T.ByColor[catalog.Red]).Count()) * 60
	}
	if e.o.Now.Seats <= 5 && int(e.o.Now.Turn)+int(e.o.Now.Seats) <= 6 {
		value /= 2
	}
	return min(value, 800)
}
