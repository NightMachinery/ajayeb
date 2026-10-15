import { describe, expect, it } from 'vitest';
import type { CardHints, CostView, HintFamily, SlotView, View } from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import {
  PIP_PRIORITY,
  cardPip,
  cardSections,
  denyNotes,
  denyPip,
  destroySections,
  discardNotes,
  discardSections,
  hintScreen,
  perspectiveOf,
  pipLook,
  pipTooltip,
  sessionHintScreen,
  shortByDiscard,
  type HintNames,
  type Perspective
} from './hints';

const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
const NAMES: HintNames = { names: ['Sun', 'Moon'], you: 0 };
const SEAT0: Perspective = { kind: 'seat', seat: 0 };
const SEAT1: Perspective = { kind: 'seat', seat: 1 };

function hints(over: Partial<CardHints> = {}): CardHints {
  return {
    codes: [],
    effect: { paid: true, coins: 0, scoreDelta: 0, oppScoreDelta: 0 },
    priceShift: [],
    ...over
  };
}

const PAIR = hints({
  codes: ['pair'],
  effect: {
    paid: true,
    coins: 0,
    scoreDelta: 0,
    oppScoreDelta: 0,
    science: { symbol: 'quill', kind: 'pair', distinctAfter: 3, tokenOnBoard: true }
  }
});
const WIN = hints({
  codes: ['win'],
  effect: {
    paid: true,
    coins: 0,
    scoreDelta: 0,
    oppScoreDelta: 0,
    wins: { kind: 'military', steps: 1 }
  }
});
const LOOT = hints({
  codes: ['loot'],
  effect: {
    paid: true,
    coins: 0,
    scoreDelta: 2,
    oppScoreDelta: -1,
    military: { from: 2, to: 3, loot: [{ at: 3, lost: 2 }] }
  }
});

describe('the pip priority table', () => {
  it('ranks own win, opponent win, own pair, opponent pair, own loot, opponent loot', () => {
    expect(PIP_PRIORITY.map(([c, s]) => `${c}-${s}`)).toEqual([
      'win-own',
      'win-opp',
      'pair-own',
      'pair-opp',
      'loot-own',
      'loot-opp'
    ]);
  });

  it('a crown beats a pair, on either side', () => {
    const pip = cardPip([PAIR, WIN], SEAT0, ALL, NAMES)!;
    expect(pip.top.code).toBe('win');
    expect(pip.top.side).toBe('opp');
    expect(pip.entries.map((e) => e.code)).toEqual(['win', 'pair']);
  });

  it('an own pair beats the opponent pair and loot', () => {
    const both: CardHints = { ...LOOT, codes: ['loot'] };
    const pip = cardPip([PAIR, both], SEAT0, ALL, NAMES)!;
    expect(pip.top.code).toBe('pair');
    expect(pip.top.text).toContain('quill');
    const flipped = cardPip([both, PAIR], SEAT0, ALL, NAMES)!;
    expect(flipped.top).toMatchObject({ code: 'pair', side: 'opp' });
    expect(flipped.entries.map((e) => `${e.code}-${e.side}`)).toEqual(['pair-opp', 'loot-own']);
  });

  it('a second hint adds the hairline more dot; a single one does not', () => {
    expect(cardPip([PAIR, WIN], SEAT0, ALL, NAMES)!.more).toBe(true);
    expect(cardPip([PAIR, hints()], SEAT0, ALL, NAMES)!.more).toBe(false);
    expect(cardPip([hints(), hints()], SEAT0, ALL, NAMES)).toBeUndefined();
    expect(cardPip(undefined, SEAT0, ALL, NAMES)).toBeUndefined();
  });

  it('the tooltip lists every hint in priority order', () => {
    const pip = cardPip([PAIR, WIN], SEAT0, ALL, NAMES)!;
    expect(pipTooltip(pip).split('\n')).toEqual([
      'Moon wins by building this.',
      'Completes your quill pair: you pick a progress token.'
    ]);
  });
});

describe('solid and outlined by viewer', () => {
  it("a seat's own pair is solid gold, the other seat's outlined in its crest", () => {
    const pip0 = cardPip([PAIR, hints()], SEAT0, ALL, NAMES)!;
    expect(pip0.top).toMatchObject({ side: 'own', style: 'solid', tone: 'gold', seat: 0 });
    const pip1 = cardPip([PAIR, hints()], SEAT1, ALL, { names: ['Sun', 'Moon'], you: 1 })!;
    expect(pip1.top).toMatchObject({ side: 'opp', style: 'outline', tone: 'sun', seat: 0 });
    expect(pip1.top.text).toBe('Completes Sun’s quill pair: they would pick a progress token.');
  });

  it('hot-seat: solid follows the seat to move, in its crest tint', () => {
    const p: Perspective = { kind: 'hotseat', seat: 1 };
    expect(pipLook(p, 1)).toEqual({ style: 'solid', tone: 'moon' });
    expect(pipLook(p, 0)).toEqual({ style: 'outline', tone: 'sun' });
    const pip = cardPip([hints(), PAIR], p, ALL, { names: ['Ana', 'Bo'], you: null })!;
    expect(pip.top).toMatchObject({ side: 'own', style: 'solid', tone: 'moon' });
    expect(pip.top.text).toBe('Completes Bo’s quill pair: they would pick a progress token.');
  });

  it('spectators see each seat in its crest tint, never solid or outlined', () => {
    const p: Perspective = { kind: 'spectator', seat: 0 };
    const pip = cardPip([PAIR, WIN], p, ALL, { names: ['Ana', 'Bo'], you: null })!;
    expect(pip.entries.map((e) => [e.style, e.tone])).toEqual([
      ['tint', 'moon'],
      ['tint', 'sun']
    ]);
  });

  it('loot speaks to the loser on their own screen', () => {
    const pip = cardPip([hints(), LOOT], SEAT0, ALL, NAMES)!;
    expect(pip.top).toMatchObject({ code: 'loot', side: 'opp' });
    expect(pip.top.text).toBe('You lose 2 coins if Moon builds this.');
    const own = cardPip([LOOT, hints()], SEAT0, ALL, NAMES)!;
    expect(own.top.text).toBe('Moon loses 2 coins if you build this.');
  });
});

describe('families', () => {
  it('a family switched off, or hidden here, shows no pip', () => {
    const noScience = new Set(HINT_FAMILIES.filter((f) => f !== 'science'));
    expect(cardPip([PAIR, hints()], SEAT0, noScience, NAMES)).toBeUndefined();
    const noThreats = new Set(HINT_FAMILIES.filter((f) => f !== 'threats'));
    expect(cardPip([PAIR, WIN], SEAT0, noThreats, NAMES)!.entries.map((e) => e.code)).toEqual([
      'pair'
    ]);
  });

  it('the session screen follows the room settings, the local hide, and the result', () => {
    const view = { viewer: 1, active: 1, result: undefined } as unknown as View;
    const settings = Object.fromEntries(HINT_FAMILIES.map((f) => [f, f !== 'military'])) as never;
    const s = sessionHintScreen(
      { view, room: { mode: 'online', seats: [] }, hints: settings },
      { families: ['economy'], boardMarks: true }
    )!;
    expect(s.perspective).toEqual({ kind: 'seat', seat: 1 });
    expect(s.names.you).toBe(1);
    expect(s.families.has('military')).toBe(false);
    expect(s.families.has('economy')).toBe(false);
    expect(s.families.has('science')).toBe(true);
    expect(s.boardMarks).toBe(false);
    const over = { ...view, result: { victory: 'civilian' } } as unknown as View;
    expect(
      sessionHintScreen(
        { view: over, room: null, hints: settings },
        { families: [], boardMarks: false }
      )
    ).toBeUndefined();
  });

  it('the perspective: a seat, a hot-seat device, a spectator', () => {
    expect(perspectiveOf({ viewer: 0, active: 1 }, 'ai')).toEqual({ kind: 'seat', seat: 0 });
    expect(perspectiveOf({ viewer: 1, active: 1 }, 'hotseat')).toEqual({
      kind: 'hotseat',
      seat: 1
    });
    expect(perspectiveOf({ active: 1 }, 'online')).toEqual({ kind: 'spectator', seat: 1 });
  });

  it('wonder-first mode keeps only the opponent pips', () => {
    const pip = cardPip([PAIR, WIN], SEAT0, ALL, NAMES);
    expect(denyPip(pip)!.entries.map((e) => e.code)).toEqual(['win']);
    expect(denyPip(cardPip([PAIR, hints()], SEAT0, ALL, NAMES))).toBeUndefined();
  });
});

describe('the sheet lines', () => {
  it('lists the own section first, then the opponent, each only with lines', () => {
    const own = hints({
      ...PAIR,
      priceShift: [{ seat: 1, resource: 'clay', from: 3, to: 4 }],
      savesInView: 0
    });
    const sections = cardSections('library', [own, LOOT], SEAT0, ALL, NAMES);
    expect(sections.map((s) => [s.title, s.side, s.style])).toEqual([
      ['For you', 'own', 'solid'],
      ['For Moon', 'opp', 'outline']
    ]);
    expect(sections[0]!.lines.map((l) => l.text)).toEqual([
      'Completes the quill pair: pick a progress token.',
      'Moon’s clay: 3 to 4 coins per unit.',
      'Saves nothing in view right now.'
    ]);
    expect(sections[1]!.lines.map((l) => l.text)).toEqual([
      'Pawn 1 space toward your capital.',
      'Looting: you lose 2 coins.'
    ]);
  });

  it('a pair with no token left, a new symbol, a short win, the last producer', () => {
    const noToken = hints({
      effect: {
        paid: true,
        coins: 0,
        scoreDelta: 0,
        oppScoreDelta: 0,
        science: { symbol: 'wheel', kind: 'pair', distinctAfter: 2, tokenOnBoard: false }
      }
    });
    const fresh = hints({
      effect: {
        paid: false,
        coins: 0,
        scoreDelta: 0,
        oppScoreDelta: 0,
        science: { symbol: 'globe', kind: 'new', distinctAfter: 4, tokenOnBoard: true },
        wins: { kind: 'science', steps: 1 }
      }
    });
    const [a, b] = cardSections('school', [noToken, fresh], SEAT0, ALL, NAMES);
    expect(a!.lines.map((l) => l.text)).toEqual([
      'Completes the wheel pair, but no tokens are left.'
    ]);
    expect(b!.lines.map((l) => l.text)).toEqual([
      'It would win the game (science), but cannot be paid for now.',
      'A new symbol, globe: 4 different of 6.'
    ]);
    const last = cardSections(
      'glass-blower',
      [hints({ lastProducer: 'lastCertain' }), hints({ lastProducer: 'lastVisible' })],
      SEAT0,
      ALL,
      NAMES
    );
    expect(last.map((s) => s.lines[0]!.text)).toEqual([
      'No other glass producer can still come.',
      'The last glass producer in view; unseen cards may hold another.'
    ]);
  });

  it('worth now on yellow and purple cards only', () => {
    const worth = hints({
      effect: { paid: true, coins: 4, scoreDelta: 1, oppScoreDelta: 0 }
    });
    expect(cardSections('tavern', [worth, hints()], SEAT0, ALL, NAMES)[0]!.lines[0]!.text).toBe(
      'Worth now: 4 coins, 1 VP.'
    );
    expect(cardSections('baths', [worth, hints()], SEAT0, ALL, NAMES)).toEqual([]);
  });

  it('a card with no effect sends only its other facts', () => {
    const bare: CardHints = { codes: [], priceShift: [], lastProducer: 'lastCertain' };
    expect(
      cardSections('glass-blower', [bare, { codes: [], priceShift: [] }], SEAT0, ALL, NAMES)
    ).toEqual([
      expect.objectContaining({ seat: 0, lines: [expect.objectContaining({ id: 'last' })] })
    ]);
    expect(
      cardSections(
        'tavern',
        [
          { codes: [], priceShift: [] },
          { codes: [], priceShift: [] }
        ],
        SEAT0,
        ALL,
        NAMES
      )
    ).toEqual([]);
  });

  it('a family switched off drops its lines', () => {
    const none = new Set<HintFamily>();
    expect(cardSections('library', [PAIR, LOOT], SEAT0, none, NAMES)).toEqual([]);
  });

  it('short by discard, only when a discard covers the gap', () => {
    expect(shortByDiscard(2, 3, ALL)).toBe(
      'A discard (+3 coins) would cover it at today’s prices.'
    );
    expect(shortByDiscard(4, 3, ALL)).toBeUndefined();
    expect(shortByDiscard(2, 3, new Set())).toBeUndefined();
  });
});

function cost(over: Partial<CostView>): CostView {
  return {
    affordable: true,
    shortfall: 0,
    chainFree: false,
    printedCoins: 0,
    own: { wood: 0, clay: 0, stone: 0, glass: 0, papyrus: 0 },
    producers: [],
    trade: [],
    tradeTotal: 0,
    total: 0,
    toOpponent: 0,
    urbanismBonus: 0,
    ...over
  };
}

function player(wonders: View['players'][number]['wonders']): View['players'][number] {
  return { wonders } as View['players'][number];
}

describe('the notes beside the actions', () => {
  const slot = {
    index: 3,
    state: 'faceUp',
    card: 'library',
    costs: [cost({ affordable: false, total: 7, shortfall: 2 }), cost({ chainFree: true })]
  } as unknown as SlotView;
  const noMausoleum = { players: [player([]), player([])] as View['players'] };
  const oppMausoleum = {
    players: [
      player([]),
      player([{ wonder: 'mausoleum', built: false, removed: false, cost: cost({}) }])
    ] as View['players']
  };

  it('discarding keeps a chain-free card from the opponent', () => {
    expect(discardNotes(slot, noMausoleum, 0, ALL, NAMES).map((n) => n.text)).toEqual([
      'Also keeps it from Moon (free for them by chain).'
    ]);
  });

  it('with the opponent’s Mausoleum open, only for now', () => {
    expect(discardNotes(slot, oppMausoleum, 0, ALL, NAMES).map((n) => n.text)).toEqual([
      'Keeps it from Moon for now (free for them by chain): their Mausoleum could still rebuild it.'
    ]);
    const plain = { ...slot, costs: [cost({}), cost({})] } as unknown as SlotView;
    expect(discardNotes(plain, oppMausoleum, 0, ALL, NAMES).map((n) => n.id)).toEqual([
      'mausoleum'
    ]);
    expect(discardNotes(plain, oppMausoleum, 1, ALL, NAMES).map((n) => n.id)).toEqual([
      'mausoleumOwn'
    ]);
  });

  it('a deny target gets build and tuck notes', () => {
    expect(denyNotes([hints(), PAIR], 0, ALL, NAMES)).toEqual({
      build: 'Building it takes it from Moon.',
      wonder: 'Tucked under a wonder, it is out of the game for Moon.'
    });
    expect(denyNotes([PAIR, hints()], 0, ALL, NAMES)).toEqual({});
  });
});

describe('the Mausoleum pool and the destroy choice', () => {
  const screen = hintScreen({ viewer: 0, active: 0 }, 'online', ALL, true, ['Sun', 'Moon']);

  it('lists each discarded card with something to say, with its worth', () => {
    const view = {
      hints: {
        discard: [
          { card: 'library', seat: 0, hints: PAIR, total: 0, chainFree: true },
          { card: 'baths', seat: 0, hints: hints(), total: 4, chainFree: false },
          { card: 'garrison', seat: 1, hints: LOOT, total: 2, chainFree: false }
        ]
      }
    } as unknown as View;
    const sections = discardSections(view, ['baths', 'library', 'garrison'], 0, screen);
    expect(sections.map((s) => [s.title, s.lines.map((l) => l.text)])).toEqual([
      ['Baths', ['Worth 4 coins to build today.']],
      ['Library', ['Completes the quill pair: pick a progress token.']]
    ]);
  });

  it('a free rebuild that wins is never "cannot be paid for now"', () => {
    // The server's rebuild preview pays nothing, so effect.paid is false.
    const rebuildWin = hints({ ...WIN, effect: { ...WIN.effect!, paid: false } });
    const view = {
      hints: {
        discard: [{ card: 'garrison', seat: 0, hints: rebuildWin, total: 2, chainFree: false }]
      }
    } as unknown as View;
    const [section] = discardSections(view, ['garrison'], 0, screen);
    expect(section?.lines[0]?.text).toBe('Building it wins the game (military).');
  });

  it('names what losing each candidate changes', () => {
    const pending = {
      kind: 'destroyCard',
      seat: 0,
      cards: ['clay-pool', 'quarry'],
      tokens: [],
      hiddenTokens: 0,
      cardHints: [
        {
          card: 'clay-pool',
          priceShift: [{ seat: 0, resource: 'clay', from: 4, to: 3 }],
          savesInView: 2,
          onlySource: false
        },
        { card: 'quarry', priceShift: [], savesInView: 0, onlySource: true }
      ]
    } as unknown as View['pending'];
    const sections = destroySections(pending, ['clay-pool', 'quarry'], screen);
    expect(sections.map((s) => s.lines.map((l) => l.text))).toEqual([
      ['Your clay: 4 to 3 coins per unit.', 'Saves 2 coins on the cards and wonders in view, now.'],
      ['Saves nothing in view right now.', 'Moon’s only stone.']
    ]);
  });
});
