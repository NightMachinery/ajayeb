// The game's own Animation setting applied to a timeline (D21: the setting
// alone decides; the OS's reduced-motion preference is never read), and the
// sound cues timed by the result.
//
// scaleTimeline multiplies every start and length by the speed's factor
// (scaleMs: normal 1, fast 0.6, relaxed 1.6), and every other time of a
// step (plan.ts TIME_FIELDS: a tween, a stagger); Off gives the empty timeline,
// so the frame shows at once and nothing is read from the page.
//
// retimeCues: cues.ts spaces a frame's cues CUE_SPACING_MS apart; when the
// frame animates, each cue kind with a step of its own sounds when that
// step happens instead (plan.ts cueOffsets): a take's slide when the card
// lands, a flip with the first flip, a coin with its float, the drum with a
// looting flip or the pawn's first step, a chime with a pair's glow, the
// fanfare with the victory banner, a tick when a draft tile lands. A cue
// with no step of its own keeps its place in the order, one spacing after
// the cue before it. A frame that does not
// animate keeps cues.ts's spacing.

import { scaleMs, type AnimationSpeed } from '$lib/ui/motion.svelte';
import { CUE_SPACING_MS, type PlannedCue } from '../cues';
import {
  cueOffsets,
  EMPTY_TIMELINE,
  planTimeline,
  TIME_FIELDS,
  type PlanFrame,
  type Step,
  type Timeline
} from './plan';

/** A normal-speed timeline at `speed`. */
export function scaleTimeline(timeline: Timeline, speed: AnimationSpeed): Timeline {
  if (speed === 'off' || timeline.steps.length === 0) return EMPTY_TIMELINE;
  const ms = (base: number) => scaleMs(base, speed);
  return {
    steps: timeline.steps.map((s) => {
      const out: Record<string, unknown> = { ...s };
      for (const field of TIME_FIELDS) {
        const value = out[field];
        if (typeof value === 'number') out[field] = ms(value);
      }
      return out as unknown as Step;
    }),
    blockingMs: ms(timeline.blockingMs),
    totalMs: ms(timeline.totalMs)
  };
}

/** The timeline of the frame `next` after `prev` (cuePrev's), at `speed`. */
export function timelineFor(
  next: PlanFrame,
  prev: PlanFrame | null,
  speed: AnimationSpeed
): Timeline {
  return speed === 'off' ? EMPTY_TIMELINE : scaleTimeline(planTimeline(next, prev), speed);
}

/** A frame's cues, timed by the (scaled) timeline it plays. */
export function retimeCues(cues: readonly PlannedCue[], timeline: Timeline): PlannedCue[] {
  if (timeline.steps.length === 0) return cues.map((c) => ({ ...c }));
  const at = cueOffsets(timeline);
  let last = -CUE_SPACING_MS;
  return cues.map(({ cue, delayMs }) => {
    const anchor = at[cue];
    const next = anchor ?? Math.max(delayMs, last + CUE_SPACING_MS);
    last = next;
    return { cue, delayMs: next };
  });
}
