//go:build race

package ai_test

// raceEnabled is true when the test binary is built with the race detector
// (`go test -race`), which slows the search enough that cadence-sensitive
// assertions (how many progress reports land in a given wall-clock budget)
// must be relaxed to just "at least one intermediate report plus the final
// one" instead of "about every 100ms".
const raceEnabled = true
