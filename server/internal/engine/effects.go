package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file applies the effects of buildings and progress tokens. The
// military track is in military.go, science in science.go, and scoring and
// the end of the game in score.go. The hook contracts are in hooks.go.
//
// # The order of a building's effects
//
// The turn machine (apply.go) takes the card, puts it into the city
// (addCard), and pays for it: the printed coins go to the bank, and the
// trading part goes to an opponent who owns Economy. Then
// resolveBuildingEffects applies, in this order:
//
//  1. Urbanism's chain bonus, when the building was free through a chain;
//  2. the card's own coins (fixed coins, coins per card of some colours,
//     coins per wonder), counted with the new card already in the city;
//  3. its shields, plus Strategy's extra shield for a red building, which
//     move the pawn (looting, military supremacy);
//  4. its science symbol: six different symbols win at once, and a new pair
//     opens the board token choice when the board has a token.
//
// The order matters when coins run short: an opponent who owns Economy is
// paid the trading coins before any looting takes coins from them.
//
// # Functions for the wonder and pending-choice code (slice S5b)
//
// The wonder effects and the pending choices reuse these instead of
// repeating the rules:
//
//   - awardCoins(s, seat, n, src, ev): seat takes n coins from the bank
//     (EvCoins). For a wonder's coins use wonderCoins(s, seat, w, ev).
//   - loseCoins(s, seat, n, src, ev) int16: seat pays up to n coins to the
//     bank, never going below zero (EvCoins when anything was lost); it
//     returns the coins actually lost. The Appian Way's loss uses it, with
//     fromWonder(w) as the source.
//   - movePawn(s, seat, shields, ev) outcome: shields push the pawn toward
//     seat's opponent, taking looting tokens, and a capital ends the game.
//     For a wonder's shields use wonderShields(s, seat, w, ev), which never
//     adds Strategy's extra shield.
//   - resolveBuildingEffects(s, seat, card, viaDiscard, ev) outcome: every
//     effect of a building the Mausoleum rebuilt (call it after addCard):
//     its coins, its shields with Strategy, and its science symbol, which can
//     win the game or open the board token choice. viaDiscard pays no
//     Urbanism bonus.
//   - resolveTokenEffects(s, seat, token, ev) outcome: the immediate effects
//     of a token just added with addToken (Agriculture's and Urbanism's
//     coins, then a science supremacy that Law's scales complete), for both
//     the board pick and the Great Library pick.
//   - takeBoardToken(s, seat, token, ev) outcome (science.go): the whole
//     board token choice of a pair (the token leaves the board, EvToken,
//     then resolveTokenEffects). resolvePending's PhasePickBoardToken case
//     calls it.
//   - wondersPlayAgain(p) bool: p owns Theology, so every wonder p builds
//     grants another turn (never two).
//   - checkSupremacy(s, ev) bool (score.go) and scienceSupremacy(s, seat,
//     ev) bool (science.go) end the game on a supremacy; winNow(s, kind,
//     winner, ev) (score.go) ends it with a given winner.
//
// Every function that can end the game leaves s.Phase at PhaseGameOver with
// s.Result set (its Scores are Scores(s)) and EvGameOver emitted, and the
// callers stop when they see outcomeOver or s.Over(). A caller that has a
// choice open (Destroy, Library) must clear that choice's data before it
// calls one of these, so that a game ending there leaves no stale choice.

// coinSource names what gave or took coins, for EvCoins: at most one of the
// fields is set.
type coinSource struct {
	card   catalog.CardID
	wonder catalog.WonderID
	token  catalog.TokenID
}

func fromCard(c catalog.CardID) coinSource     { return coinSource{card: c} }
func fromWonder(w catalog.WonderID) coinSource { return coinSource{wonder: w} }
func fromToken(t catalog.TokenID) coinSource   { return coinSource{token: t} }

// awardCoins gives seat n coins from the bank and reports it with EvCoins
// (A = the gain, B = the new purse). Nothing happens when n is 0 or less.
func awardCoins(s *State, seat Seat, n int, src coinSource, ev *[]Event) {
	if n <= 0 {
		return
	}
	p := &s.Players[seat]
	before := p.Coins
	p.Coins = sat16(int(p.Coins) + n)
	emit(ev, Event{Kind: EvCoins, Seat: seat, Card: src.card, Wonder: src.wonder, Token: src.token,
		A: p.Coins - before, B: p.Coins})
}

// takeCoins removes up to n coins from seat, never going below zero, and
// returns how many were removed. It reports nothing.
func takeCoins(s *State, seat Seat, n int) int16 {
	p := &s.Players[seat]
	lost := int16(min(max(n, 0), int(p.Coins)))
	p.Coins -= lost
	return lost
}

// loseCoins makes seat pay up to n coins to the bank, never going below zero,
// and returns how many it paid. A loss is reported with EvCoins (A = minus
// the loss, B = the new purse); losing nothing reports nothing.
func loseCoins(s *State, seat Seat, n int, src coinSource, ev *[]Event) int16 {
	lost := takeCoins(s, seat, n)
	if lost > 0 {
		emit(ev, Event{Kind: EvCoins, Seat: seat, Card: src.card, Wonder: src.wonder, Token: src.token,
			A: -lost, B: s.Players[seat].Coins})
	}
	return lost
}

// tokenRule looks up a lasting rule among the tokens in ts: it returns the
// sum of the N of every effect of the given kind, and the first token (by
// id) that carries one, or 0 when none does. Unregistered tokens count for
// nothing.
func tokenRule(ts catalog.TokenSet, kind catalog.EffectKind) (n int, by catalog.TokenID) {
	for id, ok := ts.Next(1); ok; id, ok = ts.Next(id + 1) {
		d := catalog.Token(id)
		if d == nil {
			continue
		}
		for _, e := range d.Effects {
			if e.Kind == kind {
				n += int(e.N)
				if by == 0 {
					by = id
				}
			}
		}
	}
	return n, by
}

// wondersPlayAgain reports whether p owns a token that makes every wonder p
// builds grant another turn (Theology). A wonder that grants one already
// still grants only one; that is the wonder code's business.
func wondersPlayAgain(p *Player) bool {
	_, by := tokenRule(p.Tokens, catalog.EffWondersPlayAgain)
	return by != 0
}

// buildingShields returns the shields building c gives p: its printed
// shields, plus the extra shields of p's tokens (Strategy) when c is a red
// building. Strategy reaches only buildings constructed after it was taken
// because a building's shields apply once, when it is built.
func buildingShields(p *Player, c catalog.CardID) int {
	if int(c) >= len(catalog.T.Shields) {
		return 0
	}
	n := int(catalog.T.Shields[c])
	if catalog.T.Color[c] == catalog.Red {
		extra, _ := tokenRule(p.Tokens, catalog.EffExtraShield)
		n += extra
	}
	return n
}

// builtWonders counts p's built wonders.
func builtWonders(p *Player) int {
	n := 0
	for i := range p.Wonders {
		if p.Wonders[i].Built {
			n++
		}
	}
	return n
}

// colorCount counts p's cards of the colours in m.
func colorCount(p *Player, m catalog.ColorMask) int {
	n := 0
	for c := catalog.Brown; c < catalog.NumColors; c++ {
		if m.Has(c) {
			n += int(p.ColorN[c])
		}
	}
	return n
}

// scoped picks the count that an effect of scope sc uses: the owner's own,
// or the larger of the two cities' (each effect decides on its own, at the
// moment it applies).
func scoped(sc catalog.Scope, own, opp int) int {
	if sc == catalog.ScopeMaxCity {
		return max(own, opp)
	}
	return own
}

// buildingCoins returns the coins building c pays seat when it is built: its
// fixed coins, its coins per card and its coins per wonder, counted now. The
// caller has already put c into the city, so a card whose own colour is
// counted counts itself (the Lighthouse). The guilds compare the two cities
// as they are at this moment.
func buildingCoins(s *State, seat Seat, c catalog.CardID) int {
	d := catalog.Card(c)
	if d == nil {
		return 0
	}
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	n := 0
	for _, e := range d.Effects {
		switch e.Kind {
		case catalog.EffCoins:
			n += int(e.N)
		case catalog.EffCoinsPerCard:
			n += int(e.N) * scoped(e.Scope, colorCount(p, e.Colors), colorCount(o, e.Colors))
		case catalog.EffCoinsPerWonder:
			n += int(e.N) * scoped(e.Scope, builtWonders(p), builtWonders(o))
		}
	}
	return n
}

// resolveBuildingEffects applies the effects of building c, which seat has
// just put into its city (bought, free through a chain, or rebuilt from the
// discard pile), in the order given at the top of this file. It returns
// outcomeOver when the building won the game, outcomeChoice when it opened
// the board token choice, and outcomeFinish otherwise.
func resolveBuildingEffects(s *State, seat Seat, c catalog.CardID, via buildVia, ev *[]Event) outcome {
	p := &s.Players[seat]
	if via == viaChain {
		if bonus, by := tokenRule(p.Tokens, catalog.EffChainBonus); by != 0 {
			awardCoins(s, seat, bonus, fromToken(by), ev)
		}
	}
	awardCoins(s, seat, buildingCoins(s, seat, c), fromCard(c), ev)
	if n := buildingShields(p, c); n > 0 {
		if movePawn(s, seat, uint8(min(n, 255)), ev) == outcomeOver {
			return outcomeOver
		}
	}
	if int(c) < len(catalog.T.Sym) {
		if sym := catalog.T.Sym[c]; sym != catalog.NoSymbol {
			return gainSymbol(s, seat, sym, c, ev)
		}
	}
	return outcomeFinish
}

// resolveTokenEffects applies the immediate effects of token t, which seat
// has just taken (addToken has added it, and Law's scales to the science
// counts): its coins from the bank (Agriculture, Urbanism), then the
// science supremacy check that Law's scales can complete. The lasting rules
// of the other tokens need nothing here: the payment, the shields and the
// wonder code read them from the seat's tokens whenever they apply.
func resolveTokenEffects(s *State, seat Seat, t catalog.TokenID, ev *[]Event) outcome {
	if int(t) < len(catalog.T.TokenCoins) {
		awardCoins(s, seat, int(catalog.T.TokenCoins[t]), fromToken(t), ev)
	}
	if tokenSymbol(t) != catalog.NoSymbol && scienceSupremacy(s, seat, ev) {
		return outcomeOver
	}
	return outcomeFinish
}

// wonderCoins gives seat the coins of wonder w (the Temple of Artemis, the
// Hanging Gardens, the Appian Way).
func wonderCoins(s *State, seat Seat, w catalog.WonderID, ev *[]Event) {
	if int(w) < len(catalog.T.WCoins) {
		awardCoins(s, seat, int(catalog.T.WCoins[w]), fromWonder(w), ev)
	}
}

// wonderShields moves the pawn by the shields of wonder w (the Colossus, the
// Statue of Zeus, Circus Maximus). Strategy never adds to a wonder's shields.
func wonderShields(s *State, seat Seat, w catalog.WonderID, ev *[]Event) outcome {
	if int(w) >= len(catalog.T.WShields) {
		return outcomeFinish
	}
	return movePawn(s, seat, catalog.T.WShields[w], ev)
}
