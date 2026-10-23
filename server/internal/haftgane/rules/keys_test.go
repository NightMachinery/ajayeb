package rules

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func keyCatalog(t testing.TB) {
	t.Helper()
	defs := []catalog.Design{}
	for i, slug := range []string{"sample-alpha", "sample-beta", strings.Repeat("a", 30)} {
		defs = append(defs, catalog.Design{ID: catalog.DesignID(i + 1), NameID: catalog.NameID(i + 1), Slug: slug, Name: slug, Age: 1, Color: catalog.Blue, Copies: [3]uint8{3}, Effects: []catalog.Effect{{Kind: catalog.EffPoints, N: 1}}})
	}
	t.Cleanup(catalog.ReplaceForTest(defs, nil))
}
func TestUnregisteredActionIsInvalid(t *testing.T) {
	restore := catalog.ReplaceForTest(nil, nil)
	defer restore()
	if (Action{Kind: ActBuild, Design: 1}).Valid() || (Action{Kind: ActBuild, Design: 1}).Key() != "" {
		t.Fatal("unregistered design is valid")
	}
}
func TestCanonicalKeys(t *testing.T) {
	keyCatalog(t)
	for _, key := range []string{"side:day", "side:night", "build:sample-alpha:0:0", "build:sample-alpha:255:255", "stage:sample-beta:2:1", "sell:sample-alpha", "build-discarded:sample-beta", "stage:" + strings.Repeat("a", 30) + ":14:0"} {
		a, err := ParseKey(key)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if a.Key() != key {
			t.Fatalf("%s -> %+v -> %q", key, a, a.Key())
		}
	}
	for _, key := range []string{"", "side:Day", "side:none", "side:day:0", "build:sample-alpha:01:0", "build:sample-alpha:+1:0", "build:sample-alpha:256:0", "build:sample-alpha:-1:0", "build:sample-alpha:1", "build:sample-alpha:1:0:0", "build:1:0:0", "sell:Sample-Alpha", "sell:unknown", "sell:sample-alpha ", "sell:sample-alpha:0", "resign:0", "side:day\n", strings.Repeat("a", 65)} {
		if a, err := ParseKey(key); err == nil {
			t.Errorf("accepted noncanonical %q: %+v", key, a)
		}
	}
	for _, a := range []Action{{}, {Kind: ActSide, Side: catalog.Day, Design: 1}, {Kind: ActSell, Design: 1, Left: 1}, {Kind: ActBuildDiscarded, Design: 1, Side: catalog.Night}, {Kind: ActBuild, Design: 1, Side: catalog.Day}, {Kind: ActStage, Design: 81}} {
		if a.Key() != "" {
			t.Errorf("invalid union has key: %+v", a)
		}
	}
}
func TestAllActionRoundTrips(t *testing.T) {
	keyCatalog(t)
	for id := catalog.DesignID(1); id <= 3; id++ {
		for _, kind := range []ActionKind{ActBuild, ActStage} {
			for left := uint16(0); left <= 255; left++ {
				for right := uint16(0); right <= 255; right++ {
					want := Action{Kind: kind, Design: id, Left: uint8(left), Right: uint8(right)}
					got, err := ParseKey(want.Key())
					if err != nil || got != want {
						t.Fatalf("round trip %+v -> %+v, %v", want, got, err)
					}
				}
			}
		}
	}
}
func FuzzKeys(f *testing.F) {
	keyCatalog(f)
	for _, key := range []string{"side:day", "side:night", "build:sample-alpha:0:0", "stage:sample-beta:14:0", "sell:sample-alpha", "build-discarded:sample-beta", "build:sample-alpha:01:0", "", strings.Repeat("x", 100)} {
		f.Add(key)
	}
	f.Fuzz(func(t *testing.T, key string) {
		a, err := ParseKey(key)
		if err != nil {
			return
		}
		if a.Key() != key {
			t.Fatalf("accepted noncanonical %q -> %q", key, a.Key())
		}
		b, err := ParseKey(a.Key())
		if err != nil || b != a {
			t.Fatalf("unstable action %+v -> %+v: %v", a, b, err)
		}
	})
}
