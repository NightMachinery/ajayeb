// What the admin menus, the role tags and the spectator list read from a
// frame: each person's name, role, presence and (for an admin only) the
// seat they held earlier, the table's bot levels, and the seat the viewer
// held before it watched. The frame's roles and seats say who may do what;
// the server checks every request again, and nothing here adds a rule.

import type { PlatformHuman, TableState } from '$lib/wire/platform.gen';
import { humanAt, previousSeatOf, roleActions, roleOf, spectatorsOf, type Role } from './model';
import { humanName, persona } from './seat-names';

export interface SeatPerson {
  userId: string;
  name: string;
  role: Role | null;
  connected: boolean;
  /** The seat this person held earlier, when the frame says (admins only). */
  previous?: number | null;
}

export type SeatAction =
  | { kind: 'bot'; level: string }
  | { kind: 'human'; userId: string }
  | { kind: 'promote'; userId: string }
  | { kind: 'demote'; userId: string };

/** The bot levels a seat accepts (the frame's botLevels, the ones the
 *  table froze at its start), in order, each with its persona name. */
export function botLevels(levels: readonly string[]): { level: string; persona: string }[] {
  return levels.map((level) => ({ level, persona: persona(level) }));
}

export function person(frame: TableState, h: PlatformHuman): SeatPerson {
  return {
    userId: h.userId,
    name: humanName(h),
    role: roleOf(h),
    connected: h.connected,
    previous: previousSeatOf(frame, h.userId)
  };
}

/** Everyone watching, connected or not, in the frame's order. */
export function watchers(frame: TableState): SeatPerson[] {
  return spectatorsOf(frame).map((h) => person(frame, h));
}

/** The spectators an admin may seat: those connected (the server seats
 *  only a connected spectator). */
export function seatable(frame: TableState): SeatPerson[] {
  return watchers(frame).filter((p) => p.connected);
}

/** The human holding a seat, as a person, or null for a bot. */
export function occupant(frame: TableState, seat: number): SeatPerson | null {
  const h = humanAt(frame, seat);
  return h ? person(frame, h) : null;
}

/** The viewer is an admin, real or temporary: the seat powers. */
export function isAdmin(frame: TableState): boolean {
  const me = frame.humans.find((h) => h.userId === frame.you.userId);
  return !!me && (me.admin || me.temporary);
}

/** The role buttons this viewer gets for each human, by user id. */
export function roleMap(frame: TableState): Record<string, { promote: boolean; demote: boolean }> {
  return Object.fromEntries(frame.humans.map((h) => [h.userId, roleActions(frame, h.userId)]));
}

/**
 * Who plays the seat this viewer held, while the viewer watches a game in
 * play: the server names that seat in the viewer's own frame
 * (you.formerSeat), on every tab and device, and only while the game is in
 * play and the viewer holds no seat. Null otherwise.
 */
export function formerOf(
  frame: TableState
): { seat: number; by: 'bot' | 'human'; name: string } | null {
  const seat = frame.you.formerSeat;
  if (seat === undefined || frame.status !== 'playing') return null;
  const s = frame.seats.find((x) => x.seat === seat);
  if (!s) return null;
  const h = humanAt(frame, seat);
  return h ? { seat, by: 'human', name: humanName(h) } : { seat, by: 'bot', name: '' };
}

/** The viewer is the temporary admin. */
export function isTemporary(frame: TableState): boolean {
  return !!frame.humans.find((h) => h.userId === frame.you.userId)?.temporary;
}
