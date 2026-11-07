package tableroom

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/bot"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

// A dormant sampled-root call is injected at the existing worker hook, not as
// a released policy. Pause inside its polling callback while the actor handles
// another request, then structurally cancel the seat. No sampled work belongs
// on the actor, and cancellation must not accept even a Greedy replacement.
func TestSamplerWorkerActorResponsiveAndCancelled(t *testing.T) {
	testSampleWorkerCancellation(t, false)
}
func TestAgeWorkerActorResponsiveAndCancelled(t *testing.T) { testSampleWorkerCancellation(t, true) }

func testSampleWorkerCancellation(t *testing.T, age bool) {
	t.Helper()
	input := make(chan botInput, 1)
	started := make(chan struct{}, 1)
	reports := make(chan BotReport, 8)
	returned := make(chan error, 1)
	hook := func(ctx context.Context, seat int, _ string) error {
		if seat != 1 {
			<-ctx.Done()
			return ctx.Err()
		}
		var in botInput
		select {
		case in = <-input:
		case <-ctx.Done():
			return ctx.Err()
		}
		var err error
		if age {
			controlled := &actorAgeContext{Context: ctx, started: started}
			var result bot.AgeDecision
			result, err = bot.ChooseAge(controlled, in.observation, in.offered, in.entropy, bot.AgeWork{Attempts: 2, Candidates: 4, SamplerChecks: rules.MaxSamplerChecks})
			if result != (bot.AgeDecision{}) {
				err = errors.New("cancelled Age rollout exposed result")
			}
		} else {
			controlled := &actorSampleContext{Context: ctx, started: started}
			var result bot.RootSample
			result, err = bot.SampleRoot(controlled, in.observation, in.offered, in.entropy, rules.MaxSamplerChecks)
			if result != (bot.RootSample{}) {
				err = errors.New("cancelled root exposed result")
			}
		}
		returned <- err
		return err
	}
	pool := ai.NewPool(1)
	h := newTH(t, Config{Pool: pool, BotWork: hook, BotFinished: func(r BotReport) { reports <- r }}, soloSpec(7, 1218, BotNormal))
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	var own botInput
	if err := a.call(h.ctx, func() { own = policyInput(&a.st.rec, &a.st.game, 1) }); err != nil {
		t.Fatal(err)
	}
	input <- own
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("sampler did not enter worker")
	}
	probe, cancel := context.WithTimeout(h.ctx, time.Second)
	defer cancel()
	if err := a.call(probe, func() { _ = a.st.game.Point() }); err != nil {
		t.Fatal("sampling blocked actor", err)
	}
	h.want(tabletest.SeatClient(0), toHuman(1, 1, "watcher"), outAccepted, "cancel sampled worker")
	select {
	case err := <-returned:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("worker did not cancel", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sample cancellation late")
	}
	select {
	case r := <-reports:
		if !r.Cancelled || r.Fallback != "" {
			t.Fatal("cancel accepted fallback", r)
		}
	case <-time.After(time.Second):
		t.Fatal("missing worker report")
	}
	if len(h.record().BotFallbacks) != 0 {
		t.Fatal("ordinary watchdog fallback")
	}
}

type actorSampleContext struct {
	context.Context
	calls   int
	started chan<- struct{}
}

func (c *actorSampleContext) Err() error {
	c.calls++
	if c.calls == 4 {
		c.started <- struct{}{}
		<-c.Done()
	}
	return c.Context.Err()
}

// Test-only stack inspection pauses in the actual rollout rather than at an
// input-ranking or sampling poll. Production policies import no runtime API.
type actorAgeContext struct {
	context.Context
	started chan<- struct{}
	paused  bool
}

func (c *actorAgeContext) Err() error {
	if !c.paused {
		var pcs [24]uintptr
		n := runtime.Callers(2, pcs[:])
		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, "/bot.simulateAge") {
				c.paused = true
				c.started <- struct{}{}
				<-c.Done()
				break
			}
			if !more {
				break
			}
		}
	}
	return c.Context.Err()
}
