// Package bot reserves the observation-only policy boundary. Policies land in
// the bot milestone; this slice establishes their pure input dependencies.
// A policy receives rules.Observation and the real root offer, never a Game.
package bot
