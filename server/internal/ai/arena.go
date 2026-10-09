package ai

import (
	"sync"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// This file is the tree search's storage: an arena of fixed-size nodes that
// refer to each other by index, so that a tree is a few large blocks of
// memory instead of one heap object per node, and the garbage collector
// never walks it.
//
// Nodes live in chunks of arenaChunk nodes. A chunk is taken only when the
// tree grows into it, so the memory a search uses follows the iterations it
// actually runs, and finished searches hand their chunks back to a shared
// pool for the next one. The arena's limit, set from the search budget,
// caps the node count: a search of n iterations adds at most one node per
// iteration, so it never needs more than n+1 nodes (the root included). A
// full arena stops growing; the search goes on in the tree it has.

// actKey is an action packed into 32 bits: the type, card, wonder, token and
// starter seat, in that order from the high bits down. Every valid action
// has its own key, and keys sort by type first, then card, then wonder, so
// that the root statistics of several workers merge in a fixed order.
type actKey uint32

// keyWins flags a node whose action ended the game at once with a victory
// for the player who chose it (a decisive move). No action key uses this
// bit: an action type fits in four bits.
const keyWins actKey = 1 << 31

// keyOf packs a.
func keyOf(a engine.Action) actKey {
	return actKey(a.Type)<<24 | actKey(a.Card)<<16 | actKey(a.Wonder)<<8 | actKey(a.Token)<<1 | actKey(a.Starter&1)
}

// action unpacks k.
func (k actKey) action() engine.Action {
	return engine.Action{
		Type:    engine.ActionType(k >> 24),
		Card:    catalog.CardID(k >> 16),
		Wonder:  catalog.WonderID(k >> 8),
		Token:   catalog.TokenID(k >> 1 & 0x7f),
		Starter: engine.Seat(k & 1),
	}
}

// node is one node of a search tree: the action that leads to it from its
// parent and the statistics of the iterations that went through it. It is 24
// bytes.
//
// The tree is a single-observer information-set tree: a node stands for a
// sequence of actions from the root, whatever cards those actions revealed,
// and avail counts the iterations in which the node's action was legal (in
// that iteration's determinization) when its parent was reached.
type node struct {
	value   float32 // the sum of the rewards backed up through the node, for the player who chose its action
	visits  uint32  // iterations that went through the node
	avail   uint32  // iterations in which the node's action was available at its parent
	child   int32   // the first child, or 0 for none (the root, node 0, is nobody's child)
	sibling int32   // the next child of the same parent, or 0 for none
	act     actKey  // the action from the parent to this node, with keyWins for a decisive move
}

// key returns the node's action key, without the keyWins flag.
func (n *node) key() actKey { return n.act &^ keyWins }

// wins reports whether the node's action wins the game at once for the
// player who chose it.
func (n *node) wins() bool { return n.act&keyWins != 0 }

// Arena sizes.
const (
	arenaChunkBits = 10
	arenaChunk     = 1 << arenaChunkBits // nodes per chunk (24 KiB)
	arenaChunkMask = arenaChunk - 1

	// MaxArenaNodes caps the nodes of one worker's tree (24 MiB), whatever
	// the budget.
	MaxArenaNodes = 1 << 20
)

// nodeChunk is one block of nodes.
type nodeChunk [arenaChunk]node

// chunkPool recycles the chunks of finished searches. A chunk taken from it
// holds stale nodes; alloc overwrites each node it hands out.
var chunkPool = sync.Pool{New: func() any { return new(nodeChunk) }}

// arena holds one tree. The zero arena has no room; call init first.
type arena struct {
	chunks []*nodeChunk
	n      int32 // nodes in use
	limit  int32 // the most nodes the arena may hold
}

// init empties the arena and lets it hold up to limit nodes (at least one,
// at most MaxArenaNodes). It keeps the chunks it already has.
func (a *arena) init(limit int) {
	limit = min(max(limit, 1), MaxArenaNodes)
	a.n, a.limit = 0, int32(limit)
	if need := (limit + arenaChunkMask) >> arenaChunkBits; cap(a.chunks) < need {
		chunks := make([]*nodeChunk, len(a.chunks), need)
		copy(chunks, a.chunks)
		a.chunks = chunks
	}
}

// release hands the arena's chunks back to the pool and leaves the arena
// without room.
func (a *arena) release() {
	for i, c := range a.chunks {
		chunkPool.Put(c)
		a.chunks[i] = nil
	}
	a.chunks = a.chunks[:0]
	a.n, a.limit = 0, 0
}

// len returns the number of nodes in use.
func (a *arena) len() int { return int(a.n) }

// full reports whether the arena has no room for another node.
func (a *arena) full() bool { return a.n >= a.limit }

// alloc adds a node for action k, with no statistics and no links, and
// returns its index; ok is false when the arena is full. It allocates memory
// only when the tree grows into a new chunk and the pool has none to spare.
func (a *arena) alloc(k actKey) (i int32, ok bool) {
	if a.n >= a.limit {
		return 0, false
	}
	i = a.n
	if c := int(i >> arenaChunkBits); c == len(a.chunks) {
		a.chunks = append(a.chunks, chunkPool.Get().(*nodeChunk))
	}
	a.n++
	*a.at(i) = node{act: k}
	return i, true
}

// at returns node i, which must be in use.
func (a *arena) at(i int32) *node {
	return &a.chunks[i>>arenaChunkBits][i&arenaChunkMask]
}
