// The rooms this browser has played in, newest first, kept in localStorage
// under ajayeb:rooms so Home can offer to continue them. Only the code, the
// mode, bot level, age and when the room was last seen are kept. No cards
// or hidden information are stored here.
// A room opened through a device link also keeps its claim (./claim.ts), so
// that reopening it from Home plays the same seat again.

import type { BotLevel, Mode } from '$lib/wire';
import { isValidClaim } from './claim';
import type { StorageLike } from './identity';

export const ROOMS_KEY = 'ajayeb:rooms';

/** How many rooms are remembered; the oldest are dropped first. */
export const MAX_SAVED_ROOMS = 20;

export interface SavedRoom {
  code: string;
  mode: Mode;
  /** When this browser last saw the room, in ms since the epoch. */
  lastSeen: number;
  /** The device link's claim this browser plays the room with, if any. */
  claim?: string;
  bot?: BotLevel;
  age?: number;
  /** A tutorial room (Home labels it "Tutorial"); absent for any other. */
  tutorial?: true;
}

const MODES: readonly string[] = ['online', 'hotseat', 'ai'];

function isSavedRoom(value: unknown): value is SavedRoom {
  if (!value || typeof value !== 'object') return false;
  const r = value as Record<string, unknown>;
  return (
    typeof r['code'] === 'string' &&
    /^[A-Z0-9]{1,32}$/.test(r['code']) &&
    typeof r['mode'] === 'string' &&
    MODES.includes(r['mode']) &&
    typeof r['lastSeen'] === 'number' &&
    Number.isFinite(r['lastSeen']) &&
    (r['claim'] === undefined || isValidClaim(r['claim']))
  );
}

function entry(
  code: string,
  mode: Mode,
  lastSeen: number,
  claim?: string,
  details: { bot?: unknown; age?: unknown; tutorial?: unknown } = {}
): SavedRoom {
  const item: SavedRoom =
    claim === undefined ? { code, mode, lastSeen } : { code, mode, lastSeen, claim };
  if (
    mode === 'ai' &&
    typeof details.bot === 'string' &&
    ['easy', 'normal', 'hard'].includes(details.bot)
  ) {
    item.bot = details.bot as BotLevel;
  }
  if (
    typeof details.age === 'number' &&
    Number.isInteger(details.age) &&
    details.age >= 0 &&
    details.age <= 3
  ) {
    item.age = details.age;
  }
  if (mode === 'ai' && details.tutorial === true) item.tutorial = true;
  return item;
}

/** Parses the stored list, dropping anything malformed. */
export function parseSavedRooms(raw: string | null): SavedRoom[] {
  if (!raw) return [];
  let data: unknown;
  try {
    data = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(data)) return [];
  const out: SavedRoom[] = [];
  for (const item of data) {
    if (!isSavedRoom(item) || out.some((r) => r.code === item.code)) continue;
    out.push(entry(item.code, item.mode, item.lastSeen, item.claim, item));
  }
  return sortRooms(out).slice(0, MAX_SAVED_ROOMS);
}

function sortRooms(rooms: SavedRoom[]): SavedRoom[] {
  return rooms.sort((a, b) => b.lastSeen - a.lastSeen);
}

function readRaw(storage: StorageLike | null): string | null {
  try {
    return storage?.getItem(ROOMS_KEY) ?? null;
  } catch {
    return null; // blocked storage reads as empty
  }
}

function defaultStorage(): StorageLike | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

export class SavedRooms {
  #storage: StorageLike | null;
  /** The remembered rooms, newest first. */
  list = $state.raw<readonly SavedRoom[]>([]);

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    this.list = parseSavedRooms(readRaw(storage));
  }

  /** Reloads another tab's list without writing it back. */
  refresh(raw: string | null = readRaw(this.#storage)): void {
    this.list = parseSavedRooms(raw);
  }

  /** Stores the latest server-sent summary without moving the room. */
  updateDetails(code: string, details: Pick<SavedRoom, 'bot' | 'age' | 'tutorial'>): void {
    const room = this.list.find((r) => r.code === code);
    if (
      !room ||
      (room.bot === details.bot && room.age === details.age && room.tutorial === details.tutorial)
    ) {
      return;
    }
    this.#save(
      this.list.map((r) =>
        r.code === code ? entry(r.code, r.mode, r.lastSeen, r.claim, details) : r
      )
    );
  }

  /**
   * Remembers a room as seen now (or at `at`), moving it to the front. A
   * claim replaces the room's saved one; null drops it; undefined (the
   * default) keeps whatever claim the room had.
   */
  remember(code: string, mode: Mode, at: number = Date.now(), claim?: string | null): void {
    const kept = claim === undefined ? this.claimFor(code) : claim;
    const item = entry(
      code,
      mode,
      at,
      kept ?? undefined,
      this.list.find((r) => r.code === code)
    );
    if (!isSavedRoom(item)) return;
    const rest = this.list.filter((r) => r.code !== code);
    this.#save(sortRooms([item, ...rest]).slice(0, MAX_SAVED_ROOMS));
  }

  /** The claim this browser plays a room with, or null. */
  claimFor(code: string): string | null {
    return this.list.find((r) => r.code === code)?.claim ?? null;
  }

  /** Drops a room's claim (it stopped working), keeping the room. */
  dropClaim(code: string): void {
    const room = this.list.find((r) => r.code === code);
    if (!room?.claim) return;
    this.#save(
      this.list.map((r) => (r.code === code ? entry(r.code, r.mode, r.lastSeen, undefined, r) : r))
    );
  }

  /** Forgets a room, for example one the server no longer knows. */
  forget(code: string): void {
    if (!this.list.some((r) => r.code === code)) return;
    this.#save(this.list.filter((r) => r.code !== code));
  }

  #save(list: SavedRoom[]): void {
    this.list = list;
    try {
      this.#storage?.setItem(ROOMS_KEY, JSON.stringify(list));
    } catch {
      // Blocked or full: the list still holds for this page.
    }
  }
}
