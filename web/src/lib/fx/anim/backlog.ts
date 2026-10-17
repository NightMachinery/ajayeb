// Carry only an age deal through a burst of live frames. Ordinary queued
// moves still catch up instantly. Deal the surviving cards of the latest
// snapshot, never resurrect a card already taken by a zero-delay bot.
import type { DealStep, PlanFrame, Timeline } from './plan';

export class DealCarry {
  #pending: { code: string; step: DealStep } | null = null;

  clear(): void {
    this.#pending = null;
  }

  plan(next: PlanFrame, prev: PlanFrame | null, timeline: Timeline, backlog: number): Timeline {
    if (
      !prev ||
      prev.code !== next.code ||
      next.version < prev.version ||
      this.#pending?.code !== next.code ||
      this.#pending?.step.age !== next.view?.age
    ) {
      this.clear();
    }
    const deal = timeline.steps.find((s): s is DealStep => s.kind === 'deal');
    if (deal) this.#pending = { code: next.code, step: deal };
    if (backlog > 0) return { steps: [], blockingMs: 0, totalMs: 0 };
    const pending = this.#pending;
    this.clear();
    if (!pending || deal || next.view?.phase !== 'play') return timeline;
    return {
      ...timeline,
      steps: [pending.step, ...timeline.steps],
      totalMs: Math.max(timeline.totalMs, pending.step.at + pending.step.dur)
    };
  }
}
