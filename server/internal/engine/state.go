package engine

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file defines the game state. It is frozen after slice S1: change it
// only through the coordinator.
//
// State is one fixed-size value made only of arrays, structs, bools and
// fixed-width integers: no pointers, slices, maps, strings, interfaces or
// platform-sized ints. So Clone is a plain copy, two states compare with ==,
// and Hash can digest the bytes. Guard tests in state_test.go enforce this.

// Seat is a player position, 0 or 1. Seat 0 is not necessarily the first
// player; State.FirstSeat says who is.
type Seat uint8

// Opp returns the other seat.
func (s Seat) Opp() Seat { return s ^ 1 }

// Structure sizes. They size the arrays of State.
const (
	NumAges          = 3
	SlotsPerAge      = 20 // cards in each age's structure
	RemovedPerAge    = 3  // cards set aside unseen from each age deck
	GuildsInPlay     = 3  // guilds shuffled into Age III
	NumBoxGuilds     = 4  // guilds left in the box
	WondersPerPlayer = 4
	NumDraftWonders  = 8 // wonders offered in the draft, two rounds of 4
	NumBoxWonders    = 4 // wonders never offered
	NumBoardTokens   = 5 // progress tokens placed on the board
	NumBoxTokens     = 5 // progress tokens left in the box
	LibraryDraw      = 3 // tokens the Great Library draws from the box
	MaxChoice        = 4 // flexible producers in the game (2 cards, 2 wonders)
)

// Phase says what kind of decision the game is waiting for.
type Phase uint8

// The phases. There is at most one open choice at a time.
const (
	PhaseWonderDraft      Phase = iota // ToMove picks a wonder
	PhasePlay                          // ToMove takes an accessible card
	PhaseChooseStarter                 // ToMove picks who starts the new age
	PhaseDestroyCard                   // Statue of Zeus or Circus Maximus: ToMove destroys an opponent card of colour Destroy
	PhaseBuildFromDiscard              // Mausoleum: ToMove builds a discarded card for free
	PhasePickBoardToken                // science pair: ToMove takes a token from the board
	PhasePickLibraryToken              // Great Library: ToMove keeps one of the 3 tokens in Library
	PhaseGameOver
	numPhases
)

var phaseNames = [numPhases]string{
	"wonderDraft", "play", "chooseStarter", "destroyCard", "buildFromDiscard",
	"pickBoardToken", "pickLibraryToken", "gameOver",
}

// String returns the wire name, for example "pickBoardToken".
func (p Phase) String() string { return enumString(phaseNames[:], uint8(p), "phase") }

// MarshalText encodes the phase as its wire name.
func (p Phase) MarshalText() ([]byte, error) { return enumText(phaseNames[:], uint8(p), "phase") }

// UnmarshalText decodes a wire name exactly.
func (p *Phase) UnmarshalText(b []byte) error {
	v, err := parseEnum(phaseNames[:], b, "phase")
	if err == nil {
		*p = Phase(v)
	}
	return err
}

// Slot is one place in the current age's structure.
type Slot struct {
	Card   catalog.CardID // 0 once the card has been taken
	FaceUp bool
	Back   catalog.Back // kept so a redacted copy still shows the back
}

// WonderSlot is one of a player's four wonders.
type WonderSlot struct {
	ID      catalog.WonderID
	Built   bool
	Removed bool           // returned to the box by the seven-wonders rule
	Tucked  catalog.CardID // the card used to build it (public: it was face up when taken)
}

// Player is one seat's city, purse and wonders.
type Player struct {
	City    catalog.CardSet // buildings constructed
	Coins   int16           // never negative
	Tokens  catalog.TokenSet
	Wonders [WondersPerPlayer]WonderSlot // in draft order

	// Derived caches, kept so payment and legality need no scan of the city.
	// They are written ONLY by city.go; CheckInvariants recomputes them from
	// City, Tokens and Wonders.
	Chains   uint32                      // bit per chain symbol owned (catalog.Chain.Bit)
	Prod     [catalog.NumResources]uint8 // fixed production of brown and grey cards (sets the opponent's prices)
	Choice   [MaxChoice]catalog.ResMask  // flexible producers: yellow cards and built wonders, Choice[:NChoice]
	NChoice  uint8                       // number of entries used in Choice
	TradeFix catalog.ResMask             // resources this player buys at 1 coin per unit
	ColorN   [catalog.NumColors]uint8    // cards per colour
	Sci      [catalog.NumSymbols]uint8   // cards (and Law) per science symbol
}

// FirstPlayerRule says how NewGame picks the first player.
type FirstPlayerRule uint8

// The first-player rules. The zero value is the standard random draw.
const (
	FirstPlayerRandom FirstPlayerRule = iota
	FirstPlayerSeat0
	FirstPlayerSeat1
	numFirstPlayerRules
)

var firstPlayerNames = [numFirstPlayerRules]string{"random", "seat0", "seat1"}

// String returns the wire name.
func (r FirstPlayerRule) String() string {
	return enumString(firstPlayerNames[:], uint8(r), "firstPlayer")
}

// MarshalText encodes the rule as its wire name.
func (r FirstPlayerRule) MarshalText() ([]byte, error) {
	return enumText(firstPlayerNames[:], uint8(r), "firstPlayer")
}

// UnmarshalText decodes a wire name exactly.
func (r *FirstPlayerRule) UnmarshalText(b []byte) error {
	v, err := parseEnum(firstPlayerNames[:], b, "firstPlayer")
	if err == nil {
		*r = FirstPlayerRule(v)
	}
	return err
}

// CenterPawnRule says who starts the next age when the conflict pawn is in
// the centre (rules decision 1). When the pawn is off centre, the player on
// whose side it stands always chooses.
type CenterPawnRule uint8

// The centre-pawn rules. The zero value follows the reference implementation.
const (
	LastActiveChooses CenterPawnRule = iota // the player who took the age's last card chooses who starts
	LastActiveStarts                        // that player starts (a literal reading of the rulebook)
	numCenterPawnRules
)

var centerPawnNames = [numCenterPawnRules]string{"lastActiveChooses", "lastActiveStarts"}

// String returns the wire name.
func (r CenterPawnRule) String() string {
	return enumString(centerPawnNames[:], uint8(r), "centerPawn")
}

// MarshalText encodes the rule as its wire name.
func (r CenterPawnRule) MarshalText() ([]byte, error) {
	return enumText(centerPawnNames[:], uint8(r), "centerPawn")
}

// UnmarshalText decodes a wire name exactly.
func (r *CenterPawnRule) UnmarshalText(b []byte) error {
	v, err := parseEnum(centerPawnNames[:], b, "centerPawn")
	if err == nil {
		*r = CenterPawnRule(v)
	}
	return err
}

// Options are the per-game rule variants. The zero value is the standard
// game: random first player, the last active player chooses on a centred
// pawn, and hidden cards are revealed at game over.
type Options struct {
	FirstPlayer FirstPlayerRule `json:"firstPlayer"`
	CenterPawn  CenterPawnRule  `json:"centerPawn"`
	HideAtEnd   bool            `json:"hideAtEnd"` // keep removed cards, box guilds, box wonders and box tokens hidden after the game
}

// VictoryKind says how the game ended.
type VictoryKind uint8

// The victory kinds.
const (
	VictoryNone     VictoryKind = iota // the game is not over
	VictoryMilitary                    // the pawn reached a capital
	VictoryScience                     // a player gathered six different science symbols
	VictoryCivilian                    // points after Age III
	VictoryForfeit                     // the other seat resigned (ActResign; not in the rulebook, see docs/rules/decisions.md)
	numVictoryKinds
)

var victoryNames = [numVictoryKinds]string{"none", "military", "science", "civilian", "forfeit"}

// String returns the wire name.
func (k VictoryKind) String() string { return enumString(victoryNames[:], uint8(k), "victory") }

// MarshalText encodes the kind as its wire name.
func (k VictoryKind) MarshalText() ([]byte, error) {
	return enumText(victoryNames[:], uint8(k), "victory")
}

// UnmarshalText decodes a wire name exactly.
func (k *VictoryKind) UnmarshalText(b []byte) error {
	v, err := parseEnum(victoryNames[:], b, "victory")
	if err == nil {
		*k = VictoryKind(v)
	}
	return err
}

// SharedVictory is the Result.Winner of a civilian game tied on points and on
// blue-card points.
const SharedVictory int8 = -1

// Score is one player's points by category, as on the score sheet.
type Score struct {
	Blue     int16 `json:"blue"`
	Green    int16 `json:"green"`
	Yellow   int16 `json:"yellow"`
	Purple   int16 `json:"purple"`
	Wonders  int16 `json:"wonders"`
	Tokens   int16 `json:"tokens"`
	Coins    int16 `json:"coins"` // one point per full three coins
	Military int16 `json:"military"`
	Total    int16 `json:"total"`
}

// Result is the outcome of a finished game. While the game runs it is the
// zero value.
type Result struct {
	Kind   VictoryKind `json:"kind"`
	Winner int8        `json:"winner"` // the winning seat (0 or 1), or SharedVictory; meaningful only when Kind != VictoryNone
	Scores [2]Score    `json:"scores"`
}

// Looting tokens on the military track, as bits of State.Looting. A bit is
// set while its token is still on the board. The track runs from -9 (seat
// 0's capital) to +9 (seat 1's capital); a token at -3 or -6 is taken from
// seat 0, a token at +3 or +6 from seat 1.
const (
	LootMinus3 uint8 = 1 << iota // 2 coins, from seat 0
	LootMinus6                   // 5 coins, from seat 0
	LootPlus3                    // 2 coins, from seat 1
	LootPlus6                    // 5 coins, from seat 1
	LootAll    = LootMinus3 | LootMinus6 | LootPlus3 | LootPlus6
)

// State is the complete game state, hidden parts included. See the top of
// this file for the shape rules.
type State struct {
	Seed      uint64 // the setup seed: hidden from clients, kept for diagnostics
	Opts      Options
	Phase     Phase
	Age       uint8  // 1 to 3; 0 during the wonder draft
	Ply       uint16 // number of actions applied so far; clients send it back to reject stale clicks
	ToMove    Seat   // who must decide now
	Active    Seat   // whose turn it is; differs from ToMove only in PhaseChooseStarter. At the end of an age it is the last active player.
	FirstSeat Seat   // the first player: starts draft round 1 and Age I
	ExtraTurn bool   // Active plays again once this turn is finished

	Destroy catalog.Color                // the colour to destroy; set only in PhaseDestroyCard
	Library [LibraryDraw]catalog.TokenID // the Great Library offer; set only in PhasePickLibraryToken

	Players [2]Player

	Slots     [SlotsPerAge]Slot                    // the current age's structure, in layout order (layout.go)
	Deals     [NumAges][SlotsPerAge]catalog.CardID // each age's deal in layout order; a row is zeroed once moved into Slots
	Removed   [NumAges][RemovedPerAge]catalog.CardID
	BoxGuilds [NumBoxGuilds]catalog.CardID
	Discard   catalog.CardSet // the discard pile; its order does not matter to the rules
	OutOfPlay catalog.CardSet // scenario-only sink for cards a test scenario takes out; empty in real games

	DraftOrder [NumDraftWonders]catalog.WonderID // [0:4] round 1, [4:8] round 2
	DraftTaken uint8                             // bit i set when DraftOrder[i] has been taken
	BoxWonders [NumBoxWonders]catalog.WonderID

	BoardTokens catalog.TokenSet
	BoxTokens   [NumBoxTokens]catalog.TokenID // the box as an ordered draw pile (front first); 0 marks an empty place

	Military     int8  // conflict pawn, -9 to 9; positive means toward seat 1's capital (seat 0 leads)
	Looting      uint8 // Loot* bits of the looting tokens still on the track
	WondersBuilt uint8 // wonders built by both players together, at most 7
	Result       Result

	// The derived counters below record what the course of the game
	// implies but the rest of the state no longer shows. Both are public
	// and follow from the action keys alone, so a replay always rebuilds
	// them. They stay the last fields of State: Hash (replay.go) digests
	// every byte before them and none of them, so adding them changed the
	// hash of no recorded game (a test in hints_test.go pins this).
	Shields      [2]int8 // by seat, the shields the seat has placed: every shield of its buildings and wonders, Strategy's included, counted by movePawn
	PlayingAgain bool    // the current turn was granted by an extra turn; set by finishTurn
}

// Clone returns an independent copy. The state holds no references, so this
// is a plain value copy.
func (s *State) Clone() State { return *s }

// Over reports whether the game has ended.
func (s *State) Over() bool { return s.Phase == PhaseGameOver }

// enumString, enumText and parseEnum implement the text forms of the small
// enums above from a table of names indexed by value.
func enumString(names []string, v uint8, kind string) string {
	if int(v) < len(names) {
		return names[v]
	}
	return fmt.Sprintf("%s#%d", kind, v)
}

func enumText(names []string, v uint8, kind string) ([]byte, error) {
	if int(v) < len(names) {
		return []byte(names[v]), nil
	}
	return nil, fmt.Errorf("engine: invalid %s %d", kind, v)
}

func parseEnum(names []string, b []byte, kind string) (uint8, error) {
	for i, n := range names {
		if string(b) == n {
			return uint8(i), nil
		}
	}
	if len(b) > 64 {
		b = b[:64]
	}
	return 0, fmt.Errorf("engine: unknown %s %q", kind, b)
}
