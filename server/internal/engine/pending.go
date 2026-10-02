package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file resolves the pending choices: the destroy choice of the Statue
// of Zeus and Circus Maximus, the Mausoleum's build from the discard pile,
// the board token choice of a science pair, and the Great Library's token
// pick. The wonders open their choices in wonders.go and a pair opens the
// board choice in science.go; the hook contract is in hooks.go.
//
// A choice belongs to the turn that opened it. The active player decides it
// (ToMove is Active), the cards the turn freed stay face down until it is
// made, and once it is made the turn ends as usual (finishTurn): the
// reveals, the end of the age or the game, the extra turn or the next
// player. A choice can open another one (a rebuilt green card can complete
// a pair) or end the game (a supremacy), and the next one then waits in its
// turn. The starter choice of a new age is not a pending choice; apply.go
// resolves it.

// resolvePending resolves the open pending choice with action a, which the
// caller has checked with IsLegal. The phase's data (Destroy, Library) is
// cleared before anything can end the game. It returns outcomeChoice when
// the resolution opened another choice, outcomeOver when it ended the game,
// and outcomeFinish otherwise.
func resolvePending(s *State, a Action, ev *[]Event) outcome {
	seat := s.ToMove
	switch s.Phase {
	case PhaseDestroyCard:
		return destroyCard(s, seat, a.Card, ev)
	case PhaseBuildFromDiscard:
		return reviveCard(s, seat, a.Card, ev)
	case PhasePickBoardToken:
		return takeBoardToken(s, seat, a.Token, ev)
	case PhasePickLibraryToken:
		return takeLibraryToken(s, seat, a.Token, ev)
	}
	return outcomeFinish
}

// destroyCard resolves the destroy choice: seat's opponent loses building c
// (of the colour s.Destroy) to the discard pile, so its production, and with
// it seat's trade price, drops at once (EvDestroy). Brown and grey cards
// carry no chain or science symbol, so nothing else changes.
func destroyCard(s *State, seat Seat, c catalog.CardID, ev *[]Event) outcome {
	s.Destroy = catalog.NoColor
	removeCard(s, seat.Opp(), c)
	s.Discard.Add(c)
	emit(ev, Event{Kind: EvDestroy, Seat: seat, Card: c})
	return outcomeFinish
}

// reviveCard resolves the Mausoleum's choice: seat builds card c from the
// discard pile for free (EvRebuild), and every effect of the building
// applies except Urbanism's chain bonus (resolveBuildingEffects with
// viaDiscard). Its science symbol can win the game or open the board token
// choice.
func reviveCard(s *State, seat Seat, c catalog.CardID, ev *[]Event) outcome {
	s.Discard.Remove(c)
	addCard(s, seat, c)
	emit(ev, Event{Kind: EvRebuild, Seat: seat, Card: c})
	return resolveBuildingEffects(s, seat, c, viaDiscard, ev)
}

// takeLibraryToken resolves the Great Library's pick: seat keeps token t of
// the offer (EvToken, A = 1), the other drawn tokens go back to the bottom
// of the box, unannounced, and the kept token's immediate effects apply
// (resolveTokenEffects: Agriculture's and Urbanism's coins; Law can win the
// game).
func takeLibraryToken(s *State, seat Seat, t catalog.TokenID, ev *[]Event) outcome {
	offer := s.Library
	s.Library = [LibraryDraw]catalog.TokenID{}
	for _, x := range offer {
		if x != 0 && x != t {
			returnToBox(s, x)
		}
	}
	addToken(s, seat, t)
	emit(ev, Event{Kind: EvToken, Seat: seat, Token: t, A: 1})
	return resolveTokenEffects(s, seat, t, ev)
}

// returnToBox puts token t at the bottom of the box, the first empty place.
// The box always has room: the tokens returned were drawn from it.
func returnToBox(s *State, t catalog.TokenID) {
	for i := range s.BoxTokens {
		if s.BoxTokens[i] == 0 {
			s.BoxTokens[i] = t
			return
		}
	}
}
