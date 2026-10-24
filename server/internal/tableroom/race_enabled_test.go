//go:build race

package tableroom

// RaceEnabled is true when the test binary is built with the race detector
// (`go test -race`). The race build plays the long sweeps on a sample (see
// SweepSeats and each test that says so); the full sweeps run without it,
// in verify-quick and CI's go job.
const RaceEnabled = true
