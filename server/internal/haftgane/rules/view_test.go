package rules

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

func TestViewPublicAndOwnFacts(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 41, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for seat := Seat(0); seat < 7; seat++ {
		v := g.ViewFor(seat)
		if v.You == nil || v.You.Seat != int(seat) || len(v.You.Hand) != 7 || len(v.You.Choices) != len(g.Legal(seat)) || len(v.You.Neighbours) != 2 {
			t.Fatal("own screen fields missing", v)
		}
		if v.Seats[seat].Relative != 0 || v.Seats[seat.Left(7)].Relation != "left" || v.Seats[seat.Right(7)].Relation != "right" {
			t.Fatal("relative seats", v.Seats)
		}
		for s, c := range v.Seats {
			if c.Score.Total != int(scoreCity(&g.State, Seat(s)).Total) || c.Shields != int(g.State.Cities[s].Shields) {
				t.Fatal("running public score or shield arithmetic differs")
			}
		}
		for i, c := range v.You.Choices {
			want := g.Legal(seat)[i]
			if c.Key != want.Key() || c.Total != int(want.Total) || c.Printed != int(want.Printed) || len(c.Witness) != int(want.Payment.Units) {
				t.Fatal("choice presentation differs from offered key", c, want)
			}
		}
	}
	if g.ViewFor(NoSeat).You != nil || g.ViewFor(7).You != nil {
		t.Fatal("spectator got a private section")
	}
}

func TestViewNeighbourPricesAndOutlook(t *testing.T) {
	g := abilityScenario(t, 2, 3, [MaxSeats]wonderCity{
		0: {board: catalog.BoardGizah, buildings: []catalog.DesignID{21, 22, 14}},
		1: {board: catalog.BoardHalikarnassos, buildings: []catalog.DesignID{1, 11}},
		6: {board: catalog.BoardAlexandria, buildings: []catalog.DesignID{7, 15, 16}},
	}, [MaxSeats][]catalog.DesignID{})
	v := g.ViewFor(0)
	left, right := v.You.Neighbours[0], v.You.Neighbours[1]
	if left.Seat != 1 || left.Direction != "left" || left.Outlook != 3 || right.Seat != 6 || right.Outlook != -1 {
		t.Fatal("Age II conflict outlook", left, right)
	}
	rawChoice, glass, ore := false, false, false
	for _, u := range left.Units {
		if u.Choice {
			rawChoice = true
			if u.Count != 1 || len(u.Options) != 2 {
				t.Fatal("one shared producer sold as separate units", u)
			}
			for _, p := range u.Options {
				if p.Price != 2 || p.Discount != nil {
					t.Fatal("east post incorrectly discounted left raw", p)
				}
			}
		}
		for _, p := range u.Options {
			if p.Resource == "glass" {
				glass = true
				if p.Price != 1 || p.Discount == nil || p.Discount.Design != "marketplace" {
					t.Fatal("manufactured discount source", p)
				}
			}
		}
	}
	for _, u := range right.Units {
		for _, p := range u.Options {
			if p.Resource == "ore" {
				ore = true
				if p.Price != 1 || p.Discount == nil || p.Discount.Design != "east-trading-post" {
					t.Fatal("right raw discount source", p)
				}
			}
		}
	}
	if !rawChoice || !glass || !ore {
		t.Fatal("neighbour purchase units missing")
	}
}

func TestViewDisabledReasons(t *testing.T) {
	g := abilityScenario(t, 1, 3, [MaxSeats]wonderCity{0: {board: catalog.BoardGizah, buildings: []catalog.DesignID{17}}}, [MaxSeats][]catalog.DesignID{0: {17, 23, 16}})
	g.State.Cities[0].Coins = 0
	reasons := make(map[string]bool)
	for _, d := range g.ViewFor(0).You.Disabled {
		reasons[d.Reason] = true
		if d.Reason == "insufficient-coins" && d.ShortBy <= 0 {
			t.Fatal("short purse needs positive shortfall")
		}
	}
	if !reasons["already-built"] || !reasons["cannot-cover"] {
		t.Fatal("distinct name and capacity reasons", reasons)
	}
	// A coverable printed coin cost has no resource-cover problem.
	g = abilityScenario(t, 1, 3, [MaxSeats]wonderCity{0: {board: catalog.BoardGizah}}, [MaxSeats][]catalog.DesignID{0: {1}})
	g.State.Cities[0].Coins = 0
	found := false
	for _, d := range g.ViewFor(0).You.Disabled {
		if d.Kind == "build" && d.Design == "clay-pit" {
			found = true
			if d.Reason != "insufficient-coins" || d.ShortBy != 1 {
				t.Fatal("coin shortfall", d)
			}
		}
	}
	if !found {
		t.Fatal("missing unaffordable construction")
	}
	g = abilityScenario(t, 2, 3, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, stages: 2}}, [MaxSeats][]catalog.DesignID{0: {39}})
	found = false
	for _, d := range g.ViewFor(0).You.Disabled {
		if d.Kind == "stage" {
			found = true
			if d.Reason != "wonder-complete" {
				t.Fatal(d)
			}
		}
	}
	if !found {
		t.Fatal("complete wonder not explained")
	}
}

func TestViewEndPolicy(t *testing.T) {
	for _, hide := range []bool{false, true} {
		g, err := NewGame(Setup{Seats: 3, Seed: 99, Options: Options{Sides: SidesDay, HideAtEnd: hide}})
		if err != nil {
			t.Fatal(err)
		}
		for g.State.Phase != PhaseOver {
			for _, s := range []Seat{0, 1, 2, NoSeat} {
				v := g.ViewFor(s)
				if v.Reveal != nil || v.Result != nil || v.CanDownload {
					t.Fatal("premature end information")
				}
			}
			g, _, err = g.Resolve(salesBatch(g))
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, s := range []Seat{0, 1, 2, NoSeat} {
			v := g.ViewFor(s)
			if v.Result == nil || v.CanDownload == hide || (v.Reveal == nil) != hide {
				t.Fatal("end policy", hide, v)
			}
			if !hide && (len(v.Reveal.Leftovers) != 9 || len(v.Reveal.Pile) != 63 || len(v.Reveal.Box) != 5) {
				t.Fatal("end reveal missing cards", v.Reveal)
			}
			b, err := json.Marshal(v)
			if err != nil || strings.Contains(string(b), "seed") || strings.Contains(string(b), "fingerprint") {
				t.Fatal("private setup root in view", err)
			}
		}
	}
}

func TestViewDTOShape(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			walk(typ.Elem())
			return
		}
		if typ.Kind() != reflect.Struct {
			if typ.Kind() != reflect.Int && typ.Kind() != reflect.String && typ.Kind() != reflect.Bool {
				t.Error("unexpected DTO kind", typ)
			}
			return
		}
		if seen[typ] {
			return
		}
		seen[typ] = true
		names := map[string]bool{}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			name, opt, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.IsExported() || name == "" || name == "-" || names[name] {
				t.Error("invalid JSON field", typ, f)
			}
			names[name] = true
			if (opt == "omitempty") != (f.Type.Kind() == reflect.Pointer) || opt != "" && opt != "omitempty" {
				t.Error("invalid omission", typ, f)
			}
			if strings.Contains(strings.ToLower(name), "seed") || strings.Contains(strings.ToLower(name), "fingerprint") {
				t.Error("secret DTO field", f)
			}
			walk(f.Type)
		}
	}
	walk(reflect.TypeFor[view.Table]())
	walk(reflect.TypeFor[view.HistoryEntry]())
}

func assertViewSlices(t *testing.T, v any) {
	t.Helper()
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, path string) {
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				walk(v.Elem(), path)
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i), path+"."+v.Type().Field(i).Name)
			}
		case reflect.Slice:
			if v.IsNil() {
				t.Error("nil DTO slice", path)
			}
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), path+"[]")
			}
		}
	}
	walk(reflect.ValueOf(v), "view")
}
