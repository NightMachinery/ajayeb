// Package store persists rooms on disk, one JSON file per room at
// <data>/rooms/<code>.json. A file holds a versioned header (schema, rules
// version, room code, mode, seats, options, and the seed as a decimal
// string), the action log as canonical keys, the last seq seen per client,
// the status and the result.
//
// # Rules
//
//   - Writes are atomic: a temporary file in the same directory is written,
//     synced and renamed over the old file. A crash leaves either the old or
//     the new version, never a torn file.
//   - Rooms are restored by replaying the seed and the actions through the
//     engine and checking the state hash. A major rules-version mismatch or a
//     failed replay marks the room archived; its file is kept for bug reports
//     rather than deleted.
//   - The reaper deletes finished rooms after 7 days and idle rooms after 14
//     days. It takes an injected clock so expiry is testable.
//   - Only this package reads or writes the rooms directory. Files never
//     contain tokens; seats are identified by derived user ids.
package store
