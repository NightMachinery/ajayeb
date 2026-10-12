import { describe, expect, it } from 'vitest';
import type { GameEvent, HistoryEntry, ResourceCounts } from '$lib/wire';
import {
  EMPTY_RECAP,
  isCurrentRun,
  nextRecap,
  priceTicks,
  revealedSince,
  type RecapFrame,
  type RecapState
} from './recap';

function prices(p: Partial<ResourceCounts> = {}): ResourceCounts {
  return { wood: 2, clay: 2, stone: 2, glass: 2, papyrus: 2, ...p };
}

function ev(e: Partial<GameEvent> & Pick<GameEvent, 'kind'>): GameEvent {
  return { private: false, ...e };
}

function frame(
  version: number,
  p0: Partial<ResourceCounts>,
  p1: Partial<ResourceCounts>,
  events: GameEvent[] = [],
  more: { age?: number; code?: string; history?: HistoryEntry[] } = {}
): RecapFrame {
  return {
    code: more.code ?? 'ROOM',
    version,
    events,
    ...(more.history ? { history: more.history } : {}),
    view: {
      age: more.age ?? 1,
      players: [{ tradePrices: prices(p0) }, { tradePrices: prices(p1) }]
    } as unknown as RecapFrame['view']
  };
}

function run(...frames: RecapFrame[]): { state: RecapState; last: RecapFrame } {
  let state = EMPTY_RECAP;
  for (const f of frames) state = nextRecap(state, f);
  return { state, last: frames[frames.length - 1]! };
}

describe('price-changed-tick', () => {
  it("ticks the other seat's price, with the build that raised it", () => {
    const { state, last } = run(
      frame(1, {}, {}),
      frame(2, {}, { wood: 3 }, [ev({ kind: 'build', seat: 0, card: 'lumber-yard' })])
    );
    expect(priceTicks(state, last.view, 1)).toEqual([
      { resource: 'wood', from: 2, to: 3, cause: { seat: 0, card: 'lumber-yard' } }
    ]);
    // The mover's own move never ticks for the mover.
    expect(priceTicks(state, last.view, 0)).toEqual([]);
  });

  it('keeps a baseline per seat on a hot-seat device', () => {
    const { state, last } = run(
      frame(1, {}, {}),
      frame(2, {}, { wood: 3 }, [ev({ kind: 'build', seat: 0, card: 'lumber-yard' })]),
      frame(3, { clay: 3 }, { wood: 3 }, [ev({ kind: 'build', seat: 1, card: 'clay-pool' })])
    );
    // Seat 1 moved: its ticks ended with its move. Seat 0 sees the clay.
    expect(priceTicks(state, last.view, 1)).toEqual([]);
    expect(priceTicks(state, last.view, 0)).toEqual([
      { resource: 'clay', from: 2, to: 3, cause: { seat: 1, card: 'clay-pool' } }
    ]);
  });

  it('adds up the other side moves until the seat moves, and names no single cause for two', () => {
    const { state, last } = run(
      frame(1, {}, {}),
      frame(2, {}, { clay: 3 }, [ev({ kind: 'build', seat: 0, card: 'clay-pool' })]),
      frame(3, {}, { clay: 4, stone: 1 }, [ev({ kind: 'build', seat: 0, card: 'clay-pit' })])
    );
    expect(priceTicks(state, last.view, 1)).toEqual([
      { resource: 'clay', from: 2, to: 4 },
      { resource: 'stone', from: 2, to: 1 }
    ]);
  });

  it('restarts with no tick when an age begins', () => {
    const { state, last } = run(
      frame(1, {}, {}),
      frame(2, {}, { wood: 3 }, [ev({ kind: 'build', seat: 0, card: 'lumber-yard' })], { age: 2 })
    );
    expect(priceTicks(state, last.view, 1)).toEqual([]);
    expect(priceTicks(state, last.view, 0)).toEqual([]);
  });

  it('restarts after an undo or a resync, an old frame and another room', () => {
    const build = [ev({ kind: 'build', seat: 0, card: 'lumber-yard' })];
    for (const f of [
      frame(2, {}, { wood: 3 }, build, { history: [] }),
      frame(1, {}, { wood: 3 }, build),
      frame(2, {}, { wood: 3 }, build, { code: 'OTHER' })
    ]) {
      const { state, last } = run(frame(1, {}, {}), f);
      expect(priceTicks(state, last.view, 1)).toEqual([]);
    }
  });

  it('has nothing before the first frame', () => {
    expect(priceTicks(EMPTY_RECAP, frame(1, {}, {}).view, 0)).toEqual([]);
  });
});

describe('isCurrentRun', () => {
  const history: HistoryEntry[] = [
    { ply: 4, seat: 0, events: [] },
    { ply: 5, seat: 1, events: [] }
  ];
  it('holds for the summary of the moves since the viewer last moved', () => {
    expect(isCurrentRun({ seat: 1, plies: [5] }, history)).toBe(true);
    expect(isCurrentRun({ seat: 1, plies: [3] }, history)).toBe(false);
    expect(isCurrentRun({ seat: 0, plies: [4] }, history)).toBe(false);
    expect(isCurrentRun(null, history)).toBe(false);
    expect(isCurrentRun({ seat: 1, plies: [5] }, [])).toBe(false);
  });
});

describe('revealedSince', () => {
  it("lists the cards turned up since the seat's own last move", () => {
    const { state } = run(
      frame(1, {}, {}),
      frame(2, {}, {}, [
        ev({ kind: 'build', seat: 0, card: 'lumber-yard' }),
        ev({ kind: 'reveal', card: 'quarry' })
      ]),
      frame(3, {}, {}, [ev({ kind: 'reveal', card: 'press' })])
    );
    expect(revealedSince(state, 1)).toEqual(['quarry', 'press']);
    expect(revealedSince(state, 0)).toEqual(['press']);
  });
});
