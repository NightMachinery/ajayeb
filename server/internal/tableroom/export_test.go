package tableroom

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
)

// The external test package (ws_test.go) drives tables over the real
// WebSocket with the same snapshot, reference and leak-oracle helpers as
// the in-process tests.
var (
	SnapshotOf  = snapshotOf
	RefString   = refString
	ParseRef    = parseRef
	FrameLeaks  = frameLeaks
	ViewerLeaks = viewerLeaks
	TruthFrom   = truthFrom
	FrameConfig = frameConfig
	// PendingCanaries runs the pending-extras canaries on a raw frame.
	PendingCanaries = pendingCanaries
)

// FrameViewer is who received a frame, for FrameLeaks.
type FrameViewer = frameViewer

// RecordTruth is the oracle's ground truth of the game in a table
// record's bytes, under chosen seating: the socket tests' source for a
// game whose log stays hidden after the end, or whose seed a lobby's Start
// chose.
func RecordTruth(t testing.TB, data []byte) *leakcheck.Truth {
	t.Helper()
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Setup.RandomSeating {
		t.Fatal("RecordTruth: the table seats at random")
	}
	return truthFrom(t, canonicalOf(r), len(r.Seats), r.Setup.Seed, r.Setup.HideAtEnd)
}
