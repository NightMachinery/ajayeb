package tableroom

// The external test package (ws_test.go) drives tables over the real
// WebSocket with the same snapshot, reference and leak-oracle helpers as
// the in-process tests.
var (
	SnapshotOf  = snapshotOf
	RefString   = refString
	ParseRef    = parseRef
	LeakDocs    = leakDocs
	TruthFrom   = truthFrom
	FrameConfig = frameConfig
)
