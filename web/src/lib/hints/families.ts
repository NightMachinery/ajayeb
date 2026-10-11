// The hint families and the pure helpers around a game's hint settings
// (docs/dev/decisions.md D23). The settings themselves come from the
// server (RoomInfo.hints); this file only names the families, re-shapes
// the settings for the editor and the summary, and combines them with
// this device's local hide (local.svelte.ts). It decides no rule: whether
// a hint applies is the server's business, which sends only the families
// the game has on.

import type { HintFamily, HintSettings, Mode } from '$lib/wire';

/** Every hint family, in the order the server lists them (wire.HintFamilies). */
export const HINT_FAMILIES = [
  'prices',
  'economy',
  'science',
  'military',
  'threats',
  'tempo',
  'wonders',
  'tokens',
  'scoring',
  'counting',
  'recap',
  'advisor'
] as const satisfies readonly HintFamily[];

/** The family list must name every key of HintSettings, and nothing else. */
type MissingFamily = Exclude<keyof HintSettings, (typeof HINT_FAMILIES)[number]>;
const _everyFamily: MissingFamily extends never ? true : never = true;
void _everyFamily;

/** Whether a string is a hint family. */
export function isHintFamily(value: unknown): value is HintFamily {
  return (HINT_FAMILIES as readonly unknown[]).includes(value);
}

/**
 * The settings a game of `mode` starts with when nobody chose any: every
 * family on, except the advisor outside games against the bot. The same
 * defaults as the server's room.DefaultHints; the server applies its own
 * when a create names no settings.
 */
export function defaultHints(mode: Mode): HintSettings {
  return {
    prices: true,
    economy: true,
    science: true,
    military: true,
    threats: true,
    tempo: true,
    wonders: true,
    tokens: true,
    scoring: true,
    counting: true,
    recap: true,
    advisor: mode === 'ai'
  };
}

/** A copy of `h` with one family switched. */
export function withFamily(h: HintSettings, family: HintFamily, on: boolean): HintSettings {
  return { ...h, [family]: on };
}

/** Whether two settings switch every family the same way. */
export function sameHints(a: HintSettings, b: HintSettings): boolean {
  return HINT_FAMILIES.every((f) => a[f] === b[f]);
}

/** The families that are on, in family order. */
export function enabledFamilies(h: HintSettings): HintFamily[] {
  return HINT_FAMILIES.filter((f) => h[f]);
}

/**
 * What this device hides on its own screen: whole families, and with
 * `boardMarks` every hint mark drawn on the board (pips and glows), while
 * the sheets and tooltips still say what the game's families allow. Display
 * only: hiding never shows anything the game's settings turned off.
 */
export interface LocalHide {
  readonly families: readonly HintFamily[];
  readonly boardMarks: boolean;
}

export const NO_LOCAL_HIDE: LocalHide = { families: [], boardMarks: false };

/**
 * The families this screen shows: on in the game's settings and not hidden
 * here. The hint components of later slices read this, never the room's
 * settings alone, so a local hide can only take away.
 */
export function visibleFamilies(
  room: HintSettings | null | undefined,
  hide: LocalHide = NO_LOCAL_HIDE
): Set<HintFamily> {
  const out = new Set<HintFamily>();
  if (!room) return out;
  for (const f of HINT_FAMILIES) {
    if (room[f] && !hide.families.includes(f)) out.add(f);
  }
  return out;
}

/** The message key of a family's name (en/hints.ts). */
export type FamilyNameKey = `hints.family.${HintFamily}`;
/** The message key of a family's one-line description (en/hints.ts). */
export type FamilyHintKey = `hints.family.${HintFamily}.hint`;

/** The message keys of a family: its name and its one-line description. */
export function familyText(family: HintFamily): { name: FamilyNameKey; hint: FamilyHintKey } {
  return { name: `hints.family.${family}`, hint: `hints.family.${family}.hint` };
}

/** One row of the settings editor. */
export interface HintRow {
  readonly family: HintFamily;
  /** On in the settings shown (the game's, or the draft of a new game). */
  readonly on: boolean;
  /** Hidden on this screen (the local hide). */
  readonly hidden: boolean;
  readonly name: FamilyNameKey;
  readonly hint: FamilyHintKey;
}

/** The editor's rows: one per family, in family order. */
export function hintRows(h: HintSettings, hide: LocalHide = NO_LOCAL_HIDE): HintRow[] {
  return HINT_FAMILIES.map((family) => ({
    family,
    on: h[family],
    hidden: hide.families.includes(family),
    ...familyText(family)
  }));
}

/**
 * The read-only summary of a game's settings: everything on, everything
 * off, or the families that differ from "all on" (fewer to read). `off`
 * lists the families switched off when most are on; `on` the families
 * switched on when most are off.
 */
export type HintSummary =
  | { readonly kind: 'all' }
  | { readonly kind: 'none' }
  | { readonly kind: 'allBut'; readonly off: HintFamily[] }
  | { readonly kind: 'only'; readonly on: HintFamily[] };

export function summarizeHints(h: HintSettings): HintSummary {
  const on = enabledFamilies(h);
  if (on.length === HINT_FAMILIES.length) return { kind: 'all' };
  if (on.length === 0) return { kind: 'none' };
  const off = HINT_FAMILIES.filter((f) => !h[f]);
  return off.length <= on.length ? { kind: 'allBut', off } : { kind: 'only', on };
}

/**
 * Reads a settings object that may come from an older or newer build (the
 * local default): a family it lacks, or holds as a non-boolean, takes the
 * fallback's value, and unknown keys are dropped.
 */
export function parseHints(value: unknown, fallback: HintSettings): HintSettings {
  const out = { ...fallback };
  if (!value || typeof value !== 'object') return out;
  const r = value as Record<string, unknown>;
  for (const f of HINT_FAMILIES) {
    const v = r[f];
    if (typeof v === 'boolean') out[f] = v;
  }
  return out;
}
