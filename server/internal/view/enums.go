package view

// The small vocabularies of the view. Each is a string type with one
// constant per value, so tygo turns it into a TypeScript union of string
// literals. The values are the engine's and the catalog's own wire names;
// a test in the engine checks that the two lists agree exactly, in both
// directions.

// Phase is what the game is waiting for.
type Phase string

// The phases.
const (
	PhaseWonderDraft      Phase = "wonderDraft"
	PhasePlay             Phase = "play"
	PhaseChooseStarter    Phase = "chooseStarter"
	PhaseDestroyCard      Phase = "destroyCard"
	PhaseBuildFromDiscard Phase = "buildFromDiscard"
	PhasePickBoardToken   Phase = "pickBoardToken"
	PhasePickLibraryToken Phase = "pickLibraryToken"
	PhaseGameOver         Phase = "gameOver"
)

// PendingKind is the kind of an open choice that is not a plain turn: the
// starter of a new age, or a choice that a card or a wonder opened. Its
// values are the names of the matching phases.
type PendingKind string

// The pending kinds.
const (
	PendingKindChooseStarter    PendingKind = "chooseStarter"
	PendingKindDestroyCard      PendingKind = "destroyCard"
	PendingKindBuildFromDiscard PendingKind = "buildFromDiscard"
	PendingKindPickBoardToken   PendingKind = "pickBoardToken"
	PendingKindPickLibraryToken PendingKind = "pickLibraryToken"
)

// ChoiceKind is the kind of an action: the first part of its key.
type ChoiceKind string

// The choice kinds.
const (
	ChoiceKindDraft     ChoiceKind = "draft"
	ChoiceKindConstruct ChoiceKind = "construct"
	ChoiceKindDiscard   ChoiceKind = "discard"
	ChoiceKindWonder    ChoiceKind = "wonder"
	ChoiceKindDestroy   ChoiceKind = "destroy"
	ChoiceKindRevive    ChoiceKind = "revive"
	ChoiceKindToken     ChoiceKind = "token"
	ChoiceKindStarter   ChoiceKind = "starter"
)

// Reason says why an action is not available. The server's reject codes use
// the same names.
type Reason string

// The reasons.
const (
	ReasonGameOver          Reason = "gameOver"          // the game has ended
	ReasonWrongPhase        Reason = "wrongPhase"        // the game is waiting for another kind of action
	ReasonMalformed         Reason = "malformed"         // the action is not well formed
	ReasonNotInDraft        Reason = "notInDraft"        // the wonder is not on offer in this draft round
	ReasonNotAccessible     Reason = "notAccessible"     // the card cannot be taken from the structure now
	ReasonWonderUnavailable Reason = "wonderUnavailable" // not one of the mover's open wonders, or seven wonders are built
	ReasonUnaffordable      Reason = "unaffordable"      // the mover cannot pay the cost
	ReasonNotTarget         Reason = "notTarget"         // not an opponent card of the colour to destroy
	ReasonNotInDiscard      Reason = "notInDiscard"      // not in the discard pile
	ReasonTokenUnavailable  Reason = "tokenUnavailable"  // the token is not on offer
)

// SlotState says what a structure slot shows.
type SlotState string

// The slot states.
const (
	SlotStateFaceUp   SlotState = "faceUp"
	SlotStateFaceDown SlotState = "faceDown"
	SlotStateEmpty    SlotState = "empty"
)

// Back is the printed back of a card, the only thing a face-down card shows.
type Back string

// The card backs. Guilds have their own back.
const (
	BackAge1  Back = "age1"
	BackAge2  Back = "age2"
	BackAge3  Back = "age3"
	BackGuild Back = "guild"
)

// Resource is one of the five resources.
type Resource string

// The resources, in the order the engine uses everywhere.
const (
	ResourceWood    Resource = "wood"
	ResourceClay    Resource = "clay"
	ResourceStone   Resource = "stone"
	ResourceGlass   Resource = "glass"
	ResourcePapyrus Resource = "papyrus"
)

// Color is a building's colour.
type Color string

// The building colours.
const (
	ColorBrown  Color = "brown"
	ColorGrey   Color = "grey"
	ColorRed    Color = "red"
	ColorGreen  Color = "green"
	ColorYellow Color = "yellow"
	ColorBlue   Color = "blue"
	ColorPurple Color = "purple"
)

// ScienceSymbol is a science symbol. The scales are only on the Law token.
type ScienceSymbol string

// The science symbols.
const (
	ScienceSymbolGlobe     ScienceSymbol = "globe"
	ScienceSymbolScales    ScienceSymbol = "scales"
	ScienceSymbolSundial   ScienceSymbol = "sundial"
	ScienceSymbolMortar    ScienceSymbol = "mortar"
	ScienceSymbolSetSquare ScienceSymbol = "set-square"
	ScienceSymbolQuill     ScienceSymbol = "quill"
	ScienceSymbolWheel     ScienceSymbol = "wheel"
)

// Chain is a chain symbol: a card that shows it makes one later card free.
type Chain string

// The chain symbols.
const (
	ChainHorseshoe    Chain = "horseshoe"
	ChainSword        Chain = "sword"
	ChainTower        Chain = "tower"
	ChainBook         Chain = "book"
	ChainGear         Chain = "gear"
	ChainAmphora      Chain = "amphora"
	ChainMask         Chain = "mask"
	ChainMoon         Chain = "moon"
	ChainDrop         Chain = "drop"
	ChainTarget       Chain = "target"
	ChainHelmet       Chain = "helmet"
	ChainLyre         Chain = "lyre"
	ChainLamp         Chain = "lamp"
	ChainBarrel       Chain = "barrel"
	ChainSun          Chain = "sun"
	ChainColumn       Chain = "column"
	ChainTempleFacade Chain = "temple-facade"
)

// VictoryKind says how a finished game was won.
type VictoryKind string

// The victory kinds.
const (
	VictoryKindMilitary VictoryKind = "military" // the pawn reached a capital
	VictoryKindScience  VictoryKind = "science"  // six different science symbols
	VictoryKindCivilian VictoryKind = "civilian" // points after Age III
	VictoryKindForfeit  VictoryKind = "forfeit"  // the other player resigned
)

// EventKind is what a game event reports. GameEvent lists the fields each
// kind uses.
type EventKind string

// The event kinds.
const (
	EventKindDraft         EventKind = "draft"
	EventKindDraftRound    EventKind = "draftRound"
	EventKindAgeStart      EventKind = "ageStart"
	EventKindBuild         EventKind = "build"
	EventKindDiscard       EventKind = "discard"
	EventKindWonder        EventKind = "wonder"
	EventKindPay           EventKind = "pay"
	EventKindReveal        EventKind = "reveal"
	EventKindCoins         EventKind = "coins"
	EventKindPawn          EventKind = "pawn"
	EventKindLoot          EventKind = "loot"
	EventKindPair          EventKind = "pair"
	EventKindToken         EventKind = "token"
	EventKindLibraryDraw   EventKind = "libraryDraw"
	EventKindDestroy       EventKind = "destroy"
	EventKindRebuild       EventKind = "rebuild"
	EventKindWonderRemoved EventKind = "wonderRemoved"
	EventKindEffectSkipped EventKind = "effectSkipped"
	EventKindExtraTurn     EventKind = "extraTurn"
	EventKindExtraTurnLost EventKind = "extraTurnLost"
	EventKindStarterChosen EventKind = "starterChosen"
	EventKindGameOver      EventKind = "gameOver"
	EventKindResign        EventKind = "resign"
)
