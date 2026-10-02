package engine_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Tests of scoring through the public API: the eight categories of the score
// sheet, the guilds' points, the military points by zone, the coins, the
// winner of a civilian game with its tie-break and shared victory, and the
// score preview.

func TestScoreSheet(t *testing.T) {
	s := enginetest.New(t, 41).Age(3).
		Wonders(0, "pyramids", "sphinx").Wonders(1, "colossus").
		Built(0, "pyramids", "").Built(0, "sphinx", "").Built(1, "colossus", "").
		City(0, "theater", "baths", "workshop", "apothecary", "tavern", "lighthouse", "merchants-guild").
		City(1, "palace", "library", "stone-reserve", "clay-reserve", "wood-reserve").
		Tokens(0, "agriculture", "mathematics").Tokens(1, "philosophy").
		Coins(0, 14).Coins(1, 5).Pawn(4).State()
	got := engine.Scores(&s)
	want := [2]engine.Score{
		// Merchants Guild: 3 yellow cards in seat 1's city, against 2.
		// Mathematics: 3 per token, 2 tokens. Pawn on space 4: 5 points.
		{Blue: 6, Green: 2, Yellow: 3, Purple: 3, Wonders: 15, Tokens: 4 + 6, Coins: 4, Military: 5, Total: 48},
		{Blue: 7, Green: 2, Yellow: 0, Purple: 0, Wonders: 3, Tokens: 7, Coins: 1, Military: 0, Total: 20},
	}
	if got != want {
		t.Fatalf("scores\n%+v\n%+v\nwant\n%+v\n%+v", got[0], got[1], want[0], want[1])
	}
}

func TestScoreGuildPoints(t *testing.T) {
	cases := []struct {
		guild  string
		mine   []string
		theirs []string
		coins  [2]int
		built  [2]int // wonders built by each seat
		want   int16
	}{
		{guild: "merchants-guild", mine: []string{"tavern"}, theirs: []string{"stone-reserve", "clay-reserve", "wood-reserve"}, want: 3},
		{guild: "merchants-guild", mine: []string{"tavern", "brewery"}, theirs: []string{"forum"}, want: 2},
		// Brown and grey cards count together, from one city: 3 against 1 + 2,
		// not the 3 brown of one city plus the 2 grey of the other.
		{guild: "shipowners-guild", mine: []string{"lumber-yard", "quarry", "clay-pool"}, theirs: []string{"logging-camp", "glassworks", "press"}, want: 3},
		{guild: "shipowners-guild", mine: []string{"press"}, theirs: []string{"logging-camp", "glassworks", "sawmill", "brickyard"}, want: 4},
		{guild: "builders-guild", built: [2]int{1, 3}, want: 6},
		{guild: "builders-guild", built: [2]int{2, 0}, want: 4},
		{guild: "magistrates-guild", mine: []string{"theater", "altar", "baths"}, theirs: []string{"temple"}, want: 3},
		{guild: "scientists-guild", theirs: []string{"scriptorium", "pharmacist"}, want: 2},
		{guild: "moneylenders-guild", coins: [2]int{4, 14}, want: 4},
		{guild: "moneylenders-guild", coins: [2]int{11, 2}, want: 3},
		{guild: "tacticians-guild", mine: []string{"stable"}, theirs: []string{"palisade", "garrison"}, want: 2},
		{guild: "tacticians-guild", want: 0},
	}
	all := []string{"pyramids", "sphinx", "colossus", "piraeus", "mausoleum", "great-library", "appian-way", "circus-maximus"}
	for _, c := range cases {
		b := enginetest.New(t, 42).Age(3).Wonders(0, all[:4]...).Wonders(1, all[4:]...).
			City(0, c.guild).City(0, c.mine...).City(1, c.theirs...).Coins(0, c.coins[0]).Coins(1, c.coins[1])
		for i := range c.built[0] {
			b.Built(0, all[i], "")
		}
		for i := range c.built[1] {
			b.Built(1, all[4+i], "")
		}
		s := b.State()
		sc := engine.Scores(&s)
		if sc[0].Purple != c.want || sc[1].Purple != 0 {
			t.Errorf("%s (mine %v, theirs %v, coins %v, wonders %v): %d and %d guild points, want %d",
				c.guild, c.mine, c.theirs, c.coins, c.built, sc[0].Purple, sc[1].Purple, c.want)
		}
	}
}

func TestScoreMilitaryPoints(t *testing.T) {
	cases := []struct {
		pawn int
		want [2]int16
	}{
		{0, [2]int16{0, 0}},
		{1, [2]int16{2, 0}}, {2, [2]int16{2, 0}},
		{3, [2]int16{5, 0}}, {5, [2]int16{5, 0}},
		{6, [2]int16{10, 0}}, {8, [2]int16{10, 0}},
		{-1, [2]int16{0, 2}}, {-2, [2]int16{0, 2}},
		{-3, [2]int16{0, 5}}, {-5, [2]int16{0, 5}},
		{-6, [2]int16{0, 10}}, {-8, [2]int16{0, 10}},
	}
	for _, c := range cases {
		s := enginetest.New(t, 43).Pawn(c.pawn).State()
		sc := engine.Scores(&s)
		if got := [2]int16{sc[0].Military, sc[1].Military}; got != c.want {
			t.Errorf("pawn %d: military points %v, want %v", c.pawn, got, c.want)
		}
	}
}

func TestScoreCoinsAndWonders(t *testing.T) {
	for coins, want := range map[int]int16{0: 0, 2: 0, 3: 1, 14: 4, 15: 5, 40: 13} {
		s := enginetest.New(t, 44).Coins(0, coins).State()
		if got := engine.Scores(&s)[0].Coins; got != want {
			t.Errorf("%d coins: %d points, want %d", coins, got, want)
		}
	}
	// Only built wonders score; the wonder removed by the seven-wonder rule
	// scores nothing.
	s := enginetest.New(t, 44).Coins(0, 0).Coins(1, 0).
		Wonders(0, "pyramids", "sphinx", "colossus", "piraeus").
		Wonders(1, "mausoleum", "great-library", "appian-way", "circus-maximus").
		Built(0, "pyramids", "").Built(0, "sphinx", "").Built(0, "colossus", "").
		Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "appian-way", "").Built(1, "circus-maximus", "").State()
	if !s.Players[0].Wonders[3].Removed {
		t.Fatal("Piraeus was not removed")
	}
	sc := engine.Scores(&s)
	if sc[0].Wonders != 9+6+3 || sc[1].Wonders != 2+4+3+3 {
		t.Fatalf("wonder points %d and %d, want 18 and 12", sc[0].Wonders, sc[1].Wonders)
	}
}

// TestCivilianResult plays the last card of Age III and checks the winner:
// the higher total, then the more blue points, then a shared victory.
func TestCivilianResult(t *testing.T) {
	cases := []struct {
		name   string
		mine   []string
		theirs []string
		winner int8
	}{
		{"the higher total wins, with fewer blue points", []string{"theater"}, []string{"library", "workshop", "apothecary"}, 1},
		{"equal totals: more blue points win", []string{"theater"}, []string{"library", "workshop"}, 0},
		{"equal totals: more blue points win (seat 1)", []string{"library", "apothecary"}, []string{"altar"}, 1},
		{"equal totals and blue points: a shared victory", []string{"theater"}, []string{"altar"}, engine.SharedVictory},
		{"nothing at all: a shared victory", nil, nil, engine.SharedVictory},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := enginetest.New(t, 45).Age(3).OnlyLeft("palace").FirstPlayer(0).
				Coins(0, 0).Coins(1, 0).City(0, c.mine...).City(1, c.theirs...).State()
			ev := enginetest.Do(t, &s, "discard palace") // 2 coins: no point
			if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Winner != c.winner {
				t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
			}
			if s.Result.Scores != engine.Scores(&s) {
				t.Fatalf("result scores %+v, Scores %+v", s.Result.Scores, engine.Scores(&s))
			}
			if last := ev[len(ev)-1]; last != (engine.Event{Kind: engine.EvGameOver, A: int16(engine.VictoryCivilian), B: int16(c.winner)}) {
				t.Fatalf("last event %+v", last)
			}
		})
	}
}

// TestScorePreview: Scores is the sheet as if the game ended now, and reading
// it changes nothing.
func TestScorePreview(t *testing.T) {
	s := enginetest.New(t, 46).Age(2).City(1, "theater", "glassworks").Coins(1, 9).Pawn(-4).
		Tokens(1, "philosophy").State()
	h := engine.Hash(&s)
	got := engine.Scores(&s)
	if got[1] != (engine.Score{Blue: 3, Tokens: 7, Coins: 3, Military: 5, Total: 18}) || got[0].Military != 0 {
		t.Fatalf("preview %+v", got)
	}
	if engine.Scores(&s) != got || engine.Hash(&s) != h {
		t.Fatal("Scores is not pure")
	}
	if s.Result != (engine.Result{}) {
		t.Fatal("the preview set a result")
	}
}

// TestCatalogPointsByColour checks the facts the score sheet relies on: the
// brown, grey and red buildings, which have no category, carry no points,
// and only guilds score by counting.
func TestCatalogPointsByColour(t *testing.T) {
	for d := range catalog.Cards() {
		for _, e := range d.Effects {
			switch e.Kind {
			case catalog.EffPoints:
				if d.Color == catalog.Brown || d.Color == catalog.Grey || d.Color == catalog.Red {
					t.Errorf("%s (%s) has %d points but its colour has no score category", d.Slug, d.Color, e.N)
				}
			case catalog.EffPointsPerCard, catalog.EffPointsPerWonder, catalog.EffPointsPerCoins:
				if d.Color != catalog.Purple {
					t.Errorf("%s (%s) scores by counting but is not a guild", d.Slug, d.Color)
				}
			}
		}
	}
}
