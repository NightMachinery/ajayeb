package bot

import (
	"context"
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func TestSampleRootAndFallback(t *testing.T) {
	for n := uint8(3); n <= 7; n++ {
		g, err := rules.NewGame(rules.Setup{Seats: n, Seed: 712, Options: rules.Options{Sides: rules.SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		for s := rules.Seat(0); s < rules.Seat(n); s++ {
			o, cs := g.Observe(s), g.Legal(s)
			first, err := SampleRoot(context.Background(), o, cs, 91, rules.MaxSamplerChecks)
			if err != nil || !first.Ready || first.Counters.Worlds != 1 || first.Counters.Attempts != 1 || first.Counters.NoWorldFallbacks != 0 {
				t.Fatal("sample root", first, err)
			}
			state, ok := first.State()
			if !ok || state.Seed != 0 || rules.Project(&state, s) != o.Now {
				t.Fatal("wrong root state")
			}
			repeat, err := SampleRoot(context.Background(), o, cs, 91, rules.MaxSamplerChecks)
			if err != nil || repeat != first {
				t.Fatal("sample not repeatable", err)
			}
			fallback, err := SampleRoot(context.Background(), o, cs, 91, 0)
			greedy, e := Fallback(context.Background(), o, cs)
			if err != nil || e != nil || fallback.Ready || fallback.Fallback != greedy || fallback.Counters.NoWorldFallbacks != 1 || fallback.Counters.BudgetFallbacks != 1 || fallback.Counters.Attempts != 1 {
				t.Fatal("budget fallback", fallback, err, e)
			}
			if _, ok := fallback.State(); ok {
				t.Fatal("fallback exposed state")
			}
			// Fault injection: no true legal past has this claimed resolved pick
			// with an empty Seen hand. It forces constraint failure, not exhaustion.
			bad := o
			bad.Know.Picks[1][0] = o.Now.Hand[0]
			failed, err := SampleRoot(context.Background(), bad, cs, 91, rules.MaxSamplerChecks)
			if err != nil || failed.Ready || !failed.Fallback.Complete || failed.Counters.NoWorldFallbacks != 1 || failed.Counters.BudgetFallbacks != 0 {
				t.Fatal("no-world fallback", failed, err)
			}
		}
	}
}

func TestSampleRootCancellation(t *testing.T) {
	g, err := rules.NewGame(rules.Setup{Seats: 7, Seed: 91})
	if err != nil {
		t.Fatal(err)
	}
	o, cs := g.Observe(0), g.Legal(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if out, err := SampleRoot(ctx, o, cs, 1, rules.MaxSamplerChecks); !errors.Is(err, context.Canceled) || out != (RootSample{}) {
		t.Fatal("cancel produced replacement", out, err)
	}
	// An injected context changes only at deterministic sampler polls.
	for at := 3; at < 7; at++ {
		ctx := &sampleCancelContext{Context: context.Background(), at: at}
		if out, err := SampleRoot(ctx, o, cs, 1, rules.MaxSamplerChecks); !errors.Is(err, context.Canceled) || out != (RootSample{}) {
			t.Fatal("mid-sample cancellation", at, out, err)
		}
	}
}

type sampleCancelContext struct {
	context.Context
	at, calls int
}

func (c *sampleCancelContext) Err() error {
	c.calls++
	if c.calls >= c.at {
		return context.Canceled
	}
	return nil
}
