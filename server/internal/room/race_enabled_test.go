//go:build race

package room_test

// raceEnabled is true when the test binary is built with the race detector
// (`go test -race`), which slows the room actor and any real bot search
// enough that fixed wall-clock waits need a wider margin (see waitLimit in
// helpers_test.go).
const raceEnabled = true
