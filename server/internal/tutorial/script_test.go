package tutorial

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file replays every shipped script from its seed and checks, at each
// entry, that the key is legal for the entry's seat and that the lesson's
// precondition holds on the real deal (design section 5.3). A rules change
// that breaks a script fails here, not in front of a player.

// shipped lists every script ByName knows.
var shipped = []*Script{&FirstGame}

func TestByName(t *testing.T) {
	for _, sc := range shipped {
		got, ok := ByName(sc.Name)
		if !ok || got != sc {
			t.Errorf("ByName(%q) = %p, %v; want the script", sc.Name, got, ok)
		}
	}
	if _, ok := ByName("no-such-script"); ok {
		t.Error("ByName knows an unknown name")
	}
	if _, ok := ByName(""); ok {
		t.Error("ByName knows the empty name")
	}
}

// choiceOf returns the view's choice with this key, for the seat to move.
func choiceOf(s *engine.State, key string) (view.Choice, bool) {
	v := engine.ViewFor(s, engine.SeatViewer(s.ToMove))
	for _, c := range v.Choices {
		if c.Key == key {
			return c, true
		}
	}
	return view.Choice{}, false
}

// isAccessible reports whether card c is accessible in the structure.
func isAccessible(s *engine.State, c catalog.CardID) bool {
	i := engine.SlotOf(s, c)
	return i >= 0 && engine.Accessible(s)&(1<<i) != 0
}

func TestScriptShape(t *testing.T) {
	for _, sc := range shipped {
		if sc.Human != 0 {
			t.Errorf("%s: the player's seat is %d; you() builds seat 0 entries", sc.Name, sc.Human)
		}
		if len(sc.Outro) == 0 || sc.Outro[len(sc.Outro)-1].ID != wire.TutorialLessonPlayOn {
			t.Errorf("%s: the outro %v must end with playOn", sc.Name, sc.Outro)
		}
		for i, e := range sc.Entries {
			if e.Lessons == nil {
				t.Errorf("%s entry %d: nil lessons", sc.Name, i)
			}
			human := e.Seat == sc.Human
			if human != (e.Action != nil) {
				t.Errorf("%s entry %d (%s): seat %d with action %v", sc.Name, i, e.Key, e.Seat, e.Action)
			}
			if !human && len(e.Lessons) > 0 {
				t.Errorf("%s entry %d (%s): a bot entry with lessons", sc.Name, i, e.Key)
			}
			if e.Action != nil && e.Action.Key != e.Key {
				t.Errorf("%s entry %d: action key %q, entry key %q", sc.Name, i, e.Action.Key, e.Key)
			}
			a, err := engine.ParseKey(e.Key)
			if err != nil {
				t.Errorf("%s entry %d: %v", sc.Name, i, err)
				continue
			}
			if e.Action != nil {
				// The action names exactly the key's card and wonder.
				if (e.Action.Card != nil) != (a.Card != 0) || (e.Action.Card != nil && *e.Action.Card != catalog.Card(a.Card).Slug) {
					t.Errorf("%s entry %d: action card %v for key %s", sc.Name, i, e.Action.Card, e.Key)
				}
				if (e.Action.Wonder != nil) != (a.Wonder != 0) || (e.Action.Wonder != nil && *e.Action.Wonder != catalog.Wonder(a.Wonder).Slug) {
					t.Errorf("%s entry %d: action wonder %v for key %s", sc.Name, i, e.Action.Wonder, e.Key)
				}
			}
			for _, l := range e.Lessons {
				if l.Card != nil {
					if _, ok := catalog.CardBySlug(*l.Card); !ok {
						t.Errorf("%s entry %d: lesson %s names unknown card %q", sc.Name, i, l.ID, *l.Card)
					}
				}
				if l.Wonder != nil {
					if _, ok := catalog.WonderBySlug(*l.Wonder); !ok {
						t.Errorf("%s entry %d: lesson %s names unknown wonder %q", sc.Name, i, l.ID, *l.Wonder)
					}
				}
			}
		}
	}
}

func TestScriptReplays(t *testing.T) {
	for _, sc := range shipped {
		t.Run(sc.Name, func(t *testing.T) { checkScript(t, sc) })
	}
}

// checkScript replays sc and checks every entry's lessons.
func checkScript(t *testing.T, sc *Script) {
	s := engine.NewGame(sc.Seed, sc.Options)
	var prevEvents []engine.Event           // the previous action's events
	pushed := make([]bool, len(sc.Entries)) // the entry moved the pawn toward seat 1
	botRed := false                         // the bot has built a red card
	sawMilitary := false
	for i, e := range sc.Entries {
		if s.ToMove != e.Seat {
			t.Fatalf("entry %d (%s): seat %d to move, the entry is seat %d's", i, e.Key, s.ToMove, e.Seat)
		}
		a, ok := legal(&s, e.Key)
		if !ok {
			t.Fatalf("entry %d: %s is not legal", i, e.Key)
		}
		// The info lessons, on the position before the move.
		for _, l := range e.Lessons {
			switch l.ID {
			case wire.TutorialLessonReveal:
				c, _ := catalog.CardBySlug(*l.Card)
				if !slices.ContainsFunc(prevEvents, func(ev engine.Event) bool { return ev.Kind == engine.EvReveal && ev.Card == c }) {
					t.Errorf("entry %d: reveal %s, but the previous action revealed no such card", i, *l.Card)
				}
				if sc.Entries[i-1].Seat == sc.Human {
					t.Errorf("entry %d: reveal follows the player's own move", i)
				}
			case wire.TutorialLessonMilitary:
				sawMilitary = true
				if sc.Entries[i-2].Seat != sc.Human || !pushed[i-2] || botRed {
					t.Errorf("entry %d: military, but the player's previous move did not push the pawn, or the bot has built a red card", i)
				}
			case wire.TutorialLessonAgeOne:
				if s.Age != 1 || s.Phase != engine.PhasePlay {
					t.Errorf("entry %d: ageOne outside Age I", i)
				}
			case wire.TutorialLessonDraftIntro, wire.TutorialLessonDraftRound2:
				if s.Phase != engine.PhaseWonderDraft {
					t.Errorf("entry %d: %s outside the draft", i, l.ID)
				}
			}
		}
		if e.Action != nil {
			checkAction(t, i, &s, sc, e, a)
		}

		pawn := s.Military
		var ev []engine.Event
		if err := engine.Apply(&s, a, &ev); err != nil {
			t.Fatalf("entry %d: %s: %v", i, e.Key, err)
		}
		if e.Seat != sc.Human && a.Type == engine.ActConstructBuilding && catalog.Card(a.Card).Color == catalog.Red {
			botRed = true
		}
		pushed[i] = s.Military > pawn
		// No entry opens a choice or gives an extra turn.
		switch s.Phase {
		case engine.PhaseWonderDraft, engine.PhasePlay:
		default:
			t.Fatalf("entry %d (%s) opened phase %s", i, e.Key, s.Phase)
		}
		if s.ExtraTurn || s.PlayingAgain || slices.ContainsFunc(ev, func(x engine.Event) bool { return x.Kind == engine.EvExtraTurn }) {
			t.Fatalf("entry %d (%s) gave an extra turn", i, e.Key)
		}
		if a.Type != engine.ActDraftWonder && s.ToMove == e.Seat {
			t.Fatalf("entry %d (%s): the same seat moves again", i, e.Key)
		}
		prevEvents = ev
	}
	if !sawMilitary {
		t.Error("the script has no military lesson")
	}
	// The end: the player to move, in Age I, with cards left to finish it.
	if s.Age != 1 || s.Phase != engine.PhasePlay || s.ToMove != sc.Human {
		t.Errorf("the script ends in age %d, phase %s, seat %d to move", s.Age, s.Phase, s.ToMove)
	}
	if n := cardsLeft(&s); n < 4 {
		t.Errorf("the script ends with %d cards left, want at least 4", n)
	}
	for _, l := range sc.Outro {
		if l.ID == wire.TutorialLessonTokens && s.Players[sc.Human].Sci == [catalog.NumSymbols]uint8{} {
			t.Error("the outro teaches the tokens, but the player has no science symbol")
		}
	}
}

// checkAction checks the precondition of the player's action lesson on the
// position s before the move, and its effect.
func checkAction(t *testing.T, i int, s *engine.State, sc *Script, e Entry, a engine.Action) {
	t.Helper()
	ch, ok := choiceOf(s, e.Key)
	if !ok {
		t.Errorf("entry %d: %s is not among the view's choices", i, e.Key)
		return
	}
	var d *catalog.CardDef
	if a.Card != 0 {
		d = catalog.Card(a.Card)
	}
	wantType := func(tp engine.ActionType) {
		if a.Type != tp {
			t.Errorf("entry %d: lesson %s with key %s", i, e.Action.ID, e.Key)
		}
	}
	after := *s
	var ev []engine.Event
	if err := engine.Apply(&after, a, &ev); err != nil {
		t.Fatalf("entry %d: %v", i, err)
	}
	switch e.Action.ID {
	case wire.TutorialLessonDraftPick, wire.TutorialLessonDraftPickAgain:
		wantType(engine.ActDraftWonder)
		k := slices.Index(s.DraftOrder[:], a.Wonder)
		if s.Phase != engine.PhaseWonderDraft || k < 0 || s.DraftTaken&(1<<k) != 0 {
			t.Errorf("entry %d: %s is not on offer", i, e.Key)
		}
	case wire.TutorialLessonBuildFree:
		wantType(engine.ActConstructBuilding)
		if !isAccessible(s, a.Card) || d.Color != catalog.Brown || ch.Cost == nil || ch.Cost.Total != 0 || ch.Cost.ChainFree || !d.Cost.Free() {
			t.Errorf("entry %d: buildFree %s is not an accessible brown card that costs nothing (cost %+v)", i, d.Slug, ch.Cost)
		}
	case wire.TutorialLessonBuildOwn:
		wantType(engine.ActConstructBuilding)
		pay := engine.BuildingPayment(s, sc.Human, a.Card)
		own := 0
		for _, u := range pay.Own {
			own += int(u)
		}
		if ch.Cost == nil || ch.Cost.Total != 0 || ch.Cost.ChainFree || d.Cost.Units() == 0 || own != d.Cost.Units() || d.Cost.Coins != 0 {
			t.Errorf("entry %d: buildOwn %s is not paid by the player's own production alone (cost %+v)", i, d.Slug, ch.Cost)
		}
		// The military lesson that follows speaks of this move's shield.
		if d.Color != catalog.Red || after.Military <= s.Military {
			t.Errorf("entry %d: buildOwn %s did not move the pawn toward seat 1", i, d.Slug)
		}
	case wire.TutorialLessonTrade:
		wantType(engine.ActConstructBuilding)
		if ch.Cost == nil || len(ch.Cost.Trade) == 0 || ch.Cost.TradeTotal == 0 || !ch.Cost.Affordable {
			t.Errorf("entry %d: trade %s has no affordable trade line (cost %+v)", i, d.Slug, ch.Cost)
		}
	case wire.TutorialLessonDiscard:
		wantType(engine.ActDiscardBuilding)
	case wire.TutorialLessonBuildWonder:
		wantType(engine.ActConstructWonder)
		if !ownsWonder(s, sc.Human, a.Wonder) || ch.Cost == nil || !ch.Cost.Affordable {
			t.Errorf("entry %d: buildWonder %s is not the player's unbuilt, affordable wonder", i, e.Key)
		}
		w := catalog.Wonder(a.Wonder)
		if _, ok := catalog.FindEffect(w.Effects, catalog.EffPlayAgain); ok {
			t.Errorf("entry %d: %s gives an extra turn", i, w.Slug)
		}
	case wire.TutorialLessonScience:
		wantType(engine.ActConstructBuilding)
		if d.Color != catalog.Green {
			t.Errorf("entry %d: science %s is not green", i, d.Slug)
		}
	case wire.TutorialLessonChain:
		wantType(engine.ActConstructBuilding)
		if d.ChainOut == catalog.NoChain {
			t.Errorf("entry %d: chain %s shows no chain symbol", i, d.Slug)
		}
	default:
		t.Errorf("entry %d: unexpected action lesson %s", i, e.Action.ID)
	}
}
