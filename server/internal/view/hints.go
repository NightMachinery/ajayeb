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
// Its fields are zero or empty while their family is off. A client reads
// the room's settings (RoomInfo.hints) to know which fields are
// meaningful.
type HintsView struct {
	// Discard is what rebuilding each discarded card would do, for each
	// seat whose Mausoleum is open, and for the seat choosing a card to
	// rebuild while it does (the wonders family): one entry per card and
	// seat that has something to say (a pip, an effect or a price it
	// moves), by seat, then in ascending card order.
	Discard []DiscardHintView `json:"discard"`
}

// CardHints is what building one face-up card would do for one seat now,
// as the cities stand: SlotView.hints holds one per seat, for every face-up
// card, covered or not (as if it were accessible), while the game runs and
// any per-card family is on. The numbers are those of now; the opponent's
// next move can change them.
type CardHints struct {
	Codes        []HintCode    `json:"codes"`                  // the board pips this card earns for this seat, in a fixed order (win, pair, loot); the client ranks them
	Effect       *EffectView   `json:"effect,omitempty"`       // the immediate result of building it, when one of its parts is on and says something
	PriceShift   []PriceShift  `json:"priceShift"`             // economy: the trade prices building it would change, for either seat
	SavesInView  *int          `json:"savesInView,omitempty"`  // economy, producer cards only (brown, grey, flexible producers, fixed prices): the coins it would save this seat on the other face-up cards and its open wonders, summed
	LastProducer *LastProducer `json:"lastProducer,omitempty"` // counting, brown and grey cards only, when no other producer of its resource is in view
}

// EffectView is the immediate result of building a card for one seat:
// payment (when the seat can pay), the building's coins, shields and
// looting, and science, before any choice it opens. To keep the view small
// it is sent only when one of its parts says something: a green card's
// science, a red card's pawn move, a win, or the worth of a yellow card or
// a guild (coins and scoreDelta are sent for those only).
type EffectView struct {
	Paid          bool           `json:"paid"`               // the seat can afford it now, and the preview paid the cheapest payment; otherwise it paid nothing
	Coins         int            `json:"coins"`              // scoring: coins the building gives on construction (Urbanism's chain bonus included), the payment not counted
	ScoreDelta    int            `json:"scoreDelta"`         // scoring: change of this seat's score preview
	OppScoreDelta int            `json:"oppScoreDelta"`      // scoring: change of the other seat's score preview (looting, the pawn)
	Science       *ScienceEffect `json:"science,omitempty"`  // science: a green card's symbol
	Military      *PawnMoveView  `json:"military,omitempty"` // military: the pawn move, when the card has shields
	Wins          *WinView       `json:"wins,omitempty"`     // threats: building it wins, at once or through the board token its pair lets the seat take
}

// ScienceEffect is what a green card does to its builder's symbols.
type ScienceEffect struct {
	Symbol        ScienceSymbol     `json:"symbol"`
	Kind          ScienceEffectKind `json:"kind"`
	DistinctAfter int               `json:"distinctAfter"` // the builder's different symbols afterwards
	TokenOnBoard  bool              `json:"tokenOnBoard"`  // a pair opens the board token choice (the board still has a token)
}

// PawnMoveView is how an action moves the conflict pawn, and the looting
// tokens it takes.
type PawnMoveView struct {
	From int         `json:"from"`
	To   int         `json:"to"`
	Loot []LootTaken `json:"loot"` // in the order the pawn reaches them
}

// LootTaken is one looting token a pawn move takes.
type LootTaken struct {
	At   int `json:"at"`   // its space: -6, -3, 3 or 6
	Lost int `json:"lost"` // coins the victim actually loses (no more than it has)
}

// WinView is a win an action would bring its mover.
type WinView struct {
	Kind  VictoryKind `json:"kind"`
	Steps int         `json:"steps"`         // 1 for a win this turn; 2 through the mover's own extra turn
	Via   *string     `json:"via,omitempty"` // the follow-up action key that wins, such as "token:law"; absent for a win at once
}

// PriceShift is one trade price that changes: seat's price of one unit of
// resource.
type PriceShift struct {
	Seat     int      `json:"seat"` // whose price changes
	Resource Resource `json:"resource"`
	From     int      `json:"from"`
	To       int      `json:"to"`
}

// DiscardHintView is what rebuilding one discarded card with the Mausoleum
// would do for one seat (the Mausoleum's owner).
type DiscardHintView struct {
	Card      string    `json:"card"`
	Seat      int       `json:"seat"`      // the seat it is computed for
	Hints     CardHints `json:"hints"`     // as for a face-up card; the rebuild is free, so effect.paid is false
	Total     int       `json:"total"`     // what the seat would pay to build it now, for "worth N coins to you"
	ChainFree bool      `json:"chainFree"` // the seat would build it free by chain anyway
}

// PendingCardHint is what destroying one of the candidates of a destroy
// choice (the Statue of Zeus, Circus Maximus) would do (the economy
// family). PendingView.cardHints lists one per candidate, in both seats'
// views, since the cities are public.
type PendingCardHint struct {
	Card        string       `json:"card"`
	PriceShift  []PriceShift `json:"priceShift"`  // the trade prices the loss would change (the destroyer's fall)
	SavesInView int          `json:"savesInView"` // coins the loss would save the destroyer on the face-up cards and its open wonders
	OnlySource  bool         `json:"onlySource"`  // the card is the victim's only fixed producer of a resource it makes
}

// OutcomeView is what one legal action of the decider would do, before
// any choice it opens (the hints design, section 4.4): Choice.outcome, in
// the decider's view only, while the game runs and one of the families it
// serves is on. Each part follows its family's switch and is zero or
// absent while that family is off. Everything is measured on a copy of the
// state with the action applied, reading only public fields, so it is the
// same on every state the decider cannot tell apart from the real one.
//
// To keep the decider's view small, every rare part is a pointer that is
// absent when it says nothing: a typical outcome is only the coins and the
// score change.
type OutcomeView struct {
	Coins      [2]Amount     `json:"coins" tstype:"Pair<Amount>"` // scoring: both seats' coins afterwards (payment, Economy, rewards, looting)
	Score      [2]Amount     `json:"score" tstype:"Pair<Amount>"` // scoring: the change of each seat's score preview total
	Pawn       *PawnMoveView `json:"pawn,omitempty"`              // military: the pawn move and the looting tokens it takes
	Distinct   *int          `json:"distinct,omitempty"`          // science: the mover's different symbols afterwards, when they change
	Opens      *PendingKind  `json:"opens,omitempty"`             // the choice the action opens for its mover (a progress token, a card to destroy or rebuild)
	Tempo      *TempoView    `json:"tempo,omitempty"`             // tempo: an extra turn gained or lost, the age's last card; absent when the turn simply passes
	Wins       *WinView      `json:"wins,omitempty"`              // threats: the action wins, at once or through the mover's own follow-up
	End        *ResultView   `json:"end,omitempty"`               // scoring: the action ends the game on points (Age III's last card)
	Gives      *[]ThreatView `json:"gives,omitempty"`             // threats: wins the opponent would have after the action that it does not have now
	Blocks     *[]ThreatView `json:"blocks,omitempty"`            // threats: wins the opponent has now that the action takes away
	Threatens  *[]ThreatView `json:"threatens,omitempty"`         // threats: wins the mover would have next turn that it does not have now; for a starter choice, the starting seat's wins at once
	Flips      *[]AffordFlip `json:"flips,omitempty"`             // economy: cards in view and open wonders whose affordability the action changes
	Prices     *[]PriceShift `json:"prices,omitempty"`            // economy: the trade prices the action changes, for either seat
	RevealRisk *Odds         `json:"revealRisk,omitempty"`        // threats: the chance that a card the action turns over lets the opponent win
}

// Amount is a whole number in a pair by seat (coins, points). It is a
// named type only so the pair carries a TypeScript name (Pair<Amount>).
type Amount int

// TempoView is what an action does to the turn order.
type TempoView struct {
	ExtraTurn      ExtraTurnState  `json:"extraTurn"`                // the mover plays again, loses the extra turn (the age's last card), or neither
	Source         ExtraTurnSource `json:"source"`                   // why a wonder grants the extra turn
	LastCard       bool            `json:"lastCard"`                 // the action takes the age's last card
	StarterChooser *int            `json:"starterChooser,omitempty"` // a last card of Age I or II: the seat that will choose who starts the next age
}

// ThreatView is a win a seat would have with one action: building or
// discarding a card, or building a wonder (with any card).
type ThreatView struct {
	Seat   int         `json:"seat"`             // the seat that would win
	Card   *string     `json:"card,omitempty"`   // the card it would take, for a card action
	Wonder *string     `json:"wonder,omitempty"` // the wonder it would build
	Kind   VictoryKind `json:"kind"`
}

// AffordFlip is a card or an open wonder that one seat can afford before
// an action and not after it, or the other way round.
type AffordFlip struct {
	Seat       int     `json:"seat"`
	Card       *string `json:"card,omitempty"`
	Wonder     *string `json:"wonder,omitempty"`
	Affordable bool    `json:"affordable"` // after the action
}

// Odds is a chance, num in den, of something that a card the action turns
// over would allow.
type Odds struct {
	Num   int `json:"num"`
	Den   int `json:"den"`
	Freed int `json:"freed"` // face-down cards the action turns over
}

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

// HintCode is a board pip: a fact about building one card, for one seat.
type HintCode string

// The hint codes.
const (
	HintCodeWin  HintCode = "win"  // building it wins at once, or through the board token its pair lets the seat take
	HintCodePair HintCode = "pair" // building it completes this seat's science pair while a board token remains
	HintCodeLoot HintCode = "loot" // building it takes a looting token from the other seat
)

// LastProducer says that no other producer of a brown or grey card's
// resource is in view.
type LastProducer string

// The last-producer states.
const (
	LastProducerCertain LastProducer = "lastCertain" // no other producer of the resource can still come into play
	LastProducerVisible LastProducer = "lastVisible" // none is in view, but unseen cards may hold one
)
