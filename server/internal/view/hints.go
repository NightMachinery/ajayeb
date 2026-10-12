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

	// The board-level hints (M10.HS4), each under its family's switch.

	WondersLeft   int                `json:"wondersLeft"`             // wonders: how many more wonders the game allows (7 minus those built); 1 starts the race for the last one
	LastCardTo    *int               `json:"lastCardTo,omitempty"`    // tempo: the seat that takes the age's last card if nobody plays again; absent while a new age's starter is chosen and in the draft
	NextStarter   *NextStarterView   `json:"nextStarter,omitempty"`   // tempo: who decides the next age's starter, as the pawn stands (Ages I and II)
	PreviewWinner *PreviewWinnerView `json:"previewWinner,omitempty"` // scoring: who would win on points if the game ended now
	Pairs         []SciencePairView  `json:"pairs"`                   // science: the six card symbols, each with its two cards and where they are
	Looting       []LootHint         `json:"looting"`                 // military: the looting tokens still on the track, in track order, with what each would take now
	Tokens        []TokenValueView   `json:"tokens"`                  // tokens: each board token's value now, for each seat (by token, then seat)
	BoxTokens     []string           `json:"boxTokens"`               // tokens: the tokens in the box (and in an open Great Library offer), named by elimination, while a seat's Great Library is open; their order stays secret
	GuildPreview  []GuildPreviewView `json:"guildPreview"`            // counting, Age III: what each guild not yet seen would be worth to each seat now
	Unseen        []UnseenView       `json:"unseen"`                  // counting: for each back face down in the structure, the cards that may lie under it
}

// NextStarterView is who will choose the next age's starter, as the pawn
// stands now (ages-scoring.md "Who starts the next age").
type NextStarterView struct {
	Chooser          *int `json:"chooser,omitempty"` // the weaker seat (the pawn is on its side); absent with a centred pawn
	ByLastCard       bool `json:"byLastCard"`        // the pawn is centred: the player who takes the age's last card decides
	LastActiveStarts bool `json:"lastActiveStarts"`  // the room option: with a centred pawn that player simply starts
}

// PreviewWinnerView is who would win on points if the game ended now, and
// how the tie-break decides it.
type PreviewWinnerView struct {
	Winner *int     `json:"winner,omitempty"` // absent for a shared victory
	By     TieBreak `json:"by"`
	Margin int      `json:"margin"` // the lead in points (in blue points when the totals are tied)
}

// SciencePairView is one card symbol and the two green cards that carry it.
type SciencePairView struct {
	Symbol ScienceSymbol  `json:"symbol"`
	Cards  []PairCardView `json:"cards"` // in ascending card order
}

// PairCardView is one card of a science pair and its public place.
type PairCardView struct {
	Card   string      `json:"card"`
	Place  PublicPlace `json:"place"`
	Seat   *int        `json:"seat,omitempty"`   // city and tucked: whose
	Slot   *int        `json:"slot,omitempty"`   // faceUp: the structure slot
	InPlay *Odds       `json:"inPlay,omitempty"` // unseen: the chance it is dealt (num in den)
}

// LootHint is one looting token still on the track.
type LootHint struct {
	At        int `json:"at"`        // its space: -6, -3, 3 or 6
	WouldLose int `json:"wouldLose"` // coins its victim would lose if the pawn entered it now (no more than it has)
}

// TokenValueView is what taking one progress token would do for one seat
// now (the tokens family).
type TokenValueView struct {
	Token   string `json:"token"`
	Seat    int    `json:"seat"`
	Coins   int    `json:"coins"`   // coins gained at once
	VP      int    `json:"vp"`      // change of the seat's score preview
	Wins    bool   `json:"wins"`    // taking it wins the game (Law as the sixth symbol)
	Savings int    `json:"savings"` // Masonry, Architecture: coins it saves on the cards in view and the seat's open wonders; Economy: the trade coins the other seat would pay on the cards it can take now
	Count   int    `json:"count"`   // Urbanism: chains whose target may still come; Theology: open wonders that would gain play again; Mathematics: tokens the seat would own
}

// GuildPreviewView is what building one guild would bring each seat now:
// the coins on construction and the change of each seat's score preview.
type GuildPreviewView struct {
	Guild string    `json:"guild"`
	Coins [2]Amount `json:"coins" tstype:"Pair<Amount>"` // by builder
	VP    [2]Amount `json:"vp" tstype:"Pair<Amount>"`    // by builder: that seat's change of points
}

// UnseenView is one back face down in the structure and the cards that may
// lie under it: the cards of its kind in no public place.
type UnseenView struct {
	Back     Back     `json:"back"`
	Cards    []string `json:"cards"`    // the pool, in ascending card order
	FaceDown int      `json:"faceDown"` // face-down slots with this back
	SetAside int      `json:"setAside"` // cards of the pool that are not in the structure (set aside, or the guilds not in the game)
}

// PlayerHintsView holds the board-level hints about one seat
// (PlayerView.hints): its science, its capital, its chains and guilds, and
// the wins it would have if it were its turn now. Each part is zero or
// empty while its family is off.
type PlayerHintsView struct {
	CoinsToNextVp int               `json:"coinsToNextVp"`     // scoring: coins still missing for the next point from coins
	Symbols       []SymbolHint      `json:"symbols"`           // science: the seven symbols, in symbol order
	Need          int               `json:"need"`              // science: different symbols still needed for a scientific supremacy
	Reachable     bool              `json:"reachable"`         // science: enough missing symbols may still come to make up the need
	LawWins       bool              `json:"lawWins"`           // science: Law on the board would be this seat's sixth symbol, and it has a pair that may still complete
	ChainTargets  []ChainTargetView `json:"chainTargets"`      // counting: the card each chain symbol of the city frees, and where it is
	GuildPoints   []GuildPointsView `json:"guildPoints"`       // scoring: the points each owned guild is worth now
	ToCapital     int               `json:"toCapital"`         // military: shields the other seat still needs to take this seat's capital
	CapitalSafe   bool              `json:"capitalSafe"`       // military: the other seat can no longer gather that many shields
	Threats       *[]ThreatView     `json:"threats,omitempty"` // threats: the wins this seat would have with one action if it were its turn now; absent in a phase that does not compute them (a pending choice), where the last list still holds
}

// SymbolHint is where a seat stands on one science symbol.
type SymbolHint struct {
	Symbol    ScienceSymbol `json:"symbol"`
	State     SymbolState   `json:"state"`
	OpenCards int           `json:"openCards"` // cards of it that may still come to this seat (for the scales: 1 while Law may still come)
}

// ChainTargetView is the card one chain symbol makes free, and where it is.
type ChainTargetView struct {
	Chain  Chain       `json:"chain"`
	Target string      `json:"target"`
	Place  PublicPlace `json:"place"`
	Seat   *int        `json:"seat,omitempty"`   // city and tucked: whose
	InPlay *Odds       `json:"inPlay,omitempty"` // unseen: the chance it is dealt
	Worth  *int        `json:"worth,omitempty"`  // while it may still come to the seat: what it would cost without the chain today
}

// GuildPointsView is one owned guild's points now.
type GuildPointsView struct {
	Card       string `json:"card"`
	VP         int    `json:"vp"`
	CountsSeat int    `json:"countsSeat"` // the city it counts now (the one with more of what it counts)
}

// WonderEffectView is what building one open wonder would do now, for its
// owner (WonderView.effect, the wonders family): the effects up to its
// special choice, which it only names.
type WonderEffectView struct {
	Paid      bool            `json:"paid"`                        // the owner can afford it now, and the preview paid
	Coins     [2]Amount       `json:"coins" tstype:"Pair<Amount>"` // change of both seats' coins (payment, rewards, looting, the Appian Way)
	Score     [2]Amount       `json:"score" tstype:"Pair<Amount>"` // change of both seats' score preview totals
	Pawn      *PawnMoveView   `json:"pawn,omitempty"`              // military: the pawn move and the looting
	Wins      *WinView        `json:"wins,omitempty"`              // threats: building it wins, at once or through its rebuild
	ExtraTurn ExtraTurnSource `json:"extraTurn"`                   // play again: by the wonder itself, by Theology, or none
	Removes   *string         `json:"removes,omitempty"`           // the wonder the seven-wonder rule would send back to the box
	Opens     *PendingKind    `json:"opens,omitempty"`             // the special choice it opens
	Pool      int             `json:"pool"`                        // that choice's size: destroy candidates, discarded cards, or tokens the Great Library would draw
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

// SymbolState is where a seat stands on one of the seven science symbols.
type SymbolState string

// The symbol states.
const (
	SymbolStatePaired SymbolState = "paired" // both cards of the symbol
	SymbolStateLive   SymbolState = "live"   // one card, and the other may still come to the seat
	SymbolStateOwned  SymbolState = "owned"  // one card (or the scales), and the other can never come
	SymbolStateOpen   SymbolState = "open"   // none yet, and one may still come
	SymbolStateDead   SymbolState = "dead"   // none, and none can come
)

// TieBreak says how the score preview's leader is decided.
type TieBreak string

// The tie-breaks.
const (
	TieBreakTotal  TieBreak = "total"  // the higher total
	TieBreakBlue   TieBreak = "blue"   // equal totals: the more blue points
	TieBreakShared TieBreak = "shared" // equal totals and blue points: a shared victory
)

// LastProducer says that no other producer of a brown or grey card's
// resource is in view.
type LastProducer string

// The last-producer states.
const (
	LastProducerCertain LastProducer = "lastCertain" // no other producer of the resource can still come into play
	LastProducerVisible LastProducer = "lastVisible" // none is in view, but unseen cards may hold one
)

// ReviewView is the move review (the advisor family): whether a clearly
// better move than the one a seat just made existed in that position, and
// why. The room computes it from the mover's own observation of the
// position before the move (never from the real state), so it uses only
// what the mover knew: it names only the move made, the better one and a
// card or wonder that was in view, never a card the move turned over.
//
// Only the mover's own view carries it (View.review), from that move until
// the seat's next play action, while the room's advisor family is on;
// spectators and the opponent never get it. It also stays in the view at
// the end of the game, for the final summary.
type ReviewView struct {
	Ply    int          `json:"ply"`              // the ply of the reviewed move: the seat's last play action (build, discard or wonder)
	Played string       `json:"played"`           // that move's action key
	Better *string      `json:"better,omitempty"` // a clearly better action's key; absent when none was found
	Reason ReviewReason `json:"reason"`           // why it is better; none without better
	Gap    int          `json:"gap"`              // how much better, in points (the bot's victory-point equivalents, rounded); 0 without better
	// The reason's details, each only for its reason.
	Threat *ThreatView    `json:"threat,omitempty"` // stopsWin: the opponent's win the move left open (a card or a wonder in view before the move)
	Symbol *ScienceSymbol `json:"symbol,omitempty"` // pair and deniesPair: the science symbol
	Coins  *int           `json:"coins,omitempty"`  // loot: the coins the better move's looting would have taken
}

// ReviewReason is why a move review's better move is better, from the
// facts that tell the two moves apart, in this order of priority.
type ReviewReason string

// The review reasons.
const (
	ReviewReasonNone       ReviewReason = "none"       // no clearly better move was found
	ReviewReasonWins       ReviewReason = "wins"       // the better move wins the game at once (or through the mover's own follow-up)
	ReviewReasonStopsWin   ReviewReason = "stopsWin"   // the move left the opponent a win with one action; the better move does not
	ReviewReasonPair       ReviewReason = "pair"       // the better move completes a science pair of the mover's
	ReviewReasonDeniesPair ReviewReason = "deniesPair" // the better move takes the card that completes the opponent's pair
	ReviewReasonLoot       ReviewReason = "loot"       // the better move's pawn takes more coins from the opponent
	ReviewReasonPoints     ReviewReason = "points"     // no single fact: the bot's evaluation rates it gap points higher
)
