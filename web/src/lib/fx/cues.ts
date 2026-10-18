// Which sound cues and which vibration a frame's live events make (UX
// design section 5, "Sound and haptics"). Pure: it reads the events the
// server sent and the presentation catalog (a card's colour), and decides
// nothing about the rules. fx/index.ts plays the plan.
//
// Live events only. A frame plays nothing when there is no frame before it
// on screen (the first snapshot after opening or rejoining a room), when it
// belongs to another room, or when its version does not move forward (an
// undo, a resync): those events, if any, are history, not something that
// just happened in front of the player.
//
// One frame may carry many events (a build that pays, moves the pawn and
// loots twice). Each cue kind plays at most once per frame, in the order
// its first event came, spaced CUE_SPACING_MS apart so they do not blur
// into one sound. Every cue keeps a delay, so the animation slice can later
// retime it to the animation it belongs to.

import { cardBySlug } from '$lib/catalog';
import type { GameEvent } from '$lib/wire';

// The player/gallery inventory. Event wiring stays in eventCues below.
export const CUES = [
  'slide',
  'coin',
  'flip',
  'drum',
  'chime',
  'fanfare',
  'tick',
  'wonder',
  'token',
  'loot',
  'ageStart',
  'defeat',
  'yourTurn',
  'destroy',
  'rebuild',
  'libraryDraw',
  'extraTurn'
] as const;
/**
 * slide: a card taken or built; coin: coins gained or paid; flip: a
 * face-down card revealed; drum: shields, the pawn, looting; chime: a
 * science symbol, a pair, a progress token; fanfare: the game's end; tick:
 * a draft pick.
 */
export type Cue = (typeof CUES)[number];

/** The gap between two cues of one frame. */
export const CUE_SPACING_MS = 90;

/** navigator.vibrate patterns, in ms (on, off, on ...). */
export const HAPTIC_TAKE: readonly number[] = [8];
export const HAPTIC_MILITARY: readonly number[] = [18, 40, 18];
export const HAPTIC_LOOT: readonly number[] = [24, 40, 24, 40, 36];

export interface PlannedCue {
  cue: Cue;
  delayMs: number;
}

export interface CuePlan {
  cues: PlannedCue[];
  /** The vibration pattern of this frame, or null for none. */
  haptic: readonly number[] | null;
}

/** The part of a state frame the planner reads. */
export interface CueFrame {
  code: string;
  version: number;
  events: readonly GameEvent[];
}

function isScienceCard(slug: string | undefined): boolean {
  return slug !== undefined && cardBySlug(slug)?.color === 'green';
}

/** The cues one event makes, in the order they should sound. */
export function eventCues(e: GameEvent): Cue[] {
  switch (e.kind) {
    case 'build':
    case 'rebuild':
      return isScienceCard(e.card) ? ['slide', 'chime'] : ['slide'];
    case 'discard':
      return (e.amount ?? 0) > 0 ? ['slide', 'coin'] : ['slide'];
    case 'wonder':
    case 'destroy':
      return ['slide'];
    case 'pay':
    case 'coins':
      return (e.amount ?? 0) !== 0 ? ['coin'] : [];
    case 'reveal':
      return ['flip'];
    case 'pawn':
    case 'loot':
      return ['drum'];
    case 'pair':
    case 'token':
      return ['chime'];
    case 'gameOver':
      return ['fanfare'];
    case 'draft':
    case 'starterChosen':
      return ['tick'];
    default:
      return [];
  }
}

const TAKE_KINDS: ReadonlySet<string> = new Set(['build', 'rebuild', 'discard', 'wonder', 'draft']);

/** The vibration of a list of events: looting beats the pawn beats a take. */
export function eventsHaptic(events: readonly GameEvent[]): readonly number[] | null {
  if (events.some((e) => e.kind === 'loot')) return HAPTIC_LOOT;
  if (events.some((e) => e.kind === 'pawn')) return HAPTIC_MILITARY;
  if (events.some((e) => TAKE_KINDS.has(e.kind))) return HAPTIC_TAKE;
  return null;
}

/** Whether `next`'s events are live: something that just happened after `prev`. */
export function isLive(next: CueFrame, prev: CueFrame | null): boolean {
  return (
    prev !== null &&
    prev.code === next.code &&
    next.version > prev.version &&
    next.events.length > 0
  );
}

const NONE: CuePlan = { cues: [], haptic: null };

/** The cues and vibration of the frame `next`, shown after `prev`. */
export function planCues(next: CueFrame, prev: CueFrame | null): CuePlan {
  if (!isLive(next, prev)) return NONE;
  const seen = new Set<Cue>();
  const cues: PlannedCue[] = [];
  for (const e of next.events) {
    for (const cue of eventCues(e)) {
      if (seen.has(cue)) continue;
      seen.add(cue);
      cues.push({ cue, delayMs: cues.length * CUE_SPACING_MS });
    }
  }
  return { cues, haptic: eventsHaptic(next.events) };
}
