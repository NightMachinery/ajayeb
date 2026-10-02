package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file decides which actions are legal. LegalActions lists them and
// IsLegal checks one; the two always agree (the property tests check that
// IsLegal accepts exactly the listed actions). The server accepts a key only
// if it is in the list for the seat to move.
//
// The legal actions of each phase:
//
//   - PhaseWonderDraft: draft each wonder of the current round not taken yet
//     (4, 3 or 2 of them; the round's last wonder is given automatically).
//   - PhasePlay: for each accessible card, in slot order: construct it if
//     the mover can pay for it; discard it (always legal); and build each of
//     the mover's wonders with it, in wonder-slot order, when the wonder is
//     unbuilt, not removed and affordable and fewer than seven wonders have
//     been built. Any accessible card may be tucked under a wonder,
//     whatever its own cost.
//   - PhaseChooseStarter: starter:0 and starter:1.
//   - PhaseDestroyCard: destroy each opponent card of colour s.Destroy.
//   - PhaseBuildFromDiscard: revive each card of the discard pile.
//   - PhasePickBoardToken: take each token on the board.
//   - PhasePickLibraryToken: take each token of the Great Library offer.
//   - PhaseGameOver: none.
//
// Lists are in ascending id order wherever no slot order applies.

// MaxLegalActions bounds the length of a LegalActions list (the discard
// pile can offer at most 60 cards to revive; the play phase has at most 6
// accessible cards with 6 actions each). A buffer of this capacity never
// grows.
const MaxLegalActions = 64

// LegalActions appends the legal actions of the player to move to buf and
// returns the extended slice. It never allocates when buf has room for
// MaxLegalActions more entries, so callers in hot paths reuse one buffer:
// legal = LegalActions(s, legal[:0]). The list is empty only when the game
// is over.
func LegalActions(s *State, buf []Action) []Action {
	seat := s.ToMove
	switch s.Phase {
	case PhaseWonderDraft:
		n := draftPicks(s)
		if n >= NumDraftWonders {
			return buf
		}
		round := draftRound(n)
		for i := 4 * round; i < 4*round+4; i++ {
			if s.DraftTaken&(1<<i) == 0 {
				buf = append(buf, DraftAction(s.DraftOrder[i]))
			}
		}
	case PhasePlay:
		wonders := affordableWonders(s, seat)
		acc := Accessible(s)
		for i := range s.Slots {
			if acc&(1<<i) == 0 {
				continue
			}
			c := s.Slots[i].Card
			if BuildingQuote(s, seat, c).Affordable {
				buf = append(buf, ConstructAction(c))
			}
			buf = append(buf, DiscardAction(c))
			for k := range s.Players[seat].Wonders {
				if wonders&(1<<k) != 0 {
					buf = append(buf, WonderAction(s.Players[seat].Wonders[k].ID, c))
				}
			}
		}
	case PhaseChooseStarter:
		buf = append(buf, StarterAction(0), StarterAction(1))
	case PhaseDestroyCard:
		opp := &s.Players[seat.Opp()]
		for c, ok := opp.City.Next(1); ok; c, ok = opp.City.Next(c + 1) {
			if cardColor(c) == s.Destroy {
				buf = append(buf, DestroyAction(c))
			}
		}
	case PhaseBuildFromDiscard:
		for c, ok := s.Discard.Next(1); ok; c, ok = s.Discard.Next(c + 1) {
			buf = append(buf, ReviveAction(c))
		}
	case PhasePickBoardToken:
		for t, ok := s.BoardTokens.Next(1); ok; t, ok = s.BoardTokens.Next(t + 1) {
			buf = append(buf, TokenAction(t))
		}
	case PhasePickLibraryToken:
		for _, t := range s.Library {
			if t != 0 {
				buf = append(buf, TokenAction(t))
			}
		}
	}
	return buf
}

// IsLegal reports whether action a is legal for the player to move. It
// returns nil or an *IllegalError whose Reason says why not. It never
// changes s.
func IsLegal(s *State, a Action) error {
	if r := legality(s, a); r != ReasonNone {
		return illegal(s, a, r)
	}
	return nil
}

// phaseActions is the action type each phase accepts (PhasePlay accepts
// three, checked separately).
var phaseActions = [numPhases]ActionType{
	PhaseWonderDraft:      ActDraftWonder,
	PhaseChooseStarter:    ActChooseStarter,
	PhaseDestroyCard:      ActDestroyCard,
	PhaseBuildFromDiscard: ActBuildFromDiscard,
	PhasePickBoardToken:   ActChooseProgressToken,
	PhasePickLibraryToken: ActChooseProgressToken,
}

// legality returns why a is not legal now, or ReasonNone.
func legality(s *State, a Action) Reason {
	if s.Phase == PhaseGameOver {
		return ReasonGameOver
	}
	if s.Phase >= numPhases {
		return ReasonWrongPhase
	}
	if a.Validate() != nil {
		return ReasonMalformed
	}
	seat := s.ToMove
	if s.Phase == PhasePlay {
		switch a.Type {
		case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
		default:
			return ReasonWrongPhase
		}
	} else if a.Type != phaseActions[s.Phase] {
		return ReasonWrongPhase
	}

	switch a.Type {
	case ActDraftWonder:
		if draftIndex(s, a.Wonder) < 0 {
			return ReasonNotInDraft
		}
	case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
		i := SlotOf(s, a.Card)
		if i < 0 || Accessible(s)&(1<<i) == 0 || !s.Slots[i].FaceUp {
			return ReasonNotAccessible
		}
		switch a.Type {
		case ActConstructBuilding:
			if !BuildingQuote(s, seat, a.Card).Affordable {
				return ReasonUnaffordable
			}
		case ActConstructWonder:
			k := wonderSlot(&s.Players[seat], a.Wonder)
			if k < 0 || !wonderOpen(s, seat, k) {
				return ReasonWonderUnavailable
			}
			if !WonderQuote(s, seat, a.Wonder).Affordable {
				return ReasonUnaffordable
			}
		}
	case ActDestroyCard:
		if !s.Players[seat.Opp()].City.Has(a.Card) || cardColor(a.Card) != s.Destroy {
			return ReasonNotTarget
		}
	case ActBuildFromDiscard:
		if !s.Discard.Has(a.Card) {
			return ReasonNotInDiscard
		}
	case ActChooseProgressToken:
		if s.Phase == PhasePickBoardToken {
			if !s.BoardTokens.Has(a.Token) {
				return ReasonTokenUnavailable
			}
		} else if libraryIndex(s, a.Token) < 0 {
			return ReasonTokenUnavailable
		}
	case ActChooseStarter:
		// Either seat may be chosen.
	}
	return ReasonNone
}

// draftIndex returns the index in DraftOrder of wonder w if it is on offer
// in the current draft round, or -1.
func draftIndex(s *State, w catalog.WonderID) int {
	n := draftPicks(s)
	if n >= NumDraftWonders {
		return -1
	}
	round := draftRound(n)
	for i := 4 * round; i < 4*round+4; i++ {
		if s.DraftOrder[i] == w && s.DraftTaken&(1<<i) == 0 {
			return i
		}
	}
	return -1
}

// wonderSlot returns the index of wonder w among p's wonders, or -1.
func wonderSlot(p *Player, w catalog.WonderID) int {
	if w == 0 {
		return -1
	}
	for k := range p.Wonders {
		if p.Wonders[k].ID == w {
			return k
		}
	}
	return -1
}

// wonderOpen reports whether seat's wonder in slot k may still be built:
// it is unbuilt and not removed, and fewer than seven wonders have been
// built in the game.
func wonderOpen(s *State, seat Seat, k int) bool {
	ws := &s.Players[seat].Wonders[k]
	return ws.ID != 0 && !ws.Built && !ws.Removed && s.WondersBuilt < MaxWondersBuilt
}

// affordableWonders returns the wonder slots of seat that may be built now,
// as a bitmask over the four slots. A wonder's cost does not depend on the
// card tucked under it, so this is computed once per LegalActions call.
func affordableWonders(s *State, seat Seat) uint8 {
	var m uint8
	for k := range s.Players[seat].Wonders {
		if wonderOpen(s, seat, k) && WonderQuote(s, seat, s.Players[seat].Wonders[k].ID).Affordable {
			m |= 1 << k
		}
	}
	return m
}

// libraryIndex returns the place of token t in the Great Library offer, or
// -1.
func libraryIndex(s *State, t catalog.TokenID) int {
	if t == 0 {
		return -1
	}
	for i, x := range s.Library {
		if x == t {
			return i
		}
	}
	return -1
}

// cardColor returns the colour of card c, or NoColor for an id that is not
// registered.
func cardColor(c catalog.CardID) catalog.Color {
	if int(c) < len(catalog.T.Color) {
		return catalog.T.Color[c]
	}
	return catalog.NoColor
}
