package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// gameSummary is what --games prints, one line per finished game.
type gameSummary struct {
	mode    string
	code    string
	seed    uint64
	plies   int
	victory string
	winner  string
	shared  bool
	elapsed time.Duration
}

func (s gameSummary) String() string {
	return fmt.Sprintf("mode=%s code=%s pick-seed=%d plies=%d victory=%s winner=%s shared=%t duration=%s",
		s.mode, s.code, s.seed, s.plies, s.victory, s.winner, s.shared, s.elapsed.Round(time.Millisecond))
}

// pickChoice returns the key of a uniformly random legal choice.
func pickChoice(rng *rand.Rand, choices []view.Choice) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no legal choices")
	}
	return choices[rng.IntN(len(choices))].Key, nil
}

// summarize turns a finished game's final state into a gameSummary.
func summarize(mode, code string, seed uint64, final wire.State, start time.Time) (gameSummary, error) {
	if final.View.Result == nil {
		return gameSummary{}, fmt.Errorf("game ended without a result")
	}
	r := final.View.Result
	winner := "none"
	if r.Winner != nil {
		winner = fmt.Sprintf("%d", *r.Winner)
	}
	return gameSummary{
		mode: mode, code: code, seed: seed,
		plies: final.View.Ply, victory: string(r.Victory), winner: winner, shared: r.Shared,
		elapsed: time.Since(start),
	}, nil
}

// fakeAddr derives a synthetic client address for connection j of game i, so
// a server started with --trust-proxy naming this process's own address
// sees each connection as coming from a distinct peer: internal/ws's
// per-address socket cap (16 by default) is meant for one browser's tabs,
// not for a driver's --concurrency, and is otherwise the only thing
// standing between --games N and N genuinely concurrent sockets.
func fakeAddr(i, j int) string {
	n := i*2 + j
	return fmt.Sprintf("10.%d.%d.%d", (n>>16)&0xff, (n>>8)&0xff, n&0xff)
}

// playOnline plays both seats of an online room itself, over two distinct
// WebSocket connections (two distinct identity tokens, exactly as two real
// browsers would be), the way the prod smoke test exercises the full
// online path against a single running server.
func playOnline(ctx context.Context, wsURL string, seed uint64, gameIndex int) (gameSummary, error) {
	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	a, err := dial(ctx, wsURL, "drive-a", fakeAddr(gameIndex, 0))
	if err != nil {
		return gameSummary{}, err
	}
	defer a.close()
	if _, err := a.create(ctx, wire.ModeOnline, nil, wire.SeatRandom); err != nil {
		return gameSummary{}, err
	}
	code := a.code

	b, err := dial(ctx, wsURL, "drive-b", fakeAddr(gameIndex, 1))
	if err != nil {
		return gameSummary{}, err
	}
	defer b.close()
	if _, err := b.join(ctx, code); err != nil {
		return gameSummary{}, err
	}

	conns := []*client{a, b}
	if err := syncTo(ctx, conns, maxVersion(conns)); err != nil {
		return gameSummary{}, err
	}
	if len(a.latest.You.Seats) != 1 || len(b.latest.You.Seats) != 1 {
		return gameSummary{}, fmt.Errorf("online: want one seat each, got a=%v b=%v", a.latest.You.Seats, b.latest.You.Seats)
	}
	bySeat := map[int]*client{a.latest.You.Seats[0]: a, b.latest.You.Seats[0]: b}
	if len(bySeat) != 2 {
		return gameSummary{}, fmt.Errorf("online: a and b hold the same seat %v", a.latest.You.Seats)
	}

	for a.latest.View.Result == nil {
		toMove := *a.latest.View.ToMove
		actor, ok := bySeat[toMove]
		if !ok {
			return gameSummary{}, fmt.Errorf("online: no connection holds seat %d", toMove)
		}
		key, err := pickChoice(rng, actor.latest.View.Choices)
		if err != nil {
			return gameSummary{}, fmt.Errorf("online seat %d: %w", toMove, err)
		}
		if _, err := actor.act(ctx, key, nil); err != nil {
			return gameSummary{}, err
		}
		if err := syncTo(ctx, conns, maxVersion(conns)); err != nil {
			return gameSummary{}, err
		}
	}
	return summarize("online", code, seed, a.latest, start)
}

// playHotseat plays a full hot-seat game, one connection holding both
// seats.
func playHotseat(ctx context.Context, wsURL string, seed uint64, gameIndex int) (gameSummary, error) {
	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	c, err := dial(ctx, wsURL, "drive-hotseat", fakeAddr(gameIndex, 0))
	if err != nil {
		return gameSummary{}, err
	}
	defer c.close()
	if _, err := c.create(ctx, wire.ModeHotseat, nil, wire.SeatNumber(0)); err != nil {
		return gameSummary{}, err
	}
	code := c.code

	for c.latest.View.Result == nil {
		toMove := *c.latest.View.ToMove
		key, err := pickChoice(rng, c.latest.View.Choices)
		if err != nil {
			return gameSummary{}, fmt.Errorf("hotseat seat %d: %w", toMove, err)
		}
		seat := toMove
		if _, err := c.act(ctx, key, &seat); err != nil {
			return gameSummary{}, err
		}
	}
	return summarize("hotseat", code, seed, c.latest, start)
}

// playAI plays a full game against a bot: the driver holds seat 0, a bot
// of the given level holds seat 1.
func playAI(ctx context.Context, wsURL string, seed uint64, bot wire.BotLevel, gameIndex int) (gameSummary, error) {
	start := time.Now()
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))

	c, err := dial(ctx, wsURL, "drive-ai", fakeAddr(gameIndex, 0))
	if err != nil {
		return gameSummary{}, err
	}
	defer c.close()
	if _, err := c.create(ctx, wire.ModeAI, &bot, wire.SeatNumber(0)); err != nil {
		return gameSummary{}, err
	}
	code := c.code

	for c.latest.View.Result == nil {
		if c.latest.You.CanAct {
			key, err := pickChoice(rng, c.latest.View.Choices)
			if err != nil {
				return gameSummary{}, fmt.Errorf("ai: %w", err)
			}
			if _, err := c.act(ctx, key, nil); err != nil {
				return gameSummary{}, err
			}
			continue
		}
		if _, err := c.untilState(ctx, c.latest.Version+1); err != nil {
			return gameSummary{}, err
		}
	}
	return summarize("ai", code, seed, c.latest, start)
}

// syncTo waits until every connection has seen a state at version v or
// later.
func syncTo(ctx context.Context, conns []*client, v int) error {
	for _, c := range conns {
		if _, err := c.untilState(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

// maxVersion returns the highest Version among the connections' c.latest.
func maxVersion(conns []*client) int {
	v := 0
	for _, c := range conns {
		if c.latest.Version > v {
			v = c.latest.Version
		}
	}
	return v
}

// playGame dispatches to the driver for mode. gameIndex only derives the
// synthetic per-connection address (fakeAddr); it is not the seed.
func playGame(ctx context.Context, wsURL, mode string, seed uint64, bot wire.BotLevel, gameIndex int) (gameSummary, error) {
	switch mode {
	case "online":
		return playOnline(ctx, wsURL, seed, gameIndex)
	case "hotseat":
		return playHotseat(ctx, wsURL, seed, gameIndex)
	case "ai":
		return playAI(ctx, wsURL, seed, bot, gameIndex)
	default:
		return gameSummary{}, fmt.Errorf("unknown mode %q", mode)
	}
}
