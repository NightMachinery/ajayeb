import { describe, expect, it } from 'vitest';
import type { HintFamily, View } from '$lib/wire';
import {
  MAX_TUTORIAL_ROOMS,
  NEW_TUTORIAL,
  NO_TIPS_SEEN,
  TIPS_KEY,
  TipsStore,
  dueTip,
  parseTips,
  TIP_IDS,
  TIPS,
  tipSelector,
  tipsInView,
  type StorageLike,
  type TipView
} from './tips.svelte';

function memory(initial: Record<string, string> = {}): StorageLike & { data: Map<string, string> } {
  const data = new Map(Object.entries(initial));
  return {
    data,
    getItem: (k) => data.get(k) ?? null,
    setItem: (k, v) => void data.set(k, v)
  };
}

const blocked: StorageLike = {
  getItem: () => {
    throw new Error('SecurityError');
  },
  setItem: () => {
    throw new Error('SecurityError');
  }
};

type Codes = [string[], string[]];

function view(
  slots: { state?: string; codes?: Codes }[],
  over = false
): Pick<View, 'structure' | 'result'> {
  return {
    structure: slots.map((s) => ({
      state: s.state ?? 'faceUp',
      ...(s.codes
        ? {
            hints: [
              { codes: s.codes[0], priceShift: [] },
              { codes: s.codes[1], priceShift: [] }
            ]
          }
        : {})
    })),
    ...(over ? { result: {} } : {})
  } as unknown as Pick<View, 'structure' | 'result'>;
}

const ALL = new Set<HintFamily>(['science', 'threats', 'military']);

describe('the tips data (ajayeb:tips)', () => {
  it('reads leniently', () => {
    expect(parseTips(null)).toEqual(NO_TIPS_SEEN);
    expect(parseTips('x')).toEqual(NO_TIPS_SEEN);
    expect(parseTips({ off: 'yes', seen: ['pair', 'bogus', 'pair', 3] })).toEqual({
      off: false,
      seen: ['pair'],
      tutorial: NEW_TUTORIAL
    });
    expect(parseTips({ off: true, seen: 'pair' })).toEqual({
      off: true,
      seen: [],
      tutorial: NEW_TUTORIAL
    });
  });

  it('remembers each tip seen, once, and survives a reload', () => {
    const storage = memory();
    const a = new TipsStore(storage);
    a.markSeen('pair');
    a.markSeen('pair');
    a.markSeen('loot');
    expect(a.value.seen).toEqual(['pair', 'loot']);
    const b = new TipsStore(storage);
    expect(b.value).toEqual({ off: false, seen: ['pair', 'loot'], tutorial: NEW_TUTORIAL });
    expect(JSON.parse(storage.data.get(TIPS_KEY)!)).toEqual({
      off: false,
      seen: ['pair', 'loot'],
      tutorial: NEW_TUTORIAL
    });
  });

  it('turns off and on, and shows every tip again after a reset', () => {
    const storage = memory({ [TIPS_KEY]: JSON.stringify({ off: false, seen: ['win'] }) });
    const s = new TipsStore(storage);
    s.setOff(true);
    expect(new TipsStore(storage).value).toEqual({
      off: true,
      seen: ['win'],
      tutorial: NEW_TUTORIAL
    });
    s.setOff(false);
    s.resetSeen();
    expect(new TipsStore(storage).value).toEqual(NO_TIPS_SEEN);
  });

  it('works for the page when storage is blocked', () => {
    const s = new TipsStore(blocked);
    expect(s.value).toEqual(NO_TIPS_SEEN);
    s.markSeen('win');
    expect(s.value.seen).toEqual(['win']);
  });
});

describe('the tutorial section (M10.tutorial-client)', () => {
  it('reads the old {off, seen} shape with a new tutorial', () => {
    const storage = memory({ [TIPS_KEY]: JSON.stringify({ off: true, seen: ['win'] }) });
    expect(new TipsStore(storage).value.tutorial).toEqual(NEW_TUTORIAL);
  });

  it('reads leniently: wrong fields take their defaults, bad rooms are dropped', () => {
    expect(parseTips({ tutorial: 'x' }).tutorial).toEqual(NEW_TUTORIAL);
    expect(
      parseTips({
        tutorial: {
          offer: 'declined',
          done: 'yes',
          rooms: { ABC: 'paused', DEF: 'completed', bad: 'paused', GHI: 'other', JKL: 3 }
        }
      }).tutorial
    ).toEqual({ offer: 'declined', done: false, rooms: { ABC: 'paused', DEF: 'completed' } });
    expect(parseTips({ tutorial: { offer: 'later', rooms: [] } }).tutorial).toEqual(NEW_TUTORIAL);
  });

  it('declines the offer, and Settings resets it with done', () => {
    const storage = memory();
    const s = new TipsStore(storage);
    s.declineOffer();
    expect(new TipsStore(storage).value.tutorial.offer).toBe('declined');
    s.markCompleted('ROOM1');
    expect(s.value.tutorial.done).toBe(true);
    s.resetOffer();
    expect(new TipsStore(storage).value.tutorial).toMatchObject({ offer: 'new', done: false });
  });

  it('pauses and resumes a room; a completed room stays completed', () => {
    const storage = memory();
    const s = new TipsStore(storage);
    s.pause('AB12');
    expect(new TipsStore(storage).isPaused('AB12')).toBe(true);
    s.resume('AB12');
    expect(s.isPaused('AB12')).toBe(false);
    expect(s.value.tutorial.rooms).toEqual({});
    s.markCompleted('AB12');
    s.pause('AB12');
    expect(s.isCompleted('AB12')).toBe(true);
    expect(s.isPaused('AB12')).toBe(false);
    expect(s.isPaused(null)).toBe(false);
  });

  it('keeps the newest rooms only', () => {
    const s = new TipsStore(memory());
    for (let i = 0; i < MAX_TUTORIAL_ROOMS + 5; i++) s.pause(`R${i}`);
    const codes = Object.keys(s.value.tutorial.rooms);
    expect(codes).toHaveLength(MAX_TUTORIAL_ROOMS);
    expect(codes[0]).toBe('R5');
    expect(codes.at(-1)).toBe(`R${MAX_TUTORIAL_ROOMS + 4}`);
  });
});

describe('which tip is due', () => {
  it('finds the marks the board shows for each side, in priority order', () => {
    const v = view([
      { codes: [['pair'], []] },
      { codes: [['loot'], ['win']] },
      { state: 'faceDown', codes: [['win'], []] }
    ]);
    expect(tipsInView(v, 0, ALL, true)).toEqual(['pair', 'winOpp', 'loot']);
    // The other seat sees the same card from its side.
    expect(tipsInView(v, 1, ALL, true)).toEqual(['win', 'denyPair']);
  });

  it('shows nothing without board marks, for a hidden family, or once the game is over', () => {
    const v = view([{ codes: [['pair'], ['win']] }]);
    expect(tipsInView(v, 0, ALL, false)).toEqual([]);
    expect(tipsInView(v, 0, new Set<HintFamily>(['threats']), true)).toEqual(['winOpp']);
    expect(tipsInView(view([{ codes: [['pair'], []] }], true), 0, ALL, true)).toEqual([]);
  });

  it('picks the first tip not seen yet, and none while tips are off', () => {
    expect(dueTip(['pair', 'loot'], { off: false, seen: [] })).toBe('pair');
    expect(dueTip(['pair', 'loot'], { off: false, seen: ['pair'] })).toBe('loot');
    expect(dueTip(['pair', 'loot'], { off: false, seen: ['pair', 'loot'] })).toBeNull();
    expect(dueTip(['pair'], { off: true, seen: [] })).toBeNull();
  });

  it("points at the board pip's attributes", () => {
    expect(tipSelector('denyPair')).toBe('[data-hint="pair"][data-hint-side="opp"]');
  });
});

describe('the tips beyond the pips (M10.F3)', () => {
  const FAMILIES = new Set<HintFamily>(['science', 'threats', 'military', 'wonders']);

  function full(over: Record<string, unknown>): TipView {
    return {
      structure: [],
      players: [
        { hints: { lawWins: false }, wonders: [] },
        { hints: { lawWins: false }, wonders: [] }
      ],
      boardTokens: [],
      ...over
    } as unknown as TipView;
  }

  it('the Law laurel, while Law is on the board and would be a sixth symbol', () => {
    const v = full({
      boardTokens: ['law'],
      players: [
        { hints: { lawWins: false }, wonders: [] },
        { hints: { lawWins: true }, wonders: [] }
      ]
    });
    expect(tipsInView(v, 0, FAMILIES, true)).toEqual(['lawLaurel']);
    expect(tipsInView(v, 0, FAMILIES, false)).toEqual([]);
    expect(tipsInView({ ...v, boardTokens: [] }, 0, FAMILIES, true)).toEqual([]);
    expect(tipSelector('lawLaurel')).toBe('.progress-token-row [data-hint="law"]');
  });

  it('a looting token within reach, pointing at the token a card would take', () => {
    const v = full({
      structure: [
        {
          state: 'faceUp',
          accessible: true,
          hints: [
            { codes: ['loot'], priceShift: [] },
            { codes: [], priceShift: [] }
          ]
        }
      ],
      hints: {
        looting: [
          { at: -3, wouldLose: 2 },
          { at: 3, wouldLose: 2 },
          { at: 6, wouldLose: 4 }
        ],
        wondersLeft: 3
      }
    });
    expect(tipsInView(v, 0, FAMILIES, true)).toEqual(['loot', 'ghostLoot']);
    // Seat 0's shields push toward seat 1's capital: the nearest token there.
    expect(tipSelector('ghostLoot', v, 0)).toBe('[data-loot-at="3"]');
    expect(tipsInView(v, 0, new Set<HintFamily>(['science']), true)).toEqual([]);
  });

  it('the hourglass once one wonder is left to build', () => {
    const v = full({
      hints: { looting: [], wondersLeft: 1 },
      players: [
        { hints: { lawWins: false }, wonders: [{ built: false, removed: false }] },
        { hints: { lawWins: false }, wonders: [{ built: true, removed: false }] }
      ]
    });
    expect(tipsInView(v, 1, FAMILIES, true)).toEqual(['hourglass']);
    expect(tipSelector('hourglass')).toBe('[data-hint="hourglass"], [data-hourglass]');
  });

  it('the chain badge and the guild back need no hint family nor board marks', () => {
    const v = full({
      structure: [
        { state: 'faceUp', accessible: true, costs: [{ chainFree: true }, { chainFree: false }] },
        { state: 'faceDown', back: 'guild' }
      ]
    });
    expect(tipsInView(v, 0, new Set<HintFamily>(), false)).toEqual(['freeChain', 'guildBack']);
    expect(tipsInView(v, 1, new Set<HintFamily>(), false)).toEqual(['guildBack']);
    expect(tipSelector('freeChain')).toBe('.card-structure [data-afford="chain"]');
    expect(tipSelector('guildBack')).toBe('.card-structure [data-back="guild"]');
  });

  it('the concept tips: the pawn, science, the starter and a new age', () => {
    const none = new Set<HintFamily>();
    const base = full({
      phase: 'play',
      age: 1,
      toMove: 0,
      military: { position: 0, looting: [] },
      players: [
        { wonders: [], science: { symbols: [], distinct: 0 } },
        { wonders: [], science: { symbols: [{ symbol: 'globe', count: 1 }], distinct: 1 } }
      ]
    });
    expect(tipsInView(base, 0, none, false)).toEqual([]);
    // The other seat's science counts for that seat only.
    expect(tipsInView(base, 1, none, false)).toEqual(['science']);
    expect(
      tipsInView({ ...base, military: { position: -1, looting: [] } }, 0, none, false)
    ).toEqual(['track']);
    expect(tipsInView({ ...base, phase: 'chooseStarter', age: 2 }, 0, none, false)).toEqual([
      'starter'
    ]);
    expect(tipsInView({ ...base, phase: 'chooseStarter', age: 2 }, 1, none, false)).toEqual([
      'science'
    ]);
    expect(tipsInView({ ...base, age: 2 }, 0, none, false)).toEqual(['newAge']);
    expect(tipsInView({ ...base, age: 3, result: {} } as TipView, 0, none, false)).toEqual([]);
    expect(tipSelector('track')).toBe('[data-tut="track"] [data-pawn]');
    expect(tipSelector('science')).toBe('[data-tut="home"] [data-science]');
    expect(tipSelector('starter')).toBe('[data-starter-choice]');
    expect(tipSelector('newAge')).toBe('[data-tut="age"]');
  });

  it('every tip names a rules section', () => {
    for (const id of TIP_IDS) expect(TIPS[id].rules).toBeTruthy();
  });
});
