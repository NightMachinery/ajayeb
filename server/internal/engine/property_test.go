package engine_test

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The property sweep plays seeded games to the end and checks, after every
// action:
//
//   - CheckInvariants (card conservation, coins, the pawn, wonders, tokens,
//     caches recomputed from scratch, phase data, the structure);
//   - there is a legal action unless the game is over, IsLegal accepts
//     every listed action, and every listed action applies;
//   - random illegal actions (malformed ones and near misses) fail with an
//     *IllegalError, are rejected by IsLegal, and leave the state and its
//     hash unchanged;
//   - the game ends within maxPlies actions;
//   - replaying the recorded log gives the same hash at every ply.
//
// RandomPolicy games: 500, or AJAYEB_SWEEP if larger (100 with -short).
// SimplePolicy games: AJAYEB_SWEEP, default 300 (50 with -short).

const maxPlies = 200

// sweepSize returns the game counts of the sweep.
func sweepSize(t *testing.T) (random, simple int) {
	random, simple = 500, 300
	if v := os.Getenv("AJAYEB_SWEEP"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			t.Fatalf("AJAYEB_SWEEP=%q is not a count", v)
		}
		random, simple = max(random, n), n
	}
	if testing.Short() {
		random, simple = min(random, 100), min(simple, 50)
	}
	return random, simple
}

// sweepOptions varies the options with the seed.
func sweepOptions(seed uint64) engine.Options {
	o := engine.Options{FirstPlayer: engine.FirstPlayerRule(seed % 3)}
	if seed%4 == 3 {
		o.CenterPawn = engine.LastActiveStarts
	}
	return o
}

func TestPropertyRandomGames(t *testing.T) {
	n, _ := sweepSize(t)
	var stats sweepStats
	for i := range n {
		seed := uint64(1_000 + i)
		pol := enginetest.NewRandomPolicy(seed ^ 0x5eed)
		checkGame(t, seed, sweepOptions(seed), [2]enginetest.Policy{pol, pol}, i < 50, &stats)
		if t.Failed() {
			return
		}
	}
	stats.report(t, n)
}

func TestPropertySimpleGames(t *testing.T) {
	_, n := sweepSize(t)
	var stats sweepStats
	for i := range n {
		seed := uint64(50_000 + i)
		pols := [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)}
		checkGame(t, seed, sweepOptions(seed), pols, i < 20, &stats)
		if t.Failed() {
			return
		}
	}
	stats.report(t, n)
}

// sweepStats counts what the sweep exercised, so a silent regression in the
// flow (games that never reach Age III, wonders never built, no starter
// choice by a weaker player, no science pair) shows up.
type sweepStats struct {
	plies, maxPly, ages3, chooseStarter, wonders, sevenWonders, builds, discards, reveals int

	offCentre  int    // starter choices made by the weaker player (the pawn off centre)
	boardPicks int    // board token choices opened by a science pair
	loots      int    // looting tokens taken
	early      int    // games won by a supremacy before Age III
	victories  [4]int // finished games by engine.VictoryKind
}

func (st *sweepStats) report(t *testing.T, games int) {
	t.Helper()
	t.Logf("%d games, %d actions (longest %d): %d reached Age III, %d starter choices (%d off centre), %d builds, %d discards, %d wonders, %d games with seven wonders, %d reveals, %d board token choices, %d looting tokens taken; victories military %d, science %d, civilian %d (%d before Age III)",
		games, st.plies, st.maxPly, st.ages3, st.chooseStarter, st.offCentre, st.builds, st.discards, st.wonders, st.sevenWonders, st.reveals,
		st.boardPicks, st.loots, st.victories[engine.VictoryMilitary], st.victories[engine.VictoryScience], st.victories[engine.VictoryCivilian], st.early)
	if games >= 20 && (st.ages3+st.early < games || st.builds == 0 || st.wonders == 0 || st.reveals == 0 || st.chooseStarter == 0 ||
		st.offCentre == 0 || st.boardPicks == 0 || st.loots == 0 || st.victories[engine.VictoryCivilian] == 0) {
		t.Errorf("the sweep did not exercise the flow: %+v", *st)
	}
}

// checkGame plays one game and checks every property. With deep set it also
// applies every legal action to a copy and checks the invariants there.
func checkGame(t *testing.T, seed uint64, opts engine.Options, pols [2]enginetest.Policy, deep bool, st *sweepStats) {
	t.Helper()
	s := engine.NewGame(seed, opts)
	if err := engine.CheckInvariants(&s); err != nil {
		t.Fatalf("seed %d: the new game breaks invariants: %v", seed, err)
	}
	log := engine.NewLog(seed, opts, nil)
	hashes := []uint64{engine.Hash(&s)}
	junk := engine.NewRNG(seed ^ 0x6a756e6b)
	legal := make([]engine.Action, 0, engine.MaxLegalActions)
	var ev []engine.Event
	reachedIII := false
	for !s.Over() {
		if int(s.Ply) >= maxPlies {
			t.Fatalf("seed %d: the game is not over after %d actions (phase %s, age %d)", seed, maxPlies, s.Phase, s.Age)
		}
		legal = engine.LegalActions(&s, legal[:0])
		if len(legal) == 0 {
			t.Fatalf("seed %d ply %d: no legal action in phase %s", seed, s.Ply, s.Phase)
		}
		checkLegalSet(t, seed, &s, legal, deep)
		checkIllegal(t, seed, &s, legal, &junk)

		a := pols[s.ToMove].Choose(&s, legal)
		log.Record(&s, a)
		ev = ev[:0]
		if err := engine.Apply(&s, a, &ev); err != nil {
			t.Fatalf("seed %d ply %d: legal action %s failed: %v", seed, s.Ply, a, err)
		}
		if err := engine.CheckInvariants(&s); err != nil {
			t.Fatalf("seed %d ply %d after %s: %v", seed, s.Ply, a, err)
		}
		hashes = append(hashes, engine.Hash(&s))
		st.plies++
		switch a.Type {
		case engine.ActConstructBuilding:
			st.builds++
		case engine.ActDiscardBuilding:
			st.discards++
		case engine.ActConstructWonder:
			st.wonders++
		}
		for _, e := range ev {
			switch e.Kind {
			case engine.EvReveal:
				st.reveals++
			case engine.EvLoot:
				st.loots++
			}
		}
		switch s.Phase {
		case engine.PhaseChooseStarter:
			st.chooseStarter++
			if s.Military != 0 {
				st.offCentre++
			}
		case engine.PhasePickBoardToken:
			st.boardPicks++
		}
		if s.Age == 3 {
			reachedIII = true
		}
	}
	if got := engine.LegalActions(&s, legal[:0]); len(got) != 0 {
		t.Fatalf("seed %d: %d legal actions after the game is over", seed, len(got))
	}
	st.maxPly = max(st.maxPly, int(s.Ply))
	if s.Result.Scores != engine.Scores(&s) {
		t.Fatalf("seed %d: the result's scores %+v are not the final score sheet %+v", seed, s.Result.Scores, engine.Scores(&s))
	}
	st.victories[s.Result.Kind]++
	switch {
	case reachedIII:
		st.ages3++
	case s.Result.Kind == engine.VictoryMilitary || s.Result.Kind == engine.VictoryScience:
		st.early++
	}
	if s.WondersBuilt == engine.MaxWondersBuilt {
		st.sevenWonders++
	}
	log.Seal(&s)
	checkReplay(t, seed, log, hashes)
}

// checkLegalSet checks that IsLegal accepts every listed action, that the
// list has no duplicates, and (when deep) that each listed action applies
// to a copy of the state and keeps the invariants.
func checkLegalSet(t *testing.T, seed uint64, s *engine.State, legal []engine.Action, deep bool) {
	t.Helper()
	for i, a := range legal {
		if err := engine.IsLegal(s, a); err != nil {
			t.Fatalf("seed %d ply %d: listed action %s is not legal: %v", seed, s.Ply, a, err)
		}
		for _, b := range legal[:i] {
			if a == b {
				t.Fatalf("seed %d ply %d: %s is listed twice", seed, s.Ply, a)
			}
		}
		c := s.Clone()
		if err := engine.Apply(&c, a, nil); err != nil {
			t.Fatalf("seed %d ply %d: listed action %s fails: %v", seed, s.Ply, a, err)
		}
		if deep {
			if err := engine.CheckInvariants(&c); err != nil {
				t.Fatalf("seed %d ply %d after listed action %s: %v", seed, s.Ply, a, err)
			}
		}
	}
}

// checkIllegal tries actions that are not in the legal list: random field
// values (mostly malformed) and near misses built from real ids. Each must
// be rejected without touching the state.
func checkIllegal(t *testing.T, seed uint64, s *engine.State, legal []engine.Action, r *engine.RNG) {
	t.Helper()
	isListed := func(a engine.Action) bool {
		for _, b := range legal {
			if a == b {
				return true
			}
		}
		return false
	}
	before, hash := *s, engine.Hash(s)
	try := func(a engine.Action) {
		if isListed(a) {
			return
		}
		if err := engine.IsLegal(s, a); err == nil {
			t.Fatalf("seed %d ply %d: IsLegal accepts %s, which LegalActions does not list", seed, s.Ply, a)
		}
		err := engine.Apply(s, a, nil)
		var ie *engine.IllegalError
		if err == nil || !errors.As(err, &ie) {
			t.Fatalf("seed %d ply %d: illegal action %s: got %v, want an *IllegalError", seed, s.Ply, a, err)
		}
		if *s != before || engine.Hash(s) != hash {
			t.Fatalf("seed %d ply %d: the rejected action %s changed the state", seed, s.Ply, a)
		}
	}
	for range 3 {
		try(engine.Action{
			Type:    engine.ActionType(r.IntN(10)),
			Card:    catalog.CardID(r.IntN(80)),
			Wonder:  catalog.WonderID(r.IntN(14)),
			Token:   catalog.TokenID(r.IntN(12)),
			Starter: engine.Seat(r.IntN(3)),
		})
	}
	// Near misses: well-formed actions of every type with real ids.
	c := catalog.CardID(1 + r.IntN(catalog.NumCards))
	w := catalog.WonderID(1 + r.IntN(catalog.NumWonders))
	tk := catalog.TokenID(1 + r.IntN(catalog.NumTokens))
	for _, a := range []engine.Action{
		engine.DraftAction(w), engine.ConstructAction(c), engine.DiscardAction(c), engine.WonderAction(w, c),
		engine.DestroyAction(c), engine.ReviveAction(c), engine.TokenAction(tk), engine.StarterAction(engine.Seat(r.IntN(2))),
	} {
		try(a)
	}
	// A listed action with one field changed.
	a := legal[r.IntN(len(legal))]
	switch r.IntN(3) {
	case 0:
		a.Card = catalog.CardID(1 + r.IntN(catalog.NumCards))
	case 1:
		a.Wonder = catalog.WonderID(1 + r.IntN(catalog.NumWonders))
	default:
		a.Token = catalog.TokenID(1 + r.IntN(catalog.NumTokens))
	}
	try(a)
}

// checkReplay replays the log (also after a JSON round trip) and compares
// the hash after every action with the one recorded while playing.
func checkReplay(t *testing.T, seed uint64, log *engine.Log, hashes []uint64) {
	t.Helper()
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("seed %d: marshal the log: %v", seed, err)
	}
	var back engine.Log
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("seed %d: unmarshal the log: %v", seed, err)
	}
	for _, l := range []*engine.Log{log, &back} {
		final, err := engine.ReplaySteps(l, func(i int, s *engine.State, _ []engine.Event) error {
			if h := engine.Hash(s); h != hashes[i+1] {
				t.Fatalf("seed %d: replay hash after action %d is %016x, the game had %016x", seed, i, h, hashes[i+1])
			}
			return nil
		})
		if err != nil {
			t.Fatalf("seed %d: replay: %v", seed, err)
		}
		if engine.Hash(&final) != hashes[len(hashes)-1] {
			t.Fatalf("seed %d: the replay ends in another state", seed)
		}
	}
}

// TestLegalActionsAllocs checks that LegalActions never allocates with a
// buffer of MaxLegalActions, in states sampled from a game.
func TestLegalActionsAllocs(t *testing.T) {
	s := engine.NewGame(7, engine.Options{})
	pol := enginetest.NewRandomPolicy(7)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	checked := 0
	for !s.Over() {
		st := s
		allocs := testing.AllocsPerRun(20, func() { buf = engine.LegalActions(&st, buf[:0]) })
		if allocs != 0 {
			t.Fatalf("ply %d (phase %s): LegalActions allocates %.1f times per call", s.Ply, s.Phase, allocs)
		}
		checked++
		legal := engine.LegalActions(&s, buf[:0])
		engine.ApplyTrusted(&s, pol.Choose(&s, legal))
	}
	if checked < 50 {
		t.Fatalf("checked only %d states", checked)
	}
}

// TestApplyTrustedMatchesApply plays the same games with Apply and with
// ApplyTrusted and expects identical states at every step.
func TestApplyTrustedMatchesApply(t *testing.T) {
	for seed := uint64(1); seed <= 30; seed++ {
		a := engine.NewGame(seed, sweepOptions(seed))
		b := a
		pol := enginetest.NewRandomPolicy(seed)
		buf := make([]engine.Action, 0, engine.MaxLegalActions)
		for !a.Over() {
			act := pol.Choose(&a, engine.LegalActions(&a, buf[:0]))
			if err := engine.Apply(&a, act, nil); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			engine.ApplyTrusted(&b, act)
			if a != b {
				t.Fatalf("seed %d ply %d: ApplyTrusted(%s) differs from Apply", seed, a.Ply, act)
			}
		}
	}
}

// TestEventsDoNotChangeTheState checks that asking for events never changes
// what Apply does.
func TestEventsDoNotChangeTheState(t *testing.T) {
	a := engine.NewGame(99, engine.Options{})
	b := a
	pol := enginetest.NewRandomPolicy(99)
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	var ev []engine.Event
	for !a.Over() {
		act := pol.Choose(&a, engine.LegalActions(&a, buf[:0]))
		ev = ev[:0]
		if err := engine.Apply(&a, act, &ev); err != nil {
			t.Fatal(err)
		}
		if err := engine.Apply(&b, act, nil); err != nil {
			t.Fatal(err)
		}
		if a != b {
			t.Fatalf("ply %d: Apply with events differs from Apply without", a.Ply)
		}
		if len(ev) == 0 {
			t.Fatalf("ply %d: %s reported no event", a.Ply, act)
		}
	}
}

func BenchmarkRandomPlayout(b *testing.B) {
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	pol := enginetest.NewRandomPolicy(1)
	seed := uint64(0)
	for b.Loop() {
		seed++
		s := engine.NewGame(seed, engine.Options{})
		for !s.Over() {
			legal := engine.LegalActions(&s, buf[:0])
			engine.ApplyTrusted(&s, pol.Choose(&s, legal))
		}
	}
}

func BenchmarkLegalActions(b *testing.B) {
	s := enginetest.New(b, 3).Age(2).State()
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for b.Loop() {
		buf = engine.LegalActions(&s, buf[:0])
	}
}
