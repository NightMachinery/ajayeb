package rules

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

var observationSink Observation
var gameCloneSink Game
var tableViewSink view.Table

func benchmarkObservedGame(b *testing.B) Game {
	b.Helper()
	g, err := NewGame(Setup{Seats: 7, Seed: 42, Options: Options{Sides: SidesDay}})
	if err != nil {
		b.Fatal(err)
	}
	for g.State.Age < 3 || g.State.Turn < 4 {
		g, _, err = g.Resolve(observationBatch(g, 42))
		if err != nil {
			b.Fatal(err)
		}
	}
	return g
}
func BenchmarkCloneSeven(b *testing.B) {
	g := benchmarkObservedGame(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for i := 0; i < 7; i++ {
			gameCloneSink = g.Clone()
		}
	}
}
func BenchmarkObserveSeven(b *testing.B) {
	g := benchmarkObservedGame(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for s := Seat(0); s < 7; s++ {
			observationSink = g.Observe(s)
		}
	}
}
func BenchmarkViewForSeven(b *testing.B) {
	g := benchmarkObservedGame(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for s := Seat(0); s < 7; s++ {
			tableViewSink = g.ViewFor(s)
		}
	}
}

// Capacity stress is a serialization measurement, not a reachable-position
// claim. It holds 217 keys with seven-unit witnesses, the full notebook, three
// full pile looks and seven cities at once. The release gate may replace these
// conservative dimensions with tighter catalog-specific maxima.
func TestViewCapacityBytes(t *testing.T) {
	longest := ""
	for _, d := range catalog.BaseDesigns() {
		if len(d.Slug) > len(longest) {
			longest = d.Slug
		}
	}
	g, err := NewGame(Setup{Seats: 7, Seed: 42, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	v := g.ViewFor(0)
	for i := range v.Seats {
		v.Seats[i].Built = []string{}
		for n := 0; n < 24; n++ {
			v.Seats[i].Built = append(v.Seats[i].Built, longest)
		}
		v.Seats[i].Coins = 32767
	}
	witness := make([]view.PaymentUnit, 7)
	for i := range witness {
		witness[i] = view.PaymentUnit{Resource: "papyrus", Source: "right-choice", Producer: 9, Price: 2}
	}
	v.You.Choices = make([]view.Choice, MaxTurnChoices)
	for i := range v.You.Choices {
		v.You.Choices[i] = view.Choice{Key: strings.Repeat("k", MaxKeyBytes), Kind: "build", Design: longest, Stage: 4, Left: 14, Right: 14, Printed: 255, Total: 269, Free: "first-colour", Witness: witness}
	}
	v.You.Disabled = make([]view.Disabled, 14)
	for i := range v.You.Disabled {
		v.You.Disabled[i] = view.Disabled{Kind: "stage", Design: longest, Stage: 4, Reason: "insufficient-coins", Cheapest: 269, ShortBy: 269}
	}
	v.You.Notebook.Seen = []view.SeenEntry{}
	for age := uint8(1); age <= 3; age++ {
		for turn := uint8(1); turn <= 7; turn++ {
			hand := []string{}
			for i := 0; i < 8-int(turn); i++ {
				hand = append(hand, longest)
			}
			kind := PointTurn
			if turn == 7 {
				kind = PointLastCard
			}
			v.You.Notebook.Seen = append(v.You.Notebook.Seen, view.SeenEntry{Point: (Point{Kind: kind, Age: age, Turn: turn}).ID(), Hand: hand, Played: longest, Action: "stage"})
		}
	}
	for i := 0; i < 3; i++ {
		look := view.PileLook{Point: "age-3-turn-6-discard-build", Cards: []view.DesignCount{}}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			look.Cards = append(look.Cards, view.DesignCount{Design: designSlug(id), Count: 3})
		}
		v.You.Notebook.PileLooks = append(v.You.Notebook.PileLooks, look)
	}
	v.You.Offer = []view.OfferEntry{}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		v.You.Offer = append(v.You.Offer, view.OfferEntry{Design: designSlug(id), Count: 3, Legal: true})
		v.You.Notebook.Statistics.Returning = append(v.You.Notebook.Statistics.Returning, view.DesignBound{Design: designSlug(id), Bounds: view.Bounds{Certain: 1, AtMost: 3}})
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("capacity stress: %d uncompressed JSON bytes, %d choices, %d witness units per choice; longest slug %d bytes", len(b), len(v.You.Choices), len(witness), len(longest))
}
