package rules

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

func TestObservationFixedShape(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[Observation](), reflect.TypeFor[Current](), reflect.TypeFor[PublicCity](), reflect.TypeFor[Knowledge](), reflect.TypeFor[PileLook](), reflect.TypeFor[Statistics]()} {
		if bad := puritycheck.FixedShapeViolations(typ, typ.Name()); len(bad) > 0 {
			t.Error(bad)
		}
		if !typ.Comparable() {
			t.Error("observation has references", typ)
		}
	}
	if reflect.TypeFor[Observation]().Size() > 2048 {
		t.Fatal("observation exceeds the 2 KiB design estimate")
	}
}
