package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// csvOutputs renders the verification tables, all three or just one.
func csvOutputs(only string) ([]output, error) {
	tbls, err := catalog.RecordTables()
	if err != nil {
		return nil, err
	}
	var outs []output
	for _, t := range tbls {
		if only != "" && t.Name != only {
			continue
		}
		var buf bytes.Buffer
		if err := t.WriteCSV(&buf); err != nil {
			return nil, err
		}
		outs = append(outs, output{t.Name + ".csv", buf.Bytes()})
	}
	return outs, nil
}

// The presentation catalog. It follows the wire conventions: every field has
// an explicit json tag, lists are never null, and only pointer fields are
// optional (omitempty).

// jsonSchema is the version of the presentation catalog's shape.
const jsonSchema = 1

type jsonCatalog struct {
	Schema  int          `json:"schema"`
	Cards   []jsonCard   `json:"cards"`
	Wonders []jsonWonder `json:"wonders"`
	Tokens  []jsonToken  `json:"tokens"`
}

type jsonCost struct {
	Coins   int `json:"coins"`
	Wood    int `json:"wood"`
	Clay    int `json:"clay"`
	Stone   int `json:"stone"`
	Glass   int `json:"glass"`
	Papyrus int `json:"papyrus"`
}

type jsonCard struct {
	ID       int          `json:"id"`
	Slug     string       `json:"slug"`
	Name     string       `json:"name"`
	AltNames []string     `json:"altNames"`
	Age      int          `json:"age"`
	Back     string       `json:"back"`
	Color    string       `json:"color"`
	Cost     jsonCost     `json:"cost"`
	Effects  []jsonEffect `json:"effects"`
	// ChainIn is the chain symbol that makes this card free, and ChainFrom
	// the slug of the card that shows it.
	ChainIn   *string `json:"chainIn,omitempty"`
	ChainFrom *string `json:"chainFrom,omitempty"`
	// ChainOut is the chain symbol this card shows, and ChainTo the slug of
	// the card it makes free.
	ChainOut *string `json:"chainOut,omitempty"`
	ChainTo  *string `json:"chainTo,omitempty"`
}

type jsonWonder struct {
	ID      int          `json:"id"`
	Slug    string       `json:"slug"`
	Name    string       `json:"name"`
	Cost    jsonCost     `json:"cost"`
	Effects []jsonEffect `json:"effects"`
}

type jsonToken struct {
	ID      int          `json:"id"`
	Slug    string       `json:"slug"`
	Name    string       `json:"name"`
	Effects []jsonEffect `json:"effects"`
}

// jsonEffect is one effect. Kind is the catalog's kebab-case effect kind;
// the other fields appear only when the kind uses them. N is the kind's count:
// units, shields, points or coins, except that for points-per-coins it is the
// number of coins that make one point.
type jsonEffect struct {
	Kind      string    `json:"kind"`
	N         *int      `json:"n,omitempty"`
	Resource  *string   `json:"resource,omitempty"`  // produce
	Resources *[]string `json:"resources,omitempty"` // produce-choice, trade-fix
	Colors    *[]string `json:"colors,omitempty"`    // per-card counts, destroy
	Symbol    *string   `json:"symbol,omitempty"`    // science
	Scope     *string   `json:"scope,omitempty"`     // "self" or "max"
	Target    *string   `json:"target,omitempty"`    // discount: "blue" or "wonder"
}

func ptr[T any](v T) *T { return &v }

func toJSONCost(c catalog.Cost) jsonCost {
	return jsonCost{
		Coins: int(c.Coins), Wood: int(c.Res[catalog.Wood]), Clay: int(c.Res[catalog.Clay]),
		Stone: int(c.Res[catalog.Stone]), Glass: int(c.Res[catalog.Glass]), Papyrus: int(c.Res[catalog.Papyrus]),
	}
}

func toJSONEffects(fx []catalog.Effect) []jsonEffect {
	out := make([]jsonEffect, 0, len(fx))
	for _, e := range fx {
		j := jsonEffect{Kind: e.Kind.String()}
		// Every field an effect kind uses is non-zero, except Res: Wood is
		// resource zero, so Res counts only for the one kind that uses it.
		if e.N != 0 {
			j.N = ptr(int(e.N))
		}
		if e.Kind == catalog.EffProduce {
			j.Resource = ptr(e.Res.String())
		}
		if e.Mask != 0 {
			j.Resources = ptr(resourceNames(e.Mask))
		}
		if e.Colors != 0 {
			j.Colors = ptr(colorNames(e.Colors))
		}
		if e.Symbol != catalog.NoSymbol {
			j.Symbol = ptr(e.Symbol.String())
		}
		if e.Scope != catalog.NoScope {
			j.Scope = ptr(e.Scope.String())
		}
		if e.Target != catalog.NoTarget {
			j.Target = ptr(e.Target.String())
		}
		out = append(out, j)
	}
	return out
}

func resourceNames(m catalog.ResMask) []string {
	out := []string{}
	for r := range catalog.NumResources {
		if m.Has(r) {
			out = append(out, r.String())
		}
	}
	return out
}

func colorNames(m catalog.ColorMask) []string {
	out := []string{}
	for c := catalog.Brown; c < catalog.NumColors; c++ {
		if m.Has(c) {
			out = append(out, c.String())
		}
	}
	return out
}

// presentationJSON renders the client presentation catalog, indented, with a
// final newline.
func presentationJSON() ([]byte, error) {
	cat := jsonCatalog{Schema: jsonSchema, Cards: []jsonCard{}, Wonders: []jsonWonder{}, Tokens: []jsonToken{}}
	for d := range catalog.Cards() {
		c := jsonCard{
			ID: int(d.ID), Slug: d.Slug, Name: d.Name, AltNames: append([]string{}, d.AltNames...),
			Age: int(d.Age), Back: d.Back.String(), Color: d.Color.String(),
			Cost: toJSONCost(d.Cost), Effects: toJSONEffects(d.Effects),
		}
		if d.ChainIn != catalog.NoChain {
			c.ChainIn = ptr(d.ChainIn.String())
			if src := catalog.Card(catalog.T.ChainSource[d.ChainIn]); src != nil {
				c.ChainFrom = ptr(src.Slug)
			}
		}
		if d.ChainOut != catalog.NoChain {
			c.ChainOut = ptr(d.ChainOut.String())
			if dst := catalog.Card(catalog.T.ChainTarget[d.ChainOut]); dst != nil {
				c.ChainTo = ptr(dst.Slug)
			}
		}
		cat.Cards = append(cat.Cards, c)
	}
	for d := range catalog.Wonders() {
		cat.Wonders = append(cat.Wonders, jsonWonder{
			ID: int(d.ID), Slug: d.Slug, Name: d.Name, Cost: toJSONCost(d.Cost), Effects: toJSONEffects(d.Effects),
		})
	}
	for d := range catalog.Tokens() {
		cat.Tokens = append(cat.Tokens, jsonToken{ID: int(d.ID), Slug: d.Slug, Name: d.Name, Effects: toJSONEffects(d.Effects)})
	}
	b, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// The prose catalog: one paragraph per item, one sentence per effect, in our
// own words. It is written for a reviewer who checks the data's meaning
// against the rules, so it states every count, scope and timing explicitly.

// itemKind says what an effect belongs to, which changes some wording.
type itemKind uint8

const (
	itemBuilding itemKind = iota
	itemWonder
	itemToken
)

func proseCatalog() []byte {
	var b bytes.Buffer
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("Ajayeb catalog in plain English: every building, wonder and progress token, one line per effect.\n")
	p("Generated by ajayeb-gen catalog -format prose from the engine catalog; do not edit.\n")

	p("\nBUILDINGS\n")
	for d := range catalog.Cards() {
		kind := "building"
		if d.IsGuild() {
			kind = "guild"
		}
		p("\n%d. %s (%s): Age %s %s %s, %s back. Cost: %s.\n",
			d.ID, d.Name, d.Slug, romanAge(d.Age), d.Color, kind, d.Back, costProse(d.Cost))
		if len(d.AltNames) > 0 {
			p("   - Also printed as %s.\n", joinList(d.AltNames, "or"))
		}
		for _, e := range d.Effects {
			p("   - %s\n", effectProse(e, itemBuilding, d.Color))
		}
		if d.ChainIn != catalog.NoChain {
			p("   - Free to construct if you already own %s, which shows the %s chain symbol.\n",
				cardName(catalog.T.ChainSource[d.ChainIn]), d.ChainIn)
		}
		if d.ChainOut != catalog.NoChain {
			p("   - Shows the %s chain symbol, which makes %s free to construct later.\n",
				d.ChainOut, cardName(catalog.T.ChainTarget[d.ChainOut]))
		}
	}

	p("\nWONDERS\n")
	for d := range catalog.Wonders() {
		p("\n%d. %s (%s): wonder. Cost: %s.\n", d.ID, d.Name, d.Slug, costProse(d.Cost))
		if len(d.Effects) == 0 {
			p("   - No effect.\n")
		}
		for _, e := range d.Effects {
			p("   - %s\n", effectProse(e, itemWonder, catalog.NoColor))
		}
	}

	p("\nPROGRESS TOKENS\n")
	for d := range catalog.Tokens() {
		p("\n%d. %s (%s): progress token.\n", d.ID, d.Name, d.Slug)
		for _, e := range d.Effects {
			p("   - %s\n", effectProse(e, itemToken, catalog.NoColor))
		}
	}
	return b.Bytes()
}

func cardName(id catalog.CardID) string {
	if d := catalog.Card(id); d != nil {
		return d.Name
	}
	return "an unregistered card (" + strconv.Itoa(int(id)) + ")"
}

func romanAge(age uint8) string {
	if age >= 1 && age <= 3 {
		return [...]string{"I", "II", "III"}[age-1]
	}
	return strconv.Itoa(int(age))
}

// plural writes "1 coin", "3 coins" and so on.
func plural(n uint8, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(int(n)) + " " + many
}

// joinList writes "a", "a or b", or "a, b or c" (with "and" likewise).
func joinList(items []string, conj string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conj + " " + items[len(items)-1]
}

func costProse(c catalog.Cost) string {
	var parts []string
	if c.Coins > 0 {
		parts = append(parts, plural(c.Coins, "coin", "coins"))
	}
	for r := range catalog.NumResources {
		if n := c.Res[r]; n > 0 {
			parts = append(parts, strconv.Itoa(int(n))+" "+r.String())
		}
	}
	if len(parts) == 0 {
		return "free"
	}
	return joinList(parts, "and")
}

// countedCards describes the cards a per-card effect counts, and the city it
// counts them in, for example "each yellow card in your city" or "each brown
// and each grey card in the city that has the most brown and grey cards
// together (yours or your opponent's)".
func countedCards(e catalog.Effect, own catalog.Color) string {
	names := colorNames(e.Colors)
	var each []string
	for _, n := range names {
		each = append(each, "each "+n)
	}
	s := joinList(each, "and") + " card"
	switch e.Scope {
	case catalog.ScopeSelf:
		s += " in your city"
		if e.Colors.Has(own) {
			s += ", this card included"
		}
	case catalog.ScopeMaxCity:
		if len(names) > 1 {
			s += " in the city that has the most " + joinList(names, "and") + " cards together (yours or your opponent's)"
		} else {
			s += " in the city that has the most " + names[0] + " cards (yours or your opponent's)"
		}
	}
	return s
}

// wonderCity names the city a per-wonder effect counts.
func wonderCity(s catalog.Scope) string {
	if s == catalog.ScopeMaxCity {
		return "built in the city that has built the most wonders (yours or your opponent's)"
	}
	return "you have built"
}

// effectProse renders one effect as a sentence. own is the colour of the
// building carrying it (NoColor for wonders and tokens).
func effectProse(e catalog.Effect, item itemKind, own catalog.Color) string {
	when := "when built"
	if item == itemToken {
		when = "when you take this token"
	}
	switch e.Kind {
	case catalog.EffProduce:
		return fmt.Sprintf("Produces %d %s every turn. Each unit also raises by 1 coin the price your opponent pays to buy %s.",
			e.N, e.Res, e.Res)
	case catalog.EffProduceChoice:
		return fmt.Sprintf("Produces 1 unit of your choice of %s every turn. This never changes your opponent's trading prices.",
			joinList(resourceNames(e.Mask), "or"))
	case catalog.EffTradeFix:
		return fmt.Sprintf("From now on you buy %s from the bank at 1 coin per unit.", joinList(resourceNames(e.Mask), "and"))
	case catalog.EffShields:
		return fmt.Sprintf("%s: moves the conflict pawn %s toward your opponent's capital.",
			plural(e.N, "shield", "shields"), plural(e.N, "space", "spaces"))
	case catalog.EffScience:
		return fmt.Sprintf("Shows the %s science symbol.", e.Symbol)
	case catalog.EffPoints:
		return fmt.Sprintf("Worth %s at the end of the game.", plural(e.N, "victory point", "victory points"))
	case catalog.EffCoins:
		return fmt.Sprintf("You gain %s once, %s.", plural(e.N, "coin", "coins"), when)
	case catalog.EffCoinsPerCard:
		return fmt.Sprintf("Once, %s: you gain %s for %s.", when, plural(e.N, "coin", "coins"), countedCards(e, own))
	case catalog.EffCoinsPerWonder:
		return fmt.Sprintf("Once, %s: you gain %s for each wonder %s.", when, plural(e.N, "coin", "coins"), wonderCity(e.Scope))
	case catalog.EffPointsPerCard:
		return fmt.Sprintf("At the end of the game: worth %s for %s.",
			plural(e.N, "victory point", "victory points"), countedCards(e, own))
	case catalog.EffPointsPerWonder:
		return fmt.Sprintf("At the end of the game: worth %s for each wonder %s.",
			plural(e.N, "victory point", "victory points"), wonderCity(e.Scope))
	case catalog.EffPointsPerCoins:
		city := "your city"
		if e.Scope == catalog.ScopeMaxCity {
			city = "the city that has the most coins (yours or your opponent's)"
		}
		return fmt.Sprintf("At the end of the game: worth 1 victory point for every full %s in %s.",
			plural(e.N, "coin", "coins"), city)
	case catalog.EffPointsPerToken:
		return fmt.Sprintf("At the end of the game: worth %s for each progress token you own, this one included.",
			plural(e.N, "victory point", "victory points"))
	case catalog.EffPlayAgain:
		return "You take another turn right after this one."
	case catalog.EffDestroy:
		return fmt.Sprintf("You choose one %s card in your opponent's city and put it on the discard pile (nothing happens if they have no such card).",
			joinList(colorNames(e.Colors), "or"))
	case catalog.EffBuildFromDiscard:
		return "You choose one card from the discard pile and construct it for free (nothing happens if the pile is empty)."
	case catalog.EffDrawTokens:
		return fmt.Sprintf("You draw %d of the progress tokens set aside at setup, keep 1 of them and put the others back.", e.N)
	case catalog.EffOpponentLosesCoins:
		return fmt.Sprintf("Your opponent loses %s to the bank, never going below zero.", plural(e.N, "coin", "coins"))
	case catalog.EffDiscount:
		what := "blue card"
		if e.Target == catalog.ForWonders {
			what = "wonder"
		}
		return fmt.Sprintf("Every %s you construct from now on costs %s fewer, of your choice.",
			what, plural(e.N, "resource unit", "resource units"))
	case catalog.EffTradeIncome:
		return "From now on, whenever your opponent buys resources from the bank, you receive the coins they pay for them (but not their printed coin costs)."
	case catalog.EffExtraShield:
		return fmt.Sprintf("Every red building you construct from now on gives %s more.", plural(e.N, "shield", "shields"))
	case catalog.EffWondersPlayAgain:
		return "Every wonder you construct from now on gives you another turn (a wonder that already gives one still gives only one)."
	case catalog.EffChainBonus:
		return fmt.Sprintf("From now on, whenever a chain symbol lets you construct a building for free, you gain %s.",
			plural(e.N, "coin", "coins"))
	}
	return "Unknown effect " + e.String() + "."
}
