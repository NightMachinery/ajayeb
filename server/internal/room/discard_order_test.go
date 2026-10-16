package room_test

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestDiscardOrderLiveResyncUndoAndReload(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for range 8 {
		a.act(st.View.Choices[0].Key, st.Version, nil)
		v := st.Version + 1
		st = a.waitState(func(s wire.State) bool { return s.Version == v })
	}
	want := []string{}
	ids := []catalog.CardID{}
	for n := range 3 {
		key, slug := "", ""
		var chosen catalog.CardID
		for _, c := range st.View.Choices {
			if c.Kind != "discard" || c.Card == nil {
				continue
			}
			id, ok := catalog.CardBySlug(*c.Card)
			if !ok {
				t.Fatal(*c.Card)
			}
			if key == "" || (n == 0 && id > chosen) || (n != 0 && id < chosen) {
				key, slug, chosen = c.Key, *c.Card, id
			}
		}
		if key == "" {
			t.Fatal("no discard")
		}
		want = append(want, slug)
		ids = append(ids, chosen)
		a.act(key, st.Version, nil)
		v := st.Version + 1
		st = a.waitState(func(s wire.State) bool { return s.Version == v })
		if !reflect.DeepEqual(st.View.Discard, want) {
			t.Fatalf("live pile %v, want %v", st.View.Discard, want)
		}
	}
	if ids[0] <= ids[1] {
		t.Fatal("test did not distinguish ID order from chronology")
	}
	if got := a.resync().View.Discard; !reflect.DeepEqual(got, want) {
		t.Fatal("resync reordered pile")
	}
	a.undo(st.Version)
	v := st.Version + 1
	st = a.waitState(func(s wire.State) bool { return s.Version == v })
	want = want[:2]
	if !reflect.DeepEqual(st.View.Discard, want) {
		t.Fatal("undo did not restore previous pile")
	}
	h.evict(code, a)
	b := h.client("user-a", "tab-b", "Ann")
	after := b.join(code)
	if !reflect.DeepEqual(after.View.Discard, want) {
		t.Fatal("reloaded pile differs")
	}
}

func TestDiscardDestroyReviveActorUndoReload(t *testing.T) {
	for _, name := range []string{"zeus-destroy", "mausoleum"} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			keys := mustScenarioTrigger(t, name)
			code, err := h.m.CreateFromSpec(room.Spec{Mode: store.ModeHotseat, Seed: 7, Scenario: name, Actions: keys, Seats: [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "user-a"}, {Kind: store.SeatKindHuman, UserID: "user-a"}}})
			if err != nil {
				t.Fatal(err)
			}
			a := h.client("user-a", "tab-a", "Ann")
			st := a.join(code)
			final := []string{}
			before := []string{"baths"}
			if name == "zeus-destroy" {
				final = []string{"quarry"}
				before = []string{}
			}
			if !reflect.DeepEqual(st.View.Discard, final) {
				t.Fatalf("played pile %v, want %v", st.View.Discard, final)
			}
			a.undo(st.Version)
			v := st.Version + 1
			st = a.waitState(func(s wire.State) bool { return s.Version == v })
			if !reflect.DeepEqual(st.View.Discard, before) {
				t.Fatal("undo failed to restore pre-destroy/revive pile")
			}
			if st.View.Pending != nil && st.View.Pending.Kind == "buildFromDiscard" && !reflect.DeepEqual(st.View.Pending.Cards, before) {
				t.Fatal("pending pile differs")
			}
			h.evict(code, a)
			b := h.client("user-a", "tab-b", "Ann")
			st = b.join(code)
			if !reflect.DeepEqual(st.View.Discard, before) {
				t.Fatal("reload after undo differs")
			}
			b.act(keys[len(keys)-1], st.Version, nil)
			v = st.Version + 1
			st = b.waitState(func(s wire.State) bool { return s.Version == v })
			if !reflect.DeepEqual(st.View.Discard, final) {
				t.Fatal("replayed final action differs")
			}
		})
	}
}
