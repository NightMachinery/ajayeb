package rules

import (
	"sort"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func TestScienceOracle(t *testing.T) {
	leaves := 0
	for compass := uint8(0); compass <= 7; compass++ {
		for gear := uint8(0); gear <= 7; gear++ {
			for tablet := uint8(0); tablet <= 7; tablet++ {
				for choices := uint8(0); choices <= 7; choices++ {
					counts := [catalog.NumSymbols]uint8{catalog.Compass: compass, catalog.Gear: gear, catalog.Tablet: tablet}
					best := int16(-1)
					var bestAssignment [catalog.NumSymbols]uint8
					var visit func(uint8, [catalog.NumSymbols]uint8)
					visit = func(left uint8, picks [catalog.NumSymbols]uint8) {
						if left != 0 {
							for symbol := catalog.Compass; symbol <= catalog.Tablet; symbol++ {
								p := picks
								p[symbol]++
								visit(left-1, p)
							}
							return
						}
						leaves++
						a, b, c := int16(counts[catalog.Compass]+picks[catalog.Compass]), int16(counts[catalog.Gear]+picks[catalog.Gear]), int16(counts[catalog.Tablet]+picks[catalog.Tablet])
						sets := a
						if b < sets {
							sets = b
						}
						if c < sets {
							sets = c
						}
						score := a*a + b*b + c*c + 7*sets
						if score > best {
							best = score
							bestAssignment = picks
						}
					}
					visit(choices, [catalog.NumSymbols]uint8{})
					got := scienceScore(counts, choices)
					if got.Points != best || got.Choices != bestAssignment {
						t.Fatalf("science %v plus%d: %+v want%d/%v", counts, choices, got, best, bestAssignment)
					}
					for s := catalog.Compass; s <= catalog.Tablet; s++ {
						if got.Counts[s] != counts[s]+got.Choices[s] {
							t.Fatal("reported assignment/counts differ")
						}
					}
				}
			}
		}
	}
	t.Logf("independent ordered assignments checked: %d", leaves)
}

func TestScoreSevenCategories(t *testing.T) {
	g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, 64)
	g.State.Cities[0].Coins = 8
	g.State.Cities[0].Victories = [NumAges]uint8{2, 1, 2}
	g.State.Cities[0].Defeats = 3
	want := Score{Wonder: 8, Treasure: 2, Military: 12, Blue: 6, Yellow: 3, Green: 2, Purple: 6, Total: 39}
	if got := scoreCity(&g.State, 0); got != want {
		t.Fatalf("categories %+v want %+v", got, want)
	}
	g.State.Result = scoreResult(&g.State)
	if err := g.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestScoreRanksAtEveryTieLevel(t *testing.T) {
	type city struct {
		card  catalog.DesignID
		coins int16
	}
	for _, tc := range []struct {
		name    string
		cities  []city
		ranks   []uint8
		winners uint8
	}{
		{"total before coins", []city{{57, 0}, {18, 14}, {0, 0}}, []uint8{1, 2, 3}, 1},
		{"coins break total tie", []city{{17, 3}, {18, 4}, {0, 0}}, []uint8{2, 1, 3}, 2},
		{"shared victory", []city{{17, 4}, {18, 4}, {0, 0}}, []uint8{1, 1, 3}, 3},
		{"shared lower rank skips places", []city{{57, 0}, {17, 3}, {18, 3}, {0, 0}}, []uint8{1, 2, 2, 4}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := State{Seats: uint8(len(tc.cities)), Phase: PhaseOver, Age: 3, Turn: 6, Decider: NoSeat}
			for s, fixture := range tc.cities {
				c := City{Board: catalog.BoardID(s + 1), Side: catalog.Day, Coins: fixture.coins}
				if fixture.card != 0 {
					c.Buildings.Add(fixture.card)
				}
				var err error
				st.Cities[s], err = cityCaches(c)
				if err != nil {
					t.Fatal(err)
				}
			}
			before := st
			got := scoreResult(&st)
			if !got.Over || got.Winners != tc.winners {
				t.Fatal("winner tie rule", got)
			}
			for s := 0; s < MaxSeats; s++ {
				if s < len(tc.ranks) {
					if got.Rank[s] != tc.ranks[s] {
						t.Fatal("competition rank", got)
					}
				} else if got.Rank[s] != 0 || got.Scores[s] != (Score{}) {
					t.Fatal("unused result slots nonzero")
				}
			}
			if st != before {
				t.Fatal("scoring mutated input")
			}
		})
	}
}

func TestScoreTreasureAndNegativeMilitary(t *testing.T) {
	st := State{Seats: 3}
	c, err := cityCaches(City{Board: 1, Side: catalog.Day, Defeats: 6})
	if err != nil {
		t.Fatal(err)
	}
	st.Cities[0] = c
	for coins := int16(0); coins < 10; coins++ {
		st.Cities[0].Coins = coins
		got := scoreCity(&st, 0)
		if got.Treasure != coins/3 || got.Military != -6 || got.Total != coins/3-6 {
			t.Fatal("rounding/negative military", coins, got)
		}
	}
	st.Cities[0].Coins = 130
	if got := scoreCity(&st, 0); got.Treasure != 43 {
		t.Fatal("coin supply capped scoring", got)
	}
}

func TestEffectKindTimingInventory(t *testing.T) {
	// End-of-turn scheduling is the next slice; the cache and scoring/effect
	// functions here must recognize every other timing and ignore that phase.
	var seen [catalog.NumEffectKinds]bool
	effects := []catalog.Effect{}
	for _, d := range catalog.BaseDesigns() {
		effects = append(effects, d.Effects...)
	}
	for _, b := range catalog.BaseBoards() {
		for _, side := range b.Sides {
			for _, s := range side.Stages[:side.NumStages] {
				effects = append(effects, s.Effects...)
			}
		}
	}
	g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, 0)
	for _, e := range effects {
		seen[e.Kind] = true
		st := g.State
		if _, err := applyWhenPlaced(&st, 0, []catalog.Effect{e}); err != nil {
			t.Fatal(e.Kind, err)
		}
		endPoints(&g.State, 0, e)
		c := g.State.Cities[0]
		if err := cacheEffect(&c, e); err != nil {
			t.Fatal(e.Kind, err)
		}
	}
	for kind := catalog.EffectKind(1); kind < catalog.NumEffectKinds; kind++ {
		if !seen[kind] {
			t.Fatal("effect kind lacks a fixture", kind)
		}
	}
}

func TestScienceChoiceHoldersShareGreen(t *testing.T) {
	g := effectWorld(t, catalog.BoardBabylon, catalog.Night, 2, 76)
	score := scoreCity(&g.State, 0)
	if score.Green != 13 || score.Purple != 6 || score.Wonder != 0 {
		t.Fatal("guild and stage science categories", score)
	}
	assignment := scienceScore(g.State.Cities[0].Science, g.State.Cities[0].ScienceChoice)
	if assignment.Choices != ([catalog.NumSymbols]uint8{catalog.Compass: 1, catalog.Gear: 1}) {
		t.Fatal("science optimum tie not canonical", assignment)
	}
}

func TestDecoratorCompleteUsesOwnerSide(t *testing.T) {
	d, _ := catalog.DesignByID(73)
	for _, stages := range []uint8{2, 3} {
		g := effectWorld(t, catalog.BoardGizah, catalog.Day, stages, 73)
		want := int16(0)
		if stages == 3 {
			want = 7
		}
		if got := endPoints(&g.State, 0, d.Effects[0]); got != want {
			t.Fatal("decorator completion predicate", stages, got)
		}
	}
}

// This score oracle uses literal card ids/rules and raw buildings, not the
// production effect interpreter, colour/science caches, or rank algorithm.
func referenceResult(st *State) Result {
	var colors [MaxSeats][catalog.NumColors]int16
	for seat, c := range st.Cities {
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if c.Buildings.Has(id) {
				d, _ := catalog.DesignByID(id)
				colors[seat][d.Color]++
			}
		}
	}
	blue := [catalog.NumDesigns + 1]int16{17: 3, 18: 3, 19: 3, 20: 3, 39: 5, 40: 4, 41: 4, 42: 4, 56: 5, 57: 8, 58: 7, 59: 6, 60: 6}
	wonder := [catalog.NumBoards + 1][2][MaxStages]int16{
		1: {{3, 0, 7}, {0, 0, 7}}, 2: {{3, 0, 7}, {0, 0}}, 3: {{3, 0, 7}, {2, 3, 5}}, 4: {{3, 5, 7}, {3, 5, 5, 7}}, 5: {{3, 0, 7}, {2, 1, 0}}, 6: {{3, 0, 7}, {2, 3, 5}}, 7: {{3, 0, 7}, {3, 4}},
	}
	result := Result{Over: true}
	for seat := Seat(0); seat < Seat(st.Seats); seat++ {
		c := &st.Cities[seat]
		left, right := seat.Left(st.Seats), seat.Right(st.Seats)
		score := Score{Treasure: c.Coins / 3, Military: int16(c.Victories[0]) + 3*int16(c.Victories[1]) + 5*int16(c.Victories[2]) - int16(c.Defeats)}
		for k := uint8(0); k < c.Stages; k++ {
			score.Wonder += wonder[c.Board][c.Side-1][k]
		}
		for id, want := range blue {
			if c.Buildings.Has(catalog.DesignID(id)) {
				score.Blue += want
			}
		}
		own := colors[seat]
		if c.Buildings.Has(61) {
			score.Yellow += int16(c.Stages)
		}
		if c.Buildings.Has(62) {
			score.Yellow += 2 * own[catalog.Grey]
		}
		if c.Buildings.Has(63) {
			score.Yellow += own[catalog.Brown]
		}
		if c.Buildings.Has(64) {
			score.Yellow += own[catalog.Yellow]
		}
		if c.Buildings.Has(65) {
			score.Yellow += own[catalog.Red]
		}
		if c.Buildings.Has(71) {
			score.Purple += int16(c.Stages) + int16(st.Cities[left].Stages) + int16(st.Cities[right].Stages)
		}
		if c.Buildings.Has(72) {
			score.Purple += 2 * (colors[left][catalog.Grey] + colors[right][catalog.Grey])
		}
		if c.Buildings.Has(73) {
			complete := uint8(3)
			if c.Side == catalog.Night {
				switch c.Board {
				case catalog.BoardBabylon, catalog.BoardRhodos:
					complete = 2
				case catalog.BoardGizah:
					complete = 4
				}
			}
			if c.Stages == complete {
				score.Purple += 7
			}
		}
		for _, pair := range []struct {
			id    catalog.DesignID
			color catalog.Color
		}{{74, catalog.Blue}, {75, catalog.Green}, {78, catalog.Red}, {79, catalog.Yellow}, {80, catalog.Brown}} {
			if c.Buildings.Has(pair.id) {
				score.Purple += colors[left][pair.color] + colors[right][pair.color]
			}
		}
		if c.Buildings.Has(77) {
			score.Purple += own[catalog.Brown] + own[catalog.Grey] + own[catalog.Purple]
		}
		var sci [3]int16
		for symbol, ids := range [3][4]catalog.DesignID{{25, 47, 66, 67}, {27, 48, 68, 69}, {26, 49, 50, 70}} {
			for _, id := range ids {
				if c.Buildings.Has(id) {
					sci[symbol]++
				}
			}
		}
		choices := 0
		if c.Buildings.Has(76) {
			choices++
		}
		if c.Board == catalog.BoardBabylon && c.Stages >= 2 {
			choices++
		}
		combinations := 1
		for k := 0; k < choices; k++ {
			combinations *= 3
		}
		best := int16(-1)
		for code := 0; code < combinations; code++ {
			counts := sci
			digits := code
			for k := 0; k < choices; k++ {
				counts[digits%3]++
				digits /= 3
			}
			points := counts[0]*counts[0] + counts[1]*counts[1] + counts[2]*counts[2] + 7*min(counts[0], counts[1], counts[2])
			if points > best {
				best = points
			}
		}
		score.Green = best
		score.Total = score.Wonder + score.Treasure + score.Military + score.Blue + score.Yellow + score.Green + score.Purple
		result.Scores[seat] = score
	}
	order := make([]int, st.Seats)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if result.Scores[a].Total != result.Scores[b].Total {
			return result.Scores[a].Total > result.Scores[b].Total
		}
		return st.Cities[a].Coins > st.Cities[b].Coins
	})
	rank := uint8(1)
	for i, seat := range order {
		if i > 0 {
			prev := order[i-1]
			if result.Scores[prev].Total != result.Scores[seat].Total || st.Cities[prev].Coins != st.Cities[seat].Coins {
				rank = uint8(i + 1)
			}
		}
		result.Rank[seat] = rank
		if rank == 1 {
			result.Winners |= 1 << seat
		}
	}
	return result
}

func TestScoreInvariantRejectsBalancedFabrication(t *testing.T) {
	g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, 64)
	g.State.Result.Scores[0].Blue++
	g.State.Result.Scores[0].Total++
	if err := checkResult(&g.State); err != nil {
		t.Fatal("negative fixture is not internally balanced", err)
	}
	if err := g.CheckInvariants(); err == nil {
		t.Fatal("fabricated but balanced final score accepted")
	}
}
