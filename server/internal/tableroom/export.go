package tableroom

import (
	"errors"
	"io/fs"

	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// LogFormat is the table game log's format id. Duel's stays ajayeb-log/1.
const LogFormat = "ajayeb-log/2"

// Log is a finished table's game log: the game, its rules version, its
// setup, the complete canonical batches and the strict final rules hash.
// It holds no seqs, arrivals, work ledger or transport references. The
// setup holds the seed, which decides every hidden card, so the log is
// served only once the end of the game has revealed them all.
type Log struct {
	Format  string        `json:"format"`
	Game    string        `json:"game"`
	Rules   int           `json:"rules"`
	Setup   SetupRecord   `json:"setup"`
	Batches []BatchRecord `json:"batches"`
	Hash    string        `json:"hash"`
}

// Errors of ReadTableLog.
var (
	// ErrNotTable: no table has this code (it may be a Duel room).
	ErrNotTable = errors.New("tableroom: no table with that code")
	// ErrLogRunning: the game has not ended.
	ErrLogRunning = errors.New("tableroom: the game has not ended")
	// ErrLogHidden: the table turned the final reveal off, which also
	// turns the download off: its keys would expose the hidden cards.
	ErrLogHidden = errors.New("tableroom: the table keeps its hidden cards hidden after the end")
	// ErrLogArchived: the table was archived; its log is not trusted.
	ErrLogArchived = errors.New("tableroom: the table was archived")
	// ErrLogReplay: the record does not replay to its rules hash.
	ErrLogReplay = errors.New("tableroom: the table's log does not replay")
)

// ReadTableLog returns a finished table's log, under the end policy. The
// record is read under its lock without any write; a running, hidden,
// archived or unreplayable table gets its own error, and anything that is
// not a table (a missing code, a Duel room) gets ErrNotTable.
func (t *Tables) ReadTableLog(code string) (*Log, error) {
	if store.ValidateCode(code) != nil {
		return nil, ErrNotTable
	}
	data, archived, err := t.store().PeekGame(code, Game)
	switch {
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, store.ErrSchema), errors.Is(err, store.ErrGame):
		return nil, ErrNotTable
	case err != nil:
		return nil, err
	case archived:
		return nil, ErrLogArchived
	}
	r, err := decodeRecord(data)
	if err != nil {
		return nil, ErrLogReplay
	}
	switch {
	case r.Status != store.StatusFinished:
		return nil, ErrLogRunning
	case r.Setup.HideAtEnd:
		return nil, ErrLogHidden
	case r.Rules != RulesVersion:
		return nil, ErrLogArchived
	}
	n := r.Setup.Seats
	if n < MinSeats || n > MaxSeats || len(r.Batches) > maxBatches {
		return nil, ErrLogReplay
	}
	g, err := replay(makeSetup(r.Setup.Seed, n, r.Setup.RandomSeating, r.Setup.HideAtEnd), engineBatches(r.Batches))
	if err != nil || !pointOver(g.Point()) || g.Fingerprint() != r.Hash {
		return nil, ErrLogReplay
	}
	l := &Log{Format: LogFormat, Game: Game, Rules: r.Rules, Setup: r.Setup, Batches: r.Batches, Hash: r.Hash}
	if l.Batches == nil {
		l.Batches = []BatchRecord{}
	}
	return l, nil
}
