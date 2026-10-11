package engine

import (
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file defines the events that Apply reports. It is frozen after slice
// S1: change it only through the coordinator.
//
// Events describe what an action did, in order, so that clients can animate
// it and announce it. The state never stores events: replaying the actions
// regenerates them. An event is a small comparable value; which fields
// matter depends on Kind, as listed below, and unused fields are zero.
//
// Most events are public. An event with Private set may be shown only to
// Seat; the per-viewer event filter must hide it from everyone else (it may
// replace it with a public summary). The filter must also treat every
// EvLibraryDraw as private, whatever its Private field says.

// EventKind is what happened.
type EventKind uint8

// The event kinds and the fields each one uses.
const (
	EvNone EventKind = iota

	// Draft and setup of the ages.
	EvDraft      // Seat took Wonder in the draft; A = 1 when the pick was automatic (the last wonder of a group)
	EvDraftRound // the second draft round's four wonders were revealed; A = the round number (2)
	EvAgeStart   // A = the age that starts (1 to 3); its structure has been dealt

	// Taking a card.
	EvBuild   // Seat built Card
	EvDiscard // Seat discarded Card; A = coins gained
	EvWonder  // Seat built Wonder, tucking Card
	EvPay     // Seat paid A coins in total; B of them went to the opponent (Economy)
	EvReveal  // the face-down card in structure slot A turned face up: Card

	// Effects.
	EvCoins         // Seat's coins changed by A (signed) to B; the source is Card, Wonder or Token (whichever is set)
	EvPawn          // the conflict pawn moved from A to B, pushed by Seat
	EvLoot          // a looting token was taken: Seat lost A coins (the token is worth B)
	EvPair          // Seat completed a science pair of symbol A
	EvToken         // Seat took Token; A = 1 when it came from the Great Library offer
	EvLibraryDraw   // Private to Seat: Token is one of the tokens the Great Library drew; A = its place in the offer (0 to 2)
	EvDestroy       // Seat destroyed the opponent's Card, which went to the discard pile
	EvRebuild       // Seat built Card from the discard pile (Mausoleum)
	EvWonderRemoved // Seat's Wonder went back to the box (seven-wonders rule)
	EvEffectSkipped // Wonder's (or, with Card set, the card's) special choice had no option and was skipped; A = the phase it would have opened
	EvExtraTurn     // Seat will play again
	EvExtraTurnLost // Seat's extra turn was lost because the age ended
	EvStarterChosen // Seat chose seat A to start the age
	EvGameOver      // the game ended: A = the VictoryKind, B = the winner seat or SharedVictory (-1)
	EvResign        // Seat resigned (ActResign); EvGameOver follows with the other seat as the winner
	numEventKinds
)

var eventNames = [numEventKinds]string{
	"none",
	"draft", "draftRound", "ageStart",
	"build", "discard", "wonder", "pay", "reveal",
	"coins", "pawn", "loot", "pair", "token", "libraryDraw", "destroy", "rebuild",
	"wonderRemoved", "effectSkipped", "extraTurn", "extraTurnLost", "starterChosen", "gameOver",
	"resign",
}

// String returns the wire name, for example "libraryDraw".
func (k EventKind) String() string {
	if k < numEventKinds {
		return eventNames[k]
	}
	return "event#" + strconv.Itoa(int(k))
}

// MarshalText encodes the kind as its wire name.
func (k EventKind) MarshalText() ([]byte, error) {
	return enumText(eventNames[:], uint8(k), "event")
}

// UnmarshalText decodes a wire name exactly.
func (k *EventKind) UnmarshalText(b []byte) error {
	v, err := parseEnum(eventNames[:], b, "event")
	if err == nil {
		*k = EventKind(v)
	}
	return err
}

// Event is one thing an action did. See the kinds above for the fields.
type Event struct {
	Kind    EventKind
	Seat    Seat
	Card    catalog.CardID
	Wonder  catalog.WonderID
	Token   catalog.TokenID
	A, B    int16
	Private bool // only Seat may see this event
}

// emit appends e to *ev. A nil ev records nothing, so hot paths (the AI's
// ApplyTrusted) can pass nil.
func emit(ev *[]Event, e Event) {
	if ev != nil {
		*ev = append(*ev, e)
	}
}
