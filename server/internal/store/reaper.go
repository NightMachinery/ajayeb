package store

import "time"

// Clock returns the current time. Production code passes time.Now; a test
// passes a func that returns a fixed or stepped value, so expiry is
// checkable without sleeping for real.
type Clock func() time.Time

// Reaper deletes room files that have outlived their purpose: a finished
// game after FinishedAfter, or a room that never finished (still lobby or
// playing) after IdleAfter of no update. The zero value is not valid; use
// NewReaper.
type Reaper struct {
	st  *Store
	now Clock

	// FinishedAfter and IdleAfter default to 7 and 14 days (NewReaper); a
	// caller may change them before the first Sweep.
	FinishedAfter time.Duration
	IdleAfter     time.Duration
}

// NewReaper returns a Reaper over st. now is the clock Sweep reads; nil
// means time.Now.
func NewReaper(st *Store, now Clock) *Reaper {
	if now == nil {
		now = time.Now
	}
	return &Reaper{
		st:            st,
		now:           now,
		FinishedAfter: 7 * 24 * time.Hour,
		IdleAfter:     14 * 24 * time.Hour,
	}
}

// Sweep deletes every room file eligible for removal and returns the codes
// it removed. It reads each file's header only (PeekHeader), never a full
// replay: the reaper's job is age-based cleanup, not integrity checking
// (Load already handles that, on demand, and archives what it cannot
// trust). A header Sweep cannot even parse is left alone; whatever is wrong
// with it, deleting it silently is not the reaper's call to make.
//
//   - StatusFinished or StatusArchived: eligible once now - Updated exceeds
//     FinishedAfter.
//   - StatusLobby or StatusPlaying: eligible once now - Updated exceeds
//     IdleAfter (an abandoned room that never reached an end).
func (r *Reaper) Sweep() ([]string, error) {
	codes, err := r.st.Codes()
	if err != nil {
		return nil, err
	}
	now := r.now()

	var removed []string
	for _, code := range codes {
		f, err := r.st.PeekHeader(code)
		if err != nil {
			continue
		}
		age := now.Sub(f.Updated)
		var eligible bool
		switch f.Status {
		case StatusFinished, StatusArchived:
			eligible = age > r.FinishedAfter
		case StatusLobby, StatusPlaying:
			eligible = age > r.IdleAfter
		}
		if !eligible {
			continue
		}
		if err := r.st.Delete(code); err != nil {
			return removed, err
		}
		removed = append(removed, code)
	}
	return removed, nil
}
