package engine

import (
	"encoding/json"
	"fmt"
	"testing"
)

// The leak guard of the hints (hints.go, "Public information only"). A
// hint for viewer v may depend only on what v may see, so it must come out
// the same on the real state and on every state Determinize can draw from
// v's observation. The guard checks that on whole views: the view of v
// built with every hint family on, on the real state and on N samples of
// Determinize(Observe(s, v)), must be identical (the end-of-game reveal
// aside). The decider's own Great Library offer is kept, since Observe
// keeps it for the seat choosing from it; a spectator is checked against
// the observation of the seat not deciding, which sees no offer. One
// equality catches a read of a face-down slot, the deals, the removed
// cards, the box guilds, the box order or a hidden offer by any hint,
// present or future, and by the view itself.
//
// The other two checks watch the simulations: shared hints never simulate
// for a seat that is not deciding while the Great Library offer is open,
// and every follow-up of a simulated action (WinsNow) stays inside the
// face-up mask, with nothing revealed. A probe that runs WinsNow for each
// seat on a normalised clone, as the threat hints will, must also agree
// across determinizations.
//
// FuzzHintsLeak runs the same checks on fuzzer-chosen games; the nightly CI
// job fuzzes every Fuzz target of the module, this one included. Helpers
// are prefixed hl.

// hlSamples is the number of determinizations each state is checked on.
const hlSamples = 3

// hlViewers are the three viewers of a room.
var hlViewers = [3]Viewer{0, 1, Spectator}

// hlCorpus returns states to check: every second state of random games,
// and every named scenario with its trigger and a few random moves after
// it, which reach the rarer phases (the Great Library offer, the destroy
// and rebuild choices, the board token choice, the starter choice).
func hlCorpus(tb testing.TB, games int) []State {
	tb.Helper()
	var out []State
	buf := make([]Action, 0, MaxLegalActions)
	play := func(s State, r *RNG, plies int) {
		for n := 0; n < plies && !s.Over(); n++ {
			if n%2 == 0 {
				out = append(out, s)
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
	}
	for g := range games {
		seed := uint64(7000 + 31*g)
		r := NewRNG(seed)
		play(NewGame(seed, Options{FirstPlayer: FirstPlayerRule(g % 3)}), &r, 400)
	}
	for i, name := range NamedScenarioNames() {
		sc := NamedScenario(name)
		s, err := NewGameFromScenario(uint64(100+i), Options{}, sc)
		if err != nil {
			tb.Fatalf("%s: %v", name, err)
		}
		out = append(out, s)
		for _, k := range sc.Trigger {
			a, err := ParseKey(k)
			if err != nil {
				tb.Fatal(err)
			}
			if err := Apply(&s, a, nil); err != nil {
				tb.Fatalf("%s: %s: %v", name, k, err)
			}
			out = append(out, s)
		}
		r := NewRNG(uint64(i))
		play(s, &r, 12)
	}
	return out
}

// hlView is viewer v's view of s with every hint family on, as JSON,
// without the end-of-game reveal (which shows the hidden cards by design).
func hlView(s *State, v Viewer) string {
	h := ComputeHints(s, hnAllHints)
	vw := ViewWithHints(s, v, &h)
	vw.Revealed = nil
	b, err := json.Marshal(vw)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// hlObserver is the seat whose observation viewer v's view must agree
// with: v's own seat, or for a spectator the seat that is not deciding,
// which never sees a Great Library offer.
func hlObserver(s *State, v Viewer) Seat {
	if seat, ok := v.Seat(); ok {
		return seat
	}
	return s.ToMove.Opp()
}

// hlThreatProbe runs WinsNow for seat on a normalised clone of s, for each
// of seat's actions there, limited to the face-up mask of s, and lists
// every action it simulated, at every depth, and the wins. Both must be
// public: a simulated action on a card the viewer cannot see would differ
// across determinizations even when it wins nothing. It follows the
// phase gate of the threat hints: only in PhasePlay and
// PhaseChooseStarter. A non-nil c also sees every simulation.
func hlThreatProbe(s *State, seat Seat, c *simCtx) string {
	if s.Phase != PhasePlay && s.Phase != PhaseChooseStarter {
		return ""
	}
	mask := FaceUpSlots(s)
	x := normalised(s, seat)
	out := ""
	probe := &simCtx{trace: func(n simNote) {
		out += fmt.Sprintf("%d:%s ", n.Depth, n.Action)
		if c != nil {
			c.apply(&n.State, n.Action, n.Depth, n.Mask)
		}
	}}
	var buf [MaxLegalActions]Action
	for _, a := range LegalActions(&x, buf[:0]) {
		if w, ok := winsNow(&x, a, mask, probe); ok {
			out += fmt.Sprintf("%s=%s/%d/%s ", a, w.Kind, w.Steps, w.Via)
		}
	}
	return out
}

// hlCheck runs the view equality and the threat probe on s for every
// viewer, with samples determinizations each.
func hlCheck(tb testing.TB, s *State, r *RNG, samples int) {
	tb.Helper()
	if s.Over() {
		return
	}
	var probe [2]string
	for seat := range Seat(2) {
		probe[seat] = hlThreatProbe(s, seat, nil)
	}
	for _, v := range hlViewers {
		want := hlView(s, v)
		o := Observe(s, hlObserver(s, v))
		for range samples {
			d := o.Determinize(r)
			if got := hlView(&d, v); got != want {
				tb.Fatalf("ply %d (%s), viewer %d: the view differs on a determinization of the viewer's observation:\nreal: %s\ndet:  %s",
					s.Ply, s.Phase, v, want, got)
			}
			if v == Spectator {
				for seat := range Seat(2) {
					if got := hlThreatProbe(&d, seat, nil); got != probe[seat] {
						tb.Fatalf("ply %d (%s): seat %d's wins on a normalised clone differ on a determinization: %q, real %q",
							s.Ply, s.Phase, seat, got, probe[seat])
					}
				}
			}
		}
	}
}

// TestHintsLeakGuard is the leak guard over the corpus.
func TestHintsLeakGuard(t *testing.T) {
	games := 24
	if testing.Short() {
		games = 6
	}
	r := NewRNG(1)
	states := hlCorpus(t, games)
	for i := range states {
		hlCheck(t, &states[i], &r, hlSamples)
	}
	t.Logf("%d states, %d determinizations per viewer", len(states), hlSamples)
}

// TestHintsSimulationsStayPublic watches every simulation through a traced
// context: no shared hint simulates for a seat that is not deciding while
// the Great Library offer is open, and each follow-up stays in the
// face-up mask of the state the hint was asked about, with nothing
// revealed and no offer drawn.
func TestHintsSimulationsStayPublic(t *testing.T) {
	games := 12
	if testing.Short() {
		games = 4
	}
	var notes []simNote
	ctx := &simCtx{trace: func(n simNote) { notes = append(notes, n) }}
	checked := 0
	for _, s := range hlCorpus(t, games) {
		if s.Over() {
			continue
		}
		notes = notes[:0]
		computeHints(&s, hnAllHints, ctx)
		for _, n := range notes {
			if s.Phase == PhasePickLibraryToken && n.For != s.ToMove {
				t.Fatalf("ply %d: a shared hint simulated %s for seat %d during the Great Library offer", s.Ply, n.Action, n.For)
			}
		}
		hlCheckNotes(t, &s, notes)

		mask := FaceUpSlots(&s)
		var buf [MaxLegalActions]Action
		for _, a := range LegalActions(&s, buf[:0]) {
			notes = notes[:0]
			winsNow(&s, a, mask, ctx)
			hlCheckNotes(t, &s, notes)
		}
		for seat := range Seat(2) {
			notes = notes[:0]
			hlThreatProbe(&s, seat, ctx)
			hlCheckNotes(t, &s, notes)
			checked += len(notes)
		}
	}
	if checked == 0 {
		t.Fatal("the probe simulated nothing")
	}
	t.Logf("%d probe simulations checked", checked)
}

// hlCheckNotes checks the traced simulations of one hint asked about s.
func hlCheckNotes(t *testing.T, s *State, notes []simNote) {
	t.Helper()
	faceUp := FaceUpSlots(s)
	for _, n := range notes {
		if n.Mask&^faceUp != 0 {
			t.Fatalf("ply %d: a simulation's mask %#x reaches beyond the face-up slots %#x", s.Ply, n.Mask, faceUp)
		}
		switch n.Action.Type {
		case ActConstructBuilding, ActDiscardBuilding, ActConstructWonder:
			if i := SlotOf(&n.State, n.Action.Card); i < 0 || faceUp&(1<<i) == 0 || !s.Slots[i].FaceUp {
				t.Fatalf("ply %d: %s (depth %d) uses slot %d, which was not face up", s.Ply, n.Action, n.Depth, i)
			}
		}
		if n.Depth == 0 {
			continue
		}
		if n.State.Phase == PhasePickLibraryToken {
			t.Fatalf("ply %d: a follow-up %s runs in a Great Library offer", s.Ply, n.Action)
		}
		if n.State.Age != s.Age || faceDownCount(&n.State) != faceDownCount(s) {
			t.Fatalf("ply %d: a follow-up %s runs after a reveal or a new age", s.Ply, n.Action)
		}
	}
}

// FuzzHintsLeak plays seed's game, picking legal[choices[i] % len(legal)]
// at each step, and runs the leak guard on the state it reaches and on
// every eighth state on the way, with determinizations drawn from sample.
func FuzzHintsLeak(f *testing.F) {
	f.Add(uint64(1), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, uint64(1))
	f.Add(uint64(2), []byte{255, 3, 0, 128, 7, 3, 9, 200, 17, 4, 4, 4, 1, 0, 2}, uint64(9))
	f.Add(uint64(3), make([]byte, 60), uint64(3))
	f.Fuzz(func(t *testing.T, seed uint64, choices []byte, sample uint64) {
		s := NewGame(seed, Options{})
		r := NewRNG(sample)
		buf := make([]Action, 0, MaxLegalActions)
		n := min(len(choices), 2*MaxGameActions)
		for i := 0; i < n && !s.Over(); i++ {
			if i%8 == 7 {
				hlCheck(t, &s, &r, 1)
			}
			legal := LegalActions(&s, buf[:0])
			ApplyTrusted(&s, legal[int(choices[i])%len(legal)])
		}
		hlCheck(t, &s, &r, 2)
	})
}
