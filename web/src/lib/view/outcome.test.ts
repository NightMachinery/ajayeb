import { describe, expect, it } from 'vitest';
import type { HintFamily, OutcomeView, ScoreView, SlotView } from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import { freedBy, freedSplit } from './freed';
import {
  afterPickLines,
  blunderLine,
  keyTargetName,
  openingCards,
  outcomeTerms,
  selectionGhost,
  starterThreatLines,
  type OutcomeContext
} from './outcome';
import type { HintNames } from './hints';

const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
const NAMES: HintNames = { names: ['Sun', 'Moon'], you: 0 };
const HOT: HintNames = { names: ['Sun', 'Moon'], you: null };

function ctx(over: Partial<OutcomeContext> = {}): OutcomeContext {
  return { mover: 0, coins: [7, 7], age: 1, families: ALL, names: NAMES, ...over };
}

function outcome(over: Partial<OutcomeView> = {}): OutcomeView {
  return { coins: [7, 7], score: [0, 0], ...over };
}

function score(total: number, blue = 0): ScoreView {
  return {
    blue,
    green: 0,
    yellow: 0,
    purple: 0,
    wonders: 0,
    tokens: 0,
    coins: 0,
    military: 0,
    total
  };
}

function slot(index: number, over: Partial<SlotView> = {}): SlotView {
  return {
    index,
    row: 1,
    col: 1,
    state: 'faceUp',
    accessible: false,
    coveredBy: [],
    ...over
  };
}

describe('freed', () => {
  // Slot 0 lies under 1 and 2; slot 3 only under 1; slot 4 under 2; slot 5 is face down under 1.
  const structure: SlotView[] = [
    slot(0, { card: 'baths', coveredBy: [1, 2] }),
    slot(1, { card: 'lumber-yard', accessible: true }),
    slot(2, { card: 'stone-pit', accessible: true }),
    slot(3, { card: 'tavern', coveredBy: [1] }),
    slot(4, { card: 'altar', coveredBy: [2] }),
    slot(5, { state: 'faceDown', coveredBy: [1] }),
    slot(6, { state: 'empty', coveredBy: [1] })
  ];

  it('frees the slots covered only by the card taken', () => {
    expect(freedBy(1, structure).map((s) => s.index)).toEqual([3, 5]);
    expect(freedBy(2, structure).map((s) => s.index)).toEqual([4]);
    expect(freedBy(3, structure)).toEqual([]);
  });

  it('splits them into face-up cards and a face-down count', () => {
    const f = freedSplit(1, structure);
    expect(f.faceUp.map((s) => s.card)).toEqual(['tavern']);
    expect(f.faceDown).toBe(1);
  });
});

describe('keyTargetName', () => {
  it('names the card, wonder or token of a key', () => {
    expect(keyTargetName('construct:baths')).toBe('Baths');
    expect(keyTargetName('wonder:pyramids:baths')).toBe('Pyramids');
    expect(keyTargetName('token:law')).toBe('Law');
    expect(keyTargetName('starter:1')).toBeUndefined();
  });
});

describe('outcomeTerms', () => {
  it('says nothing without an outcome', () => {
    expect(outcomeTerms(undefined, ctx())).toEqual([]);
  });

  it('drops terms that change nothing', () => {
    expect(outcomeTerms(outcome(), ctx())).toEqual([]);
  });

  it('puts a win first, with its crown, and leaves out the points', () => {
    const terms = outcomeTerms(
      outcome({ wins: { kind: 'military', steps: 1 }, coins: [4, 7], score: [-3, 0] }),
      ctx()
    );
    expect(terms[0]).toMatchObject({ id: 'win', icon: 'crown', tone: 'win' });
    expect(terms[0]!.text).toBe('Wins the game (military).');
    expect(terms.map((x) => x.id)).toEqual(['win', 'coins']);
  });

  it('words Law and two-step wins', () => {
    const law = outcomeTerms(
      outcome({ wins: { kind: 'science', steps: 1, via: 'token:law' } }),
      ctx()
    );
    expect(law[0]!.text).toContain('Law');
    const two = outcomeTerms(
      outcome({ wins: { kind: 'military', steps: 2, via: 'construct:arsenal' } }),
      ctx()
    );
    expect(two[0]!.text).toBe('Wins in the extra turn, with Arsenal.');
  });

  it('shows the coins after with their change, and points', () => {
    const terms = outcomeTerms(outcome({ coins: [4, 7], score: [3, 0] }), ctx());
    expect(terms.map((x) => x.id)).toEqual(['vp', 'coins']);
    expect(terms[0]!.short).toBe('+3');
    expect(terms[1]!.text).toBe('4 coins after (\u22123)');
  });

  it('moves the pawn and loots, without repeating the loss as coins', () => {
    const terms = outcomeTerms(
      outcome({
        coins: [5, 5],
        pawn: { from: 1, to: 3, loot: [{ at: 3, lost: 2 }] }
      }),
      ctx()
    );
    expect(terms.map((x) => x.id)).toEqual(['pawn', 'loot', 'coins']);
    expect(terms[0]!.text).toBe('Pawn 2 spaces toward Moon’s capital');
    expect(terms[0]!.short).toBe('+2');
    expect(terms[1]!.text).toBe('Moon loses 2 coins to looting');
  });

  it('speaks to seat 1 as the mover', () => {
    const terms = outcomeTerms(
      outcome({ coins: [7, 6], pawn: { from: 0, to: -1, loot: [] } }),
      ctx({ mover: 1, names: { names: ['Sun', 'Moon'], you: 1 } })
    );
    expect(terms[0]!.text).toBe('Pawn 1 space toward Sun’s capital');
    expect(terms[0]!.short).toBe('+1');
  });

  it('follows the families', () => {
    const o = outcome({
      coins: [4, 7],
      score: [2, 0],
      pawn: { from: 0, to: 1, loot: [] },
      wins: { kind: 'military', steps: 1 }
    });
    const none = ctx({ families: new Set() });
    expect(outcomeTerms(o, none)).toEqual([]);
    const military = ctx({ families: new Set(['military']) });
    expect(outcomeTerms(o, military).map((x) => x.id)).toEqual(['pawn']);
  });

  it('names the end of the game and its winner', () => {
    const end = outcome({
      end: {
        victory: 'civilian',
        winner: 1,
        shared: false,
        scores: [score(50), score(55)]
      }
    });
    expect(outcomeTerms(end, ctx())[0]!.text).toBe('Ends the game: Moon wins on points, 55 to 50.');
    expect(outcomeTerms(end, ctx())[0]!.tone).toBe('warn');
  });

  it('lets the end on points carry a civilian win', () => {
    const end = outcome({
      wins: { kind: 'civilian', steps: 1 },
      end: { victory: 'civilian', winner: 0, shared: false, scores: [score(60), score(50)] }
    });
    const terms = outcomeTerms(end, ctx());
    expect(terms.map((x) => x.id)).toEqual(['end']);
    expect(terms[0]).toMatchObject({ icon: 'crown', tone: 'win' });
    expect(terms[0]!.text).toBe('Ends the game: you win on points, 60 to 50.');
    const threatsOnly = outcomeTerms(end, ctx({ families: new Set(['threats']) }));
    expect(threatsOnly.map((x) => x.text)).toEqual(['Wins the game (points).']);
  });

  it('says the turn order and the choice it opens', () => {
    const terms = outcomeTerms(
      outcome({
        tempo: { extraTurn: 'none', source: 'none', lastCard: true, starterChooser: 1 },
        opens: 'pickBoardToken'
      }),
      ctx()
    );
    expect(terms.map((x) => x.text)).toEqual([
      'Moon chooses who starts Age 2',
      'Then pick a progress token'
    ]);
    const again = outcomeTerms(
      outcome({ tempo: { extraTurn: 'gain', source: 'wonder', lastCard: false } }),
      ctx()
    );
    expect(again.map((x) => x.id)).toEqual(['playAgain']);
  });
});

describe('blunderLine', () => {
  const win = {
    key: 'construct:arsenal',
    outcome: outcome({ wins: { kind: 'military', steps: 1 } })
  };
  const plain = { key: 'construct:baths', outcome: outcome() };
  const gift = {
    key: 'discard:baths',
    outcome: outcome({ gives: [{ seat: 1, card: 'arsenal', kind: 'military' }] })
  };

  it('warns about a gift', () => {
    expect(blunderLine(gift, [gift], ctx())).toBe('This hands Moon a win: Arsenal.');
  });

  it('warns about a missed win', () => {
    expect(blunderLine(plain, [plain, win], ctx())).toBe(
      'You could win this turn instead: Arsenal.'
    );
    expect(blunderLine(plain, [plain, win], ctx({ names: HOT }))).toBe(
      'Sun could win this turn instead: Arsenal.'
    );
  });

  it('is quiet for the win itself, without threats, or without wins', () => {
    expect(blunderLine(win, [plain, win], ctx())).toBeUndefined();
    expect(blunderLine(plain, [plain], ctx())).toBeUndefined();
    expect(
      blunderLine(plain, [plain, win], ctx({ families: new Set(['scoring']) }))
    ).toBeUndefined();
  });
});

describe('afterPickLines', () => {
  const structure: SlotView[] = [
    slot(0, {
      card: 'arsenal',
      coveredBy: [2],
      hints: [
        { codes: [], priceShift: [] },
        { codes: ['win'], priceShift: [] }
      ]
    }),
    slot(1, { state: 'faceDown', coveredBy: [2] }),
    slot(2, { card: 'baths', accessible: true })
  ];

  it('lists what it uncovers, the reveal and the gift', () => {
    const lines = afterPickLines(structure[2]!, structure, [], ctx());
    expect(lines.map((l) => l.id)).toEqual(['frees', 'reveal', 'gift-0']);
    expect(lines[0]!.text).toBe('Uncovers 2 cards: Arsenal and 1 face-down card.');
    expect(lines[2]!.text).toBe('Arsenal would then win the game for Moon.');
  });

  it('says once a threat both moves share, and labels ones that differ', () => {
    const gives = outcome({ gives: [{ seat: 1, card: 'arsenal', kind: 'military' }] });
    const same = afterPickLines(
      structure[2]!,
      structure,
      [
        { label: 'Build', outcome: gives },
        { label: 'Discard', outcome: gives }
      ],
      ctx({ families: new Set(['threats']) })
    );
    expect(same).toEqual([
      { id: 'gives', text: 'Moon could then win with Arsenal.', threat: true }
    ]);
    const differ = afterPickLines(
      structure[2]!,
      structure,
      [
        { label: 'Build', outcome: outcome() },
        { label: 'Discard', outcome: gives }
      ],
      ctx({ families: new Set(['threats']) })
    );
    expect(differ.map((l) => l.text)).toEqual(['Discard: Moon could then win with Arsenal.']);
  });

  it('gives the reveal risk as odds', () => {
    const lines = afterPickLines(
      structure[2]!,
      structure,
      [{ label: 'Build', outcome: outcome({ revealRisk: { num: 1, den: 4, freed: 1 } }) }],
      ctx({ families: new Set(['threats']) })
    );
    expect(lines.map((l) => l.text)).toEqual([
      'A 1 in 4 chance the card turned over lets Moon win.'
    ]);
  });
});

describe('starter and ghosts', () => {
  it('lists the starter threats in seat order', () => {
    const lines = starterThreatLines(
      [
        {
          kind: 'starter',
          starter: 1,
          outcome: outcome({ threatens: [{ seat: 1, card: 'arsenal', kind: 'military' }] })
        },
        { kind: 'starter', starter: 0, outcome: outcome() }
      ],
      ALL,
      NAMES
    );
    expect(lines).toEqual([
      { seat: 1, text: 'If Moon starts, they can win at once with Arsenal.' }
    ]);
  });

  it('lists the cards an age opens with', () => {
    expect(
      openingCards([
        slot(0, { card: 'baths', accessible: true }),
        slot(1, { card: 'altar' }),
        slot(2, { state: 'faceDown', accessible: true })
      ])
    ).toEqual(['baths']);
  });

  it('builds the selection ghost from the outcome', () => {
    const o = outcome({ coins: [3, 9], score: [2, -1], pawn: { from: 0, to: 2, loot: [] } });
    expect(selectionGhost(o, [10, 12], ALL)).toEqual({
      pawn: o.pawn,
      coins: [3, 9],
      score: [12, 11]
    });
    expect(selectionGhost(o, [10, 12], new Set(['military']))).toEqual({ pawn: o.pawn });
    expect(selectionGhost(o, [10, 12], new Set())).toBeUndefined();
  });
});
