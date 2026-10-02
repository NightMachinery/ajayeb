package engine_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/bits"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The redaction sweep plays seeded games to the end and, before the first
// action and after every action, builds the view of seat 0, seat 1 and a
// spectator and filters the action's events for each of them. It marshals
// both to JSON and checks:
//
//   - no hidden card, wonder or token appears as any JSON string, or as any
//     ':'-separated part of one (so an action key cannot smuggle a slug).
//     Hidden means: the cards in face-down slots, the deals not laid out
//     yet, the removed cards, the box guilds, the box wonders, the second
//     draft round during the first, the box tokens, the scenario-only
//     out-of-play cards, and the Great Library offer for everyone but the
//     seat deciding it. Matching is exact: slugs are unique across cards,
//     wonders and tokens, so "great-library" never matches "library";
//   - the seed never appears, in decimal or in hex (the seeds are large, so
//     a chance match of an unrelated number is impossible);
//   - choices and disabled are empty except in the view of the seat to move,
//     which has at least one choice until the game is over;
//   - the deciding seat sees the whole Great Library offer, the others only
//     its size;
//   - at game over, the revealed section lists exactly the hidden items, and
//     the rest of the view still shows none of them; with HideAtEnd there is
//     no revealed section;
//   - no slice anywhere in the view or the events is nil.
//
// Every phase must occur in the sweep. Helpers are prefixed redact.

// redactSeed returns the seed of sweep game i: a large, distinct number.
func redactSeed(i int) uint64 {
	return 0x9e3779b97f4a7c15 ^ (uint64(i)+1)*0xbf58476d1ce4e5b9
}

func TestRedactionSweep(t *testing.T) {
	games := 200
	if testing.Short() {
		games = 40
	}
	var st redactStats
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for i := range games {
		seed := redactSeed(i)
		opts := engine.Options{FirstPlayer: engine.FirstPlayerRule(i % 3), HideAtEnd: i%5 == 4}
		if i%4 == 3 {
			opts.CenterPawn = engine.LastActiveStarts
		}
		var pols [2]enginetest.Policy
		if i%2 == 0 {
			p := enginetest.NewRandomPolicy(seed ^ 0x5eed)
			pols = [2]enginetest.Policy{p, p}
		} else {
			pols = [2]enginetest.Policy{enginetest.NewSimplePolicy(seed), enginetest.NewSimplePolicy(seed + 1)}
		}
		s := engine.NewGame(seed, opts)
		redactCheck(t, &s, nil, seed, &st)
		for n := 0; !s.Over(); n++ {
			if n > 300 {
				t.Fatalf("game %d: not over after %d actions", i, n)
			}
			a := pols[s.ToMove].Choose(&s, engine.LegalActions(&s, buf[:0]))
			var ev []engine.Event
			if err := engine.Apply(&s, a, &ev); err != nil {
				t.Fatalf("game %d: %s: %v", i, a, err)
			}
			redactCheck(t, &s, ev, seed, &st)
			if t.Failed() {
				t.Fatalf("game %d (seed %d), after %s", i, seed, a)
			}
		}
	}
	t.Logf("%d games, %d states, %d views; phases %v; %d library offers; %d game-over views revealed, %d hidden",
		games, st.states, st.views, st.phases, st.library, st.revealed, st.hiddenEnd)
	for p, n := range st.phases {
		if n == 0 {
			t.Errorf("the sweep never reached phase %d (%s)", p, engine.Phase(p))
		}
	}
	if st.library == 0 || st.revealed == 0 || st.hiddenEnd == 0 {
		t.Errorf("the sweep missed a case: %+v", st)
	}
}

// TestRedactionScenarios runs the same checks on the position of every
// protocol fixture, which reaches each phase on purpose (including both
// sides of a Great Library offer and a game over that hides its secrets).
func TestRedactionScenarios(t *testing.T) {
	var st redactStats
	for _, fx := range protoFixtures(t) {
		s := fx.state
		redactCheck(t, &s, fx.events, s.Seed, &st)
		if t.Failed() {
			t.Fatalf("fixture %s", fx.name)
		}
	}
	for p, n := range st.phases {
		if n == 0 {
			t.Errorf("no fixture reaches phase %s", engine.Phase(p))
		}
	}
}

// redactStats counts what the sweep checked.
type redactStats struct {
	states, views, library, revealed, hiddenEnd int
	phases                                      [8]int
}

// redactViewers are the three viewers every state is checked for.
var redactViewers = []engine.Viewer{0, 1, engine.Spectator}

// redactCheck checks the views of s and the events ev (of the action that
// led to s) for every viewer.
func redactCheck(t *testing.T, s *engine.State, ev []engine.Event, seed uint64, st *redactStats) {
	t.Helper()
	st.states++
	if int(s.Phase) < len(st.phases) {
		st.phases[s.Phase]++
	}
	seedDec := []byte(strconv.FormatUint(seed, 10))
	seedHex := []byte(fmt.Sprintf("%016x", seed))
	for _, v := range redactViewers {
		st.views++
		where := fmt.Sprintf("ply %d, phase %s, viewer %d", s.Ply, s.Phase, v)
		vw := engine.ViewFor(s, v)
		data := redactJSON(t, vw)
		if bytes.Contains(data, seedDec) || bytes.Contains(data, seedHex) {
			t.Fatalf("%s: the view contains the seed %d", where, seed)
		}
		if nils := redactNilSlices(vw); len(nils) > 0 {
			t.Fatalf("%s: nil slices in the view: %v", where, nils)
		}
		hidden := redactHidden(s, v)

		seat, seated := v.Seat()
		decider := seated && !s.Over() && seat == s.ToMove
		if !decider && (len(vw.Choices) > 0 || len(vw.Disabled) > 0) {
			t.Fatalf("%s: %d choices and %d disabled for a viewer who is not to move", where, len(vw.Choices), len(vw.Disabled))
		}
		if decider && len(vw.Choices) == 0 {
			t.Fatalf("%s: the seat to move has no choices", where)
		}

		if s.Phase == engine.PhasePickLibraryToken {
			redactCheckLibrary(t, s, &vw, decider, where)
			st.library++
		}

		scan := data
		if s.Over() {
			if s.Opts.HideAtEnd {
				if vw.Revealed != nil {
					t.Fatalf("%s: HideAtEnd, but the view reveals %+v", where, *vw.Revealed)
				}
				st.hiddenEnd++
			} else {
				if vw.Revealed == nil {
					t.Fatalf("%s: the game is over, but nothing is revealed", where)
				}
				redactCheckRevealed(t, &vw, hidden, where)
				rest := vw
				rest.Revealed = nil
				scan = redactJSON(t, rest)
				st.revealed++
			}
		} else if vw.Revealed != nil || vw.Result != nil {
			t.Fatalf("%s: a result or revealed section before the game is over", where)
		}
		if leak := redactLeaks(scan, hidden); leak != "" {
			t.Fatalf("%s: the view leaks %s\n%s", where, leak, data)
		}

		fe := engine.FilterEvents(ev, v)
		evData := redactJSON(t, fe)
		if bytes.Contains(evData, seedDec) || bytes.Contains(evData, seedHex) {
			t.Fatalf("%s: the events contain the seed", where)
		}
		if nils := redactNilSlices(fe); len(nils) > 0 {
			t.Fatalf("%s: nil slices in the events: %v", where, nils)
		}
		if leak := redactLeaks(evData, hidden); leak != "" {
			t.Fatalf("%s: the events leak %s\n%s", where, leak, evData)
		}
	}
}

// redactCheckLibrary checks the Great Library offer in one view.
func redactCheckLibrary(t *testing.T, s *engine.State, vw *view.View, decider bool, where string) {
	t.Helper()
	var offer []string
	for _, tk := range s.Library {
		if tk != 0 {
			offer = append(offer, tk.String())
		}
	}
	p := vw.Pending
	if p == nil || p.Kind != view.PendingKindPickLibraryToken {
		t.Fatalf("%s: pending %+v, want the library pick", where, p)
	}
	if decider {
		if !slices.Equal(p.Tokens, offer) || p.HiddenTokens != 0 {
			t.Fatalf("%s: the decider sees tokens %v (hidden %d), want %v", where, p.Tokens, p.HiddenTokens, offer)
		}
		var keys []string
		for _, c := range vw.Choices {
			keys = append(keys, c.Key)
		}
		for _, tk := range offer {
			if !slices.Contains(keys, "token:"+tk) {
				t.Fatalf("%s: the decider has no choice for %s: %v", where, tk, keys)
			}
		}
	} else if len(p.Tokens) != 0 || p.HiddenTokens != len(offer) {
		t.Fatalf("%s: another viewer sees tokens %v (hidden %d), want none of the %d", where, p.Tokens, p.HiddenTokens, len(offer))
	}
}

// redactCheckRevealed checks that a game-over view reveals exactly the
// hidden items.
func redactCheckRevealed(t *testing.T, vw *view.View, hidden map[string]string, where string) {
	t.Helper()
	got := map[string]bool{}
	redactEachString(redactJSON(t, vw.Revealed), func(s string) {
		if _, ok := hidden[s]; ok {
			got[s] = true
		}
	})
	want := 0
	for slug, what := range hidden {
		if what == redactOutOfPlay {
			// Scenario scaffolding, not part of a game: never shown.
			if got[slug] {
				t.Fatalf("%s: the out-of-play %s is revealed", where, slug)
			}
			continue
		}
		want++
		if !got[slug] {
			t.Fatalf("%s: %s (%s) is not revealed: %+v", where, slug, what, *vw.Revealed)
		}
	}
	r := vw.Revealed
	n := len(r.FaceDown) + len(r.BoxGuilds) + len(r.BoxWonders) + len(r.BoxTokens)
	for _, ac := range r.Removed {
		n += len(ac.Cards)
	}
	for _, ac := range r.Undealt {
		n += len(ac.Cards)
	}
	if n != want {
		t.Fatalf("%s: %d items revealed, want the %d hidden ones: %+v", where, n, want, *r)
	}
}

// redactOutOfPlay marks, in redactHidden's result, the cards a scenario put
// out of play: hidden like the rest, but never revealed either.
const redactOutOfPlay = "out of play"

// redactHidden returns the slugs that viewer v may not see in s, each with
// where it is.
func redactHidden(s *engine.State, v engine.Viewer) map[string]string {
	h := map[string]string{}
	for i, sl := range s.Slots {
		if sl.Card != 0 && !sl.FaceUp {
			h[sl.Card.String()] = "face-down slot " + strconv.Itoa(i)
		}
	}
	for a := range s.Deals {
		for _, c := range s.Deals[a] {
			if c != 0 {
				h[c.String()] = "the Age " + strconv.Itoa(a+1) + " deal"
			}
		}
		for _, c := range s.Removed[a] {
			if c != 0 {
				h[c.String()] = "removed from Age " + strconv.Itoa(a+1)
			}
		}
	}
	for _, c := range s.BoxGuilds {
		if c != 0 {
			h[c.String()] = "the box guilds"
		}
	}
	for c := range s.OutOfPlay.All() {
		h[c.String()] = redactOutOfPlay
	}
	for _, w := range s.BoxWonders {
		if w != 0 {
			h[w.String()] = "the box wonders"
		}
	}
	if bits.OnesCount8(s.DraftTaken) < 4 {
		for _, w := range s.DraftOrder[4:] {
			h[w.String()] = "draft round 2"
		}
	}
	for _, tk := range s.BoxTokens {
		if tk != 0 {
			h[tk.String()] = "the box tokens"
		}
	}
	seat, seated := v.Seat()
	if !seated || seat != s.ToMove || s.Phase != engine.PhasePickLibraryToken {
		for _, tk := range s.Library {
			if tk != 0 {
				h[tk.String()] = "the Great Library offer of another seat"
			}
		}
	}
	return h
}

// redactLeaks returns a description of the first hidden slug found in the
// JSON data, as a whole string or as a ':'-separated part of one (object
// keys included), or "" when there is none.
func redactLeaks(data []byte, hidden map[string]string) string {
	leak := ""
	redactEachString(data, func(s string) {
		if leak != "" {
			return
		}
		for part := range splitColon(s) {
			if what, ok := hidden[part]; ok {
				leak = fmt.Sprintf("%q (in %q, from %s)", part, s, what)
				return
			}
		}
	})
	return leak
}

// splitColon yields s and, if it has colons, each part between them.
func splitColon(s string) func(func(string) bool) {
	return func(yield func(string) bool) {
		if !yield(s) {
			return
		}
		for i := 0; i < len(s); {
			j := i
			for j < len(s) && s[j] != ':' {
				j++
			}
			if j < len(s) || i > 0 {
				if !yield(s[i:j]) {
					return
				}
			}
			i = j + 1
		}
	}
}

// redactEachString calls f with every JSON string in data, keys and values
// alike, as it is written (escapes are left in; slugs never need any).
func redactEachString(data []byte, f func(string)) {
	for i := 0; i < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		j := i + 1
		for j < len(data) && data[j] != '"' {
			if data[j] == '\\' {
				j++
			}
			j++
		}
		if j > len(data) {
			j = len(data)
		}
		f(string(data[i+1 : j]))
		i = j
	}
}

// redactJSON marshals v or fails the test.
func redactJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	return data
}

// redactNilSlices returns the paths of the nil slices reachable from v.
func redactNilSlices(v any) []string {
	var out []string
	var walk func(rv reflect.Value, path string)
	walk = func(rv reflect.Value, path string) {
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			if !rv.IsNil() {
				walk(rv.Elem(), path)
			}
		case reflect.Slice:
			if rv.IsNil() {
				out = append(out, path)
				return
			}
			for i := range rv.Len() {
				walk(rv.Index(i), path+"["+strconv.Itoa(i)+"]")
			}
		case reflect.Array:
			for i := range rv.Len() {
				walk(rv.Index(i), path+"["+strconv.Itoa(i)+"]")
			}
		case reflect.Struct:
			for i := range rv.NumField() {
				walk(rv.Field(i), path+"."+rv.Type().Field(i).Name)
			}
		}
	}
	walk(reflect.ValueOf(v), reflect.TypeOf(v).String())
	return out
}

// TestRedactHelpers checks the scanner on small inputs, so a scanner that
// silently finds nothing cannot pass the sweep.
func TestRedactHelpers(t *testing.T) {
	hidden := map[string]string{"baths": "test", "law": "test"}
	cases := []struct {
		json string
		leak bool
	}{
		{`{"a":"baths"}`, true},
		{`{"baths":1}`, true},
		{`["construct:baths"]`, true},
		{`["wonder:pyramids:baths"]`, true},
		{`["token:law"]`, true},
		{`{"a":"bathsx","b":"great-library","c":"lawyer"}`, false},
		{`{"a":"\"baths\""}`, false},
		{`{"a":"x\\","b":"baths"}`, true},
	}
	for _, c := range cases {
		if got := redactLeaks([]byte(c.json), hidden) != ""; got != c.leak {
			t.Errorf("redactLeaks(%s) = %v, want %v", c.json, got, c.leak)
		}
	}
	type inner struct{ L []int }
	type outer struct {
		A []string
		B *inner
		C [2]inner
	}
	if got := redactNilSlices(outer{A: []string{}, B: &inner{}, C: [2]inner{{L: []int{}}, {}}}); len(got) != 2 {
		t.Errorf("redactNilSlices found %v, want B.L and C[1].L", got)
	}
}
