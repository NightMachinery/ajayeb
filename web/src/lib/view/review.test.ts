import { afterEach, describe, expect, it } from 'vitest';
import { i18n, stripBidiMarks } from '$lib/i18n';
import type { HistoryEntry, ReviewView } from '$lib/wire';
import type { MoveSummary } from './move-summary';
import { moveText, playedText, reviewForSummary, reviewLines, verdictText } from './review';

const names = { names: ['Sun', 'Moon'] as const, seat: 0 as const };

afterEach(async () => {
  await i18n.set('en');
});

function review(r: Partial<ReviewView>): ReviewView {
  return { ply: 10, played: 'discard:walls', reason: 'none', gap: 0, strength: 'quick', ...r };
}

function entry(ply: number, seat: 0 | 1): HistoryEntry {
  return { ply, seat, events: [] };
}

function summary(seat: 0 | 1, plies: number[]): MoveSummary {
  return {
    seat,
    plies,
    actions: [],
    coins: [0, 0],
    paid: [0, 0],
    paidToOpponent: [0, 0],
    pawn: null,
    loot: []
  } as unknown as MoveSummary;
}

describe('the review wording', () => {
  it('says what was played, naming the tucked card for wonders', () => {
    expect(playedText('construct:walls')).toBe('You built Walls.');
    expect(playedText('discard:clay-pit')).toBe('You discarded Clay Pit.');
    expect(playedText('wonder:pyramids:walls')).toBe('You built the wonder Pyramids using Walls.');
    expect(playedText('wonder:pyramids')).toBe('You built the wonder Pyramids.');
    expect(playedText('token:law')).toBe('Your last move.');
    expect(moveText('construct:walls')).toBe('Building Walls');
    expect(moveText('discard:walls')).toBe('Discarding Walls');
    expect(moveText('wonder:pyramids:walls')).toBe('Building Pyramids with Walls');
    expect(moveText('wonder:pyramids')).toBe('Building Pyramids');
  });

  it('disambiguates same-wonder/different-card, same-card/different-wonder, and build/discard comparisons', () => {
    expect(
      verdictText(
        review({
          played: 'wonder:pyramids:press',
          better: 'wonder:pyramids:baths',
          reason: 'points',
          gap: 5
        }),
        names
      )
    ).toBe('Building Pyramids with Baths instead looked about 5 points better.');

    expect(
      verdictText(
        review({
          played: 'wonder:pyramids:baths',
          better: 'wonder:colossus:baths',
          reason: 'points',
          gap: 4
        }),
        names
      )
    ).toBe('Using Baths for Colossus instead looked about 4 points better.');

    expect(
      verdictText(
        review({
          played: 'wonder:pyramids:press',
          better: 'wonder:colossus:baths',
          reason: 'points',
          gap: 4
        }),
        names
      )
    ).toBe('Building Colossus with Baths looked about 4 points better.');

    expect(
      verdictText(
        review({
          played: 'discard:walls',
          better: 'construct:walls',
          reason: 'wins',
          gap: 30
        }),
        names
      )
    ).toBe('Building Walls would have won the game.');

    expect(
      verdictText(
        review({
          played: 'construct:baths',
          better: 'discard:baths',
          reason: 'points',
          gap: 3
        }),
        names
      )
    ).toBe('Discarding Baths looked about 3 points better.');
  });

  it('describes both moves of a play-again turn and identifies which move the advice is about', () => {
    expect(playedText('wonder:pyramids:baths construct:lumber-yard')).toBe(
      'Your two moves: you built the wonder Pyramids using Baths, then built Lumber Yard.'
    );
    expect(playedText('wonder:pyramids:baths wonder:colossus:press discard:lumber-yard')).toBe(
      'Your 3 moves: you built the wonder Pyramids using Baths, then built the wonder Colossus using Press, then discarded Lumber Yard.'
    );

    expect(
      verdictText(
        review({
          played: 'wonder:appian-way:walls construct:rostrum',
          better: 'construct:walls',
          reason: 'wins',
          gap: 30
        }),
        names
      )
    ).toBe('First move: Building Walls would have won the game.');

    expect(
      verdictText(
        review({
          played: 'wonder:appian-way:rostrum discard:walls',
          better: 'wonder:appian-way:rostrum construct:walls',
          reason: 'wins',
          gap: 30
        }),
        names
      )
    ).toBe('Extra turn: Building Walls would have won the game.');

    expect(
      verdictText(
        review({
          played: 'wonder:appian-way:rostrum wonder:pyramids:press',
          better: 'wonder:appian-way:rostrum wonder:pyramids:baths',
          reason: 'points',
          gap: 5
        }),
        names
      )
    ).toBe('Extra turn: Building Pyramids with Baths instead looked about 5 points better.');
  });

  it('omits the verdict line when no better move was found', () => {
    const lines = reviewLines(review({ played: 'construct:walls' }), names);
    expect(lines).toEqual({
      played: 'You built Walls.',
      verdict: '',
      better: false
    });
  });

  it('gives each reason in one sentence, naming the opponent', () => {
    const better = 'construct:walls';
    expect(verdictText(review({ better, reason: 'wins', gap: 30 }), names)).toBe(
      'Building Walls would have won the game.'
    );
    expect(
      verdictText(
        review({ better, reason: 'stopsWin', gap: 30, threat: { seat: 1, kind: 'military' } }),
        names
      )
    ).toBe('Building Walls would have kept Moon from taking your capital.');
    expect(
      verdictText(
        review({
          better,
          reason: 'stopsWin',
          gap: 30,
          threat: { seat: 1, kind: 'science', card: 'clay-pit' }
        }),
        names
      )
    ).toBe('Building Walls would have kept Moon from winning with Clay Pit.');
    expect(
      verdictText(
        review({
          better,
          reason: 'stopsWin',
          gap: 30,
          threat: { seat: 1, kind: 'science', wonder: 'pyramids' }
        }),
        names
      )
    ).toBe('Building Walls would have kept Moon from winning with Pyramids.');
    expect(verdictText(review({ better, reason: 'pair', gap: 4, symbol: 'wheel' }), names)).toBe(
      'Building Walls would have completed your wheel pair.'
    );
    expect(
      verdictText(review({ better, reason: 'deniesPair', gap: 4, symbol: 'wheel' }), names)
    ).toBe('Building Walls would have kept Moon from a wheel pair.');
    expect(verdictText(review({ better, reason: 'loot', gap: 4, coins: 2 }), names)).toBe(
      'Building Walls would have looted 2 coins from Moon.'
    );
    expect(verdictText(review({ better, reason: 'points', gap: 1 }), names)).toBe(
      'Building Walls looked about 1 point better.'
    );
    expect(verdictText(review({ better, reason: 'points', gap: 5 }), names)).toBe(
      'Building Walls looked about 5 points better.'
    );
  });

  it('words the points by the strength that measured them', async () => {
    const better = 'construct:walls';
    expect(
      verdictText(review({ better, reason: 'points', gap: 4, strength: 'twoStage' }), names)
    ).toBe('Building Walls looked about 4 points better, and a deeper search agreed.');
    // A deep review has no points: the search's verdict names no number.
    expect(verdictText(review({ better, reason: 'points', gap: 0, strength: 'deep' }), names)).toBe(
      'Building Walls looked clearly stronger to a deeper search.'
    );
    // A fact is a fact, whatever the strength.
    expect(verdictText(review({ better, reason: 'wins', gap: 0, strength: 'deep' }), names)).toBe(
      'Building Walls would have won the game.'
    );
    await i18n.set('fa');
    expect(
      stripBidiMarks(
        verdictText(review({ better, reason: 'points', gap: 0, strength: 'deep' }), names)
      )
    ).not.toMatch(/[0-9۰-۹]/);
    expect(
      stripBidiMarks(
        verdictText(review({ better, reason: 'points', gap: 4, strength: 'twoStage' }), names)
      )
    ).toContain('۴');
  });

  it('names the other seat for the second hot-seat player', () => {
    const r = review({ better: 'discard:walls', reason: 'loot', gap: 3, coins: 1 });
    expect(verdictText(r, { names: ['Sun', 'Moon'], seat: 1 })).toBe(
      'Discarding Walls would have looted 1 coin from Sun.'
    );
  });

  it('formats wonder, comparison, and play-again phrases in Persian', async () => {
    await i18n.set('fa');
    expect(stripBidiMarks(playedText('wonder:pyramids:baths'))).toBe(
      'شما شگفتی هرم‌های بزرگ را با گرمابه ساختید.'
    );
    expect(
      stripBidiMarks(
        verdictText(
          review({
            played: 'wonder:pyramids:press',
            better: 'wonder:pyramids:baths',
            reason: 'points',
            gap: 5
          }),
          names
        )
      )
    ).toBe('ساختن هرم‌های بزرگ با گرمابه به جای آن حدود ۵ امتیاز بهتر به نظر می‌رسید.');
    expect(
      stripBidiMarks(
        verdictText(
          review({
            played: 'wonder:pyramids:baths',
            better: 'wonder:colossus:baths',
            reason: 'points',
            gap: 4
          }),
          names
        )
      )
    ).toBe('استفاده از گرمابه برای ساختن غول‌پیکر رودس حدود ۴ امتیاز بهتر به نظر می‌رسید.');
    expect(stripBidiMarks(playedText('wonder:pyramids:baths construct:lumber-yard'))).toBe(
      'دو حرکت شما: نخست شگفتی هرم‌های بزرگ را با گرمابه ساختید، سپس انبار الوار را ساختید.'
    );
    expect(
      stripBidiMarks(
        verdictText(
          review({
            played: 'wonder:appian-way:rostrum discard:walls',
            better: 'wonder:appian-way:rostrum construct:walls',
            reason: 'wins',
            gap: 30
          }),
          names
        )
      )
    ).toBe('در نوبت دوباره: با ساختن بارو بازی را می‌بردید.');
  });
});

describe('reviewForSummary', () => {
  const r = review({ ply: 10 });
  const history = [entry(9, 1), entry(10, 0), entry(11, 1), entry(12, 1)];

  it('belongs to the summary of the moves right after the reviewed one', () => {
    expect(reviewForSummary(r, summary(1, [11, 12]), history)).toBe(r);
  });

  it('is absent without a review or a summary', () => {
    expect(reviewForSummary(undefined, summary(1, [11]), history)).toBeNull();
    expect(reviewForSummary(r, null, history)).toBeNull();
    expect(reviewForSummary(r, summary(1, []), history)).toBeNull();
  });

  it('is absent from an older summary, before the reviewed move', () => {
    expect(reviewForSummary(r, summary(1, [9]), history)).toBeNull();
  });

  it('is absent when the other side moved between the review and the summary', () => {
    // The summary of 11 carried the review of 10; by 13 it is an old lesson.
    const h = [entry(10, 0), entry(11, 1), entry(12, 0), entry(13, 1)];
    expect(reviewForSummary(r, summary(1, [13]), h)).toBeNull();
  });

  it("follows the viewer's own choice that came after the reviewed move", () => {
    // A wonder at 10, then its progress token pick at 11: the summary of 12.
    const h = [entry(10, 0), entry(11, 0), entry(12, 1)];
    expect(reviewForSummary(r, summary(1, [12]), h)).toBe(r);
  });
});
