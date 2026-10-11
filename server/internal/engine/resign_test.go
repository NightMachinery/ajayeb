package engine_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The forfeit (docs/rules/decisions.md): resign:<seat> ends the game at once
// with the other seat's victory, in any phase and whoever is to move. It is
// never listed by LegalActions, so a bot never resigns.

func TestResignKeys(t *testing.T) {
	for seat := range engine.Seat(2) {
		a := engine.ResignAction(seat)
		key := a.Key()
		if want := "resign:" + string(rune('0'+seat)); key != want {
			t.Fatalf("ResignAction(%d).Key() = %q, want %q", seat, key, want)
		}
		got, err := engine.ParseKey(key)
		if err != nil || got != a {
			t.Fatalf("ParseKey(%q) = %+v, %v; want %+v", key, got, err, a)
		}
	}
	for _, bad := range []string{"resign:2", "resign:", "resign", "resign:00", "resign:-1", "Resign:0", "resign:0:0"} {
		if _, err := engine.ParseKey(bad); !errors.Is(err, engine.ErrBadKey) {
			t.Errorf("ParseKey(%q) = %v, want ErrBadKey", bad, err)
		}
	}
	for _, a := range []engine.Action{
		{Type: engine.ActResign, Starter: 2},
		{Type: engine.ActResign, Card: catalog.CardID(1)},
		{Type: engine.ActResign, Token: catalog.TokenLaw},
	} {
		if err := a.Validate(); !errors.Is(err, engine.ErrMalformedAction) {
			t.Errorf("Validate(%+v) = %v, want ErrMalformedAction", a, err)
		}
	}
}

// TestResignEveryPly plays seeded games and, at every ply, has each seat
// resign on a copy: the game must end at once with the other seat's
// forfeit victory, the invariants must hold, and nothing else may change
// but the dropped open choice. LegalActions must never offer resign.
func TestResignEveryPly(t *testing.T) {
	games := 40
	if testing.Short() {
		games = 8
	}
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for g := range games {
		seed := uint64(0x7e5_1600 + g)
		s := engine.NewGame(seed, engine.Options{})
		pol := enginetest.NewRandomPolicy(seed)
		phases := map[engine.Phase]bool{}
		for !s.Over() {
			legal := engine.LegalActions(&s, buf[:0])
			if slices.ContainsFunc(legal, func(a engine.Action) bool { return a.Type == engine.ActResign }) {
				t.Fatalf("seed %d ply %d: LegalActions lists a resign", seed, s.Ply)
			}
			phases[s.Phase] = true
			for seat := range engine.Seat(2) {
				checkResign(t, &s, seat)
			}
			engine.ApplyTrusted(&s, pol.Choose(&s, legal))
		}
		// Over: resign is refused like any other action.
		for seat := range engine.Seat(2) {
			before := engine.Hash(&s)
			err := engine.Apply(&s, engine.ResignAction(seat), nil)
			var ie *engine.IllegalError
			if !errors.As(err, &ie) || ie.Reason != engine.ReasonGameOver || engine.Hash(&s) != before {
				t.Fatalf("seed %d: resign after the end: %v", seed, err)
			}
		}
		if g == 0 && !phases[engine.PhaseWonderDraft] {
			t.Fatalf("seed %d: the sweep never saw the draft", seed)
		}
	}
}

// checkResign has seat resign on a copy of s and checks the outcome.
func checkResign(t *testing.T, s *engine.State, seat engine.Seat) {
	t.Helper()
	c := s.Clone()
	a := engine.ResignAction(seat)
	if err := engine.IsLegal(&c, a); err != nil {
		t.Fatalf("ply %d phase %s: IsLegal(%s) = %v", s.Ply, s.Phase, a, err)
	}
	var ev []engine.Event
	if err := engine.Apply(&c, a, &ev); err != nil {
		t.Fatalf("ply %d phase %s: Apply(%s) = %v", s.Ply, s.Phase, a, err)
	}
	if err := engine.CheckInvariants(&c); err != nil {
		t.Fatalf("ply %d phase %s: after %s: %v", s.Ply, s.Phase, a, err)
	}
	if !c.Over() || c.Result.Kind != engine.VictoryForfeit || c.Result.Winner != int8(seat.Opp()) {
		t.Fatalf("ply %d: after %s: over %v, result %+v", s.Ply, a, c.Over(), c.Result)
	}
	if c.Ply != s.Ply+1 {
		t.Fatalf("ply %d: resign moved the ply to %d", s.Ply, c.Ply)
	}
	if c.Result.Scores != engine.Scores(s) {
		t.Fatalf("ply %d: the forfeit's score sheet is not the position's", s.Ply)
	}
	want := []engine.Event{
		{Kind: engine.EvResign, Seat: seat},
		{Kind: engine.EvGameOver, A: int16(engine.VictoryForfeit), B: int16(seat.Opp())},
	}
	if !slices.Equal(ev, want) {
		t.Fatalf("ply %d: events %+v, want %+v", s.Ply, ev, want)
	}
	// Nothing but the phase, the result, the ply, the extra turn and an open
	// choice's data changes.
	d := c
	d.Phase, d.Result, d.Ply, d.ExtraTurn = s.Phase, s.Result, s.Ply, s.ExtraTurn
	d.Destroy, d.Library, d.BoxTokens = s.Destroy, s.Library, s.BoxTokens
	if d != *s {
		t.Fatalf("ply %d: resign changed more than the end of the game", s.Ply)
	}
	if len(engine.LegalActions(&c, nil)) != 0 {
		t.Fatalf("ply %d: actions are legal after the forfeit", s.Ply)
	}
}

// TestResignDuringLibraryChoice: the Great Library's offer goes back to the
// box, so every token is still somewhere, and the seat that was not
// deciding may resign.
func TestResignDuringLibraryChoice(t *testing.T) {
	sc := engine.NamedScenario("great-library")
	s, err := engine.NewGameFromScenario(7, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	enginetest.Do(t, &s, sc.Trigger[0])
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase %s, want the library choice", s.Phase)
	}
	offer := s.Library
	waiting := s.ToMove.Opp()
	enginetest.Do(t, &s, engine.ResignAction(waiting).Key())
	if s.Library != ([engine.LibraryDraw]catalog.TokenID{}) {
		t.Fatalf("the offer %v stayed after the forfeit", s.Library)
	}
	for _, tk := range offer {
		if tk != 0 && !slices.Contains(s.BoxTokens[:], tk) {
			t.Fatalf("token %s did not go back to the box %v", tk, s.BoxTokens)
		}
	}
	v := engine.ViewFor(&s, engine.Spectator)
	if v.Result == nil || v.Result.Victory != view.VictoryKindForfeit || v.Result.Winner == nil || *v.Result.Winner != int(waiting.Opp()) {
		t.Fatalf("view result %+v", v.Result)
	}
	if v.ToMove != nil || len(v.Choices) != 0 || v.Pending != nil || v.Revealed == nil {
		t.Fatalf("a forfeited game's view: toMove %v, choices %d, pending %+v, revealed %v", v.ToMove, len(v.Choices), v.Pending, v.Revealed != nil)
	}
}

// TestResignInDraft: a forfeit during the wonder draft ends the game in age
// 0, and the end-of-game reveal works there too.
func TestResignInDraft(t *testing.T) {
	s := engine.NewGame(11, engine.Options{})
	enginetest.Do(t, &s, enginetest.Keys(&s)[0], "resign:0")
	if s.Age != 0 || s.Result.Kind != engine.VictoryForfeit || s.Result.Winner != 1 {
		t.Fatalf("age %d, result %+v", s.Age, s.Result)
	}
	for _, v := range []engine.Viewer{engine.SeatViewer(0), engine.SeatViewer(1), engine.Spectator} {
		vw := engine.ViewFor(&s, v)
		if vw.Draft != nil || vw.Revealed == nil || len(vw.Revealed.Undealt) != engine.NumAges {
			t.Fatalf("viewer %v: draft %v, revealed %+v", v, vw.Draft, vw.Revealed)
		}
	}
}

// TestResignReplays: a log ending in a resign replays to the same state;
// its entry records the seat that was to move, like every entry, while the
// key names the seat that resigned.
func TestResignReplays(t *testing.T) {
	const seed = 0x5eed
	s := engine.NewGame(seed, engine.Options{})
	log := engine.NewLog(seed, engine.Options{}, nil)
	pol := enginetest.NewSimplePolicy(seed)
	for range 30 {
		a := pol.Choose(&s, engine.LegalActions(&s, nil))
		log.Record(&s, a)
		enginetest.Do(t, &s, a.Key())
	}
	r := engine.ResignAction(s.ToMove.Opp())
	log.Record(&s, r)
	enginetest.Do(t, &s, r.Key())
	log.Seal(&s)
	got, err := engine.Replay(log)
	if err != nil {
		t.Fatal(err)
	}
	if got != s {
		t.Fatal("the replay of a forfeited game differs")
	}
}

func TestFilterResign(t *testing.T) {
	evs := []engine.Event{{Kind: engine.EvResign, Seat: 1, A: 9, Card: catalog.CardID(3)}}
	for _, v := range []engine.Viewer{engine.SeatViewer(0), engine.SeatViewer(1), engine.Spectator} {
		got := engine.FilterEvents(evs, v)
		if len(got) != 1 || got[0].Kind != view.EventKindResign || got[0].Seat == nil || *got[0].Seat != 1 ||
			got[0].Card != nil || got[0].Amount != nil || got[0].Private {
			t.Fatalf("viewer %v: %+v", v, got)
		}
	}
}
