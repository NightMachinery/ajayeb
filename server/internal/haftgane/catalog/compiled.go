package catalog

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// This file derives two things from the registered catalog:
//
//   - T, read-only lookup tables indexed by id, built when the catalog is
//     registered and again whenever ReplaceForTest swaps it;
//   - the record tables: the catalog flattened into the schema of the blind
//     verification tables of M13, plus our ids. ajayeb-gen catalog -game
//     haftgane writes them, golden_test.go pins them in
//     testdata/*.golden.csv, and verify_test.go diffs them against the blind
//     tables.

// Tables are lookup tables derived from the catalog. Arrays indexed by an id
// have an unused entry 0. Never write to them.
type Tables struct {
	All     DesignSet               // every design
	Ages    [4]DesignSet            // the non-guild designs of Ages 1 to 3 (entry 0 is empty)
	Guilds  DesignSet               // the guilds
	ByColor [NumColors]DesignSet    // the designs of each colour
	ByName  [NumNames + 1]DesignSet // the designs of each name: one, or a pair of twins

	// Per chain symbol: the one design it frees and the designs that provide
	// it (the second provider is 0 for every symbol but one).
	ChainTarget    [NumChains + 1]DesignID
	ChainProviders [NumChains + 1][2]DesignID
	// Per design: the symbols it provides, as Chain.Bit values.
	ChainOutBits [NumDesigns + 1]uint32
}

func compile(r *registry) Tables {
	var t Tables
	for _, d := range r.designs {
		if d.ID == 0 {
			continue
		}
		t.All.Add(d.ID)
		if d.Color == Purple {
			t.Guilds.Add(d.ID)
		} else {
			t.Ages[d.Age].Add(d.ID)
		}
		t.ByColor[d.Color].Add(d.ID)
		t.ByName[d.NameID].Add(d.ID)
		if d.ChainIn != 0 {
			t.ChainTarget[d.ChainIn] = d.ID
		}
		for _, c := range d.ChainOut {
			if c == 0 {
				continue
			}
			t.ChainOutBits[d.ID] |= c.Bit()
			if t.ChainProviders[c][0] == 0 {
				t.ChainProviders[c][0] = d.ID
			} else {
				t.ChainProviders[c][1] = d.ID
			}
		}
	}
	return t
}

// CopiesAt is the number of copies of a design in the deck for the given
// number of players: the copies whose marker is at most that number. Guilds
// have no markers; setup draws them separately.
func CopiesAt(id DesignID, players uint8) uint8 {
	d, ok := DesignByID(id)
	if !ok {
		return 0
	}
	n := uint8(0)
	for _, m := range d.Copies {
		if m != 0 && m <= players {
			n++
		}
	}
	return n
}

// RecordTable is one record table: a header row and the item rows.
type RecordTable struct {
	Name   string
	Header []string
	Rows   [][]string
}

// WriteCSV writes the table as CSV with bare newlines.
func (t RecordTable) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(t.Header); err != nil {
		return err
	}
	if err := cw.WriteAll(t.Rows); err != nil {
		return err
	}
	return cw.Error()
}

// The record table headers. The columns after our ids are those of the M13
// blind tables, in the same notation (docs/haftgane/data-rulings.md), with
// the engine's resource name cloth where M13 wrote textile:
//
// cards, one row per design: id, slug, age, name, name_id, colour,
// player_counts (the copy markers joined by ";", or "guild"), cost ("free",
// "coins:N", or resources such as "wood:1 clay:2"), chain_in (the symbol
// that frees it), chain_from (the names of the designs providing that
// symbol, joined by ";"), chain_out (the symbols it provides, joined by
// ";"), chain_to (the names of the designs they free), effects (the terms of
// every effect, sorted, joined by ";"; see EffectTerms).
//
// wonders, one row per stage: board, side, starting_resource, stage (from
// 1), cost, effects.
//
// boards: id, slug, name, wonder, starting_resource, day_stages,
// night_stages.
//
// The notation has no for-sale flag: validation fixes it by holder (brown
// and grey production and the brown choices are for sale; yellow and stage
// production is not), so the tables lose nothing.
var (
	DesignColumns = []string{"id", "slug", "age", "name", "name_id", "colour", "player_counts", "cost",
		"chain_in", "chain_from", "chain_out", "chain_to", "effects"}
	WonderColumns = []string{"board", "side", "starting_resource", "stage", "cost", "effects"}
	BoardColumns  = []string{"id", "slug", "name", "wonder", "starting_resource", "day_stages", "night_stages"}
)

// tableResources is the resource order of the M13 notation.
var tableResources = [NumResources]Resource{Wood, Stone, Clay, Ore, Glass, Papyrus, Cloth}

// RecordTables flattens the current catalog into the record tables: cards,
// wonders and boards, in that order. It fails on an effect the notation
// cannot write.
func RecordTables() ([]RecordTable, error) {
	var errs []error
	cards := RecordTable{Name: "cards", Header: DesignColumns, Rows: [][]string{}}
	for id := DesignID(1); id <= NumDesigns; id++ {
		d, ok := DesignByID(id)
		if !ok {
			continue
		}
		fx, err := effectsCell(d.Effects)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Slug, err))
		}
		var chainIn, chainOut []string
		var from, to []string
		if d.ChainIn != 0 {
			chainIn = append(chainIn, strconv.Itoa(int(d.ChainIn)))
			for _, p := range T.ChainProviders[d.ChainIn] {
				if p != 0 {
					from = append(from, reg.designs[p].Name)
				}
			}
		}
		for _, c := range d.ChainOut {
			if c != 0 {
				chainOut = append(chainOut, strconv.Itoa(int(c)))
				to = append(to, reg.designs[T.ChainTarget[c]].Name)
			}
		}
		slices.Sort(from)
		slices.Sort(to)
		cards.Rows = append(cards.Rows, []string{
			strconv.Itoa(int(d.ID)), d.Slug, strconv.Itoa(int(d.Age)), d.Name, strconv.Itoa(int(d.NameID)),
			d.Color.String(), copiesCell(d), CostCell(d.Cost), strings.Join(chainIn, ";"), strings.Join(from, ";"),
			strings.Join(chainOut, ";"), strings.Join(to, ";"), fx,
		})
	}

	wonders := RecordTable{Name: "wonders", Header: WonderColumns, Rows: [][]string{}}
	boards := RecordTable{Name: "boards", Header: BoardColumns, Rows: [][]string{}}
	for id := BoardID(1); id <= NumBoards; id++ {
		b, ok := BoardByID(id)
		if !ok {
			continue
		}
		boards.Rows = append(boards.Rows, []string{
			strconv.Itoa(int(b.ID)), b.Slug, b.Name, b.Wonder, b.Start.String(),
			strconv.Itoa(int(b.Sides[0].NumStages)), strconv.Itoa(int(b.Sides[1].NumStages)),
		})
		for side, def := range b.Sides {
			for i := range def.NumStages {
				st := def.Stages[i]
				fx, err := effectsCell(st.Effects)
				if err != nil {
					errs = append(errs, fmt.Errorf("%s %s stage %d: %w", b.Slug, Side(side+1), i+1, err))
				}
				wonders.Rows = append(wonders.Rows, []string{
					b.Slug, Side(side + 1).String(), b.Start.String(), strconv.Itoa(int(i) + 1), CostCell(st.Cost), fx,
				})
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return []RecordTable{cards, wonders, boards}, nil
}

func copiesCell(d Design) string {
	if d.Color == Purple {
		return "guild"
	}
	var ms []string
	for _, m := range d.Copies {
		if m != 0 {
			ms = append(ms, strconv.Itoa(int(m)))
		}
	}
	return strings.Join(ms, ";")
}

// CostCell writes a cost in the M13 notation: "free", "coins:N", or each
// resource with its count, such as "stone:2 ore:1 glass:1".
func CostCell(c Cost) string {
	if c.Free() {
		return "free"
	}
	var parts []string
	if c.Coins != 0 {
		parts = append(parts, "coins:"+strconv.Itoa(int(c.Coins)))
	}
	for _, r := range tableResources {
		if c.Res[r] != 0 {
			parts = append(parts, r.String()+":"+strconv.Itoa(int(c.Res[r])))
		}
	}
	return strings.Join(parts, " ")
}

func effectsCell(fx []Effect) (string, error) {
	var terms []string
	for _, e := range fx {
		ts, err := EffectTerms(e)
		if err != nil {
			return "", err
		}
		terms = append(terms, ts...)
	}
	slices.Sort(terms)
	return strings.Join(terms, ";"), nil
}

// EffectTerms writes one effect in the M13 notation. A per-card effect over
// several colours is one term per colour, as M13 wrote the Shipowners Guild.
func EffectTerms(e Effect) ([]string, error) {
	n := strconv.Itoa(int(e.N))
	switch e.Kind {
	case EffProduce:
		if e.N == 1 {
			return []string{"produce:" + e.Res.String()}, nil
		}
		return []string{"produce:" + e.Res.String() + "*" + n}, nil
	case EffProduceChoice:
		var rs []string
		for _, r := range tableResources {
			if e.Mask.Has(r) {
				rs = append(rs, r.String())
			}
		}
		return []string{"produce:" + strings.Join(rs, "/")}, nil
	case EffTradeDiscount:
		family := "raw"
		if e.Mask == Manufactured {
			family = "manufactured"
		}
		if e.N != 1 || e.Mask != Raw && e.Mask != Manufactured {
			break
		}
		switch e.Scope {
		case Left:
			return []string{"trade:" + family + ":left"}, nil
		case Right:
			return []string{"trade:" + family + ":right"}, nil
		case Left | Right:
			return []string{"trade:" + family + ":both"}, nil
		}
	case EffShields, EffPoints, EffCoins:
		return []string{e.Kind.String() + ":" + n}, nil
	case EffScience:
		return []string{"science:" + e.Symbol.String()}, nil
	case EffScienceChoice:
		return []string{"science:any"}, nil
	case EffCoinsPerCard, EffPointsPerCard, EffCoinsPerStage, EffPointsPerStage:
		whose, ok := whoseCell(e.Scope)
		if !ok {
			break
		}
		reward := "points" + n
		if e.Kind == EffCoinsPerCard || e.Kind == EffCoinsPerStage {
			reward = "coins" + n
		}
		if e.Kind == EffCoinsPerStage || e.Kind == EffPointsPerStage {
			return []string{"per:wonder-stage:" + whose + ":" + reward}, nil
		}
		var terms []string
		for c := Brown; c < NumColors; c++ {
			if e.Colors.Has(c) {
				terms = append(terms, "per:"+c.String()+":"+whose+":"+reward)
			}
		}
		return terms, nil
	case EffPointsIfWonderComplete:
		return []string{"other:points" + n + "-if-all-own-wonder-stages-built"}, nil
	case EffPlayLastCard:
		return []string{"other:play-last-card-of-age"}, nil
	case EffBuildFromDiscard:
		return []string{"other:build-from-discard"}, nil
	case EffFreeBuild:
		switch e.When {
		case FirstOfColour:
			return []string{"other:first-card-of-each-colour-free"}, nil
		case FirstTurnOfAge:
			return []string{"other:first-card-of-each-age-free"}, nil
		case LastTurnOfAge:
			return []string{"other:last-card-of-each-age-free"}, nil
		}
	}
	return nil, fmt.Errorf("effect %s has no record notation: %+v", e.Kind, e)
}

func whoseCell(s SeatScope) (string, bool) {
	switch s {
	case Self:
		return "self", true
	case Left | Right:
		return "neighbours", true
	case Self | Left | Right:
		return "self+neighbours", true
	}
	return "", false
}
