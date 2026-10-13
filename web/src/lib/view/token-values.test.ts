import { describe, expect, it } from 'vitest';
import type { HintFamily, PendingView, TokenValueView, View } from '$lib/wire';
import { HINT_FAMILIES } from '$lib/hints/families';
import type { HintScreen } from './hints';
import { tokenPickLines } from './token-values';

const ALL: ReadonlySet<HintFamily> = new Set(HINT_FAMILIES);
const SCREEN: HintScreen = {
  perspective: { kind: 'seat', seat: 1 },
  families: ALL,
  boardMarks: true,
  names: { names: ['Sun', 'Moon'], you: 1 }
};

function value(token: string, seat: number, over: Partial<TokenValueView> = {}): TokenValueView {
  return { token, seat, coins: 0, vp: 0, wins: false, savings: 0, count: 0, ...over };
}

function view(
  tokens: TokenValueView[],
  pending?: PendingView
): Pick<View, 'players' | 'hints' | 'pending'> {
  return {
    players: [{}, {}] as unknown as View['players'],
    hints: {
      discard: [],
      wondersLeft: 7,
      pairs: [],
      looting: [],
      tokens,
      boxTokens: [],
      guildPreview: [],
      unseen: []
    },
    ...(pending ? { pending } : {})
  };
}

const library = (tokenValues: TokenValueView[]): PendingView => ({
  kind: 'pickLibraryToken',
  seat: 1,
  cards: [],
  tokens: ['agriculture', 'law', 'urbanism'],
  hiddenTokens: 0,
  tokenValues
});

describe('tokenPickLines', () => {
  it("a board pick shows the decider's value, then the other seat's", () => {
    const v = view([
      value('agriculture', 0, { coins: 6, vp: 4 }),
      value('agriculture', 1, { coins: 6, vp: 3 })
    ]);
    expect(tokenPickLines(v, 'agriculture', false, SCREEN)).toEqual([
      'For you: +6 coins and +3 VP.',
      'For Sun: +6 coins and +4 VP.'
    ]);
  });

  it("a Great Library pick shows the decider's own values only", () => {
    const v = view(
      [],
      library([value('law', 1, { wins: true }), value('urbanism', 1, { count: 2 })])
    );
    expect(tokenPickLines(v, 'law', true, SCREEN)).toEqual(['For you: wins the game.']);
    expect(tokenPickLines(v, 'urbanism', true, SCREEN)).toEqual(['For you: 2 chains still open.']);
    expect(tokenPickLines(v, 'agriculture', true, SCREEN)).toEqual([]);
  });

  it('shows nothing with the tokens family hidden or without a hint screen', () => {
    const v = view([value('agriculture', 1, { coins: 6 })]);
    const hidden: HintScreen = { ...SCREEN, families: new Set<HintFamily>(['threats']) };
    expect(tokenPickLines(v, 'agriculture', false, hidden)).toEqual([]);
    expect(tokenPickLines(v, 'agriculture', false, undefined)).toEqual([]);
  });

  it('names the decider on a shared hot-seat device', () => {
    const hotseat: HintScreen = {
      ...SCREEN,
      perspective: { kind: 'hotseat', seat: 1 },
      names: { names: ['Sara', 'Omid'], you: null }
    };
    const v = view([], library([value('law', 1, { vp: 3 })]));
    expect(tokenPickLines(v, 'law', true, hotseat)).toEqual(['For Omid: +3 VP.']);
  });
});
