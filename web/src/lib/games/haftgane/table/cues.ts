// Which sound cues and which vibration a Haftgane frame makes, as Duel's
// $lib/fx/cues.ts does for Duel. Pure: it reads the events the server sent
// this viewer and the frame's decision point, and decides nothing about
// the rules. sound.ts plays the plan through the app's player ($lib/fx).
//
// Live frames only. A frame plays nothing when it carries a history (a
// join, a reload, a resync, an undo), when it belongs to another room or
// another seat, when there is no frame before it, or when its revision does
// not move forward: those events restate the game, they did not just
// happen in front of the viewer.
//
// Up to seven seats resolve at once, so nothing here is per seat. A batch
// makes one grouped cue for its reveal (a flip for the plays or the sides,
// a rebuild for a build from the discard pile, a tick for a pile with
// nothing to build), one for its conflicts (a drum), one for a new Age's
// deal and one for the result; a seated viewer adds only their own salient
// moments: what they did (a slide for a building, the wonder for a stage, a
// coin for a sale), coins they earned or received, the new hand arriving
// (a slide), their own confirmation reaching the table (a tick), a
// decision that opens for them alone (their turn), the moment every other
// person at the table has confirmed and the table waits on them (their
// turn again; never for a table of bots, which always wait) and a last
// card kept for an extra turn. At most three cues sound per frame, chosen
// by PRIORITY (the result, the Age, the conflicts, the reveal, then the
// viewer's own) and played in ORDER, CUE_SPACING_MS apart.

import type { Cue } from '$lib/fx/cues';
import type { TableEvent } from '$lib/wire/platform.gen';

/** The gap between two cues of one frame. */
export const CUE_SPACING_MS = 90;
/** The most cues one frame plays. */
export const CUE_BUDGET = 3;

/** navigator.vibrate patterns, in ms (on, off, on ...). */
export const HAPTIC_REVEAL: readonly number[] = [8];
export const HAPTIC_CONFLICT: readonly number[] = [18, 40, 18];

export interface PlannedCue {
  cue: Cue;
  delayMs: number;
}

export interface TableCuePlan {
  cues: PlannedCue[];
  /** The vibration of this frame, or null for none. */
  haptic: readonly number[] | null;
}

/** The part of a table frame the planner reads. */
export interface CueTable {
  code: string;
  revision: number;
  events: readonly TableEvent[];
  history?: readonly unknown[];
  you: { seat?: number };
  seats: readonly { seat: number; kind: string }[];
  point?: {
    ref: { id: string };
    kind: string;
    required: readonly number[];
    confirmed: readonly number[];
  };
  view: { seats: readonly unknown[]; result?: { winners: readonly number[] } };
}

/** The decisions a seat takes alone, which open with their own notice. */
const OWN_DECISIONS: ReadonlySet<string> = new Set(['discard-build', 'last-card']);

/** Which cue survives a busy frame: the shared moments, then the viewer's own. */
const PRIORITY: readonly Cue[] = [
  'fanfare',
  'defeat',
  'chime',
  'ageStart',
  'drum',
  'flip',
  'rebuild',
  'extraTurn',
  'yourTurn',
  'wonder',
  'coin',
  'slide',
  'tick'
];

/** When the retained cues sound: the reveal, the viewer's part, the Age's end, the next decision. */
const ORDER: readonly Cue[] = [
  'flip',
  'rebuild',
  'tick',
  'wonder',
  'coin',
  'slide',
  'drum',
  'ageStart',
  'fanfare',
  'defeat',
  'chime',
  'extraTurn',
  'yourTurn'
];

const NONE: TableCuePlan = { cues: [], haptic: null };

/** Whether `next`'s events just happened after `prev`, rather than restating the game. */
export function isLive(next: CueTable, prev: CueTable | null): boolean {
  return (
    prev !== null &&
    next.history === undefined &&
    prev.code === next.code &&
    (prev.you.seat ?? null) === (next.you.seat ?? null) &&
    next.revision > prev.revision
  );
}

/** What the viewer's own play sounds like. */
function playCue(e: TableEvent): Cue | null {
  switch (e.play) {
    case 'build':
      return 'slide';
    case 'stage':
      return 'wonder';
    case 'sell':
      return 'coin';
  }
  return null;
}

/** Whether coins came to `seat`: its own income, or a neighbour paying it for a trade. */
function gainsCoins(events: readonly TableEvent[], seat: number, count: number): boolean {
  return events.some(
    (e) =>
      (e.kind === 'coins' && e.seat === seat && e.delta > 0) ||
      (e.kind === 'play' &&
        e.seat !== seat &&
        ((e.left > 0 && (e.seat + 1) % count === seat) ||
          (e.right > 0 && (e.seat + count - 1) % count === seat)))
  );
}

/** Whether `next` opens a decision for `seat` alone that `prev` did not hold. */
export function opensOwnDecision(next: CueTable, prev: CueTable, seat: number): boolean {
  const p = next.point;
  return (
    !!p &&
    OWN_DECISIONS.has(p.kind) &&
    p.required.includes(seat) &&
    !p.confirmed.includes(seat) &&
    prev.point?.ref.id !== p.ref.id
  );
}

/**
 * Whether `next` leaves `seat` the last to confirm the same point, where
 * `prev` still waited on someone else, and someone else is a person.
 */
export function waitsOnlyFor(next: CueTable, prev: CueTable, seat: number): boolean {
  const p = next.point;
  const q = prev.point;
  if (!p || !q || p.ref.id !== q.ref.id || !p.required.includes(seat)) return false;
  if (p.confirmed.includes(seat)) return false;
  const others = p.required.filter((s) => s !== seat);
  const person = next.seats.some((s) => s.kind === 'human' && others.includes(s.seat));
  return (
    person &&
    others.every((s) => p.confirmed.includes(s)) &&
    others.some((s) => !q.confirmed.includes(s))
  );
}

/** Whether `next` shows `seat`'s own confirmation of the point `prev` showed unconfirmed. */
export function confirmsOwn(next: CueTable, prev: CueTable, seat: number): boolean {
  const p = next.point;
  const q = prev.point;
  return (
    !!p && !!q && p.ref.id === q.ref.id && p.confirmed.includes(seat) && !q.confirmed.includes(seat)
  );
}

/** The cues of a live frame before the budget, as a set. */
function frameCues(next: CueTable, prev: CueTable): Set<Cue> {
  const events = next.events;
  const has = (kind: string) => events.some((e) => e.kind === kind);
  const seen = new Set<Cue>();
  if (has('play') || has('side-chosen')) seen.add('flip');
  if (has('discard-build')) seen.add('rebuild');
  else if (has('discard-build-nothing')) seen.add('tick');
  if (has('military')) seen.add('drum');
  if (has('age-start')) seen.add('ageStart');
  const me = next.you.seat;
  if (has('game-over')) {
    const winners = next.view.result?.winners;
    seen.add(me === undefined || !winners ? 'chime' : winners.includes(me) ? 'fanfare' : 'defeat');
  }
  if (me === undefined) return seen;
  for (const e of events) {
    if (e.seat !== me) continue;
    if (e.kind === 'play') {
      const cue = playCue(e);
      if (cue) seen.add(cue);
    }
    if (e.kind === 'card-kept') seen.add('extraTurn');
  }
  if (gainsCoins(events, me, next.view.seats.length)) seen.add('coin');
  if (has('hands-passed')) seen.add('slide');
  if (confirmsOwn(next, prev, me)) seen.add('tick');
  if (!seen.has('extraTurn') && (opensOwnDecision(next, prev, me) || waitsOnlyFor(next, prev, me)))
    seen.add('yourTurn');
  return seen;
}

/** The vibration of a live frame: a conflict of the viewer's own, else a reveal or a decision of theirs. */
function frameHaptic(next: CueTable, seen: ReadonlySet<Cue>): readonly number[] | null {
  const me = next.you.seat;
  if (me === undefined) return null;
  if (next.events.some((e) => e.kind === 'military' && (e.seat === me || e.other === me)))
    return HAPTIC_CONFLICT;
  const own: readonly Cue[] = ['flip', 'rebuild', 'tick', 'yourTurn', 'extraTurn'];
  return own.some((c) => seen.has(c)) ? HAPTIC_REVEAL : null;
}

/** The cues and the vibration of a frame the table shows after `prev`. */
export function planTableCues(next: CueTable, prev: CueTable | null): TableCuePlan {
  if (!prev || !isLive(next, prev)) return NONE;
  const seen = frameCues(next, prev);
  const kept = new Set(PRIORITY.filter((c) => seen.has(c)).slice(0, CUE_BUDGET));
  const cues = ORDER.filter((c) => kept.has(c)).map((cue, i) => ({
    cue,
    delayMs: i * CUE_SPACING_MS
  }));
  return { cues, haptic: frameHaptic(next, seen) };
}
