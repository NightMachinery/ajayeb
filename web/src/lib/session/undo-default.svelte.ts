// This device's "Allow undo" choice for new games (ajayeb:allowUndo in
// localStorage), one per mode that has undo: the bot and hot-seat sheets
// start their switch from it, save it when a game is created, and the
// Session's create reads it (app.svelte.ts), next to the hint settings'
// local default. The game itself keeps the value it was created with: the
// server fixes it for the game's life (docs/protocol.md, create).
// Versionless JSON, read leniently: a value of the wrong shape means "not
// chosen yet", which is on.

import type { Mode } from '$lib/wire';

export const UNDO_DEFAULT_KEY = 'ajayeb:allowUndo';

/** The modes with undo; an online game has none. */
export type UndoMode = Exclude<Mode, 'online'>;

/** The stored choice, by mode; a mode absent here is on. */
export type SavedUndo = Readonly<Partial<Record<UndoMode, boolean>>>;

/** The part of Storage this store uses. */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

function defaultStorage(): StorageLike | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

/** Reads a stored choice leniently. */
export function parseSavedUndo(raw: string | null): SavedUndo {
  if (!raw) return {};
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return {};
  }
  if (!value || typeof value !== 'object') return {};
  const r = value as Record<string, unknown>;
  const out: Partial<Record<UndoMode, boolean>> = {};
  for (const mode of ['ai', 'hotseat'] as const) {
    const v = r[mode];
    if (typeof v === 'boolean') out[mode] = v;
  }
  return out;
}

/** The "Allow undo" choice a new game starts from, by mode. */
export class UndoDefaultStore {
  #storage: StorageLike | null;
  saved = $state.raw<SavedUndo>({});

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    let raw: string | null = null;
    try {
      raw = storage?.getItem(UNDO_DEFAULT_KEY) ?? null;
    } catch {
      // Blocked storage reads as empty.
    }
    this.saved = parseSavedUndo(raw);
  }

  /** Whether a new game of `mode` allows undo: on until the player turns it off. */
  for(mode: UndoMode): boolean {
    return this.saved[mode] ?? true;
  }

  /** Remembers the choice of a game of `mode` just created. */
  save(mode: UndoMode, allow: boolean): void {
    this.saved = { ...this.saved, [mode]: allow };
    try {
      this.#storage?.setItem(UNDO_DEFAULT_KEY, JSON.stringify(this.saved));
    } catch {
      // Blocked or full: the choice still holds for this page.
    }
  }
}

/** The app's choice, read and written by the new-game sheets and the Session's create. */
export const undoDefault = new UndoDefaultStore();
