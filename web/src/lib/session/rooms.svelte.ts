// The rooms this browser has played in, newest first, kept in localStorage
// under ajayeb:rooms so Home can offer to continue them. Only the code, the
// mode and when the room was last seen are kept: nothing about the game.

import type { Mode } from '$lib/wire';
import type { StorageLike } from './identity';

export const ROOMS_KEY = 'ajayeb:rooms';

/** How many rooms are remembered; the oldest are dropped first. */
export const MAX_SAVED_ROOMS = 20;

export interface SavedRoom {
  code: string;
  mode: Mode;
  /** When this browser last saw the room, in ms since the epoch. */
  lastSeen: number;
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
    Number.isFinite(r['lastSeen'])
  );
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
    out.push({ code: item.code, mode: item.mode, lastSeen: item.lastSeen });
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

  /** Remembers a room as seen now (or at `at`), moving it to the front. */
  remember(code: string, mode: Mode, at: number = Date.now()): void {
    const entry: SavedRoom = { code, mode, lastSeen: at };
    if (!isSavedRoom(entry)) return;
    const rest = this.list.filter((r) => r.code !== code);
    this.#save(sortRooms([entry, ...rest]).slice(0, MAX_SAVED_ROOMS));
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
