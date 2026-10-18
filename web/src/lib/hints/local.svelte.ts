// This device's two hint stores (docs/dev/decisions.md D23), both in
// localStorage, both versionless JSON read leniently (a field of the wrong
// shape takes its default, as ui/settings.svelte.ts does):
//
//   - the local default (ajayeb:hints): the settings a game created on this
//     device starts from. It is written when a game is created and when its
//     admin changes the settings mid-game. The families (and the advisor's
//     strength) are shared by every mode, except the advisor's switch,
//     which keeps one value per mode because its
//     default differs (on against the bot, off in hot-seat and online
//     games), so a review turned on against the bot does not follow the
//     player into an online game.
//   - the local hide (ajayeb:hintsHidden): the families this player hides
//     on their own screen, and whether hint marks show on the board at all.
//     Display only: it can take hints away, never add one the game's
//     settings turned off (see visibleFamilies).

import type { HintFamily, HintSettings, Mode } from '$lib/wire';
import { defaultHints, isHintFamily, parseHints, type LocalHide } from './families';

export const HINT_DEFAULTS_KEY = 'ajayeb:hints';
export const HINT_HIDE_KEY = 'ajayeb:hintsHidden';

/** The part of Storage these stores use. */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

function defaultStorage(): StorageLike | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

function readJson(storage: StorageLike | null, key: string): unknown {
  let raw: string | null;
  try {
    raw = storage?.getItem(key) ?? null;
  } catch {
    return null; // blocked storage reads as empty
  }
  if (!raw) return null;
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return null;
  }
}

function writeJson(storage: StorageLike | null, key: string, value: unknown): void {
  try {
    storage?.setItem(key, JSON.stringify(value));
  } catch {
    // Blocked or full: the choice still holds for this page.
  }
}

const MODES = ['ai', 'hotseat', 'online'] as const satisfies readonly Mode[];

/** The local default as stored: the shared families and the advisor by mode. */
export interface SavedHints {
  /** The families of the last settings saved; the advisor here is unused. */
  readonly hints: HintSettings | null;
  readonly advisor: Readonly<Partial<Record<Mode, boolean>>>;
}

/** Reads a stored local default leniently. */
export function parseSavedHints(value: unknown): SavedHints {
  if (!value || typeof value !== 'object') return { hints: null, advisor: {} };
  const r = value as Record<string, unknown>;
  const hints =
    r['hints'] && typeof r['hints'] === 'object'
      ? parseHints(r['hints'], defaultHints('ai'))
      : null;
  const advisor: Partial<Record<Mode, boolean>> = {};
  const a = r['advisor'];
  if (a && typeof a === 'object') {
    for (const mode of MODES) {
      const v = (a as Record<string, unknown>)[mode];
      if (typeof v === 'boolean') advisor[mode] = v;
    }
  }
  return { hints, advisor };
}

/** The settings a new game of `mode` starts from, given what this device saved. */
export function startingHints(saved: SavedHints, mode: Mode): HintSettings {
  const base = saved.hints ? { ...saved.hints } : defaultHints(mode);
  base.advisor = saved.advisor[mode] ?? defaultHints(mode).advisor;
  return base;
}

/** `saved` after saving the settings of a game of `mode`. */
export function withSavedHints(saved: SavedHints, mode: Mode, h: HintSettings): SavedHints {
  return { hints: { ...h }, advisor: { ...saved.advisor, [mode]: h.advisor } };
}

/** The local default: the settings a game created on this device starts from. */
export class HintDefaultsStore {
  #storage: StorageLike | null;
  saved = $state.raw<SavedHints>({ hints: null, advisor: {} });

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    this.saved = parseSavedHints(readJson(storage, HINT_DEFAULTS_KEY));
  }

  /** The settings a new game of `mode` starts from. */
  for(mode: Mode): HintSettings {
    return startingHints(this.saved, mode);
  }

  /** Remembers the settings of a game of `mode` (created, or changed by its admin). */
  save(mode: Mode, h: HintSettings): void {
    this.saved = withSavedHints(this.saved, mode, h);
    writeJson(this.#storage, HINT_DEFAULTS_KEY, this.saved);
  }
}

/** Reads a stored local hide leniently: unknown families are dropped. */
export function parseLocalHide(value: unknown): LocalHide {
  if (!value || typeof value !== 'object') return { families: [], boardMarks: false };
  const r = value as Record<string, unknown>;
  const fams = Array.isArray(r['families']) ? r['families'].filter(isHintFamily) : [];
  return {
    families: fams.filter((f, i) => fams.indexOf(f) === i),
    boardMarks: typeof r['boardMarks'] === 'boolean' ? r['boardMarks'] : false
  };
}

/** The local hide: what this player hides on their own screen. */
export class HintHideStore {
  #storage: StorageLike | null;
  value = $state.raw<LocalHide>({ families: [], boardMarks: false });

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    this.value = parseLocalHide(readJson(storage, HINT_HIDE_KEY));
  }

  /** Hides a family on this screen, or shows it again. */
  setFamily(family: HintFamily, hidden: boolean): void {
    const rest = this.value.families.filter((f) => f !== family);
    this.#save({ ...this.value, families: hidden ? [...rest, family] : rest });
  }

  /** Hides every hint mark on the board, or shows them again. */
  setBoardMarks(hidden: boolean): void {
    this.#save({ ...this.value, boardMarks: hidden });
  }

  #save(next: LocalHide): void {
    this.value = next;
    writeJson(this.#storage, HINT_HIDE_KEY, next);
  }
}

/** The app's local default, read and written from the new-game sheets and the game menu. */
export const hintDefaults = new HintDefaultsStore();

/** The app's local hide. */
export const hintHide = new HintHideStore();
