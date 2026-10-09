package protocoltest

import (
	"bytes"
	"math/bits"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// trueHidden returns the slugs that must not be visible to viewer right
// now, reconstructed directly from the engine's own exported state fields
// (not from view.go, which is the production code these tests exist to
// check). viewer nil means a spectator, the strictest case: a spectator
// never sees the Great Library offer either.
//
// This is a best-effort reconstruction, not a certified mirror of view.go's
// whitelist (see doc.go): it covers every category view.go's own doc
// comment lists (face-down slots, removed cards, later ages' deals, box
// guilds, box wonders, draft round two before round one ends, and the
// Great Library offer), which is precise enough for an integration leak
// check, since every id is unique within one game. Once the game is over,
// RevealView exposes all of this (the rooms this package builds use the
// default Options, so HideAtEnd is false), so trueHidden returns empty.
func trueHidden(s *engine.State, viewer *engine.Seat) map[string]bool {
	hidden := map[string]bool{}
	if s.Over() {
		return hidden
	}
	addCard := func(id catalog.CardID) {
		if id == 0 {
			return
		}
		if d := catalog.Card(id); d != nil {
			hidden[d.Slug] = true
		}
	}
	addWonder := func(id catalog.WonderID) {
		if id == 0 {
			return
		}
		if d := catalog.Wonder(id); d != nil {
			hidden[d.Slug] = true
		}
	}
	addToken := func(id catalog.TokenID) {
		if id == 0 {
			return
		}
		if d := catalog.Token(id); d != nil {
			hidden[d.Slug] = true
		}
	}

	for _, slot := range s.Slots {
		if !slot.FaceUp {
			addCard(slot.Card)
		}
	}
	for _, deal := range s.Deals {
		for _, id := range deal {
			addCard(id)
		}
	}
	// The cards set aside from every age's deck, not just the current one's:
	// an age that is over keeps its removed cards hidden until the game is.
	for _, removed := range s.Removed {
		for _, id := range removed {
			addCard(id)
		}
	}
	for _, id := range s.BoxGuilds {
		addCard(id)
	}
	for _, id := range s.BoxWonders {
		addWonder(id)
	}
	// Draft round two stays hidden until round one's four wonders are all
	// taken (view.go's viewDraft: round := draftRound(draftPicks(s)), where
	// draftPicks is popcount(DraftTaken)); this is not simply "fewer than
	// four actions applied" because the round's last wonder can be taken
	// automatically (no real choice left) in the same Apply call as the
	// third, so Ply alone undercounts picks taken near the round boundary.
	if s.Phase == engine.PhaseWonderDraft && bits.OnesCount8(s.DraftTaken) < 4 {
		for _, id := range s.DraftOrder[4:] {
			addWonder(id)
		}
	}
	for _, id := range s.BoxTokens {
		addToken(id)
	}
	for id := range s.OutOfPlay.All() {
		addCard(id)
	}

	// The Great Library offer: drawLibrary moves the drawn tokens out of
	// BoxTokens into Library, so they are not covered by the box above.
	// Only the deciding seat may see them; the other seat and a spectator
	// see only how many there are.
	decider := viewer != nil && s.Phase == engine.PhasePickLibraryToken && *viewer == s.ToMove
	if !decider {
		for _, id := range s.Library {
			addToken(id)
		}
	}
	return hidden
}

// leakTracker checks the new bytes a connection receives, since the last
// call, against the hidden set computed from the ground truth at the
// moment of the call. It also checks, over every byte ever received, that
// the room's seed never appears (the view package never includes it at
// all, not even once the game is over, so there is no "before game over"
// exception to carve out).
type leakTracker struct {
	t         *testing.T
	seed      []byte // the seed's decimal digits, as they would appear unquoted in JSON
	seedFound bool
	checked   map[*tclient]int // chunks already scanned, per connection
}

func newLeakTracker(t *testing.T, seed uint64) *leakTracker {
	return &leakTracker{t: t, seed: []byte(strconv.FormatUint(seed, 10)), checked: map[*tclient]int{}}
}

// check scans c's new bytes for anything in hidden (as a quoted JSON
// string, so "law" cannot spuriously match inside a longer word) and for
// the seed, labeling failures with who.
func (lt *leakTracker) check(c *tclient, who string, hidden map[string]bool) {
	lt.t.Helper()
	chunks, total := c.raw.since(lt.checked[c])
	lt.checked[c] = total
	patterns := make([][]byte, 0, len(hidden))
	for slug := range hidden {
		patterns = append(patterns, []byte(`"`+slug+`"`))
	}
	for _, chunk := range chunks {
		for _, pat := range patterns {
			if bytes.Contains(chunk, pat) {
				lt.t.Errorf("leak: %s received a hidden id %s: %s", who, pat, chunk)
			}
		}
		if !lt.seedFound && bytes.Contains(chunk, lt.seed) {
			lt.seedFound = true
			lt.t.Errorf("leak: %s received the room's seed: %s", who, chunk)
		}
	}
}
