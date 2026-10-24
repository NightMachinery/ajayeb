package tableroom

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// kickBots starts the table's one bot job when it can run: no job is
// queued or active, a human participant is attached, and the open point
// requires a bot seat that has not confirmed. The job picks for that one
// seat; its result starts the next. The job holds a pool permit only while
// it searches, under the smallest attached human participant's address,
// as Duel's bots do.
func (a *actor) kickBots() {
	if a.botRunning || a.botBusy || a.ctx.Err() != nil || !a.humanAttached() {
		return
	}
	r := &a.st.rec
	g := &a.st.game
	p := g.Point()
	if pointOver(p) {
		return
	}
	seat := -1
	for _, s := range p.Required.Seats() {
		if r.Seats[s].Kind != SeatBot {
			continue
		}
		if ps := pendingOf(r, int(s)); ps != nil {
			if !ps.Confirmed {
				a.transact(nil, "bot-confirm", int(s), r.clone(), ps.Key, true, nil, 0)
				return
			}
			continue
		}
		seat = int(s)
		break
	}

	if seat < 0 {
		return
	}
	i := slices.IndexFunc(r.Work, func(w WorkRow) bool { return w.Seat == seat })
	if i < 0 {
		return // restore and openPoint guarantee a row; never search without one
	}
	row := r.Work[i]
	obs := g.Observe(seatNum(seat))
	input := botInput{row: row, observation: obs, offered: g.Legal(seatNum(seat)), entropy: botTape(r.Setup.Seed, p.ID, seat, obs.Fingerprint(), row.BotPreset)}
	a.botGen++
	gen := a.botGen
	ctx, cancel := context.WithCancel(a.ctx)
	a.botBusy, a.botCancel = true, cancel
	a.botRunning, a.botRunID, a.botSeat = true, gen, seat
	a.botPoint = p.ID
	key := a.poolKey()
	pool, hook, observer := a.t.cfg.Pool, a.t.cfg.BotWork, a.t.cfg.BotFinished
	watchdog := a.t.cfg.BotWatchdog
	if watchdog <= 0 {
		watchdog = 30 * time.Second
	}
	a.botWG.Add(1)
	go func() {
		defer a.botWG.Done()
		res := msgBotResult{owner: a, gen: gen, point: p.ID, seat: seat}
		report := BotReport{Seat: seat, Point: p.ID}
		waitStart := time.Now()
		var err error
		if pool != nil {
			err = pool.AcquireFor(ctx, key)
		}
		report.PoolWait = time.Since(waitStart)
		if err == nil {
			// Cancellation can race admission. Release exactly the permit acquired,
			// including a grant that won a race with the context's cancellation.
			func() {
				if pool != nil {
					defer pool.ReleaseFor(key)
				}
				active := time.Now()
				defer func() { report.Active = time.Since(active) }()
				search, cancelSearch := context.WithTimeout(ctx, watchdog)
				defer cancelSearch()
				if hook != nil {
					err = hook(search, seat, p.ID)
				}
				var decision bot.Decision
				if err == nil {
					decision, err = runBot(search, input)
				}
				if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
					decision, err = bot.Fallback(ctx, input.observation, input.offered)
					if err == nil && decision.Complete {
						res.fallback = "watchdog"
					}
				}
				if err == nil && (!decision.Complete || decision.Key == "") {
					err = bot.ErrInput
				}
				report.Counters, report.Fallback = decision.Counters, res.fallback
				res.key = decision.Key
			}()
		}
		if err == nil {
			err = ctx.Err()
		}
		report.Cancelled = err != nil
		if observer != nil {
			observer(report)
		}
		res.err = err
		// Even cancelled work acknowledges physical completion. A new job waits
		// for this message, so cancellation never overlaps two workers in a room.
		select {
		case a.inbox <- res:
		case <-a.ctx.Done():
		}
	}()

}

// stopBots cancels the bot job, if any; a result it still sends is stale
// by its generation.
func (a *actor) stopBots() {
	if a.botBusy {
		a.botCancel()
		a.botBusy = false
		a.botGen++
	}
}

// botResult accepts a bot's pick, confirmed, after checking that it is for
// the current job, point and controller, the seat has not confirmed, and
// the key is offered.
func (a *actor) botResult(res msgBotResult) {
	if res.owner != a || !a.botRunning || res.gen != a.botRunID || res.point != a.botPoint || res.seat != a.botSeat {
		return
	}
	a.botRunning = false
	if res.gen != a.botGen || !a.botBusy {
		a.kickBots()
		return
	}
	a.botBusy = false
	a.botCancel()
	if res.err != nil {
		a.t.log.Warn("table bot stopped", "room", a.code, "seat", res.seat)
		return
	}
	r := &a.st.rec
	p := a.st.game.Point()
	if res.seat < 0 || res.seat >= len(r.Seats) {
		a.kickBots()
		return
	}
	ps := pendingOf(r, res.seat)
	if pointOver(p) || p.ID != res.point || !p.Required.Has(seatNum(res.seat)) || r.Seats[res.seat].Kind != SeatBot || ps != nil || !slices.Contains(legalKeys(&a.st.game, res.seat), res.key) {
		a.kickBots()
		return
	}
	cand := r.clone()
	if res.fallback != "" {
		cand.BotFallbacks = append(cand.BotFallbacks, BotFallback{At: len(r.Batches), Point: p.ID, Seat: res.seat, Reason: res.fallback})
		if len(cand.BotFallbacks) > maxRoomLog {
			cand.BotFallbacks = slices.Clone(cand.BotFallbacks[len(cand.BotFallbacks)-maxRoomLog:])
		}
	}
	a.transact(nil, "bot-pick", res.seat, cand, res.key, true, nil, 0)
}

// poolKey is the smallest attached participant's address.
func (a *actor) poolKey() string {
	key, found := "", false
	for _, c := range a.conns {
		if a.participant(a.userOf(c)) && (!found || c.Addr < key) {
			key, found = c.Addr, true
		}
	}
	return key
}

// policyRoot derives the bots' entropy root from the setup root under its
// own domain label, so the policy never sees the setup root itself.
func policyRoot(seed uint64) [32]byte {
	b := append([]byte("ajayeb/haftgane/bot-policy-root/v1\x00"), make([]byte, 8)...)
	binary.BigEndian.PutUint64(b[len(b)-8:], seed)
	return sha256.Sum256(b)
}

// botTape is a bot's random tape: the policy root, game, policy version,
// logical point, seat and permitted-observation hash. Never the state or
// rules hash, the wire point reference, pending keys, the task
// generation, the clock or arrival order.
func botTape(seed uint64, pointID string, seat int, obs string, preset BotPreset) uint64 {
	root := policyRoot(seed)
	b := append([]byte{}, root[:]...)
	for _, part := range []string{Game, preset.Policy, strconv.Itoa(preset.PolicyVersion), strconv.Itoa(preset.Preset), preset.Level, preset.Algorithm, pointID, strconv.Itoa(seat), obs} {
		b = append(append(b, part...), 0)
	}
	sum := sha256.Sum256(b)
	return binary.BigEndian.Uint64(sum[:8])
}

// botInput is the complete policy boundary: no Game, State, setup seed,
// pending key, transport revision, worker generation or wall clock.
type botInput struct {
	row         WorkRow
	observation rules.Observation
	offered     []rules.Choice
	entropy     uint64
}

func runBot(ctx context.Context, input botInput) (bot.Decision, error) {
	level, err := bot.ParseLevel(input.row.Algorithm)
	if err != nil {
		return bot.Decision{}, err
	}
	return bot.Choose(ctx, level, input.observation, input.offered, input.entropy, input.row.work())
}

// BotReport contains only safe diagnostics. Counters describe completed policy
// work, not elapsed-time strength; active time includes the cancellation hook.
type BotReport struct {
	Seat      int
	Point     string
	Counters  bot.Counters
	PoolWait  time.Duration
	Active    time.Duration
	Fallback  string
	Cancelled bool
}
