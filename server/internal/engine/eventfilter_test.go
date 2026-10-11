package engine_test

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The event filter tests check FilterEvents: the whitelist of fields per
// event kind, the private events, and the Great Library's draw, which is
// private by kind. The redaction sweep (redaction_test.go) checks the
// filtered events of real games for leaks. Helpers are prefixed evf.

// evfJSON marshals the filtered events of evs for viewer v, one JSON
// object per event.
func evfJSON(t *testing.T, evs []engine.Event, v engine.Viewer) []string {
	t.Helper()
	out := []string{}
	for _, ge := range engine.FilterEvents(evs, v) {
		data, err := json.Marshal(ge)
		if err != nil {
			t.Fatalf("marshal %+v: %v", ge, err)
		}
		out = append(out, string(data))
	}
	return out
}

// TestFilterEventsWhitelist gives every public kind of event every engine
// field (a seat, a card, a wonder, a token and both numbers) and checks
// that each viewer receives exactly the fields view.GameEvent lists for
// that kind, so a stray field on an engine event never reaches a client.
func TestFilterEventsWhitelist(t *testing.T) {
	baths, ok := catalog.CardBySlug("baths")
	if !ok {
		t.Fatal("no baths")
	}
	full := func(k engine.EventKind, a, b int16) engine.Event {
		return engine.Event{Kind: k, Seat: 1, Card: baths, Wonder: catalog.WonderPyramids, Token: catalog.TokenLaw, A: a, B: b}
	}
	const p = `{"kind":`
	cases := []struct {
		ev   engine.Event
		want string
	}{
		{full(engine.EvDraft, 1, 9), p + `"draft","private":false,"seat":1,"wonder":"pyramids","auto":true}`},
		{full(engine.EvDraft, 0, 9), p + `"draft","private":false,"seat":1,"wonder":"pyramids","auto":false}`},
		{full(engine.EvDraftRound, 2, 9), p + `"draftRound","private":false,"round":2}`},
		{full(engine.EvAgeStart, 3, 9), p + `"ageStart","private":false,"age":3}`},
		{full(engine.EvBuild, 9, 9), p + `"build","private":false,"seat":1,"card":"baths"}`},
		{full(engine.EvDiscard, 4, 9), p + `"discard","private":false,"seat":1,"card":"baths","amount":4}`},
		{full(engine.EvWonder, 9, 9), p + `"wonder","private":false,"seat":1,"card":"baths","wonder":"pyramids"}`},
		{full(engine.EvPay, 7, 4), p + `"pay","private":false,"seat":1,"amount":7,"toOpponent":4}`},
		{full(engine.EvReveal, 9, 9), p + `"reveal","private":false,"card":"baths","slot":9}`},
		{full(engine.EvCoins, -3, 5), p + `"coins","private":false,"seat":1,"card":"baths","wonder":"pyramids","token":"law","amount":-3,"balance":5}`},
		{engine.Event{Kind: engine.EvCoins, Seat: 0, Token: catalog.TokenAgriculture, A: 6, B: 13}, p + `"coins","private":false,"seat":0,"token":"agriculture","amount":6,"balance":13}`},
		{full(engine.EvPawn, 2, -1), p + `"pawn","private":false,"seat":1,"from":2,"to":-1}`},
		{full(engine.EvLoot, 2, 5), p + `"loot","private":false,"seat":1,"amount":2,"value":5}`},
		{full(engine.EvPair, int16(catalog.Quill), 9), p + `"pair","private":false,"seat":1,"symbol":"quill"}`},
		{full(engine.EvToken, 1, 9), p + `"token","private":false,"seat":1,"token":"law","fromLibrary":true}`},
		{full(engine.EvToken, 0, 9), p + `"token","private":false,"seat":1,"token":"law","fromLibrary":false}`},
		{full(engine.EvDestroy, 9, 9), p + `"destroy","private":false,"seat":1,"card":"baths"}`},
		{full(engine.EvRebuild, 9, 9), p + `"rebuild","private":false,"seat":1,"card":"baths"}`},
		{full(engine.EvWonderRemoved, 9, 9), p + `"wonderRemoved","private":false,"seat":1,"wonder":"pyramids"}`},
		{full(engine.EvEffectSkipped, int16(engine.PhaseDestroyCard), 9), p + `"effectSkipped","private":false,"seat":1,"card":"baths","wonder":"pyramids","skipped":"destroyCard"}`},
		{engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Card: baths, A: int16(engine.PhasePickBoardToken)}, p + `"effectSkipped","private":false,"seat":0,"card":"baths","skipped":"pickBoardToken"}`},
		{full(engine.EvExtraTurn, 9, 9), p + `"extraTurn","private":false,"seat":1}`},
		{full(engine.EvExtraTurnLost, 9, 9), p + `"extraTurnLost","private":false,"seat":1}`},
		{full(engine.EvStarterChosen, 0, 9), p + `"starterChosen","private":false,"seat":1,"starter":0}`},
		{full(engine.EvGameOver, int16(engine.VictoryScience), 1), p + `"gameOver","private":false,"victory":"science","winner":1}`},
		{full(engine.EvGameOver, int16(engine.VictoryCivilian), int16(engine.SharedVictory)), p + `"gameOver","private":false,"victory":"civilian"}`},
		{full(engine.EvGameOver, int16(engine.VictoryForfeit), 0), p + `"gameOver","private":false,"victory":"forfeit","winner":0}`},
		{full(engine.EvResign, 9, 9), p + `"resign","private":false,"seat":1}`},
	}
	covered := map[string]bool{engine.EvLibraryDraw.String(): true} // TestFilterEventsLibraryDraw
	for _, c := range cases {
		covered[c.ev.Kind.String()] = true
		for _, v := range redactViewers {
			got := evfJSON(t, []engine.Event{c.ev}, v)
			if len(got) != 1 || got[0] != c.want {
				t.Errorf("%s for viewer %d:\n got %v\nwant %s", c.ev.Kind, v, got, c.want)
			}
		}
	}
	for _, k := range vwEnum(func(i uint8) string { return engine.EventKind(i).String() }, 1) {
		if !covered[k] {
			t.Errorf("event kind %s has no whitelist case", k)
		}
	}
}

// TestFilterEventsLibraryDraw checks the Great Library's draw from the real
// events of building it: the drawer receives each drawn token, marked
// private; the opponent and a spectator receive the same events with the
// token left out; every other event reaches everyone unchanged.
func TestFilterEventsLibraryDraw(t *testing.T) {
	s, ev := protoNamed(t, protoSeed(200), "great-library", engine.Options{}, 1)
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase %s", s.Phase)
	}
	drawer := evfJSON(t, ev, 0)
	var draws []string
	for _, e := range drawer {
		if strings.Contains(e, `"libraryDraw"`) {
			draws = append(draws, e)
		}
	}
	want := []string{
		`{"kind":"libraryDraw","private":true,"seat":0,"token":"law","place":0}`,
		`{"kind":"libraryDraw","private":true,"seat":0,"token":"philosophy","place":1}`,
		`{"kind":"libraryDraw","private":true,"seat":0,"token":"strategy","place":2}`,
	}
	if !slices.Equal(draws, want) {
		t.Fatalf("the drawer's draw events %v, want %v", draws, want)
	}
	for _, v := range []engine.Viewer{1, engine.Spectator} {
		got := evfJSON(t, ev, v)
		if len(got) != len(drawer) {
			t.Fatalf("viewer %d gets %d events, the drawer %d", v, len(got), len(drawer))
		}
		place := 0
		for i, e := range got {
			if !strings.Contains(e, `"libraryDraw"`) {
				if e != drawer[i] {
					t.Errorf("viewer %d: event %d is %s, the drawer's %s", v, i, e, drawer[i])
				}
				continue
			}
			if w := `{"kind":"libraryDraw","private":false,"seat":0,"place":` + strconv.Itoa(place) + `}`; e != w {
				t.Errorf("viewer %d: %s, want %s", v, e, w)
			}
			place++
		}
		if place != 3 {
			t.Errorf("viewer %d sees %d draws, want 3", v, place)
		}
	}

	// The draw is private by kind: an EvLibraryDraw without the Private
	// flag is filtered all the same.
	e := engine.Event{Kind: engine.EvLibraryDraw, Seat: 1, Token: catalog.TokenEconomy, A: 2}
	if got := evfJSON(t, []engine.Event{e}, 0); !slices.Equal(got, []string{`{"kind":"libraryDraw","private":false,"seat":1,"place":2}`}) {
		t.Errorf("an unflagged draw reaches seat 0 as %v", got)
	}
	if got := evfJSON(t, []engine.Event{e}, 1); !slices.Equal(got, []string{`{"kind":"libraryDraw","private":true,"seat":1,"token":"economy","place":2}`}) {
		t.Errorf("an unflagged draw reaches its drawer as %v", got)
	}
}

// TestFilterEventsPrivate checks that a private event of another kind
// reaches only its seat, marked private, and is dropped for everyone else,
// with the order of the other events kept.
func TestFilterEventsPrivate(t *testing.T) {
	evs := []engine.Event{
		{Kind: engine.EvExtraTurn, Seat: 0},
		{Kind: engine.EvCoins, Seat: 1, Token: catalog.TokenAgriculture, A: 6, B: 13, Private: true},
		{Kind: engine.EvExtraTurnLost, Seat: 0},
	}
	public := []string{`{"kind":"extraTurn","private":false,"seat":0}`, `{"kind":"extraTurnLost","private":false,"seat":0}`}
	for _, v := range []engine.Viewer{0, engine.Spectator, 4} {
		if got := evfJSON(t, evs, v); !slices.Equal(got, public) {
			t.Errorf("viewer %d gets %v, want %v", v, got, public)
		}
	}
	want := []string{public[0], `{"kind":"coins","private":true,"seat":1,"token":"agriculture","amount":6,"balance":13}`, public[1]}
	if got := evfJSON(t, evs, 1); !slices.Equal(got, want) {
		t.Errorf("seat 1 gets %v, want %v", got, want)
	}
}

// TestFilterEventsInvalid checks that events with no valid kind are dropped
// and that the result is never nil.
func TestFilterEventsInvalid(t *testing.T) {
	evs := []engine.Event{{Kind: engine.EvNone}, {Kind: engine.EventKind(200), Seat: 1}, {Kind: engine.EvExtraTurn, Seat: 1}}
	if got := evfJSON(t, evs, 1); !slices.Equal(got, []string{`{"kind":"extraTurn","private":false,"seat":1}`}) {
		t.Errorf("got %v", got)
	}
	for _, evs := range [][]engine.Event{nil, {}, {{Kind: engine.EvNone}}} {
		got := engine.FilterEvents(evs, 0)
		if got == nil || len(got) != 0 {
			t.Errorf("FilterEvents(%v) = %#v, want an empty, non-nil slice", evs, got)
		}
		if data, _ := json.Marshal(got); string(data) != "[]" {
			t.Errorf("FilterEvents(%v) marshals as %s", evs, data)
		}
	}
}
