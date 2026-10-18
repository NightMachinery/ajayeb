package ai

import (
	"context"
	"math"
	"math/bits"
	"slices"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file is the tree search: single-observer information-set Monte Carlo
// tree search (SO-ISMCTS) with blind reveals, the approach of Cowling,
// Powley and Whitehouse (2012) as applied to this game by indjev99's agents.
//
// The search grows one tree over the observer's information sets. A node is
// a sequence of actions from the root, whatever cards those actions turned
// over: the tree never branches on a reveal, and since an action names its
// card, "take card X" exists only below the nodes where X was face up. Both
// players' moves live in the same tree; the opponent knows almost exactly
// what the observer knows (the face-down cards are hidden from both), so
// its decisions are searched the same way.
//
// One iteration:
//
//  1. Determinize: draw a complete state d that agrees with the observation
//     (Observation.Determinize), so every hidden card has a value.
//  2. Descend from the root using only the actions legal in d. At each node,
//     every child whose action is legal in d gets one more availability.
//     If one of them is a decisive move (its action won the game at once
//     for the player who chose it, when the child was added), the descent
//     takes it. Otherwise, if some legal action has no child yet, one of
//     them, chosen at random, is added as a new node (expansion) and the
//     descent stops; else the descent follows the child of highest UCB, for
//     the player to move, Q/N + C*sqrt(ln(avail)/N): Q the child's total
//     reward, N its visits, avail its availability count and C about 0.7
//     (rewards are in [0, 1]).
//  3. Roll out from there with the heuristic policy (rollout.go) and score
//     the position reached: a reward in [0, 1] for seat 0.
//  4. Back up: every node of the path gets a visit and the reward, as seen
//     by the player who chose its action (r for seat 0, 1-r for seat 1).
//
// Decisive moves matter because a win is often several actions (one wonder
// with each of the cards it can tuck) next to moves the evaluation already
// rates near 1 (the pawn one space from the capital): plain UCB would share
// the visits among them, and the most visited child need not win. Once a
// node knows its winning move, every iteration through it takes the move,
// so a win at the root gets the visits, and an opponent's win below one of
// the observer's moves counts as a loss in every iteration that reaches it.
// A win at once depends only on public information (the shields, symbols
// and scores that the path's actions bring), so the flag holds in every
// determinization in which the action is legal.
//
// The move played is the root's most visited child; ties go to the higher
// total reward, then to the first in the order of the legal actions.
// Several workers can search at once, each with its own tree and random
// generator (root parallelism); their root statistics are summed by action
// (parallel.go). If not one iteration finished (the context was done from
// the start), the move is the rollout policy's choice in the first
// determinization, without its random moves.
//
// Before searching, Choose plays a sure win if there is one: an action that
// wins at once, or through a choice of its own, using only public
// information (the same check as Greedy's, greedySearch.sureWin). The search
// would find these too, but a sure win needs no statistics.
//
// # Budgets and determinism
//
// Budget.MaxIterations caps the iterations of the whole search: with W
// workers each runs about MaxIterations/W. Budget.MaxTime caps the wall
// clock. A Budget that sets neither gets the bot's defaults (MCTSConfig.Budget,
// or DefaultMCTSIterations), and Workers 0 means the bot's default worker
// count. Under an iteration budget with no time limit, the search is
// deterministic for a given bot seed and observation, whatever the number of
// workers and however the goroutines are scheduled: each worker's random
// stream depends only on the seed, the observation and the worker's index,
// each runs a fixed number of iterations, and the merge sums the statistics
// in worker order.
//
// The search stops early when the context is done or MaxTime has passed;
// every worker looks at a shared stop flag before each iteration (the
// context, with MaxTime as its timeout, sets it through context.AfterFunc),
// so a cancelled search returns within about one iteration. Every
// checkEvery iterations each worker also looks at the context itself.
// Choose then returns the best action found so far. Progress reports go out
// about every progressEvery (worker 0 looks at the clock every checkEvery
// iterations) and once at the end.
//
// # Memory
//
// Each worker's tree lives in an arena (arena.go) whose limit is the
// worker's share of the iteration budget plus one (the root), and, under a
// time limit, MaxTime times arenaNodesPerSecond; never more than
// MaxArenaNodes. A search adds at most one node per iteration, so the limit
// is never in the way of an iteration budget. The arena takes memory only
// as the tree grows, 24 KiB (1024 nodes of 24 bytes) at a time, and
// recycles it between searches. An iteration allocates nothing. At the
// rates measured on an Apple M2, a Medium search (0.4 s, one worker) grows
// a tree of about 1 MB, and a Hard search (2 s, four workers, rollouts to
// the end of the game) about 1 to 2 MB per worker in the middle game and up
// to 5 MB per worker late in Age III.

// Defaults of the tree search.
const (
	DefaultUCBC           = 0.7   // the UCB exploration constant, for rewards in [0, 1]
	DefaultMCTSIterations = 20000 // the iteration budget of a bot configured with no budget at all
	MaxSearchWorkers      = 8     // the most workers one search runs

	// arenaNodesPerSecond bounds the iterations one worker runs per
	// second, to size an arena from a time limit. Measured on an Apple M2:
	// about 95k per second in the draft and Ages I and II, 230k late in
	// Age III. A search that outruns it keeps searching in the tree it has.
	arenaNodesPerSecond = 250_000

	// maxTreeDepth bounds the depth of a descent; deeper, the iteration
	// rolls out from where it stands. A tree reaches it only if a search
	// runs far more iterations than any budget allows.
	maxTreeDepth = 256
)

// MCTSConfig configures a tree-search bot. The zero value searches
// DefaultMCTSIterations iterations on one worker, with the default UCB
// constant, rollouts and evaluation weights.
type MCTSConfig struct {
	// Budget is the default budget: Choose uses its MaxIterations and
	// MaxTime when its own Budget sets neither, and its Workers when its
	// own Budget's Workers is 0.
	Budget  Budget        `json:"-"`
	C       float64       // the UCB exploration constant; 0 means DefaultUCBC
	Rollout RolloutConfig // the rollouts; the zero value is the default policy
	Weights *Weights      `json:"-"` // the evaluation's weights; nil means DefaultWeights
}

// MCTSBot is the tree-search bot (the Medium and Hard levels). It holds no
// mutable state, so one bot may serve several goroutines.
type MCTSBot struct {
	seed    uint64
	budget  Budget
	c       float64
	rollout RolloutConfig
	w       Weights
}

// NewMCTS returns a tree-search bot seeded with seed.
func NewMCTS(seed uint64, cfg MCTSConfig) *MCTSBot {
	m := &MCTSBot{seed: seed, budget: cfg.Budget, c: cfg.C, rollout: cfg.Rollout.resolved(), w: DefaultWeights()}
	if m.c <= 0 || math.IsNaN(m.c) {
		m.c = DefaultUCBC
	}
	if cfg.Weights != nil {
		m.w = *cfg.Weights
	}
	return m
}

// Choose picks an action by tree search, as described at the top of this
// file.
func (m *MCTSBot) Choose(ctx context.Context, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error) {
	var rootBuf [engine.MaxLegalActions]engine.Action
	legal, err := choiceSetup(&obs, rootBuf[:0])
	if err != nil {
		return engine.Action{}, err
	}
	if len(legal) == 1 {
		return legal[0], nil
	}
	rng := callRNG(m.seed, &obs)
	seeds := [2]uint64{rng.Uint64(), rng.Uint64()}
	first := obs.Determinize(&rng)
	sr := greedySearch{me: obs.Seat(), w: &m.w, ctx: ctx}
	if i, ok := sr.sureWin(&first, legal); ok {
		return legal[i], nil
	}
	return m.search(ctx, &obs, legal, m.plan(b), seeds, &first, onProgress), nil
}

// searchPlan is a budget resolved for one search.
type searchPlan struct {
	iterations int           // over all workers; 0: no cap (a time limit stops the search)
	maxTime    time.Duration // 0: no limit beyond the context
	workers    int           // at least 1, at most MaxSearchWorkers, and at most iterations when that is set
}

// plan resolves b against the bot's defaults.
func (m *MCTSBot) plan(b Budget) searchPlan {
	p := searchPlan{iterations: max(b.MaxIterations, 0), maxTime: max(b.MaxTime, 0), workers: b.Workers}
	if p.iterations == 0 && p.maxTime == 0 {
		p.iterations, p.maxTime = max(m.budget.MaxIterations, 0), max(m.budget.MaxTime, 0)
		if p.iterations == 0 && p.maxTime == 0 {
			p.iterations = DefaultMCTSIterations
		}
	}
	if p.workers <= 0 {
		p.workers = m.budget.Workers
	}
	p.workers = min(max(p.workers, 1), MaxSearchWorkers)
	if p.iterations > 0 {
		p.workers = min(p.workers, p.iterations)
	}
	return p
}

// worker returns worker i's iteration cap (0: none) and arena limit.
func (p *searchPlan) worker(i int) (limit, nodes int) {
	nodes = MaxArenaNodes
	if p.iterations > 0 {
		limit = p.iterations / p.workers
		if i < p.iterations%p.workers {
			limit++
		}
		nodes = min(nodes, limit+1)
	}
	if p.maxTime > 0 {
		nodes = min(nodes, int(p.maxTime.Seconds()*arenaNodesPerSecond)+1)
	}
	return limit, nodes
}

// mctsWorker is one search worker: a tree, a random stream and the scratch
// space of an iteration. Only its own goroutine touches it while it runs,
// except snap, which mu guards.
type mctsWorker struct {
	tree  arena
	rng   engine.RNG
	obs   *engine.Observation
	sr    *searchRun // the search the worker belongs to
	c     float64
	roll  rolloutPolicy
	limit int // iterations to run; 0: until stopped
	done  int // iterations run

	// forced, when set, makes the search a comparison (review.go): the
	// root's action is forced[done % len(forced)] rather than chosen by
	// UCB, and each round of len(forced) iterations shares one
	// determinization, pair.
	forced []actKey
	pair   engine.State

	legal [engine.MaxLegalActions]engine.Action
	keys  [engine.MaxLegalActions]actKey
	path  [maxTreeDepth]pathStep

	own  rootStats  // scratch for the worker's root statistics
	mu   sync.Mutex // guards snap
	snap rootStats  // the root statistics as of the last publish, for progress reports
}

// pathStep is one node of a descent and the seat that chose its action.
type pathStep struct {
	node  int32
	mover engine.Seat
}

// init prepares the worker for a search. The tree gets its root.
func (w *mctsWorker) init(m *MCTSBot, obs *engine.Observation, rng engine.RNG, limit, nodes int, sr *searchRun) {
	w.rng, w.obs, w.sr, w.c, w.limit, w.done = rng, obs, sr, m.c, limit, 0
	w.forced = nil
	w.roll.init(m.rollout, &m.w)
	w.tree.init(nodes)
	w.tree.alloc(0) // the root; its action is never read
}

// running reports whether the worker should run another iteration.
func (w *mctsWorker) running() bool {
	return (w.limit == 0 || w.done < w.limit) && !w.sr.stop.Load()
}

// iterate runs one iteration: determinize, descend, expand, roll out, back
// up (see the top of this file).
func (w *mctsWorker) iterate() {
	var d engine.State
	t := &w.tree
	v := int32(0)
	depth := 0
	descend := true
	if w.forced != nil {
		v, descend = w.forcedRoot(&d)
		if v != 0 {
			depth = 1
		}
	} else {
		d = w.obs.Determinize(&w.rng)
	}
	for descend && !d.Over() && depth < maxTreeDepth {
		legal := engine.LegalActions(&d, w.legal[:0])
		keys := w.keys[:len(legal)]
		for j, a := range legal {
			keys[j] = keyOf(a)
		}
		// Count availability and find the best child legal in d: a
		// decisive one if there is one, else the one of highest UCB.
		var tried uint64
		best, bestU := int32(0), math.Inf(-1)
		win := int32(0)
		for c := t.at(v).child; c != 0; {
			cn := t.at(c)
			if j := slices.Index(keys, cn.key()); j >= 0 {
				tried |= 1 << j
				cn.avail++
				if cn.wins() {
					win = c
				}
				if u := w.ucb(cn); u > bestU {
					best, bestU = c, u
				}
			}
			c = cn.sibling
		}
		mover := d.ToMove
		if win != 0 {
			best = win
		} else if untried := len(legal) - bits.OnesCount64(tried); untried > 0 && (best == 0 || !t.full()) {
			// Expand one untried action, chosen at random. In a full arena
			// with no tried child to follow, play it without a node.
			j := nthZero(tried, w.rng.IntN(untried))
			engine.ApplyTrusted(&d, legal[j])
			k := keys[j]
			if d.Over() && d.Result.Winner == int8(mover) {
				k |= keyWins
			}
			if c, ok := t.alloc(k); ok {
				cn, pn := t.at(c), t.at(v)
				cn.avail = 1
				cn.sibling, pn.child = pn.child, c
				w.path[depth] = pathStep{node: c, mover: mover}
				depth++
			}
			break
		}
		engine.ApplyTrusted(&d, t.at(best).key().action())
		w.path[depth] = pathStep{node: best, mover: mover}
		depth++
		v = best
	}

	r := w.roll.run(&d, &w.rng)
	t.at(0).visits++
	for _, st := range w.path[:depth] {
		n := t.at(st.node)
		n.visits++
		if st.mover == 0 {
			n.value += float32(r)
		} else {
			n.value += float32(1 - r)
		}
	}
	w.done++
}

// forcedRoot starts an iteration of a comparison (see forced): it puts
// the round's determinization in d, plays the root action whose turn it
// is, and records the root's child for it as the first step of the path.
// The child is added on the action's first iteration, and the descent
// then stops there, as after any expansion; the arena always has room for
// it (a search of n iterations adds at most n nodes). It returns the child
// (0 if it could not be stored) and whether the descent goes on below it.
func (w *mctsWorker) forcedRoot(d *engine.State) (int32, bool) {
	j := w.done % len(w.forced)
	if j == 0 {
		w.pair = w.obs.Determinize(&w.rng)
	}
	*d = w.pair
	k := w.forced[j]
	mover := d.ToMove
	t := &w.tree
	c := t.at(0).child
	for c != 0 && t.at(c).key() != k {
		c = t.at(c).sibling
	}
	engine.ApplyTrusted(d, k.action())
	descend := c != 0
	if c == 0 {
		nk := k
		if d.Over() && d.Result.Winner == int8(mover) {
			nk |= keyWins
		}
		var ok bool
		if c, ok = t.alloc(nk); !ok {
			return 0, false
		}
		cn, rn := t.at(c), t.at(0)
		cn.sibling, rn.child = rn.child, c
	}
	t.at(c).avail++
	w.path[0] = pathStep{node: c, mover: mover}
	return c, descend
}

// ucb is the child's UCB value for the player choosing it. A child always
// has a visit once its first iteration is backed up; an unvisited one comes
// first.
func (w *mctsWorker) ucb(n *node) float64 {
	if n.visits == 0 {
		return math.Inf(1)
	}
	nv := float64(n.visits)
	return float64(n.value)/nv + w.c*math.Sqrt(math.Log(float64(n.avail))/nv)
}

// nthZero returns the index of the k-th (from 0) zero bit of mask.
func nthZero(mask uint64, k int) int {
	for j := 0; ; j++ {
		if mask&(1<<j) == 0 {
			if k == 0 {
				return j
			}
			k--
		}
	}
}
