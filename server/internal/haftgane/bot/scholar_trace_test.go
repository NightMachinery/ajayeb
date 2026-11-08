package bot

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

// Diagnostic only: stack capture adds overhead. The cancellation acceptance
// measurement remains TestScholarCancellationMeasurements without this tracer.
type scholarTraceContext struct {
	context.Context
	last          time.Time
	gap           time.Duration
	previous      uintptr
	before, after uintptr
	stack         [8]uintptr
}

func (c *scholarTraceContext) Err() error {
	now := time.Now()
	gap := now.Sub(c.last)
	pc, _, _, _ := runtime.Caller(1)
	if gap > c.gap {
		c.gap = gap
		c.before = c.previous
		c.after = pc
		runtime.Callers(2, c.stack[:])
	}
	c.last = now
	c.previous = pc
	return c.Context.Err()
}
func TestScholarPollTrace(t *testing.T) {
	if os.Getenv("HAFTGANE_SCHOLAR_TRACE") != "1" {
		t.Skip("set HAFTGANE_SCHOLAR_TRACE=1")
	}
	stop := scholarCPULoad(1)
	defer stop()
	positions := append(scholarLaterPositions(t), ageTail(t))
	var worst time.Duration
	for _, g := range positions {
		seat := rules.Seat(0)
		for !g.Point().Required.Has(seat) {
			seat++
		}
		for tape := uint64(0); tape < 8; tape++ {
			ctx := &scholarTraceContext{Context: context.Background(), last: time.Now()}
			out, err := ChooseAge(ctx, g.Observe(seat), g.Legal(seat), tape, ScholarWork())
			if err != nil || out.Fallback {
				t.Fatal(err, out.Counters)
			}
			if ctx.gap > 10*time.Millisecond {
				describe := func(pc uintptr) (string, string, int) {
					if pc == 0 {
						return "entry", "", 0
					}
					f := runtime.FuncForPC(pc)
					file, line := f.FileLine(pc)
					return f.Name(), file, line
				}
				a, af, al := describe(ctx.before)
				b, bf, bl := describe(ctx.after)
				t.Logf("gap n%d age%d turn%d tape%d ms=%.3f before=%s %s:%d after=%s %s:%d", g.State.Seats, g.State.Age, g.State.Turn, tape, float64(ctx.gap)/1e6, a, af, al, b, bf, bl)
				if ctx.gap > worst {
					worst = ctx.gap
					frames := runtime.CallersFrames(ctx.stack[:])
					for {
						f, more := frames.Next()
						t.Logf("worst_stack %s %s:%d", f.Function, f.File, f.Line)
						if !more {
							break
						}
					}
				}
			}
		}
	}
}
