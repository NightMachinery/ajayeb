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
// Each cue kind plays once per frame. Busy moves keep at most three cues,
// favouring distinct milestones over routine card, coin and reveal sounds.
// The retained cues keep event order and CUE_SPACING_MS between starts;
// the animation planner later aligns them with their visible steps.

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
  /** History snapshots are never live, even if their version increased. */
  history?: readonly unknown[];
  room?: { mode: string };
  you?: { seats: readonly number[] };
  view?: {
    viewer?: number;
    active?: number;
    toMove?: number;
    phase?: string;
    age?: number;
    playingAgain?: boolean;
  };
}

function isScienceCard(slug: string | undefined): boolean {
  return slug !== undefined && cardBySlug(slug)?.color === 'green';
}

/** The cues one event makes, in the order they should sound. */
export function eventCues(e: GameEvent, frame?: CueFrame): Cue[] {
  switch (e.kind) {
    case 'build':
      return isScienceCard(e.card) ? ['slide', 'chime'] : ['slide'];
    case 'discard':
      return (e.amount ?? 0) > 0 ? ['slide', 'coin'] : ['slide'];
    case 'wonder':
      return ['slide', 'wonder'];
    case 'rebuild':
      return ['rebuild'];
    case 'destroy':
      return ['destroy'];
    case 'pay':
    case 'coins':
      return (e.amount ?? 0) !== 0 ? ['coin'] : [];
    case 'reveal':
    case 'draftRound':
      return ['flip'];
    case 'pawn':
      return ['drum'];
    case 'loot':
      return ['loot'];
    case 'pair':
      return ['chime'];
    case 'token':
      return ['token'];
    case 'libraryDraw':
      return ['libraryDraw'];
    case 'extraTurn':
      return ['extraTurn'];
    case 'ageStart':
      return ['ageStart'];
    case 'gameOver': {
      if (e.winner === undefined) return ['chime'];
      if (frame?.room?.mode === 'hotseat') return ['fanfare'];
      const viewer = viewerSeat(frame);
      return [viewer === undefined ? 'chime' : viewer === e.winner ? 'fanfare' : 'defeat'];
    }
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
    next.history === undefined &&
    next.events.length > 0
  );
}

const NONE: CuePlan = { cues: [], haptic: null };

/** A personal cue requires both a viewer seat and permission for that seat. */
function viewerSeat(frame: CueFrame | undefined): number | undefined {
  const viewer = frame?.view?.viewer;
  return (viewer === 0 || viewer === 1) && frame?.you?.seats.includes(viewer) ? viewer : undefined;
}

/** A rival's completed turn hands an online player a normal board turn. */
function isYourTurn(next: CueFrame, prev: CueFrame): boolean {
  const viewer = viewerSeat(next);
  return (
    next.room?.mode === 'online' &&
    prev.room?.mode === 'online' &&
    viewer !== undefined &&
    viewerSeat(prev) === viewer &&
    next.view?.phase === 'play' &&
    next.view.active === viewer &&
    next.view.toMove === viewer &&
    next.view.playingAgain === false &&
    prev.view?.active === 1 - viewer &&
    // A starter choice can temporarily give the other seat toMove without
    // handing over a turn. Its answer is the age opening, not a rival move.
    prev.view.phase !== 'chooseStarter' &&
    prev.view.phase !== 'wonderDraft' &&
    !next.events.some((e) => e.kind === 'starterChosen' || e.kind === 'ageStart')
  );
}

// Higher-ranked moments survive a busy move. Ties keep first-event order.
// Retained cues are played in that order, not in priority order.
const PRIORITY: readonly Cue[] = [
  'fanfare',
  'defeat',
  'yourTurn',
  'extraTurn',
  'ageStart',
  'token',
  'libraryDraw',
  'loot',
  'wonder',
  'destroy',
  'rebuild',
  'drum',
  'chime',
  'slide',
  'flip',
  'coin',
  'tick'
];

/** Cues for a shown frame. playedDeal, when supplied, is the runner's actual deal decision. */
export function planCues(next: CueFrame, prev: CueFrame | null, playedDeal?: boolean): CuePlan {
  const live = isLive(next, prev);
  // A real carried deal may flush on an eventless, same-version thinking
  // frame. The runner already proved it came from a live age opening.
  const carried =
    playedDeal === true &&
    prev !== null &&
    prev.code === next.code &&
    next.version >= prev.version &&
    next.history === undefined;
  if ((!live && !carried) || !prev) return NONE;
  const seen = new Set<Cue>();
  // With animation, use the runner's actual recovered/carried deal. Off
  // has no visual timeline: server events and phase/age transitions retain
  // the age feedback, including a coalesced starter/draft answer.
  const deal =
    playedDeal ??
    (live &&
      ((next.events.some((e) => e.kind === 'ageStart') && next.view?.phase !== 'chooseStarter') ||
        next.events.some((e) => e.kind === 'starterChosen') ||
        (next.view?.phase === 'play' &&
          (next.view.age ?? 0) > 0 &&
          (prev.view?.phase === 'chooseStarter' ||
            prev.view?.phase === 'wonderDraft' ||
            (prev.view?.age !== undefined && next.view.age !== prev.view.age)))));
  if (live)
    for (const e of next.events) {
      if (e.kind === 'ageStart') {
        if (deal) seen.add('ageStart');
        continue;
      }
      for (const cue of eventCues(e, next)) seen.add(cue);
    }
  if (deal) seen.add('ageStart');
  if (live && isYourTurn(next, prev)) seen.add('yourTurn');
  // These routine sounds repeat information already carried by a richer cue.
  if (seen.has('loot')) seen.delete('coin');
  if (seen.has('ageStart')) {
    seen.delete('flip');
    seen.delete('tick');
  }
  const result = next.events.find((e) => e.kind === 'gameOver');
  const priority: readonly Cue[] =
    result && eventCues(result, next)[0] === 'chime' ? ['chime', ...PRIORITY] : PRIORITY;
  const selected = new Set(
    priority.filter((cue, i) => seen.has(cue) && priority.indexOf(cue) === i).slice(0, 3)
  );
  const cues = [...seen]
    .filter((cue) => selected.has(cue))
    .map((cue, i) => ({
      cue,
      delayMs: i * CUE_SPACING_MS
    }));
  return { cues, haptic: live ? eventsHaptic(next.events) : null };
}
