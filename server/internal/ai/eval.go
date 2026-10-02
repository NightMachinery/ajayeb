package ai

import (
	"math/bits"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file is the static evaluation: how good a position is for one seat,
// in victory-point equivalents (VP). It is not rules code, so it uses
// floating point; it is deterministic all the same (no maps, no randomness).
//
// The evaluation reads only public information: the cities, purses, tokens
// and wonders, the face-up cards of the structure, the discard pile, the
// military track, the board tokens, the age and the phase. It never reads
// the face-down cards, the future deals, the removed cards or the box, so it
// gives the same value on an observation's Public state as on any of its
// determinizations, and a search may call it on either.
//
// The features, each scored for both seats and taken as "mine minus the
// opponent's":
//
//   - the score preview without coins and military: blue, green, yellow,
//     purple (guilds counting the cities as they are now), wonders and
//     tokens;
//   - coins, at a value per coin that falls with the age (a coin buys more
//     early on; at the end three coins make one point);
//   - military: the points of the pawn's zone for the player it favours,
//     a little per space it has moved, and a threat bonus when the pawn is
//     one, two or three spaces from a capital;
//   - science: a convex value of the number of different symbols, kept in
//     full only while the missing symbols can still be found, plus a little
//     for each symbol held once whose second card can still be had while
//     the board has tokens to win;
//   - the economy, weighted by how much of the game is left: fixed
//     production (with diminishing returns), the trade price it imposes on
//     the opponent, flexible producers, fixed trade prices, and chain
//     symbols whose free card is still to come;
//   - unbuilt wonders, as a share of what each is worth, while fewer than
//     seven wonders are built;
//   - the ongoing rules of the progress tokens (Strategy, Economy, Masonry,
//     Architecture, Theology, Urbanism).
//
// A finished game is worth Win (or -Win) plus a little per point of the
// final score difference. Pending extra turns are the search's business
// (Weights.Tempo); the static value ignores who is to move.
//
// The weights live in Weights so that tournaments can tune them (slice
// S12). Arrays indexed by age use index 0 for the wonder draft.

// Weights are the evaluation's parameters, all in VP.
type Weights struct {
	Win    float64 // a won game (a lost one is -Win, a shared victory 0)
	Margin float64 // per point of score difference at the end of a game

	Coin [4]float64 // per coin, by age

	MilitaryVP float64    // factor on the military points of the score
	Space      float64    // per space of the pawn from the centre, so that every shield counts
	Threat     [4]float64 // bonus to the leader when the pawn is 1, 2 or 3 spaces from a capital (index = distance)

	Science     [engine.ScienceSupremacy]float64 // by the number of different symbols, 0 to 5
	ScienceDead float64                          // share of Science kept when too few missing symbols can still be found
	OpenPair    float64                          // per symbol held once whose second card can still be had, while the board has tokens

	Production   [4]float64 // per unit of fixed production (first unit; later units count less), by age
	Manufactured float64    // factor on Production for glass and papyrus
	Imposed      [4]float64 // per unit of fixed production raising the opponent's price (up to 3 per resource), by age
	Choice       [4]float64 // per flexible producer, by age
	TradeFix     [4]float64 // per resource bought at 1 coin, by age
	Chain        [4]float64 // per chain symbol owned whose free card can still be had, by age

	WonderPotential [4]float64 // share of an unbuilt wonder's worth, by age
	TokenRule       [4]float64 // factor on the ongoing token rules, by age

	Tempo float64 // a turn more (an extra turn, or an open choice of one's own) at the edge of the search
}

// DefaultWeights returns the evaluation's default parameters. They are
// first guesses, set by hand from the design's feature list; slice S12 tunes
// them with tournaments.
func DefaultWeights() Weights {
	return Weights{
		Win:    100,
		Margin: 0.1,

		Coin: [4]float64{0.5, 0.5, 0.45, 0.34},

		MilitaryVP: 1,
		Space:      0.5,
		Threat:     [4]float64{0, 25, 12, 6},

		Science:     [engine.ScienceSupremacy]float64{0, 0, 1, 3, 6, 12},
		ScienceDead: 0.25,
		OpenPair:    0.75,

		Production:   [4]float64{1.6, 1.6, 1.0, 0.3},
		Manufactured: 1.25,
		Imposed:      [4]float64{0.5, 0.5, 0.35, 0.1},
		Choice:       [4]float64{1.8, 1.8, 1.2, 0.4},
		TradeFix:     [4]float64{1.0, 1.0, 0.7, 0.25},
		Chain:        [4]float64{1.2, 1.2, 0.8, 0.4},

		WonderPotential: [4]float64{0.6, 0.6, 0.5, 0.35},
		TokenRule:       [4]float64{1.0, 1.0, 0.7, 0.35},

		Tempo: 1.5,
	}
}

// defaultWeights is the shared read-only copy used by Evaluate.
var defaultWeights = DefaultWeights()

// Evaluate scores s for seat me with the default weights. See
// Weights.Evaluate.
func Evaluate(s *engine.State, me engine.Seat) float64 { return defaultWeights.Evaluate(s, me) }

// Evaluate scores s for seat me, in VP: positive when me is ahead. It reads
// only public information (see the top of this file) and allocates nothing.
func (w *Weights) Evaluate(s *engine.State, me engine.Seat) float64 {
	if s.Over() {
		return w.terminal(s, me)
	}
	age := int(min(s.Age, engine.NumAges))
	sc := engine.Scores(s)
	var e evalCtx
	e.init(s)
	v := w.side(s, &e, &sc[me], me, age) - w.side(s, &e, &sc[me.Opp()], me.Opp(), age)
	return v + w.military(s.Military, me)
}

// terminal is the value of a finished game for me.
func (w *Weights) terminal(s *engine.State, me engine.Seat) float64 {
	r := &s.Result
	margin := w.Margin * float64(r.Scores[me].Total-r.Scores[me.Opp()].Total)
	switch r.Winner {
	case int8(me):
		return w.Win + margin
	case engine.SharedVictory:
		return margin
	}
	return -w.Win + margin
}

// side is everything one seat's own position is worth, apart from military
// (which is shared between the seats).
func (w *Weights) side(s *engine.State, e *evalCtx, sc *engine.Score, seat engine.Seat, age int) float64 {
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	v := float64(sc.Blue + sc.Green + sc.Yellow + sc.Purple + sc.Wonders + sc.Tokens)
	v += w.Coin[age] * float64(p.Coins)
	v += w.science(s, e, p, o)
	v += w.economy(e, p, o, age)
	v += w.wonders(s, p, age)
	v += w.TokenRule[age] * tokenRules(p)
	return v
}

// military values the pawn for me: the score's military points for the
// leader, its distance from the centre, and the threat bonus near a
// capital.
func (w *Weights) military(pawn int8, me engine.Seat) float64 {
	if pawn == 0 {
		return 0
	}
	d := int(pawn)
	sign := 1.0 // the leader is seat 0
	if d < 0 {
		d, sign = -d, -1
	}
	v := w.MilitaryVP*float64(militaryZone(d)) + w.Space*float64(d)
	if dist := engine.MilitaryCapital - d; dist >= 1 && dist < len(w.Threat) {
		v += w.Threat[dist]
	}
	if me == 1 {
		sign = -sign
	}
	return sign * v
}

// militaryZone returns the military points of the leader at distance d from
// the centre (2, 5 or 10), as on the score sheet.
func militaryZone(d int) int {
	switch {
	case d <= 0:
		return 0
	case d <= 2:
		return 2
	case d <= 5:
		return 5
	}
	return 10
}

// science values p's science symbols.
func (w *Weights) science(s *engine.State, e *evalCtx, p, o *engine.Player) float64 {
	distinct := engine.DistinctSymbols(p)
	if distinct >= engine.ScienceSupremacy {
		return w.Win // only reachable on a state that is not over yet, which the engine never produces
	}
	alive := 0
	pairs := 0
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		have := p.Sci[sym]
		if have == 0 {
			if e.symbolObtainable(s, sym, o) {
				alive++
			}
			continue
		}
		if have == 1 && sym != catalog.Scales && !s.BoardTokens.Empty() && e.symbolObtainable(s, sym, o) {
			pairs++
		}
	}
	v := w.Science[distinct]
	if alive < engine.ScienceSupremacy-distinct {
		v *= w.ScienceDead
	}
	return v + w.OpenPair*float64(pairs)
}

// prodCurve is the worth of n units of one resource's fixed production in
// units of the first one: later units are needed less often.
var prodCurve = [...]float64{0, 1, 1.7, 2.2, 2.5}

// economy values p's production, trade prices and chains.
func (w *Weights) economy(e *evalCtx, p, o *engine.Player, age int) float64 {
	v := 0.0
	for r := catalog.Resource(0); r < catalog.NumResources; r++ {
		n := int(p.Prod[r])
		if n == 0 {
			continue
		}
		pv := w.Production[age] * prodCurve[min(n, len(prodCurve)-1)]
		if r >= catalog.Glass {
			pv *= w.Manufactured
		}
		v += pv
		if !o.TradeFix.Has(r) {
			v += w.Imposed[age] * float64(min(n, 3))
		}
	}
	v += w.Choice[age] * float64(p.NChoice)
	v += w.TradeFix[age] * float64(bits.OnesCount8(uint8(p.TradeFix)))
	for ch := catalog.Chain(1); ch < catalog.NumChains; ch++ {
		if p.Chains&ch.Bit() == 0 {
			continue
		}
		if t := catalog.T.ChainTarget[ch]; t != 0 && !p.City.Has(t) && e.obtainable(t) {
			v += w.Chain[age]
		}
	}
	return v
}

// wonders values p's unbuilt wonders while fewer than seven are built.
func (w *Weights) wonders(s *engine.State, p *engine.Player, age int) float64 {
	if s.WondersBuilt >= engine.MaxWondersBuilt {
		return 0
	}
	v := 0.0
	for k := range p.Wonders {
		if ws := &p.Wonders[k]; ws.ID != 0 && !ws.Built && !ws.Removed {
			v += wonderWorth(ws.ID)
		}
	}
	return w.WonderPotential[age] * v
}

// wonderWorth is a rough VP worth of building wonder id: its points, plus
// its other effects in VP equivalents.
func wonderWorth(id catalog.WonderID) float64 {
	t := &catalog.T
	if int(id) >= len(t.WVP) {
		return 0
	}
	v := float64(t.WVP[id]) + 1.5*float64(t.WShields[id]) + 0.4*float64(t.WCoins[id]) + 0.4*float64(t.WOppLoses[id])
	if t.WPlayAgain[id] {
		v += 3
	}
	if t.WChoice[id] != 0 {
		v += 2.5
	}
	if t.WDestroy[id] != catalog.NoColor {
		v += 2
	}
	if t.WRebuild[id] {
		v += 2.5
	}
	if t.WDraw[id] > 0 {
		v += 2.5
	}
	return v
}

// tokenRules values the ongoing rules of p's progress tokens (their points,
// coins and science symbols are counted elsewhere).
func tokenRules(p *engine.Player) float64 {
	v := 0.0
	for tk, ok := p.Tokens.Next(1); ok; tk, ok = p.Tokens.Next(tk + 1) {
		d := catalog.Token(tk)
		if d == nil {
			continue
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case catalog.EffExtraShield:
				v += 2
			case catalog.EffTradeIncome:
				v += 1.5
			case catalog.EffDiscount:
				v += 1.5
			case catalog.EffWondersPlayAgain:
				v += 0.75 * float64(unbuiltWonders(p))
			case catalog.EffChainBonus:
				v += 1
			}
		}
	}
	return v
}

// unbuiltWonders counts p's wonders still to build.
func unbuiltWonders(p *engine.Player) int {
	n := 0
	for k := range p.Wonders {
		if ws := &p.Wonders[k]; ws.ID != 0 && !ws.Built && !ws.Removed {
			n++
		}
	}
	return n
}

// evalCtx holds what the evaluation derives once per position: which cards
// are visible where, so that "can this card still be had" is a set lookup.
type evalCtx struct {
	faceUp  catalog.CardSet // the face-up cards of the structure
	visible catalog.CardSet // cards in a visible place: face-up slots, cities, under wonders, the discard pile, out of play
	age     uint8
}

// init fills the context from the public parts of s.
func (e *evalCtx) init(s *engine.State) {
	e.age = s.Age
	for i := range s.Slots {
		if sl := &s.Slots[i]; sl.Card != 0 && sl.FaceUp {
			e.faceUp.Add(sl.Card)
		}
	}
	e.visible = e.faceUp.Or(s.Discard).Or(s.OutOfPlay)
	for seat := range s.Players {
		p := &s.Players[seat]
		e.visible = e.visible.Or(p.City)
		for _, ws := range p.Wonders {
			if ws.Tucked != 0 {
				e.visible.Add(ws.Tucked)
			}
		}
	}
}

// obtainable reports whether card c may still be built by someone: it is
// face up in the structure, or it is in no visible place and its age is not
// over (so it is face down, still to be dealt, or, unknowably, removed).
func (e *evalCtx) obtainable(c catalog.CardID) bool {
	if e.faceUp.Has(c) {
		return true
	}
	return !e.visible.Has(c) && catalog.T.Age[c] >= e.age
}

// symbolObtainable reports whether a player whose opponent is o can still
// gain science symbol sym: a green card showing it can still be had, or, for
// the scales, Law is still on the board.
func (e *evalCtx) symbolObtainable(s *engine.State, sym catalog.Symbol, o *engine.Player) bool {
	if sym == catalog.Scales {
		return s.BoardTokens.Has(catalog.TokenLaw) && !o.Tokens.Has(catalog.TokenLaw)
	}
	set := symbolCards[sym]
	for c, ok := set.Next(1); ok; c, ok = set.Next(c + 1) {
		if e.obtainable(c) {
			return true
		}
	}
	return false
}

// symbolCards lists the green cards showing each science symbol.
var symbolCards = buildSymbolCards()

func buildSymbolCards() [catalog.NumSymbols]catalog.CardSet {
	var out [catalog.NumSymbols]catalog.CardSet
	for c, ok := catalog.T.All.Next(1); ok; c, ok = catalog.T.All.Next(c + 1) {
		if sym := catalog.T.Sym[c]; sym != catalog.NoSymbol && sym < catalog.NumSymbols {
			out[sym].Add(c)
		}
	}
	return out
}
