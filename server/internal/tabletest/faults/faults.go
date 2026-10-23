// Package faults holds the failure-injection fixtures for the Haftgane room
// actor's storage, restore and bot-work paths
// (docs/haftgane/platform-architecture.md, "Persistence, replay and
// compatibility" and "Bots: information, work and lifecycle"):
//
//   - FileStore writes a record atomically (temporary file, sync, rename)
//     and can fail a write, or crash before or after the rename;
//   - PendingSubsets and InvalidPendings enumerate the pending blocks a
//     restore must accept and refuse;
//   - Permits is a counting work pool that reports held permits and
//     waiters, so a cancellation test can prove nothing leaked;
//   - Work is a bot search that blocks until released or cancelled, and
//     counts how often it started, finished and was cancelled, so a test can
//     prove a paused room started none.
//
// The fixtures depend on no engine and no actor; their tests exercise them
// against the fake engine through package direct.
package faults

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrInjected is a storage error the store reports without writing.
var ErrInjected = errors.New("faults: injected storage error")

// ErrCrash stands for the process dying inside a write. The caller must
// drop its in-memory state and restore from what is on disk.
var ErrCrash = errors.New("faults: injected crash")

// Fault is what a FileStore does to one write.
type Fault int

const (
	None              Fault = iota
	FailWrite               // write nothing and return ErrInjected
	CrashBeforeRename       // leave the written temporary file, never rename, return ErrCrash
	CrashAfterRename        // install the record, then return ErrCrash before anyone hears of it
)

// FileStore keeps one record in a directory, written the way the room
// store writes room files.
type FileStore struct {
	Path string

	mu     sync.Mutex
	queued []Fault
	saves  int
}

// NewFileStore returns a store for dir/room.json.
func NewFileStore(dir string) *FileStore {
	return &FileStore{Path: filepath.Join(dir, "room.json")}
}

// Inject queues faults for the coming writes, one per write, in order.
func (s *FileStore) Inject(faults ...Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queued = append(s.queued, faults...)
}

// Saves returns how many writes were attempted.
func (s *FileStore) Saves() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saves
}

// Save writes data atomically, applying the next queued fault.
func (s *FileStore) Save(ctx context.Context, data []byte) error {
	s.mu.Lock()
	f := None
	if len(s.queued) > 0 {
		f, s.queued = s.queued[0], s.queued[1:]
	}
	s.saves++
	s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if f == FailWrite {
		return ErrInjected
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), ".room-*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if f == CrashBeforeRename {
		return ErrCrash
	}
	if err := os.Rename(tmp.Name(), s.Path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if f == CrashAfterRename {
		return ErrCrash
	}
	return nil
}

// Load reads the installed record.
func (s *FileStore) Load() ([]byte, error) { return os.ReadFile(s.Path) }

// Temps lists the temporary files a crash left behind.
func (s *FileStore) Temps() ([]string, error) {
	entries, err := os.ReadDir(filepath.Dir(s.Path))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".room-") && strings.HasSuffix(e.Name(), ".tmp") {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// IsCrash reports whether err is an injected crash.
func IsCrash(err error) bool { return errors.Is(err, ErrCrash) }

// Slot is one seat's pending pick.
type Slot struct {
	Seat      int
	Key       string
	Confirmed bool
}

// Pending is a pending block: the slots of the required seats that have a
// pick, by seat.
type Pending []Slot

// PendingSubsets returns every valid pending block for a point: each
// required seat has no pick, an unconfirmed pick or a confirmed pick (its
// key from key(seat)), except the block where every required seat has
// confirmed, which would have resolved. For k required seats that is
// 3^k - 1 blocks, the empty block first.
func PendingSubsets(required []int, key func(seat int) string) []Pending {
	total := 1
	for range required {
		total *= 3
	}
	out := make([]Pending, 0, total-1)
	for code := 0; code < total; code++ {
		p := Pending{}
		c, confirmed := code, 0
		for _, s := range required {
			switch c % 3 {
			case 1:
				p = append(p, Slot{Seat: s, Key: key(s)})
			case 2:
				p = append(p, Slot{Seat: s, Key: key(s), Confirmed: true})
				confirmed++
			}
			c /= 3
		}
		if confirmed < len(required) {
			out = append(out, p)
		}
	}
	return out
}

// Named is a pending block with what makes it invalid.
type Named struct {
	Name    string
	Pending Pending
}

// InvalidPendings returns pending blocks a restore must refuse: every
// required seat confirmed; one seat twice; a key the seat is not offered;
// a seat the point does not require (outsider, which must not be
// required); and a slot without a key.
func InvalidPendings(required []int, key func(seat int) string, unoffered string, outsider int) []Named {
	all := Pending{}
	for _, s := range required {
		all = append(all, Slot{Seat: s, Key: key(s), Confirmed: true})
	}
	first := required[0]
	return []Named{
		{"every required seat confirmed", all},
		{"a seat twice", Pending{{Seat: first, Key: key(first)}, {Seat: first, Key: key(first)}}},
		{"a key the seat is not offered", Pending{{Seat: first, Key: unoffered}}},
		{"a seat the point does not require", Pending{{Seat: outsider, Key: key(first)}}},
		{"a slot without a key", Pending{{Seat: first, Confirmed: true}}},
	}
}

// Permits is a counting work pool. Acquire waits for a free permit or the
// context; Held and Waiting let a test prove that cancellation leaked no
// permit and no waiter.
type Permits struct {
	free    chan struct{}
	mu      sync.Mutex
	held    int
	waiting int
}

// NewPermits returns a pool of n permits.
func NewPermits(n int) *Permits {
	p := &Permits{free: make(chan struct{}, n)}
	for range n {
		p.free <- struct{}{}
	}
	return p
}

// Acquire takes a permit, or returns the context's error.
func (p *Permits) Acquire(ctx context.Context) error {
	p.mu.Lock()
	p.waiting++
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.waiting--
		p.mu.Unlock()
	}()
	select {
	case <-p.free:
		p.mu.Lock()
		p.held++
		p.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release returns a permit.
func (p *Permits) Release() {
	p.mu.Lock()
	p.held--
	p.mu.Unlock()
	p.free <- struct{}{}
}

// Held returns the permits taken and not released.
func (p *Permits) Held() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.held
}

// Waiting returns the callers blocked in Acquire.
func (p *Permits) Waiting() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waiting
}

// Work is a bot search for tests. A blocking Work holds each search until
// Release or the context ends; an immediate one returns at once.
type Work struct {
	blocking bool
	release  chan struct{}
	started  chan Start

	mu                          sync.Mutex
	starts, finished, cancelled int
}

// Start is one search that began.
type Start struct {
	Seat  int
	Point string
}

// NewWork returns a Work; blocking chooses whether searches wait for
// Release.
func NewWork(blocking bool) *Work {
	return &Work{blocking: blocking, release: make(chan struct{}), started: make(chan Start, 64)}
}

// Run is the search: pass it as the actor's (or direct's) work function.
func (w *Work) Run(ctx context.Context, seat int, point string) error {
	w.mu.Lock()
	w.starts++
	w.mu.Unlock()
	select {
	case w.started <- Start{Seat: seat, Point: point}:
	default:
	}
	if w.blocking {
		select {
		case <-w.release:
		case <-ctx.Done():
			w.mu.Lock()
			w.cancelled++
			w.mu.Unlock()
			return ctx.Err()
		}
	}
	w.mu.Lock()
	w.finished++
	w.mu.Unlock()
	return nil
}

// Started delivers each search as it begins (buffered; extra starts beyond
// the buffer are counted but not delivered).
func (w *Work) Started() <-chan Start { return w.started }

// Release lets one blocked search finish.
func (w *Work) Release() { w.release <- struct{}{} }

// Counts returns how many searches started, finished and were cancelled.
func (w *Work) Counts() (started, finished, cancelled int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.starts, w.finished, w.cancelled
}
