// The app Settings sheet's persisted preferences: theme lives in its own
// module (theme.svelte.ts, key ajayeb:theme) because it predates this file
// and is read before the first paint by the inline bootstrap in
// src/app.html; everything else the Settings screen offers lives here,
// under ajayeb:settings. Values are plain, versionless JSON: a field this
// build does not recognise is dropped and replaced with its default, the
// same way parseSavedRooms (session/rooms.svelte.ts) treats a malformed
// saved room, so an older or newer build sharing the same browser never
// crashes on the other's shape.

import { cancelHaptic } from '$lib/fx/haptic-pending';
import { i18n, isLocale, type Locale } from '$lib/i18n/locale.svelte';

export const ANIMATION_SPEEDS = ['normal', 'fast', 'relaxed', 'off'] as const;
export type AnimationSpeed = (typeof ANIMATION_SPEEDS)[number];

export const CONFIRM_MOVES_OPTIONS = ['onlineOnly', 'always', 'never'] as const;
export type ConfirmMoves = (typeof CONFIRM_MOVES_OPTIONS)[number];

export const CARD_TEXT_SIZES = ['normal', 'large'] as const;
export type CardTextSize = (typeof CARD_TEXT_SIZES)[number];

export const SOUND_VOLUMES = ['quiet', 'medium', 'loud'] as const;
/** The Sound volume step (fx/sfx.ts maps each step to a gain). */
export type SoundVolume = (typeof SOUND_VOLUMES)[number];

export const ORIENTATIONS = ['face', 'fixed'] as const;
/** Hot-seat board orientation (UX design §6): 'face' turns the board to the
 *  active player, 'fixed' keeps Sun at the bottom. */
export type Orientation = (typeof ORIENTATIONS)[number];

export interface Settings {
  /** Base FX duration multiplier; see ui/motion.svelte.ts for the effective
   *  speed, which (D21) is simply this preference: the OS's own
   *  prefers-reduced-motion is deliberately never read. */
  animationSpeed: AnimationSpeed;
  /** Whether a move needs a second tap to confirm (ui/ConfirmButton). */
  confirmMoves: ConfirmMoves;
  /** A pattern added to each card colour's band, for colour-blind play. */
  colorPatterns: boolean;
  cardText: CardTextSize;
  /** The hot-seat sheet's last choices, which the game screen (M5.S04)
   *  reads: whether to show the pass-device interstitial, and how to orient
   *  the board. null until the first hot-seat game; the sheet then offers
   *  the device default (on and 'face' for a coarse pointer or a phone). */
  hotseatPassDevice: boolean | null;
  hotseatOrientation: Orientation | null;
  /** Synthesized sound cues (fx/), off by default. Its own switch: it does
   *  not follow animationSpeed. */
  sound: boolean;
  soundVolume: SoundVolume;
  /** Short vibrations through navigator.vibrate where the device has it;
   *  the Settings screen hides the switch elsewhere. */
  haptics: boolean;
  /** The opponent move summary (components/game/MoveSummary.svelte): a
   *  popup of what the other side just did opens by itself after their
   *  move. Off, it opens only from the top bar's button. */
  showOpponentMove: boolean;
  /** The app's language (i18n/locale.svelte.ts LOCALES), chosen in Settings
   *  and never detected from the browser while only English is complete.
   *  Text and direction switch together once its messages have loaded. */
  locale: Locale;
}

export const DEFAULT_SETTINGS: Settings = {
  animationSpeed: 'normal',
  confirmMoves: 'onlineOnly',
  colorPatterns: true,
  cardText: 'normal',
  hotseatPassDevice: null,
  hotseatOrientation: null,
  sound: false,
  soundVolume: 'medium',
  haptics: true,
  showOpponentMove: true,
  locale: 'en'
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

function isOrientation(value: unknown): value is Orientation {
  return (ORIENTATIONS as readonly unknown[]).includes(value);
}

function isSoundVolume(value: unknown): value is SoundVolume {
  return (SOUND_VOLUMES as readonly unknown[]).includes(value);
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
    cardText: isCardTextSize(r['cardText']) ? r['cardText'] : DEFAULT_SETTINGS.cardText,
    hotseatPassDevice:
      typeof r['hotseatPassDevice'] === 'boolean'
        ? r['hotseatPassDevice']
        : DEFAULT_SETTINGS.hotseatPassDevice,
    hotseatOrientation: isOrientation(r['hotseatOrientation'])
      ? r['hotseatOrientation']
      : DEFAULT_SETTINGS.hotseatOrientation,
    sound: typeof r['sound'] === 'boolean' ? r['sound'] : DEFAULT_SETTINGS.sound,
    soundVolume: isSoundVolume(r['soundVolume']) ? r['soundVolume'] : DEFAULT_SETTINGS.soundVolume,
    haptics: typeof r['haptics'] === 'boolean' ? r['haptics'] : DEFAULT_SETTINGS.haptics,
    showOpponentMove:
      typeof r['showOpponentMove'] === 'boolean'
        ? r['showOpponentMove']
        : DEFAULT_SETTINGS.showOpponentMove,
    locale: isLocale(r['locale']) ? r['locale'] : DEFAULT_SETTINGS.locale
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
  #applyLocale: ((locale: Locale) => void) | undefined;
  value = $state<Settings>(DEFAULT_SETTINGS);

  /** applyLocale, when given, is called with the stored locale at once and
   *  again on every save, so the app's language follows this store without
   *  an effect (the app's store passes i18n.set). */
  constructor(
    storage: StorageLike | null = defaultStorage(),
    applyLocale?: (locale: Locale) => void
  ) {
    this.#storage = storage;
    this.#applyLocale = applyLocale;
    this.value = parseSettings(readRaw(storage));
    applyLocale?.(this.value.locale);
  }

  /** Updates one field and persists the whole object. */
  set<K extends keyof Settings>(key: K, value: Settings[K]): void {
    this.#save({ ...this.value, [key]: value });
  }

  /** Updates several fields at once, with one write. */
  update(patch: Partial<Settings>): void {
    this.#save({ ...this.value, ...patch });
  }

  /** Restores every field to its default. */
  reset(): void {
    this.#save({ ...DEFAULT_SETTINGS });
  }

  #save(next: Settings): void {
    if (!next.haptics) cancelHaptic();
    this.value = next;
    this.#applyLocale?.(next.locale);
    try {
      this.#storage?.setItem(SETTINGS_KEY, JSON.stringify(next));
    } catch {
      // Blocked or full: the choice still holds for this page.
    }
  }
}

/** The app's one settings store, read and written from anywhere (Home's
 *  gear icon, the /settings route, and later the in-room TopBar). */
export const settings = new SettingsStore(defaultStorage(), (locale) => i18n.set(locale));
