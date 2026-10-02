package enginetest

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// A step names one action, either by its canonical key or by a short
// sentence of words separated by spaces:
//
//	construct:temple         build temple          construct temple
//	discard:baths            discard baths         sell baths
//	wonder:mausoleum:stable  wonder mausoleum stable
//	revive:baths             rebuild baths         revive baths
//	destroy:sawmill          destroy sawmill
//	token:law                token law             take law
//	starter:1                starter 1             start 1
//	draft:sphinx             draft sphinx
//
// Slugs are exact, as in keys (no leading "the-").

// stepVerbs maps the words of a step to key verbs.
var stepVerbs = []struct{ word, verb string }{
	{"build", "construct"}, {"construct", "construct"},
	{"discard", "discard"}, {"sell", "discard"},
	{"wonder", "wonder"},
	{"rebuild", "revive"}, {"revive", "revive"},
	{"destroy", "destroy"},
	{"token", "token"}, {"take", "token"},
	{"starter", "starter"}, {"start", "starter"},
	{"draft", "draft"},
}

// ParseStep turns a step into an action.
func ParseStep(step string) (engine.Action, error) {
	fields := strings.Fields(step)
	if len(fields) == 1 {
		return engine.ParseKey(fields[0])
	}
	if len(fields) == 0 {
		return engine.Action{}, errorf("empty step")
	}
	for _, v := range stepVerbs {
		if fields[0] == v.word {
			return engine.ParseKey(v.verb + ":" + strings.Join(fields[1:], ":"))
		}
	}
	return engine.Action{}, errorf("step %q: unknown verb %q", step, fields[0])
}

// Do applies the steps to *s in order, checking the invariants after each,
// and returns the events of all of them. It stops the test at the first step
// that does not parse, is not legal, or breaks an invariant, listing the
// legal keys.
func Do(tb testing.TB, s *engine.State, steps ...string) []engine.Event {
	tb.Helper()
	var all []engine.Event
	for i, step := range steps {
		a, err := ParseStep(step)
		if err != nil {
			tb.Fatalf("enginetest: step %d: %v", i, err)
		}
		var ev []engine.Event
		if err := engine.Apply(s, a, &ev); err != nil {
			tb.Fatalf("enginetest: step %d (%s): %v\nlegal: %s", i, step, err, strings.Join(Keys(s), " "))
		}
		if err := engine.CheckInvariants(s); err != nil {
			tb.Fatalf("enginetest: after step %d (%s): %v", i, step, err)
		}
		all = append(all, ev...)
	}
	return all
}

// MustFail checks that step is not legal in *s: Apply must return an error
// and leave the state (and so its hash) unchanged. It returns the error's
// reason.
func MustFail(tb testing.TB, s *engine.State, step string) engine.Reason {
	tb.Helper()
	a, err := ParseStep(step)
	if err != nil {
		tb.Fatalf("enginetest: MustFail(%q): %v", step, err)
	}
	before, hash := *s, engine.Hash(s)
	err = engine.Apply(s, a, nil)
	if err == nil {
		tb.Fatalf("enginetest: %s was applied, but it should be illegal", step)
	}
	if *s != before || engine.Hash(s) != hash {
		tb.Fatalf("enginetest: the failed %s changed the state", step)
	}
	var ie *engine.IllegalError
	if !errors.As(err, &ie) {
		tb.Fatalf("enginetest: %s failed with %v, not an *engine.IllegalError", step, err)
	}
	return ie.Reason
}

// Keys returns the keys of the legal actions in *s.
func Keys(s *engine.State) []string {
	legal := engine.LegalActions(s, make([]engine.Action, 0, engine.MaxLegalActions))
	keys := make([]string, len(legal))
	for i, a := range legal {
		keys[i] = a.Key()
	}
	return keys
}

// IsLegal reports whether step is legal in *s. A step that does not parse is
// not legal.
func IsLegal(s *engine.State, step string) bool {
	a, err := ParseStep(step)
	return err == nil && engine.IsLegal(s, a) == nil
}

func errorf(format string, args ...any) error {
	return fmt.Errorf("enginetest: "+format, args...)
}
