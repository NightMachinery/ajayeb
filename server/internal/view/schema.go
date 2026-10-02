package view

// SchemaVersion is the version of the view DTO. A View carries it in its
// schema field, and an incompatible change to any type in this package bumps
// it, so a client can refuse a view it does not understand.
const SchemaVersion = 1

// Pair is the TypeScript type of a pair indexed by seat: a Go array of two
// (both players, both seats' costs, both scores) is tagged
// tstype:"Pair<T>", so the client knows it has exactly two entries. (A
// tstype tag cannot spell the tuple itself: tygo reads a comma in a tag as
// the start of its options.)
//
//tygo:emit
var _ = `export type Pair<T> = [T, T];`
