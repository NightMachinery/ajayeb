package catalog

import (
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// This file is the small language the data files are written in: one
// constructor per effect kind, Fx to list a card's effects, and MustCost to
// write a cost in the reference notation ("SS C W G", "3c", "2c + G P").
// Registration validates every effect, so a malformed one fails at init.

// Fx lists a card's, wonder's or token's effects.
func Fx(effects ...Effect) []Effect { return effects }

// Produce makes n units of r every turn (brown and grey cards).
func Produce(r Resource, n uint8) Effect { return Effect{Kind: EffProduce, Res: r, N: n} }

// ProduceChoice makes one unit of any of rs every turn (Forum, Caravansery,
// Great Lighthouse, Piraeus).
func ProduceChoice(rs ...Resource) Effect {
	return Effect{Kind: EffProduceChoice, Mask: MaskOf(rs...)}
}

// TradeFix fixes the price of rs at 1 coin per unit (the reserves and the
// Customs House).
func TradeFix(rs ...Resource) Effect { return Effect{Kind: EffTradeFix, Mask: MaskOf(rs...)} }

// Shields moves the pawn n spaces toward the opponent.
func Shields(n uint8) Effect { return Effect{Kind: EffShields, N: n} }

// Science gives the science symbol s.
func Science(s Symbol) Effect { return Effect{Kind: EffScience, Symbol: s} }

// Points gives n victory points.
func Points(n uint8) Effect { return Effect{Kind: EffPoints, N: n} }

// Coins gives n coins once, when built or taken.
func Coins(n uint8) Effect { return Effect{Kind: EffCoins, N: n} }

// CoinsPerCard gives n coins per card of the given colours, once, counted in
// the scope's city after the new card has entered its owner's city.
func CoinsPerCard(scope Scope, n uint8, colors ...Color) Effect {
	return Effect{Kind: EffCoinsPerCard, Scope: scope, N: n, Colors: ColorsOf(colors...)}
}

// CoinsPerWonder gives n coins per built wonder in the scope's city, once.
func CoinsPerWonder(scope Scope, n uint8) Effect {
	return Effect{Kind: EffCoinsPerWonder, Scope: scope, N: n}
}

// PointsPerCard gives n VP per card of the given colours in the scope's city,
// counted at game end. Several colours count together, in one city.
func PointsPerCard(scope Scope, n uint8, colors ...Color) Effect {
	return Effect{Kind: EffPointsPerCard, Scope: scope, N: n, Colors: ColorsOf(colors...)}
}

// PointsPerWonder gives n VP per built wonder in the scope's city at game end.
func PointsPerWonder(scope Scope, n uint8) Effect {
	return Effect{Kind: EffPointsPerWonder, Scope: scope, N: n}
}

// PointsPerCoins gives 1 VP per full `per` coins in the scope's city at game
// end.
func PointsPerCoins(scope Scope, per uint8) Effect {
	return Effect{Kind: EffPointsPerCoins, Scope: scope, N: per}
}

// PointsPerToken gives n VP per progress token the owner has, itself
// included.
func PointsPerToken(n uint8) Effect { return Effect{Kind: EffPointsPerToken, N: n} }

// PlayAgain gives the builder another turn.
func PlayAgain() Effect { return Effect{Kind: EffPlayAgain} }

// Destroy lets the builder discard one of the opponent's cards of colour c.
func Destroy(c Color) Effect { return Effect{Kind: EffDestroy, Colors: c.Mask()} }

// BuildFromDiscard lets the builder construct a discarded card for free.
func BuildFromDiscard() Effect { return Effect{Kind: EffBuildFromDiscard} }

// DrawTokens lets the builder draw n tokens from the box and keep one.
func DrawTokens(n uint8) Effect { return Effect{Kind: EffDrawTokens, N: n} }

// OpponentLosesCoins makes the opponent lose n coins, never below zero.
func OpponentLosesCoins(n uint8) Effect { return Effect{Kind: EffOpponentLosesCoins, N: n} }

// Discount waives n resource units of every future item of the target kind.
func Discount(t DiscountTarget, n uint8) Effect {
	return Effect{Kind: EffDiscount, Target: t, N: n}
}

// TradeIncome gives the owner the coins the opponent pays for trading.
func TradeIncome() Effect { return Effect{Kind: EffTradeIncome} }

// ExtraShield adds n shields to each red building the owner builds later.
func ExtraShield(n uint8) Effect { return Effect{Kind: EffExtraShield, N: n} }

// WondersPlayAgain gives another turn for every wonder the owner builds later.
func WondersPlayAgain() Effect { return Effect{Kind: EffWondersPlayAgain} }

// ChainBonus gives n coins whenever a chain makes one of the owner's
// buildings free.
func ChainBonus(n uint8) Effect { return Effect{Kind: EffChainBonus, N: n} }

// FindEffect returns the first effect of the given kind.
func FindEffect(effects []Effect, kind EffectKind) (Effect, bool) {
	for _, e := range effects {
		if e.Kind == kind {
			return e, true
		}
	}
	return Effect{}, false
}

// Which Effect fields a kind uses.
const (
	usesN uint8 = 1 << iota
	usesRes
	usesMask
	usesColors
	usesSymbol
	usesScope
	usesTarget
)

var effectFields = [numEffectKinds]uint8{
	EffProduce:            usesN | usesRes,
	EffProduceChoice:      usesMask,
	EffTradeFix:           usesMask,
	EffShields:            usesN,
	EffScience:            usesSymbol,
	EffPoints:             usesN,
	EffCoins:              usesN,
	EffCoinsPerCard:       usesN | usesColors | usesScope,
	EffCoinsPerWonder:     usesN | usesScope,
	EffPointsPerCard:      usesN | usesColors | usesScope,
	EffPointsPerWonder:    usesN | usesScope,
	EffPointsPerCoins:     usesN | usesScope,
	EffPointsPerToken:     usesN,
	EffPlayAgain:          0,
	EffDestroy:            usesColors,
	EffBuildFromDiscard:   0,
	EffDrawTokens:         usesN,
	EffOpponentLosesCoins: usesN,
	EffDiscount:           usesN | usesTarget,
	EffTradeIncome:        0,
	EffExtraShield:        usesN,
	EffWondersPlayAgain:   0,
	EffChainBonus:         usesN,
}

// validColors holds every real colour (bit 0, NoColor, excluded).
const validColors = ColorMask(1<<NumColors-1) &^ 1

// Validate reports whether the effect is well formed: a known kind, every
// field the kind uses in range, and every other field zero.
func (e Effect) Validate() error {
	if e.Kind == EffNone || e.Kind >= numEffectKinds {
		return fmt.Errorf("unknown effect kind %d", e.Kind)
	}
	uses := effectFields[e.Kind]
	bad := func(field string) error { return fmt.Errorf("effect %s: invalid %s in %+v", e.Kind, field, e) }
	if uses&usesN != 0 {
		if e.N == 0 {
			return bad("N (must be at least 1)")
		}
	} else if e.N != 0 {
		return bad("N (unused by this kind)")
	}
	if uses&usesRes != 0 {
		if e.Res >= NumResources {
			return bad("Res")
		}
	} else if e.Res != 0 {
		return bad("Res (unused by this kind)")
	}
	if uses&usesMask != 0 {
		if e.Mask == 0 || e.Mask&^AllResources != 0 {
			return bad("Mask")
		}
	} else if e.Mask != 0 {
		return bad("Mask (unused by this kind)")
	}
	if uses&usesColors != 0 {
		if e.Colors == 0 || e.Colors&^validColors != 0 {
			return bad("Colors")
		}
		if e.Kind == EffDestroy && bits.OnesCount8(uint8(e.Colors)) != 1 {
			return bad("Colors (destroy takes exactly one colour)")
		}
	} else if e.Colors != 0 {
		return bad("Colors (unused by this kind)")
	}
	if uses&usesSymbol != 0 {
		if e.Symbol == NoSymbol || e.Symbol >= NumSymbols {
			return bad("Symbol")
		}
	} else if e.Symbol != NoSymbol {
		return bad("Symbol (unused by this kind)")
	}
	if uses&usesScope != 0 {
		if e.Scope != ScopeSelf && e.Scope != ScopeMaxCity {
			return bad("Scope")
		}
	} else if e.Scope != NoScope {
		return bad("Scope (unused by this kind)")
	}
	if uses&usesTarget != 0 {
		if e.Target != ForBlue && e.Target != ForWonders {
			return bad("Target")
		}
	} else if e.Target != NoTarget {
		return bad("Target (unused by this kind)")
	}
	return nil
}

// String renders the effect compactly for debugging, for example
// "produce W2" or "coins-per-card 1 yellow self". It is not the player-facing
// text, which the client builds from the structured data.
func (e Effect) String() string {
	if e.Kind >= numEffectKinds {
		return e.Kind.String()
	}
	uses := effectFields[e.Kind]
	var b strings.Builder
	b.WriteString(e.Kind.String())
	if uses&usesRes != 0 {
		b.WriteByte(' ')
		b.WriteByte(e.Res.Letter())
		b.WriteString(strconv.Itoa(int(e.N)))
	} else if uses&usesN != 0 {
		b.WriteByte(' ')
		b.WriteString(strconv.Itoa(int(e.N)))
	}
	if uses&usesMask != 0 {
		b.WriteByte(' ')
		for r := range NumResources {
			if e.Mask.Has(r) {
				b.WriteByte(r.Letter())
			}
		}
	}
	if uses&usesColors != 0 {
		sep := byte(' ')
		for c := Brown; c < NumColors; c++ {
			if e.Colors.Has(c) {
				b.WriteByte(sep)
				b.WriteString(c.String())
				sep = '+'
			}
		}
	}
	if uses&usesSymbol != 0 {
		b.WriteByte(' ')
		b.WriteString(e.Symbol.String())
	}
	if uses&usesScope != 0 {
		b.WriteByte(' ')
		b.WriteString(e.Scope.String())
	}
	if uses&usesTarget != 0 {
		b.WriteByte(' ')
		b.WriteString(e.Target.String())
	}
	return b.String()
}

// ErrBadCost is wrapped by every ParseCost error.
var ErrBadCost = errors.New("catalog: bad cost")

// ParseCost reads a cost in the reference notation. Items are separated by
// spaces; a lone "+" may separate the coins from the resources:
//
//   - "" or "free" is no cost;
//   - "3c" is 3 coins: a positive number without leading zeros, then "c",
//     written once and before any resource;
//   - "SS" is 2 stone: a run of one resource letter (W, C, S, G or P), each
//     letter in at most one run.
//
// So "SS C W G", "3c", "3c + C" and "2c + G P" are valid, while "3", "SW",
// "S S", "3c 2c", "C 3c", "0c" and "w" are not.
func ParseCost(s string) (Cost, error) {
	var c Cost
	items := strings.Fields(s)
	if len(items) == 0 || (len(items) == 1 && items[0] == "free") {
		return c, nil
	}
	fail := func(why string) (Cost, error) {
		return Cost{}, fmt.Errorf("%w %q: %s", ErrBadCost, s, why)
	}
	var seen [NumResources]bool
	sawRes := false
	for i, item := range items {
		switch {
		case item == "+":
			if i == 0 || i == len(items)-1 || items[i-1] == "+" {
				return fail(`"+" must sit between two items`)
			}
		case item[len(item)-1] == 'c':
			digits := item[:len(item)-1]
			if c.Coins != 0 {
				return fail("coins written twice")
			}
			if sawRes {
				return fail("coins must come before the resources")
			}
			if digits == "" || digits[0] == '0' {
				return fail("coins need a positive number without leading zeros")
			}
			n, err := strconv.ParseUint(digits, 10, 8)
			if err != nil {
				return fail("coins must be a number from 1 to 255")
			}
			c.Coins = uint8(n)
		default:
			idx := strings.IndexByte(resourceLetters, item[0])
			if idx < 0 {
				return fail("unknown item " + strconv.Quote(item))
			}
			r := Resource(idx)
			if strings.Count(item, item[:1]) != len(item) {
				return fail("a run must repeat a single letter: " + strconv.Quote(item))
			}
			if seen[r] {
				return fail("resource " + item[:1] + " written in two runs")
			}
			if len(item) > 255 {
				return fail("run too long")
			}
			seen[r] = true
			sawRes = true
			c.Res[r] = uint8(len(item))
		}
	}
	return c, nil
}

// MustCost is ParseCost for the data files: it panics on a bad cost, which
// makes the package fail at init and every test fail loudly.
func MustCost(s string) Cost {
	c, err := ParseCost(s)
	if err != nil {
		panic(err)
	}
	return c
}
