// What the table screen derives from a frame, without rules: who sits
// where around the viewer, each seat's status badge, how a hand card's
// server choices group and which badge they earn, and the printed facts the
// catalog adds (a card's colour, a board's stages). Every price, legality
// and score comes from the frame; nothing here adds or compares costs
// beyond choosing the cheapest of the totals the server sent.

import type { Choice, Disabled, Seat } from '$lib/wire/haftgane.gen';
import {
  hgBoards,
  hgDesigns,
  type HgBoard,
  type HgColour,
  type HgDesign,
  type HgEffect,
  type HgResource,
  type HgStage
} from '$lib/wire/haftgane-catalog.gen';
import type { PlatformHuman, TableState } from '$lib/wire/platform.gen';

/** The order of colour counts and city rows: resources first, then civic,
 *  commerce, military, science and guilds, as phone-layout.md draws them. */
export const COLOURS = [
  'brown',
  'grey',
  'blue',
  'yellow',
  'red',
  'green',
  'purple'
] as const satisfies readonly HgColour[];

/** The turns of an Age, for the top bar's "turn of six": a printed fact of
 *  the game (seven cards dealt, the last one discarded), not a rule the
 *  screen applies. */
export const TURNS_PER_AGE = 6;

export function design(slug: string): HgDesign | undefined {
  return (hgDesigns as Record<string, HgDesign | undefined>)[slug];
}

export function board(slug: string): HgBoard | undefined {
  return (hgBoards as Record<string, HgBoard | undefined>)[slug];
}

/** A board side's stages, or none while the side is not chosen. */
export function stagesOf(seat: Pick<Seat, 'board' | 'side'>): readonly HgStage[] {
  const b = board(seat.board);
  if (!b) return [];
  return seat.side === 'night' ? b.night : seat.side === 'day' ? b.day : [];
}

// The seat ring ---------------------------------------------------------------

/**
 * The seats around the anchor: the viewer's own seat, or for a spectator
 * the seat they follow (seat 0 until they pick one). Left is the next seat (the one hands pass to in Ages I and
 * III), right the previous one, and far lists the rest from the left
 * neighbour's side to the right neighbour's (relative 2 up to n - 2).
 */
export interface Ring {
  anchor: Seat;
  left: Seat;
  right: Seat;
  far: Seat[];
  /** The anchor is the viewer's own seat. */
  own: boolean;
}

export function ring(seats: readonly Seat[], follow: number | null = null): Ring | null {
  const n = seats.length;
  if (n < 3) return null;
  const self = seats.find((s) => s.relation === 'self');
  const anchor = self ?? seats.find((s) => s.seat === follow) ?? seats[0]!;
  const rel = (s: Seat) => (self ? s.relative : (s.seat - anchor.seat + n) % n);
  const at = (r: number) => seats.find((s) => rel(s) === r)!;
  const far = seats.filter((s) => rel(s) >= 2 && rel(s) <= n - 2).sort((a, b) => rel(a) - rel(b));
  return { anchor, left: at(1), right: at(n - 1), far, own: !!self };
}

/** Landscape splits the far seats into two side columns, each in ring
 *  order: the start column holds the left neighbour's half and is drawn
 *  bottom up (it climbs away from the left neighbour), the end column the
 *  right neighbour's half, drawn top down toward the right neighbour. */
export function farColumns(far: readonly Seat[]): { start: Seat[]; end: Seat[] } {
  const half = Math.ceil(far.length / 2);
  return { start: far.slice(0, half), end: far.slice(half) };
}

// Names and statuses ---------------------------------------------------------

export type SeatStatus = 'confirmed' | 'bot' | 'choosing' | 'away' | 'idle';

/** A seat's badge: confirmed, a bot or a human still choosing, a human
 *  who is away, or nothing to decide this point. */
export function seatStatus(frame: TableState, seat: number): SeatStatus {
  const point = frame.point;
  if (!point || !point.required.includes(seat)) return 'idle';
  if (point.confirmed.includes(seat)) return 'confirmed';
  const p = frame.seats[seat];
  if (p?.kind === 'bot') return 'bot';
  const human = frame.humans.find((h) => h.userId === p?.userId);
  return human && !human.connected ? 'away' : 'choosing';
}

/** The human in a seat, if a human holds it. */
export function humanAt(frame: TableState, seat: number): PlatformHuman | undefined {
  const p = frame.seats[seat];
  return p?.kind === 'human' ? frame.humans.find((h) => h.userId === p.userId) : undefined;
}

// Roles and spectators ---------------------------------------------------------

/** The role tag a human carries: "creator" for the creator while an admin,
 *  "admin" for any other admin, "temporary" for the temporary admin. */
export type Role = 'creator' | 'admin' | 'temporary';

export function roleOf(h: PlatformHuman | undefined): Role | null {
  if (!h) return null;
  if (h.admin) return h.creator ? 'creator' : 'admin';
  return h.temporary ? 'temporary' : null;
}

/** The humans in the room who hold no seat, in the frame's order. */
export function spectatorsOf(frame: TableState): PlatformHuman[] {
  return frame.humans.filter(
    (h) => !frame.seats.some((s) => s.kind === 'human' && s.userId === h.userId)
  );
}

/** The seat a user held earlier in this game. The frame names it only to
 *  an admin (you.previousSeats), so for anyone else this is always null. */
export function previousSeatOf(frame: TableState, userId: string): number | null {
  return frame.you.previousSeats?.find((p) => p.userId === userId)?.seat ?? null;
}

/**
 * The role buttons this viewer gets for another user, read from the roles
 * the frame lists, as the lobby reads them: a real admin may make any human
 * who is not one an admin, and may remove an admin they promoted, or, as
 * the creator, any admin but the creator. A temporary admin gets none. The
 * server checks every request again.
 */
export function roleActions(
  frame: TableState,
  userId: string
): { promote: boolean; demote: boolean } {
  const me = frame.humans.find((h) => h.userId === frame.you.userId);
  const them = frame.humans.find((h) => h.userId === userId);
  if (!me?.admin || !them || them.userId === me.userId || frame.status !== 'playing')
    return { promote: false, demote: false };
  return {
    promote: !them.admin,
    demote: them.admin && !them.creator && (me.creator || them.promotedBy === me.userId)
  };
}

/** How many of the seats this point asks have confirmed. */
export function chosenCount(frame: TableState): { done: number; of: number } {
  const point = frame.point;
  if (!point) return { done: 0, of: 0 };
  return { done: point.confirmed.length, of: point.required.length };
}

// Colours and production -------------------------------------------------------

/** How many cards of each colour a city holds, in COLOURS order. */
export function colourCounts(built: readonly string[]): { colour: HgColour; count: number }[] {
  const counts = new Map<HgColour, number>();
  for (const slug of built) {
    const c = design(slug)?.colour;
    if (c) counts.set(c, (counts.get(c) ?? 0) + 1);
  }
  return COLOURS.map((colour) => ({ colour, count: counts.get(colour) ?? 0 }));
}

/** One printed source of resources: a single kind, or one of several. */
export interface Making {
  res: readonly HgResource[];
  /** Units for a single kind; always 1 for a choice. */
  n: number;
  choice: boolean;
}

/**
 * What a city's printed cards and built stages make, from the catalog: the
 * board's starting resource, each card's production and each built
 * stage's. A display of printed symbols, as the city sheet lists them.
 */
export function makings(seat: Pick<Seat, 'board' | 'side' | 'stages' | 'built'>): Making[] {
  const out: Making[] = [];
  const b = board(seat.board);
  if (b) out.push({ res: [b.start], n: 1, choice: false });
  const add = (fx: readonly HgEffect[]) => {
    for (const e of fx) {
      if (e.kind === 'produce' && e.res) out.push({ res: e.res, n: e.n ?? 1, choice: false });
      if (e.kind === 'produce-choice' && e.res) out.push({ res: e.res, n: 1, choice: true });
    }
  };
  for (const slug of seat.built) add(design(slug)?.effects ?? []);
  for (const st of stagesOf(seat).slice(0, seat.stages)) add(st.effects);
  return out;
}

// A hand card's choices --------------------------------------------------------

/** The server's choices and disabled entries for one design in the hand. */
export interface CardChoices {
  builds: Choice[];
  stages: Choice[];
  sells: Choice[];
  disabled: Disabled[];
}

export function cardChoices(
  slug: string,
  choices: readonly Choice[],
  disabled: readonly Disabled[]
): CardChoices {
  const mine = choices.filter((c) => c.design === slug);
  return {
    builds: mine.filter((c) => c.kind === 'build' || c.kind === 'build-discarded'),
    stages: mine.filter((c) => c.kind === 'stage'),
    sells: mine.filter((c) => c.kind === 'sell'),
    disabled: disabled.filter((d) => d.design === slug)
  };
}

/** The badge a hand card shows for building it. */
export type Afford =
  /** Free through a chain, a wonder or the discard ability. */
  | { kind: 'free'; via: string }
  /** The cheapest offered build: its total, and whether it buys from a neighbour. */
  | { kind: 'paid'; total: number; trade: boolean }
  /** Offered with nothing to pay. */
  | { kind: 'none' }
  | { kind: 'built' }
  /** Not offered: the reason and, for coins, how many are missing. */
  | { kind: 'blocked'; reason: string; shortBy: number };

/** The cheapest of the server's totals; ties keep the server's order. */
export function cheapest(options: readonly Choice[]): Choice | undefined {
  let best: Choice | undefined;
  for (const c of options) if (!best || c.total < best.total) best = c;
  return best;
}

export function afford(cc: CardChoices): Afford | null {
  const free = cc.builds.find((c) => c.free !== '');
  if (free) return { kind: 'free', via: free.free };
  const best = cheapest(cc.builds);
  if (best) {
    if (best.total === 0) return { kind: 'none' };
    return { kind: 'paid', total: best.total, trade: best.left + best.right > 0 };
  }
  const why = cc.disabled.find((d) => d.kind === 'build');
  if (!why) return null;
  if (why.reason === 'already-built') return { kind: 'built' };
  return { kind: 'blocked', reason: why.reason, shortBy: why.shortBy };
}

/** Whether any hand card may go to the next wonder stage now. */
export function stageOpen(choices: readonly Choice[]): boolean {
  return choices.some((c) => c.kind === 'stage');
}
