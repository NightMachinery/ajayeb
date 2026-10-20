import { describe, expect, it } from 'vitest';
import { DealCarry } from './backlog';
import { EMPTY_TIMELINE, planTimeline, type PlanFrame } from './plan';
import type { GameEvent } from '$lib/wire';

function frame(version: number, kinds: GameEvent['kind'][] = [], age = 1): PlanFrame {
  return {
    code: 'ROOM',
    version,
    events: kinds.map((kind) => ({ kind, private: false, age })),
    view: { phase: 'play', age, players: [] }
  };
}

describe('live age deals during catch-up', () => {
  it.each(['starterChosen', 'ageStart'] as const)(
    'carries %s through a zero-delay bot burst exactly once',
    (kind) => {
      const carry = new DealCarry();
      const before = frame(1);
      const start = frame(2, [kind], 2);
      expect(carry.plan(start, before, planTimeline(start, before), 2)).toEqual(EMPTY_TIMELINE);
      const thinking = frame(2, [], 2);
      expect(carry.plan(thinking, start, EMPTY_TIMELINE, 1)).toEqual(EMPTY_TIMELINE);
      const answer = frame(3, ['build'], 2);
      const caught = carry.plan(answer, thinking, planTimeline(answer, thinking), 0);
      expect(caught.steps.filter((s) => s.kind === 'deal')).toHaveLength(1);
      expect(carry.plan(frame(4, [], 2), answer, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    }
  );

  it('deals when the Presenter coalesced the last wonderDraft frame before capture', () => {
    const before = frame(1, [], 0);
    before.view!.phase = 'wonderDraft';
    const answer = frame(4, ['build'], 1);
    const timeline = planTimeline(answer, before);
    expect(timeline.steps.filter((s) => s.kind === 'deal')).toHaveLength(1);
    expect(planTimeline(answer, null)).toEqual(EMPTY_TIMELINE);
  });

  it('never deals again after a starter choice on screen: its age was dealt when it opened', () => {
    const before = frame(1, [], 2);
    before.view!.phase = 'chooseStarter';
    const answer = frame(4, ['build'], 2);
    expect(planTimeline(answer, before).steps.filter((s) => s.kind === 'deal')).toHaveLength(0);
  });

  it('carries an opening deal into the starter choice frame, holding input to its end', () => {
    const carry = new DealCarry();
    const before = frame(1);
    const start = frame(2, ['ageStart'], 2);
    start.view!.phase = 'chooseStarter';
    expect(carry.plan(start, before, planTimeline(start, before), 1)).toEqual(EMPTY_TIMELINE);
    const shown = frame(3, [], 2);
    shown.view!.phase = 'chooseStarter';
    const caught = carry.plan(shown, start, EMPTY_TIMELINE, 0);
    const deal = caught.steps.find((s) => s.kind === 'deal')!;
    expect(caught.blockingMs).toBe(deal.at + deal.dur);
  });

  it('still skips ordinary catch-up moves and drops the deal on resync/undo or a different age', () => {
    const carry = new DealCarry();
    const before = frame(1);
    const start = frame(2, ['starterChosen'], 2);
    carry.plan(start, before, planTimeline(start, before), 1);
    expect(carry.plan(frame(3), start, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    carry.plan(start, before, planTimeline(start, before), 1);
    expect(carry.plan(frame(4, [], 2), null, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    carry.plan(start, before, planTimeline(start, before), 1);
    const undo = frame(1, [], 2);
    expect(carry.plan(undo, start, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    carry.plan(start, before, planTimeline(start, before), 1);
    const otherRoom = frame(4, [], 2);
    otherRoom.code = 'OTHER';
    expect(carry.plan(otherRoom, start, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    carry.plan(start, before, planTimeline(start, before), 1);
    carry.clear(); // runner.finish on room clear/hurry/destroy
    expect(carry.plan(frame(4, [], 2), start, EMPTY_TIMELINE, 0)).toEqual(EMPTY_TIMELINE);
    const move = frame(5);
    move.events = [{ kind: 'build', private: false, seat: 0, card: 'baths' }];
    const ordinary = planTimeline(move, before);
    expect(ordinary.steps.length).toBeGreaterThan(0);
    expect(ordinary.blockingMs).toBeGreaterThan(0);
    expect(carry.plan(move, before, ordinary, 1)).toEqual(EMPTY_TIMELINE);
  });
});
