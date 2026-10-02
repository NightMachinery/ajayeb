package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file is the only writer of the derived caches in Player (Chains,
// Prod, Choice, NChoice, TradeFix, ColorN, Sci). Every change to a city, a
// player's tokens or a player's built wonders goes through one of the
// functions below, so the caches always match a recompute from scratch;
// CheckInvariants verifies that after every step of the property tests.
//
// The flexible producers in Choice are kept in a canonical order, the cards
// of the city by ascending id and then the built wonders in slot order, the
// same order that cost.go names them in. Rebuilding the list whenever a
// producer comes or goes keeps that order without any bookkeeping.

// addCard puts building c into seat's city and updates the caches. The card
// must not be in the city already.
func addCard(s *State, seat Seat, c catalog.CardID) {
	p := &s.Players[seat]
	p.City.Add(c)
	t := &catalog.T
	p.ColorN[t.Color[c]]++
	p.Chains |= t.ChainOut[c].Bit()
	for r := range catalog.NumResources {
		p.Prod[r] += t.Prod[c][r]
	}
	p.TradeFix |= t.TradeFix[c]
	if sym := t.Sym[c]; sym != catalog.NoSymbol {
		p.Sci[sym]++
	}
	if t.Choice[c] != 0 {
		rebuildChoice(p)
	}
}

// removeCard takes building c out of seat's city (the Statue of Zeus and
// Circus Maximus) and recomputes the caches. Removal is rare, so a full
// recompute is simpler than undoing each cache by hand.
func removeCard(s *State, seat Seat, c catalog.CardID) {
	p := &s.Players[seat]
	p.City.Remove(c)
	recomputeCaches(p)
}

// addToken gives seat the progress token tk. A token with a science symbol
// (Law) adds that symbol to the science counts. Taking the token from the
// board or from the Great Library offer is the caller's job.
func addToken(s *State, seat Seat, tk catalog.TokenID) {
	p := &s.Players[seat]
	p.Tokens.Add(tk)
	if sym := tokenSymbol(tk); sym != catalog.NoSymbol {
		p.Sci[sym]++
	}
}

// buildWonder marks seat's wonder in slot i as built with the card tucked
// under it, counts it in WondersBuilt, and starts its flexible production.
// It applies no effect and not the seven-wonder rule: resolveWonderEffects
// does that.
func buildWonder(s *State, seat Seat, i int, tucked catalog.CardID) {
	p := &s.Players[seat]
	ws := &p.Wonders[i]
	ws.Built = true
	ws.Tucked = tucked
	s.WondersBuilt++
	if catalog.T.WChoice[ws.ID] != 0 {
		rebuildChoice(p)
	}
}

// MaxWondersBuilt is the number of wonders that can be built in one game.
// When the seventh is built, the last unbuilt wonder goes back to the box.
const MaxWondersBuilt = 7

// sevenWonderRule applies the seven-wonder rule once seven wonders have been
// built: every wonder still neither built nor removed is marked Removed, with
// an EvWonderRemoved event. It does nothing before the seventh wonder and is
// idempotent, so calling it after every wonder is safe. The rule belongs to
// resolveWonderEffects (wonders.go); this is the bookkeeping it calls.
func sevenWonderRule(s *State, ev *[]Event) {
	if s.WondersBuilt < MaxWondersBuilt {
		return
	}
	for seat := range Seat(2) {
		for i := range s.Players[seat].Wonders {
			ws := &s.Players[seat].Wonders[i]
			if ws.ID != 0 && !ws.Built && !ws.Removed {
				ws.Removed = true
				emit(ev, Event{Kind: EvWonderRemoved, Seat: seat, Wonder: ws.ID})
			}
		}
	}
}

// tokenSymbol returns the science symbol a progress token carries (Scales
// for Law), or NoSymbol.
func tokenSymbol(tk catalog.TokenID) catalog.Symbol {
	d := catalog.Token(tk)
	if d == nil {
		return catalog.NoSymbol
	}
	if e, ok := catalog.FindEffect(d.Effects, catalog.EffScience); ok {
		return e.Symbol
	}
	return catalog.NoSymbol
}

// rebuildChoice recomputes p's flexible producers in the canonical order.
func rebuildChoice(p *Player) {
	p.Choice = [MaxChoice]catalog.ResMask{}
	p.NChoice = 0
	add := func(m catalog.ResMask) {
		if p.NChoice < MaxChoice {
			p.Choice[p.NChoice] = m
			p.NChoice++
		}
	}
	for c, ok := p.City.Next(1); ok; c, ok = p.City.Next(c + 1) {
		if int(c) < len(catalog.T.Choice) && catalog.T.Choice[c] != 0 {
			add(catalog.T.Choice[c])
		}
	}
	for i := range p.Wonders {
		ws := &p.Wonders[i]
		if ws.Built && int(ws.ID) < len(catalog.T.WChoice) && catalog.T.WChoice[ws.ID] != 0 {
			add(catalog.T.WChoice[ws.ID])
		}
	}
}

// recomputeCaches rebuilds every cache of p from its city, tokens and built
// wonders.
func recomputeCaches(p *Player) {
	p.Chains = 0
	p.Prod = [catalog.NumResources]uint8{}
	p.TradeFix = 0
	p.ColorN = [catalog.NumColors]uint8{}
	p.Sci = [catalog.NumSymbols]uint8{}
	t := &catalog.T
	for c, ok := p.City.Next(1); ok; c, ok = p.City.Next(c + 1) {
		if int(c) >= len(t.Color) {
			continue
		}
		p.ColorN[t.Color[c]]++
		p.Chains |= t.ChainOut[c].Bit()
		for r := range catalog.NumResources {
			p.Prod[r] += t.Prod[c][r]
		}
		p.TradeFix |= t.TradeFix[c]
		if sym := t.Sym[c]; sym != catalog.NoSymbol {
			p.Sci[sym]++
		}
	}
	for tk, ok := p.Tokens.Next(1); ok; tk, ok = p.Tokens.Next(tk + 1) {
		if sym := tokenSymbol(tk); sym != catalog.NoSymbol {
			p.Sci[sym]++
		}
	}
	rebuildChoice(p)
}

// gainCoins adds n (at least 0) coins to seat, saturating at the int16 limit.
func gainCoins(s *State, seat Seat, n int16) {
	p := &s.Players[seat]
	p.Coins = sat16(int(p.Coins) + int(n))
}
