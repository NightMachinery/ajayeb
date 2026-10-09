package ai

import (
	"math"
	"math/bits"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file is the tree search's default policy: how an iteration plays on
// from the node it added, and what the position it reaches is worth.
//
// A rollout plays a determinized state forward with a cheap heuristic,
// epsilon-greedy policy, and stops when the game ends, when the age ends, or
// after a fixed number of plies (16 by default), whichever comes first. The
// position it stops in is scored with the static evaluation (eval.go): the
// reward for seat 0 is sigmoid(Evaluate(s, 0) / 8), a number between 0 and
// 1, and a finished game is worth 1, 0.5 or 0 (a win, a shared victory or a
// loss for seat 0). Stopping at the end of the age keeps the rollout from
// guessing at an age whose cards nobody has seen yet.
//
// The policy, in the play phase:
//
//   - with probability Epsilon it plays at random: a random accessible card,
//     then uniformly one of the things it may do with it (build it if
//     affordable, discard it, or tuck it under an affordable wonder);
//   - otherwise it values every option and plays the best one. Building a
//     card is worth the card's value to the mover (points, coins, shields
//     by how they move the pawn, science by how many different symbols it
//     makes, production while the age is young, a chain while one can
//     follow), less its price in coins; a wonder is worth wonderWorth, less
//     its price. Every option also takes a card away from the opponent, so
//     each gets a share of the taken card's value to the opponent: for a
//     wonder or a discard the policy picks the card the opponent wants
//     most. Discarding is worth its coins.
//
// A move that wins the game at once (the pawn reaching a capital, a sixth
// different science symbol, Law completing six) is always worth more than
// anything else, so rollouts take a supremacy when they can and deny the
// opponent's when that card is on offer.
//
// The play phase is the hot path: the policy prices each accessible card and
// each open wonder once (BuildingQuote, WonderQuote) instead of listing the
// legal actions. The other phases (the draft, the starter choice, the
// pending choices) are rare and short, so the policy lists their legal
// actions and picks the best by a simple value, or at random with
// probability Epsilon (the starter choice is always random).
//
// At the start of each rollout each player is given a mode for the whole
// rollout: with probability ModeProb science (green cards are worth more),
// with the same probability military (red cards are worth more), otherwise
// none. The modes make the rollouts follow the game's three strategies
// instead of one average line.
//
// Everything here reads only the determinized state and the catalog, uses
// only the worker's random generator, and allocates nothing.

// Defaults of the rollouts.
const (
	DefaultRolloutPlies    = 16   // plies before a rollout is cut off
	DefaultRolloutEpsilon  = 0.1  // share of random moves
	DefaultRolloutModeProb = 0.15 // chance of each of the science and military modes, per player and rollout

	// rewardScale is the evaluation difference, in VP, that the reward's
	// sigmoid maps to about 0.73.
	rewardScale = 8
)

// RolloutConfig configures the rollouts of a tree search. The zero value is
// the default policy.
type RolloutConfig struct {
	Plies    int     // plies before a rollout is cut off; 0 means DefaultRolloutPlies
	Epsilon  float64 // share of random moves; 0 means DefaultRolloutEpsilon, a negative value none
	ModeProb float64 // chance of each mode; 0 means DefaultRolloutModeProb, a negative value none
}

// resolved returns the configuration with the defaults filled in and the
// shares clamped to [0, 1] (each mode to [0, 0.5]).
func (c RolloutConfig) resolved() RolloutConfig {
	if c.Plies <= 0 {
		c.Plies = DefaultRolloutPlies
	}
	c.Epsilon = rolloutShare(c.Epsilon, DefaultRolloutEpsilon, 1)
	c.ModeProb = rolloutShare(c.ModeProb, DefaultRolloutModeProb, 0.5)
	return c
}

// rolloutShare resolves a probability setting: 0 means def, a negative
// value 0, and the result is at most top.
func rolloutShare(v, def, top float64) float64 {
	switch {
	case v == 0:
		v = def
	case v < 0 || math.IsNaN(v):
		v = 0
	}
	return min(v, top)
}

// rollMode is a player's plan for one rollout.
type rollMode uint8

const (
	modeNone rollMode = iota
	modeScience
	modeMilitary
)

// rollChance is the resolution of the policy's probabilities: a chance p is
// the threshold p*rollChance on a 16-bit uniform draw.
const rollChance = 1 << 16

// rollWin is the value of a move that wins the game at once: more than any
// other option can be worth.
const rollWin = 1000

// Weights of the policy's option values, in VP. Arrays are indexed by age
// (index 0, the draft, never prices a card).
var (
	rollProd     = [4]float64{1.5, 1.5, 0.8, 0.1} // per unit of fixed raw production
	rollManuf    = 1.3                            // factor on rollProd for glass and papyrus
	rollChoice   = [4]float64{1.5, 1.5, 1.0, 0.2} // a flexible producer
	rollTradeFix = [4]float64{0.8, 0.8, 0.6, 0.1} // per resource fixed at 1 coin
	rollChain    = [4]float64{1.0, 1.0, 0.6, 0}   // a chain symbol (a later card free)
)

const (
	rollGuild      = 4.0  // a guild, whose points depend on the cities
	rollPair       = 3.0  // a second card of a symbol while the board has a token: a token choice
	rollNewSymbol  = 1.0  // a new different science symbol, besides the table's step
	rollBuildBonus = 0.5  // a card in one's city rather than in the discard pile (chains, guilds, colours)
	rollDeny       = 0.4  // share of the taken card's value to the opponent
	rollModeSci    = 3.0  // a green card in science mode
	rollModeMil    = 1.5  // per shield in military mode
	rollWonderBias = -0.5 // wonders are valued generously by wonderWorth
)

// rollTokenValue is the rough worth of taking each progress token, in VP
// (Mathematics also gets 3 per token already owned; Law's symbol is scored
// like a science card).
var rollTokenValue = [catalog.NumTokens + 1]float64{
	catalog.TokenAgriculture:  5,
	catalog.TokenArchitecture: 3,
	catalog.TokenEconomy:      3,
	catalog.TokenLaw:          3,
	catalog.TokenMasonry:      2.5,
	catalog.TokenMathematics:  3,
	catalog.TokenPhilosophy:   7,
	catalog.TokenStrategy:     4,
	catalog.TokenTheology:     3.5,
	catalog.TokenUrbanism:     4,
}

// rolloutPolicy plays rollouts. Each search worker has its own.
type rolloutPolicy struct {
	w     *Weights
	plies int
	eps   uint64 // random-move threshold out of rollChance
	modeP uint64 // each mode's threshold out of rollChance
	mode  [2]rollMode
	legal [engine.MaxLegalActions]engine.Action // scratch for the phases other than play
}

// init sets the policy up from a resolved configuration and the
// evaluation's weights.
func (p *rolloutPolicy) init(c RolloutConfig, w *Weights) {
	p.w = w
	p.plies = c.Plies
	p.eps = uint64(c.Epsilon * rollChance)
	p.modeP = uint64(c.ModeProb * rollChance)
	p.mode = [2]rollMode{}
}

// chance returns true with probability thr/rollChance. It draws nothing for
// a zero threshold.
func (p *rolloutPolicy) chance(r *engine.RNG, thr uint64) bool {
	return thr > 0 && r.Uint64()>>48 < thr
}

// run plays d forward (see the top of this file) and returns the reward of
// the position reached, for seat 0.
func (p *rolloutPolicy) run(d *engine.State, r *engine.RNG) float64 {
	for seat := range p.mode {
		p.mode[seat] = modeNone
		if p.modeP > 0 {
			switch u := r.Uint64() >> 48; {
			case u < p.modeP:
				p.mode[seat] = modeScience
			case u < 2*p.modeP:
				p.mode[seat] = modeMilitary
			}
		}
	}
	age := d.Age
	for n := 0; n < p.plies && !d.Over() && d.Age == age; n++ {
		engine.ApplyTrusted(d, p.move(d, r))
	}
	return p.reward(d)
}

// reward is the value of s for seat 0, from 0 to 1.
func (p *rolloutPolicy) reward(s *engine.State) float64 {
	if s.Over() {
		switch s.Result.Winner {
		case 0:
			return 1
		case 1:
			return 0
		}
		return 0.5
	}
	return 1 / (1 + math.Exp(-p.w.Evaluate(s, 0)/rewardScale))
}

// move returns the policy's action in s, which must not be over.
func (p *rolloutPolicy) move(s *engine.State, r *engine.RNG) engine.Action {
	if s.Phase == engine.PhasePlay {
		return p.playMove(s, r)
	}
	legal := engine.LegalActions(s, p.legal[:0])
	if len(legal) == 1 || s.Phase == engine.PhaseChooseStarter || p.chance(r, p.eps) {
		return legal[r.IntN(len(legal))]
	}
	var x rollCtx
	x.init(s)
	seat := s.ToMove
	best, bestV := 0, math.Inf(-1)
	for i, a := range legal {
		var v float64
		switch a.Type {
		case engine.ActDraftWonder:
			v = wonderWorth(a.Wonder)
		case engine.ActDestroyCard:
			v = p.cardValue(s, &x, seat.Opp(), a.Card) // what the opponent loses
		case engine.ActBuildFromDiscard:
			v = p.cardValue(s, &x, seat, a.Card)
		case engine.ActChooseProgressToken:
			v = p.tokenValue(s, &x, seat, a.Token)
		}
		if v > bestV {
			best, bestV = i, v
		}
	}
	return legal[best]
}

// rollCtx holds what the option values of one position share.
type rollCtx struct {
	age      int    // 1 to 3 (the draft counts as Age I)
	distinct [2]int // different science symbols of each seat
}

func (x *rollCtx) init(s *engine.State) {
	x.age = int(min(max(s.Age, 1), engine.NumAges))
	for seat := range x.distinct {
		x.distinct[seat] = engine.DistinctSymbols(&s.Players[seat])
	}
}

// playMove is the policy's action in the play phase.
func (p *rolloutPolicy) playMove(s *engine.State, r *engine.RNG) engine.Action {
	seat := s.ToMove
	var cards [len(s.Slots)]catalog.CardID
	n := 0
	for m := engine.Accessible(s); m != 0; m &= m - 1 {
		cards[n] = s.Slots[bits.TrailingZeros32(m)].Card
		n++
	}
	if p.chance(r, p.eps) {
		return p.randomPlay(s, r, cards[r.IntN(n)])
	}

	var x rollCtx
	x.init(s)
	coin := p.w.Coin[x.age]
	// The card the opponent wants most: the one to discard or tuck.
	var oppV [len(s.Slots)]float64
	deny := 0
	for i, c := range cards[:n] {
		oppV[i] = p.cardValue(s, &x, seat.Opp(), c)
		if oppV[i] > oppV[deny] {
			deny = i
		}
	}
	denyV := rollDeny * oppV[deny]

	best := engine.DiscardAction(cards[deny])
	bestV := coin*float64(engine.DiscardValue(s, seat)) + denyV
	for i, c := range cards[:n] {
		q := engine.BuildingQuote(s, seat, c)
		if !q.Affordable {
			continue
		}
		v := p.cardValue(s, &x, seat, c) + rollBuildBonus - coin*float64(q.Total) + rollDeny*oppV[i]
		if v > bestV {
			best, bestV = engine.ConstructAction(c), v
		}
	}
	if s.WondersBuilt < engine.MaxWondersBuilt {
		pl := &s.Players[seat]
		for k := range pl.Wonders {
			ws := &pl.Wonders[k]
			if ws.ID == 0 || ws.Built || ws.Removed {
				continue
			}
			q := engine.WonderQuote(s, seat, ws.ID)
			if !q.Affordable {
				continue
			}
			v := p.wonderValue(s, seat, ws.ID) - coin*float64(q.Total) + denyV
			if v > bestV {
				best, bestV = engine.WonderAction(ws.ID, cards[deny]), v
			}
		}
	}
	return best
}

// randomPlay returns one of the legal actions with card c, uniformly.
func (p *rolloutPolicy) randomPlay(s *engine.State, r *engine.RNG, c catalog.CardID) engine.Action {
	seat := s.ToMove
	var opts [2 + len(s.Players[0].Wonders)]engine.Action
	opts[0] = engine.DiscardAction(c)
	k := 1
	if engine.BuildingQuote(s, seat, c).Affordable {
		opts[k] = engine.ConstructAction(c)
		k++
	}
	if s.WondersBuilt < engine.MaxWondersBuilt {
		pl := &s.Players[seat]
		for i := range pl.Wonders {
			ws := &pl.Wonders[i]
			if ws.ID != 0 && !ws.Built && !ws.Removed && engine.WonderQuote(s, seat, ws.ID).Affordable {
				opts[k] = engine.WonderAction(ws.ID, c)
				k++
			}
		}
	}
	return opts[r.IntN(k)]
}

// cardValue is the worth of building card c to seat, in VP, before its
// price: rollWin if it wins the game at once.
func (p *rolloutPolicy) cardValue(s *engine.State, x *rollCtx, seat engine.Seat, c catalog.CardID) float64 {
	t := &catalog.T
	pl := &s.Players[seat]
	v := float64(t.VP[c]) + p.w.Coin[x.age]*float64(t.Coins[c])
	color := t.Color[c]
	if sh := int(t.Shields[c]); sh > 0 {
		if color == catalog.Red && pl.Tokens.Has(catalog.TokenStrategy) {
			sh++
		}
		mv, win := p.pawnGain(s, seat, sh)
		if win {
			return rollWin
		}
		v += mv
		if p.mode[seat] == modeMilitary {
			v += rollModeMil * float64(sh)
		}
	}
	if sym := t.Sym[c]; sym != catalog.NoSymbol {
		switch pl.Sci[sym] {
		case 0:
			d := x.distinct[seat]
			if d+1 >= engine.ScienceSupremacy {
				return rollWin
			}
			v += rollNewSymbol + p.w.Science[d+1] - p.w.Science[d]
		case 1:
			if !s.BoardTokens.Empty() {
				v += rollPair
			}
		}
		if p.mode[seat] == modeScience {
			v += rollModeSci
		}
	}
	prod := &t.Prod[c]
	if raw := int(prod[catalog.Wood]) + int(prod[catalog.Clay]) + int(prod[catalog.Stone]); raw > 0 {
		v += rollProd[x.age] * float64(raw)
	}
	if manuf := int(prod[catalog.Glass]) + int(prod[catalog.Papyrus]); manuf > 0 {
		v += rollProd[x.age] * rollManuf * float64(manuf)
	}
	if t.Choice[c] != 0 {
		v += rollChoice[x.age]
	}
	if f := t.TradeFix[c]; f != 0 {
		v += rollTradeFix[x.age] * float64(bits.OnesCount8(uint8(f)))
	}
	if t.ChainOut[c] != catalog.NoChain {
		v += rollChain[x.age]
	}
	if color == catalog.Purple {
		v += rollGuild
	}
	return v
}

// wonderValue is the worth of building wonder w to seat, before its price:
// rollWin if its shields win the game at once.
func (p *rolloutPolicy) wonderValue(s *engine.State, seat engine.Seat, w catalog.WonderID) float64 {
	if sh := int(catalog.T.WShields[w]); sh > 0 {
		if _, win := p.pawnGain(s, seat, sh); win {
			return rollWin
		}
	}
	return wonderWorth(w) + rollWonderBias
}

// tokenValue is the worth of taking progress token tk to seat: rollWin if
// it wins the game at once (Law, completing six symbols).
func (p *rolloutPolicy) tokenValue(s *engine.State, x *rollCtx, seat engine.Seat, tk catalog.TokenID) float64 {
	pl := &s.Players[seat]
	v := rollTokenValue[min(int(tk), catalog.NumTokens)]
	switch tk {
	case catalog.TokenLaw:
		if pl.Sci[catalog.Scales] == 0 {
			d := x.distinct[seat]
			if d+1 >= engine.ScienceSupremacy {
				return rollWin
			}
			v += p.w.Science[d+1] - p.w.Science[d]
		}
	case catalog.TokenMathematics:
		v += 3 * float64(pl.Tokens.Count())
	}
	if p.mode[seat] == modeScience && tk == catalog.TokenLaw {
		v += rollModeSci
	}
	return v
}

// pawnGain is how much moving the pawn sh spaces toward seat's opponent is
// worth to seat (by the evaluation's military terms), and whether it reaches
// the capital.
func (p *rolloutPolicy) pawnGain(s *engine.State, seat engine.Seat, sh int) (float64, bool) {
	pos := int(s.Military)
	next := pos + sh
	if seat == 1 {
		next = pos - sh
	}
	if next >= engine.MilitaryCapital || next <= -engine.MilitaryCapital {
		return 0, true
	}
	return p.w.military(int8(next), seat) - p.w.military(int8(pos), seat), false
}
