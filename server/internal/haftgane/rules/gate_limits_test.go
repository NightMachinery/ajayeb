package rules

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

func TestRulesGateDimensions(t *testing.T) {
	// Bounds follow simultaneous point counts, the unique boards and the
	// seven-unit printed cost bound. They do not assert that every bound is tight.
	if MaxPoints != 1+NumAges*TurnsPerAge+MaxPileLooks+NumAges || MaxResolutionEvents < 4*MaxSeats+3 {
		t.Fatal("point/event proof no longer fits")
	}
	if MaxPaymentOutcomes != 2*catalog.MaxCostUnits+1 || MaxTurnChoices != HandSize*(2*MaxPaymentOutcomes+1) {
		t.Fatal("frontier/offer proof no longer fits")
	}
	if MaxPileCards != NumAges*HandSize*MaxSeats || MaxKnownHands != NumAges*MaxTurns || MaxSeenCards != NumAges*HandSize*(HandSize+1)/2 {
		t.Fatal("card/notebook proof no longer fits")
	}
	if MaxHistoryEntries != MaxPoints+1 {
		t.Fatal("history prefix proof no longer fits")
	}
	maxBuildings, maxLooks, maxChoice, maxSaleChoice := 0, 0, 0, 0
	maxStages := 0
	for _, b := range catalog.BaseBoards() {
		for _, side := range b.Sides {
			maxStages = max(maxStages, int(side.NumStages))
			for built := 0; built <= int(side.NumStages); built++ {
				extra, looks, ownChoices := 0, 0, 0
				for _, s := range side.Stages[:built] {
					for _, e := range s.Effects {
						switch e.Kind {
						case catalog.EffPlayLastCard:
							extra = NumAges
						case catalog.EffBuildFromDiscard:
							looks++
						case catalog.EffProduceChoice:
							ownChoices++
						}
					}
				}
				maxBuildings = max(maxBuildings, NumAges*TurnsPerAge+extra+looks-built)
				maxLooks = max(maxLooks, looks)
				// Every own choice-design producer can coexist by name; wonder producers
				// add to those. Saleable choices exclude every wonder and yellow card.
				choices, sale := ownChoices, 0
				for _, d := range catalog.BaseDesigns() {
					for _, e := range d.Effects {
						if e.Kind == catalog.EffProduceChoice {
							choices++
							if e.ForSale {
								sale++
							}
						}
					}
				}
				maxChoice = max(maxChoice, choices)
				maxSaleChoice = max(maxSaleChoice, sale)
			}
		}
	}
	if maxBuildings != MaxCityBuildings || maxLooks != MaxPileLooks || maxStages != MaxStages {
		t.Fatal("board ability dimensions", maxBuildings, maxLooks, maxStages)
	}
	if maxChoice > MaxChoice || maxSaleChoice > MaxForSaleChoice {
		t.Fatal("producer capacities", maxChoice, maxSaleChoice)
	}
	longest := 0
	for _, d := range catalog.BaseDesigns() {
		if d.Cost.Units() > catalog.MaxCostUnits {
			t.Fatal("cost above proof domain")
		}
		for _, kind := range []ActionKind{ActBuild, ActStage, ActSell, ActBuildDiscarded} {
			a := Action{Kind: kind, Design: d.ID}
			if kind == ActBuild || kind == ActStage {
				a.Left, a.Right = 255, 255
			}
			key := a.Key()
			if len(key) > MaxCanonicalKeyBytes || len(key) > MaxKeyBytes {
				t.Fatal("canonical key above bound", key)
			}
			longest = max(longest, len(key))
		}
	}
	for _, item := range []struct {
		typ   reflect.Type
		bound uintptr
	}{{reflect.TypeFor[State](), MaxStateBytes}, {reflect.TypeFor[Record](), MaxRecordBytes}, {reflect.TypeFor[Game](), MaxGameBytes}, {reflect.TypeFor[Observation](), MaxObservationBytes}} {
		if item.typ.Size() > item.bound {
			t.Fatal("fixed memory bound", item.typ, item.typ.Size(), item.bound)
		}
		t.Logf("%s memory %d bytes (cap %d)", item.typ, item.typ.Size(), item.bound)
	}
	if binary.Size(Game{}) != GameEncodingBytes || len(appendGameEncoding(nil, &Game{})) != GameEncodingBytes || binary.Size(Observation{}) != ObservationEncodingBytes || ObservationEncodingSize != ObservationEncodingBytes {
		t.Fatal("field encoding dimensions changed")
	}
	t.Logf("board proof: max buildings %d, pile looks %d, own choices %d/%d, sale choices %d/%d, longest current key %d/%d", maxBuildings, maxLooks, maxChoice, MaxChoice, maxSaleChoice, MaxForSaleChoice, longest, MaxCanonicalKeyBytes)
}

// Each cardinality comes from an engine loop or fixed array. Using all 80
// designs for a scenario's public city also covers partial-history scenarios,
// which need not obey a natural playout's tighter 20-building bound.
var gateSliceBounds = map[string]int{
	"Table.Required": MaxSeats, "Table.Seats": MaxSeats, "Table.Events": MaxResolutionEvents,
	"Seat.Built": catalog.NumDesigns, "Seat.Victories": NumAges,
	"Private.Hand": HandSize, "Private.Tucked": MaxStages, "Private.Choices": MaxTurnChoices, "Private.Disabled": 2 * HandSize, "Private.Offer": catalog.NumDesigns, "Private.Neighbours": 2,
	"Notebook.Seen": MaxKnownHands, "Notebook.PileLooks": MaxPileLooks, "SeenEntry.Hand": HandSize, "PileLook.Cards": catalog.NumDesigns,
	"Statistics.Circulating": int(catalog.NumColors) - 1, "Statistics.Returning": catalog.NumDesigns, "Statistics.NeighbourPicks": 2 * MaxTurns,
	"Neighbour.Units": int(catalog.NumResources) + MaxForSaleChoice, "TradeUnit.Options": int(catalog.NumResources), "Choice.Witness": catalog.MaxCostUnits,
	"Science.Counts": int(catalog.NumSymbols) - 1, "Science.Choices": int(catalog.NumSymbols) - 1,
	"Result.Scores": MaxSeats, "Result.Rank": MaxSeats, "Result.Winners": MaxSeats,
	"Reveal.Tucked": MaxSeats, "Reveal.Tucked[]": MaxStages, "Reveal.Leftovers": NumAges * MaxSeats, "Reveal.Pile": MaxPileCards, "Reveal.Box": MaxBoxGuilds,
}

func maximalGateDTO(t *testing.T, typ reflect.Type, field string) reflect.Value {
	t.Helper()
	v := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(typ.Elem()))
		v.Elem().Set(maximalGateDTO(t, typ.Elem(), field))
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			v.Field(i).Set(maximalGateDTO(t, f.Type, typ.Name()+"."+f.Name))
		}
	case reflect.Slice:
		n, ok := gateSliceBounds[field]
		if !ok {
			t.Fatal("new DTO slice needs a cardinality proof", field)
		}
		v.Set(reflect.MakeSlice(typ, n, n))
		for i := 0; i < n; i++ {
			v.Index(i).Set(maximalGateDTO(t, typ.Elem(), field+"[]"))
		}
	case reflect.String:
		v.SetString(strings.Repeat("x", MaxViewStringBytes))
	case reflect.Int:
		v.SetInt(-32768) // Six characters, at least as wide as any emitted scalar.
	case reflect.Bool:
		v.SetBool(false)
	default:
		t.Fatal("unbounded/new DTO type", typ, field)
	}
	return v
}
func gateDTOProblem(v reflect.Value, field string) string {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			return gateDTOProblem(v.Elem(), field)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if bad := gateDTOProblem(v.Field(i), v.Type().Name()+"."+f.Name); bad != "" {
				return bad
			}
		}
	case reflect.Slice:
		n, ok := gateSliceBounds[field]
		if !ok || v.Len() > n {
			return field
		}
		for i := 0; i < v.Len(); i++ {
			if bad := gateDTOProblem(v.Index(i), field+"[]"); bad != "" {
				return bad
			}
		}
	case reflect.String:
		if len(v.String()) > MaxViewStringBytes {
			return field
		}
		for _, b := range []byte(v.String()) {
			if b < 32 || b > 126 || b == '"' || b == '\\' || b == '<' || b == '>' || b == '&' {
				return field
			}
		}
	case reflect.Int:
		if v.Int() < -32768 || v.Int() > 65535 {
			return field
		}
	case reflect.Bool:
	default:
		return field
	}
	return ""
}
func TestRulesGateViewByteBound(t *testing.T) {
	// The structural product includes mutually exclusive optional sections. It
	// is a proven upper bound on this DTO grammar, not an attainable position.
	biggest := maximalGateDTO(t, reflect.TypeFor[view.Table](), "").Interface().(view.Table)
	b, err := json.Marshal(biggest)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != MaxViewJSONBytes {
		t.Fatal("view shape exceeds published bound", len(b), MaxViewJSONBytes)
	}
	if bad := gateDTOProblem(reflect.ValueOf(biggest), ""); bad != "" {
		t.Fatal("maximal DTO violates its own grammar", bad)
	}
	for seed := uint64(0); seed < 3; seed++ {
		g, err := NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SideRule(seed)}})
		if err != nil {
			t.Fatal(err)
		}
		for {
			for _, viewer := range []Seat{0, 1, 2, 3, 4, 5, 6, NoSeat} {
				v := g.ViewFor(viewer)
				if bad := gateDTOProblem(reflect.ValueOf(v), ""); bad != "" {
					t.Fatal("emitted DTO violates byte-bound premise", bad)
				}
				encoded, err := json.Marshal(v)
				if err != nil || len(encoded) > MaxViewJSONBytes {
					t.Fatal("view above bound", len(encoded), err)
				}
			}
			if g.State.Phase == PhaseOver {
				break
			}
			g, _, err = g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	bad := biggest
	own := *biggest.You
	bad.You = &own
	bad.You.Choices = append(append([]view.Choice{}, own.Choices...), view.Choice{})
	if got := gateDTOProblem(reflect.ValueOf(bad), ""); got != "Private.Choices" {
		t.Fatal("oversized choice-list negative accepted", got)
	}
	bad = biggest
	bad.Point = strings.Repeat("x", MaxViewStringBytes+1)
	if got := gateDTOProblem(reflect.ValueOf(bad), ""); got != "Table.Point" {
		t.Fatal("oversized string negative accepted", got)
	}
	bad = biggest
	bad.Pile = 65536
	if got := gateDTOProblem(reflect.ValueOf(bad), ""); got != "Table.Pile" {
		t.Fatal("oversized integer negative accepted", got)
	}
	t.Logf("maximal structural DTO %d JSON bytes <= published %d, without transport cap; actual reachable views use the smaller catalog strings and exclusive sections", len(b), MaxViewJSONBytes)
}
