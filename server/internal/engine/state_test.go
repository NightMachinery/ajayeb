package engine

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

// States, actions and events are comparable values. This line fails to
// compile if any of them gains a slice, map or function field.
var _ = func(a, b State, c, d Action, e, f Event) bool { return a == b && c == d && e == f }

// Keep the checked roots and size limits in Duel, not the shared walker.
func fixedShapeViolations(t reflect.Type, path string) []string {
	return puritycheck.FixedShapeViolations(t, path)
}

func TestStateIsFixedSize(t *testing.T) {
	n := binary.Size(State{})
	if n <= 0 {
		t.Fatalf("binary.Size(State{}) = %d: State holds a slice, map, string, pointer or int", n)
	}
	mem := reflect.TypeFor[State]().Size()
	t.Logf("State: %d bytes encoded, %d bytes in memory", n, mem)
	// A clone should stay one small memory copy.
	if mem > 512 {
		t.Errorf("State is %d bytes in memory; keep it at most 512", mem)
	}
	for _, v := range []any{Player{}, Slot{}, WonderSlot{}, Options{}, Score{}, Result{}, Action{}, Event{}} {
		if binary.Size(v) <= 0 {
			t.Errorf("binary.Size(%T{}) <= 0", v)
		}
	}
}

func TestFixedValueTypesHaveNoReferences(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[State](), reflect.TypeFor[Action](), reflect.TypeFor[Event]()} {
		if bad := fixedShapeViolations(typ, typ.Name()); len(bad) > 0 {
			t.Errorf("%s is not a fixed value type:\n%s", typ.Name(), strings.Join(bad, "\n"))
		}
		if !typ.Comparable() {
			t.Errorf("%s is not comparable", typ.Name())
		}
	}
}

// The shape check itself must catch every forbidden kind.
func TestFixedShapeCheckRejects(t *testing.T) {
	type bad struct {
		A int
		B uint
		C uintptr
		D float64
		E string
		F []uint8
		G map[uint8]uint8
		H *uint8
		I any
		J func()
		K chan uint8
		L [2]struct{ M float32 }
		N complex64
		O uint8 // allowed
	}
	got := fixedShapeViolations(reflect.TypeFor[bad](), "bad")
	if len(got) != 13 {
		t.Errorf("found %d violations, want 13:\n%s", len(got), strings.Join(got, "\n"))
	}
}

// filledState returns a state with a non-zero value in every kind of field.
func filledState() State {
	var s State
	s.Seed = 0xdeadbeef
	s.Opts = Options{FirstPlayer: FirstPlayerSeat1, CenterPawn: LastActiveStarts, HideAtEnd: true}
	s.Phase = PhasePickLibraryToken
	s.Age = 3
	s.Ply = 77
	s.ToMove, s.Active, s.FirstSeat = 1, 1, 0
	s.ExtraTurn = true
	s.Destroy = catalog.Grey
	s.Library = [LibraryDraw]catalog.TokenID{1, 4, 9}
	for i := range s.Players {
		p := &s.Players[i]
		p.City = catalog.CardSetOf(1, 9, 68)
		p.Coins = int16(10 + i)
		p.Tokens = catalog.TokenSetOf(catalog.TokenLaw)
		p.Wonders[2] = WonderSlot{ID: catalog.WonderPyramids, Built: true, Tucked: 23}
		p.Chains = catalog.ChainHorseshoe.Bit()
		p.Prod[catalog.Stone] = 2
		p.Choice[0], p.NChoice = catalog.RawGoods, 1
		p.TradeFix = catalog.Manufactured
		p.ColorN[catalog.Red] = 1
		p.Sci[catalog.Scales] = 1
	}
	s.Slots[5] = Slot{Card: 40, FaceUp: true, Back: catalog.BackAge2}
	s.Deals[2][19] = 66
	s.Removed[1][2] = 30
	s.BoxGuilds[3] = 73
	s.Discard = catalog.CardSetOf(12)
	s.OutOfPlay = catalog.CardSetOf(13)
	s.DraftOrder[7] = catalog.WonderSphinx
	s.DraftTaken = 0b1011
	s.BoxWonders[0] = catalog.WonderColossus
	s.BoardTokens = catalog.TokenSetOf(catalog.TokenEconomy)
	s.BoxTokens[4] = catalog.TokenUrbanism
	s.Military = -4
	s.Looting = LootMinus6 | LootPlus3
	s.WondersBuilt = 6
	s.Result = Result{Kind: VictoryScience, Winner: 1, Scores: [2]Score{{Total: 3}, {Blue: 4}}}
	return s
}

func TestCloneIsIndependent(t *testing.T) {
	s := filledState()
	orig := s
	c := s.Clone()
	if c != s {
		t.Fatal("a clone differs from its source")
	}
	c.Players[0].City.Add(50)
	c.Players[1].Wonders[2].Built = false
	c.Players[0].Prod[catalog.Stone] = 9
	c.Slots[5].Card = 0
	c.Deals[2][19] = 0
	c.Removed[1][2] = 0
	c.Library[0] = 0
	c.DraftOrder[7] = 0
	c.BoxTokens[4] = 0
	c.Result.Scores[1].Blue = 0
	c.Discard.Add(14)
	if s != orig {
		t.Fatal("mutating a clone changed the source")
	}
	if c == s {
		t.Fatal("the clone did not change")
	}
	if s.Over() {
		t.Fatal("Over is true before the game ends")
	}
	s.Phase = PhaseGameOver
	if !s.Over() {
		t.Fatal("Over is false in PhaseGameOver")
	}
}

func TestSeat(t *testing.T) {
	if Seat(0).Opp() != 1 || Seat(1).Opp() != 0 {
		t.Error("Opp is wrong")
	}
	if LootAll != 0b1111 || LootMinus3|LootMinus6|LootPlus3|LootPlus6 != LootAll {
		t.Error("looting bits are wrong")
	}
}

type textEnum interface {
	String() string
	MarshalText() ([]byte, error)
}

// checkTextEnum round-trips every value below n through MarshalText and
// UnmarshalText, and checks the invalid value n and a bad name.
func checkTextEnum[T interface {
	~uint8
	textEnum
}, P interface {
	*T
	UnmarshalText([]byte) error
}](t *testing.T, n uint8, wantNames []string) {
	t.Helper()
	if int(n) != len(wantNames) {
		t.Fatalf("%T has %d values, the test lists %d names", T(0), n, len(wantNames))
	}
	for i := range n {
		v := T(i)
		b, err := v.MarshalText()
		if err != nil || string(b) != wantNames[i] || v.String() != wantNames[i] {
			t.Errorf("%T(%d) = %q, %v; String %q; want %q", v, i, b, err, v.String(), wantNames[i])
			continue
		}
		var back T
		if err := P(&back).UnmarshalText(b); err != nil || back != v {
			t.Errorf("%T: UnmarshalText(%q) = %d, %v", v, b, back, err)
		}
	}
	if _, err := T(n).MarshalText(); err == nil {
		t.Errorf("%T(%d).MarshalText succeeded", T(0), n)
	}
	if !strings.Contains(T(n).String(), "#") {
		t.Errorf("%T(%d).String() = %q", T(0), n, T(n).String())
	}
	keep := T(1)
	for _, bad := range []string{"", "nope", strings.ToUpper(wantNames[1]), " " + wantNames[1], wantNames[1] + " "} {
		if err := P(&keep).UnmarshalText([]byte(bad)); err == nil {
			t.Errorf("%T: UnmarshalText(%q) succeeded", T(0), bad)
		}
	}
	if keep != T(1) {
		t.Errorf("%T: a failed UnmarshalText changed the value", T(0))
	}
}

func TestTextEnums(t *testing.T) {
	checkTextEnum[Phase](t, uint8(numPhases), []string{"wonderDraft", "play", "chooseStarter", "destroyCard",
		"buildFromDiscard", "pickBoardToken", "pickLibraryToken", "gameOver"})
	checkTextEnum[FirstPlayerRule](t, uint8(numFirstPlayerRules), []string{"random", "seat0", "seat1"})
	checkTextEnum[CenterPawnRule](t, uint8(numCenterPawnRules), []string{"lastActiveChooses", "lastActiveStarts"})
	checkTextEnum[VictoryKind](t, uint8(numVictoryKinds), []string{"none", "military", "science", "civilian", "forfeit"})
	checkTextEnum[Reason](t, uint8(numReasons), []string{"none", "gameOver", "wrongPhase", "malformed", "notInDraft",
		"notAccessible", "wonderUnavailable", "unaffordable", "notTarget", "notInDiscard", "tokenUnavailable"})
	checkTextEnum[EventKind](t, uint8(numEventKinds), []string{"none", "draft", "draftRound", "ageStart", "build",
		"discard", "wonder", "pay", "reveal", "coins", "pawn", "loot", "pair", "token", "libraryDraw", "destroy",
		"rebuild", "wonderRemoved", "effectSkipped", "extraTurn", "extraTurnLost", "starterChosen", "gameOver",
		"resign"})
}

func TestOptionsJSON(t *testing.T) {
	b, err := json.Marshal(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"firstPlayer":"random","centerPawn":"lastActiveChooses","hideAtEnd":false}`; string(b) != want {
		t.Errorf("zero Options = %s, want %s", b, want)
	}
	in := Options{FirstPlayer: FirstPlayerSeat1, CenterPawn: LastActiveStarts, HideAtEnd: true}
	b, err = json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Options
	if err := json.Unmarshal(b, &out); err != nil || out != in {
		t.Errorf("round trip of %s = %+v, %v", b, out, err)
	}
	if err := json.Unmarshal([]byte(`{"firstPlayer":"seat2"}`), &out); err == nil {
		t.Error("a bad first-player rule was accepted")
	}

	res := Result{Kind: VictoryCivilian, Winner: SharedVictory, Scores: [2]Score{{Blue: 3, Total: 3}, {Total: 3}}}
	b, err = json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var back Result
	if err := json.Unmarshal(b, &back); err != nil || back != res {
		t.Errorf("Result round trip of %s = %+v, %v", b, back, err)
	}
	if !strings.HasPrefix(string(b), `{"kind":"civilian","winner":-1,"scores":[{"blue":3,`) {
		t.Errorf("Result JSON = %s", b)
	}
}

func TestEmit(t *testing.T) {
	emit(nil, Event{Kind: EvBuild}) // must not panic
	var ev []Event
	emit(&ev, Event{Kind: EvBuild, Card: 9})
	emit(&ev, Event{Kind: EvLibraryDraw, Seat: 1, Token: catalog.TokenLaw, Private: true})
	if len(ev) != 2 || ev[0].Card != 9 || !ev[1].Private {
		t.Errorf("events = %+v", ev)
	}
}

func TestHookEnumNames(t *testing.T) {
	names := []string{viaPurchase.String(), viaChain.String(), viaDiscard.String(),
		outcomeFinish.String(), outcomeChoice.String(), outcomeOver.String()}
	want := []string{"purchase", "chain", "discard", "finish", "choice", "over"}
	for i := range names {
		if names[i] != want[i] {
			t.Errorf("name %d = %q, want %q", i, names[i], want[i])
		}
	}
	if buildVia(9).String() != "via#?" || outcome(9).String() != "outcome#?" {
		t.Error("invalid values are not marked")
	}
}
