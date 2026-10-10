import { describe, expect, it } from 'vitest';
import type { State } from '$lib/wire';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import { Presenter, type FrameSource } from './presenter.svelte';
import type { Unsubscribe } from './transport';

function frame(version: number, thinking?: 0 | 1): State {
  const i = (fixtureNames as readonly string[]).indexOf('state-play');
  const f = structuredClone(fixtures[i]) as State;
  delete f.ack;
  f.version = version;
  if (thinking !== undefined) f.thinking = thinking;
  else delete f.thinking;
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
    p.enqueue(frame(2)); // its move
    await settle();
    expect(c.sleeps).toEqual([500]);
    expect(p.frame?.version).toBe(2);

    // A move that comes after the dwell is shown at once.
    p.enqueue(frame(3, 1));
    await settle();
    c.advance(900);
    p.enqueue(frame(4));
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
