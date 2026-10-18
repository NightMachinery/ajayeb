import { afterEach, describe, expect, it } from 'vitest';
import { i18n, stripBidiMarks } from '$lib/i18n';
import type {
  HintFamily,
  HintsView,
  PlayerHintsView,
  SymbolHint,
  View,
  WonderView
} from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import {
  ageTip,
  capitalText,
  deadPairLines,
  effectOutcome,
  lootText,
  placeText,
  scienceHints,
  tokenHint,
  trackHints,
  wonderButtonHint,
  wonderHint,
  wonderSheetLines
} from './board-hints';
import type { HintNames, HintScreen } from './hints';
import { seatGhost } from './outcome';

const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
const NAMES: HintNames = { names: ['Sun', 'Moon'], you: 0 };
const SCREEN: HintScreen = {
  perspective: { kind: 'seat', seat: 0 },
  families: ALL,
  boardMarks: true,
  names: NAMES
};

const SYMBOLS = ['globe', 'scales', 'sundial', 'mortar', 'set-square', 'quill', 'wheel'] as const;

function ph(over: Partial<PlayerHintsView> = {}): PlayerHintsView {
  return {
    coinsToNextVp: 3,
    symbols: SYMBOLS.map((symbol): SymbolHint => ({ symbol, state: 'open', openCards: 2 })),
    need: 6,
    reachable: true,
    lawWins: false,
    chainTargets: [],
    guildPoints: [],
    toCapital: 9,
    capitalSafe: false,
    threats: [],
    ...over
  };
}

function hv(over: Partial<HintsView> = {}): HintsView {
  return {
    discard: [],
    wondersLeft: 7,
    pairs: [],
    looting: [],
    tokens: [],
    boxTokens: [],
    guildPreview: [],
    unseen: [],
    ...over
  };
}

function wonder(slug: string, over: Partial<WonderView> = {}): WonderView {
  return { wonder: slug, built: false, removed: false, ...over };
}

type Players = View['players'];

function view(
  hints: HintsView,
  a: PlayerHintsView | undefined,
  b: PlayerHintsView | undefined,
  over: Partial<{ wonders: [WonderView[], WonderView[]]; coins: [number, number] }> = {}
): Pick<View, 'players' | 'hints' | 'age' | 'phase' | 'structure'> {
  const player = (seat: 0 | 1, h: PlayerHintsView | undefined) =>
    ({
      coins: over.coins?.[seat] ?? 7,
      wonders: over.wonders?.[seat] ?? [],
      ...(h ? { hints: h } : {})
    }) as unknown as Players[0];
  return {
    players: [player(0, a), player(1, b)] as unknown as Players,
    hints,
    age: 2,
    phase: 'play',
    structure: []
  };
}

describe('scienceHints (the tracker states)', () => {
  it('maps each symbol to its state, with the partner card and its place in the tooltip', () => {
    const own = ph({
      symbols: [
        { symbol: 'globe', state: 'paired', openCards: 0 },
        { symbol: 'scales', state: 'dead', openCards: 0 },
        { symbol: 'sundial', state: 'live', openCards: 1 },
        { symbol: 'mortar', state: 'owned', openCards: 0 },
        { symbol: 'set-square', state: 'open', openCards: 2 },
        { symbol: 'quill', state: 'dead', openCards: 0 },
        { symbol: 'wheel', state: 'open', openCards: 1 }
      ],
      need: 3,
      reachable: false
    });
    const v = view(
      hv({
        pairs: [
          {
            symbol: 'quill',
            cards: [
              { card: 'scriptorium', place: 'tucked', seat: 1 },
              { card: 'library', place: 'city', seat: 1 }
            ]
          },
          {
            symbol: 'sundial',
            cards: [
              { card: 'apothecary', place: 'city', seat: 0 },
              {
                card: 'dispensary',
                place: 'unseenLaterAge',
                inPlay: { num: 20, den: 23, freed: 0 }
              }
            ]
          }
        ]
      }),
      own,
      ph()
    );
    const sh = scienceHints(v, 0, ALL, NAMES)!;
    expect(sh.slots.globe?.state).toBe('paired');
    expect(sh.slots.sundial?.state).toBe('live');
    expect(sh.slots.mortar?.state).toBe('owned');
    expect(sh.slots['set-square']?.state).toBe('open');
    expect(sh.slots.quill?.state).toBe('dead');
    expect(sh.slots.quill?.tip).toContain('tucked under a wonder');
    expect(sh.slots.quill?.tip).toContain('in Moon’s city');
    expect(sh.slots.sundial?.tip).toContain('in your city');
    expect(sh.slots.sundial?.tip).toContain('20 in 23');
    expect(sh.reach).toBe('You can no longer win by science.');
  });

  it('is absent without the science family or the seat hints', () => {
    const v = view(hv(), ph(), undefined);
    expect(scienceHints(v, 0, new Set(['military']), NAMES)).toBeUndefined();
    expect(scienceHints(v, 1, ALL, NAMES)).toBeUndefined();
  });

  it('words what a reachable seat still needs, and the Law mark', () => {
    const own = ph({
      symbols: SYMBOLS.map((symbol) => ({
        symbol,
        state: symbol === 'wheel' ? 'open' : 'owned',
        openCards: 1
      })),
      need: 1,
      reachable: true,
      lawWins: true
    });
    const sh = scienceHints(view(hv(), ph(), own), 1, ALL, NAMES)!;
    expect(sh.reach).toContain('Moon needs 1 more different symbol');
    expect(sh.reach).toContain('Law would be Moon’s sixth symbol');
  });
});

describe('placeText', () => {
  it('words every public place', () => {
    expect(placeText('baths', { place: 'faceUp' }, NAMES)).toBe('face up on the board');
    expect(placeText('baths', { place: 'discard' }, NAMES)).toBe('in the discard pile');
    expect(placeText('baths', { place: 'removedKnown' }, NAMES)).toBe('set aside, out of the game');
    expect(placeText('baths', { place: 'unseenThisAge' }, NAMES)).toBe('not seen yet this age');
  });
});

describe('trackHints and lootText', () => {
  it('says how far each capital is, and the number only when close and not safe', () => {
    const v = view(
      hv({ looting: [{ at: -6, wouldLose: 3 }] }),
      ph({ toCapital: 4 }),
      ph({ toCapital: 2, capitalSafe: true })
    );
    const th = trackHints(v, ALL, NAMES)!;
    expect(th.capitals[0]?.numeral).toBe(true);
    expect(th.capitals[0]?.text).toBe('Moon needs 4 more shields to take your capital.');
    expect(th.capitals[1]?.numeral).toBe(false);
    expect(th.capitals[1]?.text).toContain('You can no longer gather that many shields.');
    expect(th.wouldLose.get(-6)).toBe(3);
    expect(trackHints(v, new Set(['science']), NAMES)).toBeUndefined();
  });

  it('words a looting token, clamped to the coins its victim has', () => {
    const loot = { at: -6, seat: 0, value: 5, present: true };
    expect(lootText(loot, 3, NAMES)).toBe(
      'When the pawn enters this space, you lose 5 coins. You have only 3 coins now.'
    );
    expect(lootText({ ...loot, seat: 1, at: 6 }, 5, NAMES)).toBe(
      'When the pawn enters this space, Moon loses 5 coins.'
    );
    expect(lootText({ ...loot, present: false }, undefined, NAMES)).toContain('Already taken');
  });
});

describe('wonder hints', () => {
  const effect = {
    paid: true,
    coins: [-4, -3] as [number, number],
    score: [3, 0] as [number, number],
    extraTurn: 'wonder' as const,
    removes: 'pyramids',
    pool: 0
  };

  it('turns a wonder effect into an outcome', () => {
    expect(effectOutcome(effect, [10, 5], 0)).toEqual({
      coins: [6, 2],
      score: [3, 0],
      tempo: { extraTurn: 'gain', source: 'wonder', lastCard: false }
    });
  });

  it("leaves out the owner's coins and points when it cannot afford the wonder now", () => {
    // The preview paid nothing, so "coins after" would overstate them; the
    // other seat's loss does not depend on the payment.
    const unpaid = {
      ...effect,
      paid: false,
      coins: [3, -3] as [number, number],
      score: [4, -1] as [number, number]
    };
    expect(effectOutcome(unpaid, [9, 4], 0)).toEqual({
      coins: [9, 1],
      score: [0, -1],
      tempo: { extraTurn: 'gain', source: 'wonder', lastCard: false }
    });
    expect(effectOutcome(unpaid, [9, 4], 1).coins).toEqual([12, 4]);
  });

  it('draws the hourglass once one wonder is left, solid when its owner can afford it', () => {
    const appian = wonder('appian-way', {
      cost: { affordable: true } as WonderView['cost'],
      effect
    });
    const v = view(hv({ wondersLeft: 1 }), ph(), ph(), {
      wonders: [[appian], [wonder('pyramids')]]
    });
    const h = wonderHint(v as View, appian, 0, SCREEN)!;
    expect(h.hourglass?.solid).toBe(true);
    expect(h.hourglass?.tip).toContain('Only one more wonder can be built.');
    expect(h.hourglass?.tip).toContain('Pyramids');
    expect(h.terms.map((x) => x.id)).toContain('playAgain');
    // Affordable but no choice builds it: the ready mark, solid for this screen's seat.
    expect(h.ready?.style).toBe('solid');
    // The decider's own buildable wonder has no ready mark.
    expect(wonderHint(v as View, appian, 0, SCREEN, true)?.ready).toBeUndefined();
    const button = wonderButtonHint(v as View, 0, SCREEN)!;
    expect(button.hourglass).toBe(true);
    expect(button.ready).toBe(1);
  });

  it('has no hourglass earlier, and no marks with the board marks hidden', () => {
    const w = wonder('appian-way', { cost: { affordable: true } as WonderView['cost'], effect });
    const v = view(hv({ wondersLeft: 3 }), ph(), ph(), { wonders: [[w], []] });
    expect(wonderHint(v as View, w, 0, SCREEN)?.hourglass).toBeUndefined();
    const hidden = { ...SCREEN, boardMarks: false };
    const v1 = view(hv({ wondersLeft: 1 }), ph(), ph(), { wonders: [[w], []] });
    const h = wonderHint(v1 as View, w, 0, hidden)!;
    expect(h.hourglass).toBeUndefined();
    expect(h.ready).toBeUndefined();
    expect(wonderHint(v1 as View, w, 0, { ...SCREEN, families: new Set() })).toBeUndefined();
  });

  it('says how many slots are left near the end of the race', () => {
    const v = view(hv({ wondersLeft: 2 }), ph(), ph(), {
      wonders: [[wonder('a'), wonder('b', { built: true })], [wonder('c')]]
    });
    expect(wonderSheetLines(v as View, ALL)).toEqual(['2 wonder slots left, 2 wonders unbuilt.']);
  });
});

describe('tokenHint', () => {
  it('gives each seat its value, own first, and the Law laurel', () => {
    const v = view(
      hv({
        tokens: [
          { token: 'law', seat: 0, coins: 0, vp: 0, wins: false, savings: 0, count: 0 },
          { token: 'law', seat: 1, coins: 0, vp: 0, wins: true, savings: 0, count: 0 },
          { token: 'agriculture', seat: 0, coins: 6, vp: 4, wins: false, savings: 0, count: 0 }
        ]
      }),
      ph(),
      ph({ lawWins: true })
    );
    const law = tokenHint(v, 'law', SCREEN)!;
    expect(law.lines).toEqual(['For you: nothing now.', 'For Moon: wins the game.']);
    expect(law.mark?.style).toBe('outline');
    expect(law.mark?.tone).toBe('moon');
    expect(tokenHint(v, 'agriculture', SCREEN)?.lines).toEqual(['For you: +6 coins and +4 VP.']);
  });
});

describe('ageTip and deadPairLines', () => {
  it('says who takes the last card and who chooses the next starter', () => {
    const v = view(
      hv({
        lastCardTo: 1,
        nextStarter: { chooser: 0, byLastCard: false, lastActiveStarts: false }
      }),
      ph(),
      ph()
    );
    expect(ageTip(v, 2, ALL, NAMES)).toBe(
      'If nobody plays again, Moon takes the last card. An extra turn taken with the last card is lost. You are the weaker player and will choose who starts Age III.'
    );
    expect(ageTip(v, 2, new Set(), NAMES)).toBeUndefined();
  });

  it('names a pair the ended age killed', () => {
    const v = view(
      hv({
        pairs: [
          {
            symbol: 'quill',
            cards: [
              { card: 'scriptorium', place: 'city', seat: 1 },
              { card: 'library', place: 'removedKnown' }
            ]
          }
        ]
      }),
      ph(),
      ph()
    );
    expect(deadPairLines(v, 2, ALL, NAMES)).toEqual([
      'Library was set aside: Moon’s Scriptorium will never make a pair.'
    ]);
    expect(deadPairLines(v, 1, ALL, NAMES)).toEqual([]);
  });
});

describe('seatGhost (the panels of a selection ghost)', () => {
  it('maps the ghost to each seat', () => {
    const g = { coins: [3, 7] as const, score: [12, 9] as const };
    expect(seatGhost(g, 0)).toEqual({ coins: 3, score: 12 });
    expect(seatGhost(g, 1)).toEqual({ coins: 7, score: 9 });
    expect(seatGhost({ pawn: { from: 0, to: 2, loot: [] } }, 0)).toBeUndefined();
    expect(seatGhost(undefined, 0)).toBeUndefined();
  });
});

describe('capital winner wording', () => {
  afterEach(() => i18n.set('en'));
  it('names the opposite winner for both players and spectators, without optional hints', () => {
    const names = ['Sara', 'Omid'] as const;
    expect(capitalText(0, { names, you: 0 })).toBe(
      'Your capital. If the conflict pawn reaches it, Omid wins at once.'
    );
    expect(capitalText(1, { names, you: 0 })).toBe(
      'Omid’s capital. If the conflict pawn reaches it, you win at once.'
    );
    expect(capitalText(1, { names, you: 1 })).toBe(
      'Your capital. If the conflict pawn reaches it, Sara wins at once.'
    );
    expect(capitalText(0, { names, you: 1 })).toBe(
      'Sara’s capital. If the conflict pawn reaches it, you win at once.'
    );
    expect(capitalText(0, { names, you: null })).toBe(
      'Sara’s capital. If the conflict pawn reaches it, Omid wins at once.'
    );
    expect(capitalText(1, { names, you: null })).toBe(
      'Omid’s capital. If the conflict pawn reaches it, Sara wins at once.'
    );
  });
  it('keeps Latin winner names isolated in Persian', async () => {
    await i18n.set('fa');
    const text = capitalText(0, { names: ['Sara', 'Omid'], you: 0 });
    expect(text).toContain('\u2068Omid\u2069');
    expect(stripBidiMarks(text)).toBe(
      'پایتخت شما. اگر مهره‌ی نظامی به اینجا برسد، Omid بی‌درنگ برنده می‌شود.'
    );
  });
});
