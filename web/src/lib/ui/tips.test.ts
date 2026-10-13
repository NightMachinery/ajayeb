import { describe, expect, it } from 'vitest';
import type { HintFamily, View } from '$lib/wire';
import {
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
      seen: ['pair']
    });
    expect(parseTips({ off: true, seen: 'pair' })).toEqual({ off: true, seen: [] });
  });

  it('remembers each tip seen, once, and survives a reload', () => {
    const storage = memory();
    const a = new TipsStore(storage);
    a.markSeen('pair');
    a.markSeen('pair');
    a.markSeen('loot');
    expect(a.value.seen).toEqual(['pair', 'loot']);
    const b = new TipsStore(storage);
    expect(b.value).toEqual({ off: false, seen: ['pair', 'loot'] });
    expect(JSON.parse(storage.data.get(TIPS_KEY)!)).toEqual({ off: false, seen: ['pair', 'loot'] });
  });

  it('turns off and on, and shows every tip again after a reset', () => {
    const storage = memory({ [TIPS_KEY]: JSON.stringify({ off: false, seen: ['win'] }) });
    const s = new TipsStore(storage);
    s.setOff(true);
    expect(new TipsStore(storage).value).toEqual({ off: true, seen: ['win'] });
    s.setOff(false);
    s.resetSeen();
    expect(new TipsStore(storage).value).toEqual({ off: false, seen: [] });
  });

  it('works for the page when storage is blocked', () => {
    const s = new TipsStore(blocked);
    expect(s.value).toEqual(NO_TIPS_SEEN);
    s.markSeen('win');
    expect(s.value.seen).toEqual(['win']);
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

  it('every tip names a rules section', () => {
    for (const id of TIP_IDS) expect(TIPS[id].rules).toBeTruthy();
  });
});
