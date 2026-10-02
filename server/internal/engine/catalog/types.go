// Package catalog holds the static data of the base game (73 buildings, 12
// wonders and 10 progress tokens) and the small effect vocabulary that
// describes them. There is no per-card code: every rule a card carries is one
// of the effect kinds below, and the rules engine interprets them.
//
// Ids are small integers. Buildings use the numbering of the public reference
// implementation (1 to 73); wonders and tokens use our own canonical order,
// fixed by the constants in this file. On the wire every id travels as a
// kebab-case English slug without a leading "the-" (lumber-yard, pyramids,
// great-library, agriculture), through MarshalText and UnmarshalText.
//
// The data files register their items through registry.go. This file is
// frozen after slice S1: change it only through the coordinator.
package catalog

import (
	"iter"
	"math/bits"
	"strconv"
)

// Item counts of the base game.
const (
	NumCards   = 73 // buildings: 23 Age I, 23 Age II, 20 Age III and 7 guilds
	NumGuilds  = 7
	NumWonders = 12
	NumTokens  = 10
)

// Unknown is the sentinel id that an AI observation puts where a card, wonder
// or token is hidden from the observer. It never appears in a real game state
// and is never registered. It is untyped so it fits every id type.
const Unknown = 0xFF

// CardID identifies a building (1 to 73). Zero means "no card".
type CardID uint8

// WonderID identifies a wonder (1 to 12). Zero means "no wonder".
type WonderID uint8

// TokenID identifies a progress token (1 to 10). Zero means "no token".
type TokenID uint8

// The wonders, in our canonical order. Their slugs are fixed: registry.go
// rejects a wonder whose slug does not match its id.
const (
	WonderPyramids WonderID = iota + 1
	WonderColossus
	WonderGreatLighthouse
	WonderTempleOfArtemis
	WonderMausoleum
	WonderGreatLibrary
	WonderPiraeus
	WonderHangingGardens
	WonderStatueOfZeus
	WonderSphinx
	WonderAppianWay
	WonderCircusMaximus
)

// The progress tokens, in our canonical (alphabetical) order. Their slugs are
// fixed like the wonders' slugs.
const (
	TokenAgriculture TokenID = iota + 1
	TokenArchitecture
	TokenEconomy
	TokenLaw
	TokenMasonry
	TokenMathematics
	TokenPhilosophy
	TokenStrategy
	TokenTheology
	TokenUrbanism
)

// wonderSlugs and tokenSlugs are the slugs that the id constants above stand
// for, indexed by id.
var (
	wonderSlugs = [NumWonders + 1]string{
		"", "pyramids", "colossus", "great-lighthouse", "temple-of-artemis",
		"mausoleum", "great-library", "piraeus", "hanging-gardens",
		"statue-of-zeus", "sphinx", "appian-way", "circus-maximus",
	}
	tokenSlugs = [NumTokens + 1]string{
		"", "agriculture", "architecture", "economy", "law", "masonry",
		"mathematics", "philosophy", "strategy", "theology", "urbanism",
	}
)

// CardSet is a set of card ids, as a 128-bit bitset in which bit i stands for
// card i. Ids from 128 up (including Unknown) are never members.
type CardSet [2]uint64

// CardSetOf returns the set holding exactly the given cards.
func CardSetOf(ids ...CardID) CardSet {
	var s CardSet
	for _, id := range ids {
		s.Add(id)
	}
	return s
}

// Has reports whether id is in the set.
func (s CardSet) Has(id CardID) bool {
	return id < 128 && s[id>>6]&(1<<(id&63)) != 0
}

// Add puts id into the set. It panics for ids from 128 up.
func (s *CardSet) Add(id CardID) { s[id>>6] |= 1 << (id & 63) }

// Remove takes id out of the set. It panics for ids from 128 up.
func (s *CardSet) Remove(id CardID) { s[id>>6] &^= 1 << (id & 63) }

// Count returns the number of cards in the set.
func (s CardSet) Count() int { return bits.OnesCount64(s[0]) + bits.OnesCount64(s[1]) }

// Empty reports whether the set has no cards.
func (s CardSet) Empty() bool { return s[0]|s[1] == 0 }

// And returns the cards in both sets.
func (s CardSet) And(o CardSet) CardSet { return CardSet{s[0] & o[0], s[1] & o[1]} }

// Or returns the cards in either set.
func (s CardSet) Or(o CardSet) CardSet { return CardSet{s[0] | o[0], s[1] | o[1]} }

// AndNot returns the cards in s that are not in o.
func (s CardSet) AndNot(o CardSet) CardSet { return CardSet{s[0] &^ o[0], s[1] &^ o[1]} }

// Next returns the smallest card in the set that is at least from, and false
// when there is none. Iterate with
//
//	for id, ok := s.Next(1); ok; id, ok = s.Next(id + 1) { ... }
func (s CardSet) Next(from CardID) (CardID, bool) {
	for w := int(from >> 6); w < len(s); w++ {
		word := s[w]
		if w == int(from>>6) {
			word &= ^uint64(0) << (from & 63)
		}
		if word != 0 {
			return CardID(w<<6 + bits.TrailingZeros64(word)), true
		}
	}
	return 0, false
}

// All yields the cards of the set in ascending id order.
func (s CardSet) All() iter.Seq[CardID] {
	return func(yield func(CardID) bool) {
		for id, ok := s.Next(0); ok; id, ok = s.Next(id + 1) {
			if !yield(id) {
				return
			}
		}
	}
}

// TokenSet is a set of progress tokens, as a bitset in which bit i stands for
// token i. Ids from 16 up (including Unknown) are never members.
type TokenSet uint16

// TokenSetOf returns the set holding exactly the given tokens.
func TokenSetOf(ids ...TokenID) TokenSet {
	var s TokenSet
	for _, id := range ids {
		s.Add(id)
	}
	return s
}

// Has reports whether id is in the set.
func (s TokenSet) Has(id TokenID) bool { return id < 16 && s&(1<<id) != 0 }

// Add puts id into the set. It panics for ids from 16 up.
func (s *TokenSet) Add(id TokenID) {
	if id >= 16 {
		panic("catalog: token id " + strconv.Itoa(int(id)) + " does not fit a TokenSet")
	}
	*s |= 1 << id
}

// Remove takes id out of the set. Ids from 16 up are never members, so
// removing one does nothing.
func (s *TokenSet) Remove(id TokenID) {
	if id < 16 {
		*s &^= 1 << id
	}
}

// Count returns the number of tokens in the set.
func (s TokenSet) Count() int { return bits.OnesCount16(uint16(s)) }

// Empty reports whether the set has no tokens.
func (s TokenSet) Empty() bool { return s == 0 }

// Next returns the smallest token in the set that is at least from, and false
// when there is none.
func (s TokenSet) Next(from TokenID) (TokenID, bool) {
	if from >= 16 {
		return 0, false
	}
	rest := uint16(s) & (^uint16(0) << from)
	if rest == 0 {
		return 0, false
	}
	return TokenID(bits.TrailingZeros16(rest)), true
}

// All yields the tokens of the set in ascending id order.
func (s TokenSet) All() iter.Seq[TokenID] {
	return func(yield func(TokenID) bool) {
		for id, ok := s.Next(0); ok; id, ok = s.Next(id + 1) {
			if !yield(id) {
				return
			}
		}
	}
}

// Resource is one of the five resources. Wood is the zero value so that
// per-resource arrays have no unused entry.
type Resource uint8

// The resources, in the order used by every per-resource array.
const (
	Wood Resource = iota
	Clay
	Stone
	Glass
	Papyrus
	NumResources
)

var resourceNames = [NumResources]string{"wood", "clay", "stone", "glass", "papyrus"}

// resourceLetters holds the one-letter codes of the reference notation.
const resourceLetters = "WCSGP"

// String returns the lower-case English name, for example "wood".
func (r Resource) String() string {
	if r < NumResources {
		return resourceNames[r]
	}
	return "resource#" + strconv.Itoa(int(r))
}

// Letter returns the one-letter code of the reference notation: W, C, S, G
// or P. It returns '?' for an invalid resource.
func (r Resource) Letter() byte {
	if r < NumResources {
		return resourceLetters[r]
	}
	return '?'
}

// Mask returns the mask holding only r.
func (r Resource) Mask() ResMask { return 1 << r }

// ResMask is a set of resources (bit r stands for resource r).
type ResMask uint8

// Resource classes. Brown cards make raw goods and grey cards manufactured
// goods; every flexible producer of the base game offers exactly one class.
const (
	RawGoods     ResMask = 1<<Wood | 1<<Clay | 1<<Stone
	Manufactured ResMask = 1<<Glass | 1<<Papyrus
	AllResources         = RawGoods | Manufactured
)

// MaskOf returns the mask holding the given resources.
func MaskOf(rs ...Resource) ResMask {
	var m ResMask
	for _, r := range rs {
		m |= r.Mask()
	}
	return m
}

// Has reports whether r is in the mask.
func (m ResMask) Has(r Resource) bool { return r < NumResources && m&r.Mask() != 0 }

// Color is a building's colour. The zero value NoColor is never a valid card
// colour, so a card whose colour was forgotten is caught at registration.
type Color uint8

// The building colours.
const (
	NoColor Color = iota
	Brown         // raw goods
	Grey          // manufactured goods
	Red           // military
	Green         // science
	Yellow        // commercial
	Blue          // civilian
	Purple        // guilds
	NumColors
)

var colorNames = [NumColors]string{"none", "brown", "grey", "red", "green", "yellow", "blue", "purple"}

// String returns the lower-case English name, for example "grey".
func (c Color) String() string {
	if c < NumColors {
		return colorNames[c]
	}
	return "color#" + strconv.Itoa(int(c))
}

// Mask returns the mask holding only c.
func (c Color) Mask() ColorMask { return 1 << c }

// ColorMask is a set of colours (bit c stands for colour c; bit 0, NoColor,
// is never set in valid data).
type ColorMask uint8

// ColorsOf returns the mask holding the given colours.
func ColorsOf(cs ...Color) ColorMask {
	var m ColorMask
	for _, c := range cs {
		m |= c.Mask()
	}
	return m
}

// Has reports whether c is in the mask.
func (m ColorMask) Has(c Color) bool { return c < NumColors && m&c.Mask() != 0 }

// Symbol is a science symbol. The numbering follows the reference
// implementation. Scales appears only on the Law token.
type Symbol uint8

// The science symbols.
const (
	NoSymbol Symbol = iota
	Globe
	Scales
	Sundial
	Mortar
	SetSquare
	Quill
	Wheel
	NumSymbols // one past the last symbol: the size of per-symbol arrays
)

var symbolNames = [NumSymbols]string{"none", "globe", "scales", "sundial", "mortar", "set-square", "quill", "wheel"}

// String returns the kebab-case name, for example "set-square".
func (s Symbol) String() string {
	if s < NumSymbols {
		return symbolNames[s]
	}
	return "symbol#" + strconv.Itoa(int(s))
}

// Chain is a chain symbol. Each of the 17 symbols is provided by exactly one
// card and makes exactly one later card free.
type Chain uint8

// The chain symbols.
const (
	NoChain Chain = iota
	ChainHorseshoe
	ChainSword
	ChainTower
	ChainBook
	ChainGear
	ChainAmphora
	ChainMask
	ChainMoon
	ChainDrop
	ChainTarget
	ChainHelmet
	ChainLyre
	ChainLamp
	ChainBarrel
	ChainSun
	ChainColumn
	ChainTempleFacade
	NumChains // one past the last chain; a uint32 bitset holds them all
)

var chainNames = [NumChains]string{
	"none", "horseshoe", "sword", "tower", "book", "gear", "amphora", "mask",
	"moon", "drop", "target", "helmet", "lyre", "lamp", "barrel", "sun",
	"column", "temple-facade",
}

// String returns the kebab-case name, for example "temple-facade".
func (c Chain) String() string {
	if c < NumChains {
		return chainNames[c]
	}
	return "chain#" + strconv.Itoa(int(c))
}

// Bit returns the chain's bit in a chain bitset (zero for NoChain).
func (c Chain) Bit() uint32 {
	if c == NoChain || c >= NumChains {
		return 0
	}
	return 1 << c
}

// Back is the printed back of a card, which stays visible when the card lies
// face down. Zero means "no back" (an empty slot).
type Back uint8

// The card backs.
const (
	NoBack Back = iota
	BackAge1
	BackAge2
	BackAge3
	BackGuild
)

// GuildBack is the back of the guild cards. Guilds have their own public back
// (rules decision 2); if that ruling changed, this constant is the one switch.
const GuildBack = BackGuild

var backNames = [...]string{"none", "age1", "age2", "age3", "guild"}

// AgeBack returns the back of the non-guild cards of an age (1 to 3), or
// NoBack for any other age.
func AgeBack(age uint8) Back {
	if age >= 1 && age <= 3 {
		return BackAge1 + Back(age-1)
	}
	return NoBack
}

// String returns the wire name: age1, age2, age3 or guild.
func (b Back) String() string {
	if int(b) < len(backNames) {
		return backNames[b]
	}
	return "back#" + strconv.Itoa(int(b))
}

// Cost is a printed cost: coins plus units of each resource.
type Cost struct {
	Coins uint8
	Res   [NumResources]uint8
}

// Free reports whether the cost is nothing at all.
func (c Cost) Free() bool { return c == Cost{} }

// Units returns the total number of resource units.
func (c Cost) Units() int {
	n := 0
	for _, u := range c.Res {
		n += int(u)
	}
	return n
}

// String returns the cost in the canonical form of the reference notation:
// "free", or the coins ("3c") followed by one letter run per resource in the
// order W C S G P, for example "2c G P" or "SS C W". ParseCost reads it back.
func (c Cost) String() string {
	if c.Free() {
		return "free"
	}
	var b []byte
	if c.Coins > 0 {
		b = strconv.AppendUint(b, uint64(c.Coins), 10)
		b = append(b, 'c')
	}
	for r := range NumResources {
		if c.Res[r] == 0 {
			continue
		}
		if len(b) > 0 {
			b = append(b, ' ')
		}
		for range c.Res[r] {
			b = append(b, r.Letter())
		}
	}
	return string(b)
}

// Scope says whose city a "per card" or "per wonder" effect counts.
type Scope uint8

// The scopes. The zero value NoScope is invalid for effects that count.
const (
	NoScope      Scope = iota
	ScopeSelf          // the owner's city only (the Age III yellow cards, the Arena)
	ScopeMaxCity       // whichever city has more (the guilds); the two cities may differ per effect
)

var scopeNames = [...]string{"none", "self", "max"}

// String returns "self" or "max".
func (s Scope) String() string {
	if int(s) < len(scopeNames) {
		return scopeNames[s]
	}
	return "scope#" + strconv.Itoa(int(s))
}

// DiscountTarget says what a discount token applies to.
type DiscountTarget uint8

// The discount targets. The zero value NoTarget is invalid for a discount.
const (
	NoTarget   DiscountTarget = iota
	ForBlue                   // Masonry: blue buildings
	ForWonders                // Architecture: wonders
)

var targetNames = [...]string{"none", "blue", "wonder"}

// String returns "blue" or "wonder".
func (t DiscountTarget) String() string {
	if int(t) < len(targetNames) {
		return targetNames[t]
	}
	return "target#" + strconv.Itoa(int(t))
}

// EffectKind is what an effect does. Each kind uses a fixed subset of the
// Effect fields; Effect.Validate rejects any other field being set.
type EffectKind uint8

// The effect kinds. "Self" and "max" refer to the effect's Scope.
const (
	EffNone EffectKind = iota

	// Production and trade.
	EffProduce       // N units of Res every turn (brown and grey cards); raises the opponent's price for Res
	EffProduceChoice // one unit from Mask every turn; never affects prices
	EffTradeFix      // buying any resource of Mask costs 1 coin per unit

	// Military and science.
	EffShields // move the pawn N spaces toward the opponent
	EffScience // the science symbol Symbol

	// Victory points and coins.
	EffPoints          // N victory points
	EffCoins           // N coins once, on build
	EffCoinsPerCard    // N coins per card of Colors (self or max city), once, counted after the card enters the city
	EffCoinsPerWonder  // N coins per built wonder (self or max city), once
	EffPointsPerCard   // N VP per card of Colors (self or max city), at game end
	EffPointsPerWonder // N VP per built wonder (self or max city), at game end
	EffPointsPerCoins  // 1 VP per full N coins (self or max city), at game end
	EffPointsPerToken  // N VP per progress token owned, this one included

	// Wonder specials.
	EffPlayAgain          // the builder takes another turn
	EffDestroy            // the builder discards one opponent card of the single colour in Colors
	EffBuildFromDiscard   // the builder constructs one discarded card for free
	EffDrawTokens         // the builder draws N tokens from the box and keeps one
	EffOpponentLosesCoins // the opponent loses N coins (never below zero)

	// Progress token rules.
	EffDiscount         // the owner's future Target items cost N fewer resource units
	EffTradeIncome      // the owner receives the coins the opponent pays for trading
	EffExtraShield      // the owner's future red buildings give N more shields
	EffWondersPlayAgain // every wonder the owner builds later gives another turn (never two)
	EffChainBonus       // the owner gains N coins whenever a chain makes a building free

	numEffectKinds
)

var effectKindNames = [numEffectKinds]string{
	"none",
	"produce", "produce-choice", "trade-fix",
	"shields", "science",
	"points", "coins", "coins-per-card", "coins-per-wonder", "points-per-card",
	"points-per-wonder", "points-per-coins", "points-per-token",
	"play-again", "destroy", "build-from-discard", "draw-tokens", "opponent-loses-coins",
	"discount", "trade-income", "extra-shield", "wonders-play-again", "chain-bonus",
}

// String returns the kebab-case name, for example "coins-per-card".
func (k EffectKind) String() string {
	if k < numEffectKinds {
		return effectKindNames[k]
	}
	return "effect#" + strconv.Itoa(int(k))
}

// Effect is one rule printed on a card, wonder or token. It is a small
// comparable value; which fields matter depends on Kind.
type Effect struct {
	Kind   EffectKind
	N      uint8
	Res    Resource
	Mask   ResMask
	Colors ColorMask
	Symbol Symbol
	Scope  Scope
	Target DiscountTarget
}

// CardDef is a building.
type CardDef struct {
	ID       CardID
	Slug     string   // wire id, for example "lumber-yard"
	Name     string   // English display name, for example "Lumber Yard"
	AltNames []string // other printed names of the same card (some sources use other names)
	Age      uint8    // 1 to 3; guilds are 3. Set by the registration call.
	Back     Back     // derived at registration: GuildBack for guilds, else the age's back
	Color    Color
	Cost     Cost
	ChainIn  Chain // the chain symbol that makes this card free
	ChainOut Chain // the chain symbol this card provides
	Effects  []Effect
}

// IsGuild reports whether the card is a guild.
func (d *CardDef) IsGuild() bool { return d.Color == Purple }

// WonderDef is a wonder.
type WonderDef struct {
	ID      WonderID
	Slug    string
	Name    string
	Cost    Cost
	Effects []Effect
}

// TokenDef is a progress token.
type TokenDef struct {
	ID      TokenID
	Slug    string
	Name    string
	Effects []Effect
}
