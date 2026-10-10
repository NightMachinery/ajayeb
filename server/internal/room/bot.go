package room

import (
	"context"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// This file runs the bot seat of a bot room. When the bot is to move and its
// human is attached, the room starts one search in its own goroutine, under
// a context that the room cancels when the search goes stale (an undo), when
// the last connection leaves, or when the room stops. The
// search sees only an engine Observation of the bot's seat. Its answer comes
// back through the inbox tagged with the generation of the search, and the
// room drops it unless that search is still the one running for the
// current version. The generation, not the version, is what tells searches
// apart: a detach cancels a search without changing the version, and the
// next attach starts another for the same version, so the cancelled one
// may still answer while the new one runs.

// botTask is the search the room has running, if any.
type botTask struct {
	cancel  context.CancelFunc // nil when no search is running
	gen     uint64             // the search's generation: Room.botGen when it started
	version int                // the version the search was started for
	seat    engine.Seat        // the bot's seat
}

// botSeed is the seed of the bot that decides at ply: a function of the
// room's seed and the position, so a game against a bot replays the same
// way and an undo cannot reroll the bot's answer.
func botSeed(roomSeed uint64, ply uint16) uint64 { return ai.MixSeed(roomSeed, uint64(ply)) }

// maybeStartBot starts a search if a bot is to move in a running game, a
// connection is attached, and no search is running for the current version
// yet. A room nobody is attached to never searches: it would spend the bot
// pool on a game nobody is watching (detach cancels the search, and the
// next attach starts it again).
func (r *Room) maybeStartBot() {
	if r.file.Mode != store.ModeAI || r.status() != store.StatusPlaying || len(r.clients) == 0 {
		return
	}
	seat := r.state.ToMove
	sd := r.file.Seats[seat]
	if sd.Kind != store.SeatKindBot {
		return
	}
	if r.bot.cancel != nil && r.bot.version == r.version {
		return
	}
	r.cancelBot()
	ctx, cancel := context.WithCancel(r.ctx)
	r.botGen++
	r.bot = botTask{cancel: cancel, gen: r.botGen, version: r.version, seat: seat}

	cfg := &r.m.cfg
	job := botJob{
		bot:     cfg.NewBot(sd.Bot, botSeed(r.file.Seed, r.state.Ply)),
		obs:     engine.Observe(&r.state, seat),
		budget:  cfg.BotBudget(sd.Bot),
		gen:     r.botGen,
		version: r.version,
		pool:    cfg.BotPool,
		clock:   cfg.Clock,
		delay:   cfg.BotDelay,
		inbox:   r.inbox,
		done:    r.done,
	}
	r.m.wg.Add(1)
	go func() {
		defer r.m.wg.Done()
		job.run(ctx)
	}()
}

// thinking returns the seat whose bot is searching for the current
// version, or nil: the state frames' "thinking".
func (r *Room) thinking() *int {
	if r.bot.cancel == nil || r.bot.version != r.version {
		return nil
	}
	seat := int(r.bot.seat)
	return &seat
}

// cancelBot cancels the running search, if any.
func (r *Room) cancelBot() {
	if r.bot.cancel != nil {
		r.bot.cancel()
	}
	r.bot = botTask{}
}

// botResult applies the bot's answer if it comes from the search that is
// running for the current version. Any other answer is stale: a search
// cancelled by an undo, by the last detach (even if a new search for the
// same version has started since), or by an earlier answer. A bot that
// failed, or answered with an action that is not legal, is a bug in the
// bot; the room then plays the first legal action instead, so the game
// never stalls, and logs it.
func (r *Room) botResult(m msgBotResult) {
	if r.bot.cancel == nil || m.gen != r.bot.gen || m.version != r.version {
		r.dropped++
		r.log.Debug("dropped a stale bot result", "gen", m.gen, "running", r.bot.gen, "for", m.version, "now", r.version)
		return
	}
	r.cancelBot()
	if r.file.Mode != store.ModeAI || r.status() != store.StatusPlaying ||
		r.file.Seats[r.state.ToMove].Kind != store.SeatKindBot {
		r.dropped++
		return
	}
	var key string
	if m.err == nil {
		key = m.action.Key()
	}
	a, ok := legalByKey(&r.state, key)
	if !ok {
		legal := engine.LegalActions(&r.state, nil)
		if len(legal) == 0 {
			return
		}
		r.log.Warn("the bot gave no legal action; playing the first legal one", "key", key, "err", m.err)
		a = legal[0]
	}
	r.commit(a, nil, nil)
}

// botJob is one search, with everything it needs copied out of the room so
// that the search goroutine never touches room state.
type botJob struct {
	bot     Bot
	obs     engine.Observation
	budget  ai.Budget
	gen     uint64
	version int
	pool    *ai.Pool
	clock   Clock
	delay   time.Duration
	inbox   chan<- any
	done    <-chan struct{}
}

// run searches, waits out the rest of the think delay, and posts the
// result. A cancelled search posts nothing once the delay is cut short;
// otherwise the result is posted even if it went stale meanwhile, and the
// room drops it by its generation.
func (j botJob) run(ctx context.Context) {
	start := j.clock.Now()
	var a engine.Action
	var err error
	if j.pool != nil {
		a, err = j.pool.Choose(ctx, j.bot, j.obs, j.budget, nil)
	} else {
		a, err = j.bot.Choose(ctx, j.obs, j.budget, nil)
	}
	if wait := j.delay - j.clock.Now().Sub(start); wait > 0 {
		fired := make(chan struct{})
		stop := j.clock.AfterFunc(wait, func() { close(fired) })
		select {
		case <-fired:
		case <-ctx.Done():
			stop()
			return
		}
	}
	select {
	case j.inbox <- msgBotResult{gen: j.gen, version: j.version, action: a, err: err}:
	case <-j.done:
	}
}
