package store

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ReadLog reads code's room file and returns its header and its game as an
// ajayeb-log/1 log (engine.Log), sealed with the final state's hash. It is a
// read helper for the replay download: unlike Load it never writes, so a
// file that does not replay is reported, not archived.
//
// A malformed code wraps ErrBadCode and a missing file wraps
// fs.ErrNotExist. A file of another schema wraps ErrSchema, one of another
// rules version ErrRulesVersion, and one whose replay does not reach its
// recorded Hash ErrHashMismatch.
func (st *Store) ReadLog(code string) (*File, *engine.Log, error) {
	f, err := st.PeekHeader(code)
	if err != nil {
		return nil, nil, err
	}
	l, err := GameLog(f)
	if err != nil {
		return f, nil, err
	}
	return f, l, nil
}

// GameLog turns a room file into the log of its game: the same seed,
// options and scenario, and one entry per action with the ply and seat it
// was applied at, which only a replay knows. The log is sealed with the
// final hash, and that hash must match f.Hash when the file has one.
func GameLog(f *File) (*engine.Log, error) {
	if f.Rules != engine.RulesVersion {
		return nil, fmt.Errorf("store: log of %s: %w: file plays rules %d, this engine plays %d",
			f.Code, ErrRulesVersion, f.Rules, engine.RulesVersion)
	}
	var sc *engine.Scenario
	if f.Scenario != "" {
		sc = engine.NamedScenario(f.Scenario)
		if sc == nil {
			return nil, fmt.Errorf("store: log of %s: unknown scenario %q", f.Code, f.Scenario)
		}
	}
	l := engine.NewLog(f.Seed, f.Options, sc)
	s, err := l.Start()
	if err != nil {
		return nil, fmt.Errorf("store: log of %s: %w", f.Code, err)
	}
	for i, key := range f.Actions {
		a, err := engine.ParseKey(key)
		if err != nil {
			return nil, fmt.Errorf("store: log of %s: action %d (%q): %w", f.Code, i, key, err)
		}
		l.Record(&s, a)
		if err := engine.Apply(&s, a, nil); err != nil {
			return nil, fmt.Errorf("store: log of %s: action %d (%s): %w", f.Code, i, key, err)
		}
	}
	l.Seal(&s)
	if f.Hash != "" && l.FinalHash != f.Hash {
		return nil, fmt.Errorf("store: log of %s: %w: replay hash %s, file says %s", f.Code, ErrHashMismatch, l.FinalHash, f.Hash)
	}
	return l, nil
}
