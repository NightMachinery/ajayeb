package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func TestAppendKeyIdentityAndStorage(t *testing.T) {
	check := func(a Action) {
		t.Helper()
		var buf [MaxKeyBytes + 4]byte
		copy(buf[:], "tag:")
		got := a.AppendKey(buf[:4])
		want := a.Key()
		if string(got) != "tag:"+want || len(want) > MaxKeyBytes {
			t.Fatal("canonical bytes changed", a, string(got), want)
		}
		if len(got) > 4 && &got[0] != &buf[0] {
			t.Fatal("did not reuse caller storage")
		}
	}
	for _, side := range []catalog.Side{catalog.NoSide, catalog.Day, catalog.Night} {
		check(Action{Kind: ActSide, Side: side})
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for _, kind := range []ActionKind{ActBuild, ActStage, ActSell, ActBuildDiscarded} {
			for _, left := range []uint8{0, 1, 7, 14, 99, 100, 255} {
				for _, right := range []uint8{0, 1, 7, 14, 99, 100, 255} {
					check(Action{Kind: kind, Design: id, Left: left, Right: right})
				}
			}
		}
	}
	check(Action{})
	check(Action{Kind: NumActionKinds, Design: 1})
	check(Action{Kind: ActBuild, Design: catalog.NumDesigns + 1})
	a := Action{Kind: ActBuild, Design: 1, Left: 255, Right: 255}
	var buf [MaxKeyBytes]byte
	if n := testing.AllocsPerRun(1000, func() { a.AppendKey(buf[:0]) }); n != 0 {
		t.Fatal("append allocates", n)
	}
}
