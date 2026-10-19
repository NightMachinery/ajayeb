package store

import (
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// Schema is the version of the room file format below. A change to File's
// shape that an older reader could misinterpret (not just a new optional
// field) bumps it. Load and PeekHeader refuse a file of any other schema
// (ErrSchema) and never rewrite it.
const Schema = 1

// Mode is the kind of room: who, or what, holds the two seats.
type Mode string

// The modes.
const (
	ModeOnline  Mode = "online"  // two separate people, over the network
	ModeHotseat Mode = "hotseat" // one person, both seats, one device
	ModeAI      Mode = "ai"      // one person against a bot
)

// SeatKind says who holds a seat.
type SeatKind string

// The seat kinds.
const (
	SeatKindHuman SeatKind = "human"
	SeatKindBot   SeatKind = "bot"
	SeatKindOpen  SeatKind = "open" // an online room's second seat, not yet claimed
)

// BotLevel is the strength of a bot seat.
type BotLevel string

// The bot levels.
const (
	BotEasy   BotLevel = "easy"
	BotNormal BotLevel = "normal"
	BotHard   BotLevel = "hard"
)

// Status is a room's lifecycle state.
type Status string

// The statuses.
const (
	StatusLobby    Status = "lobby"    // online room, waiting for its second seat
	StatusPlaying  Status = "playing"  // the game is running
	StatusFinished Status = "finished" // the game ended normally
	// StatusArchived marks a version mismatch or failed legal replay. Hash
	// differences within the same rules version are refreshed instead (D28).
	StatusArchived Status = "archived"
)

// Seat is one seat of a room: who holds it, and, once it is taken, how.
type Seat struct {
	Kind SeatKind `json:"kind"`
	// UserID is the ident-derived id of the human holding this seat. Empty
	// for SeatKindOpen and SeatKindBot.
	UserID string `json:"userId,omitempty"`
	// Name is the display name chosen at creation or join. The server never
	// trusts it as anything but text to show (see package wire).
	Name string `json:"name,omitempty"`
	// Bot is the level of a bot seat. Empty for SeatKindHuman and
	// SeatKindOpen.
	Bot BotLevel `json:"bot,omitempty"`
	// NameNo is the number that tells this seat's name apart from an equal
	// name already in the room (package room assigns it): 0 for none, else
	// 2, 3 and so on. Stored so that it never changes when people leave and
	// come back. 0 in a file written before the field existed.
	NameNo int `json:"nameNo,omitempty"`
	// Epoch is the seat's link epoch, part of every claim derived for it
	// (package ident): a link reset raises it, which revokes the claims made
	// before. 0 until the first reset, and in a file written before the
	// field existed.
	Epoch int `json:"epoch,omitempty"`
}

// File is one room, exactly as <data>/rooms/<code>.json holds it: enough to
// rebuild the game (seed, options, optional starting scenario, and the
// action log, which together describe it completely, as engine.Log does) and
// enough to run the room around it (code, mode, seats, status, per-client
// idempotency and the result once there is one).
//
// Load and Save are the only supported ways to read or write one: Save
// recomputes Hash from a fresh replay before writing, and Load replays and
// refreshes Hash after a legal replay, so a *File that came from either of
// them is always known to replay to the state it claims. A File built by
// hand (New, or a literal) carries no such guarantee until it has been
// through Save.
type File struct {
	Schema int `json:"schema"`
	// Rules is the engine.RulesVersion the game was created under. Load
	// archives the room, without attempting a replay, if this engine's
	// RulesVersion has moved on.
	Rules int    `json:"rules"`
	Code  string `json:"code"`
	Mode  Mode   `json:"mode"`

	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`

	Seats   [2]Seat        `json:"seats"`
	Options engine.Options `json:"options"`
	// Seed is a decimal string (encoding/json's `,string` option) so that a
	// JavaScript reader keeps all 64 bits.
	Seed uint64 `json:"seed,string"`
	// Scenario is the name of a named scenario (engine.NamedScenario) the
	// game started from, for rooms the dev endpoint creates; empty for an
	// ordinary game, which starts from engine.NewGame.
	Scenario string `json:"scenario,omitempty"`
	// Actions is the action log, in canonical key form (Action.Key), in the
	// order they were applied.
	Actions []string `json:"actions"`
	// Hash is engine.HashString of the state reached by replaying Seed (and
	// Scenario, and Options) plus every entry of Actions, in order. Save
	// computes it; Load refreshes it after a complete legal replay (D28).
	Hash string `json:"hash,omitempty"`
	// Version is the room's version (see package room): it goes up with
	// every accepted action and every undo, and never goes down, so it is
	// at least len(Actions). It is stored so that a room reloaded after an
	// undo does not reuse a version that already named another position. A
	// file written before the field existed reads as len(Actions).
	Version int `json:"version"`

	// LastSeq is the last applied seq of each client, keyed by user id and
	// clientId as "<userId>:<clientId>" (package room's seqKey), for
	// act/undo idempotency across a reconnect. Seqs are the wire's (JSON
	// numbers, never negative). The room keeps only its most recently used
	// clients here (a bounded LRU), so the map stays small.
	LastSeq map[string]int `json:"lastSeq"`

	Status Status `json:"status"`
	// ArchiveReason and ArchivedAt describe a durable archive. Legacy archives
	// may have neither; they still stay archived.
	ArchiveReason ArchiveReason `json:"archiveReason,omitempty"`
	ArchivedAt    *time.Time    `json:"archivedAt,omitempty"`
	// Result is the final score sheet, set once Status is StatusFinished
	// (or an StatusArchived room that had finished before it was archived).
	Result *engine.Result `json:"result,omitempty"`
	// Rematch is the code of the room created as this finished room's
	// rematch, once one exists, so that every later rematch request (also
	// after the room was evicted and reloaded) leads to that same room.
	Rematch string `json:"rematch,omitempty"`

	// Admin is the user id of the game's admin, who may change Hints: the
	// user who created the room. Empty in a file written before the field
	// existed, and in a room made with open seats (a development room)
	// until its first player claims a seat; package room then decides.
	Admin string `json:"admin,omitempty"`
	// Hints are the game's hint settings (docs/dev/decisions.md D23). They
	// are not part of the game (the replay and Hash ignore them). nil in a
	// file written before the field existed: package room then uses the
	// defaults of the mode.
	Hints *view.HintSettings `json:"hints,omitempty"`
	// RoomLog is the room-level part of the game log, oldest first: each
	// change of the hint settings. Package room bounds it.
	RoomLog []LogEntry `json:"roomLog,omitempty"`
	// NoUndo is set when the game was created with undo off (a hot-seat or
	// bot room; an online room has no undo anyway). It is fixed at creation
	// and never changes. False, and so absent, in a file written before the
	// field existed: such a room allows undo, as every room did then. Like
	// Hints it is not part of the game (the replay and Hash ignore it).
	NoUndo bool `json:"noUndo,omitempty"`
	// Tutorial is the name of the tutorial script (package tutorial) of a
	// tutorial room, empty for every other room. It is fixed at creation.
	// Like Hints and NoUndo it is not part of the game: the seed and options
	// above already describe it, and the replay and Hash ignore the name. A
	// name this build does not know leaves a normal game.
	Tutorial string `json:"tutorial,omitempty"`
}

// LogEntry is one room-level event of the game log, as the room file keeps
// it (package room turns it into wire.RoomLogEntry).
type LogEntry struct {
	// Kind is what happened: "hints" is a change of the hint settings,
	// "link" the reset of a seat's device link.
	Kind string `json:"kind"`
	// At is the game's ply when it happened.
	At int `json:"at"`
	// Seat is the seat of the player who made the change, or -1 when one
	// person holds both seats (a hot-seat room).
	Seat int `json:"seat"`
	// On and Off are the hint families switched on and off (kind hints).
	On  []string `json:"on"`
	Off []string `json:"off"`
	// Target is the seat whose link was reset (kind link). A pointer, so a
	// hints entry, and a file written before the field existed, has none.
	Target *int `json:"target,omitempty"`
}

// New returns a fresh lobby/playing File: schema and rules at their current
// values, Created and Updated at now, version 0, an empty action log and
// client-seq map (never nil, so they encode as [] and {} rather than null),
// and no result yet. Save it once to give it a correct Hash before anything
// reads it back with Load.
func New(code string, mode Mode, opts engine.Options, seed uint64, now time.Time) *File {
	return &File{
		Schema:  Schema,
		Rules:   engine.RulesVersion,
		Code:    code,
		Mode:    mode,
		Created: now,
		Updated: now,
		Options: opts,
		Seed:    seed,
		Actions: []string{},
		LastSeq: map[string]int{},
		Status:  StatusLobby,
	}
}
