package tableroom

import (
	"fmt"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestMergedFramesPassTheOracle: a connection whose writer falls behind
// gets its table frames merged by the outbox (docs/protocol.md, "The
// outbox"), so one frame can carry the events of several resolutions, and
// a history frame the events that follow its history. Every seat's and the
// spectator's frames of whole games, merged as an outbox merges them, pass
// the oracle, and a merged frame missing one of its events does not.
// Each connection also gets a resync answered halfway through, so a
// history frame meets the live frames after it. (TestWebSocketLobbyToTable
// once failed under load on such a frame: 12 events, where the oracle
// wanted the latest resolution's 2.)
func TestMergedFramesPassTheOracle(t *testing.T) {
	for _, n := range []int{MinSeats, 5, MaxSeats} {
		f := newFixture(t, Config{})
		seed := uint64(600 + n)
		p, truth := playForLeaks(t, f, n, seed, true)
		c := leakcheck.New(truth, frameConfig)
		spans, histories := 0, 0
		for id, frames := range p.frames {
			v := frameViewer{Seat: viewerOf(id), Client: string(id), Admin: viewerOf(id) == 0}
			for _, every := range []int{2, 3, 7} {
				label := fmt.Sprintf("%d seats %s merged every %d", n, id, every)
				merged := mergeFrames(t, withResync(t, frames), every)
				if leaks := frameLeaks(t, c, label, v, merged, nil, nil); len(leaks) != 0 {
					t.Fatalf("%s: %v", label, leaks[:min(len(leaks), 5)])
				}
				batch, span := 0, -1
				for i, fr := range merged {
					switch {
					case fr.History != nil && fr.Batch > historyBatch(fr):
						histories++
					case fr.History == nil && fr.Batch > batch+1:
						spans++
						span = i
					}
					batch = fr.Batch
				}
				if span < 0 {
					continue
				}
				// The canary: the span's first event dropped.
				cut := slices.Clone(merged)
				cut[span].Events = cut[span].Events[1:]
				if leaks := frameLeaks(t, c, label+" cut", v, cut, nil, nil); !slices.ContainsFunc(leaks, func(l leakcheck.Leak) bool { return l.Rule == leakcheck.Mismatch }) {
					t.Fatalf("%s: a merged frame missing an event passed: %v", label, leaks)
				}
			}
		}
		if spans == 0 || histories == 0 {
			t.Fatalf("%d seats: %d frames spanning resolutions, %d histories followed by events", n, spans, histories)
		}
	}
}

// withResync inserts, after the last frame of the middle batch, what a
// resync answers then: that frame with the history up to its batch, the
// first entries of the history the connection's last frame (the late
// resync) carries.
func withResync(t *testing.T, frames []wire.TableState) []wire.TableState {
	t.Helper()
	last := frames[len(frames)-1]
	if last.History == nil || historyBatch(last) != last.Batch || last.Batch < 2 {
		t.Fatalf("the last frame is not a late resync: batch %d", last.Batch)
	}
	k := last.Batch / 2
	at := -1
	for i, fr := range frames {
		if fr.Batch == k && fr.History == nil {
			at = i
		}
	}
	if at < 0 || (*last.History)[k-1].Batch != k {
		t.Fatalf("no frame at batch %d", k)
	}
	resync := frames[at]
	history := slices.Clone((*last.History)[:k])
	resync.History, resync.Events = &history, []wire.TableEvent{}
	return slices.Insert(slices.Clone(frames), at+1, resync)
}

// mergeFrames passes a connection's frames through an outbox whose writer
// takes what is waiting only after every few pushes.
func mergeFrames(t *testing.T, frames []wire.TableState, every int) []wire.TableState {
	t.Helper()
	o := room.NewOutbox()
	defer o.Close()
	var out []wire.TableState
	take := func() {
		for _, m := range o.Take() {
			fr, ok := m.(wire.TableState)
			if !ok {
				t.Fatalf("a %T in a table's outbox", m)
			}
			out = append(out, fr)
		}
	}
	for i, fr := range frames {
		o.Push(fr)
		if (i+1)%every == 0 {
			take()
		}
	}
	take()
	if o.Overflowed() {
		t.Fatal("the outbox overflowed")
	}
	return out
}
