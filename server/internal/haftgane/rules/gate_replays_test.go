package rules

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Cases bind a named materialized scenario builder by its complete initial
// value hash. Each line retains readable keys and hashes the full fixed-value
// game, raw events and every seat's offered choices at each point. No large
// repeated states, event arrays or views are stored. Existing view fixtures
// cover readable DTO shapes; these are replay/ordering stability fixtures.
type gateReplayFrame struct {
	Batch  Batch  `json:"batch"`
	Offers string `json:"offers_sha256"`
	Game   string `json:"game_sha256"`
	Events string `json:"events_sha256"`
}
type gateReplay struct {
	Name    string            `json:"name"`
	Initial string            `json:"initial_sha256"`
	Frames  []gateReplayFrame `json:"frames"`
}
type gateScript struct {
	t          *testing.T
	root, game Game
	last       []Event
	golden     gateReplay
}

func gateGameHash(g *Game) string {
	return fmt.Sprintf("%x", sha256.Sum256(appendGameEncoding(nil, g)))
}
func gateJSONHash(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func newGateScript(t *testing.T, name string, g Game) *gateScript {
	t.Helper()
	if err := g.CheckInvariants(); err != nil {
		t.Fatal(name, err)
	}
	return &gateScript{t: t, root: g, game: g, golden: gateReplay{Name: name, Initial: gateGameHash(&g), Frames: []gateReplayFrame{}}}
}
func (s *gateScript) step(b Batch) {
	s.t.Helper()
	var offers [MaxSeats][]Choice
	for seat := Seat(0); seat < Seat(s.game.State.Seats); seat++ {
		offers[seat] = s.game.Legal(seat)
	}
	frozen := s.game
	next, events, err := resolveAbility(s.t, s.game, b)
	if err != nil {
		s.t.Fatal(s.golden.Name, err)
	}
	if s.game != frozen {
		s.t.Fatal("caller mutated")
	}
	if err := next.CheckInvariants(); err != nil {
		s.t.Fatal(s.golden.Name, err)
	}
	if len(events) > MaxResolutionEvents {
		s.t.Fatal("event limit")
	}
	s.golden.Frames = append(s.golden.Frames, gateReplayFrame{Batch: b, Offers: gateJSONHash(s.t, offers), Game: gateGameHash(&next), Events: gateJSONHash(s.t, events)})
	s.game = next
	s.last = events
}
func (s *gateScript) finish() gateReplay {
	s.t.Helper()
	// Replay the stored batches from a materialized root, rather than rerunning
	// the generating policy. Scenarios may deliberately retain partial history.
	setup := Setup{Seed: s.root.State.Seed, Seats: s.root.State.Seats, Options: s.root.State.Opts, Scenario: ScenarioFromGame(s.root)}
	batches := make([]Batch, 0, len(s.golden.Frames))
	for _, frame := range s.golden.Frames {
		batches = append(batches, frame.Batch)
	}
	got, err := Replay(setup, batches)
	if err != nil || got != s.game {
		s.t.Fatal("materialized replay changed", s.golden.Name, err)
	}
	return s.golden
}
func gateResources() []catalog.DesignID {
	return []catalog.DesignID{2, 4, 5, 7, 8, 11, 12, 13, 28, 29, 30, 31}
}

// Drain remaining copies into distinct non-owner public cities to construct a
// valid partial-history scenario with an empty (or one duplicate) pile. Keep
// each other city's first hand name unbuilt, so it can construct rather than
// sell at the triggering point. This does not claim natural reachability.
func gateDrainPile(t *testing.T, g Game, duplicate bool) Game {
	t.Helper()
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		design, _ := catalog.DesignByID(id)
		reserve := uint8(0)
		if duplicate && id == 11 {
			reserve = 1
		}
		if g.State.Discard[id] < reserve {
			t.Fatal("duplicate pile supply")
		}
		for g.State.Discard[id] > reserve {
			chosen := NoSeat
			count := 22
			for seat := Seat(1); seat < Seat(g.State.Seats); seat++ {
				city := &g.State.Cities[seat]
				n := city.Buildings.Count() + int(city.Stages)
				if g.State.Hands[seat][0] == id || !city.Buildings.And(catalog.T.ByName[design.NameID]).Empty() || n >= 20 {
					continue
				}
				if n < count {
					chosen, count = seat, n
				}
			}
			if chosen == NoSeat {
				t.Fatal("cannot drain pile", id)
			}
			g.State.Cities[chosen].Buildings.Add(id)
			g.State.Discard[id]--
		}

	}
	out, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func gateNoSales(t *testing.T, g Game, a Action) Batch {
	t.Helper()
	b := actionBatch(g, 0, a)
	for i, p := range b.Picks {
		if p.Seat == 0 {
			continue
		}
		found := false
		for _, c := range g.Legal(p.Seat) {
			if c.Kind == ActBuild || c.Kind == ActStage {
				b.Picks[i].Key = c.Key()
				found = true
				break
			}
		}
		if !found {
			t.Fatal("non-selling counterpart missing", p.Seat)
		}
	}
	return b
}
func gateReplayCases(t *testing.T) []gateReplay {
	t.Helper()
	cases := []gateReplay{}
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for _, policy := range []string{"random", "simple"} {
			g, err := NewGame(Setup{Seed: 190 + uint64(n), Seats: n, Options: Options{Sides: SidesChoose}})
			if err != nil {
				t.Fatal(err)
			}
			s := newGateScript(t, fmt.Sprintf("natural-N%d-%s", n, policy), g)
			r := rng.NewRNG2(uint64(n), 0x676f6c64656e)
			for s.game.State.Phase != PhaseOver {
				b := Batch{Point: s.game.Point().ID(), Picks: []Pick{}}
				for seat := Seat(0); seat < Seat(n); seat++ {
					if s.game.Point().Required.Has(seat) {
						b.Picks = append(b.Picks, Pick{Seat: seat, Key: gateChoice(&s.game, seat, policy, &r).Key()})
					}
				}
				s.step(b)
			}
			cases = append(cases, s.finish())
		}
	}
	stages := 0
	for board := catalog.BoardID(1); board <= catalog.NumBoards; board++ {
		for _, side := range []catalog.Side{catalog.Day, catalog.Night} {
			def, _ := catalog.BoardByID(board)
			for stage := uint8(0); stage < def.Sides[side-1].NumStages; stage++ {
				g := abilityScenario(t, 3, 3, [MaxSeats]wonderCity{0: {board: board, side: side, stages: stage, buildings: gateResources()}}, [MaxSeats][]catalog.DesignID{0: {51}})
				s := newGateScript(t, fmt.Sprintf("board-%d-side-%d-stage-%d", board, side, stage+1), g)
				s.step(actionBatch(g, 0, stagePick(t, g, 0, 51)))
				if s.game.State.Cities[0].Stages != stage+1 {
					t.Fatal("stage not placed")
				}
				if s.game.State.Phase == PhaseDiscardBuild {
					s.step(salesBatch(s.game))
				}
				cases = append(cases, s.finish())
				stages++
			}
		}
	}
	if stages != 41 {
		t.Fatal("missing stage paths", stages)
	}
	for age := uint8(1); age <= 3; age++ {
		cards := [3][2]catalog.DesignID{{17, 18}, {39, 40}, {56, 57}}
		g := abilityScenario(t, age, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, buildings: []catalog.DesignID{8}}, 1: {board: catalog.BoardGizah}}, [MaxSeats][]catalog.DesignID{0: cards[age-1][:]})
		s := newGateScript(t, fmt.Sprintf("babylon-age-%d-stage-on-sixth", age), g)
		s.step(actionBatch(g, 0, stagePick(t, g, 0, cards[age-1][0])))
		if s.game.State.Phase != PhaseLastCard || s.game.State.Cities[0].Kept != cards[age-1][1] {
			t.Fatal("missing kept card")
		}
		s.step(requiredBatch(s.game, 0, Action{Kind: ActSell, Design: cards[age-1][1]}))
		cases = append(cases, s.finish())
	}
	for _, path := range []struct {
		side  catalog.Side
		stage uint8
	}{{catalog.Day, 2}, {catalog.Night, 1}, {catalog.Night, 2}, {catalog.Night, 3}} {
		for _, pile := range []string{"empty", "duplicates", "sixth-leftover"} {
			turn := uint8(3)
			wanted := []catalog.DesignID{51}
			if pile == "sixth-leftover" {
				turn = 6
				wanted = append(wanted, 71)
			}
			g := abilityScenario(t, 3, turn, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: path.side, stages: path.stage - 1, buildings: gateResources()}}, [MaxSeats][]catalog.DesignID{0: wanted})
			if turn != 6 {
				g = gateDrainPile(t, g, pile == "duplicates")
			}
			s := newGateScript(t, fmt.Sprintf("halik-side-%d-stage-%d-%s", path.side, path.stage, pile), g)
			a := stagePick(t, g, 0, 51)
			b := actionBatch(g, 0, a)
			if turn != 6 {
				b = gateNoSales(t, g, a)
			}
			s.step(b)
			if turn == 6 {
				if s.game.State.Phase != PhaseDiscardBuild || len(buildChoices(s.game.Legal(0), ActBuildDiscarded, 71)) != 1 {
					t.Fatal("own sixth leftover not offered")
				}
				s.step(requiredBatch(s.game, 0, Action{Kind: ActBuildDiscarded, Design: 71}))
				if s.game.State.Phase != PhaseOver || !s.game.State.Cities[0].Buildings.Has(71) {
					t.Fatal("sixth rebuild did not precede ending")
				}
			} else {
				skips := 0
				for _, event := range s.last {
					if event.Kind == EvDiscardBuildNothing && event.Seat == 0 {
						skips++
					}
				}
				if skips != 1 {
					t.Fatal("skipped search must emit exactly one owner opportunity", pile, skips)
				}
				if s.game.State.Phase != PhaseTurn || s.game.State.Turn != 4 {
					t.Fatal("empty/duplicate pile opened choice")
				}
				count := 0
				for _, v := range s.game.State.Discard {
					count += int(v)
				}
				expected := 0
				if pile == "duplicates" {
					expected = 1
				}
				if count != expected {
					t.Fatal("wrong skip pile", pile, count)
				}
			}
			cases = append(cases, s.finish())
		}
	}
	// Day stage 2 fills its colour ledger from the city at placement; the next
	// red is free, but the pre-existing blue is not.
	g := abilityScenario(t, 2, 2, [MaxSeats]wonderCity{0: {board: catalog.BoardOlympia, stages: 1, buildings: []catalog.DesignID{5, 17}}, 1: {board: catalog.BoardBabylon}}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {28, 35, 40}})
	s := newGateScript(t, "olympia-day-first-colour", g)
	s.step(actionBatch(g, 0, stagePick(t, g, 0, 39)))
	assertFree(t, s.game.Legal(0), 35, FreeFirstColour)
	assertNotFree(t, s.game.Legal(0), 40)
	s.step(actionBatch(s.game, 0, buildChoices(s.game.Legal(0), ActBuild, 35)[0].Action))
	cases = append(cases, s.finish())
	for _, mode := range []string{"first-use", "first-lost-sale", "first-lost-stage", "last-use", "last-lost-sale", "last-lost-stage"} {
		turn, stages := uint8(1), uint8(1)
		switch mode {
		case "first-lost-stage":
			stages = 0
		case "last-use", "last-lost-sale":
			turn = 5
		case "last-lost-stage":
			turn = 6
		}
		buildings := []catalog.DesignID{7}
		if turn > 1 {
			buildings = append(buildings, 28)
		}
		g := abilityScenario(t, 2, turn, [MaxSeats]wonderCity{0: {board: catalog.BoardOlympia, side: catalog.Night, stages: stages, buildings: buildings}}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {35, 40}})
		s := newGateScript(t, "olympia-night-"+mode, g)
		switch mode {
		case "first-use":
			assertFree(t, g.Legal(0), 39, FreeFirstTurn)
			s.step(actionBatch(g, 0, buildChoices(g.Legal(0), ActBuild, 39)[0].Action))
		case "first-lost-sale":
			assertFree(t, g.Legal(0), 39, FreeFirstTurn)
			s.step(actionBatch(g, 0, Action{Kind: ActSell, Design: 39}))
		case "first-lost-stage", "last-lost-stage":
			s.step(actionBatch(g, 0, stagePick(t, g, 0, 39)))
		case "last-use", "last-lost-sale":
			s.step(actionBatch(g, 0, stagePick(t, g, 0, 39)))
			assertFree(t, s.game.Legal(0), 40, FreeLastTurn)
			if mode == "last-use" {
				s.step(actionBatch(s.game, 0, buildChoices(s.game.Legal(0), ActBuild, 40)[0].Action))
			} else {
				s.step(salesBatch(s.game))
			}
		}
		for _, c := range s.game.Legal(0) {
			if c.Free == FreeFirstTurn && s.game.State.Turn != 1 || c.Free == FreeLastTurn && s.game.State.Turn != 6 {
				t.Fatal("free window retained", mode)
			}
		}
		cases = append(cases, s.finish())
	}
	// All-sales N3 ties all scores and purses. A materialized last-point root
	// with one extra coin preserves the same treasure score but breaks the tie.
	g, err := NewGame(Setup{Seed: 77, Seats: 3, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	s = newGateScript(t, "three-seat-shared-victory", g)
	for s.game.State.Age != 3 || s.game.State.Turn != 6 {
		s.step(salesBatch(s.game))
	}
	tieRoot := s.game
	s.step(salesBatch(s.game))
	if s.game.State.Result.Winners != 7 {
		t.Fatal("missing shared victory")
	}
	cases = append(cases, s.finish())
	tieRoot.State.Cities[0].Coins++
	tieRoot, err = NewGameFromScenario(tieRoot.State.Seed, tieRoot.State.Opts, ScenarioFromGame(tieRoot))
	if err != nil {
		t.Fatal(err)
	}
	s = newGateScript(t, "three-seat-coin-tiebreak", tieRoot)
	s.step(salesBatch(tieRoot))
	if s.game.State.Result.Winners != 1 || s.game.State.Result.Scores[0].Total != s.game.State.Result.Scores[1].Total {
		t.Fatal("coin tiebreak not isolated")
	}
	cases = append(cases, s.finish())
	t.Logf("replay paths: %d natural, %d stages (14 board/sides), 3 sixth-turn Babylon ages, 12 Halik (4 each empty/duplicate/sixth), 7 Olympia, 2 shared/coin cases", 10, stages)
	if len(cases) != 75 {
		t.Fatal("incomplete replay inventory", len(cases))
	}
	return cases
}

func TestRulesGateReplayGoldens(t *testing.T) {
	actual := gateReplayCases(t)
	var stream bytes.Buffer
	for _, c := range actual {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		stream.Write(b)
		stream.WriteByte('\n')
	}
	path := filepath.Join("testdata", "replays", "replays.jsonl")
	if os.Getenv("UPDATE_HAFTGANE_REPLAYS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, stream.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(expected, stream.Bytes()) {
		f, err := os.CreateTemp("", "haftgane-replays-*.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(stream.Bytes()); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		first := "stream framing"
		lines := bytes.Split(bytes.TrimSpace(expected), []byte{'\n'})
		for i, c := range actual {
			if i >= len(lines) {
				first = c.Name + " missing expected case"
				break
			}
			var want gateReplay
			if err = json.Unmarshal(lines[i], &want); err != nil {
				first = fmt.Sprintf("case%d invalid expected JSON", i)
				break
			}
			if reflect.DeepEqual(want, c) {
				continue
			}
			first = c.Name + " initial state or frame count"
			for j, frame := range c.Frames {
				if j >= len(want.Frames) || !reflect.DeepEqual(want.Frames[j], frame) {
					first = fmt.Sprintf("%s frame%d (%s)", c.Name, j, frame.Batch.Point)
					break
				}
			}
			break
		}
		t.Fatalf("replay golden differs at %s; actual %s; inspect byte diff before UPDATE_HAFTGANE_REPLAYS=1 make test-go GO_PKGS=./internal/haftgane/rules GO_RUN=^TestRulesGateReplayGoldens$", first, f.Name())
	}
	t.Logf("%d compact replay cases, %d bytes", len(actual), stream.Len())
}
