package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// Frozen after M15.haftgane-core-types. Events contain no raw action key.
type EventKind uint8

const (
	EvNone                EventKind = iota
	EvAgeStart                      // Age
	EvHandsPassed                   // Age, Turn (direction follows Age)
	EvSideChosen                    // Seat, Side
	EvMilitary                      // Seat, Other, Age, Outcome (one adjacent pair)
	EvGameOver                      // result is in State.Result
	EvCoins                         // Seat, Delta, Source; Design/Stage identifies placement or sale income
	EvPlay                          // Seat, Action, Stage, Left/Right, Printed; Design is public only for build
	EvLeftoverDiscarded             // Seat; Design private
	EvCardKept                      // Seat; Design private
	EvDiscardBuild                  // Seat, Design (public)
	EvDiscardBuildNothing           // Seat (public, including a skipped pile look)
	NumEventKinds
)

var eventNames = [NumEventKinds]string{"none", "age-start", "hands-passed", "side-chosen", "military", "game-over", "coins", "play", "leftover-discarded", "card-kept", "discard-build", "discard-build-nothing"}

func (k EventKind) String() string {
	if k < NumEventKinds {
		return eventNames[k]
	}
	return ""
}

// EvCoins reports only when-placed income (aggregated per placement) and sale
// income. Printed and trade payments are already in EvPlay; trade receipts are
// derived from neighbours' plays and never emit additional coin events.
type CoinSource uint8

const (
	CoinNone CoinSource = iota
	CoinSale
	CoinDesign
	CoinStage
	NumCoinSources
)

type ConflictOutcome uint8

const (
	ConflictTie ConflictOutcome = iota
	ConflictSeatWins
	ConflictOtherWins
	NumConflictOutcomes
)

// Only fields documented on Kind are meaningful. The filter derives redaction
// solely from Kind and Action, never Private. Private is an assertion equal to
// "this event has a hidden field": a non-build EvPlay, EvLeftoverDiscarded or
// EvCardKept. The full variant belongs to Seat; the filter builds the public
// summary by whitelist, never by copying this raw struct to another viewer.
type Event struct {
	Kind                 EventKind
	Seat, Other          Seat
	Age, Turn            uint8
	Action               ActionKind
	Design               catalog.DesignID
	Stage                uint8 // One-based stage number, zero when no stage is named.
	Side                 catalog.Side
	Left, Right, Printed uint8
	Delta                int16
	Source               CoinSource
	Outcome              ConflictOutcome
	Private              bool
}
