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
  placeTip,
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
    ).toEqual({
      offer: 'declined',
      done: false,
      rooms: { ABC: 'paused', DEF: 'completed' },
      roomOrder: ['ABC', 'DEF']
    });
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

  it('keeps numeric room codes in recency order across reloads and evicts the oldest', () => {
    const storage = memory();
    let s = new TipsStore(storage);
    s.pause('OLD');
    for (let i = 100; i > 100 - MAX_TUTORIAL_ROOMS; i--) {
      s.pause(String(i));
      s = new TipsStore(storage);
    }
    expect(s.value.tutorial.rooms['OLD']).toBeUndefined();
    expect(s.value.tutorial.roomOrder[0]).toBe('100');
    s.markCompleted('100');
    s.pause('NEW');
    s = new TipsStore(storage);
    expect(s.value.tutorial.rooms['99']).toBeUndefined();
    expect(s.value.tutorial.rooms['100']).toBe('completed');
    expect(s.value.tutorial.roomOrder.slice(-2)).toEqual(['100', 'NEW']);
    s.resume('98');
    expect(s.value.tutorial.roomOrder).not.toContain('98');
    expect(Object.keys(s.value.tutorial.rooms)).toHaveLength(MAX_TUTORIAL_ROOMS - 1);
  });

  it('migrates old room maps and repairs malformed explicit orders', () => {
    const rooms = { ABC: 'paused', '20': 'completed', DEF: 'paused' };
    expect(parseTips({ tutorial: { rooms } }).tutorial.roomOrder).toEqual(['20', 'ABC', 'DEF']);
    const migrated = parseTips({
      tutorial: { rooms, roomOrder: ['DEF', '20', 'DEF', null, 'NOPE'] }
    });
    expect(migrated.tutorial.roomOrder).toEqual(['DEF', '20', 'ABC']);
    expect(parseTips(JSON.parse(JSON.stringify(migrated)))).toEqual(migrated);
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

  it('the concept tips: the pawn, science and a new age', () => {
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
    // None at the starter choice (a modal dialog): the new age's tip waits for play.
    expect(tipsInView({ ...base, phase: 'chooseStarter', age: 2 }, 0, none, false)).toEqual([]);
    expect(tipsInView({ ...base, phase: 'chooseStarter', age: 2 }, 1, none, false)).toEqual([
      'science'
    ]);
    expect(tipsInView({ ...base, age: 2 }, 0, none, false)).toEqual(['newAge']);
    expect(tipsInView({ ...base, age: 3 }, 0, none, false)).not.toContain('newAge');
    expect(
      tipsInView(
        { ...base, age: 2, structure: [{ state: 'empty' }, { state: 'empty' }] } as TipView,
        0,
        none,
        false
      )
    ).not.toContain('newAge');
    expect(tipsInView({ ...base, age: 3, result: {} } as TipView, 0, none, false)).toEqual([]);
    // Late in a game (3/6 or 5/6 symbols, Age III, or after the pawn's opening move)
    // the concept tips do not replay out of context.
    expect(
      tipsInView(
        full({
          ...base,
          players: [
            { wonders: [], science: { symbols: [], distinct: 0 } },
            {
              wonders: [],
              science: {
                symbols: [
                  { symbol: 'globe', count: 1 },
                  { symbol: 'wheel', count: 1 },
                  { symbol: 'sundial', count: 1 }
                ],
                distinct: 3
              }
            }
          ]
        }),
        1,
        none,
        false
      )
    ).not.toContain('science');
    expect(tipsInView({ ...base, age: 3 }, 1, none, false)).not.toContain('science');
    expect(
      tipsInView({ ...base, age: 2, military: { position: -1, looting: [] } }, 0, none, false)
    ).not.toContain('track');
    expect(
      tipsInView({ ...base, military: { position: -4, looting: [] } }, 0, none, false)
    ).not.toContain('track');
    expect(
      tipsInView(
        {
          ...base,
          military: {
            position: -1,
            looting: [{ at: -3, seat: 0, value: 2, present: false }]
          }
        },
        0,
        none,
        false
      )
    ).not.toContain('track');
    expect(tipSelector('track')).toBe('[data-tut="track"] [data-pawn]');
    expect(tipSelector('science')).toBe('[data-tut="home"] [data-science]');
    expect(tipSelector('newAge')).toBe('[data-tut="age"]');
  });

  it('places a bottom-of-card tip (freeChain) outside the card face', () => {
    const card = { left: 160, top: 520, right: 230, bottom: 615, width: 70, height: 95 };
    const mark = { left: 175, top: 594, right: 215, bottom: 612, width: 40, height: 18 };
    // Room below the card on 390x844: sits below the card.
    const below = placeTip({ mark, card, panelHeight: 160, vw: 390, vh: 844 });
    expect(below.side).toBe('below');
    expect(below.top).toBeGreaterThanOrEqual(card.bottom + 8);
    // Card near the bottom edge so below does not fit: sits above the card's top edge,
    // never over the card face.
    const lowCard = { left: 160, top: 610, right: 230, bottom: 705, width: 70, height: 95 };
    const lowMark = { left: 175, top: 684, right: 215, bottom: 702, width: 40, height: 18 };
    const above = placeTip({ mark: lowMark, card: lowCard, panelHeight: 160, vw: 390, vh: 844 });
    expect(above.side).toBe('above');
    expect(above.top + 160).toBeLessThanOrEqual(lowCard.top - 8);
    // Fallback when neither vertical nor horizontal space fits: top stays clamped to the viewport.
    const cramped = placeTip({ mark, card, panelHeight: 220, vw: 320, vh: 260 });
    expect(cramped.top).toBeGreaterThanOrEqual(8);
    expect(cramped.top + 220).toBeLessThanOrEqual(260 - 8);
  });

  it('moves a tip off the cards to take and the strips, keeping its arrow on the mark', () => {
    // 740x360 landscape: a pair mark on Sawmill in the structure, the
    // opponent's strip to the right, another card to take above.
    const card = { left: 400, top: 200, right: 460, bottom: 280, width: 60, height: 80 };
    const mark = { left: 420, top: 205, right: 440, bottom: 221, width: 20, height: 16 };
    const strip = { left: 560, top: 40, right: 740, bottom: 160 };
    const other = { left: 330, top: 40, right: 390, bottom: 120 };
    const plain = placeTip({ mark, card, panelHeight: 150, vw: 740, vh: 360 });
    const overlap = (p: { left: number; top: number; width: number }, o: typeof strip) =>
      Math.min(p.left + p.width, o.right) > Math.max(p.left, o.left) && // physical-ok
      Math.min(p.top + 150, o.bottom) > Math.max(p.top, o.top);
    expect(overlap(plain, strip) || overlap(plain, other)).toBe(true);
    const moved = placeTip({
      mark,
      card,
      panelHeight: 150,
      vw: 740,
      vh: 360,
      obstacles: [strip, other]
    });
    expect(overlap(moved, strip)).toBe(false);
    expect(overlap(moved, other)).toBe(false);
    expect(overlap(moved, card)).toBe(false);
    if (moved.side !== 'beside') {
      expect(moved.left + moved.arrow + 6).toBeCloseTo(mark.left + mark.width / 2, 0);
    }
  });

  it('keeps the usual place when it covers no obstacle', () => {
    const card = { left: 160, top: 520, right: 230, bottom: 615, width: 70, height: 95 };
    const mark = { left: 175, top: 524, right: 215, bottom: 540, width: 40, height: 16 };
    const opts = { mark, card, panelHeight: 160, vw: 390, vh: 844 };
    expect(
      placeTip({ ...opts, obstacles: [{ left: 0, top: 700, right: 390, bottom: 844 }] })
    ).toEqual(placeTip(opts));
  });

  it('every tip names a rules section', () => {
    for (const id of TIP_IDS) expect(TIPS[id].rules).toBeTruthy();
  });
});
