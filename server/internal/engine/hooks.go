package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// The apply hooks are the fixed seams between the turn machine and the rules
// of individual effects, so that the slices can fill separate files without
// touching each other's code:
//
//   - apply.go and finishTurn (slice S4) own the flow: validation, taking the
//     card from the structure, payment (with the Economy transfer), the city
//     bookkeeping in city.go (addCard, addToken, buildWonder and the caches),
//     reveals, the end of an age and the starter choice, the end of the game.
//   - effects.go (slice S5a; it may split its hooks into military.go,
//     science.go and score.go) owns the effects of buildings and tokens, the
//     military track, science and scoring.
//   - wonders.go and pending.go (slice S5b) own the effects of wonders, the
//     seven-wonders rule and the four pending choices.
//
// Every hook mutates s in place and appends events through emit (ev may be
// nil). A hook never runs finishTurn; it returns an outcome and the caller
// continues from there. The hooks are plain functions; the declarations at
// the bottom of this file pin their signatures, so a slice that changed one
// would fail to compile here.
//
// resolveBuildingEffects(s, seat, card, via, ev) outcome
//
//	Runs after the building has been paid for (when bought) and added to the
//	seat's city by addCard. Applies, in this order: the Urbanism bonus when
//	via is viaChain (never for viaDiscard); coin effects, counted with the new
//	card already in the city; shields, plus Strategy's extra shield for red
//	buildings, through movePawn; then science: six different symbols end the
//	game, and a new pair opens PhasePickBoardToken if the board has a token.
//	Called by S4 for construct and by S5b for revive (Mausoleum, viaDiscard).
//
// resolveTokenEffects(s, seat, token, ev) outcome
//
//	Runs after addToken has put the token into the seat's tokens (and, for
//	Law, its symbol into the science counts). Applies the token's immediate
//	effects (the coins of Agriculture and Urbanism) and checks science
//	supremacy (Law can complete six symbols). Called by S5b for both token
//	choices.
//
// movePawn(s, seat, shields, ev) outcome
//
//	Moves the conflict pawn shields spaces toward seat's opponent, one space
//	at a time, taking each looting token the pawn reaches (the opponent loses
//	up to its value, never going below zero). Reaching the capital ends the
//	game with a military victory at once (outcomeOver). Called by S5a for red
//	buildings and by S5b for wonders with shields.
//
// checkSupremacy(s, ev) bool
//
//	Ends the game if either player has a military or science supremacy and
//	reports whether it did. Called by S4's finishTurn before an age can end.
//
// civilianResult(s) Result
//
//	Scores the game at the end of Age III: both players' Score by category,
//	the winner by total, ties broken by blue-card points, else SharedVictory.
//	Pure; S4 stores the result, sets PhaseGameOver and emits EvGameOver.
//
// Scores(s) [2]Score
//
//	The score preview of both players as if the game ended now. Pure; used by
//	civilianResult and the view.
//
// resolveWonderEffects(s, seat, wonder, ev) outcome
//
//	Runs after the wonder has been paid for (with Architecture and the
//	Economy transfer) and marked built by buildWonder (tucked card, built
//	count, production cache). Applies, in this order: the seven-wonders rule
//	when this was the seventh wonder; coins; shields through movePawn; the
//	extra turn (the wonder's own or Theology's, never two); the opponent's
//	coin loss (Appian Way); then the special choice: destroy (if the opponent
//	has a card of that colour), build from the discard pile (if it is not
//	empty) or the Great Library draw. A special choice with no option is
//	skipped (EvEffectSkipped) and the turn ends normally.
//
// resolvePending(s, action, ev) outcome
//
//	Resolves the open choice of PhaseDestroyCard, PhaseBuildFromDiscard,
//	PhasePickBoardToken or PhasePickLibraryToken with an action the caller
//	has already checked with IsLegal. It clears the phase's data (Destroy,
//	Library) and, for the library, returns the two tokens not taken to the
//	box. A revived building runs resolveBuildingEffects (viaDiscard), which
//	may open a token choice of its own. The starter choice is not a pending
//	choice: S4's flow resolves it.

// buildVia says how a building entered a city.
type buildVia uint8

const (
	viaPurchase buildVia = iota // bought (possibly for nothing, when its cost is free)
	viaChain                    // free through a chain symbol: Urbanism pays its bonus
	viaDiscard                  // built from the discard pile by the Mausoleum: no Urbanism bonus
)

// outcome tells the caller of a hook how the turn continues.
type outcome uint8

const (
	// outcomeFinish: nothing is left open. The caller sets the phase back to
	// PhasePlay (if a pending choice was being resolved) and runs finishTurn.
	outcomeFinish outcome = iota
	// outcomeChoice: the hook opened a choice: s.Phase is the pending phase,
	// its data (Destroy, Library) is set, and ToMove is still Active. The
	// turn finishes after the choice is resolved.
	outcomeChoice
	// outcomeOver: the game ended. s.Phase is PhaseGameOver, s.Result is set
	// and EvGameOver has been emitted. The caller stops.
	outcomeOver
)

// String names the value for test failures.
func (v buildVia) String() string {
	switch v {
	case viaPurchase:
		return "purchase"
	case viaChain:
		return "chain"
	case viaDiscard:
		return "discard"
	}
	return "via#?"
}

// String names the value for test failures.
func (o outcome) String() string {
	switch o {
	case outcomeFinish:
		return "finish"
	case outcomeChoice:
		return "choice"
	case outcomeOver:
		return "over"
	}
	return "outcome#?"
}

// The fixed hook signatures.
var (
	_ func(*State, Seat, catalog.CardID, buildVia, *[]Event) outcome = resolveBuildingEffects
	_ func(*State, Seat, catalog.TokenID, *[]Event) outcome          = resolveTokenEffects
	_ func(*State, Seat, uint8, *[]Event) outcome                    = movePawn
	_ func(*State, *[]Event) bool                                    = checkSupremacy
	_ func(*State) Result                                            = civilianResult
	_ func(*State) [2]Score                                          = Scores
	_ func(*State, Seat, catalog.WonderID, *[]Event) outcome         = resolveWonderEffects
	_ func(*State, Action, *[]Event) outcome                         = resolvePending
)
