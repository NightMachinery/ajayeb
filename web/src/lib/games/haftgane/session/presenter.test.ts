// The Haftgane Presenter over synthetic frames: changes play in order, and
// frames that restate the game (a history, a seat change, a skipped
// backlog) or only refresh it are shown without playing anything; a change
// without events reaches only the still hook.
import { describe, expect, it, vi } from 'vitest';
import type { TableEvent, TableState } from '$lib/wire/platform.gen';
import { TablePresenter, cosmetic, restates } from './presenter.svelte';
import { tableFrame } from './test-frames';

const event: TableEvent = {
  kind: 'build',
  other: 0,
  turn: 1,
  side: '',
  printed: 0,
  delta: 0,
  source: '',
  outcome: '',
  seat: 1,
  play: 'build',
  design: 'basalt',
  stage: 0,
  left: 0,
  right: 0,
  age: 1,
  pass: 'left'
};

function resolution(batch: number): TableState {
  return tableFrame({ batch, revision: (batch + 1) << 20, events: [event] });
}

async function settle(p: TablePresenter): Promise<void> {
  for (let i = 0; i < 20 && !p.idle; i++) await Promise.resolve();
  await new Promise((r) => setTimeout(r, 0));
}

function presenter(fastForwardAfter = 2) {
  const played: TableState[] = [];
  const p = new TablePresenter(null, {
    afterSwap: () => Promise.resolve(),
    fastForwardAfter,
    hooks: { play: (next) => void played.push(next) }
  });
  return { p, played };
}

describe('restates and cosmetic', () => {
  it('tells a restating frame from a change and a refresh', () => {
    const base = tableFrame();
    expect(restates(null, base)).toBe(true);
    expect(restates(base, tableFrame({ history: [] }))).toBe(true);
    expect(restates(base, tableFrame({}, null))).toBe(true);
    expect(restates(base, resolution(1))).toBe(false);
    expect(cosmetic(base, tableFrame())).toBe(true);
    expect(cosmetic(base, tableFrame({ revision: 2 << 20 }))).toBe(false);
    expect(cosmetic(base, resolution(0))).toBe(false);
  });
});

describe('TablePresenter', () => {
  it('shows frames in order and plays only changes with events', async () => {
    const { p, played } = presenter();
    const first = tableFrame({ history: [] });
    p.enqueue(first);
    await settle(p);
    expect(p.frame).toBe(first);
    expect(played).toEqual([]);
    const next = resolution(1);
    p.enqueue(next);
    await settle(p);
    expect(p.frame).toBe(next);
    expect(p.shownEvents).toEqual([event]);
    expect(played).toEqual([next]);
    expect(p.idle).toBe(true);
  });

  it('replaces the view whole, without playing, on a history frame or a seat change', async () => {
    const { p, played } = presenter();
    p.enqueue(tableFrame({ history: [] }));
    await settle(p);
    const history = [{ batch: 1, events: [event] }];
    p.enqueue(tableFrame({ batch: 1, revision: 2 << 20, events: [event], history }, null));
    await settle(p);
    expect(p.frame?.you.seat).toBeUndefined();
    expect(p.shownEvents).toEqual([]);
    // A seat change whose history was merged away still restates.
    p.enqueue(tableFrame({ batch: 2, revision: 3 << 20, events: [event] }, 1));
    await settle(p);
    expect(p.frame?.you.seat).toBe(1);
    expect(p.shownEvents).toEqual([]);
    expect(played).toEqual([]);
  });

  it('refreshes in place: a presence change keeps the queued move and its events', async () => {
    let release: () => void = () => {};
    const played: TableState[] = [];
    const p = new TablePresenter(null, {
      afterSwap: () => Promise.resolve(),
      hooks: {
        play: (next) => {
          played.push(next);
          return new Promise<void>((r) => (release = r));
        }
      }
    });
    p.enqueue(tableFrame({ history: [] }));
    await settle(p);
    const move = resolution(1);
    p.enqueue(move);
    await vi.waitFor(() => expect(played).toHaveLength(1));
    // Held by the play hook; the next move and its refresh queue behind it.
    const next = resolution(2);
    p.enqueue(next);
    const refresh = {
      ...next,
      events: [],
      humans: next.humans.map((h) => ({ ...h, connected: false }))
    };
    p.enqueue(refresh);
    expect(p.queued).toBe(1);
    release();
    await vi.waitFor(() => expect(played).toHaveLength(2));
    release();
    await settle(p);
    expect(p.frame).toBe(refresh);
    expect(played[1]).toBe(next);
  });

  it('jumps a backlog to the newest frame and shows it without playing', async () => {
    const { p, played } = presenter(2);
    let release: () => void = () => {};
    p.hooks.play = (next) => {
      played.push(next);
      return new Promise<void>((r) => (release = r));
    };
    p.enqueue(tableFrame({ history: [] }));
    await settle(p);
    p.enqueue(resolution(1));
    await vi.waitFor(() => expect(played).toHaveLength(1));
    p.enqueue(resolution(2));
    p.enqueue(resolution(3));
    const newest = resolution(4);
    p.enqueue(newest);
    expect(p.queued).toBe(1);
    release();
    await settle(p);
    expect(p.frame).toBe(newest);
    expect(p.shownEvents).toEqual([]);
    expect(played).toHaveLength(1);
  });

  it('tells the still hook of a change without events, never of a restatement or a refresh', async () => {
    const { p, played } = presenter();
    const still: [TableState, TableState | null][] = [];
    p.hooks.still = (next, prev) => void still.push([next, prev]);
    const first = tableFrame({ history: [] });
    p.enqueue(first);
    await settle(p);
    // A presence refresh: the same revision and no events.
    p.enqueue({ ...first, history: undefined });
    await settle(p);
    expect(still).toEqual([]);
    // Another seat confirms: a new revision and no events.
    const confirmed = tableFrame({ revision: 2 << 20 });
    p.enqueue(confirmed);
    await settle(p);
    expect(still).toEqual([[confirmed, { ...first, history: undefined }]]);
    // A change with events plays instead; a history frame does neither.
    p.enqueue(resolution(2));
    p.enqueue(tableFrame({ revision: 4 << 20, history: [] }));
    await settle(p);
    expect(still).toHaveLength(1);
    expect(played).toHaveLength(1);
  });

  it('forgets everything on clear', async () => {
    const { p } = presenter();
    p.enqueue(tableFrame({ history: [] }));
    await settle(p);
    p.clear();
    expect(p.frame).toBeNull();
    expect(p.idle).toBe(true);
  });
});
