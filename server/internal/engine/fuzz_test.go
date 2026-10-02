package engine_test

import (
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// fuzzMaxPlies bounds FuzzPlayout's steps so a fuzzer-grown choices slice
// cannot make one run arbitrarily long; the property sweep already checks
// real termination within maxPlies.
const fuzzMaxPlies = maxPlies * 2

// FuzzPlayout plays seed's game, picking legal[choices[i]%len(legal)] at each
// step, and checks CheckInvariants after every one. This is the fuzz target
// `make fuzz` runs (see the Makefile's fuzz target).
func FuzzPlayout(f *testing.F) {
	f.Add(uint64(1), []byte{0, 1, 2, 3, 4, 5, 6, 7})
	f.Add(uint64(2), []byte{255, 255, 0, 128, 7, 3})
	f.Add(uint64(0), []byte{})
	f.Add(uint64(1)<<63+12345, []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	f.Fuzz(func(t *testing.T, seed uint64, choices []byte) {
		s := engine.NewGame(seed, engine.Options{})
		if err := engine.CheckInvariants(&s); err != nil {
			t.Fatalf("seed %d: the new game breaks invariants: %v", seed, err)
		}
		legal := make([]engine.Action, 0, engine.MaxLegalActions)
		n := len(choices)
		if n > fuzzMaxPlies {
			n = fuzzMaxPlies
		}
		for i := 0; i < n && !s.Over(); i++ {
			legal = engine.LegalActions(&s, legal[:0])
			if len(legal) == 0 {
				t.Fatalf("seed %d step %d: no legal action in phase %s", seed, i, s.Phase)
			}
			a := legal[int(choices[i])%len(legal)]
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatalf("seed %d step %d: legal action %s failed: %v", seed, i, a, err)
			}
			if err := engine.CheckInvariants(&s); err != nil {
				t.Fatalf("seed %d step %d after %s: %v", seed, i, a, err)
			}
		}
	})
}

// FuzzActionKey checks that ParseKey never panics on any string, and that
// whenever it accepts one, the parsed action's own Key (and UnmarshalText)
// reproduce it exactly: Key and ParseKey are exact inverses on valid keys.
func FuzzActionKey(f *testing.F) {
	for _, k := range []string{
		"draft:pyramids", "construct:baths", "discard:baths", "wonder:pyramids:baths",
		"destroy:quarry", "revive:garrison", "token:law", "starter:0", "starter:1",
		"", ":", "construct:", "construct", "construct:the-baths", "CONSTRUCT:baths",
		"construct:baths ", " construct:baths", "construct:baths:baths", "nonsense:x",
		"draft:pyramids:baths", "wonder:pyramids", "starter:2", "starter:-1", "starter:00",
		"token:", "revive:revive:baths",
	} {
		f.Add(k)
	}
	f.Fuzz(func(t *testing.T, key string) {
		a, err := engine.ParseKey(key)
		if err != nil {
			if !errors.Is(err, engine.ErrBadKey) {
				t.Fatalf("ParseKey(%q) error %v does not wrap ErrBadKey", key, err)
			}
			return
		}
		if got := a.Key(); got != key {
			t.Fatalf("ParseKey(%q) = %+v, whose own Key() is %q", key, a, got)
		}
		var b engine.Action
		if err := b.UnmarshalText([]byte(key)); err != nil || b != a {
			t.Fatalf("UnmarshalText(%q) = %+v, %v; want %+v, nil", key, b, err, a)
		}
	})
}

// fuzzGarbageSteps bounds the random prefix FuzzApplyGarbage plays before its
// garbage action. Sweep games last about 75 actions, so a prefix of up to 79
// reaches every Age and the late pending choices, not just the wonder draft.
const fuzzGarbageSteps = 80

// FuzzApplyGarbage builds an arbitrary Action from fuzzed fields against a
// state reached by a seed-chosen number of random steps (so every phase gets
// exercised, not just the wonder draft), and applies it. A well-formed but
// illegal, or malformed, action must fail with an *IllegalError and leave the
// state untouched; one that happens to be legal must apply cleanly. Neither
// case may panic.
func FuzzApplyGarbage(f *testing.F) {
	f.Add(uint64(1), uint8(0), uint8(0), uint8(0), uint8(0), uint8(0))
	f.Add(uint64(2), uint8(2), uint8(9), uint8(0), uint8(0), uint8(0))
	f.Add(uint64(3), uint8(255), uint8(255), uint8(255), uint8(255), uint8(255))
	f.Add(uint64(4), uint8(7), uint8(1), uint8(1), uint8(1), uint8(3))
	f.Fuzz(func(t *testing.T, seed uint64, typ, card, wonder, token, starter uint8) {
		s := engine.NewGame(seed, engine.Options{})
		pol := enginetest.NewRandomPolicy(seed ^ 0xa11a5)
		buf := make([]engine.Action, 0, engine.MaxLegalActions)
		for i, steps := 0, int(seed%fuzzGarbageSteps); i < steps && !s.Over(); i++ {
			legal := engine.LegalActions(&s, buf[:0])
			engine.ApplyTrusted(&s, pol.Choose(&s, legal))
		}
		if s.Over() {
			return // nothing is legal once the game ends; IsLegal/Apply are covered elsewhere for that
		}
		a := engine.Action{
			Type:    engine.ActionType(typ),
			Card:    catalog.CardID(card),
			Wonder:  catalog.WonderID(wonder),
			Token:   catalog.TokenID(token),
			Starter: engine.Seat(starter),
		}
		before, hash := s, engine.Hash(&s)
		legalErr := engine.IsLegal(&s, a)
		err := engine.Apply(&s, a, nil)
		if legalErr == nil {
			if err != nil {
				t.Fatalf("IsLegal accepted %s but Apply failed: %v", a, err)
			}
			if err := engine.CheckInvariants(&s); err != nil {
				t.Fatalf("after applying the legal garbage action %s: %v", a, err)
			}
			return
		}
		var ie *engine.IllegalError
		if err == nil || !errors.As(err, &ie) {
			t.Fatalf("garbage action %s: got %v, want an *IllegalError (IsLegal said %v)", a, err, legalErr)
		}
		if s != before || engine.Hash(&s) != hash {
			t.Fatalf("the rejected garbage action %s changed the state", a)
		}
	})
}
