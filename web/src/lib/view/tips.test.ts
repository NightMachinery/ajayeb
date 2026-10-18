import { describe, expect, it } from 'vitest';
import type { CostView, HintFamily, HintsView, PlayerHintsView, ResourceCounts } from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import type { HintScreen } from './hints';
import {
  affordTip,
  backTip,
  cardTip,
  chainLines,
  coinsLines,
  costMoreTip,
  costPipTip,
  effectRules,
  effectTip,
  guildLines,
  payLines,
  previewText,
  scoreLines,
  sentences
} from './tips';

const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
const NONE: ReadonlySet<HintFamily> = new Set();

function screen(families: ReadonlySet<HintFamily> = ALL, you: 0 | 1 | null = 0): HintScreen {
  return {
    perspective: you === null ? { kind: 'spectator' } : { kind: 'seat', seat: you },
    families,
    boardMarks: true,
    names: { names: ['Sun', 'Moon'], you }
  } as HintScreen;
}

const ZERO: ResourceCounts = { wood: 0, clay: 0, stone: 0, glass: 0, papyrus: 0 };

function cost(over: Partial<CostView> = {}): CostView {
  return {
    affordable: true,
    shortfall: 0,
    chainFree: false,
    printedCoins: 0,
    own: ZERO,
    producers: [],
    trade: [],
    tradeTotal: 0,
    total: 0,
    toOpponent: 0,
    urbanismBonus: 0,
    ...over
  };
}

function ph(over: Partial<PlayerHintsView> = {}): PlayerHintsView {
  return {
    coinsToNextVp: 2,
    symbols: [],
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

describe('sentences', () => {
  it('adds the missing full stops and drops empty parts', () => {
    expect(sentences(['3 of 6', '', 'Close.', ' far '])).toBe('3 of 6. Close. far.');
  });
});

describe('cost pips', () => {
  it('words coins and resources, with no rules link', () => {
    expect(costPipTip({ kind: 'coins', n: 2 }).text).toBe('Costs 2 coins, paid to the bank.');
    const stone = costPipTip({ kind: 'res', resource: 'stone', n: 2 });
    expect(stone.text).toBe('Needs 2 stone.');
    expect(stone.lines).toEqual([]);
    expect(stone.rulesId).toBeUndefined();
  });

  it('says how the payment in view covers a resource', () => {
    const pay = cost({
      own: { ...ZERO, stone: 1 },
      producers: [{ card: 'forum', resource: 'stone' }],
      discount: { token: 'masonry', units: { ...ZERO, stone: 1 } },
      trade: [{ resource: 'stone', units: 1, unitPrice: 3, fixed: false, coins: 3 }]
    });
    expect(payLines('stone', pay)).toEqual([
      'Made in the city: 1.',
      'From a yellow card or a wonder: 1.',
      'Waived by Masonry: 1.',
      'Bought from the bank: 1 at 3 coins each.'
    ]);
    expect(payLines('glass', pay)).toEqual([]);
    expect(payLines('stone', cost({ chainFree: true, own: { ...ZERO, stone: 2 } }))).toEqual([]);
    expect(payLines('stone', undefined)).toEqual([]);
  });

  it('lists what a "+n" hides', () => {
    expect(
      costMoreTip([
        { kind: 'res', resource: 'glass', n: 1 },
        { kind: 'res', resource: 'papyrus', n: 2 }
      ]).text
    ).toBe('Also needs 1 glass and 2 papyrus.');
  });
});

describe('effects', () => {
  it('uses the printed sentence', () => {
    expect(effectTip({ kind: 'shields', n: 2 }).text).toBe(
      '2 shields: move the conflict pawn 2 spaces.'
    );
  });

  it('adds the paired points of a guild glyph', () => {
    const tip = effectTip({ kind: 'coins-per-card', n: 1, colors: ['yellow'], scope: 'max' }, 1);
    expect(tip.text.split('. ')).toHaveLength(2);
    expect(tip.rulesId).toBe('guilds');
  });

  it('links the rules only for a concept', () => {
    expect(effectRules({ kind: 'shields', n: 1 })).toBe('military');
    expect(effectRules({ kind: 'science', symbol: 'globe' })).toBe('science');
    expect(effectRules({ kind: 'trade-fix', resources: ['wood'] })).toBe('costs');
    expect(effectRules({ kind: 'produce', resource: 'wood', n: 1 })).toBeUndefined();
    expect(effectRules({ kind: 'points', n: 3 })).toBeUndefined();
  });
});

describe('the affordability badge', () => {
  it('says why for each state', () => {
    const base = { coins: 0, shortfall: 0 };
    expect(affordTip({ state: 'chain', ...base, chainFrom: 'baths' }).text).toContain('Baths');
    expect(affordTip({ state: 'chain', ...base }).text).toBe('Free through a chain symbol.');
    expect(affordTip({ state: 'free', ...base }).rulesId).toBeUndefined();
    expect(affordTip({ state: 'own', coins: 2, shortfall: 0 }).text).toContain('Pay 2 coins');
    expect(affordTip({ state: 'short', coins: 7, shortfall: 3 }).text).toBe(
      'Costs 7 coins: 3 coins short.'
    );
  });

  it('lists what a trade buys, from the breakdown', () => {
    const tip = affordTip(
      { state: 'trade', coins: 5, shortfall: 0 },
      cost({
        trade: [
          { resource: 'clay', units: 1, unitPrice: 3, fixed: false, coins: 3 },
          { resource: 'wood', units: 2, unitPrice: 1, fixed: true, coins: 2 }
        ]
      })
    );
    expect(tip.text).toContain('5 coins in all');
    expect(tip.lines).toEqual(['1 clay at 3 coins each', '2 wood at 1 coin each']);
    expect(tip.rulesId).toBe('costs');
  });
});

describe('a face-down back', () => {
  it('says only which back and when it turns face up', () => {
    expect(backTip('age2')).toEqual({
      text: 'A face-down Age II card.',
      lines: ['Turns face up when the cards covering it are taken.']
    });
    expect(backTip('guild')).toEqual({
      text: 'A face-down guild card.',
      lines: ['Turns face up when the cards covering it are taken.']
    });
  });
});

describe('the tallies', () => {
  it('says the coins to the next point, under the scoring family', () => {
    expect(coinsLines(ph(), screen())).toEqual(['2 more coins for the next point from coins.']);
    expect(coinsLines(ph(), screen(NONE))).toEqual([]);
    expect(coinsLines(ph({ coinsToNextVp: 0 }), screen())).toEqual([]);
    expect(coinsLines(undefined, screen())).toEqual([]);
  });

  it('says who would win on points now', () => {
    const names = screen().names;
    expect(previewText({ winner: 0, by: 'total', margin: 3 }, names)).toBe(
      'You would win on points now, by 3.'
    );
    expect(previewText({ winner: 1, by: 'blue', margin: 0 }, names)).toBe(
      'Tied on points now: Moon would win on blue points.'
    );
    expect(previewText({ by: 'shared', margin: 0 }, names)).toContain('shared victory');
    expect(
      scoreLines(hv({ previewWinner: { winner: 1, by: 'total', margin: 2 } }), screen())
    ).toEqual(['Moon would win on points now, by 2.']);
    expect(scoreLines(hv(), screen())).toEqual([]);
    expect(
      scoreLines(hv({ previewWinner: { winner: 1, by: 'total', margin: 2 } }), screen(NONE))
    ).toEqual([]);
  });
});

describe('chains and guilds', () => {
  const players = [
    {
      hints: ph({
        chainTargets: [{ chain: 'drop', target: 'aqueduct', place: 'faceUp', worth: 5 }]
      })
    },
    { hints: ph() }
  ];

  it('gives the chain status for the seat that shows the symbol', () => {
    expect(chainLines('drop', players, screen())).toEqual([
      'You show this symbol: Aqueduct is free for you. It is face up on the board.',
      'The chain saves 5 coins today.'
    ]);
    expect(chainLines('drop', players, screen(ALL, 1))[0]).toBe(
      'Sun shows this symbol: Aqueduct is free for Sun. It is face up on the board.'
    );
    expect(chainLines('moon', players, screen())).toEqual([]);
    expect(chainLines('drop', players, screen(NONE))).toEqual([]);
  });

  it('gives each owned guild’s points now', () => {
    expect(guildLines([{ card: 'builders-guild', vp: 4, countsSeat: 1 }], screen())).toEqual([
      'Builders Guild: 4 points now, counting Moon’s city.'
    ]);
    expect(guildLines([{ card: 'builders-guild', vp: 1, countsSeat: 0 }], screen())).toEqual([
      'Builders Guild: 1 point now, counting your city.'
    ]);
    expect(guildLines([{ card: 'builders-guild', vp: 4, countsSeat: 1 }], screen(NONE))).toEqual(
      []
    );
  });

  it('names a mini card and its category', () => {
    expect(cardTip('baths').text).toBe('Baths: Civilian building, Age I.');
  });
});
