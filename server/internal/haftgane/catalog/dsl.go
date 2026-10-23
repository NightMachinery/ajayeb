package catalog

import (
	"fmt"
	"strconv"
)

// This file is the small language the data files are written in: one
// constructor per effect kind, Fx to list an item's effects, At for copy
// markers, Out for provided chain symbols, and MustCost to write a cost in
// the notation of docs/haftgane/data-rulings.md. Registration validates every
// value, so a malformed one fails at package initialization.

// Fx lists a design's or a stage's effects.
func Fx(effects ...Effect) []Effect { return effects }

// Produce makes n units of r every turn, for sale to neighbours (brown and
// grey cards).
func Produce(r Resource, n uint8) Effect {
	return Effect{Kind: EffProduce, N: n, Res: r, ForSale: true}
}

// ProduceChoice makes one unit of either of two raw resources, chosen each
// time it is used, for sale to neighbours (the brown choice cards).
func ProduceChoice(a, b Resource) Effect {
	return Effect{Kind: EffProduceChoice, Mask: a.Mask() | b.Mask(), ForSale: true}
}

// ProduceFamily makes one unit of any resource of a family, chosen each time
// it is used. Neighbours may never buy it (Caravansery, Forum and the
// Alexandria stages).
func ProduceFamily(family ResMask) Effect {
	return Effect{Kind: EffProduceChoice, Mask: family}
}

// TradeDiscount sets the price of family units bought from the neighbours in
// scope to 1 coin.
func TradeDiscount(family ResMask, scope SeatScope) Effect {
	return Effect{Kind: EffTradeDiscount, N: 1, Mask: family, Scope: scope}
}

// Shields gives n shields of military strength.
func Shields(n uint8) Effect { return Effect{Kind: EffShields, N: n} }

// Science gives one science symbol.
func Science(s Symbol) Effect { return Effect{Kind: EffScience, Symbol: s} }

// ScienceChoice gives one science symbol of the owner's choice at scoring.
func ScienceChoice() Effect { return Effect{Kind: EffScienceChoice} }

// Coins gives n coins once, when placed.
func Coins(n uint8) Effect { return Effect{Kind: EffCoins, N: n} }

// CoinsPerCard gives n coins per card of the colours in the cities of scope,
// once, when placed.
func CoinsPerCard(scope SeatScope, n uint8, colors ...Color) Effect {
	return Effect{Kind: EffCoinsPerCard, N: n, Colors: colorsOf(colors), Scope: scope}
}

// CoinsPerStage gives n coins per built stage in the cities of scope, once.
func CoinsPerStage(scope SeatScope, n uint8) Effect {
	return Effect{Kind: EffCoinsPerStage, N: n, Scope: scope}
}

// Points gives n points at the end of the game.
func Points(n uint8) Effect { return Effect{Kind: EffPoints, N: n} }

// PointsPerCard gives n points per card of the colours in the cities of
// scope, at the end of the game. Several colours count together.
func PointsPerCard(scope SeatScope, n uint8, colors ...Color) Effect {
	return Effect{Kind: EffPointsPerCard, N: n, Colors: colorsOf(colors), Scope: scope}
}

// PointsPerStage gives n points per built stage in the cities of scope.
func PointsPerStage(scope SeatScope, n uint8) Effect {
	return Effect{Kind: EffPointsPerStage, N: n, Scope: scope}
}

// PointsIfWonderComplete gives n points if every stage of the owner's side
// is built.
func PointsIfWonderComplete(n uint8) Effect {
	return Effect{Kind: EffPointsIfWonderComplete, N: n}
}

// PlayLastCard keeps the sixth turn's leftover card for a last turn.
func PlayLastCard() Effect { return Effect{Kind: EffPlayLastCard} }

// BuildFromDiscard constructs one card from the discard pile for free at the
// end of the turn.
func BuildFromDiscard() Effect { return Effect{Kind: EffBuildFromDiscard} }

// FreeBuild makes a building free under the condition.
func FreeBuild(when FreeCondition) Effect { return Effect{Kind: EffFreeBuild, When: when} }

func colorsOf(colors []Color) ColorMask {
	var m ColorMask
	for _, c := range colors {
		m |= c.Mask()
	}
	return m
}

// At lists a design's copy markers: the minimum player count of each copy.
func At(markers ...uint8) [3]uint8 {
	var c [3]uint8
	if len(markers) > len(c) {
		panic("catalog: more than three copy markers")
	}
	copy(c[:], markers)
	return c
}

// Out lists the chain symbols a design provides.
func Out(chains ...Chain) [2]Chain {
	var c [2]Chain
	if len(chains) > len(c) {
		panic("catalog: more than two provided chain symbols")
	}
	copy(c[:], chains)
	return c
}

// StageOf is one wonder stage.
func StageOf(cost Cost, effects ...Effect) Stage { return Stage{Cost: cost, Effects: effects} }

// SideOf lists a side's stages, two to four.
func SideOf(stages ...Stage) SideDef {
	var s SideDef
	if len(stages) > MaxStages {
		panic("catalog: more than four stages")
	}
	copy(s.Stages[:], stages)
	s.NumStages = uint8(len(stages))
	return s
}

// costLetters are the resource letters of the cost notation.
var costLetters = [NumResources]byte{Wood: 'W', Clay: 'C', Stone: 'S', Ore: 'O', Glass: 'G', Papyrus: 'P', Cloth: 'L'}

// ParseCost reads the cost notation: "" is free, "1c" is one coin, and
// letters are one unit each (W wood, C clay, S stone, O ore, G glass,
// P papyrus, L cloth), such as "SSOG".
func ParseCost(s string) (Cost, error) {
	var c Cost
	if n := len(s); n > 1 && s[n-1] == 'c' {
		coins, err := strconv.ParseUint(s[:n-1], 10, 8)
		if err != nil || coins == 0 {
			return Cost{}, fmt.Errorf("cost %q: bad coin count", s)
		}
		c.Coins = uint8(coins)
		return c, nil
	}
	for i := 0; i < len(s); i++ {
		r := Resource(0)
		for r < NumResources && costLetters[r] != s[i] {
			r++
		}
		if r == NumResources {
			return Cost{}, fmt.Errorf("cost %q: unknown letter %q", s, s[i])
		}
		c.Res[r]++
	}
	return c, nil
}

// MustCost is ParseCost for the data files; a bad cost panics at init.
func MustCost(s string) Cost {
	c, err := ParseCost(s)
	if err != nil {
		panic("catalog: " + err.Error())
	}
	return c
}

// CostString writes a cost in the notation ParseCost reads, letters in
// resource order.
func CostString(c Cost) string {
	if c.Coins != 0 {
		return strconv.Itoa(int(c.Coins)) + "c"
	}
	var b []byte
	for r := Resource(0); r < NumResources; r++ {
		for i := uint8(0); i < c.Res[r]; i++ {
			b = append(b, costLetters[r])
		}
	}
	return string(b)
}
