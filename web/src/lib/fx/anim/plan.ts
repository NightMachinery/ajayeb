// The animation planner: a frame's live events, and the two views around
// them, turned into a timeline of steps (UX design section 5, "Animations").
// Pure: it reads only what the server sent and decides nothing about the
// rules. The runner (runner.ts) finds the elements each step moves and
// plays it; this file only says what moves, when, and for how long, at
// normal speed. The runner scales every offset and duration with the
// game's own Animation setting (scaleMs, D21).
//
// A frame's steps, in the order the events report them. First the lead,
// the thing the player did, starting at once:
//
//   take     a card leaves its structure slot: built into the city,
//            discarded for coins, or tucked under a wonder (build, discard,
//            wonder);
//   rebuild  a card leaves the discard pile for the city (the Mausoleum);
//   destroy  a card leaves the opponent's city for the discard pile;
//   token    a progress token flies to its taker's city, from the board or
//            the Great Library's offer;
//   draft    a wonder tile flies from the draft offer to its picker's row
//            (a frame may carry two: a pick and the round's last, given
//            without a choice), one after the other.
//
// Then what follows from it, once the lead lands (`after`):
//
//   flip     a face-down card turns face up (reveal), one after another;
//   coins    a seat's coins change: a float and a counter tween (pay, coins,
//            the coins of a discard, a trade's coins to the opponent; the
//            amount is the views' own before and after, less any looting);
//   pawn     the military pawn walks cell by cell from `from` to `to`;
//   loot     the pawn enters a looting token's cell: the token flips and
//            fades, and the victim's coins drop with a loss float, at the
//            moment the pawn gets there;
//   glow     a science pair: the pair's symbol glows in the taker's tracker;
//   draw     the Great Library deals one token of its offer: face up to the
//            drawer (the event names it only for the drawer), a back to
//            everyone else;
//   deal     a new age: the old structure fades out and the new one is
//            dealt card by card (it starts with the frame, beside the take
//            of the age's last card). When the age opens on the starter
//            choice (Ages II and III), whose dialog covers the board, the
//            deal waits for the frame that answers it (starterChosen), as
//            the dialog closes; the old structure is gone by then, so that
//            deal has no fade;
//   victory  the end of the game: a banner over the result screen, after
//            everything else of the frame has played.
//
// Live events only, the rule fx/cues.ts follows: nothing on the first frame
// of a room, a resync, an undo or a frame that carries a history (the
// caller passes cuePrev(prev, next) as `prev`).
//
// `blockingMs` is the part input waits for: the lead's flight up to its
// landing (the last LAND_MS of it, when the card only settles, do not hold
// input), never more than HOLD_MS. Everything after it (the landing, flips,
// floats, the pawn, the deal's tail, the banner) plays on while the player
// may already move; a new frame finishes it at once. A dealt card stays
// hidden, so unclickable, until it lands.

import type { GameEvent } from '$lib/wire';
import { HAPTIC_LOOT, HAPTIC_MILITARY, HAPTIC_TAKE, isLive, type Cue } from '../cues';

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
/** The end of a flight, when the card settles: input is back by then. */
export const LAND_MS = 80;
/** The most a frame may hold input, at normal speed (450 ms is the budget). */
export const BLOCK_CAP_MS = 350;
/** What a frame holds input at most: a take's flight up to its landing. */
export const HOLD_MS = TAKE_MS - LAND_MS;

/** A card leaving the discard pile for a city, or a city for the pile. */
export const REBUILD_MS = 420;
export const DESTROY_MS = 560;
/** A progress token's flight to its taker's city. */
export const TOKEN_MS = 420;
/** A draft tile's flight, and the gap before the next one of the frame. */
export const DRAFT_MS = 420;
export const DRAFT_STAGGER_MS = 160;
/** A looting token: its flip, then its fade. */
export const LOOT_FLIP_MS = 280;
export const LOOT_FADE_MS = 260;
/** A science pair's glow. */
export const GLOW_MS = 900;
/** The Great Library's deal: each token's flight, the gap between two, and
 *  how long a back stays up before it fades. */
export const DRAW_MS = 420;
export const DRAW_STAGGER_MS = 90;
export const DRAW_HOLD_MS = 450;
export const DRAW_FADE_MS = 240;
/** The age deal: the old structure's fade, the wait before the first card,
 *  each card's flight and the gap between two cards. */
export const DEAL_FADE_MS = 200;
export const DEAL_LEAD_MS = 120;
export const DEAL_EACH_MS = 240;
export const DEAL_STAGGER_MS = 30;
/** A structure's cards when the view does not say (an age has 20). */
export const DEAL_CARDS = 20;
/** The victory banner, from its entrance to its fade. */
export const VICTORY_MS = 1700;

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

export interface RebuildStep extends StepTime {
  kind: 'rebuild';
  seat: Seat;
  card: string;
}

export interface DestroyStep extends StepTime {
  kind: 'destroy';
  /** The seat that destroyed it; the card was the other seat's. */
  seat: Seat;
  card: string;
}

export interface TokenStep extends StepTime {
  kind: 'token';
  seat: Seat;
  token: string;
  /** Taken from the Great Library's offer rather than the board. */
  library: boolean;
}

export interface DraftStep extends StepTime {
  kind: 'draft';
  seat: Seat;
  wonder: string;
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

export interface LootStep extends StepTime {
  kind: 'loot';
  /** The seat that loses the coins. */
  seat: Seat;
  /** The token's space on the track (LootingView.at), when the views say. */
  cell?: number;
  /** The coins lost (the event's), for the loss float. */
  amount: number;
  /** The victim's counter just before and after this token. */
  from: number;
  to: number;
  /** The token's flip and fade, and the counter's tween; `dur` is the float's. */
  flip: number;
  fade: number;
  tween: number;
}

export interface GlowStep extends StepTime {
  kind: 'glow';
  seat: Seat;
  symbol: string;
}

export interface DrawStep extends StepTime {
  kind: 'draw';
  /** The drawer, and the token's place in the offer (0 to 2). */
  seat: Seat;
  place: number;
  /** The token, only when this viewer may see it (the drawer). */
  token?: string;
  /** The flight, and how long a back stays and fades. */
  each: number;
  hold: number;
  fade: number;
}

export interface DealStep extends StepTime {
  kind: 'deal';
  age: number;
  /** The old structure's fade; the first card waits `lead`. */
  fade: number;
  lead: number;
  /** Each card's flight, and the gap between two cards. */
  each: number;
  stagger: number;
}

export interface VictoryStep extends StepTime {
  kind: 'victory';
  /** The victory kind (GameEvent.victory), for the banner's tint. */
  victory?: string;
  /** The winner; absent for a shared victory. */
  winner?: Seat;
}

export type Step =
  | TakeStep
  | RebuildStep
  | DestroyStep
  | TokenStep
  | DraftStep
  | FlipStep
  | CoinStep
  | PawnStep
  | LootStep
  | GlowStep
  | DrawStep
  | DealStep
  | VictoryStep;
export type StepKind = Step['kind'];

/** The step fields that are times, which the speed scales (speed.ts). */
export const TIME_FIELDS = [
  'at',
  'dur',
  'tween',
  'flip',
  'fade',
  'each',
  'hold',
  'lead',
  'stagger'
] as const;

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

/** The part of a view the planner reads. */
export interface PlanView {
  /** Each seat's coins, for the counter's before and after. */
  players: readonly { coins: number }[];
  /** The looting tokens, to tell which ones the pawn took. */
  military?: { looting: readonly { at: number; present: boolean }[] };
  /** The structure, to count the cards an age deals. */
  structure?: readonly { state: string }[];
  /** The phase and age, to hold a deal for the starter choice. */
  phase?: string;
  age?: number;
}

/** The part of a state frame the planner reads. */
export interface PlanFrame {
  code: string;
  version: number;
  events: readonly GameEvent[];
  view?: PlanView;
}

function seatOf(n: number | undefined): Seat | null {
  return n === 0 || n === 1 ? n : null;
}

function other(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
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

/** The lead step of one event, if it is something the player did. */
function leadOf(e: GameEvent): Step | null {
  const take = takeOf(e);
  if (take) return { ...take, at: 0, dur: TAKE_MS };
  const seat = seatOf(e.seat);
  if (seat === null) return null;
  switch (e.kind) {
    case 'rebuild':
      return e.card ? { kind: 'rebuild', seat, card: e.card, at: 0, dur: REBUILD_MS } : null;
    case 'destroy':
      return e.card ? { kind: 'destroy', seat, card: e.card, at: 0, dur: DESTROY_MS } : null;
    case 'token':
      return e.token
        ? { kind: 'token', seat, token: e.token, library: !!e.fromLibrary, at: 0, dur: TOKEN_MS }
        : null;
    case 'draft':
      return e.wonder ? { kind: 'draft', seat, wonder: e.wonder, at: 0, dur: DRAFT_MS } : null;
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
  if (e.kind === 'pay' && (e.toOpponent ?? 0) > 0) return [seat, other(seat)];
  return [seat];
}

/** The looting tokens present before and gone after, nearest the pawn's start first. */
function takenLoot(prev: PlanView | undefined, next: PlanView | undefined, from: number): number[] {
  const before = prev?.military?.looting;
  const after = next?.military?.looting;
  if (!before || !after) return [];
  return before
    .filter((l) => l.present && after.some((a) => a.at === l.at && !a.present))
    .map((l) => l.at)
    .sort((a, b) => Math.abs(a - from) - Math.abs(b - from));
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

  // The lead: one take, rebuild, destroy or token, or the frame's drafts.
  let lead: Step | null = null;
  let drafts = 0;
  for (const e of events) {
    const step = leadOf(e);
    if (!step) continue;
    if (step.kind === 'draft') {
      if (lead && lead.kind !== 'draft') continue;
      step.at = drafts * DRAFT_STAGGER_MS;
      steps.push(step);
      lead ??= step;
      drafts += 1;
    } else if (!lead) {
      lead = step;
      steps.push(step);
    }
  }
  const after = lead ? lead.at + lead.dur : 0;

  let flips = 0;
  events.forEach((e, i) => {
    if (e.kind !== 'reveal' || i < firstLive || e.slot === undefined || !e.card) return;
    const at = after + FLIP_DELAY_MS + flips * FLIP_STAGGER_MS;
    steps.push({ kind: 'flip', slot: e.slot, card: e.card, at, dur: FLIP_MS });
    flips += 1;
  });

  // The pawn's walk, and the looting tokens it takes on the way.
  const pawns = events.filter(
    (e) => e.kind === 'pawn' && e.from !== undefined && e.to !== undefined
  );
  const first = pawns[0];
  const last = pawns[pawns.length - 1];
  const pusher = seatOf(first?.seat);
  let pawn: PawnStep | null = null;
  if (first && last && pusher !== null && first.from !== last.to) {
    const from = first.from ?? 0;
    const to = last.to ?? 0;
    pawn = {
      kind: 'pawn',
      seat: pusher,
      from,
      to,
      at: after,
      dur: Math.abs(to - from) * PAWN_STEP_MS
    };
  }
  const loots = events.filter((e) => e.kind === 'loot' && seatOf(e.seat) !== null);
  const lootLoss: [number, number] = [0, 0];
  for (const e of loots) lootLoss[seatOf(e.seat) ?? 0] += e.amount ?? 0;

  const seen = new Set<Seat>();
  for (const e of events) {
    for (const seat of coinSeats(e)) {
      if (seen.has(seat)) continue;
      seen.add(seat);
      const from = prev.view?.players[seat]?.coins;
      const shown = next.view?.players[seat]?.coins;
      // The counter stops short of the looting, which its own step shows.
      const to = shown === undefined ? undefined : shown + lootLoss[seat];
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

  if (pawn) steps.push(pawn);

  if (loots.length > 0) {
    const start = pawn?.from ?? 0;
    const cells = takenLoot(prev.view, next.view, start);
    // Each victim's counter, from before its first token down to the view's.
    const left: [number, number] = [lootLoss[0], lootLoss[1]];
    loots.forEach((e, i) => {
      const seat = seatOf(e.seat) ?? 0;
      const amount = e.amount ?? 0;
      const cell = cells[i];
      const reach =
        pawn && cell !== undefined
          ? pawn.at + Math.abs(cell - start) * PAWN_STEP_MS
          : (pawn ? pawn.at + pawn.dur : after) + i * LOOT_FLIP_MS;
      const from = (next.view?.players[seat]?.coins ?? 0) + left[seat];
      left[seat] -= amount;
      steps.push({
        kind: 'loot',
        seat,
        ...(cell !== undefined ? { cell } : {}),
        amount,
        from,
        to: from - amount,
        at: reach,
        dur: FLOAT_MS,
        flip: LOOT_FLIP_MS,
        fade: LOOT_FADE_MS,
        tween: TWEEN_MS
      });
    });
  }

  const pairs = new Set<string>();
  for (const e of events) {
    const seat = seatOf(e.seat);
    if (e.kind !== 'pair' || seat === null || !e.symbol || pairs.has(`${seat}:${e.symbol}`)) {
      continue;
    }
    pairs.add(`${seat}:${e.symbol}`);
    steps.push({ kind: 'glow', seat, symbol: e.symbol, at: after, dur: GLOW_MS });
  }

  let draws = 0;
  for (const e of events) {
    const seat = seatOf(e.seat);
    if (e.kind !== 'libraryDraw' || seat === null) continue;
    steps.push({
      kind: 'draw',
      seat,
      place: e.place ?? draws,
      // Only what this viewer's event says: the drawer's names the token.
      ...(e.token ? { token: e.token } : {}),
      at: after + draws * DRAW_STAGGER_MS,
      dur: DRAW_MS + DRAW_HOLD_MS + DRAW_FADE_MS,
      each: DRAW_MS,
      hold: DRAW_HOLD_MS,
      fade: DRAW_FADE_MS
    });
    draws += 1;
  }

  // The starter choice's dialog covers the board: its answer deals instead.
  const opened = events.find((e) => e.kind === 'ageStart');
  const age =
    opened && next.view?.phase !== 'chooseStarter'
      ? { age: opened.age ?? 0, fade: DEAL_FADE_MS }
      : events.some((e) => e.kind === 'starterChosen')
        ? { age: next.view?.age ?? 0, fade: 0 }
        : // The Presenter may coalesce a zero-delay burst before capture.
          // Its final live snapshot still visibly enters this age, even if
          // the final draft/starter event was in a dropped frame.
          next.view?.phase === 'play' &&
            (next.view.age ?? 0) > 0 &&
            (prev.view?.phase === 'wonderDraft' ||
              prev.view?.phase === 'chooseStarter' ||
              (prev.view?.age !== undefined && next.view.age !== prev.view.age))
          ? { age: next.view.age ?? 0, fade: prev.view?.phase === 'play' ? DEAL_FADE_MS : 0 }
          : null;
  if (age) {
    const cards = next.view?.structure?.filter((s) => s.state !== 'empty').length ?? DEAL_CARDS;
    steps.push({
      kind: 'deal',
      age: age.age,
      at: 0,
      dur: DEAL_LEAD_MS + Math.max(0, cards - 1) * DEAL_STAGGER_MS + DEAL_EACH_MS,
      fade: age.fade,
      lead: DEAL_LEAD_MS,
      each: DEAL_EACH_MS,
      stagger: DEAL_STAGGER_MS
    });
  }

  const over = events.find((e) => e.kind === 'gameOver');
  if (over) {
    // After the rest of the frame: the pawn reaching a capital, a looting, a glow.
    const ends = steps.map((s) =>
      s.kind === 'pawn' || s.kind === 'glow' ? s.at + s.dur : s.kind === 'loot' ? s.at + s.flip : 0
    );
    const winner = seatOf(over.winner);
    steps.push({
      kind: 'victory',
      ...(over.victory ? { victory: over.victory } : {}),
      ...(winner !== null ? { winner } : {}),
      at: Math.max(after, ...ends),
      dur: VICTORY_MS
    });
  }

  if (steps.length === 0) return EMPTY_TIMELINE;
  const totalMs = Math.max(...steps.map((s) => s.at + s.dur));
  const blockingMs = lead ? Math.min(lead.dur - LAND_MS, HOLD_MS) : 0;
  return { steps, blockingMs, totalMs };
}

/** When a lead step lands: a flight's end; a destroyed card leaves at once. */
function landing(s: Step): number | undefined {
  switch (s.kind) {
    case 'take':
    case 'rebuild':
    case 'draft':
    case 'token':
      return s.at + s.dur;
    case 'destroy':
      return s.at;
    default:
      return undefined;
  }
}

function firstOf<K extends StepKind>(
  timeline: Timeline,
  kind: K
): Extract<Step, { kind: K }> | undefined {
  return timeline.steps.find((s): s is Extract<Step, { kind: K }> => s.kind === kind);
}

/** Cue anchors, at the timeline's own speed. Coincident anchors are spaced by retimeCues. */
export function cueOffsets(timeline: Timeline): Partial<Record<Cue, number>> {
  const out: Partial<Record<Cue, number>> = {};
  const set = (cue: Cue, at: number | undefined) => {
    if (out[cue] === undefined && at !== undefined) out[cue] = at;
  };
  for (const s of timeline.steps) {
    if (s.kind === 'take') {
      set('slide', landing(s));
      if (s.dest === 'wonder') set('wonder', landing(s));
    }
    if (s.kind === 'rebuild') set('rebuild', landing(s));
    if (s.kind === 'destroy') set('destroy', s.at);
  }
  set('flip', firstOf(timeline, 'flip')?.at);
  set('coin', firstOf(timeline, 'coins')?.at);
  set('drum', firstOf(timeline, 'pawn')?.at);
  set('loot', firstOf(timeline, 'loot')?.at);
  const token = firstOf(timeline, 'token');
  set('token', token ? landing(token) : undefined);
  const victory = firstOf(timeline, 'victory');
  set('chime', victory?.at ?? firstOf(timeline, 'glow')?.at);
  set('fanfare', victory?.at);
  set('defeat', victory?.at);
  set('libraryDraw', firstOf(timeline, 'draw')?.at);
  const deal = firstOf(timeline, 'deal');
  set('ageStart', deal ? deal.at + deal.lead : undefined);
  if (timeline.steps.length > 0) {
    set('extraTurn', timeline.totalMs);
    set('yourTurn', timeline.blockingMs);
  }
  const draft = firstOf(timeline, 'draft');
  set('tick', draft ? landing(draft) : undefined);
  return out;
}

/**
 * When a frame's vibration should start, at the timeline's own speed: the
 * looting one with the first looting flip, the military one with the
 * pawn's first step, the take's when the taken card (or rebuilt card, or
 * draft tile) lands. 0 when the timeline has no such step.
 */
export function hapticOffset(timeline: Timeline, pattern: readonly number[]): number {
  if (pattern === HAPTIC_LOOT) {
    return firstOf(timeline, 'loot')?.at ?? firstOf(timeline, 'pawn')?.at ?? 0;
  }
  if (pattern === HAPTIC_MILITARY) return firstOf(timeline, 'pawn')?.at ?? 0;
  if (pattern === HAPTIC_TAKE) {
    const lead = timeline.steps.find(
      (s) => s.kind === 'take' || s.kind === 'rebuild' || s.kind === 'draft'
    );
    return lead ? (landing(lead) ?? 0) : 0;
  }
  return 0;
}
