package rules

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func scenarioNightGame(t *testing.T) Game {
	t.Helper()
	g, err := NewGame(Setup{Seed: 99, Seats: MaxSeats})
	if err != nil {
		t.Fatal(err)
	}
	var picks [MaxSeats]Action
	for s := range picks {
		picks[s] = Action{Kind: ActSide, Side: catalog.Night}
	}
	next, _, err := resolveSides(g, picks)
	if err != nil {
		t.Fatal(err)
	}
	return next
}

// Explicit scenarios move copies; they do not require playing irrelevant turns
// to put a board's ability under test. Continuous caches are reconstructed.
func scenarioPhases(t *testing.T) []Game {
	t.Helper()
	sides, err := NewGame(Setup{Seed: 1, Seats: 3})
	if err != nil {
		t.Fatal(err)
	}
	start := scenarioNightGame(t)
	discard := start
	var halik, babylon Seat
	for seat, c := range start.State.Cities {
		if c.Board == catalog.BoardHalikarnassos {
			halik = Seat(seat)
		}
		if c.Board == catalog.BoardBabylon {
			babylon = Seat(seat)
		}
	}
	discard.State.Phase = PhaseDiscardBuild
	discard.State.Decider = halik
	discard.State.Points++
	for seat := range discard.State.Hands {
		hand := &discard.State.Hands[seat]
		id := hand[0]
		copy(hand[:], hand[1:])
		hand[HandSize-1] = 0
		discard.Record.Picks[0][0][seat] = id
		if Seat(seat) == halik {
			discard.State.Cities[seat].Stages = 1
			discard.State.Cities[seat].Tucked[0] = id
			discard.Record.Plays[0][0][seat] = Play{Kind: ActStage, Stage: 1}
		} else {
			discard.State.Discard[id]++
			discard.Record.Plays[0][0][seat] = Play{Kind: ActSell}
		}
	}
	discard, err = NewGameFromScenario(discard.State.Seed, discard.State.Opts, ScenarioFromGame(discard))
	if err != nil {
		t.Fatal("discard fixture", err)
	}
	last := start
	last.State.Phase = PhaseLastCard
	last.State.Turn = MaxTurns
	last.State.Decider = babylon
	for seat, hand := range last.State.Hands {
		for k, id := range hand {
			switch {
			case Seat(seat) == babylon && k == 0:
				last.State.Cities[seat].Stages = 1
				last.State.Cities[seat].Tucked[0] = id
			case Seat(seat) == babylon && k == 1:
				last.State.Cities[seat].Kept = id
			default:
				last.State.Discard[id]++
			}
		}
		last.State.Hands[seat] = [HandSize]catalog.DesignID{}
	}
	alignScenarioHistory(t, &last)
	last, err = NewGameFromScenario(last.State.Seed, last.State.Opts, ScenarioFromGame(last))
	if err != nil {
		t.Fatal("last fixture", err)
	}
	over := start
	over.State.Phase = PhaseOver
	over.State.Age = NumAges
	over.State.Turn = TurnsPerAge
	over.State.Decider = NoSeat
	over.State.Result.Over = true
	over.State.Result.Winners = uint8(AllSeats(MaxSeats))
	for seat, hand := range over.State.Hands {
		for _, id := range hand {
			over.State.Discard[id]++
		}
		over.State.Hands[seat] = [HandSize]catalog.DesignID{}
		over.State.Result.Rank[seat] = 1
	}
	for age := 1; age < NumAges; age++ {
		over.Record.Packets[age] = over.State.Deals[age]
		for _, hand := range over.State.Deals[age] {
			for _, id := range hand {
				if id != 0 {
					over.State.Discard[id]++
				}
			}
		}
		over.State.Deals[age] = [MaxSeats][HandSize]catalog.DesignID{}
	}
	over.State.Result = scoreResult(&over.State)
	alignScenarioHistory(t, &over)
	if err := over.CheckInvariants(); err != nil {
		t.Fatal("over fixture", err)
	}
	return []Game{sides, start, discard, last, over}
}

func TestScenarioRoundTrips(t *testing.T) {
	for _, g := range scenarioPhases(t) {
		t.Run(g.State.Phase.String(), func(t *testing.T) {
			sc := ScenarioFromGame(g)
			data, err := json.Marshal(sc)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Scenario
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			next, err := NewGame(Setup{Seed: g.State.Seed, Seats: g.State.Seats, Options: g.State.Opts, Scenario: &decoded})
			if err != nil {
				t.Fatal(err)
			}
			if next != g || next.Fingerprint() != g.Fingerprint() {
				t.Fatal("scenario changed position or fingerprint")
			}
			sc.State.Cities[0].Coins++
			if next != g {
				t.Fatal("scenario aliased constructed game")
			}
		})
	}
	if _, err := NewGameFromScenario(0, Options{}, nil); err == nil {
		t.Fatal("nil scenario")
	}
	g := scenarioNightGame(t)
	sc := ScenarioFromGame(g)
	if _, err := NewGame(Setup{Seats: 3, Options: g.State.Opts, Scenario: sc}); err == nil {
		t.Fatal("seat mismatch")
	}
}

func TestScenarioCanonicalizesAndRebuilds(t *testing.T) {
	g := scenarioNightGame(t)
	sc := ScenarioFromGame(g)
	for s := range sc.State.Hands {
		h := &sc.State.Hands[s]
		for a, b := 0, HandSize-1; a < b; a, b = a+1, b-1 {
			h[a], h[b] = h[b], h[a]
		}
	}
	sc.State.Cities[0].Produce = [catalog.NumResources]uint8{99}
	sc.State.Cities[0].NChoice = 255
	got, err := NewGameFromScenario(g.State.Seed, g.State.Opts, sc)
	if err != nil {
		t.Fatal(err)
	}
	if got != g {
		t.Fatal("noncanonical scene did not normalize to original")
	}
	if sc.State.Cities[0].NChoice != 255 {
		t.Fatal("builder mutated source")
	}
}

func TestInvariantNegatives(t *testing.T) {
	g := scenarioNightGame(t)
	for _, tt := range []struct {
		name string
		edit func(*Game)
	}{
		{"missing copy", func(g *Game) { g.State.Hands[0][0] = 0 }},
		{"duplicate copy", func(g *Game) { g.State.Hands[0][0] = g.State.Hands[0][1] }},
		{"unknown card", func(g *Game) { g.State.Hands[0][0] = 255 }},
		{"negative purse", func(g *Game) { g.State.Cities[0].Coins = -1 }},
		{"board repeated", func(g *Game) { g.State.Cities[0].Board = g.State.Cities[1].Board }},
		{"stage overflow", func(g *Game) { g.State.Cities[0].Stages = 255 }},
		{"cache", func(g *Game) { g.State.Cities[0].Produce[0]++ }},
		{"set unused bit", func(g *Game) { g.State.Cities[0].Buildings[0] |= 1 }},
		{"unrevealed side", func(g *Game) { g.State.Cities[0].Side = catalog.NoSide }},
		{"inactive decider", func(g *Game) { g.State.Decider = 0 }},
		{"premature result", func(g *Game) { g.State.Result.Over = true }},
		{"discard zero", func(g *Game) { g.State.Discard[0] = 1 }},
		{"future tokens", func(g *Game) { g.State.Cities[0].Victories[0] = 1 }},
		{"record future", func(g *Game) { g.Record.Packets[1] = g.State.Deals[1] }},
		{"record public without private", func(g *Game) { g.Record.Plays[0][0][0] = Play{Kind: ActSell} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bad := g
			tt.edit(&bad)
			if bad == g {
				t.Fatal("inert negative fixture")
			}
			if err := bad.CheckInvariants(); err == nil {
				t.Fatal("invariant accepted corruption")
			}
		})
	}
	for _, phase := range scenarioPhases(t) {
		bad := phase
		bad.State.Hands[0][0] = 1
		if bad.State.Phase == PhaseTurn {
			bad.State.Hands[0][HandSize-1] = 0
		}
		if err := bad.CheckInvariants(); err == nil {
			t.Fatal("wrong phase hand count")
		}
	}
}

// Move every earlier pick to the pile so each turn's valid hand size is
// checked positively as well as by corruption. No resolver is borrowed here.
func TestScenarioEveryTurnHandSize(t *testing.T) {
	for _, n := range []uint8{3, 4, 5, 6, 7} {
		for age := uint8(1); age <= NumAges; age++ {
			for turn := uint8(1); turn <= TurnsPerAge; turn++ {
				g, err := NewGame(Setup{Seed: 8, Seats: n, Options: Options{Sides: SidesDay}})
				if err != nil {
					t.Fatal(err)
				}
				for a := uint8(1); a < age; a++ {
					for seat, hand := range g.State.Hands {
						for _, id := range hand {
							if id != 0 {
								g.State.Discard[id]++
							}
						}
						g.State.Hands[seat] = [HandSize]catalog.DesignID{}
					}
					if err := dealAge(&g, a+1); err != nil {
						t.Fatal(err)
					}
				}
				g.State.Turn = turn
				for seat := uint8(0); seat < n; seat++ {
					h := &g.State.Hands[seat]
					for k := uint8(0); k < turn-1; k++ {
						id := h[0]
						g.State.Discard[id]++
						copy(h[:], h[1:])
						h[HandSize-1] = 0
					}
				}
				alignScenarioHistory(t, &g)
				if err := g.CheckInvariants(); err != nil {
					t.Fatalf("n%d age%d turn%d: %v", n, age, turn, err)
				}
				got, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
				if err != nil || got != g {
					t.Fatal("turn scenario changed", err)
				}
			}
		}
	}
}

func TestFingerprintExplicitEncoding(t *testing.T) {
	// Randomize EVERY field, even invalid positions, so zero padding cannot hide
	// an omitted field in the manual encoding. binary.Write is an independent
	// test-only field walker; production does no reflection or memory dumping.
	var g Game
	n := uint64(0x1234)
	var fill func(reflect.Value)
	fill = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				fill(v.Field(i))
			}
		case reflect.Array:
			for i := 0; i < v.Len(); i++ {
				fill(v.Index(i))
			}
		case reflect.Bool:
			n = n*6364136223846793005 + 1
			v.SetBool(n&1 != 0)
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n = n*6364136223846793005 + 1
			v.SetUint(n)
		case reflect.Int16:
			n = n*6364136223846793005 + 1
			v.SetInt(int64(n))
		default:
			t.Fatal(v.Kind())
		}
	}
	fill(reflect.ValueOf(&g).Elem())
	var want bytes.Buffer
	if err := binary.Write(&want, binary.LittleEndian, g); err != nil {
		t.Fatal(err)
	}
	got := appendGameEncoding(nil, &g)
	if len(got) != gameEncodingSize || len(got) != binary.Size(g) || !bytes.Equal(got, want.Bytes()) {
		t.Fatal("manual field encoding differs")
	}
	if g.Fingerprint() != rng.Hex(rng.Digest(want.Bytes())) {
		t.Fatal("fingerprint differs from explicit bytes")
	}
	for _, edit := range []func(*Game){func(g *Game) { g.State.Seed++ }, func(g *Game) { g.State.Opts.HideAtEnd = !g.State.Opts.HideAtEnd }, func(g *Game) { g.Record.Picks[0][0][0]++ }, func(g *Game) { g.State.Cities[0].Produce[0]++ }} {
		other := g
		edit(&other)
		if g.Fingerprint() == other.Fingerprint() {
			t.Fatal("fingerprint omitted field")
		}
	}
	prefix := []byte{1, 2, 3}
	encoded := appendGameEncoding(prefix, &g)
	if !bytes.Equal(encoded[:3], prefix) || !bytes.Equal(encoded[3:], got) {
		t.Fatal("append overwrote prefix")
	}
}
