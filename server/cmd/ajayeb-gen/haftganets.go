package main

import (
	"encoding/json"
	"fmt"
	"strings"

	hg "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// This file writes Haftgane's presentation catalog as a TypeScript module
// for the web client, web/src/lib/wire/haftgane-catalog.gen.ts: every card
// design and every board's two sides with the data a screen draws (Age,
// colour, printed cost, chains and declarative effects), keyed by slug, in
// id order. It is display data only, as codes: no names or prose (the
// client's message tables name things), and nothing a client could price
// or judge legality with beyond what a printed card shows. The server's
// view still carries every price and every legal choice.
//
// Only Haftgane's lazy chunk imports the module, so Home and Duel never
// download it. Every item is one line of compact JSON, as in catalog.gen.ts.

// defaultHaftganeCatalogTSOut is relative to server/, like defaultCatalogTSOut.
const defaultHaftganeCatalogTSOut = "../web/src/lib/wire/haftgane-catalog.gen.ts"

// hgChainSlugs names the catalog's chain values, in enum order from 1: each
// is the slug of the design the chain makes free, as the constant's name
// says (catalog/chains.go).
var hgChainSlugs = [hg.NumChains]string{
	"archery-range", "stables", "aqueduct", "courthouse", "statue", "caravansery",
	"forum", "dispensary", "laboratory", "library", "circus", "fortifications",
	"siege-workshop", "gardens", "pantheon", "senate", "arena", "haven",
	"lighthouse", "academy", "lodge", "observatory", "study", "university",
}

var hgFreeConditions = [hg.NumFreeConditions]string{"", "first-colour", "first-turn", "last-turn"}

type hgJSONCost struct {
	Coins int      `json:"coins,omitempty"`
	Res   []string `json:"res,omitempty"`
}

type hgJSONEffect struct {
	Kind    string   `json:"kind"`
	N       int      `json:"n,omitempty"`
	Res     []string `json:"res,omitempty"`
	Colours []string `json:"colours,omitempty"`
	Scope   []string `json:"scope,omitempty"`
	Symbol  string   `json:"symbol,omitempty"`
	ForSale bool     `json:"forSale,omitempty"`
	When    string   `json:"when,omitempty"`
}

type hgJSONDesign struct {
	Age      int            `json:"age"`
	Colour   string         `json:"colour"`
	Cost     hgJSONCost     `json:"cost"`
	ChainIn  string         `json:"chainIn,omitempty"`
	ChainOut []string       `json:"chainOut,omitempty"`
	Effects  []hgJSONEffect `json:"effects"`
}

type hgJSONStage struct {
	Cost    hgJSONCost     `json:"cost"`
	Effects []hgJSONEffect `json:"effects"`
}

type hgJSONBoard struct {
	Start string        `json:"start"`
	Day   []hgJSONStage `json:"day"`
	Night []hgJSONStage `json:"night"`
}

func hgChainSlug(c hg.Chain) (string, error) {
	if c < 1 || c > hg.NumChains {
		return "", fmt.Errorf("chain %d out of range", c)
	}
	return hgChainSlugs[c-1], nil
}

// hgCostJSON lists a cost's resource units one per pip, in resource order.
func hgCostJSON(c hg.Cost) hgJSONCost {
	out := hgJSONCost{Coins: int(c.Coins)}
	for r := hg.Resource(0); r < hg.NumResources; r++ {
		for range c.Res[r] {
			out.Res = append(out.Res, r.String())
		}
	}
	return out
}

func hgScopeJSON(s hg.SeatScope) []string {
	var out []string
	for _, p := range []struct {
		bit  hg.SeatScope
		name string
	}{{hg.Self, "self"}, {hg.Left, "left"}, {hg.Right, "right"}} {
		if s&p.bit != 0 {
			out = append(out, p.name)
		}
	}
	return out
}

func hgColoursJSON(m hg.ColorMask) []string {
	var out []string
	for c := hg.Brown; c < hg.NumColors; c++ {
		if m.Has(c) {
			out = append(out, c.String())
		}
	}
	return out
}

func hgEffectJSON(e hg.Effect) (hgJSONEffect, error) {
	out := hgJSONEffect{Kind: e.Kind.String(), N: int(e.N), ForSale: e.ForSale}
	if out.Kind == "" || e.Kind == hg.EffNone {
		return out, fmt.Errorf("effect kind %d has no name", e.Kind)
	}
	if e.Kind == hg.EffProduce {
		out.Res = []string{e.Res.String()}
	} else {
		out.Res = haftganeResources(e.Mask)
	}
	out.Colours = hgColoursJSON(e.Colors)
	out.Scope = hgScopeJSON(e.Scope)
	if e.Symbol != hg.NoSymbol {
		out.Symbol = e.Symbol.String()
	}
	if e.When != hg.NoFreeCondition {
		if e.When >= hg.NumFreeConditions {
			return out, fmt.Errorf("free condition %d out of range", e.When)
		}
		out.When = hgFreeConditions[e.When]
	}
	return out, nil
}

func hgEffectsJSON(fx []hg.Effect) ([]hgJSONEffect, error) {
	out := []hgJSONEffect{}
	for _, e := range fx {
		j, err := hgEffectJSON(e)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

func hgStagesJSON(side hg.SideDef) ([]hgJSONStage, error) {
	out := []hgJSONStage{}
	for i := range int(side.NumStages) {
		fx, err := hgEffectsJSON(side.Stages[i].Effects)
		if err != nil {
			return nil, err
		}
		out = append(out, hgJSONStage{Cost: hgCostJSON(side.Stages[i].Cost), Effects: fx})
	}
	return out, nil
}

// hgSlugLiteral refuses a slug that a single-quoted literal cannot hold.
func hgSlugLiteral(s string) error {
	if s == "" || strings.ContainsAny(s, `'"\`+"\n") {
		return fmt.Errorf("slug %q cannot be written as a TypeScript literal", s)
	}
	return nil
}

// haftganeCatalogTS renders haftgane-catalog.gen.ts.
func haftganeCatalogTS() ([]byte, error) {
	var designs, boards strings.Builder
	var slugs []string
	for id := hg.DesignID(1); id <= hg.NumDesigns; id++ {
		d, ok := hg.DesignByID(id)
		if !ok {
			return nil, fmt.Errorf("design %d is missing", id)
		}
		if err := hgSlugLiteral(d.Slug); err != nil {
			return nil, err
		}
		fx, err := hgEffectsJSON(d.Effects)
		if err != nil {
			return nil, fmt.Errorf("design %s: %w", d.Slug, err)
		}
		j := hgJSONDesign{Age: int(d.Age), Colour: d.Color.String(), Cost: hgCostJSON(d.Cost), Effects: fx}
		if d.ChainIn != 0 {
			if j.ChainIn, err = hgChainSlug(d.ChainIn); err != nil {
				return nil, fmt.Errorf("design %s: %w", d.Slug, err)
			}
		}
		for _, c := range d.ChainOut {
			if c == 0 {
				continue
			}
			s, err := hgChainSlug(c)
			if err != nil {
				return nil, fmt.Errorf("design %s: %w", d.Slug, err)
			}
			j.ChainOut = append(j.ChainOut, s)
		}
		data, err := json.Marshal(j)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&designs, "  '%s': %s,\n", d.Slug, data)
		slugs = append(slugs, d.Slug)
	}
	var boardSlugs []string
	for id := hg.BoardID(1); id <= hg.NumBoards; id++ {
		b, ok := hg.BoardByID(id)
		if !ok {
			return nil, fmt.Errorf("board %d is missing", id)
		}
		if err := hgSlugLiteral(b.Slug); err != nil {
			return nil, err
		}
		day, err := hgStagesJSON(b.Sides[0])
		if err != nil {
			return nil, fmt.Errorf("board %s: %w", b.Slug, err)
		}
		night, err := hgStagesJSON(b.Sides[1])
		if err != nil {
			return nil, fmt.Errorf("board %s: %w", b.Slug, err)
		}
		data, err := json.Marshal(hgJSONBoard{Start: b.Start.String(), Day: day, Night: night})
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&boards, "  '%s': %s,\n", b.Slug, data)
		boardSlugs = append(boardSlugs, b.Slug)
	}

	var kinds []string
	for k := hg.EffectKind(1); k < hg.NumEffectKinds; k++ {
		kinds = append(kinds, k.String())
	}
	var resources []string
	for r := hg.Resource(0); r < hg.NumResources; r++ {
		resources = append(resources, r.String())
	}

	var b strings.Builder
	b.WriteString(`// Code generated by ajayeb-gen. DO NOT EDIT.
//
// Haftgane's presentation catalog: every card design and both sides of every
// board, with what a screen draws (Age, colour, printed cost, chains and
// effects as codes), keyed by slug in catalog order. It is display data, not
// rules: the server's view carries every price, discount and legal choice.
// Names live in the message tables (hg.card.<slug>, hg.board.<slug>). The
// same data as ` + "`ajayeb-gen catalog -game haftgane -format ts`" + `. Only
// Haftgane's lazy chunk imports it.

`)
	fmt.Fprintf(&b, "export type HgResource = %s;\n", tsUnion(resources))
	fmt.Fprintf(&b, "export type HgColour = %s;\n", tsUnion(hgColoursJSON(hg.AllColors)))
	b.WriteString("export type HgSymbol = 'compass' | 'gear' | 'tablet';\n")
	fmt.Fprintf(&b, "export type HgEffectKind = %s;\n", tsUnion(kinds))
	fmt.Fprintf(&b, "export type HgChain = %s;\n", tsUnion(hgChainSlugs[:]))
	fmt.Fprintf(&b, "export type HgFreeWhen = %s;\n\n", tsUnion(hgFreeConditions[1:]))
	b.WriteString(`/** A printed cost: coins, and one resource per unit, in resource order. */
export interface HgCost {
  readonly coins?: number;
  readonly res?: readonly HgResource[];
}

/**
 * One printed effect. n is its count (units, shields, coins, points or a
 * discount's coins); res the resource it makes (produce), the choice it
 * makes one of (produce-choice) or the family it discounts; colours and
 * scope the cards and cities a per-card or per-stage effect counts; when a
 * free build's condition. forSale marks production neighbours may buy.
 */
export interface HgEffect {
  readonly kind: HgEffectKind;
  readonly n?: number;
  readonly res?: readonly HgResource[];
  readonly colours?: readonly HgColour[];
  readonly scope?: readonly ('self' | 'left' | 'right')[];
  readonly symbol?: HgSymbol;
  readonly forSale?: boolean;
  readonly when?: HgFreeWhen;
}

/** A card design. chainIn makes it free; chainOut is what it makes free. */
export interface HgDesign {
  readonly age: 1 | 2 | 3;
  readonly colour: HgColour;
  readonly cost: HgCost;
  readonly chainIn?: HgChain;
  readonly chainOut?: readonly HgChain[];
  readonly effects: readonly HgEffect[];
}

/** One wonder stage. */
export interface HgStage {
  readonly cost: HgCost;
  readonly effects: readonly HgEffect[];
}

/** A board: its starting resource and the stages of each side. */
export interface HgBoard {
  readonly start: HgResource;
  readonly day: readonly HgStage[];
  readonly night: readonly HgStage[];
}

`)
	b.WriteString("/** Every design's slug, in catalog order. */\n")
	slugList(&b, "hgDesignSlugs", slugs)
	b.WriteString("export type HgDesignSlug = (typeof hgDesignSlugs)[number];\n\n")
	b.WriteString("/** Every board's slug, in catalog order. */\n")
	slugList(&b, "hgBoardSlugs", boardSlugs)
	b.WriteString("export type HgBoardSlug = (typeof hgBoardSlugs)[number];\n\n")
	b.WriteString("export const hgDesigns: { readonly [S in HgDesignSlug]: HgDesign } = {\n")
	b.WriteString(designs.String())
	b.WriteString("};\n\n")
	b.WriteString("export const hgBoards: { readonly [S in HgBoardSlug]: HgBoard } = {\n")
	b.WriteString(boards.String())
	b.WriteString("};\n")
	return []byte(b.String()), nil
}

// writeHaftganeCatalogTS writes the module at path, leaving an identical
// file untouched, as writeCatalogTS does.
func writeHaftganeCatalogTS(path string) error {
	data, err := haftganeCatalogTS()
	if err != nil {
		return err
	}
	return writeIfChanged(path, data)
}
