package ai

import (
	"testing"
	"unsafe"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The tests of arena.go: action keys and the node arena.

// TestActKeyRoundTrip: every legal action of every phase packs into a key
// and back, and the actions of one position have distinct keys.
func TestActKeyRoundTrip(t *testing.T) {
	var states []engine.State
	for seed := uint64(1); seed <= 3; seed++ {
		states = append(states, itestPositions(t, seed)...)
	}
	states = append(states, itestPendingPositions(t)...)
	checked := 0
	for _, s := range states {
		legal := engine.LegalActions(&s, nil)
		seen := make(map[actKey]engine.Action, len(legal))
		for _, a := range legal {
			k := keyOf(a)
			if k&keyWins != 0 {
				t.Fatalf("ply %d: the key of %s sets the decisive-move flag", s.Ply, a)
			}
			if back := k.action(); back != a {
				t.Fatalf("ply %d: %s packs to %#x and unpacks to %s", s.Ply, a, uint32(k), back)
			}
			if prev, dup := seen[k]; dup {
				t.Fatalf("ply %d: %s and %s share key %#x", s.Ply, prev, a, uint32(k))
			}
			seen[k] = a
			checked++
		}
	}
	for _, a := range []engine.Action{
		engine.StarterAction(0), engine.StarterAction(1),
		engine.TokenAction(catalog.TokenID(catalog.NumTokens)),
		engine.WonderAction(catalog.WonderID(catalog.NumWonders), catalog.CardID(catalog.NumCards)),
	} {
		if back := keyOf(a).action(); back != a {
			t.Errorf("%+v unpacks to %+v", a, back)
		}
	}
	t.Logf("%d actions checked", checked)
}

// TestNodeSize: a node is 24 bytes, so 1024 of them make a 24 KiB chunk.
func TestNodeSize(t *testing.T) {
	if n := unsafe.Sizeof(node{}); n != 24 {
		t.Fatalf("a node is %d bytes", n)
	}
}

// TestArenaLimit: an arena holds exactly its limit of nodes across chunk
// boundaries, keeps every node's contents, clamps its limit, and is
// reusable after release.
func TestArenaLimit(t *testing.T) {
	var a arena
	const limit = 2*arenaChunk + 5
	a.init(limit)
	for i := range limit {
		j, ok := a.alloc(actKey(i))
		if !ok || int(j) != i {
			t.Fatalf("alloc %d gave %d, %v", i, j, ok)
		}
		a.at(j).visits = uint32(i)
	}
	if !a.full() || a.len() != limit {
		t.Fatalf("after %d allocations: full %v, len %d", limit, a.full(), a.len())
	}
	if _, ok := a.alloc(1); ok {
		t.Fatal("a full arena gave a node")
	}
	if len(a.chunks) != 3 {
		t.Errorf("%d nodes took %d chunks", limit, len(a.chunks))
	}
	for i := range int32(limit) {
		if n := a.at(i); n.act != actKey(i) || n.visits != uint32(i) || n.child != 0 || n.sibling != 0 {
			t.Fatalf("node %d holds %+v", i, *n)
		}
	}
	a.release()
	if len(a.chunks) != 0 || a.len() != 0 || !a.full() {
		t.Fatalf("a released arena: %d chunks, len %d", len(a.chunks), a.len())
	}

	a.init(0)
	if a.limit != 1 {
		t.Errorf("init(0) gives a limit of %d, want 1", a.limit)
	}
	a.init(4 * MaxArenaNodes)
	if a.limit != MaxArenaNodes {
		t.Errorf("init(4*MaxArenaNodes) gives a limit of %d", a.limit)
	}
	// Recycled chunks hold stale nodes; alloc must clear them.
	a.init(10)
	for i := range 10 {
		j, _ := a.alloc(7)
		if n := a.at(j); n.visits != 0 || n.act != 7 {
			t.Fatalf("node %d of a recycled arena holds %+v", i, *n)
		}
	}
	a.release()
}
