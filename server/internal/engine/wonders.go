package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// This file applies the effects of a wonder once it is built. The choices
// that some wonders open are resolved in pending.go; the hook contracts are
// in hooks.go.
//
// # The order of a wonder's effects
//
// The turn machine (apply.go) takes the tucked card from the structure,
// marks the wonder built (buildWonder: the tucked card, the built count, the
// flexible production), and pays for it: the printed coins go to the bank,
// and the trading part goes to an opponent who owns Economy. Then
// resolveWonderEffects applies, in this order:
//
//  1. the seven-wonder rule, when this was the seventh wonder of the game:
//     the one wonder still unbuilt, whichever player drafted it, goes back
//     to the box;
//  2. the wonder's coins, from the bank;
//  3. its shields, which move the pawn; Strategy adds nothing to a wonder.
//     Reaching a capital ends the game here, so a destroy choice that would
//     follow never opens;
//  4. the extra turn, when the wonder grants one or its builder owns
//     Theology (one extra turn either way, never two);
//  5. the Appian Way's loss: the opponent pays 3 coins to the bank, never
//     going below zero, after any Economy income of this build;
//  6. the special choice, if the wonder has one: destroy an opponent card
//     of a colour (the Statue of Zeus, Circus Maximus), build a card from
//     the discard pile (the Mausoleum), or keep one of the tokens drawn from
//     the box (the Great Library). A choice with no option is skipped
//     (EvEffectSkipped) and the turn ends normally.
//
// The military supremacy check is part of step 3. A wonder never changes
// anyone's science symbols, so no other supremacy can start in these steps.
// The wonder's points need nothing here: the score counts the built wonders.

// resolveWonderEffects applies the effects of wonder w, which seat has just
// built, in the order given at the top of this file. It returns outcomeOver
// when the wonder's shields won the game, outcomeChoice when it opened a
// choice for seat, and outcomeFinish otherwise.
func resolveWonderEffects(s *State, seat Seat, w catalog.WonderID, ev *[]Event) outcome {
	sevenWonderRule(s, ev)
	if int(w) >= len(catalog.T.WVP) {
		return outcomeFinish
	}
	wonderCoins(s, seat, w, ev)
	if wonderShields(s, seat, w, ev) == outcomeOver {
		return outcomeOver
	}
	if catalog.T.WPlayAgain[w] || wondersPlayAgain(&s.Players[seat]) {
		s.ExtraTurn = true
		emit(ev, Event{Kind: EvExtraTurn, Seat: seat})
	}
	if n := catalog.T.WOppLoses[w]; n > 0 {
		loseCoins(s, seat.Opp(), int(n), fromWonder(w), ev)
	}
	return openWonderChoice(s, seat, w, ev)
}

// openWonderChoice opens the special choice of wonder w for seat, when the
// wonder has one and the choice has an option (outcomeChoice; seat, the
// active player, stays to move). A choice without an option is reported
// with EvEffectSkipped (Wonder = w, A = the phase it would have opened) and
// gives nothing.
func openWonderChoice(s *State, seat Seat, w catalog.WonderID, ev *[]Event) outcome {
	t := &catalog.T
	var phase Phase
	switch {
	case t.WDestroy[w] != catalog.NoColor:
		phase = PhaseDestroyCard
		if c := t.WDestroy[w]; s.Players[seat.Opp()].ColorN[c] > 0 {
			s.Destroy = c
			s.Phase = phase
			return outcomeChoice
		}
	case t.WRebuild[w]:
		phase = PhaseBuildFromDiscard
		if !s.Discard.Empty() {
			s.Phase = phase
			return outcomeChoice
		}
	case t.WDraw[w] > 0:
		phase = PhasePickLibraryToken
		if drawLibrary(s, seat, int(t.WDraw[w]), ev) > 0 {
			s.Phase = phase
			return outcomeChoice
		}
	default:
		return outcomeFinish
	}
	emit(ev, Event{Kind: EvEffectSkipped, Seat: seat, Wonder: w, A: int16(phase)})
	return outcomeFinish
}

// drawLibrary moves up to n tokens (at most LibraryDraw) from the front of
// the box into the Great Library offer, keeping the box a pile with no
// gaps, and reports each drawn token to seat alone (EvLibraryDraw, Private,
// A = its place in the offer). It returns the number drawn. The box was
// shuffled when the game was created, so its front is a random draw. A
// real game's box always holds five tokens here (nothing else draws from
// it); only a scenario can leave it shorter.
func drawLibrary(s *State, seat Seat, n int, ev *[]Event) int {
	n = min(n, LibraryDraw)
	k := 0
	for k < n && s.BoxTokens[k] != 0 {
		k++
	}
	s.Library = [LibraryDraw]catalog.TokenID{}
	copy(s.Library[:], s.BoxTokens[:k])
	copy(s.BoxTokens[:], s.BoxTokens[k:])
	for i := NumBoxTokens - k; i < NumBoxTokens; i++ {
		s.BoxTokens[i] = 0
	}
	for i := range k {
		emit(ev, Event{Kind: EvLibraryDraw, Seat: seat, Token: s.Library[i], A: int16(i), Private: true})
	}
	return k
}
