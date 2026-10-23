package rules

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func TestFixedCoreValues(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[State](), reflect.TypeFor[Record](), reflect.TypeFor[Game](), reflect.TypeFor[City](),
		reflect.TypeFor[Options](), reflect.TypeFor[Score](), reflect.TypeFor[Result](), reflect.TypeFor[Play](),
		reflect.TypeFor[Action](), reflect.TypeFor[Event](), reflect.TypeFor[Point](), reflect.TypeFor[catalog.Effect](),
		reflect.TypeFor[catalog.Cost](), reflect.TypeFor[catalog.DesignSet](),
	} {
		if bad := puritycheck.FixedShapeViolations(typ, typ.Name()); len(bad) > 0 {
			t.Error(bad)
		}
		if !typ.Comparable() {
			t.Errorf("%s not comparable", typ)
		}
		if binary.Size(reflect.New(typ).Elem().Interface()) <= 0 {
			t.Errorf("%s not binary fixed", typ)
		}
	}
	for _, item := range []struct {
		typ reflect.Type
		max uintptr
	}{
		{reflect.TypeFor[State](), 1536}, {reflect.TypeFor[Record](), 1536}, {reflect.TypeFor[Game](), 3072},
	} {
		t.Logf("%s: %d bytes memory, %d encoded", item.typ, item.typ.Size(), binary.Size(reflect.New(item.typ).Elem().Interface()))
		if item.typ.Size() > item.max {
			t.Errorf("%s exceeds %d", item.typ, item.max)
		}
	}
	g := Game{}
	g.State.Hands[0][0] = 1
	g.Record.Packets[0][0][0] = 2
	copy := g.Clone()
	copy.State.Hands[0][0] = 3
	copy.Record.Packets[0][0][0] = 4
	if g.State.Hands[0][0] != 1 || g.Record.Packets[0][0][0] != 2 {
		t.Fatal("clone aliases source")
	}
}

func TestSeatRingAndPoints(t *testing.T) {
	// rules.md ruling 30: left is where hands go in Ages I and III.
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for s := Seat(0); s < Seat(n); s++ {
			if s.Left(n) != Seat((uint8(s)+1)%n) || s.Right(n) != Seat((uint8(s)+n-1)%n) || s.Left(n).Right(n) != s {
				t.Fatalf("n=%d seat=%d ring", n, s)
			}
		}
		if AllSeats(n).Count() != int(n) || AllSeats(n).Has(NoSeat) {
			t.Fatal(n)
		}
		for _, p := range []Point{{Kind: PointSides, Required: AllSeats(n)}, {Kind: PointTurn, Age: 1, Turn: 6, Required: AllSeats(n)}, {Kind: PointDiscardBuild, Age: 3, Turn: 6, Required: Seat(0).Mask()}, {Kind: PointLastCard, Age: 2, Turn: 7, Required: Seat(0).Mask()}} {
			if err := p.Validate(n); err != nil {
				t.Fatal(err)
			}
			if p.ID() == "" {
				t.Fatal("missing point id")
			}
		}
	}
	if Seat(0).Left(0) != NoSeat || NoSeat.Right(7) != NoSeat {
		t.Fatal("invalid ring input")
	}
	ids := []struct {
		p    Point
		want string
	}{
		{Point{Kind: PointSides}, "sides"}, {Point{Kind: PointTurn, Age: 2, Turn: 4}, "age-2-turn-4"},
		{Point{Kind: PointDiscardBuild, Age: 3, Turn: 6}, "age-3-turn-6-discard-build"}, {Point{Kind: PointLastCard, Age: 1, Turn: 7}, "age-1-last-card"},
	}
	for _, c := range ids {
		if c.p.ID() != c.want {
			t.Errorf("%+v = %s", c.p, c.p.ID())
		}
	}
	for _, p := range []Point{{Kind: PointTurn, Age: 0, Turn: 1, Required: AllSeats(3)}, {Kind: PointTurn, Age: 1, Turn: 7, Required: AllSeats(3)}, {Kind: PointDiscardBuild, Age: 1, Turn: 1, Required: AllSeats(3)}, {Kind: PointLastCard, Age: 1, Turn: 6, Required: 1}, {Kind: PointSides, Age: 1, Required: AllSeats(3)}, {Kind: PointSides, Required: 128}} {
		if p.Validate(3) == nil {
			t.Errorf("invalid point passed %+v", p)
		}
	}
	if (Options{}).Validate() != nil || RulesVersion != 1 {
		t.Fatal("default options/version")
	}
	if (Options{Sides: SideRule(255)}).Validate() == nil || (Options{Seating: SeatingRule(255)}).Validate() == nil {
		t.Fatal("invalid options")
	}
	for k := EvNone + 1; k < NumEventKinds; k++ {
		if k.String() == "" {
			t.Errorf("missing event %d", k)
		}
	}
}
