package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file is the turn machine: Apply and ApplyTrusted dispatch an action
// by phase, the three play actions take a card from the structure, and
// finishTurn ends the turn (reveals, the supremacy check, the end of an age
// or of the game, the extra turn, the next player). The effects of
// buildings, wonders, tokens and pending choices are in the hook files
// (effects.go, wonders.go, pending.go; see hooks.go for the contracts).
//
// Apply validates the whole action first and only then mutates the state,
// and no step after validation can fail, so an action either applies fully
// or (with an error) leaves the state exactly as it was.
//
// The events of one action come in the order things happen. The action's
// own event comes first (EvDraft, EvBuild, EvDiscard, EvWonder,
// EvStarterChosen, or what a pending choice reports), then its consequences:
// EvPay for the coins spent, then the effects the hooks report, then the end
// of the turn (EvReveal for each card turned over, EvExtraTurnLost,
// EvAgeStart, EvGameOver).

// Apply applies action a for the player to move, after checking that it is
// legal (IsLegal). It appends the action's events to *ev unless ev is nil.
// On error the state is unchanged and the error is an *IllegalError.
func Apply(s *State, a Action, ev *[]Event) error {
	if r := legality(s, a); r != ReasonNone {
		return illegal(s, a, r)
	}
	apply(s, a, ev)
	return nil
}

// ApplyTrusted applies an action already known to be legal (one returned by
// LegalActions for this exact state), without checking it and without
// events. It is the bots' hot path. An illegal action leaves the state
// invalid.
func ApplyTrusted(s *State, a Action) { apply(s, a, nil) }

// apply performs a legal action.
func apply(s *State, a Action, ev *[]Event) {
	s.Ply++
	switch s.Phase {
	case PhaseWonderDraft:
		applyDraft(s, draftIndex(s, a.Wonder), ev)
	case PhasePlay:
		switch a.Type {
		case ActConstructBuilding:
			applyConstruct(s, a.Card, ev)
		case ActDiscardBuilding:
			applyDiscard(s, a.Card, ev)
		case ActConstructWonder:
			applyWonder(s, a.Wonder, a.Card, ev)
		}
	case PhaseChooseStarter:
		applyStarter(s, a.Starter, ev)
	case PhaseDestroyCard, PhaseBuildFromDiscard, PhasePickBoardToken, PhasePickLibraryToken:
		continueTurn(s, resolvePending(s, a, ev), ev)
	}
}

// applyDiscard discards accessible card c for coins.
func applyDiscard(s *State, c catalog.CardID, ev *[]Event) {
	seat := s.Active
	takeSlot(s, SlotOf(s, c))
	gain := DiscardValue(s, seat)
	gainCoins(s, seat, gain)
	s.Discard.Add(c)
	emit(ev, Event{Kind: EvDiscard, Seat: seat, Card: c, A: gain})
	finishTurn(s, ev)
}

// applyConstruct builds accessible card c with the cheapest payment.
func applyConstruct(s *State, c catalog.CardID, ev *[]Event) {
	seat := s.Active
	q := BuildingQuote(s, seat, c)
	takeSlot(s, SlotOf(s, c))
	addCard(s, seat, c)
	emit(ev, Event{Kind: EvBuild, Seat: seat, Card: c})
	pay(s, seat, q, ev)
	via := viaPurchase
	if q.ChainFree {
		via = viaChain
	}
	continueTurn(s, resolveBuildingEffects(s, seat, c, via, ev), ev)
}

// applyWonder builds wonder w with the cheapest payment, tucking accessible
// card c under it.
func applyWonder(s *State, w catalog.WonderID, c catalog.CardID, ev *[]Event) {
	seat := s.Active
	q := WonderQuote(s, seat, w)
	takeSlot(s, SlotOf(s, c))
	buildWonder(s, seat, wonderSlot(&s.Players[seat], w), c)
	emit(ev, Event{Kind: EvWonder, Seat: seat, Wonder: w, Card: c})
	pay(s, seat, q, ev)
	continueTurn(s, resolveWonderEffects(s, seat, w, ev), ev)
}

// pay takes a payment's coins from seat. The trading part goes to the
// opponent when the opponent owns Economy (Quote.ToOpponent); the rest goes
// to the bank. Nothing is reported for a free construction.
func pay(s *State, seat Seat, q Quote, ev *[]Event) {
	if q.Total <= 0 {
		return
	}
	p := &s.Players[seat]
	p.Coins -= q.Total
	if q.ToOpponent > 0 {
		gainCoins(s, seat.Opp(), q.ToOpponent)
	}
	emit(ev, Event{Kind: EvPay, Seat: seat, A: q.Total, B: q.ToOpponent})
}

// applyStarter makes seat start the age that has just been laid out.
func applyStarter(s *State, starter Seat, ev *[]Event) {
	emit(ev, Event{Kind: EvStarterChosen, Seat: s.ToMove, A: int16(starter)})
	s.Active = starter
	s.ToMove = starter
	s.Phase = PhasePlay
}

// continueTurn goes on after a hook returned o.
func continueTurn(s *State, o outcome, ev *[]Event) {
	if o != outcomeFinish {
		return // a choice is open (outcomeChoice) or the game is over (outcomeOver)
	}
	s.Phase = PhasePlay
	finishTurn(s, ev)
}

// finishTurn ends the active player's turn once the action and every choice
// it opened are resolved:
//
//  1. every face-down card the turn freed is turned face up;
//  2. a supremacy ends the game;
//  3. if the structure is empty the age ends: an extra turn is lost, and
//     after Age III the game is scored, otherwise the next age is laid out
//     and its starter decided (startNextAge);
//  4. otherwise the active player plays again after an extra turn, or the
//     opponent's turn begins.
func finishTurn(s *State, ev *[]Event) {
	if s.Over() {
		return
	}
	revealAccessible(s, ev)
	if checkSupremacy(s, ev) {
		return
	}
	if Occupied(s) == 0 {
		if s.ExtraTurn {
			s.ExtraTurn = false
			emit(ev, Event{Kind: EvExtraTurnLost, Seat: s.Active})
		}
		if s.Age >= NumAges {
			endGame(s, ev)
			return
		}
		startNextAge(s, ev)
		return
	}
	s.Phase = PhasePlay
	if s.ExtraTurn {
		s.ExtraTurn = false
	} else {
		s.Active = s.Active.Opp()
	}
	s.ToMove = s.Active
}

// startNextAge lays out the next age and decides who starts it. The player
// on whose side the pawn stands (the weaker one) chooses. With the pawn in
// the centre the last active player chooses, or, under the
// LastActiveStarts option, simply starts. Active stays the last active
// player until the choice is made.
func startNextAge(s *State, ev *[]Event) {
	startAge(s, s.Age+1, ev)
	chooser, choose := starterChooser(s)
	if !choose {
		s.Phase = PhasePlay
		s.ToMove = s.Active
		return
	}
	s.Phase = PhaseChooseStarter
	s.ToMove = chooser
}

// starterChooser returns who decides the next age's starter, and false when
// no choice is made because the last active player starts directly.
func starterChooser(s *State) (Seat, bool) {
	switch {
	case s.Military > 0: // toward seat 1's capital: seat 1 is weaker
		return 1, true
	case s.Military < 0:
		return 0, true
	case s.Opts.CenterPawn == LastActiveStarts:
		return s.Active, false
	}
	return s.Active, true
}

// endGame scores a game that reached the end of Age III.
func endGame(s *State, ev *[]Event) {
	s.Result = civilianResult(s)
	s.Phase = PhaseGameOver
	emit(ev, Event{Kind: EvGameOver, A: int16(s.Result.Kind), B: int16(s.Result.Winner)})
}
