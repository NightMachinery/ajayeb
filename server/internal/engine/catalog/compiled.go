package catalog

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// This file derives two things from the registered catalog:
//
//   - T, read-only lookup tables indexed by id for the engine's hot paths
//     (payment, legality, city caches, scoring), built once at init and again
//     whenever ReplaceForTest swaps the catalog;
//   - the verification records: the catalog flattened into the CSV schema
//     that the independent verification tables use, so that the data can be
//     diffed field by field against other sources. ajayeb-gen catalog writes
//     them, and catalog_test.go pins them in testdata/*.golden.csv.
//
// Building the tables also asserts the structural facts the engine relies
// on; a violation panics at init, which fails every test that imports the
// catalog.

// Tables are lookup tables derived from the catalog. Arrays indexed by a card
// id have NumCards+1 entries (entry 0, "no card", is all zero), and likewise
// for wonders and tokens. Never write to them.
type Tables struct {
	// Card sets.
	All      CardSet            // every building
	AgeCards [4]CardSet         // the non-guild cards of ages 1 to 3 (entry 0 is empty)
	Guilds   CardSet            // the 7 guilds
	ByColor  [NumColors]CardSet // the cards of each colour

	// Per building.
	Color    [NumCards + 1]Color
	Age      [NumCards + 1]uint8
	Back     [NumCards + 1]Back
	Cost     [NumCards + 1]Cost
	Prod     [NumCards + 1][NumResources]uint8 // fixed production (brown and grey cards only)
	Choice   [NumCards + 1]ResMask             // flexible production: one unit from the mask per turn
	TradeFix [NumCards + 1]ResMask             // resources this card fixes at 1 coin per unit
	Shields  [NumCards + 1]uint8               // printed shields (Strategy's extra shield is not included)
	Sym      [NumCards + 1]Symbol              // science symbol
	VP       [NumCards + 1]uint8               // static victory points
	Coins    [NumCards + 1]uint8               // static coins gained on build
	ChainIn  [NumCards + 1]Chain               // the chain symbol that makes this card free
	ChainOut [NumCards + 1]Chain               // the chain symbol this card shows

	// Per chain symbol: the card that shows it and the card it makes free.
	ChainSource [NumChains]CardID
	ChainTarget [NumChains]CardID

	// Per wonder.
	WCost      [NumWonders + 1]Cost
	WChoice    [NumWonders + 1]ResMask // flexible production
	WShields   [NumWonders + 1]uint8
	WCoins     [NumWonders + 1]uint8 // coins gained on build
	WVP        [NumWonders + 1]uint8
	WPlayAgain [NumWonders + 1]bool  // the wonder's own extra turn (Theology is separate)
	WDestroy   [NumWonders + 1]Color // colour of the opponent's card to discard, or NoColor
	WRebuild   [NumWonders + 1]bool  // build a card from the discard pile for free
	WDraw      [NumWonders + 1]uint8 // tokens drawn from the box (the Great Library)
	WOppLoses  [NumWonders + 1]uint8 // coins the opponent loses

	// Per progress token.
	TokenCoins [NumTokens + 1]uint8 // coins gained when taken
	TokenVP    [NumTokens + 1]uint8 // static victory points
}

// T holds the tables of the current catalog.
var T Tables

// maxChoiceProducers bounds the flexible producers in the whole catalog. The
// engine's per-player cache (Player.Choice, engine.MaxChoice entries) must
// hold all of them, since one player can own every one.
const maxChoiceProducers = 4

func init() {
	loadTables()
	afterReplace(loadTables)
}

// loadTables rebuilds T from the current registry. It panics if the catalog
// breaks a structural assertion.
func loadTables() {
	t, err := buildTables(reg)
	if err != nil {
		panic(err)
	}
	T = *t
}

// buildTables derives the tables from r and checks the structural facts the
// engine depends on:
//
//   - every flexible producer offers exactly the raw goods or exactly the
//     manufactured goods (the fast payment method is optimal only then);
//   - at most maxChoiceProducers flexible producers exist in total;
//   - only brown cards produce fixed raw goods and only grey cards produce
//     fixed manufactured goods (fixed production is what raises the
//     opponent's trade price); wonders and tokens never do;
//   - each chain symbol is shown by at most one card and frees at most one;
//   - no item carries two effects of the same kind.
func buildTables(r *registry) (*Tables, error) {
	t := new(Tables)
	var errs []error
	fail := func(format string, args ...any) { errs = append(errs, fmt.Errorf("catalog: "+format, args...)) }
	producers := 0

	for i := 1; i < len(r.cards); i++ {
		d := &r.cards[i]
		if d.ID == 0 {
			continue
		}
		id := d.ID
		t.All.Add(id)
		t.ByColor[d.Color].Add(id)
		if d.IsGuild() {
			t.Guilds.Add(id)
		} else {
			t.AgeCards[d.Age].Add(id)
		}
		t.Color[id], t.Age[id], t.Back[id], t.Cost[id] = d.Color, d.Age, d.Back, d.Cost
		t.ChainIn[id], t.ChainOut[id] = d.ChainIn, d.ChainOut
		if d.ChainOut != NoChain {
			if prev := t.ChainSource[d.ChainOut]; prev != 0 {
				fail("chain %s is shown by both card %d and card %d", d.ChainOut, prev, id)
			}
			t.ChainSource[d.ChainOut] = id
		}
		if d.ChainIn != NoChain {
			if prev := t.ChainTarget[d.ChainIn]; prev != 0 {
				fail("chain %s frees both card %d and card %d", d.ChainIn, prev, id)
			}
			t.ChainTarget[d.ChainIn] = id
		}
		where := "card " + strconv.Itoa(int(id)) + " (" + d.Slug + ")"
		if err := distinctKinds(d.Effects); err != nil {
			fail("%s: %v", where, err)
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduce:
				class := RawGoods
				if d.Color == Grey {
					class = Manufactured
				}
				if (d.Color != Brown && d.Color != Grey) || !class.Has(e.Res) {
					fail("%s: a %s card cannot have fixed production of %s", where, d.Color, e.Res)
				}
				t.Prod[id][e.Res] += e.N
			case EffProduceChoice:
				if err := checkChoiceMask(e.Mask); err != nil {
					fail("%s: %v", where, err)
				}
				producers++
				t.Choice[id] = e.Mask
			case EffTradeFix:
				t.TradeFix[id] = e.Mask
			case EffShields:
				t.Shields[id] = e.N
			case EffScience:
				t.Sym[id] = e.Symbol
			case EffPoints:
				t.VP[id] = e.N
			case EffCoins:
				t.Coins[id] = e.N
			}
		}
	}

	for i := 1; i < len(r.wonders); i++ {
		d := &r.wonders[i]
		if d.ID == 0 {
			continue
		}
		id := d.ID
		where := "wonder " + d.Slug
		if err := distinctKinds(d.Effects); err != nil {
			fail("%s: %v", where, err)
		}
		t.WCost[id] = d.Cost
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduce:
				fail("%s: wonders cannot have fixed production", where)
			case EffProduceChoice:
				if err := checkChoiceMask(e.Mask); err != nil {
					fail("%s: %v", where, err)
				}
				producers++
				t.WChoice[id] = e.Mask
			case EffShields:
				t.WShields[id] = e.N
			case EffCoins:
				t.WCoins[id] = e.N
			case EffPoints:
				t.WVP[id] = e.N
			case EffPlayAgain:
				t.WPlayAgain[id] = true
			case EffDestroy:
				for c := Brown; c < NumColors; c++ {
					if e.Colors.Has(c) {
						t.WDestroy[id] = c
					}
				}
			case EffBuildFromDiscard:
				t.WRebuild[id] = true
			case EffDrawTokens:
				t.WDraw[id] = e.N
			case EffOpponentLosesCoins:
				t.WOppLoses[id] = e.N
			}
		}
	}

	for i := 1; i < len(r.tokens); i++ {
		d := &r.tokens[i]
		if d.ID == 0 {
			continue
		}
		where := "token " + d.Slug
		if err := distinctKinds(d.Effects); err != nil {
			fail("%s: %v", where, err)
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduce, EffProduceChoice:
				fail("%s: tokens cannot produce resources", where)
			case EffCoins:
				t.TokenCoins[d.ID] = e.N
			case EffPoints:
				t.TokenVP[d.ID] = e.N
			}
		}
	}

	if producers > maxChoiceProducers {
		fail("%d flexible producers, but a player can hold at most %d", producers, maxChoiceProducers)
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return t, nil
}

// checkChoiceMask accepts exactly the two resource classes.
func checkChoiceMask(m ResMask) error {
	if m != RawGoods && m != Manufactured {
		return fmt.Errorf("flexible production of %s is neither exactly the raw goods nor exactly the manufactured goods",
			resLetters(m, ","))
	}
	return nil
}

// distinctKinds rejects two effects of the same kind on one item.
func distinctKinds(fx []Effect) error {
	var seen [numEffectKinds]bool
	for _, e := range fx {
		if e.Kind >= numEffectKinds {
			continue // registration has rejected it already (Effect.Validate)
		}
		if seen[e.Kind] {
			return fmt.Errorf("two %s effects", e.Kind)
		}
		seen[e.Kind] = true
	}
	return nil
}

// RecordTable is one verification table: the catalog flattened into rows of
// strings, one row per item in id order. An empty cell means none or zero.
type RecordTable struct {
	Name   string // "cards", "wonders" or "tokens"
	Header []string
	Rows   [][]string
}

// The verification CSV headers. The columns, in order:
//
// cards: id (the reference numbering 1 to 73), name (English, without a
// leading "The"), age (1 to 3; guilds 3), back (age1, age2, age3, guild),
// color, coins (printed coin cost), W C S G P (printed resource units),
// produce (fixed production, a letter and a count such as S2), produce_choice
// (letters joined by "|", such as W|C|S), trade_fix (letters joined by "|"),
// shields, science (symbol name), vp (static points), coins_gain (static coins
// on build), coins_per (coins on build: the counted colours joined by "+", or
// "wonder", then ":n:scope", for example brown+grey:1:max or wonder:2:self),
// points_per (end-game points in the same form, plus coinsN for one point per
// N coins: coins3:1:max), chain_in and chain_out (chain symbol names).
//
// wonders: id, name, coins, W C S G P, vp, shields, coins_gain,
// produce_choice, play_again (1), destroy_color, build_from_discard (1),
// draw_tokens (the count), opponent_loses (coins).
//
// tokens: id, name, coins_gain, vp, vp_per_token, discount (blue or wonder),
// trade_income (1), science (scales), extra_shield (the count),
// wonders_play_again (1), chain_bonus (coins).
var (
	cardColumns = []string{"id", "name", "age", "back", "color", "coins", "W", "C", "S", "G", "P",
		"produce", "produce_choice", "trade_fix", "shields", "science", "vp", "coins_gain",
		"coins_per", "points_per", "chain_in", "chain_out"}
	wonderColumns = []string{"id", "name", "coins", "W", "C", "S", "G", "P", "vp", "shields", "coins_gain",
		"produce_choice", "play_again", "destroy_color", "build_from_discard", "draw_tokens", "opponent_loses"}
	tokenColumns = []string{"id", "name", "coins_gain", "vp", "vp_per_token", "discount", "trade_income",
		"science", "extra_shield", "wonders_play_again", "chain_bonus"}
)

// RecordTables flattens the current catalog into the three verification
// tables: cards, wonders and tokens, in that order. It fails if an item has
// an effect the schema has no column for, or two effects for one cell.
func RecordTables() ([]RecordTable, error) {
	var errs []error
	cards := RecordTable{Name: "cards", Header: cardColumns, Rows: [][]string{}}
	for d := range Cards() {
		row := newRow(cardColumns)
		row.set("id", strconv.Itoa(int(d.ID)))
		row.set("name", d.Name)
		row.set("age", strconv.Itoa(int(d.Age)))
		row.set("back", d.Back.String())
		row.set("color", d.Color.String())
		row.cost(d.Cost)
		if d.ChainIn != NoChain {
			row.set("chain_in", d.ChainIn.String())
		}
		if d.ChainOut != NoChain {
			row.set("chain_out", d.ChainOut.String())
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduce:
				row.set("produce", string(e.Res.Letter())+strconv.Itoa(int(e.N)))
			case EffProduceChoice:
				row.set("produce_choice", resLetters(e.Mask, "|"))
			case EffTradeFix:
				row.set("trade_fix", resLetters(e.Mask, "|"))
			case EffShields:
				row.set("shields", count(e.N))
			case EffScience:
				row.set("science", e.Symbol.String())
			case EffPoints:
				row.set("vp", count(e.N))
			case EffCoins:
				row.set("coins_gain", count(e.N))
			case EffCoinsPerCard:
				row.set("coins_per", perValue(colorNamesOf(e.Colors), e.N, e.Scope))
			case EffCoinsPerWonder:
				row.set("coins_per", perValue("wonder", e.N, e.Scope))
			case EffPointsPerCard:
				row.set("points_per", perValue(colorNamesOf(e.Colors), e.N, e.Scope))
			case EffPointsPerWonder:
				row.set("points_per", perValue("wonder", e.N, e.Scope))
			case EffPointsPerCoins:
				row.set("points_per", perValue("coins"+strconv.Itoa(int(e.N)), 1, e.Scope))
			default:
				row.unsupported(e)
			}
		}
		cards.Rows = append(cards.Rows, row.cells)
		errs = append(errs, row.errs("card "+d.Slug)...)
	}

	wonders := RecordTable{Name: "wonders", Header: wonderColumns, Rows: [][]string{}}
	for d := range Wonders() {
		row := newRow(wonderColumns)
		row.set("id", strconv.Itoa(int(d.ID)))
		row.set("name", d.Name)
		row.cost(d.Cost)
		for _, e := range d.Effects {
			switch e.Kind {
			case EffPoints:
				row.set("vp", count(e.N))
			case EffShields:
				row.set("shields", count(e.N))
			case EffCoins:
				row.set("coins_gain", count(e.N))
			case EffProduceChoice:
				row.set("produce_choice", resLetters(e.Mask, "|"))
			case EffPlayAgain:
				row.set("play_again", "1")
			case EffDestroy:
				row.set("destroy_color", colorNamesOf(e.Colors))
			case EffBuildFromDiscard:
				row.set("build_from_discard", "1")
			case EffDrawTokens:
				row.set("draw_tokens", count(e.N))
			case EffOpponentLosesCoins:
				row.set("opponent_loses", count(e.N))
			default:
				row.unsupported(e)
			}
		}
		wonders.Rows = append(wonders.Rows, row.cells)
		errs = append(errs, row.errs("wonder "+d.Slug)...)
	}

	tokens := RecordTable{Name: "tokens", Header: tokenColumns, Rows: [][]string{}}
	for d := range Tokens() {
		row := newRow(tokenColumns)
		row.set("id", strconv.Itoa(int(d.ID)))
		row.set("name", d.Name)
		for _, e := range d.Effects {
			switch e.Kind {
			case EffCoins:
				row.set("coins_gain", count(e.N))
			case EffPoints:
				row.set("vp", count(e.N))
			case EffPointsPerToken:
				row.set("vp_per_token", count(e.N))
			case EffDiscount:
				row.set("discount", e.Target.String())
			case EffTradeIncome:
				row.set("trade_income", "1")
			case EffScience:
				row.set("science", e.Symbol.String())
			case EffExtraShield:
				row.set("extra_shield", count(e.N))
			case EffWondersPlayAgain:
				row.set("wonders_play_again", "1")
			case EffChainBonus:
				row.set("chain_bonus", count(e.N))
			default:
				row.unsupported(e)
			}
		}
		tokens.Rows = append(tokens.Rows, row.cells)
		errs = append(errs, row.errs("token "+d.Slug)...)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return []RecordTable{cards, wonders, tokens}, nil
}

// WriteCSV writes the table as UTF-8 CSV: the header row, then one row per
// item, comma separated, each line ending in a bare newline.
func (t RecordTable) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(t.Header); err != nil {
		return err
	}
	return cw.WriteAll(t.Rows) // WriteAll flushes
}

// record builds one row, remembering every problem for errs.
type record struct {
	header   []string
	cells    []string
	problems []string
}

func newRow(header []string) *record {
	return &record{header: header, cells: make([]string, len(header))}
}

// set fills the named cell. Filling a cell twice, or a cell the header does
// not have, is a problem.
func (r *record) set(col, v string) {
	for i, h := range r.header {
		if h == col {
			if r.cells[i] != "" {
				r.problems = append(r.problems, fmt.Sprintf("column %s set twice (%q and %q)", col, r.cells[i], v))
				return
			}
			r.cells[i] = v
			return
		}
	}
	r.problems = append(r.problems, "no column "+col)
}

// cost fills the coins and the five resource columns.
func (r *record) cost(c Cost) {
	if c.Coins > 0 {
		r.set("coins", strconv.Itoa(int(c.Coins)))
	}
	for res := range NumResources {
		if c.Res[res] > 0 {
			r.set(string(res.Letter()), strconv.Itoa(int(c.Res[res])))
		}
	}
}

func (r *record) unsupported(e Effect) {
	r.problems = append(r.problems, "effect "+e.String()+" has no column")
}

func (r *record) errs(where string) []error {
	out := make([]error, 0, len(r.problems))
	for _, p := range r.problems {
		out = append(out, fmt.Errorf("catalog: verification record of %s: %s", where, p))
	}
	return out
}

// count renders a positive count; zero is the empty cell.
func count(n uint8) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(int(n))
}

// perValue renders a "per" cell such as yellow:1:self.
func perValue(what string, n uint8, s Scope) string {
	return what + ":" + strconv.Itoa(int(n)) + ":" + s.String()
}

// resLetters lists a mask's resources as letters in W C S G P order.
func resLetters(m ResMask, sep string) string {
	var parts []string
	for res := range NumResources {
		if m.Has(res) {
			parts = append(parts, string(res.Letter()))
		}
	}
	return strings.Join(parts, sep)
}

// colorNamesOf lists a mask's colours in enum order, joined by "+".
func colorNamesOf(m ColorMask) string {
	var parts []string
	for c := Brown; c < NumColors; c++ {
		if m.Has(c) {
			parts = append(parts, c.String())
		}
	}
	return strings.Join(parts, "+")
}
