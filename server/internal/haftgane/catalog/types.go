// Package catalog defines Haftgane's design identities and effect language.
// The verified card data arrives separately. This file is frozen after
// M15.haftgane-core-types; changes require the coordinator's single writer.
package catalog

import (
	"math/bits"
)

const (
	NumDesigns   = 80
	NumNames     = 77
	NumBoards    = 7
	NumGuilds    = 10
	NumChains    = 24
	MaxStages    = 4
	MaxSlugBytes = 30
	MaxCostUnits = 7
)

type DesignID uint8
type NameID uint8
type BoardID uint8
type Side uint8

const (
	NoSide Side = iota
	Day
	Night
)
const (
	BoardAlexandria BoardID = iota + 1
	BoardBabylon
	BoardEphesos
	BoardGizah
	BoardHalikarnassos
	BoardOlympia
	BoardRhodos
)

var boardSlugs = [NumBoards + 1]string{"", "alexandria", "babylon", "ephesos", "gizah", "halikarnassos", "olympia", "rhodos"}

func (id DesignID) Valid() bool { return id >= 1 && id <= NumDesigns }
func (id NameID) Valid() bool   { return id >= 1 && id <= NumNames }
func (id BoardID) Valid() bool  { return id >= 1 && id <= NumBoards }
func (id BoardID) String() string {
	if id.Valid() {
		return boardSlugs[id]
	}
	return ""
}
func (s Side) String() string {
	switch s {
	case Day:
		return "day"
	case Night:
		return "night"
	case NoSide:
		return "none"
	}
	return ""
}

type Resource uint8

const (
	Wood Resource = iota
	Clay
	Stone
	Ore
	Glass
	Papyrus
	Cloth
	NumResources
)

var resourceNames = [NumResources]string{"wood", "clay", "stone", "ore", "glass", "papyrus", "cloth"}

func (r Resource) String() string {
	if r < NumResources {
		return resourceNames[r]
	}
	return ""
}
func (r Resource) Mask() ResMask {
	if r < NumResources {
		return 1 << r
	}
	return 0
}

type ResMask uint8

const (
	Raw          ResMask = 1<<Wood | 1<<Clay | 1<<Stone | 1<<Ore
	Manufactured ResMask = 1<<Glass | 1<<Papyrus | 1<<Cloth
	AllResources         = Raw | Manufactured
)

func (m ResMask) Has(r Resource) bool { return m&r.Mask() != 0 }
func (m ResMask) Count() int          { return bits.OnesCount8(uint8(m)) }
func (m ResMask) Valid() bool         { return m != 0 && m&^AllResources == 0 }

type Color uint8

const (
	NoColor Color = iota
	Brown
	Grey
	Red
	Blue
	Yellow
	Green
	Purple
	NumColors
)

var colorNames = [NumColors]string{"none", "brown", "grey", "red", "blue", "yellow", "green", "purple"}

func (c Color) String() string {
	if c < NumColors {
		return colorNames[c]
	}
	return ""
}
func (c Color) Mask() ColorMask {
	if c < NumColors {
		return 1 << c
	}
	return 0
}

type ColorMask uint16

const AllColors ColorMask = (1 << NumColors) - 2

func (m ColorMask) Has(c Color) bool { return c != NoColor && c < NumColors && m&c.Mask() != 0 }
func (m ColorMask) Valid() bool      { return m != 0 && m&^AllColors == 0 }

type Symbol uint8

const (
	NoSymbol Symbol = iota
	Compass
	Gear
	Tablet
	NumSymbols
)

var symbolNames = [NumSymbols]string{"none", "compass", "gear", "tablet"}

func (s Symbol) String() string {
	if s < NumSymbols {
		return symbolNames[s]
	}
	return ""
}

type Chain uint8

func (c Chain) Bit() uint32 {
	if c >= 1 && c <= NumChains {
		return 1 << c
	}
	return 0
}

type SeatScope uint8

const (
	Self SeatScope = 1 << iota
	Left
	Right
	Beyond
)
const BaseScope = Self | Left | Right

func (s SeatScope) Valid() bool { return s != 0 && s&^BaseScope == 0 }

type FreeCondition uint8

const (
	NoFreeCondition FreeCondition = iota
	FirstOfColour
	FirstTurnOfAge
	LastTurnOfAge
	NumFreeConditions
)

type EffectKind uint8

const (
	EffNone EffectKind = iota
	EffProduce
	EffProduceChoice
	EffTradeDiscount
	EffShields
	EffScience
	EffScienceChoice
	EffCoins
	EffCoinsPerCard
	EffCoinsPerStage
	EffPoints
	EffPointsPerCard
	EffPointsPerStage
	EffPointsIfWonderComplete
	EffPlayLastCard
	EffBuildFromDiscard
	EffFreeBuild
	NumEffectKinds
)

var effectNames = [NumEffectKinds]string{"none", "produce", "produce-choice", "trade-discount", "shields", "science", "science-choice", "coins", "coins-per-card", "coins-per-stage", "points", "points-per-card", "points-per-stage", "points-if-wonder-complete", "play-last-card", "build-from-discard", "free-build"}

func (k EffectKind) String() string {
	if k < NumEffectKinds {
		return effectNames[k]
	}
	return ""
}

// Timing is a mask: science has a continuous cache and end-game scoring.
type Timing uint8

const (
	Continuous Timing = 1 << iota
	WhenPlaced
	EndOfTurn
	EndOfGame
)

var effectTimings = [NumEffectKinds]Timing{
	EffProduce: Continuous, EffProduceChoice: Continuous, EffTradeDiscount: Continuous, EffShields: Continuous,
	EffScience: Continuous | EndOfGame, EffScienceChoice: Continuous | EndOfGame,
	EffCoins: WhenPlaced, EffCoinsPerCard: WhenPlaced, EffCoinsPerStage: WhenPlaced,
	EffPoints: EndOfGame, EffPointsPerCard: EndOfGame, EffPointsPerStage: EndOfGame, EffPointsIfWonderComplete: EndOfGame,
	EffPlayLastCard: Continuous, EffBuildFromDiscard: EndOfTurn, EffFreeBuild: Continuous,
}

func (k EffectKind) Timings() Timing {
	if k < NumEffectKinds {
		return effectTimings[k]
	}
	return 0
}

// N is used only for explicitly counted effects. A choice producer, science
// symbol or ability supplies one implicitly and keeps N zero. Unused fields
// must be zero. Free-build is held by a stage and targets buildings only.
type Effect struct {
	Kind    EffectKind
	N       uint8
	Res     Resource
	Mask    ResMask
	Colors  ColorMask
	Scope   SeatScope
	Symbol  Symbol
	ForSale bool
	When    FreeCondition
}

type Cost struct {
	Coins uint8
	Res   [NumResources]uint8
}

func (c Cost) Units() uint16 {
	var n uint16
	for _, units := range c.Res {
		n += uint16(units)
	}
	return n
}
func (c Cost) Free() bool { return c == Cost{} }

type Design struct {
	ID       DesignID
	Slug     string
	Name     string
	NameID   NameID
	Age      uint8
	Color    Color
	Cost     Cost
	ChainIn  Chain
	ChainOut [2]Chain
	Copies   [3]uint8
	Effects  []Effect
}
type Stage struct {
	Cost    Cost
	Effects []Effect
}
type SideDef struct {
	Stages    [MaxStages]Stage
	NumStages uint8
}
type Board struct {
	ID     BoardID
	Slug   string
	Name   string
	Wonder string
	Start  Resource
	Sides  [2]SideDef
}

// DesignSet is a 128-bit set. Bit zero and bits above NumDesigns are unused.
type DesignSet [2]uint64

func DesignSetOf(ids ...DesignID) DesignSet {
	var s DesignSet
	for _, id := range ids {
		s.Add(id)
	}
	return s
}
func (s DesignSet) Has(id DesignID) bool { return id.Valid() && s[id>>6]&(uint64(1)<<(id&63)) != 0 }
func (s *DesignSet) Add(id DesignID) {
	if !id.Valid() {
		panic("catalog: invalid design id")
	}
	s[id>>6] |= uint64(1) << (id & 63)
}
func (s *DesignSet) Remove(id DesignID) {
	if id.Valid() {
		s[id>>6] &^= uint64(1) << (id & 63)
	}
}
func (s DesignSet) Count() int                   { return bits.OnesCount64(s[0]) + bits.OnesCount64(s[1]) }
func (s DesignSet) Empty() bool                  { return s[0]|s[1] == 0 }
func (s DesignSet) And(o DesignSet) DesignSet    { return DesignSet{s[0] & o[0], s[1] & o[1]} }
func (s DesignSet) Or(o DesignSet) DesignSet     { return DesignSet{s[0] | o[0], s[1] | o[1]} }
func (s DesignSet) AndNot(o DesignSet) DesignSet { return DesignSet{s[0] &^ o[0], s[1] &^ o[1]} }
func (s DesignSet) Next(from DesignID) (DesignID, bool) {
	if from == 0 {
		from = 1
	}
	for id := uint16(from); id <= NumDesigns; id++ {
		if s.Has(DesignID(id)) {
			return DesignID(id), true
		}
	}
	return 0, false
}
