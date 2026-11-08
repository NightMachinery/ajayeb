//go:build race

package rules

// RaceEnabled samples long, single-goroutine rules sweeps under go test -race.
// Their full coverage runs without race in verify-quick and nightly CI.
const RaceEnabled = true
