// Display helpers for online rooms: the room code a player types or pastes,
// the invite link, and the presence changes the game screen announces. They
// only re-shape what the server sent (the room's seats and their
// `connected` flags); nothing here decides anything about the game.

import type { RoomInfo, State } from '$lib/wire';

/** The characters of a room code (the server's store/code.go): no I, L, O, 0 or 1. */
export const CODE_ALPHABET = 'ABCDEFGHJKMNPQRSTUVWXYZ23456789';

/** How many characters a room code has. */
export const CODE_LENGTH = 6;

const CODE_RE = new RegExp(`^[${CODE_ALPHABET}]{${CODE_LENGTH}}$`);

/**
 * What a player typed or pasted into the code field, as a code: the code
 * part of a pasted invite link (…/play/K7QX2M), upper case, with spaces,
 * dashes and other punctuation dropped, cut to the code's length. Letters
 * and digits outside the alphabet are kept, so the field shows what was
 * typed and isRoomCode can say it is not a code.
 */
export function normalizeCodeInput(raw: string): string {
  let text = raw;
  const at = text.lastIndexOf('/play/');
  if (at >= 0) text = text.slice(at + '/play/'.length).split(/[/?#]/)[0] ?? '';
  return text
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, '')
    .slice(0, CODE_LENGTH);
}

/** Whether a string is a room code the server could have made. */
export function isRoomCode(code: string): boolean {
  return CODE_RE.test(code);
}

/**
 * One of the suggested display names (a comma separated message), chosen
 * by `random` (0 to 1); empty when the list is.
 */
export function suggestName(list: string, random: number = Math.random()): string {
  const names = list
    .split(',')
    .map((n) => n.trim())
    .filter(Boolean);
  if (names.length === 0) return '';
  return names[Math.min(names.length - 1, Math.floor(random * names.length))] ?? '';
}

/**
 * Whether the room page asks for a name before it joins `code`. Only a
 * first-time visitor (no stored name) opening a room this browser has
 * never seen is asked: a join takes a seat under the name at once, so the
 * question must come first. A room this browser created or played (Home's
 * vs-computer and pass-and-play games, a dev room, a rematch) is in
 * `rooms` from the moment it is made, and opens straight away.
 */
export function needsName(
  hasName: boolean,
  code: string,
  rooms: readonly { readonly code: string }[]
): boolean {
  return !hasName && !rooms.some((r) => r.code === code);
}

/** The invite link of a room: the page's origin and the room's path. */
export function inviteUrl(origin: string, path: string): string {
  return new URL(path, origin).href;
}

export type Seat = 0 | 1;

/** A seat's presence as the strips show it. */
export type Presence = 'connected' | 'away';

/**
 * The presence of each seat a human holds, by seat, in an online room;
 * undefined for a seat nobody holds yet, a bot's, and in other modes.
 */
export function seatPresence(
  room: Pick<RoomInfo, 'mode' | 'seats'> | null | undefined
): [Presence | undefined, Presence | undefined] {
  const out: [Presence | undefined, Presence | undefined] = [undefined, undefined];
  if (!room || room.mode !== 'online') return out;
  for (const s of room.seats) {
    if ((s.seat === 0 || s.seat === 1) && s.kind === 'human') {
      out[s.seat] = s.connected ? 'connected' : 'away';
    }
  }
  return out;
}

/** Something about the other player's presence worth a toast. */
export interface PresenceChange {
  kind: 'joined' | 'left' | 'back';
  seat: Seat;
}

/**
 * What changed in the presence of the seats this connection does not hold,
 * from one state frame of an online room to the next: a seat taken (the
 * friend joined the lobby), a player who dropped, and one who came back.
 * Nothing for another room, another mode or the first frame.
 */
export function presenceChanges(prev: State | null, next: State): PresenceChange[] {
  if (!prev || prev.code !== next.code || next.room.mode !== 'online') return [];
  const out: PresenceChange[] = [];
  for (const now of next.room.seats) {
    const seat = now.seat;
    if ((seat !== 0 && seat !== 1) || now.you || now.kind !== 'human') continue;
    const before = prev.room.seats.find((s) => s.seat === seat);
    if (!before) continue;
    if (before.kind === 'open') out.push({ kind: 'joined', seat });
    else if (before.connected && !now.connected) out.push({ kind: 'left', seat });
    else if (!before.connected && now.connected) out.push({ kind: 'back', seat });
  }
  return out;
}

/**
 * One SVG path that draws the dark modules of a QR code, one unit per
 * module: each horizontal run of dark modules is one rectangle. The quiet
 * zone is the caller's (the viewBox and the padding around it).
 */
export function qrPath(modules: readonly (readonly boolean[])[]): string {
  const parts: string[] = [];
  modules.forEach((row, y) => {
    let x = 0;
    while (x < row.length) {
      if (!row[x]) {
        x++;
        continue;
      }
      let run = 1;
      while (row[x + run]) run++;
      parts.push(`M${x} ${y}h${run}v1h-${run}z`);
      x += run;
    }
  });
  return parts.join('');
}
