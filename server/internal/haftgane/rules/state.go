package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Frozen after M15.haftgane-core-types. Changes require the single writer.
type Phase uint8

const (
	PhaseSides Phase = iota
	PhaseTurn
	PhaseDiscardBuild
	PhaseLastCard
	PhaseOver
	NumPhases
)

var phaseNames = [NumPhases]string{"sides", "turn", "discard-build", "last-card", "over"}

func (p Phase) String() string {
	if p < NumPhases {
		return phaseNames[p]
	}
	return ""
}

type SideRule uint8

const (
	SidesChoose SideRule = iota
	SidesDay
	SidesRandom
	NumSideRules
)

type SeatingRule uint8

const (
	SeatingRandom SeatingRule = iota
	SeatingChosen
	NumSeatingRules
)

type Options struct {
	Sides     SideRule
	Seating   SeatingRule
	HideAtEnd bool
}

func (o Options) Validate() error {
	if o.Sides >= NumSideRules || o.Seating >= NumSeatingRules {
		return fmt.Errorf("rules: invalid options")
	}
	return nil
}

// Ability bits are derived from placed effects; no per-Age used flags exist.
const (
	AbilityPlayLastCard uint8 = 1 << iota
	AbilityFreeFirstOfColour
	AbilityFreeFirstTurn
	AbilityFreeLastTurn
)

type City struct {
	Board     catalog.BoardID
	Side      catalog.Side
	Stages    uint8
	Tucked    [MaxStages]catalog.DesignID
	Buildings catalog.DesignSet
	Coins     int16
	Victories [NumAges]uint8
	Defeats   uint8
	Kept      catalog.DesignID
	// Derived caches: city.go will be their only writer.
	Produce, ForSale [catalog.NumResources]uint8
	Choice           [MaxChoice]catalog.ResMask
	NChoice          uint8
	ChoiceForSale    [MaxForSaleChoice]catalog.ResMask
	NChoiceForSale   uint8
	Discount         [2]catalog.ResMask // left, right
	Colors           [catalog.NumColors]uint8
	Science          [catalog.NumSymbols]uint8
	ScienceChoice    uint8
	Shields          uint8
	Chains           uint32
	Abilities        uint8
}
type State struct {
	Seed      uint64 // setup only; never in views or observations
	Opts      Options
	Seats     uint8
	Phase     Phase
	Age, Turn uint8
	Points    uint16
	Decider   Seat
	Cities    [MaxSeats]City
	Hands     [MaxSeats][HandSize]catalog.DesignID
	Deals     [NumAges][MaxSeats][HandSize]catalog.DesignID
	BoxGuilds [MaxBoxGuilds]catalog.DesignID
	Discard   [catalog.NumDesigns + 1]uint8
	Result    Result
}

// Record is resolved history only. Unresolved picks belong to the room.
type Record struct {
	Packets [NumAges][MaxSeats][HandSize]catalog.DesignID
	Picks   [NumAges][MaxTurns][MaxSeats]catalog.DesignID
	Plays   [NumAges][MaxTurns][MaxSeats]Play
	Rebuilt [NumAges][TurnsPerAge][MaxSeats]catalog.DesignID
}

// Play contains only public fields. Hidden picked designs stay in Picks.
type Play struct {
	Kind        ActionKind
	Design      catalog.DesignID
	Stage       uint8
	Left, Right uint8
}
type Game struct {
	State  State
	Record Record
}

func (s State) Clone() State { return s }
func (g Game) Clone() Game   { return g }

type Score struct{ Wonder, Treasure, Military, Blue, Yellow, Green, Purple, Total int16 }
type Result struct {
	Over    bool
	Scores  [MaxSeats]Score
	Rank    [MaxSeats]uint8
	Winners uint8
}

// These fail at compile time if a frozen core value gains a reference field.
var _ = func(a, b Game, c, d Action, e, f Event, p, q Point) bool { return a == b && c == d && e == f && p == q }
