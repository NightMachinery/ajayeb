package tabletest

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
)

// ClientID names one connection, a tab of one user. Seqs, and their
// deduplication, are per client.
type ClientID string

// SeatClient is the driver's client for a seat; SpectatorClient watches.
func SeatClient(seat int) ClientID { return ClientID("seat-" + strconv.Itoa(seat)) }

// TabClient is a seat's second tab: another client of the user seated
// there. A table that runs scripts with ScriptOptions.Tabs knows it.
func TabClient(seat int) ClientID { return ClientID("seat-" + strconv.Itoa(seat) + "-tab-2") }

// SpectatorClient is the driver's spectator.
const SpectatorClient ClientID = "spectator"

// Submission is the platform's pick request {seq, point, key, confirm}: it
// sets the seat's pick to Key and its confirmed flag to Confirm. Ref is the
// wire point reference the client saw, not the logical id.
type Submission struct {
	Seq     uint64 `json:"seq"`
	Ref     string `json:"point"`
	Key     string `json:"key"`
	Confirm bool   `json:"confirm"`
}

// Outcome is a table's answer to one submission.
type Outcome string

const (
	// Accepted: the pick and confirmed flag were stored.
	Accepted Outcome = "accepted"
	// Resolved: accepted, and this confirmation completed the required set,
	// so the point resolved at once.
	Resolved Outcome = "resolved"
	// Duplicate: a seq not newer than the client's last applied one; only
	// acknowledged, nothing changes.
	Duplicate Outcome = "duplicate"
	// Stale: a seq older than the one that last changed this seat's pick
	// from the same client (reachable once the deduplication forgot it).
	Stale Outcome = "stale"
	// Illegal: the seat is not required, or the key is not offered to it.
	Illegal Outcome = "illegal"
	// Closed: the point reference is not the open point's ("decision-closed").
	Closed Outcome = "decision-closed"
	// Forbidden: the client holds no seat.
	Forbidden Outcome = "forbidden"
	// StorageError: the durable write failed; nothing changed.
	StorageError Outcome = "storage-error"
)

// Snapshot is what one client sees of the open point: its wire reference
// and logical id, the required seats and which have confirmed (public), and,
// for a required seat, its offered keys and its own pick (private).
type Snapshot struct {
	Ref       string   `json:"ref"`
	Point     string   `json:"point"`
	Required  []int    `json:"required"`
	Confirmed []int    `json:"confirmed"`
	Seat      int      `json:"seat"` // -1 for a spectator
	Legal     []string `json:"legal"`
	Pick      string   `json:"pick"`
	Confirm   bool     `json:"confirm"`
	Over      bool     `json:"over"`
}

// Table is a decision-point table under test. Submit returns an error only
// for a transport failure or a crash, never for a refusal, which is an
// Outcome.
type Table interface {
	Snapshot(ctx context.Context, c ClientID) (Snapshot, error)
	Submit(ctx context.Context, c ClientID, s Submission) (Outcome, error)
	// Canonical returns the complete batches so far and the rules hash. Over
	// WebSocket it comes from the end-of-game log download.
	Canonical(ctx context.Context) (Canonical, error)
}

// Factory opens a fresh table of one setup with the given batches already
// resolved, so a point can be replayed under many schedules.
type Factory func(ctx context.Context, prefix []Batch) (Table, error)

// Batch is one complete decision: the logical point id and one key per
// required seat, sorted by seat.
type Batch struct {
	Point string `json:"point"`
	Picks []Pick `json:"picks"`
}

// Pick is one seat's key in a batch.
type Pick struct {
	Seat int    `json:"seat"`
	Key  string `json:"key"`
}

// Canonical is the canonical rules history: complete batches and the rules
// hash. It depends on setup and logical batches only, never on seqs,
// revisions or arrival order.
type Canonical struct {
	Batches []Batch `json:"batches"`
	Hash    string  `json:"hash"`
}

// Bytes is the canonical encoding: one JSON line per batch, then the hash.
func (c Canonical) Bytes() []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, x := range c.Batches {
		_ = enc.Encode(x)
	}
	_ = enc.Encode(map[string]string{"hash": c.Hash})
	return b.Bytes()
}

// Equal reports whether two batches are the same decision.
func (b Batch) Equal(o Batch) bool {
	if b.Point != o.Point || len(b.Picks) != len(o.Picks) {
		return false
	}
	for i := range b.Picks {
		if b.Picks[i] != o.Picks[i] {
			return false
		}
	}
	return true
}
