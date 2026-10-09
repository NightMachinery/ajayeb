package room

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log/slog"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// Defaults for the zero fields of Config.
const (
	// DefaultBotDelay is the think delay a production server passes as
	// Config.BotDelay: a bot never answers faster than this, so its move
	// reads as a move. (Config.BotDelay itself is used as given; zero means
	// no delay, which is what tests and end-to-end runs want.)
	DefaultBotDelay = 400 * time.Millisecond
	// DefaultIdleTimeout is how long a room with no connection stays in
	// memory before it is evicted. It stays on disk and is reloaded on the
	// next join.
	DefaultIdleTimeout = 30 * time.Minute
	// DefaultMaxRooms bounds the rooms in memory at once.
	DefaultMaxRooms = 1024
	// DefaultMaxClients bounds the connections of one room: a new spectator
	// is admitted while the room has fewer, a seated user while that user
	// has fewer there (see Config.MaxClients).
	DefaultMaxClients = 32
	// DefaultSeqClients bounds the client ids whose last seq a room
	// remembers (an LRU).
	DefaultSeqClients = 32
	// DefaultInboxSize is the capacity of a room's inbox channel.
	DefaultInboxSize = 64
)

// Storage is what the rooms need from package store; *store.Store
// satisfies it. Save must replay the file and set its Hash, and Load must
// return the replayed state, as store does.
type Storage interface {
	NewCode() (string, error)
	Save(f *store.File) error
	Load(code string) (*store.File, engine.State, error)
}

// Bot is what a room needs from a bot. Every bot of package ai satisfies
// it. The room gives a bot only an engine Observation of its own seat,
// never the state.
type Bot interface {
	Choose(ctx context.Context, obs engine.Observation, b ai.Budget, onProgress func(ai.Progress)) (engine.Action, error)
}

// BotFactory makes the bot for one decision of a bot seat. seed is derived
// from the room's seed and the game's ply (see bot.go), so a game against a
// bot replays the same way, and an undo cannot reroll the bot's answer.
type BotFactory func(level store.BotLevel, seed uint64) Bot

// Config holds a Manager's dependencies and limits. Store is required; a
// zero field elsewhere takes the default listed with it.
type Config struct {
	// Store persists the rooms. Required.
	Store Storage
	// Clock is the source of time. Default SystemClock.
	Clock Clock
	// Logger receives the rooms' logs. Default: discard. Tokens never reach
	// it: rooms only ever see user ids.
	Logger *slog.Logger

	// NewBot makes bots. Default: DefaultBotFactory.
	NewBot BotFactory
	// BotBudget gives the search budget of a bot level. Default: the zero
	// ai.Budget (each bot's own defaults).
	BotBudget func(level store.BotLevel) ai.Budget
	// BotPool, when set, bounds the searches running at once across every
	// room (a semaphore). Default: none.
	BotPool *ai.Pool
	// BotDelay is the bot's minimum think time, used as given: zero means
	// none. Production passes DefaultBotDelay.
	BotDelay time.Duration

	// IdleTimeout is how long a room with no connection stays in memory.
	// Default DefaultIdleTimeout.
	IdleTimeout time.Duration
	// MaxRooms bounds the rooms in memory. Default DefaultMaxRooms.
	MaxRooms int
	// MaxClients bounds the connections of one room: a new spectator is
	// refused once the room has MaxClients connections, and a user who
	// holds a seat once that user has MaxClients connections there, so
	// spectators cannot lock a player out and a room holds at most three
	// times MaxClients connections. Default DefaultMaxClients.
	MaxClients int
	// SeqClients bounds the client ids whose last applied seq a room
	// remembers, in memory and in its file. Default DefaultSeqClients.
	SeqClients int
	// InboxSize is each room's inbox capacity. Default DefaultInboxSize.
	InboxSize int

	// Seed returns the seed of a new game. Default: crypto/rand.
	Seed func() uint64
}

// withDefaults returns cfg with every zero field replaced by its default.
func (cfg Config) withDefaults() Config {
	if cfg.Clock == nil {
		cfg.Clock = SystemClock{}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.NewBot == nil {
		cfg.NewBot = DefaultBotFactory
	}
	if cfg.BotBudget == nil {
		cfg.BotBudget = func(store.BotLevel) ai.Budget { return ai.Budget{} }
	}
	if cfg.BotDelay < 0 {
		cfg.BotDelay = 0
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	if cfg.MaxRooms <= 0 {
		cfg.MaxRooms = DefaultMaxRooms
	}
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = DefaultMaxClients
	}
	if cfg.SeqClients <= 0 {
		cfg.SeqClients = DefaultSeqClients
	}
	if cfg.InboxSize <= 0 {
		cfg.InboxSize = DefaultInboxSize
	}
	if cfg.Seed == nil {
		cfg.Seed = cryptoSeed
	}
	return cfg
}

// cryptoSeed returns 64 random bits from crypto/rand.
func cryptoSeed() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:]) // never fails: it crashes the program if the system source does
	return binary.LittleEndian.Uint64(b[:])
}

// DefaultBotFactory maps the room bot levels to package ai's: easy plays
// ai.Easy, normal ai.Medium and hard ai.Hard. An unknown level plays
// ai.Medium.
func DefaultBotFactory(level store.BotLevel, seed uint64) Bot {
	return ai.New(aiLevel(level), seed)
}

// aiLevel maps a room bot level to package ai's level.
func aiLevel(level store.BotLevel) ai.Level {
	switch level {
	case store.BotEasy:
		return ai.Easy
	case store.BotHard:
		return ai.Hard
	default:
		return ai.Medium
	}
}
