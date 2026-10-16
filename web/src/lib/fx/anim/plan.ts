// The animation planner: a frame's live events, and the two views around
// them, turned into a timeline of steps (UX design section 5, "Animations").
// Pure: it reads only what the server sent and decides nothing about the
// rules. The runner (runner.ts) finds the elements each step moves and
// plays it; this file only says what moves, when, and for how long, at
// normal speed. The runner scales every offset and duration with the
// game's own Animation setting (scaleMs, D21).
//
// A frame's steps, in the order the events report them:
//
//   take   a card leaves its structure slot: built into the city, discarded
//          for coins, or tucked under a wonder (build, discard, wonder);
//   flip   a face-down card turns face up (reveal), after the take lands,
//          one after another;
//   coins  a seat's coins change: a float and a counter tween, after the
//          take lands (pay, coins, the coins of a discard, a trade's coins
//          to the opponent; the amount is the view's own before and after);
//   pawn   the military pawn walks cell by cell from `from` to `to`, after
//          the take lands.
//
// Live events only, the rule fx/cues.ts follows: nothing on the first frame
// of a room, a resync, an undo or a frame that carries a history (the
// caller passes cuePrev(prev, next) as `prev`). The planner never plans
// anything the other animation slice owns (looting, tokens, the age deal,
// the draft, destroy and rebuild, the end).
//
// `blockingMs` is the part input waits for: the take's landing, at most
// BLOCK_CAP_MS. Everything after it (flips, floats, the pawn) plays on
// while the player may already move; a new frame finishes it at once.

import type { GameEvent } from '$lib/wire';
import { isLive, type Cue } from '../cues';

/** A card's flight from its slot to where it goes. */
export const TAKE_MS = 320;
/** One revealed card's flip, and the gap between two flips. */
export const FLIP_MS = 350;
export const FLIP_STAGGER_MS = 60;
/** The pause between a take landing and the first flip. */
export const FLIP_DELAY_MS = 40;
/** A coin float rising and fading, and the counter's tween. */
export const FLOAT_MS = 800;
export const TWEEN_MS = 450;
/** The pawn's time per cell. */
export const PAWN_STEP_MS = 150;
/** The most a frame may hold input, at normal speed (450 ms is the budget). */
export const BLOCK_CAP_MS = 350;

export type Seat = 0 | 1;

/** Where a taken card goes. */
export type TakeDest = 'city' | 'discard' | 'wonder';

interface StepTime {
  /** The start, in ms after the swap, at normal speed. */
  at: number;
  /** The length, in ms at normal speed. */
  dur: number;
}

export interface TakeStep extends StepTime {
  kind: 'take';
  seat: Seat;
  /** The card's slug: its slot is found by data-card in the old DOM. */
  card: string;
  dest: TakeDest;
  /** The wonder a card was tucked under. */
  wonder?: string;
}

export interface FlipStep extends StepTime {
  kind: 'flip';
  /** The structure slot that turned face up, and its new card. */
  slot: number;
  card: string;
}

export interface CoinStep extends StepTime {
  kind: 'coins';
  seat: Seat;
  /** The counter's value before and after (the views' own numbers). */
  from: number;
  to: number;
  /** How long the counter tweens; `dur` is the float's. */
  tween: number;
}

export interface PawnStep extends StepTime {
  kind: 'pawn';
  /** The pushing seat, and the pawn's space before and after. */
  seat: Seat;
  from: number;
  to: number;
}

export type Step = TakeStep | FlipStep | CoinStep | PawnStep;
export type StepKind = Step['kind'];

export interface Timeline {
  steps: Step[];
  /** How long input waits, at normal speed. */
  blockingMs: number;
  /** When the last step ends, at normal speed. */
  totalMs: number;
}

export const EMPTY_TIMELINE: Timeline = Object.freeze({
  steps: [],
  blockingMs: 0,
  totalMs: 0
}) as Timeline;

/** The part of a state frame the planner reads. */
export interface PlanFrame {
  code: string;
  version: number;
  events: readonly GameEvent[];
  /** The view: each seat's coins, for the counter's before and after. */
  view?: { players: readonly { coins: number }[] };
}

function seatOf(n: number | undefined): Seat | null {
  return n === 0 || n === 1 ? n : null;
}

/** The take of one event, if it takes a card from the structure. */
function takeOf(e: GameEvent): Omit<TakeStep, 'at' | 'dur'> | null {
  const seat = seatOf(e.seat);
  if (seat === null || !e.card) return null;
  switch (e.kind) {
    case 'build':
      return { kind: 'take', seat, card: e.card, dest: 'city' };
    case 'discard':
      return { kind: 'take', seat, card: e.card, dest: 'discard' };
    case 'wonder':
      return e.wonder
        ? { kind: 'take', seat, card: e.card, dest: 'wonder', wonder: e.wonder }
        : null;
    default:
      return null;
  }
}

/** Whether an event moves coins (the float's timing; the amount is the views'). */
function movesCoins(e: GameEvent): boolean {
  switch (e.kind) {
    case 'pay':
    case 'coins':
    case 'discard':
      return (e.amount ?? 0) !== 0;
    default:
      return false;
  }
}

/** The seats whose coins an event moves. */
function coinSeats(e: GameEvent): Seat[] {
  const seat = seatOf(e.seat);
  if (seat === null || !movesCoins(e)) return [];
  if (e.kind === 'pay' && (e.toOpponent ?? 0) > 0) return [seat, seat === 0 ? 1 : 0];
  return [seat];
}

/**
 * The timeline of the frame `next`, shown after `prev` (cuePrev's: null for
 * a frame that replays history). Empty when nothing live happened.
 */
export function planTimeline(next: PlanFrame, prev: PlanFrame | null): Timeline {
  if (!prev || !isLive(next, prev)) return EMPTY_TIMELINE;
  const events = next.events;
  // A reveal before a new age's deal belongs to the structure that left.
  let firstLive = 0;
  events.forEach((e, i) => {
    if (e.kind === 'ageStart') firstLive = i + 1;
  });

  const steps: Step[] = [];
  const take = events.map(takeOf).find((t) => t !== null) ?? null;
  if (take) steps.push({ ...take, at: 0, dur: TAKE_MS });
  const after = take ? TAKE_MS : 0;

  let flips = 0;
  events.forEach((e, i) => {
    if (e.kind !== 'reveal' || i < firstLive || e.slot === undefined || !e.card) return;
    const at = after + FLIP_DELAY_MS + flips * FLIP_STAGGER_MS;
    steps.push({ kind: 'flip', slot: e.slot, card: e.card, at, dur: FLIP_MS });
    flips += 1;
  });

  const seen = new Set<Seat>();
  for (const e of events) {
    for (const seat of coinSeats(e)) {
      if (seen.has(seat)) continue;
      seen.add(seat);
      const from = prev.view?.players[seat]?.coins;
      const to = next.view?.players[seat]?.coins;
      // Without both views (a cue-only plan) the step keeps its timing.
      if (from !== undefined && to !== undefined && from === to) continue;
      steps.push({
        kind: 'coins',
        seat,
        from: from ?? 0,
        to: to ?? 0,
        at: after,
        dur: FLOAT_MS,
        tween: TWEEN_MS
      });
    }
  }

  const pawns = events.filter(
    (e) => e.kind === 'pawn' && e.from !== undefined && e.to !== undefined
  );
  const first = pawns[0];
  const last = pawns[pawns.length - 1];
  const pusher = seatOf(first?.seat);
  if (first && last && pusher !== null && first.from !== last.to) {
    const from = first.from ?? 0;
    const to = last.to ?? 0;
    steps.push({
      kind: 'pawn',
      seat: pusher,
      from,
      to,
      at: after,
      dur: Math.abs(to - from) * PAWN_STEP_MS
    });
  }

  if (steps.length === 0) return EMPTY_TIMELINE;
  const totalMs = Math.max(...steps.map((s) => s.at + s.dur));
  return { steps, blockingMs: Math.min(after, BLOCK_CAP_MS), totalMs };
}

/**
 * The start of each cue kind's step, at normal speed, for fx/index.ts to
 * time its sound cues by: a take's slide when it lands, a coin with its
 * float, a flip with the first flip, the drum with the pawn's first step.
 * A cue kind with no step here keeps its own spacing.
 */
export function cueOffsets(timeline: Timeline): Partial<Record<Cue, number>> {
  const out: Partial<Record<Cue, number>> = {};
  const set = (cue: Cue, at: number) => {
    if (out[cue] === undefined) out[cue] = at;
  };
  for (const s of timeline.steps) {
    if (s.kind === 'take') set('slide', s.at + s.dur);
    else if (s.kind === 'flip') set('flip', s.at);
    else if (s.kind === 'coins') set('coin', s.at);
    else set('drum', s.at);
  }
  return out;
}
