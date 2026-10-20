import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { animRecords, clearRecords } from './records';
import { Runs, type Playing } from './runs';

/** A stand-in for a WAAPI Animation: it ends when finish() is called or `end` runs. */
function fake(): Playing & { end: () => void; finishes: number; plays: number } {
  let resolve!: () => void;
  const finished = new Promise<void>((r) => (resolve = r));
  const anim = {
    playState: 'running',
    ready: Promise.resolve(),
    finished,
    finishes: 0,
    plays: 0,
    pause() {
      anim.playState = 'paused';
    },
    play() {
      anim.plays += 1;
      anim.playState = 'running';
    },
    finish() {
      anim.finishes += 1;
      anim.playState = 'finished';
      resolve();
    },
    cancel() {
      anim.playState = 'idle';
    },
    end() {
      anim.playState = 'finished';
      resolve();
    }
  };
  return anim;
}

let now = 0;
beforeEach(() => {
  vi.useFakeTimers();
  now = 0;
  clearRecords();
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('Runs', () => {
  it('ends a run when its animations finish on their own, and cleans up once', async () => {
    const runs = new Runs(() => now);
    const a = fake();
    const cleanup = vi.fn();
    runs.track(5, 'take', [a], cleanup);
    expect(runs.size).toBe(1);
    now = 16;
    await vi.advanceTimersByTimeAsync(0);
    now = 320;
    a.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(runs.size).toBe(0);
    expect(cleanup).toHaveBeenCalledTimes(1);
    expect(animRecords()).toEqual([
      { version: 5, kind: 'take', startAt: 16, endAt: 320, interrupted: false }
    ]);
  });

  it('finishAll() finishes every animation at once, in its end state, and cleans up', async () => {
    const runs = new Runs(() => now);
    const take = fake();
    const flip = fake();
    const cleanups = [vi.fn(), vi.fn()];
    runs.track(5, 'take', [take], cleanups[0]!);
    runs.track(5, 'flip', [flip], cleanups[1]!);
    now = 100;
    runs.finishAll();
    expect(take.finishes).toBe(1);
    expect(flip.finishes).toBe(1);
    expect(cleanups[0]).toHaveBeenCalledTimes(1);
    expect(cleanups[1]).toHaveBeenCalledTimes(1);
    expect(runs.size).toBe(0);
    // The finished promises settle later; nothing runs twice.
    await vi.advanceTimersByTimeAsync(0);
    expect(cleanups[0]).toHaveBeenCalledTimes(1);
    expect(animRecords().map((r) => [r.kind, r.endAt, r.interrupted])).toEqual([
      ['take', 100, true],
      ['flip', 100, true]
    ]);
  });

  it('hold() lets input go after its time', async () => {
    const runs = new Runs(() => now);
    const released = vi.fn();
    let done = false;
    void runs.hold(320, released).then(() => (done = true));
    await vi.advanceTimersByTimeAsync(319);
    expect(done).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(done).toBe(true);
    expect(released).toHaveBeenCalledTimes(1);
  });

  it('a hold that drains the deal waits for its last card as well as its time', async () => {
    const runs = new Runs(() => now);
    const card = fake();
    runs.track(5, 'deal', [card], () => {});
    runs.track(5, 'take', [fake()], () => {});
    let done = false;
    void runs.hold(320, () => {}, 'deal').then(() => (done = true));
    await vi.advanceTimersByTimeAsync(400);
    expect(done, 'the card started late and still flies').toBe(false);
    card.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(done, 'other kinds do not hold it').toBe(true);
    // Drained before its time: the time still holds.
    done = false;
    void runs.hold(320, () => {}, 'deal').then(() => (done = true));
    await vi.advanceTimersByTimeAsync(319);
    expect(done).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(done).toBe(true);
  });

  it('a drain with no time left still waits for a deal that is only starting', async () => {
    const runs = new Runs(() => now);
    const card = fake();
    runs.track(5, 'deal', [card], () => {});
    let done = false;
    void runs.hold(0, () => {}, 'deal').then(() => (done = true));
    await vi.advanceTimersByTimeAsync(50);
    expect(done).toBe(false);
    card.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(done).toBe(true);
  });

  it('an all-kind drain waits for tails and counters after the input timer expires', async () => {
    const runs = new Runs(() => now);
    const take = fake();
    const pawn = fake();
    const banner = fake();
    runs.track(5, 'take', [take], () => {});
    runs.track(5, 'pawn', [pawn], () => {});
    runs.track(5, 'victory', [banner], () => {});
    const counter = runs.track(5, 'coins', [], () => {}, false);
    const released = vi.fn();
    void runs.hold(240, released, 'all');
    await vi.advanceTimersByTimeAsync(1000);
    expect(released).not.toHaveBeenCalled();
    take.end();
    pawn.end();
    banner.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(released, 'a counter has no WAAPI finished promise').not.toHaveBeenCalled();
    now = 1200;
    runs.end(counter, false);
    expect(released).toHaveBeenCalledOnce();
    expect(animRecords().every((r) => r.endAt !== null && !r.interrupted)).toBe(true);
  });

  it('a zero-time all-kind hold still waits for a run behind the paint gate', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (frame) => frames.push(frame)
    );
    const open = runs.gate();
    const pawn = fake();
    runs.track(5, 'pawn', [pawn], () => {});
    const released = vi.fn();
    void runs.hold(0, released, 'all');
    open();
    await vi.advanceTimersByTimeAsync(0);
    expect(pawn.playState).toBe('paused');
    expect(released).not.toHaveBeenCalled();
    for (let i = 0; i < 2; i++) for (const frame of frames.splice(0)) frame();
    expect(pawn.playState).toBe('running');
    pawn.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(released).toHaveBeenCalledOnce();
  });

  it('an all-kind drain includes a landing pulse created by the last cleanup', async () => {
    const runs = new Runs(() => now);
    const take = fake();
    const pulse = fake();
    runs.track(5, 'take', [take], () => {
      runs.track(5, 'take', [pulse], () => {});
    });
    const released = vi.fn(() => {
      expect(animRecords().every((r) => r.endAt !== null)).toBe(true);
    });
    void runs.hold(0, released, 'all');
    await vi.advanceTimersByTimeAsync(0);
    take.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(released).not.toHaveBeenCalled();
    pulse.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(released).toHaveBeenCalledOnce();
  });

  it('an interrupt releases an all-kind hold even while the gate is closed', async () => {
    const runs = new Runs(
      () => now,
      () => {}
    );
    runs.gate();
    const pawn = fake();
    runs.track(5, 'pawn', [pawn], () => {});
    const released = vi.fn();
    void runs.hold(240, released, 'all');
    runs.finishAll();
    await vi.advanceTimersByTimeAsync(0);
    expect(released).toHaveBeenCalledOnce();
    expect(pawn.finishes).toBe(1);
    expect(animRecords()[0]?.interrupted).toBe(true);
  });

  it('bounds a never-ending run after the paint gate opens, not while a tab is hidden', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (frame) => frames.push(frame)
    );
    const open = runs.gate();
    const cleanup = vi.fn();
    const stuck = runs.track(5, 'coins', [], cleanup, false);
    const released = vi.fn();
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    void runs.hold(0, released, 'all', 2000);
    open();
    await vi.advanceTimersByTimeAsync(20_000);
    expect(released, 'no motion started while the paint gate waited').not.toHaveBeenCalled();
    expect(warn).not.toHaveBeenCalled();
    for (let i = 0; i < 2; i++) for (const frame of frames.splice(0)) frame();
    await vi.advanceTimersByTimeAsync(1999);
    expect(released).not.toHaveBeenCalled();
    now = 22_000;
    await vi.advanceTimersByTimeAsync(1);
    expect(released).toHaveBeenCalledOnce();
    expect(cleanup).toHaveBeenCalledOnce();
    expect(stuck.record).toMatchObject({ endAt: 22_000, interrupted: true });
    expect(runs.size).toBe(0);
    expect(warn).toHaveBeenCalledExactlyOnceWith(
      'Animation completion drain timed out; finishing remaining effects'
    );
  });

  it('natural completion cancels the all-kind safety limit', async () => {
    const runs = new Runs(() => now);
    const pawn = fake();
    runs.track(5, 'pawn', [pawn], () => {});
    const released = vi.fn();
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    void runs.hold(0, released, 'all', 2000);
    await vi.advanceTimersByTimeAsync(0);
    pawn.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(released).toHaveBeenCalledOnce();
    await vi.advanceTimersByTimeAsync(20_000);
    expect(warn).not.toHaveBeenCalled();
    expect(animRecords()[0]?.interrupted).toBe(false);
    expect(released).toHaveBeenCalledOnce();
  });

  it('an interrupted gate cannot arm an old limit against a later hold', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (frame) => frames.push(frame)
    );
    const open = runs.gate();
    runs.track(5, 'coins', [], () => {}, false);
    const first = vi.fn();
    const next = vi.fn();
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    void runs.hold(0, first, 'all', 2000);
    open();
    runs.finishAll();
    void runs.hold(50_000, next);
    for (let i = 0; i < 2; i++) for (const frame of frames.splice(0)) frame();
    await vi.advanceTimersByTimeAsync(20_000);
    expect(first).toHaveBeenCalledOnce();
    expect(next).not.toHaveBeenCalled();
    expect(warn).not.toHaveBeenCalled();
    runs.finishAll();
    expect(next).toHaveBeenCalledOnce();
  });

  it('an interrupt lets input go at once', async () => {
    const runs = new Runs(() => now);
    runs.track(5, 'take', [fake()], () => {});
    let done = false;
    void runs.hold(320, () => {}).then(() => (done = true));
    runs.finishAll();
    await vi.advanceTimersByTimeAsync(0);
    expect(done).toBe(true);
  });

  it('a run that is not followed ends only when told to', async () => {
    const runs = new Runs(() => now);
    const a = fake();
    const run = runs.track(5, 'take', [a], () => {}, false);
    a.end();
    await vi.advanceTimersByTimeAsync(0);
    expect(runs.size).toBe(1);
    runs.end(run, false);
    expect(runs.size).toBe(0);
    expect(run.record.interrupted).toBe(false);
  });

  it('a run with no animation of its own starts at the next frame, not in the task that tracks it', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (run) => frames.push(run)
    );
    now = 10;
    const run = runs.track(5, 'coins', [], () => {}, false);
    await vi.advanceTimersByTimeAsync(0);
    expect(run.record.startAt).toBe(10);
    now = 26;
    frames.shift()?.();
    expect(run.record.startAt).toBe(26);
    // Ended before its first frame: the record keeps its end, not a later start.
    const early = runs.track(5, 'deal', [], () => {}, false);
    now = 30;
    runs.end(early, true);
    now = 42;
    frames.shift()?.();
    expect(early.record.startAt).toBe(26);
    expect(early.record.endAt).toBe(30);
  });

  it('a gate holds the runs tracked behind it, paused, until two frames after it opens', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (run) => frames.push(run)
    );
    const tick = () => {
      const due = frames.splice(0);
      for (const run of due) run();
    };
    const open = runs.gate();
    const a = fake();
    const run = runs.track(5, 'take', [a], () => {});
    expect(a.playState).toBe('paused');
    open();
    // Tracked after the gate opened: not held.
    const b = fake();
    runs.track(5, 'flip', [b], () => {});
    expect(b.playState).toBe('running');
    now = 16;
    tick();
    expect(a.playState).toBe('paused');
    now = 33;
    tick();
    expect(a.playState).toBe('running');
    expect(a.plays).toBe(1);
    await vi.advanceTimersByTimeAsync(0);
    expect(run.record.startAt).toBe(33);
  });

  it('a run finished while held is never played', async () => {
    const frames: (() => void)[] = [];
    const runs = new Runs(
      () => now,
      (run) => frames.push(run)
    );
    const open = runs.gate();
    const a = fake();
    const cleanup = vi.fn();
    runs.track(5, 'take', [a], cleanup);
    open();
    runs.finishAll();
    expect(a.finishes).toBe(1);
    expect(cleanup).toHaveBeenCalledTimes(1);
    for (let i = 0; i < 3; i++) for (const run of frames.splice(0)) run();
    expect(a.plays).toBe(0);
  });
});
