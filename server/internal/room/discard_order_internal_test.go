package room

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"reflect"
	"testing"
)

func TestChronologicalDiscardPublicProjection(t *testing.T) {
	card := func(slug string) catalog.CardID {
		c, ok := catalog.CardBySlug(slug)
		if !ok {
			t.Fatal(slug)
		}
		return c
	}
	hist := []histEntry{{events: []engine.Event{
		{Kind: engine.EvDiscard, Card: card("palace")},
		{Kind: engine.EvDestroy, Card: card("quarry")},
		{Kind: engine.EvRebuild, Card: card("palace")},
		{Kind: engine.EvDiscard, Card: card("baths")},
		{Kind: engine.EvDestroy, Card: card("palace")},
	}}}
	current := []string{"baths", "quarry", "tavern", "palace"}
	want := []string{"tavern", "quarry", "baths", "palace"}
	if got := chronologicalDiscard(current, hist); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !reflect.DeepEqual(current, []string{"baths", "quarry", "tavern", "palace"}) {
		t.Fatal("aliased current pile")
	}
	// A wholly synthetic initial set has no implied arrival chronology.
	if got := chronologicalDiscard(current, nil); !reflect.DeepEqual(got, current) {
		t.Fatal("initial batch reordered")
	}
	if got := chronologicalDiscard([]string{}, hist); got == nil || len(got) != 0 {
		t.Fatal("empty pile must serialize as []")
	}
}

func TestDiscardDestructionAndRevivalReplayFrames(t *testing.T) {
	for _, name := range []string{"zeus-destroy", "mausoleum"} {
		t.Run(name, func(t *testing.T) {
			h := view.HintSettings{}
			f := &store.File{Scenario: name, Seed: 7, Mode: store.ModeHotseat, Hints: &h}
			f.Seats[0].Kind = store.SeatKindHuman
			f.Seats[1].Kind = store.SeatKindHuman
			trigger := engine.NamedScenario(name).Trigger
			for n := 0; n <= len(trigger); n++ {
				s, hist, err := replayKeys(f, trigger[:n])
				if err != nil {
					t.Fatal(err)
				}
				r := &Room{file: f, state: s, hist: hist}
				for _, viewer := range []engine.Viewer{0, 1, engine.Spectator} {
					var fc frameCache
					v, _ := fc.get(&s, h, viewer, nil, hist)
					want := []string{}
					if name == "mausoleum" && n < 2 {
						want = []string{"baths"}
					}
					if name == "zeus-destroy" && n == 2 {
						want = []string{"quarry"}
					}
					if !reflect.DeepEqual(v.Discard, want) {
						t.Fatalf("n%d viewer%d: %v want%v", n, viewer, v.Discard, want)
					}
					if v.Pending != nil && v.Pending.Kind == view.PendingKindBuildFromDiscard && !reflect.DeepEqual(v.Pending.Cards, v.Discard) {
						t.Fatal("pending pile order differs")
					}
					// Frames use the same projection with and without history.
					_ = r
				}
			}
		})
	}
}
