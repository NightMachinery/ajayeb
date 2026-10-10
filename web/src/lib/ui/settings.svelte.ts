// The app Settings sheet's persisted preferences: theme lives in its own
// module (theme.svelte.ts, key ajayeb:theme) because it predates this file
// and is read before the first paint by the inline bootstrap in
// src/app.html; everything else the Settings screen offers lives here,
// under ajayeb:settings. Values are plain, versionless JSON: a field this
// build does not recognise is dropped and replaced with its default, the
// same way parseSavedRooms (session/rooms.svelte.ts) treats a malformed
// saved room, so an older or newer build sharing the same browser never
// crashes on the other's shape.

export const ANIMATION_SPEEDS = ['normal', 'fast', 'relaxed', 'off'] as const;
export type AnimationSpeed = (typeof ANIMATION_SPEEDS)[number];

export const CONFIRM_MOVES_OPTIONS = ['onlineOnly', 'always', 'never'] as const;
export type ConfirmMoves = (typeof CONFIRM_MOVES_OPTIONS)[number];

export const CARD_TEXT_SIZES = ['normal', 'large'] as const;
export type CardTextSize = (typeof CARD_TEXT_SIZES)[number];

export interface Settings {
  /** Base FX duration multiplier; see ui/motion.svelte.ts for the effective
   *  speed, which also factors in the OS's prefers-reduced-motion. */
  animationSpeed: AnimationSpeed;
  /** Whether a move needs a second tap to confirm (ui/ConfirmButton). */
  confirmMoves: ConfirmMoves;
  /** A pattern added to each card colour's band, for colour-blind play. */
  colorPatterns: boolean;
  cardText: CardTextSize;
}

export const DEFAULT_SETTINGS: Settings = {
  animationSpeed: 'normal',
  confirmMoves: 'onlineOnly',
  colorPatterns: true,
  cardText: 'normal'
};

export const SETTINGS_KEY = 'ajayeb:settings';

/** The part of Storage this module uses (matches session/identity.ts's StorageLike). */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

function isAnimationSpeed(value: unknown): value is AnimationSpeed {
  return (ANIMATION_SPEEDS as readonly unknown[]).includes(value);
}

function isConfirmMoves(value: unknown): value is ConfirmMoves {
  return (CONFIRM_MOVES_OPTIONS as readonly unknown[]).includes(value);
}

function isCardTextSize(value: unknown): value is CardTextSize {
  return (CARD_TEXT_SIZES as readonly unknown[]).includes(value);
}

/** Merges a possibly-malformed, possibly-partial value onto the defaults,
 *  keeping only the fields that have the right shape. */
export function parseSettings(raw: string | null): Settings {
  if (!raw) return { ...DEFAULT_SETTINGS };
  let data: unknown;
  try {
    data = JSON.parse(raw);
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
  if (!data || typeof data !== 'object') return { ...DEFAULT_SETTINGS };
  const r = data as Record<string, unknown>;
  return {
    animationSpeed: isAnimationSpeed(r['animationSpeed'])
      ? r['animationSpeed']
      : DEFAULT_SETTINGS.animationSpeed,
    confirmMoves: isConfirmMoves(r['confirmMoves'])
      ? r['confirmMoves']
      : DEFAULT_SETTINGS.confirmMoves,
    colorPatterns:
      typeof r['colorPatterns'] === 'boolean' ? r['colorPatterns'] : DEFAULT_SETTINGS.colorPatterns,
    cardText: isCardTextSize(r['cardText']) ? r['cardText'] : DEFAULT_SETTINGS.cardText
  };
}

function readRaw(storage: StorageLike | null): string | null {
  try {
    return storage?.getItem(SETTINGS_KEY) ?? null;
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

export class SettingsStore {
  #storage: StorageLike | null;
  value = $state<Settings>(DEFAULT_SETTINGS);

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    this.value = parseSettings(readRaw(storage));
  }

  /** Updates one field and persists the whole object. */
  set<K extends keyof Settings>(key: K, value: Settings[K]): void {
    this.#save({ ...this.value, [key]: value });
  }

  /** Restores every field to its default. */
  reset(): void {
    this.#save({ ...DEFAULT_SETTINGS });
  }

  #save(next: Settings): void {
    this.value = next;
    try {
      this.#storage?.setItem(SETTINGS_KEY, JSON.stringify(next));
    } catch {
      // Blocked or full: the choice still holds for this page.
    }
  }
}

/** The app's one settings store, read and written from anywhere (Home's
 *  gear icon, the /settings route, and later the in-room TopBar). */
export const settings = new SettingsStore();
