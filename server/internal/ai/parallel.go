package ai

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file runs one tree search on its workers (root parallelism) and
// turns their trees into a move.
//
// Every worker grows its own tree from the same root, with its own random
// stream, so the workers share nothing while they search but the stop flag.
// The first worker runs on the calling goroutine (a one-worker search starts
// no goroutine) and also sends the progress reports; the others run on
// goroutines of their own. When all have stopped, the statistics of the
// root's children are summed over the workers, action by action in the
// order of the root's legal actions, worker by worker in index order, so
// the sums (floating point included) do not depend on how the goroutines
// were scheduled. The move is the child with the most visits in the sum.
//
// A worker stops when it has run its share of the iterations or when the
// stop flag is set. The flag is set by the context (through
// context.AfterFunc, so a cancel reaches the workers within one iteration),
// by the MaxTime limit (a timeout on the same context) and, as a backstop,
// by any worker that sees the context done when it checks every checkEvery
// iterations.

// phi64 is 2^64 divided by the golden ratio: the step between the seeds of
// two workers.
const phi64 = 0x9e3779b97f4a7c15

// rootStats are the statistics of the root's children, indexed like the
// root's legal actions: a worker's own, or their sum over the workers.
type rootStats struct {
	visits     [engine.MaxLegalActions]uint64
	value      [engine.MaxLegalActions]float64 // total reward, for the observer
	iterations int                             // iterations run
}

// reset empties the statistics.
func (rs *rootStats) reset() { *rs = rootStats{} }

// add sums o into rs.
func (rs *rootStats) add(o *rootStats) {
	for j := range rs.visits {
		rs.visits[j] += o.visits[j]
		rs.value[j] += o.value[j]
	}
	rs.iterations += o.iterations
}

// best returns the index of the child to play among the first n: the most
// visited, then the one of higher total reward, then the first. It returns
// -1 when no child has a visit.
func (rs *rootStats) best(n int) int {
	best := -1
	for j := range n {
		v := rs.visits[j]
		if v == 0 {
			continue
		}
		if best < 0 || v > rs.visits[best] || (v == rs.visits[best] && rs.value[j] > rs.value[best]) {
			best = j
		}
	}
	return best
}

// share returns the part of the root's visits that went to child j.
func (rs *rootStats) share(j, n int) float64 {
	total := uint64(0)
	for k := range n {
		total += rs.visits[k]
	}
	if total == 0 || j < 0 {
		return 0
	}
	return float64(rs.visits[j]) / float64(total)
}

// searchRun is one search: the root, the workers and what they share.
type searchRun struct {
	ctx        context.Context
	legal      []engine.Action                // the root's legal actions
	keys       [engine.MaxLegalActions]actKey // their keys
	workers    []mctsWorker                   // worker 0 runs on the calling goroutine
	stop       atomic.Bool                    // set when the search must end
	start      time.Time                      // when the search began
	onProgress func(Progress)                 // nil: no reports
	lastReport time.Time                      // when worker 0 last reported
	merged     rootStats                      // scratch for the reports and the result
	fallback   rolloutPolicy                  // picks a move when no iteration finished
	first      *engine.State                  // the search's first determinization, for the fallback
	rng        engine.RNG                     // for the fallback
}

// newSearchRun prepares a search of the observation obs, whose legal
// actions are legal, under plan p. seeds derive the workers' random
// streams. ctx stops the search; the caller must call finish when done.
func (m *MCTSBot) newSearchRun(ctx context.Context, obs *engine.Observation, legal []engine.Action, p searchPlan, seeds [2]uint64, first *engine.State, onProgress func(Progress)) *searchRun {
	r := &searchRun{ctx: ctx, legal: legal, start: time.Now(), onProgress: onProgress, first: first}
	r.lastReport = r.start
	for j, a := range legal {
		r.keys[j] = keyOf(a)
	}
	r.workers = make([]mctsWorker, p.workers)
	for i := range r.workers {
		limit, nodes := p.worker(i)
		rng := engine.NewRNG2(seeds[0]^uint64(i)*phi64, seeds[1])
		r.workers[i].init(m, obs, rng, limit, nodes, r)
	}
	r.fallback.init(RolloutConfig{Epsilon: -1, ModeProb: -1}.resolved(), &m.w)
	r.rng = engine.NewRNG2(seeds[1], seeds[0])
	return r
}

// search runs a search of obs under plan p and returns the move (see the
// top of this file and of mcts.go).
func (m *MCTSBot) search(ctx context.Context, obs *engine.Observation, legal []engine.Action, p searchPlan, seeds [2]uint64, first *engine.State, onProgress func(Progress)) engine.Action {
	if p.maxTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.maxTime)
		defer cancel()
	}
	r := m.newSearchRun(ctx, obs, legal, p, seeds, first, onProgress)
	defer r.finish()
	stopOnDone := context.AfterFunc(ctx, func() { r.stop.Store(true) })
	defer stopOnDone()
	r.runWorkers()
	return r.result()
}

// runWorkers runs the workers until all have stopped: worker 0 on this
// goroutine, the others on goroutines of their own.
func (r *searchRun) runWorkers() {
	var wg sync.WaitGroup
	for i := 1; i < len(r.workers); i++ {
		w := &r.workers[i]
		wg.Go(func() { w.run(false) })
	}
	r.workers[0].run(true)
	wg.Wait()
}

// result merges the workers' trees, sends the final progress report and
// returns the move. It must run after every worker has stopped.
func (r *searchRun) result() engine.Action {
	n := len(r.legal)
	r.merged.reset()
	for i := range r.workers {
		w := &r.workers[i]
		w.collect(&w.own)
		r.merged.add(&w.own)
	}
	j := r.merged.best(n)
	var a engine.Action
	if j >= 0 {
		a = r.legal[j]
	} else {
		// Not one iteration finished (the context was done from the
		// start): play the rollout policy's choice, without randomness.
		d := *r.first
		a = r.fallback.move(&d, &r.rng)
	}
	if r.onProgress != nil {
		r.onProgress(Progress{Iterations: r.merged.iterations, Elapsed: time.Since(r.start), Best: a, BestShare: r.merged.share(j, n)})
	}
	return a
}

// finish hands the workers' memory back for the next search.
func (r *searchRun) finish() {
	for i := range r.workers {
		r.workers[i].tree.release()
	}
}

// run runs the worker's iterations until its share is done or the search
// stops. Worker 0 (lead) also sends the progress reports; the other workers
// publish snapshots of their root statistics for it.
func (w *mctsWorker) run(lead bool) {
	r := w.sr
	for w.running() {
		w.iterate()
		if w.done%checkEvery != 0 {
			continue
		}
		if r.ctx.Err() != nil {
			r.stop.Store(true)
		}
		if r.onProgress == nil {
			continue
		}
		if lead {
			r.report()
		} else {
			w.publish()
		}
	}
	if !lead && r.onProgress != nil {
		w.publish()
	}
}

// collect writes the worker's root statistics into rs.
func (w *mctsWorker) collect(rs *rootStats) {
	rs.reset()
	rs.iterations = w.done
	keys := w.sr.keys[:len(w.sr.legal)]
	t := &w.tree
	for c := t.at(0).child; c != 0; {
		cn := t.at(c)
		for j, k := range keys {
			if k == cn.key() {
				rs.visits[j] = uint64(cn.visits)
				rs.value[j] = float64(cn.value)
				break
			}
		}
		c = cn.sibling
	}
}

// publish stores a snapshot of the worker's root statistics for worker 0's
// progress reports.
func (w *mctsWorker) publish() {
	w.collect(&w.own)
	w.mu.Lock()
	w.snap = w.own
	w.mu.Unlock()
}

// report sends a progress report if progressEvery has passed since the
// last one. Only worker 0 calls it, while the others are still searching:
// it sums its own live statistics and the others' latest snapshots.
func (r *searchRun) report() {
	now := time.Now()
	if now.Sub(r.lastReport) < progressEvery {
		return
	}
	r.lastReport = now
	n := len(r.legal)
	r.merged.reset()
	w0 := &r.workers[0]
	w0.collect(&w0.own)
	r.merged.add(&w0.own)
	for i := 1; i < len(r.workers); i++ {
		w := &r.workers[i]
		w.mu.Lock()
		r.merged.add(&w.snap)
		w.mu.Unlock()
	}
	j := r.merged.best(n)
	best := r.legal[0]
	if j >= 0 {
		best = r.legal[j]
	}
	r.onProgress(Progress{Iterations: r.merged.iterations, Elapsed: now.Sub(r.start), Best: best, BestShare: r.merged.share(j, n)})
}
