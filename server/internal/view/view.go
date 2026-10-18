package view

// View is one viewer's view of a game at one moment: everything that viewer
// may see, and nothing else. The engine builds it (engine.ViewFor) for a
// seat or for a spectator after every applied action.
//
// Seats are 0 and 1. Cards, wonders and tokens are slugs. Lists that follow
// no layout are in ascending id order.
type View struct {
	Schema      int    `json:"schema"`           // SchemaVersion
	CatalogHash string `json:"catalogHash"`      // fingerprint of the server's card data; a client whose presentation catalog has another hash is stale
	Ply         int    `json:"ply"`              // actions applied so far; an action is sent with the ply it was chosen at
	Viewer      *int   `json:"viewer,omitempty"` // the seat this view is for; absent for a spectator
	Phase       Phase  `json:"phase"`
	Age         int    `json:"age"`              // 1 to 3; 0 during the wonder draft
	ToMove      *int   `json:"toMove,omitempty"` // the seat that must decide now; absent once the game is over
	Active      int    `json:"active"`           // whose turn it is; differs from toMove only while a new age's starter is chosen
	FirstPlayer int    `json:"firstPlayer"`      // the seat that started the draft and Age I
	ExtraTurn   bool   `json:"extraTurn"`        // the active seat plays again once this turn is over
	// PlayingAgain says that the current turn was granted by an extra
	// turn: the active seat is playing again. It is false while a new
	// age's starter is chosen and once the game is over.
	PlayingAgain bool `json:"playingAgain"`

	Draft        *DraftView        `json:"draft,omitempty"`                   // only during the wonder draft
	Structure    []SlotView        `json:"structure"`                         // the current age's 20 slots in layout order; empty during the draft
	Players      [2]PlayerView     `json:"players" tstype:"Pair<PlayerView>"` // by seat
	Military     MilitaryView      `json:"military"`
	CardTooltips []CardTooltipView `json:"cardTooltips"` // public quotes and locations of already visible cards

	BoardTokens   []string `json:"boardTokens"`   // progress tokens still on the board
	BoxTokenCount int      `json:"boxTokenCount"` // progress tokens in the box (never their names before the end)
	RemovedCount  int      `json:"removedCount"`  // cards set aside unseen from the current age's deck
	Discard       []string `json:"discard"`       // public discard pile; room frames order arrivals oldest first (scenario initial batch uses canonical order)
	WondersBuilt  int      `json:"wondersBuilt"`  // by both players together; at seven the last unbuilt wonder leaves the game

	Pending  *PendingView     `json:"pending,omitempty"` // an open starter choice or a choice that a card or wonder opened
	Choices  []Choice         `json:"choices"`           // the legal actions, only in the view of the seat to move
	Disabled []DisabledChoice `json:"disabled"`          // play actions the seat to move cannot take now, with the reason

	ScorePreview [2]ScoreView `json:"scorePreview" tstype:"Pair<ScoreView>"` // both seats' points if the game ended now
	Result       *ResultView  `json:"result,omitempty"`                      // once the game is over
	Revealed     *RevealView  `json:"revealed,omitempty"`                    // the hidden cards, wonders and tokens, once the game is over (unless the game hides them)

	Hints *HintsView `json:"hints,omitempty"` // the board-level hints (hints.go); absent once the game is over and when the view was built without hint settings

	Review *ReviewView `json:"review,omitempty"` // the move review of the viewer's own last play action (hints.go), added by the room while its advisor family is on; never in a spectator's view
}

// DraftView is the wonder draft in progress.
type DraftView struct {
	Round int              `json:"round"` // 1 or 2
	Offer []DraftOfferView `json:"offer"` // the round's four wonders, in the order they were dealt; the second round's stay hidden until the first is over
	Order []int            `json:"order"` // the seats making the round's four picks, in pick order (the last pick is automatic)
}

// DraftOfferView is one wonder of the current draft round.
type DraftOfferView struct {
	Wonder  string `json:"wonder"`
	TakenBy *int   `json:"takenBy,omitempty"` // the seat that took it
}

// SlotView is one place of the current age's structure.
type SlotView struct {
	Index      int           `json:"index"` // 0 to 19, in layout order (row by row from the top, left to right)
	Row        int           `json:"row"`   // 1 at the top
	Col        int           `json:"col"`   // in half-card widths, 1 at the left
	State      SlotState     `json:"state"`
	Back       *Back         `json:"back,omitempty"`                           // the card's back; absent for an empty slot
	Card       *string       `json:"card,omitempty"`                           // only when face up
	Accessible bool          `json:"accessible"`                               // holds a card and no card covers it
	CoveredBy  []int         `json:"coveredBy"`                                // the slots still covering this one
	Costs      *[2]CostView  `json:"costs,omitempty" tstype:"Pair<CostView>"`  // both seats' cost of building the card now, by seat, as if it were accessible; for every face-up card, covered or not
	Hints      *[2]CardHints `json:"hints,omitempty" tstype:"Pair<CardHints>"` // what building the card would do for each seat, by seat (hints.go); for every face-up card while the game runs and a per-card hint family is on
}

// ResourceCounts holds a number per resource.
type ResourceCounts struct {
	Wood    int `json:"wood"`
	Clay    int `json:"clay"`
	Stone   int `json:"stone"`
	Glass   int `json:"glass"`
	Papyrus int `json:"papyrus"`
}

// CostView is the cheapest way for one seat to pay for a building or a
// wonder now, with its breakdown. The engine always applies this payment;
// the client only shows it. A revive choice instead carries its actual
// free payment: zero total, no chain, and the unchanged payer purse.
//
// For an ordinary build, unless it is chain-free, every resource unit of the printed cost is
// covered once: by own production, by a flexible producer, by a discount or
// by trade.
type CostView struct {
	Affordable bool `json:"affordable"`
	// CoinsAfter is the payer's purse immediately after payment, before
	// Urbanism or any building/wonder effect. Absent when unaffordable.
	CoinsAfter    *int           `json:"coinsAfter,omitempty"`
	Shortfall     int            `json:"shortfall"` // coins missing when not affordable, else 0
	ChainFree     bool           `json:"chainFree"` // free through a chain symbol: nothing else applies
	ChainFrom     *string        `json:"chainFrom,omitempty"`
	PrintedCoins  int            `json:"printedCoins"` // paid to the bank
	Own           ResourceCounts `json:"own"`          // units covered by the seat's own brown and grey cards
	Producers     []ProducerUse  `json:"producers"`    // units covered by flexible producers
	Discount      *DiscountView  `json:"discount,omitempty"`
	Trade         []TradeView    `json:"trade"`         // units bought, one entry per resource bought
	TradeTotal    int            `json:"tradeTotal"`    // coins paid for the bought units
	Total         int            `json:"total"`         // every coin the seat spends: printedCoins + tradeTotal
	ToOpponent    int            `json:"toOpponent"`    // the part of total the opponent receives (Economy)
	UrbanismBonus int            `json:"urbanismBonus"` // coins the seat receives for a chain-free build (Urbanism)
}

// ProducerUse is one flexible producer (a yellow card or a built wonder)
// covering one unit of a payment.
type ProducerUse struct {
	Card     *string  `json:"card,omitempty"`
	Wonder   *string  `json:"wonder,omitempty"`
	Resource Resource `json:"resource"`
}

// DiscountView is a progress token's discount on a payment. It is present
// whenever the token applies to the item, even when it had nothing left to
// waive.
type DiscountView struct {
	Token string         `json:"token"`
	Units ResourceCounts `json:"units"` // units waived
}

// TradeView is the units of one resource a payment buys.
type TradeView struct {
	Resource  Resource `json:"resource"`
	Units     int      `json:"units"`
	UnitPrice int      `json:"unitPrice"`
	Fixed     bool     `json:"fixed"` // the seat buys this resource at the fixed price of 1
	Coins     int      `json:"coins"` // units times unitPrice
}

// PlayerView is one seat's city, purse and wonders.
type PlayerView struct {
	Seat         int            `json:"seat"`
	Coins        int            `json:"coins"`
	City         []string       `json:"city"`   // buildings constructed
	Tokens       []string       `json:"tokens"` // progress tokens owned
	Wonders      []WonderView   `json:"wonders"`
	Production   ProductionView `json:"production"`
	TradePrices  ResourceCounts `json:"tradePrices"` // what this seat pays per unit bought now
	Science      ScienceView    `json:"science"`
	Chains       []Chain        `json:"chains"`       // chain symbols this seat's city shows
	DiscardValue int            `json:"discardValue"` // coins this seat receives for discarding a card now
	Shields      int            `json:"shields"`      // shields this seat has placed: every shield of its buildings and wonders, Strategy's included
	// Hints are the board-level hints about this seat (hints.go), while
	// the game runs and the room asked for hints.
	Hints *PlayerHintsView `json:"hints,omitempty"`
}

// WonderView is one of a seat's wonders, in draft order.
type WonderView struct {
	Wonder  string    `json:"wonder"`
	Built   bool      `json:"built"`
	Removed bool      `json:"removed"`          // left the game unbuilt when the seventh wonder was built
	Tucked  *string   `json:"tucked,omitempty"` // the card under a built wonder
	Cost    *CostView `json:"cost,omitempty"`   // the owner's cost, while the wonder can still be built
	// Effect is what building it would do now (hints.go), while it can
	// still be built and the wonders hints are on.
	Effect *WonderEffectView `json:"effect,omitempty"`
}

// ProductionView is what a seat produces every turn.
type ProductionView struct {
	Fixed     ResourceCounts `json:"fixed"`     // brown and grey cards; this also sets the opponent's trade prices
	Producers []Producer     `json:"producers"` // flexible producers: one unit of one of their resources per payment
	TradeFix  []Resource     `json:"tradeFix"`  // resources this seat buys at 1 coin per unit
}

// Producer is a flexible producer: a yellow card or a built wonder that
// makes one unit of any of its resources.
type Producer struct {
	Card      *string    `json:"card,omitempty"`
	Wonder    *string    `json:"wonder,omitempty"`
	Resources []Resource `json:"resources"`
}

// ScienceView is a seat's science symbols.
type ScienceView struct {
	Symbols  []SymbolCount `json:"symbols"`  // each symbol the seat owns, in symbol order; a count of 2 is a pair
	Distinct int           `json:"distinct"` // different symbols; six win the game
}

// SymbolCount is how many times a seat owns one science symbol.
type SymbolCount struct {
	Symbol ScienceSymbol `json:"symbol"`
	Count  int           `json:"count"`
}

// MilitaryView is the military track.
type MilitaryView struct {
	Position int           `json:"position"` // the conflict pawn, -9 to 9: positive is toward seat 1's capital (seat 0 leads), negative toward seat 0's
	Looting  []LootingView `json:"looting"`  // the four looting tokens, in track order from -6 to 6
}

// LootingView is one looting token of the track.
type LootingView struct {
	At      int  `json:"at"`      // its space: -6, -3, 3 or 6
	Seat    int  `json:"seat"`    // the seat that loses coins when the pawn reaches it
	Value   int  `json:"value"`   // coins lost
	Present bool `json:"present"` // still on the track
}

// PendingView is an open choice that is not a plain turn.
type PendingView struct {
	Kind         PendingKind        `json:"kind"`
	Seat         int                `json:"seat"`                // who decides
	Color        *Color             `json:"color,omitempty"`     // destroyCard: the colour of the card to destroy
	Cards        []string           `json:"cards"`               // destroyCard: the opponent's cards that may be destroyed; buildFromDiscard: the discard pile
	Tokens       []string           `json:"tokens"`              // pickBoardToken: the board tokens; pickLibraryToken: the offer, only in the decider's view
	HiddenTokens int                `json:"hiddenTokens"`        // pickLibraryToken: the tokens on offer that this viewer may not see
	CardHints    *[]PendingCardHint `json:"cardHints,omitempty"` // destroyCard, while the economy hints are on: what losing each candidate would change (hints.go)

	// TokenValues is, during a Great Library offer and only in the view of
	// the seat choosing from it, what each token on offer would do for it
	// (hints.go), while the tokens hints are on.
	TokenValues *[]TokenValueView `json:"tokenValues,omitempty"`
}

// Choice is one legal action of the seat to move. Only the fields its kind
// uses are set.
type Choice struct {
	Key     string     `json:"key"` // send this to take the action
	Kind    ChoiceKind `json:"kind"`
	Card    *string    `json:"card,omitempty"`    // construct, discard, wonder (the card tucked), destroy, revive
	Wonder  *string    `json:"wonder,omitempty"`  // draft, wonder
	Token   *string    `json:"token,omitempty"`   // token
	Starter *int       `json:"starter,omitempty"` // starter: the seat that will start the age
	Cost    *CostView  `json:"cost,omitempty"`    // construct, wonder and revive: what the seat pays
	Gain    *int       `json:"gain,omitempty"`    // discard: coins received
	// Outcome is what the action would do (hints.go), only in the
	// decider's view, while a family that needs it is on.
	Outcome *OutcomeView `json:"outcome,omitempty"`
}

// DisabledChoice is a play action the seat to move cannot take now: a card
// it cannot afford to build, or a wonder it cannot build with that card.
type DisabledChoice struct {
	Key    string     `json:"key"`
	Kind   ChoiceKind `json:"kind"`
	Card   *string    `json:"card,omitempty"`
	Wonder *string    `json:"wonder,omitempty"`
	Reason Reason     `json:"reason"`
	Cost   *CostView  `json:"cost,omitempty"` // what it would cost
	Wins   *WinView   `json:"wins,omitempty"` // threats: a supremacy the action would win if the seat had the coins (hints.go)
}

// ScoreView is one seat's points by category, as on the score sheet.
type ScoreView struct {
	Blue     int `json:"blue"`
	Green    int `json:"green"`
	Yellow   int `json:"yellow"`
	Purple   int `json:"purple"`
	Wonders  int `json:"wonders"`
	Tokens   int `json:"tokens"`
	Coins    int `json:"coins"` // one point per full three coins
	Military int `json:"military"`
	Total    int `json:"total"`
}

// ResultView is the outcome of a finished game.
type ResultView struct {
	Victory VictoryKind  `json:"victory"`
	Winner  *int         `json:"winner,omitempty"`                // absent for a shared victory
	Shared  bool         `json:"shared"`                          // a civilian tie on points and on blue points
	Scores  [2]ScoreView `json:"scores" tstype:"Pair<ScoreView>"` // the final score sheet, by seat
}

// RevealView lists what stayed hidden during the game, for review after it.
// The seed is never part of a view.
type RevealView struct {
	Removed    []AgeCards `json:"removed"`    // the cards set aside unseen from each age's deck
	Undealt    []AgeCards `json:"undealt"`    // the deals of ages never laid out, in layout order
	FaceDown   []SlotCard `json:"faceDown"`   // the cards still face down in the structure
	BoxGuilds  []string   `json:"boxGuilds"`  // the guilds that never entered the game
	BoxWonders []string   `json:"boxWonders"` // the wonders never offered in the draft
	BoxTokens  []string   `json:"boxTokens"`  // the progress tokens in the box, in draw order
}

// AgeCards is a list of cards of one age.
type AgeCards struct {
	Age   int      `json:"age"`
	Cards []string `json:"cards"`
}

// SlotCard is the card in one structure slot.
type SlotCard struct {
	Slot int    `json:"slot"`
	Card string `json:"card"`
}
