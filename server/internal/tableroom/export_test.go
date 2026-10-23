package tableroom

// The external test package (ws_test.go) drives tables over the real
// WebSocket with the same snapshot, reference and leak-oracle helpers as
// the in-process tests.
var (
	SnapshotOf  = snapshotOf
	RefString   = refString
	ParseRef    = parseRef
	FrameLeaks  = frameLeaks
	TruthFrom   = truthFrom
	FrameConfig = frameConfig
	// PendingCanaries runs the pending-extras canaries on a raw frame.
	PendingCanaries = pendingCanaries
)

// FrameViewer is who received a frame, for FrameLeaks.
type FrameViewer = frameViewer
