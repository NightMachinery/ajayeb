package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file is science: the symbols of the green cards and of Law, pairs and
// the board token choice they open, and scientific supremacy.
//
// Player.Sci counts each symbol a player owns (city.go keeps it: a green card
// adds its symbol, Law adds the scales). Each of the six card symbols is on
// exactly two green cards, so a count reaching 2 is the one pair of that
// symbol in the game: it opens PhasePickBoardToken when the board still has
// a token, and gives nothing otherwise. The scales are only on Law, so they
// never pair. Six different symbols out of the seven win the game at once.

// ScienceSupremacy is the number of different science symbols that wins the
// game at once.
const ScienceSupremacy = 6

// DistinctSymbols counts the different science symbols p owns, the scales of
// Law included.
func DistinctSymbols(p *Player) int {
	n := 0
	for sym := catalog.Globe; sym < catalog.NumSymbols; sym++ {
		if p.Sci[sym] > 0 {
			n++
		}
	}
	return n
}

// scienceSupremacy ends the game with seat's science victory if seat owns
// ScienceSupremacy different symbols, and reports whether it did.
func scienceSupremacy(s *State, seat Seat, ev *[]Event) bool {
	if DistinctSymbols(&s.Players[seat]) < ScienceSupremacy {
		return false
	}
	winNow(s, VictoryScience, seat, ev)
	return true
}

// gainSymbol finishes seat's gain of symbol sym from building c (already
// counted in the seat's Sci by addCard): a sixth different symbol wins the
// game (outcomeOver); otherwise a newly completed pair reports EvPair (A =
// the symbol) and opens the board token choice (outcomeChoice), or, with no
// token left on the board, reports EvEffectSkipped (Card = c, A = the phase
// it would have opened) and gives nothing.
func gainSymbol(s *State, seat Seat, sym catalog.Symbol, c catalog.CardID, ev *[]Event) outcome {
	if scienceSupremacy(s, seat, ev) {
		return outcomeOver
	}
	if s.Players[seat].Sci[sym] != 2 {
		return outcomeFinish
	}
	emit(ev, Event{Kind: EvPair, Seat: seat, A: int16(sym)})
	if s.BoardTokens.Empty() {
		emit(ev, Event{Kind: EvEffectSkipped, Seat: seat, Card: c, A: int16(PhasePickBoardToken)})
		return outcomeFinish
	}
	s.Phase = PhasePickBoardToken // the active seat, who built c, stays to move
	return outcomeChoice
}

// takeBoardToken resolves the board token choice that a pair opened: seat
// takes token t from the board (EvToken, A = 0), then the token's immediate
// effects apply (resolveTokenEffects; Law can win the game). The caller has
// checked that t is on the board, and runs the rest of the turn as for any
// resolved choice. resolvePending calls it for PhasePickBoardToken.
func takeBoardToken(s *State, seat Seat, t catalog.TokenID, ev *[]Event) outcome {
	s.BoardTokens.Remove(t)
	addToken(s, seat, t)
	emit(ev, Event{Kind: EvToken, Seat: seat, Token: t})
	return resolveTokenEffects(s, seat, t, ev)
}
