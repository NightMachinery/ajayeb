package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// PublicFacts are exact derived facts from public cities. They have no hidden
// holder, seed, record or reference to a game. Policies use these facts rather
// than reimplementing the engine's score, science assignment or production.
type PublicFacts struct {
	Score          Score
	Produce        [catalog.NumResources]uint8
	Choices        [MaxChoice]catalog.ResMask
	NumChoices     uint8
	Discount       [2]catalog.ResMask
	Science        [catalog.NumSymbols]uint8
	ScienceChoices uint8
	Shields        uint8
	Abilities      uint8
	Chains         uint32
	Conflict       [2]int16
}

func factsFromPublic(st *State, seat Seat) PublicFacts {
	c := &st.Cities[seat]
	out := PublicFacts{Score: scoreCity(st, seat), Produce: c.Produce, Choices: c.Choice, NumChoices: c.NChoice, Discount: c.Discount, Science: c.Science, ScienceChoices: c.ScienceChoice, Shields: c.Shields, Abilities: c.Abilities, Chains: c.Chains}
	if st.Age >= 1 && st.Age <= NumAges {
		for i, other := range [2]Seat{seat.Left(st.Seats), seat.Right(st.Seats)} {
			switch {
			case c.Shields > st.Cities[other].Shields:
				out.Conflict[i] = [NumAges]int16{1, 3, 5}[st.Age-1]
			case c.Shields < st.Cities[other].Shields:
				out.Conflict[i] = -1
			}
		}
	}
	return out
}
func (c *Current) Facts(seat Seat) (PublicFacts, bool) {
	if c == nil || !seat.Valid(c.Seats) {
		return PublicFacts{}, false
	}
	st, ok := c.publicState()
	if !ok {
		return PublicFacts{}, false
	}
	return factsFromPublic(&st, seat), true
}

// Preview projects one supplied, engine-offered choice into public city facts.
// This is an isolated placement, NOT a legal-move generator or sampled world:
// other seats' unknown simultaneous placements and trade receipts are absent.
// It applies the offered effective payment, own placement/income and public
// trade credits using the engine caches/effect/scoring functions. It does not
// advance a point, resolve military, open abilities or alter hand/knowledge.
// The returned Current is only for hypothetical public-fact quotes. Actual
// legality continues to be checked by membership in the real Game's offers.
func (o *Observation) Preview(choice Choice) (Current, PublicFacts, bool) {
	if o == nil || !o.Now.Seat.Valid(o.Now.Seats) || !choice.Valid() {
		return Current{}, PublicFacts{}, false
	}
	st, ok := o.Now.publicState()
	if !ok {
		return Current{}, PublicFacts{}, false
	}
	seat := o.Now.Seat
	c := &st.Cities[seat]
	if int(choice.Total) != int(choice.Printed)+int(choice.Left)+int(choice.Right) || int(choice.Total) > int(c.Coins) {
		return Current{}, PublicFacts{}, false
	}
	var effects []catalog.Effect
	switch choice.Kind {
	case ActSide:
		if c.Stages != 0 {
			return Current{}, PublicFacts{}, false
		}
		c.Side = choice.Side
	case ActBuild, ActBuildDiscarded:
		d, exists := catalog.DesignByID(choice.Design)
		if !exists || !c.Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
			return Current{}, PublicFacts{}, false
		}
		c.Buildings.Add(choice.Design)
		effects = d.Effects
	case ActStage:
		b, _ := catalog.BoardByID(c.Board)
		if c.Side == catalog.NoSide || c.Stages >= b.Sides[c.Side-1].NumStages {
			return Current{}, PublicFacts{}, false
		}
		effects = b.Sides[c.Side-1].Stages[c.Stages].Effects
		c.Stages++
	case ActSell:
		if c.Coins > 32764 {
			return Current{}, PublicFacts{}, false
		}
		c.Coins += 3
	default:
		return Current{}, PublicFacts{}, false
	}
	c.Coins -= int16(choice.Total)
	for i, other := range [2]Seat{seat.Left(st.Seats), seat.Right(st.Seats)} {
		amount := [2]uint8{choice.Left, choice.Right}[i]
		if int(st.Cities[other].Coins)+int(amount) > 32767 {
			return Current{}, PublicFacts{}, false
		}
		st.Cities[other].Coins += int16(amount)
	}
	var err error
	*c, err = cityCaches(*c)
	if err != nil {
		return Current{}, PublicFacts{}, false
	}
	if _, err = applyWhenPlaced(&st, seat, effects); err != nil {
		return Current{}, PublicFacts{}, false
	}
	current := o.Now
	for s := Seat(0); s < Seat(st.Seats); s++ {
		p := &current.Cities[s]
		next := &st.Cities[s]
		p.Side, p.Stages, p.Buildings, p.Coins = next.Side, next.Stages, next.Buildings, next.Coins
	}
	return current, factsFromPublic(&st, seat), true
}
