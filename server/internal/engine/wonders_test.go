package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The per-wonder tests: each of the twelve wonders built through Apply, with
// its coins, shields, production, extra turn, special choice and points,
// and the rules around wonders: Theology never doubles an extra turn,
// Strategy never adds to a wonder's shields, the Appian Way's loss comes
// after the Economy income, the seven-wonder rule removes the last unbuilt
// wonder whoever owns it, and the effects come in a fixed order. The
// transitions of the pending choices that some wonders open, and the
// victories reached through them, are in pending_test.go. Helpers are
// prefixed wd; pending_test.go shares them.

// wdTuck is the card the tests tuck under the wonders they build. wdNew makes
// it accessible.
const wdTuck = "rostrum"

// wdNew starts an Age II position with seat 0 to move and 40 coins, and the
// tuck card accessible. Age I's cards are out of play there, so the tests
// can put any of them in a city or the discard pile.
func wdNew(t *testing.T) *enginetest.Builder {
	t.Helper()
	return enginetest.New(t, 21).Age(2).FirstPlayer(0).ToMove(0).Coins(0, 40).Accessible(wdTuck)
}

// wdBuild builds wonder w for the seat to move, tucking wdTuck under it.
func wdBuild(t *testing.T, s *engine.State, w string) []engine.Event {
	t.Helper()
	return enginetest.Do(t, s, "wonder:"+w+":"+wdTuck)
}

// wdWonder returns the id of a wonder slug.
func wdWonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	w, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("no wonder %q", slug)
	}
	return w
}

// wdToken returns the id of a token slug.
func wdToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("no token %q", slug)
	}
	return tk
}

// wdPaid returns the two events every wonder build starts with: the wonder
// (with wdTuck under it) and its payment q, when it cost anything.
func wdPaid(t *testing.T, seat engine.Seat, w string, q engine.Quote) []engine.Event {
	t.Helper()
	ev := []engine.Event{{Kind: engine.EvWonder, Seat: seat, Wonder: wdWonder(t, w), Card: cardID(t, wdTuck)}}
	if q.Total > 0 {
		ev = append(ev, engine.Event{Kind: engine.EvPay, Seat: seat, A: q.Total, B: q.ToOpponent})
	}
	return ev
}

// wdNoReveals drops the EvReveal events, which depend on the slots the
// scenario happened to use.
func wdNoReveals(ev []engine.Event) []engine.Event {
	out := []engine.Event{}
	for _, e := range ev {
		if e.Kind != engine.EvReveal {
			out = append(out, e)
		}
	}
	return out
}

// wdEvents compares events, ignoring reveals.
func wdEvents(t *testing.T, what string, got, want []engine.Event) {
	t.Helper()
	if got = wdNoReveals(got); !slices.Equal(got, want) {
		t.Fatalf("%s: events\n  %+v\nwant\n  %+v", what, got, want)
	}
}

// wdTurnPassed checks that the turn is over and the opponent of seat is to
// play.
func wdTurnPassed(t *testing.T, s *engine.State, seat engine.Seat) {
	t.Helper()
	if s.Phase != engine.PhasePlay || s.ToMove != seat.Opp() || s.Active != seat.Opp() || s.ExtraTurn {
		t.Fatalf("phase %s, seat %d to move, active %d, extra turn %v: want seat %d's turn",
			s.Phase, s.ToMove, s.Active, s.ExtraTurn, seat.Opp())
	}
}

// wdPlaysAgain checks that seat plays again: its extra turn has begun, and
// the action after it passes the turn.
func wdPlaysAgain(t *testing.T, s *engine.State, seat engine.Seat) {
	t.Helper()
	if s.Phase != engine.PhasePlay || s.ToMove != seat || s.Active != seat || s.ExtraTurn {
		t.Fatalf("phase %s, seat %d to move, active %d, extra turn %v: want seat %d's extra turn",
			s.Phase, s.ToMove, s.Active, s.ExtraTurn, seat)
	}
	x := *s
	enginetest.Do(t, &x, "discard "+x.Slots[firstAccessible(&x)].Card.String())
	if x.ToMove != seat.Opp() {
		t.Fatalf("seat %d played a second extra turn", seat)
	}
}

// wdWonderPoints checks the wonder points of seat.
func wdWonderPoints(t *testing.T, s *engine.State, seat engine.Seat, want int16) {
	t.Helper()
	if got := engine.Scores(s)[seat].Wonders; got != want {
		t.Fatalf("seat %d's wonder points: %d, want %d", seat, got, want)
	}
}

func TestWonderPyramids(t *testing.T) {
	s := wdNew(t).Wonders(0, "pyramids").State()
	q := engine.WonderQuote(&s, 0, catalog.WonderPyramids)
	ev := wdBuild(t, &s, "pyramids")
	wdEvents(t, "the pyramids", ev, wdPaid(t, 0, "pyramids", q))
	if !s.Players[0].Wonders[0].Built || s.Players[0].Coins != 40-q.Total || s.Military != 0 {
		t.Fatalf("wonder %+v, coins %d, pawn %d", s.Players[0].Wonders[0], s.Players[0].Coins, s.Military)
	}
	wdTurnPassed(t, &s, 0)
	wdWonderPoints(t, &s, 0, 9)
}

func TestWonderColossus(t *testing.T) {
	// Two shields push the pawn from 2 to 4, past seat 1's 2-coin looting
	// token. Strategy adds nothing to a wonder's shields.
	s := wdNew(t).Wonders(0, "colossus").Tokens(0, "strategy").Pawn(2).Coins(1, 5).State()
	q := engine.WonderQuote(&s, 0, catalog.WonderColossus)
	ev := wdBuild(t, &s, "colossus")
	want := append(wdPaid(t, 0, "colossus", q),
		engine.Event{Kind: engine.EvPawn, Seat: 0, A: 2, B: 4},
		engine.Event{Kind: engine.EvLoot, Seat: 1, A: 2, B: 2})
	wdEvents(t, "the colossus", ev, want)
	if s.Military != 4 || s.Players[1].Coins != 3 || s.Looting&engine.LootPlus3 != 0 {
		t.Fatalf("pawn %d, seat 1 coins %d, looting %#x", s.Military, s.Players[1].Coins, s.Looting)
	}
	wdTurnPassed(t, &s, 0)
	wdWonderPoints(t, &s, 0, 3)
}

func TestWonderGreatLighthouse(t *testing.T) {
	// From now on one unit of wood, clay or stone each turn: the Pyramids
	// (SSS P) need one stone fewer. The opponent's prices do not change.
	s := wdNew(t).Wonders(0, "great-lighthouse", "pyramids").State()
	before := engine.WonderQuote(&s, 0, catalog.WonderPyramids).Total
	oppPrices := engine.TradePrices(&s, 1)
	q := engine.WonderQuote(&s, 0, catalog.WonderGreatLighthouse)
	ev := wdBuild(t, &s, "great-lighthouse")
	wdEvents(t, "the great lighthouse", ev, wdPaid(t, 0, "great-lighthouse", q))
	p := &s.Players[0]
	if p.NChoice != 1 || p.Choice[0] != catalog.RawGoods {
		t.Fatalf("flexible production %v (%d)", p.Choice, p.NChoice)
	}
	if after := engine.WonderQuote(&s, 0, catalog.WonderPyramids).Total; before != 8 || after != 6 {
		t.Fatalf("the pyramids cost %d before and %d after, want 8 and 6", before, after)
	}
	if engine.TradePrices(&s, 1) != oppPrices {
		t.Fatalf("the opponent's prices changed: %v, then %v", oppPrices, engine.TradePrices(&s, 1))
	}
	wdTurnPassed(t, &s, 0)
	wdWonderPoints(t, &s, 0, 4)
}

func TestWonderTempleOfArtemis(t *testing.T) {
	s := wdNew(t).Wonders(0, "temple-of-artemis").State()
	q := engine.WonderQuote(&s, 0, catalog.WonderTempleOfArtemis)
	ev := wdBuild(t, &s, "temple-of-artemis")
	want := append(wdPaid(t, 0, "temple-of-artemis", q),
		engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderTempleOfArtemis, A: 12, B: 40 - q.Total + 12},
		engine.Event{Kind: engine.EvExtraTurn, Seat: 0})
	wdEvents(t, "the temple of artemis", ev, want)
	if s.Players[0].Coins != 40-q.Total+12 {
		t.Fatalf("coins %d", s.Players[0].Coins)
	}
	wdPlaysAgain(t, &s, 0)
	wdWonderPoints(t, &s, 0, 0)
}

func TestWonderMausoleum(t *testing.T) {
	garrison, baths := cardID(t, "garrison"), cardID(t, "baths")
	t.Run("builds a discarded card for free", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "mausoleum").Discard("garrison", "baths").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderMausoleum)
		ev := wdBuild(t, &s, "mausoleum")
		wdEvents(t, "the mausoleum", ev, wdPaid(t, 0, "mausoleum", q))
		if s.Phase != engine.PhaseBuildFromDiscard || s.ToMove != 0 || s.Active != 0 {
			t.Fatalf("phase %s, seat %d to move, active %d", s.Phase, s.ToMove, s.Active)
		}
		if got := enginetest.Keys(&s); !slices.Equal(got, []string{"revive:garrison", "revive:baths"}) {
			t.Fatalf("legal %v", got)
		}
		if r := enginetest.MustFail(t, &s, "revive tavern"); r != engine.ReasonNotInDiscard {
			t.Fatalf("revive a card not discarded: reason %s", r)
		}
		coins := s.Players[0].Coins
		ev = enginetest.Do(t, &s, "revive garrison")
		wdEvents(t, "the rebuild", ev, []engine.Event{
			{Kind: engine.EvRebuild, Seat: 0, Card: garrison},
			{Kind: engine.EvPawn, Seat: 0, A: 0, B: 1},
		})
		if s.Players[0].Coins != coins || !s.Players[0].City.Has(garrison) || s.Discard != catalog.CardSetOf(baths) {
			t.Fatalf("coins %d (had %d), city %v, discard %v", s.Players[0].Coins, coins, s.Players[0].City, s.Discard)
		}
		wdTurnPassed(t, &s, 0)
		wdWonderPoints(t, &s, 0, 2)
	})
	t.Run("an empty discard pile skips the choice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "mausoleum").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderMausoleum)
		ev := wdBuild(t, &s, "mausoleum")
		want := append(wdPaid(t, 0, "mausoleum", q), engine.Event{Kind: engine.EvEffectSkipped, Seat: 0,
			Wonder: catalog.WonderMausoleum, A: int16(engine.PhaseBuildFromDiscard)})
		wdEvents(t, "the mausoleum", ev, want)
		if !s.Players[0].Wonders[0].Built {
			t.Fatal("the mausoleum is not built")
		}
		wdTurnPassed(t, &s, 0)
	})
	t.Run("Urbanism pays nothing and Strategy applies", func(t *testing.T) {
		// Horse Breeders chain from the Stable, but a rebuild is not a
		// construction through a chain.
		s := wdNew(t).Wonders(0, "mausoleum").City(0, "stable").Tokens(0, "urbanism", "strategy").
			Discard("horse-breeders").State()
		wdBuild(t, &s, "mausoleum")
		coins := s.Players[0].Coins
		ev := enginetest.Do(t, &s, "revive horse-breeders")
		wdEvents(t, "the rebuild", ev, []engine.Event{
			{Kind: engine.EvRebuild, Seat: 0, Card: cardID(t, "horse-breeders")},
			{Kind: engine.EvPawn, Seat: 0, A: 0, B: 2},
		})
		if s.Players[0].Coins != coins {
			t.Fatalf("coins %d, had %d", s.Players[0].Coins, coins)
		}
	})
	t.Run("a rebuilt card pays its coins", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "mausoleum").Discard("tavern").State()
		wdBuild(t, &s, "mausoleum")
		coins := s.Players[0].Coins
		tavern := cardID(t, "tavern")
		ev := enginetest.Do(t, &s, "revive tavern")
		wdEvents(t, "the rebuild", ev, []engine.Event{
			{Kind: engine.EvRebuild, Seat: 0, Card: tavern},
			{Kind: engine.EvCoins, Seat: 0, Card: tavern, A: 4, B: coins + 4},
		})
	})
}

func TestWonderGreatLibrary(t *testing.T) {
	agri, law, philo := wdToken(t, "agriculture"), wdToken(t, "law"), wdToken(t, "philosophy")
	s := wdNew(t).Wonders(0, "great-library").
		BoardTokens("economy", "masonry", "strategy", "theology", "urbanism").
		BoxOrder("agriculture", "law", "philosophy").State()
	box := s.BoxTokens
	if box[0] != agri || box[1] != law || box[2] != philo || box[4] == 0 {
		t.Fatalf("the box is %v", box)
	}
	q := engine.WonderQuote(&s, 0, catalog.WonderGreatLibrary)
	ev := wdBuild(t, &s, "great-library")
	want := wdPaid(t, 0, "great-library", q)
	for i, tk := range []catalog.TokenID{agri, law, philo} {
		want = append(want, engine.Event{Kind: engine.EvLibraryDraw, Seat: 0, Token: tk, A: int16(i), Private: true})
	}
	wdEvents(t, "the great library", ev, want)
	if s.Phase != engine.PhasePickLibraryToken || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("phase %s, seat %d to move, active %d", s.Phase, s.ToMove, s.Active)
	}
	if s.Library != [engine.LibraryDraw]catalog.TokenID{agri, law, philo} ||
		s.BoxTokens != [engine.NumBoxTokens]catalog.TokenID{box[3], box[4]} {
		t.Fatalf("offer %v, box %v", s.Library, s.BoxTokens)
	}
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"token:agriculture", "token:law", "token:philosophy"}) {
		t.Fatalf("legal %v", got)
	}
	for _, step := range []string{"token economy", "token " + box[3].String()} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonTokenUnavailable {
			t.Fatalf("%s: reason %s", step, r)
		}
	}
	coins := s.Players[0].Coins
	ev = enginetest.Do(t, &s, "token agriculture")
	wdEvents(t, "the pick", ev, []engine.Event{
		{Kind: engine.EvToken, Seat: 0, Token: agri, A: 1},
		{Kind: engine.EvCoins, Seat: 0, Token: agri, A: 6, B: coins + 6},
	})
	// The two tokens not taken go back to the box, behind the two left in it.
	if s.Library != ([engine.LibraryDraw]catalog.TokenID{}) ||
		s.BoxTokens != [engine.NumBoxTokens]catalog.TokenID{box[3], box[4], law, philo} {
		t.Fatalf("offer %v, box %v", s.Library, s.BoxTokens)
	}
	if !s.Players[0].Tokens.Has(agri) || s.Players[0].Tokens.Count() != 1 || s.BoardTokens.Count() != 5 {
		t.Fatalf("tokens %v, board %v", s.Players[0].Tokens, s.BoardTokens)
	}
	wdTurnPassed(t, &s, 0)
	wdWonderPoints(t, &s, 0, 4)
	if got := engine.Scores(&s)[0].Tokens; got != 4 {
		t.Fatalf("token points %d, want Agriculture's 4", got)
	}
}

func TestWonderPiraeus(t *testing.T) {
	// From now on one unit of glass or papyrus each turn: the Pyramids
	// (SSS P) need no papyrus. No coins.
	s := wdNew(t).Wonders(0, "piraeus", "pyramids").State()
	oppPrices := engine.TradePrices(&s, 1)
	q := engine.WonderQuote(&s, 0, catalog.WonderPiraeus)
	ev := wdBuild(t, &s, "piraeus")
	wdEvents(t, "piraeus", ev, append(wdPaid(t, 0, "piraeus", q), engine.Event{Kind: engine.EvExtraTurn, Seat: 0}))
	p := &s.Players[0]
	if p.NChoice != 1 || p.Choice[0] != catalog.Manufactured || p.Coins != 40-q.Total {
		t.Fatalf("flexible production %v (%d), coins %d", p.Choice, p.NChoice, p.Coins)
	}
	if got := engine.WonderQuote(&s, 0, catalog.WonderPyramids).Total; got != 6 {
		t.Fatalf("the pyramids cost %d after Piraeus, want 6", got)
	}
	if engine.TradePrices(&s, 1) != oppPrices {
		t.Fatal("the opponent's prices changed")
	}
	wdPlaysAgain(t, &s, 0)
	wdWonderPoints(t, &s, 0, 2)
}

func TestWonderHangingGardens(t *testing.T) {
	s := wdNew(t).Wonders(0, "hanging-gardens").State()
	q := engine.WonderQuote(&s, 0, catalog.WonderHangingGardens)
	ev := wdBuild(t, &s, "hanging-gardens")
	want := append(wdPaid(t, 0, "hanging-gardens", q),
		engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderHangingGardens, A: 6, B: 40 - q.Total + 6},
		engine.Event{Kind: engine.EvExtraTurn, Seat: 0})
	wdEvents(t, "the hanging gardens", ev, want)
	wdPlaysAgain(t, &s, 0)
	wdWonderPoints(t, &s, 0, 3)
}

func TestWonderStatueOfZeus(t *testing.T) {
	quarry := cardID(t, "quarry")
	t.Run("destroys a brown card", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "statue-of-zeus").City(1, "clay-pool", "quarry", "glassworks").State()
		if p := engine.TradePrices(&s, 0)[catalog.Stone]; p != 3 {
			t.Fatalf("stone costs seat 0 %d before, want 3", p)
		}
		q := engine.WonderQuote(&s, 0, catalog.WonderStatueOfZeus)
		ev := wdBuild(t, &s, "statue-of-zeus")
		wdEvents(t, "the statue of zeus", ev, append(wdPaid(t, 0, "statue-of-zeus", q),
			engine.Event{Kind: engine.EvPawn, Seat: 0, A: 0, B: 1}))
		if s.Phase != engine.PhaseDestroyCard || s.Destroy != catalog.Brown || s.ToMove != 0 || s.Active != 0 {
			t.Fatalf("phase %s, destroy %s, seat %d to move", s.Phase, s.Destroy, s.ToMove)
		}
		if got := enginetest.Keys(&s); !slices.Equal(got, []string{"destroy:clay-pool", "destroy:quarry"}) {
			t.Fatalf("legal %v", got)
		}
		if r := enginetest.MustFail(t, &s, "destroy glassworks"); r != engine.ReasonNotTarget {
			t.Fatalf("destroy a grey card: reason %s", r)
		}
		ev = enginetest.Do(t, &s, "destroy quarry")
		wdEvents(t, "the destruction", ev, []engine.Event{{Kind: engine.EvDestroy, Seat: 0, Card: quarry}})
		if s.Players[1].City.Has(quarry) || !s.Discard.Has(quarry) || s.Players[1].Prod[catalog.Stone] != 0 {
			t.Fatalf("seat 1's city %v, discard %v", s.Players[1].City, s.Discard)
		}
		if p := engine.TradePrices(&s, 0)[catalog.Stone]; p != 2 || s.Destroy != catalog.NoColor {
			t.Fatalf("stone costs seat 0 %d after, want 2 (destroy %s)", p, s.Destroy)
		}
		wdTurnPassed(t, &s, 0)
		wdWonderPoints(t, &s, 0, 3)
	})
	t.Run("no brown card skips the choice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "statue-of-zeus").City(1, "glassworks").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderStatueOfZeus)
		ev := wdBuild(t, &s, "statue-of-zeus")
		wdEvents(t, "the statue of zeus", ev, append(wdPaid(t, 0, "statue-of-zeus", q),
			engine.Event{Kind: engine.EvPawn, Seat: 0, A: 0, B: 1},
			engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Wonder: catalog.WonderStatueOfZeus, A: int16(engine.PhaseDestroyCard)}))
		wdTurnPassed(t, &s, 0)
	})
	t.Run("a military supremacy ends the game before the choice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "statue-of-zeus").City(1, "quarry").Pawn(8).State()
		q := engine.WonderQuote(&s, 0, catalog.WonderStatueOfZeus)
		ev := wdBuild(t, &s, "statue-of-zeus")
		wdEvents(t, "the statue of zeus", ev, append(wdPaid(t, 0, "statue-of-zeus", q),
			engine.Event{Kind: engine.EvPawn, Seat: 0, A: 8, B: 9},
			engine.Event{Kind: engine.EvGameOver, A: int16(engine.VictoryMilitary), B: 0}))
		if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Destroy != catalog.NoColor {
			t.Fatalf("phase %s, result %+v, destroy %s", s.Phase, s.Result, s.Destroy)
		}
		if !s.Players[1].City.Has(quarry) {
			t.Fatal("the quarry was destroyed after the game ended")
		}
	})
}

func TestWonderSphinx(t *testing.T) {
	s := wdNew(t).Wonders(0, "sphinx").State()
	q := engine.WonderQuote(&s, 0, catalog.WonderSphinx)
	ev := wdBuild(t, &s, "sphinx")
	wdEvents(t, "the sphinx", ev, append(wdPaid(t, 0, "sphinx", q), engine.Event{Kind: engine.EvExtraTurn, Seat: 0}))
	wdPlaysAgain(t, &s, 0)
	wdWonderPoints(t, &s, 0, 6)
}

func TestWonderAppianWay(t *testing.T) {
	t.Run("the opponent's loss is floored at zero", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "appian-way").Coins(1, 2).State()
		q := engine.WonderQuote(&s, 0, catalog.WonderAppianWay)
		ev := wdBuild(t, &s, "appian-way")
		wdEvents(t, "the appian way", ev, append(wdPaid(t, 0, "appian-way", q),
			engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: 3, B: 40 - q.Total + 3},
			engine.Event{Kind: engine.EvExtraTurn, Seat: 0},
			engine.Event{Kind: engine.EvCoins, Seat: 1, Wonder: catalog.WonderAppianWay, A: -2, B: 0}))
		if s.Players[0].Coins != 40-q.Total+3 || s.Players[1].Coins != 0 {
			t.Fatalf("coins %d and %d", s.Players[0].Coins, s.Players[1].Coins)
		}
		wdPlaysAgain(t, &s, 0)
		wdWonderPoints(t, &s, 0, 3)
	})
	t.Run("after the Economy income", func(t *testing.T) {
		// Seat 1 has no coins but owns Economy: it receives the 10 trading
		// coins first, then loses 3.
		s := wdNew(t).Wonders(0, "appian-way").Coins(1, 0).Tokens(1, "economy").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderAppianWay)
		if q.Total != 10 || q.ToOpponent != 10 {
			t.Fatalf("quote %+v, want 10 coins of trading", q)
		}
		ev := wdBuild(t, &s, "appian-way")
		wdEvents(t, "the appian way", ev, append(wdPaid(t, 0, "appian-way", q),
			engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: 3, B: 33},
			engine.Event{Kind: engine.EvExtraTurn, Seat: 0},
			engine.Event{Kind: engine.EvCoins, Seat: 1, Wonder: catalog.WonderAppianWay, A: -3, B: 7}))
	})
	t.Run("a small Economy income", func(t *testing.T) {
		// Seat 0 produces the stone and clay and buys only the papyrus: seat
		// 1 receives 2 coins and then loses those 2.
		s := wdNew(t).Wonders(0, "appian-way").City(0, "stone-pit", "quarry", "clay-pool", "clay-pit").
			Coins(1, 0).Tokens(1, "economy").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderAppianWay)
		if q.Total != 2 || q.ToOpponent != 2 {
			t.Fatalf("quote %+v, want 2 coins of trading", q)
		}
		ev := wdBuild(t, &s, "appian-way")
		wdEvents(t, "the appian way", ev, append(wdPaid(t, 0, "appian-way", q),
			engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: 3, B: 41},
			engine.Event{Kind: engine.EvExtraTurn, Seat: 0},
			engine.Event{Kind: engine.EvCoins, Seat: 1, Wonder: catalog.WonderAppianWay, A: -2, B: 0}))
	})
}

func TestWonderCircusMaximus(t *testing.T) {
	press := cardID(t, "press")
	t.Run("destroys a grey card", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "circus-maximus").City(1, "glassworks", "press", "quarry").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderCircusMaximus)
		ev := wdBuild(t, &s, "circus-maximus")
		wdEvents(t, "circus maximus", ev, append(wdPaid(t, 0, "circus-maximus", q),
			engine.Event{Kind: engine.EvPawn, Seat: 0, A: 0, B: 1}))
		if s.Phase != engine.PhaseDestroyCard || s.Destroy != catalog.Grey {
			t.Fatalf("phase %s, destroy %s", s.Phase, s.Destroy)
		}
		if got := enginetest.Keys(&s); !slices.Equal(got, []string{"destroy:glassworks", "destroy:press"}) {
			t.Fatalf("legal %v", got)
		}
		if r := enginetest.MustFail(t, &s, "destroy quarry"); r != engine.ReasonNotTarget {
			t.Fatalf("destroy a brown card: reason %s", r)
		}
		ev = enginetest.Do(t, &s, "destroy press")
		wdEvents(t, "the destruction", ev, []engine.Event{{Kind: engine.EvDestroy, Seat: 0, Card: press}})
		if s.Players[1].City.Has(press) || !s.Discard.Has(press) || s.Players[1].Prod[catalog.Papyrus] != 0 ||
			engine.TradePrices(&s, 0)[catalog.Papyrus] != 2 {
			t.Fatalf("seat 1's city %v, discard %v, prices %v", s.Players[1].City, s.Discard, engine.TradePrices(&s, 0))
		}
		wdTurnPassed(t, &s, 0)
		wdWonderPoints(t, &s, 0, 3)
	})
	t.Run("no grey card skips the choice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "circus-maximus").City(1, "quarry").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderCircusMaximus)
		ev := wdBuild(t, &s, "circus-maximus")
		wdEvents(t, "circus maximus", ev, append(wdPaid(t, 0, "circus-maximus", q),
			engine.Event{Kind: engine.EvPawn, Seat: 0, A: 0, B: 1},
			engine.Event{Kind: engine.EvEffectSkipped, Seat: 0, Wonder: catalog.WonderCircusMaximus, A: int16(engine.PhaseDestroyCard)}))
		wdTurnPassed(t, &s, 0)
	})
}

func TestWonderTheology(t *testing.T) {
	t.Run("every wonder plays again", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "pyramids").Tokens(0, "theology").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderPyramids)
		ev := wdBuild(t, &s, "pyramids")
		wdEvents(t, "the pyramids", ev, append(wdPaid(t, 0, "pyramids", q), engine.Event{Kind: engine.EvExtraTurn, Seat: 0}))
		wdPlaysAgain(t, &s, 0)
	})
	t.Run("never twice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "sphinx").Tokens(0, "theology").State()
		ev := wdBuild(t, &s, "sphinx")
		if n := len(kindsOf(ev, engine.EvExtraTurn)); n != 1 {
			t.Fatalf("%d extra turns", n)
		}
		wdPlaysAgain(t, &s, 0)
	})
	t.Run("after a choice", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "statue-of-zeus").Tokens(0, "theology").City(1, "quarry").State()
		wdBuild(t, &s, "statue-of-zeus")
		if s.Phase != engine.PhaseDestroyCard || !s.ExtraTurn {
			t.Fatalf("phase %s, extra turn %v", s.Phase, s.ExtraTurn)
		}
		enginetest.Do(t, &s, "destroy quarry")
		wdPlaysAgain(t, &s, 0)
	})
	t.Run("only the owner's wonders", func(t *testing.T) {
		s := wdNew(t).Wonders(0, "pyramids").Tokens(1, "theology").State()
		wdBuild(t, &s, "pyramids")
		wdTurnPassed(t, &s, 0)
	})
}

// kindsOf returns the events of one kind.
func kindsOf(ev []engine.Event, k engine.EventKind) []engine.Event {
	var out []engine.Event
	for _, e := range ev {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

func TestWonderSevenWonderRule(t *testing.T) {
	t.Run("the opponent's last wonder", func(t *testing.T) {
		s := wdNew(t).
			Wonders(0, "pyramids", "colossus", "great-lighthouse", "sphinx").
			Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
			Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").
			Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "piraeus", "").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderSphinx)
		ev := wdBuild(t, &s, "sphinx")
		wdEvents(t, "the seventh wonder", ev, append(wdPaid(t, 0, "sphinx", q),
			engine.Event{Kind: engine.EvWonderRemoved, Seat: 1, Wonder: catalog.WonderHangingGardens},
			engine.Event{Kind: engine.EvExtraTurn, Seat: 0}))
		hg := s.Players[1].Wonders[3]
		if s.WondersBuilt != 7 || !hg.Removed || hg.Built {
			t.Fatalf("built %d, the hanging gardens %+v", s.WondersBuilt, hg)
		}
		wdWonderPoints(t, &s, 1, 2+4+2) // the removed wonder scores nothing
		// Seat 0 plays again (the Sphinx), and seat 1 then cannot build.
		for range 2 {
			for _, k := range enginetest.Keys(&s) {
				if strings.HasPrefix(k, "wonder:") {
					t.Fatalf("seat %d may still build a wonder: %s", s.ToMove, k)
				}
			}
			enginetest.Do(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String())
		}
	})
	t.Run("the builder's own last wonder", func(t *testing.T) {
		s := wdNew(t).
			Wonders(0, "pyramids", "colossus", "sphinx", "great-lighthouse").
			Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
			Built(0, "pyramids", "").Built(0, "colossus", "").
			Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "piraeus", "").Built(1, "hanging-gardens", "").State()
		q := engine.WonderQuote(&s, 0, catalog.WonderSphinx)
		ev := wdBuild(t, &s, "sphinx")
		wdEvents(t, "the seventh wonder", ev, append(wdPaid(t, 0, "sphinx", q),
			engine.Event{Kind: engine.EvWonderRemoved, Seat: 0, Wonder: catalog.WonderGreatLighthouse},
			engine.Event{Kind: engine.EvExtraTurn, Seat: 0}))
		if gl := s.Players[0].Wonders[3]; !gl.Removed || gl.Built || s.Players[0].NChoice != 0 {
			t.Fatalf("the great lighthouse %+v, flexible producers %d", gl, s.Players[0].NChoice)
		}
		wdWonderPoints(t, &s, 0, 9+3+6)
	})
	t.Run("before the seventh wonder's choice", func(t *testing.T) {
		// Seat 1 builds the Mausoleum as the seventh wonder: seat 0's last
		// wonder goes first, then the choice opens.
		s := wdNew(t).ToMove(1).Coins(1, 40).Discard("baths").
			Wonders(0, "pyramids", "colossus", "great-lighthouse", "sphinx").
			Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
			Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").
			Built(1, "great-library", "").Built(1, "piraeus", "").Built(1, "hanging-gardens", "").State()
		q := engine.WonderQuote(&s, 1, catalog.WonderMausoleum)
		ev := wdBuild(t, &s, "mausoleum")
		wdEvents(t, "the seventh wonder", ev, append(wdPaid(t, 1, "mausoleum", q),
			engine.Event{Kind: engine.EvWonderRemoved, Seat: 0, Wonder: catalog.WonderSphinx}))
		if s.Phase != engine.PhaseBuildFromDiscard || !s.Players[0].Wonders[3].Removed {
			t.Fatalf("phase %s, the sphinx %+v", s.Phase, s.Players[0].Wonders[3])
		}
		enginetest.Do(t, &s, "revive baths")
		wdTurnPassed(t, &s, 1)
	})
}

func TestWonderEffectOrder(t *testing.T) {
	// The seventh wonder is the Appian Way, against an opponent with
	// Economy and no coins: the payment (whose trading coins go to the
	// opponent), the seven-wonder rule, the builder's coins, the extra
	// turn, then the opponent's loss, taken from the Economy income.
	s := wdNew(t).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "appian-way").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").
		Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "piraeus", "").
		Tokens(1, "economy").Coins(1, 0).State()
	q := engine.WonderQuote(&s, 0, catalog.WonderAppianWay)
	if q.ToOpponent < 3 || q.ToOpponent != q.Total {
		t.Fatalf("quote %+v", q)
	}
	ev := wdBuild(t, &s, "appian-way")
	wdEvents(t, "the appian way", ev, append(wdPaid(t, 0, "appian-way", q),
		engine.Event{Kind: engine.EvWonderRemoved, Seat: 1, Wonder: catalog.WonderHangingGardens},
		engine.Event{Kind: engine.EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: 3, B: 40 - q.Total + 3},
		engine.Event{Kind: engine.EvExtraTurn, Seat: 0},
		engine.Event{Kind: engine.EvCoins, Seat: 1, Wonder: catalog.WonderAppianWay, A: -3, B: q.ToOpponent - 3}))
}
