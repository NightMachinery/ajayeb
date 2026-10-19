import { afterEach, describe, expect, it, vi } from 'vitest';
import type { GameEvent, State } from '$lib/wire';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import { planCues } from '$lib/fx/cues';
import { cuePrev, isBotAnswer, isBotTake, Presenter, type FrameSource } from './presenter.svelte';
import type { Unsubscribe } from './transport';

function frame(version: number, thinking?: 0 | 1): State {
  const i = (fixtureNames as readonly string[]).indexOf('state-play');
  const f = structuredClone(fixtures[i]) as State;
  delete f.ack;
  f.events = [];
  f.version = version;
  if (thinking !== undefined) f.thinking = thinking;
  else delete f.thinking;
  return f;
}

/** A bot's answer: a frame whose events say seat 1 did something. */
function botMove(version: number, kind: GameEvent['kind'], thinking?: 0 | 1): State {
  const f = frame(version, thinking);
  f.events = [{ kind, private: false, seat: 1, card: 'lumber-yard' }];
  return f;
}

class Source implements FrameSource {
  fns: ((f: State) => void)[] = [];
  subscribe(fn: (f: State) => void): Unsubscribe {
    this.fns.push(fn);
    return () => {
      this.fns = this.fns.filter((x) => x !== fn);
    };
  }
  push(f: State) {
    for (const fn of this.fns) fn(f);
  }
}

/** A manual clock and sleep, so dwell times are exact and instant. */
function clock() {
  let now = 0;
  const sleeps: number[] = [];
  return {
    now: () => now,
    sleep: async (ms: number) => {
      sleeps.push(ms);
      now += ms;
    },
    advance: (ms: number) => {
      now += ms;
    },
    sleeps
  };
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('Presenter', () => {
  it('presents frames in order and is idle when done', async () => {
    const source = new Source();
    const order: string[] = [];
    const p = new Presenter(source, {
      afterSwap: async () => void order.push('tick'),
      hooks: {
        capture: (next, prev) => {
          order.push(`capture ${prev?.version ?? '-'}>${next.version}`);
          return next.version;
        },
        play: (next, _prev, captured) => {
          order.push(`play ${next.version} ${String(captured)}`);
        }
      }
    });
    expect(p.idle).toBe(true);
    source.push(frame(1));
    expect(p.idle).toBe(false);
    await settle();
    expect(p.frame?.version).toBe(1);
    expect(p.view).toBe(p.frame?.view);
    expect(p.idle).toBe(true);
    source.push(frame(2));
    await settle();
    expect(order).toEqual(['capture ->1', 'tick', 'play 1 1', 'capture 1>2', 'tick', 'play 2 2']);
  });

  it('coalesces queued cosmetic updates without losing a live move or treating it as backlog', async () => {
    let release!: () => void;
    const seen: { version: number; events: GameEvent[]; queued: number }[] = [];
    const p = new Presenter(null, {
      afterSwap: async () => {},
      hooks: {
        capture: (next) =>
          seen.push({ version: next.version, events: next.events, queued: p.queued }),
        play: (next) => (next.version === 1 ? new Promise<void>((r) => (release = r)) : undefined)
      }
    });
    p.enqueue(frame(1, 1));
    await settle();
    const move = botMove(2, 'build');
    p.enqueue(move);
    const review = frame(2);
    review.room = { ...review.room, spectators: 3 };
    p.enqueue(review);
    const latest = { ...review, room: { ...review.room, spectators: 4 } };
    p.enqueue(latest);
    expect(p.queued).toBe(1);
    release();
    await settle();
    expect(seen.find((f) => f.version === 2)).toEqual({
      version: 2,
      events: move.events,
      queued: 0
    });
    expect(p.frame?.room.spectators).toBe(4);
    expect(p.frame).toBe(latest); // preserves Presenter/Session ready() identity
    expect(p.frame?.events).toEqual([]); // server snapshot remains unchanged
    p.destroy();
  });

  it('updates cosmetic data without capturing or playing again; history still resets effects', async () => {
    const capture = vi.fn();
    const play = vi.fn();
    const p = new Presenter(null, { afterSwap: async () => {}, hooks: { capture, play } });
    p.enqueue(botMove(2, 'build'));
    await settle();
    const cosmetic = frame(2);
    cosmetic.room = { ...cosmetic.room, spectators: 3 };
    p.enqueue(cosmetic);
    await settle();
    expect(p.frame).toBe(cosmetic);
    expect(capture).toHaveBeenCalledTimes(1);
    expect(play).toHaveBeenCalledTimes(1);
    p.enqueue({ ...cosmetic, history: [] });
    await settle();
    expect(capture).toHaveBeenCalledTimes(2);
    expect(play).toHaveBeenCalledTimes(2);
    p.destroy();
  });

  it('waits for a slow animation before the next frame', async () => {
    const source = new Source();
    let release!: () => void;
    const p = new Presenter(source, {
      afterSwap: async () => {},
      hooks: {
        play: (next) => (next.version === 1 ? new Promise<void>((r) => (release = r)) : undefined)
      }
    });
    source.push(frame(1));
    source.push(frame(2));
    await settle();
    expect(p.frame?.version).toBe(1);
    expect(p.idle).toBe(false);
    release();
    await settle();
    expect(p.frame?.version).toBe(2);
    expect(p.idle).toBe(true);
  });

  it('hurry(), clear() and destroy() call the finish hook, which ends a playing animation', async () => {
    let release: (() => void) | null = null;
    const finish = vi.fn(() => {
      release?.();
      release = null;
    });
    const p = new Presenter(null, {
      afterSwap: async () => {},
      hooks: { play: () => new Promise<void>((r) => (release = r)), finish }
    });
    p.enqueue(frame(1));
    await settle();
    expect(p.idle).toBe(false);
    p.hurry();
    expect(finish).toHaveBeenCalledTimes(1);
    await settle();
    expect(p.idle).toBe(true);
    // Even idle, hurry() ends the tail of an animation that no longer blocks.
    p.hurry();
    expect(finish).toHaveBeenCalledTimes(2);
    p.clear();
    expect(finish).toHaveBeenCalledTimes(3);
    p.destroy();
    expect(finish).toHaveBeenCalledTimes(4);
  });

  it('says how many frames wait behind the one being shown', async () => {
    const seen: number[] = [];
    let release!: () => void;
    const p = new Presenter(null, { afterSwap: async () => {} });
    p.hooks = {
      capture: () => void seen.push(p.queued),
      play: (next) => (next.version === 1 ? new Promise<void>((r) => (release = r)) : undefined)
    };
    p.enqueue(frame(1));
    await settle();
    p.enqueue(frame(2));
    p.enqueue(frame(3));
    expect(p.queued).toBe(2);
    release();
    await settle();
    // Frame 1 had nothing behind it; frame 2 had frame 3; frame 3 nothing.
    expect(seen).toEqual([0, 1, 0]);
  });

  it('fast-forwards a long queue to the newest frame', async () => {
    const p = new Presenter(null, { afterSwap: async () => {}, fastForwardAfter: 2 });
    const seen: number[] = [];
    p.hooks = { play: (next) => void seen.push(next.version) };
    for (let v = 1; v <= 6; v++) p.enqueue(frame(v));
    await settle();
    expect(p.frame?.version).toBe(6);
    expect(seen[0]).toBe(1);
    expect(seen.at(-1)).toBe(6);
    expect(seen.length).toBeLessThan(6);
  });

  it('holds a bot move until the thinking frame had its minimum dwell', async () => {
    const c = clock();
    const p = new Presenter(null, {
      afterSwap: async () => {},
      minThinkingMs: 700,
      now: c.now,
      sleep: c.sleep
    });
    p.enqueue(frame(1, 1)); // the bot starts thinking
    await settle();
    c.advance(200);
    p.enqueue(botMove(2, 'build')); // its move
    await settle();
    expect(c.sleeps).toEqual([500]);
    expect(p.frame?.version).toBe(2);

    // A move that comes after the dwell is shown at once.
    p.enqueue(frame(3, 1));
    await settle();
    c.advance(900);
    p.enqueue(botMove(4, 'build'));
    await settle();
    expect(c.sleeps).toEqual([500]);
  });

  it('applies frames immediately with no dwell setting', async () => {
    const c = clock();
    const p = new Presenter(null, { afterSwap: async () => {}, now: c.now, sleep: c.sleep });
    p.enqueue(frame(1, 1));
    p.enqueue(frame(2));
    await settle();
    expect(c.sleeps).toEqual([]);
    expect(p.frame?.version).toBe(2);
  });

  it('keeps going when an animation throws', async () => {
    const p = new Presenter(null, {
      afterSwap: async () => {},
      hooks: {
        play: () => {
          throw new Error('boom');
        }
      }
    });
    const original = console.error;
    console.error = () => {};
    try {
      p.enqueue(frame(1));
      p.enqueue(frame(2));
      await settle();
    } finally {
      console.error = original;
    }
    expect(p.frame?.version).toBe(2);
    expect(p.idle).toBe(true);
  });

  it('stops listening when destroyed', async () => {
    const source = new Source();
    const p = new Presenter(source, { afterSwap: async () => {} });
    p.destroy();
    source.push(frame(1));
    await settle();
    expect(p.frame).toBeNull();
    expect(source.fns).toHaveLength(0);
  });
});

describe('Presenter: pacing a bot move (fake timers)', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  /** A presenter on the real clock and setTimeout, which the fake timers drive. */
  function paced(opts: { dwell?: number | (() => number); preRoll?: number } = {}) {
    vi.useFakeTimers();
    return new Presenter(null, {
      afterSwap: async () => {},
      minThinkingMs: opts.dwell ?? 700,
      preRollMs: opts.preRoll ?? 300
    });
  }

  it('dwells, then previews the pick, then shows the move, with input locked throughout', async () => {
    const p = paced();
    p.enqueue(frame(1, 1)); // the human's move was acked; the bot thinks
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(1);
    expect(p.idle).toBe(true);

    await vi.advanceTimersByTimeAsync(400); // the server's bot delay
    const move = botMove(2, 'build');
    p.enqueue(move);
    await vi.advanceTimersByTimeAsync(0);
    expect(p.idle).toBe(false);
    expect(p.frame?.version).toBe(1);
    expect(p.preview).toBeNull();

    // The thinking frame has been up 400 ms of its 700: 300 more to wait.
    await vi.advanceTimersByTimeAsync(299);
    expect(p.preview).toBeNull();
    await vi.advanceTimersByTimeAsync(1);
    expect(p.preview).toBe(move); // the pick is highlighted on the old frame
    expect(p.frame?.version).toBe(1);
    expect(p.idle).toBe(false);

    await vi.advanceTimersByTimeAsync(299);
    expect(p.frame?.version).toBe(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(p.frame?.version).toBe(2);
    expect(p.preview).toBeNull();
    expect(p.idle).toBe(true);
  });

  it('previews at once when the bot answered after the dwell', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(900);
    p.enqueue(botMove(2, 'discard'));
    await vi.advanceTimersByTimeAsync(0);
    expect(p.preview?.version).toBe(2);
    await vi.advanceTimersByTimeAsync(300);
    expect(p.frame?.version).toBe(2);
  });

  it('dwells but does not preview an answer that takes nothing from the board', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'starterChosen'));
    await vi.advanceTimersByTimeAsync(699);
    expect(p.frame?.version).toBe(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(p.frame?.version).toBe(2);
    expect(p.preview).toBeNull();
  });

  it('paces each move of a bot that plays twice', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'build', 1)); // its move, and it plays again
    await vi.advanceTimersByTimeAsync(1000);
    expect(p.frame?.version).toBe(2);
    p.enqueue(botMove(3, 'build'));
    await vi.advanceTimersByTimeAsync(999);
    expect(p.frame?.version).toBe(2); // a fresh dwell for the second move
    await vi.advanceTimersByTimeAsync(1);
    expect(p.frame?.version).toBe(3);
  });

  it('does not pace an undo, which carries no events', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(frame(2)); // the undo's answer: the human's turn again
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(2);
    expect(p.idle).toBe(true);
  });

  it('does not pace a move that is not answering a thinking frame', async () => {
    const p = paced();
    p.enqueue(frame(1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'build'));
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(2);
  });

  it('reads the dwell when it is needed, so a speed change applies to the next move', async () => {
    let dwell = 700;
    const p = paced({ dwell: () => dwell, preRoll: 0 });
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    dwell = 300; // animation off: the short floor
    p.enqueue(botMove(2, 'build'));
    await vi.advanceTimersByTimeAsync(299);
    expect(p.frame?.version).toBe(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(p.frame?.version).toBe(2);
  });

  it('hurry() cuts the dwell and the preview short', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'build'));
    await vi.advanceTimersByTimeAsync(100);
    expect(p.frame?.version).toBe(1);
    p.hurry();
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(2);
    expect(p.preview).toBeNull();
    expect(p.idle).toBe(true);

    // The next bot move is paced again.
    p.enqueue(frame(3, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(4, 'build'));
    await vi.advanceTimersByTimeAsync(700);
    expect(p.preview?.version).toBe(4);
    p.hurry();
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(4);
    expect(p.preview).toBeNull();
  });

  it('skips the preview when frames queue up behind the answer', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'build'));
    p.enqueue(frame(3)); // arrives during the dwell
    await vi.advanceTimersByTimeAsync(699);
    expect(p.frame?.version).toBe(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(p.preview).toBeNull();
    expect(p.frame?.version).toBe(3);
  });

  it('clear() forgets the room, even in the middle of a dwell', async () => {
    const p = paced();
    p.enqueue(frame(1, 1));
    await vi.advanceTimersByTimeAsync(0);
    p.enqueue(botMove(2, 'build'));
    await vi.advanceTimersByTimeAsync(100);
    p.clear();
    expect(p.frame).toBeNull();
    expect(p.idle).toBe(true);
    await vi.advanceTimersByTimeAsync(1000);
    expect(p.frame).toBeNull();

    // The next room's frames show as usual.
    p.enqueue(frame(7));
    await vi.advanceTimersByTimeAsync(0);
    expect(p.frame?.version).toBe(7);
  });
});

describe('isBotAnswer and isBotTake', () => {
  it('needs a thinking frame on screen and events in the next one', () => {
    expect(isBotAnswer(null, botMove(2, 'build'))).toBe(false);
    expect(isBotAnswer(frame(1), botMove(2, 'build'))).toBe(false);
    expect(isBotAnswer(frame(1, 1), frame(2))).toBe(false);
    expect(isBotAnswer(frame(1, 1), botMove(2, 'token'))).toBe(true);
  });

  it('is a take only for a card or draft pick by the thinking seat', () => {
    for (const kind of ['build', 'discard', 'wonder', 'draft'] as const) {
      expect(isBotTake(frame(1, 1), botMove(2, kind))).toBe(true);
    }
    expect(isBotTake(frame(1, 1), botMove(2, 'token'))).toBe(false);
    expect(isBotTake(frame(1, 0), botMove(2, 'build'))).toBe(false);
  });

  it('is never an answer across rooms', () => {
    const other = botMove(2, 'build');
    other.code = 'OTHER1';
    expect(isBotAnswer(frame(1, 1), other)).toBe(false);
  });
});

describe('cuePrev', () => {
  it('a live frame plays its cues against the frame before it', () => {
    const prev = frame(1);
    const next = botMove(2, 'build');
    expect(cuePrev(prev, next)).toBe(prev);
    expect(planCues(next, cuePrev(prev, next)).cues.length).toBeGreaterThan(0);
  });

  it('a resync frame with a newer version replays no cue of a move already heard', () => {
    const heard = botMove(2, 'build');
    const resync = botMove(3, 'build');
    resync.history = [{ ply: 1, seat: 1, events: heard.events }];
    expect(cuePrev(heard, resync)).toBeNull();
    expect(planCues(resync, cuePrev(heard, resync))).toEqual({ cues: [], haptic: null });
  });
});
