package view

// HintSettings are a game's hint switches (docs/dev/decisions.md D23): one
// bool per hint family, true when the family is on. They belong to the
// room, not to the engine's options, so they never change the rules or a
// replay: package room keeps them with the room file and sends the same
// settings to every connection in RoomInfo.hints, and the game's admin
// may change them at any time while the game runs (the wire's "hints"
// frame). The families are those of the hints design; the defaults (every
// family on, except advisor outside games against the bot) are
// room.DefaultHints.
//
// A struct, not a map, so the client gets a fixed shape and the order of
// the families never depends on map iteration. A new family is a new
// field: a room file written before it existed reads it as false, so the
// field's arrival must also teach package room's loader its default.
type HintSettings struct {
	Prices   bool `json:"prices"`
	Economy  bool `json:"economy"`
	Science  bool `json:"science"`
	Military bool `json:"military"`
	Threats  bool `json:"threats"`
	Tempo    bool `json:"tempo"`
	Wonders  bool `json:"wonders"`
	Tokens   bool `json:"tokens"`
	Scoring  bool `json:"scoring"`
	Counting bool `json:"counting"`
	Recap    bool `json:"recap"`
	// Advisor is the post-move review, shown only to the player who moved.
	Advisor bool `json:"advisor"`
}

// HintsView holds a view's board-level hints: the facts the server
// computes from public information for every viewer alike (the shared
// hints of the hints design), once per broadcast with the room's hint
// settings. A view carries it while the game runs and the room asked for
// hints (engine.ViewWithHints); a gameOver view never does, and neither
// does a view built without settings (engine.ViewFor, used by the
// simulator, tests and replays).
//
// It is empty for now: the hints slices after M10.HS1 add its fields, each
// zero or empty while its family is off. A client reads the room's
// settings (RoomInfo.hints) to know which fields are meaningful.
type HintsView struct{}

// The hint vocabularies below are the engine's own names (engine
// PublicPlace, ExtraTurnState, ExtraTurnSource and ScienceEffectKind); a
// test in the engine checks that both sides list the same values in the
// same order. They live here, next to the hint types, rather than in
// enums.go, whose own test covers the base view's vocabularies.

// PublicPlace is where a card is, as far as both players can know.
type PublicPlace string

// The public places.
const (
	PublicPlaceFaceUp         PublicPlace = "faceUp"         // face up in a structure slot
	PublicPlaceCity           PublicPlace = "city"           // in a player's city
	PublicPlaceTucked         PublicPlace = "tucked"         // under a built wonder
	PublicPlaceDiscard        PublicPlace = "discard"        // in the discard pile
	PublicPlaceUnseenThisAge  PublicPlace = "unseenThisAge"  // face down or set aside in the current age, unknown which
	PublicPlaceUnseenLaterAge PublicPlace = "unseenLaterAge" // in a later age's deck, dealt or set aside
	PublicPlaceRemovedKnown   PublicPlace = "removedKnown"   // an age that is over, and the card was never seen
)

// ExtraTurnState is what an action does to the mover's turn.
type ExtraTurnState string

// The extra-turn states.
const (
	ExtraTurnStateNone ExtraTurnState = "none" // the turn passes as usual
	ExtraTurnStateGain ExtraTurnState = "gain" // the mover plays again
	ExtraTurnStateLost ExtraTurnState = "lost" // an extra turn is granted on the age's last card, so it is lost
)

// ExtraTurnSource says why a wonder grants an extra turn.
type ExtraTurnSource string

// The extra-turn sources.
const (
	ExtraTurnSourceNone     ExtraTurnSource = "none"     // no extra turn
	ExtraTurnSourceWonder   ExtraTurnSource = "wonder"   // the wonder's own effect
	ExtraTurnSourceTheology ExtraTurnSource = "theology" // the builder owns Theology
)

// ScienceEffectKind is what a green card does to its builder's symbols.
type ScienceEffectKind string

// The science effect kinds.
const (
	ScienceEffectKindNew  ScienceEffectKind = "new"  // a symbol the seat did not have
	ScienceEffectKindPair ScienceEffectKind = "pair" // the second card of a symbol
)
