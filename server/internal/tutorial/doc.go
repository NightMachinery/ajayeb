// Package tutorial holds the guided first game: its script, a fixed list of
// action keys for the opening of a game on a fixed seed with the lessons the
// player sees before each of their scripted moves, and the derivation of the
// tutorial object of a state frame (wire.TutorialState) from that script and
// a room's action log.
//
// The package is pure: no clock, no randomness, no I/O and no room state. It
// reads the engine state it is given and the script's literal data, so the
// tutorial object is a function of the room file alone, and a reload, a
// reconnect or an undo shows the same lesson. The engine is unchanged by the
// tutorial: a tutorial game is an ordinary game whose seed and options the
// script fixes.
//
// # Terms
//
//   - Entry: one item of the script, a seat and an action key, plus, on the
//     player's entries, the info lessons shown before the move and the
//     action lesson that asks for it.
//   - Guided turn: one of the player's entries.
//   - On script: the log's keys are a prefix of the script's keys.
//   - Left the path: the log stops matching the script before the script's
//     end (the player made another move).
//
// # Changing a script
//
// A script is frozen once shipped: rooms on disk name it and replay their
// lessons from it. A changed script gets a new name (and the old one stays,
// or its rooms simply play on as normal games: an unknown name means no
// tutorial object). script_test.go replays every script from its seed and
// checks each lesson's precondition, so a rules change that breaks a script
// fails a test. search_test.go (run with AJAYEB_TUTORIAL_SEARCH=1) finds
// seeds and builds candidate scripts for a new one.
package tutorial
