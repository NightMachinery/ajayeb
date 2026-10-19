// Package store persists rooms on disk, one JSON file per room at
// <data>/rooms/<code>.json, or <data>/rooms-archive/<code>.json once
// archived. A file holds a versioned header (schema, rules
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
//   - Within the same explicit RulesVersion, seed/options/canonical keys
//     are authoritative. Replay selects engine-generated legal actions,
//     with the room's explicit legal resignation exception. A differing hash
//     is atomically refreshed and logged without changing Updated or metadata.
//     A rules-version mismatch or failed replay archives durably with a safe
//     reason and timestamp and stamps Updated for archive retention. Existing
//     archives stay archived. Markers are saved before a same-filesystem
//     link/unlink move to rooms-archive, without overwriting another archive.
//     Interrupted moves are retryable; legacy archives migrate byte-for-byte.
//     Failed writes are errors, never successful loads. Per-code locks protect
//     restores, saves and reaping without blocking unrelated rooms.
//   - Incompatible setup, RNG, catalog and rules changes require a RulesVersion
//     bump. Representation/derived-field changes alone do not (decision D28).
//     Replay downloads keep strict hash validation; hashes do not authenticate
//     data and cannot replace version discipline or backups.
//   - CheckDataDir is read-only: no mkdir, identity initialization, hash
//     refresh, archive or listener. It scans both room directories and
//     counts existing archives only as archived, never as live risk.
//     It reports live archive risk separately
//     from schema/read failures so deployment can refuse before stopping.
//   - A file of another schema is refused and never rewritten, so a file
//     from a newer server survives a rollback intact.
//   - The startup/hourly reaper deletes finished rooms after 7 days and
//     idle rooms after 14 days. Archives expire 30 days after ArchivedAt
//     (Updated for legacy files), with caps of 1000 files or 64 MiB,
//     oldest first. Caps can be exceeded between sweeps. Malformed and
//     unknown-schema records are preserved outside the managed caps.
//     The store and reaper take injected clocks so expiry is testable.
//   - Only this package reads or writes the room directories. Files never
//     contain tokens; seats are identified by derived user ids.
package store
