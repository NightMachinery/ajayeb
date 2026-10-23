package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	sharedrng "github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
)

// Budget bounds one search. The zero Budget means "the bot's own defaults".
//
// Tests and tournaments set MaxIterations and leave MaxTime zero, which makes
// a search fully deterministic for a given bot seed and observation. A wall
// clock limit (MaxTime, or a deadline on the context) is the only source of
// nondeterminism a bot has.
type Budget struct {
	// MaxIterations caps the work. What an iteration is depends on the bot:
	// for Greedy it is one determinization searched in full, for the tree
	// search one descent and rollout. 0 means the bot's default.
	MaxIterations int
	// MaxTime caps the wall-clock time of the search. 0 means no limit
	// beyond the context's deadline.
	MaxTime time.Duration
	// Workers is the number of parallel search workers. Only the tree search
	// uses it; 0 means one.
	Workers int
}

// Progress describes a running search. A bot sends it to the onProgress
// callback about every progressEvery while it searches, and once more when
// it has finished.
type Progress struct {
	Iterations int           // iterations completed so far
	Elapsed    time.Duration // time since the search started
	Best       engine.Action // the action the bot would play if stopped now
	BestShare  float64       // how sure the bot is of Best, from 0 to 1 (the share of iterations that preferred it)
}

// progressEvery is how often a search reports its progress.
const progressEvery = 100 * time.Millisecond

// checkEvery is how many units of work (evaluations, iterations) a search
// does between two looks at the context, the clock and the progress timer.
const checkEvery = 256

// Bot chooses moves. A bot sees the game only through an Observation of its
// own seat, so it cannot read hidden information. Choose returns one of the
// legal actions of the observed position.
//
// Choose is deterministic for a given bot (level and seed), observation and
// Budget, as long as no wall-clock limit cuts the search short: the bot
// derives its random numbers from its seed and the observation, never from
// the order of earlier calls. The bots in this package hold no mutable
// state, so one bot may serve several games and goroutines at once.
//
// When ctx is done, or MaxTime runs out, the search stops early and Choose
// returns the best action found so far, with a nil error: a cut-short search
// is not a failure. Choose returns an error only when there is nothing to
// choose: the game is over (ErrGameOver) or the observer is not the seat to
// move (ErrNotToMove). A position with a single legal action returns at once,
// without searching and without reporting progress.
//
// onProgress may be nil. It runs on the searching goroutine, so it must be
// quick.
type Bot interface {
	Choose(ctx context.Context, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error)
}

// Errors of Choose.
var (
	ErrGameOver  = errors.New("ai: the game is over")
	ErrNotToMove = errors.New("ai: the observer is not the seat to move")
)

// Level is a bot's strength as the rooms offer it.
type Level uint8

// The levels.
const (
	Random Level = iota // uniform over the legal actions
	Easy                // Greedy values sampled by a softmax, so it errs like a beginner
	Medium              // tree search, 0.4 s on one worker
	Hard                // tree search, 2 s on up to 4 workers
	numLevels
)

var levelNames = [numLevels]string{"random", "easy", "medium", "hard"}

// String returns the level's name, for example "easy".
func (l Level) String() string {
	if l < numLevels {
		return levelNames[l]
	}
	return fmt.Sprintf("level#%d", uint8(l))
}

// MarshalText encodes the level as its name.
func (l Level) MarshalText() ([]byte, error) {
	if l >= numLevels {
		return nil, fmt.Errorf("ai: invalid level %d", uint8(l))
	}
	return []byte(levelNames[l]), nil
}

// UnmarshalText decodes a level name exactly.
func (l *Level) UnmarshalText(b []byte) error {
	v, err := ParseLevel(string(b))
	if err == nil {
		*l = v
	}
	return err
}

// ParseLevel parses a level name: random, easy, medium or hard.
func ParseLevel(name string) (Level, error) {
	for i, n := range levelNames {
		if name == n {
			return Level(i), nil
		}
	}
	if len(name) > 32 {
		name = name[:32] + "..."
	}
	return 0, fmt.Errorf("ai: unknown level %q (want random, easy, medium or hard)", name)
}

// New returns a bot of the given level, seeded with seed. It panics on a
// level that is not one of the four.
//
//   - Random plays uniformly at random.
//   - Easy is Greedy with softmax sampling (EasyTemperature).
//   - Medium is the tree search (mcts.go) with a default budget of 0.4
//     seconds on one worker.
//   - Hard is the tree search with rollouts to the end of the game
//     (HardRollout) and a default budget of 2 seconds on up to 4 workers
//     (no more than GOMAXPROCS).
//
// The default budget is the level's clock budget (Level.Budget). A Budget
// passed to Choose replaces it (see MCTSBot); tournaments pass the level's
// FixedBudget. levels.go holds the presets.
func New(level Level, seed uint64) Bot {
	switch level {
	case Random:
		return NewRandom(seed)
	case Easy:
		return NewGreedy(seed, GreedyConfig{Temperature: EasyTemperature})
	case Medium:
		return NewMCTS(seed, Medium.SearchConfig())
	case Hard:
		return NewMCTS(seed, Hard.SearchConfig())
	}
	panic(fmt.Sprintf("ai: New with invalid level %d", uint8(level)))
}

// choiceSetup is the common start of every Choose: it checks that the
// observer is to move in a running game and lists the legal actions into
// buf.
func choiceSetup(obs *engine.Observation, buf []engine.Action) ([]engine.Action, error) {
	pub := obs.Public()
	if pub.Over() {
		return nil, ErrGameOver
	}
	if pub.ToMove != obs.Seat() {
		return nil, ErrNotToMove
	}
	legal := engine.LegalActions(pub, buf)
	if len(legal) == 0 {
		// LegalActions is empty only at game over, which was handled above.
		return nil, ErrGameOver
	}
	return legal, nil
}

// callRNG returns the random stream of one Choose call: a function of the
// bot's seed and of the observation (its seat included), so equal
// observations give equal streams and replaying a game replays the bot.
func callRNG(seed uint64, obs *engine.Observation) engine.RNG {
	h := engine.Hash(obs.Public())
	return engine.NewRNG2(seed^0x6a09e667f3bcc909, h^uint64(obs.Seat())<<63)
}

// MixSeed derives a well-spread 64-bit seed from a base seed and a tag (for
// example a game index or a seat), with the splitmix64 finalizer. Tools use
// it to give every game and every bot its own seed from one command-line
// seed.
func MixSeed(seed, tag uint64) uint64 { return sharedrng.MixSeed(seed, tag) }

// PairedGame returns the setup of game i of a paired match played from a
// base seed: the deal's seed, the seat of the match's first bot (A), and the
// seeds of the two bots (A, then B). Games 2k and 2k+1 share one deal, with A
// in seat 0 and then in seat 1, so neither bot profits from the deal or from
// moving first. The slow tournaments and ajayeb-sim match both use it, so a
// test's tournament can be rerun from the command line.
func PairedGame(seed uint64, i int) (gameSeed uint64, aSeat engine.Seat, botSeeds [2]uint64) {
	gameSeed = MixSeed(seed, uint64(i/2))
	return gameSeed, engine.Seat(i % 2), [2]uint64{MixSeed(gameSeed, 1), MixSeed(gameSeed, 2)}
}

// MaxGamePlies bounds the length of a game PlayGame plays. A real game ends
// within about 100 actions; the property tests check 200.
const MaxGamePlies = 400

// MoveInfo describes one move of a game played by PlayGame.
type MoveInfo struct {
	Seat    engine.Seat
	Action  engine.Action
	Elapsed time.Duration // how long the bot took to choose
}

// PlayGame plays a game between two bots, indexed by seat, from state s
// (NewGame, or a scenario's NewGameFromScenario), until it is over. Each bot
// chooses from an observation of its own seat with its own budget, and every
// choice goes through engine.Apply, so a bot that returned an illegal action
// fails the game with an error. If log is not nil, each action is recorded in
// it and the final state's hash is sealed into it. onMove, if not nil, is
// called after each move.
//
// It returns the last state reached, also on error. When ctx is done the
// game stops before the next move, with ctx's error.
func PlayGame(ctx context.Context, s engine.State, bots [2]Bot, budgets [2]Budget, log *engine.Log, onMove func(MoveInfo)) (engine.State, error) {
	for !s.Over() {
		if err := ctx.Err(); err != nil {
			return s, err
		}
		if int(s.Ply) >= MaxGamePlies {
			return s, fmt.Errorf("ai: the game is not over after %d actions", MaxGamePlies)
		}
		seat := s.ToMove
		start := time.Now()
		a, err := bots[seat].Choose(ctx, engine.Observe(&s, seat), budgets[seat], nil)
		elapsed := time.Since(start)
		if err != nil {
			return s, fmt.Errorf("ai: seat %d at ply %d: %w", seat, s.Ply, err)
		}
		if log != nil {
			log.Record(&s, a)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			return s, fmt.Errorf("ai: seat %d at ply %d chose %s: %w", seat, s.Ply, a, err)
		}
		if onMove != nil {
			onMove(MoveInfo{Seat: seat, Action: a, Elapsed: elapsed})
		}
	}
	if log != nil {
		log.Seal(&s)
	}
	return s, nil
}
