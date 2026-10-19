import { describe, expect, it } from 'vitest';
import type { HistoryEntry, ReviewView } from '$lib/wire';
import { endReview } from './end-review';

const REVIEW: ReviewView = {
  entries: [],
  ply: 40,
  played: 'construct:arsenal',
  reason: 'none',
  gap: 0,
  strength: 'twoStage'
};

function entry(ply: number, seat: 0 | 1): HistoryEntry {
  return { ply, seat, events: [] };
}

describe('endReview', () => {
  it("shows the review when the viewer's own move ended the game", () => {
    const history = [entry(38, 1), entry(39, 0), entry(40, 1)];
    expect(endReview(REVIEW, 1, history, 'military')).toBe(REVIEW);
  });

  it('counts a token pick after the reviewed move as the same move', () => {
    const history = [entry(39, 0), entry(40, 1), entry(41, 1)];
    expect(endReview(REVIEW, 1, history, 'science')).toBe(REVIEW);
  });

  it("leaves it to the summary when the opponent's move ended the game", () => {
    const history = [entry(40, 1), entry(41, 0)];
    expect(endReview(REVIEW, 1, history, 'civilian')).toBeNull();
  });

  it('shows nothing for a forfeit, a spectator or a game without a review', () => {
    const history = [entry(39, 0), entry(40, 1)];
    expect(endReview(REVIEW, 1, history, 'forfeit')).toBeNull();
    expect(endReview(REVIEW, undefined, history, 'civilian')).toBeNull();
    expect(endReview(undefined, 1, history, 'civilian')).toBeNull();
    expect(endReview(REVIEW, 1, [], 'civilian')).toBeNull();
  });
});
