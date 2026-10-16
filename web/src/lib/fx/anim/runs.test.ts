import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { animRecords, clearRecords } from './records';
import { Runs, type Playing } from './runs';

/** A stand-in for a WAAPI Animation: it ends when finish() is called or `end` runs. */
function fake(): Playing & { end: () => void; finishes: number } {
  let resolve!: () => void;
  const finished = new Promise<void>((r) => (resolve = r));
  const anim = {
    playState: 'running',
    ready: Promise.resolve(),
    finished,
    finishes: 0,
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
});
