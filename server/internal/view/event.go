package view

// GameEvent is one thing an action did, as one viewer may see it. The
// events of an action come in the order things happened, so a client can
// animate them and announce them. Only the fields the kind uses are set:
//
//   - draft: seat, wonder, auto (the last wonder of a round, given without a
//     choice)
//   - draftRound: round (2: the second round's wonders were revealed)
//   - ageStart: age (its structure has been laid out)
//   - build: seat, card
//   - discard: seat, card, amount (coins received)
//   - wonder: seat, wonder, card (tucked under it)
//   - pay: seat, amount (coins paid in total), toOpponent (the part that
//     went to the opponent)
//   - reveal: slot, card (a face-down card turned face up)
//   - coins: seat, amount (the change, which may be negative), balance (the
//     coins after it), and the source: card, wonder or token
//   - pawn: seat (who pushed it), from, to
//   - loot: seat (who lost coins), amount (coins lost), value (the token's
//     value)
//   - pair: seat, symbol
//   - token: seat, token, fromLibrary
//   - libraryDraw: seat, place (in the offer, 0 to 2), and token only in the
//     drawer's own events; everyone else sees that a token was drawn, never
//     which one
//   - destroy: seat (who destroyed), card (the opponent's card, now in the
//     discard pile)
//   - rebuild: seat, card (built from the discard pile)
//   - wonderRemoved: seat (the owner), wonder
//   - effectSkipped: seat, wonder or card (whose choice had no option),
//     skipped (the choice it would have opened)
//   - extraTurn, extraTurnLost: seat
//   - starterChosen: seat (who chose), starter
//   - gameOver: victory, winner (absent for a shared victory)
//   - resign: seat (who forfeited; a gameOver with the other seat as the
//     winner follows)
type GameEvent struct {
	Kind        EventKind      `json:"kind"`
	Private     bool           `json:"private"` // only this viewer sees this event as it is
	Seat        *int           `json:"seat,omitempty"`
	Card        *string        `json:"card,omitempty"`
	Wonder      *string        `json:"wonder,omitempty"`
	Token       *string        `json:"token,omitempty"`
	Slot        *int           `json:"slot,omitempty"`
	Place       *int           `json:"place,omitempty"`
	Amount      *int           `json:"amount,omitempty"`
	Balance     *int           `json:"balance,omitempty"`
	ToOpponent  *int           `json:"toOpponent,omitempty"`
	From        *int           `json:"from,omitempty"`
	To          *int           `json:"to,omitempty"`
	Value       *int           `json:"value,omitempty"`
	Symbol      *ScienceSymbol `json:"symbol,omitempty"`
	Age         *int           `json:"age,omitempty"`
	Round       *int           `json:"round,omitempty"`
	Auto        *bool          `json:"auto,omitempty"`
	FromLibrary *bool          `json:"fromLibrary,omitempty"`
	Skipped     *PendingKind   `json:"skipped,omitempty"`
	Starter     *int           `json:"starter,omitempty"`
	Victory     *VictoryKind   `json:"victory,omitempty"`
	Winner      *int           `json:"winner,omitempty"`
}
