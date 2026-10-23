package tableroom

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"strconv"
)

// kickBots starts the table's one bot job when it can run: no job is
// queued or active, a human participant is attached, and the open point
// requires a bot seat that has not confirmed. The job picks for that one
// seat; its result starts the next. The job holds a pool permit only while
// it searches, under the smallest attached human participant's address,
// as Duel's bots do.
func (a *actor) kickBots() {
	if a.botBusy || a.ctx.Err() != nil || !a.humanAttached() {
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
		if ps := pendingOf(r, int(s)); r.Seats[s].Kind == SeatBot && (ps == nil || !ps.Confirmed) {
			seat = int(s)
			break
		}
	}
	if seat < 0 {
		return
	}
	i := slices.IndexFunc(r.Work, func(w WorkRow) bool { return w.Seat == seat })
	if i < 0 {
		return // restore and openPoint guarantee a row; never search without one
	}
	row := r.Work[i]
	legal := legalKeys(g, seat)
	tape := botTape(r.Setup.Seed, p.ID, seat, observationHash(g, seat))
	a.botGen++
	gen := a.botGen
	ctx, cancel := context.WithCancel(a.ctx)
	a.botBusy, a.botCancel = true, cancel
	key := a.poolKey()
	pool, work := a.t.cfg.Pool, a.t.cfg.BotWork
	a.botWG.Add(1)
	go func() {
		defer a.botWG.Done()
		res := msgBotResult{gen: gen, point: p.ID, seat: seat}
		var err error
		if pool != nil {
			err = pool.AcquireFor(ctx, key)
		}
		if err == nil {
			if work != nil {
				err = work(ctx, seat, p.ID)
			}
			res.key = choose(legal, tape, row.Allowance)
			if pool != nil {
				pool.ReleaseFor(key)
			}
		}
		if err == nil {
			err = ctx.Err()
		}
		res.err = err
		select {
		case a.inbox <- res:
		case <-ctx.Done():
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
	if res.gen != a.botGen || !a.botBusy {
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
	ps := pendingOf(r, res.seat)
	if pointOver(p) || p.ID != res.point || r.Seats[res.seat].Kind != SeatBot || (ps != nil && ps.Confirmed) ||
		!slices.Contains(legalKeys(&a.st.game, res.seat), res.key) {
		a.kickBots()
		return
	}
	// A storage failure launches no successor; the next change retries.
	a.transact(nil, "bot-pick", res.seat, r.clone(), res.key, true, nil, 0)
}

// poolKey is the smallest attached human participant's address.
func (a *actor) poolKey() string {
	key, found := "", false
	for _, c := range a.conns {
		if a.seatOf(c.UserID) >= 0 && (!found || c.Addr < key) {
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
func botTape(seed uint64, pointID string, seat int, obs string) uint64 {
	root := policyRoot(seed)
	b := append([]byte{}, root[:]...)
	for _, part := range []string{Game, policyVersion, pointID, strconv.Itoa(seat), obs} {
		b = append(append(b, part...), 0)
	}
	sum := sha256.Sum256(b)
	return binary.BigEndian.Uint64(sum[:8])
}

// choose is the fake policy: a deterministic choice among the offered keys
// from the tape, one draw per unit of allowance. M17 replaces it.
func choose(legal []string, tape uint64, allowance int) string {
	if len(legal) == 0 {
		return ""
	}
	x := tape
	for range max(allowance, 1) {
		x ^= x >> 33
		x *= 0xff51afd7ed558ccd
		x ^= x >> 33
	}
	return legal[x%uint64(len(legal))]
}
