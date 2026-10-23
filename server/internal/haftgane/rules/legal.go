package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// FreeReason is per construction, never part of the ordinary cost cache.
type FreeReason uint8

const (
	FreeNone FreeReason = iota
	FreeChain
	FreeFirstColour
	FreeFirstTurn
	FreeLastTurn
)

// Choice embeds the engine-owned action. Key() is its canonical serialization;
// the core can reuse a caller's buffer without allocating key strings. All
// amounts are effective amounts, including Printed=0 on a free construction.
type Choice struct {
	Action
	Stage   uint8
	Printed uint8
	Total   uint16
	Free    FreeReason
	Payment Payment
}

// CostAssessment retains the facts needed for distinct disabled reasons.
// Cheapest includes printed coins and is meaningful exactly when Coverable.
// QuoteState keeps its existing cheapest-AFFORDABLE bot API; this assessment
// deliberately does not discard information when the actual purse is short.
type CostAssessment struct {
	Coverable  bool
	Cheapest   uint16
	Affordable bool
}

func AssessCostState(st *State, seat Seat, cost catalog.Cost) CostAssessment {
	q := ordinaryCost(st, seat, cost)
	return q.status(st, seat)
}

type ordinaryQuote struct {
	cost     catalog.Cost
	payments Payments // complete undominated frontier, before the purse filter
}

func ordinaryCost(st *State, seat Seat, cost catalog.Cost) ordinaryQuote {
	q := ordinaryQuote{cost: cost}
	in, ok := tradeFromState(st, seat, cost)
	if !ok || st.Cities[seat].Coins < 0 {
		return q
	}
	// At most 255 printed coins plus fourteen trade coins. An unbounded purse
	// finds coverability/cheapest and witnesses in one solver call per cost.
	in.purse = 32767
	q.payments = solvePayment(in)
	return q
}
func (q *ordinaryQuote) status(st *State, seat Seat) CostAssessment {
	if q.payments.Count == 0 || st == nil || !seat.Valid(st.Seats) {
		return CostAssessment{}
	}
	p := q.payments.Outcomes[0]
	cheapest := uint16(q.payments.Printed) + uint16(p.Left) + uint16(p.Right)
	return CostAssessment{Coverable: true, Cheapest: cheapest, Affordable: st.Cities[seat].Coins >= int16(cheapest)}
}

type quoteCache struct {
	entries [HandSize + 1]ordinaryQuote
	used    uint8
}

func (c *quoteCache) get(st *State, seat Seat, cost catalog.Cost) *ordinaryQuote {
	for i := uint8(0); i < c.used; i++ {
		if c.entries[i].cost == cost {
			return &c.entries[i]
		}
	}
	// Legal has at most seven different card costs and one stage cost.
	q := &c.entries[c.used]
	*q = ordinaryCost(st, seat, cost)
	c.used++
	return q
}

type construction struct {
	cost  catalog.Cost
	stage uint8
	free  FreeReason
}

// constructionFor is shared by offering and Pay. The name check precedes the
// free check, which precedes any lookup in the ordinary quote cache.
func constructionFor(st *State, seat Seat, kind ActionKind, id catalog.DesignID) (construction, bool) {
	d, ok := catalog.DesignByID(id)
	if !ok {
		return construction{}, false
	}
	c := &st.Cities[seat]
	switch kind {
	case ActBuild:
		if !c.Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
			return construction{}, false
		}
		return construction{cost: d.Cost, free: freeBuilding(st, c, d)}, true
	case ActStage:
		b, found := catalog.BoardByID(c.Board)
		if !found || c.Side < catalog.Day || c.Side > catalog.Night || c.Stages >= b.Sides[c.Side-1].NumStages {
			return construction{}, false
		}
		return construction{cost: b.Sides[c.Side-1].Stages[c.Stages].Cost, stage: c.Stages + 1}, true
	}
	return construction{}, false
}
func freeBuilding(st *State, c *City, d catalog.Design) FreeReason {
	switch {
	case d.ChainIn != 0 && c.Chains&d.ChainIn.Bit() != 0:
		return FreeChain
	case c.Abilities&AbilityFreeFirstOfColour != 0 && c.Colors[d.Color] == 0:
		return FreeFirstColour
	case st.Phase == PhaseTurn && st.Turn == 1 && c.Abilities&AbilityFreeFirstTurn != 0:
		return FreeFirstTurn
	case st.Phase == PhaseTurn && st.Turn == TurnsPerAge && c.Abilities&AbilityFreeLastTurn != 0:
		return FreeLastTurn
	default:
		return FreeNone
	}
}

// LegalState reads only State and reuses buf from its beginning. Choices follow
// design id, then build/stage/sell, then payment total and left amount. Equal
// physical copies produce one set of actions. Seed and Record are never read.
func LegalState(st *State, seat Seat, buf []Choice) []Choice {
	out := buf[:0]
	if st == nil || !seat.Valid(st.Seats) {
		return out
	}
	if st.Phase == PhaseSides {
		if actions, ok := sideActions(st, seat); ok {
			for _, a := range actions {
				out = append(out, Choice{Action: a})
			}
		}
		return out
	}
	if st.Phase != PhaseTurn || st.Age < 1 || st.Age > NumAges || st.Turn < 1 || st.Turn > TurnsPerAge {
		return out
	}
	var hand catalog.DesignSet
	for _, id := range st.Hands[seat] {
		if id.Valid() {
			hand.Add(id)
		}
	}
	var cache quoteCache
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if !hand.Has(id) {
			continue
		}
		for _, kind := range [2]ActionKind{ActBuild, ActStage} {
			terms, ok := constructionFor(st, seat, kind, id)
			if !ok {
				continue
			}
			a := Action{Kind: kind, Design: id}
			if terms.free != FreeNone {
				out = append(out, Choice{Action: a, Free: terms.free})
				continue
			}
			q := cache.get(st, seat, terms.cost)
			for i := uint8(0); i < q.payments.Count; i++ {
				p := q.payments.Outcomes[i]
				total := uint16(q.payments.Printed) + uint16(p.Left) + uint16(p.Right)
				if int16(total) > st.Cities[seat].Coins {
					continue
				}
				a.Left, a.Right = p.Left, p.Right
				out = append(out, Choice{Action: a, Stage: terms.stage, Printed: q.payments.Printed, Total: total, Payment: p})
			}
		}
		out = append(out, Choice{Action: Action{Kind: ActSell, Design: id}})
	}
	return out
}
func (g *Game) Legal(seat Seat) []Choice {
	if g == nil {
		return []Choice{}
	}
	return LegalState(&g.State, seat, make([]Choice, 0, 32))
}
