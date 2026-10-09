// Package store persists rooms on disk, one JSON file per room at
// <data>/rooms/<code>.json. A file holds a versioned header (schema, rules
// version, room code, mode, seats, options, and the seed as a decimal
// string), the action log as canonical keys, the room version, the last
// applied seq per client, the status, the result and the code of the
// rematch room once there is one.
//
// # Rules
//
//   - Writes are atomic: a temporary file in the same directory is written,
//     synced and renamed over the old file. A crash leaves either the old or
//     the new version, never a torn file.
//   - Rooms are restored by replaying the seed and the actions through the
//     engine and checking the state hash. A rules-version mismatch or a
//     failed replay marks the room archived and stamps its Updated time; its
//     file is kept for bug reports rather than deleted.
//   - A file of another schema is refused and never rewritten, so a file
//     from a newer server survives a rollback intact.
//   - The reaper deletes finished and archived rooms 7 days after their
//     last update and idle rooms after 14 days. The store and the reaper
//     take an injected clock so expiry is testable.
//   - Only this package reads or writes the rooms directory. Files never
//     contain tokens; seats are identified by derived user ids.
package store
