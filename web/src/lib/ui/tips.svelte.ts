// First-time tips (the hints design, 2.13, M10.HS5): the first time a hint
// mark appears on this device, a small callout beside it explains what it
// means, once. Device-local, like the local hide: a tip explains a rule
// and carries no game information, so it is not one of the game's hint
// settings (D23).
//
// The store keeps, in localStorage under ajayeb:tips, whether tips are off
// and the set of tips already seen. It is versionless JSON read leniently:
// unknown ids are dropped, a field of the wrong shape takes its default.
//
// Which tip is due is a pure function of the view: the pip codes the
// server sent for the screen's own seat and the other one (the same codes
// the board's pips show), the families this screen shows and whether hint
// marks are on the board. The callout (components/hints/FirstTips.svelte)
// then needs the mark itself on the page as its anchor.

import type { HintFamily, View } from '$lib/wire';
import type { RuleSectionId } from '$lib/components/rules/sections';

export const TIPS_KEY = 'ajayeb:tips';

/** Every tip, in priority order: the one that shows when several are due. */
export const TIP_IDS = ['win', 'pair', 'denyPair', 'winOpp', 'loot'] as const;
export type TipId = (typeof TIP_IDS)[number];

export function isTipId(value: unknown): value is TipId {
  return (TIP_IDS as readonly unknown[]).includes(value);
}

/** What the store keeps. */
export interface TipsData {
  /** The player turned first-time tips off (Settings). */
  readonly off: boolean;
  /** The tips already shown on this device. */
  readonly seen: readonly TipId[];
}

export const NO_TIPS_SEEN: TipsData = { off: false, seen: [] };

/** Reads stored tips data leniently. */
export function parseTips(value: unknown): TipsData {
  if (!value || typeof value !== 'object') return NO_TIPS_SEEN;
  const r = value as Record<string, unknown>;
  const seen = Array.isArray(r['seen']) ? r['seen'].filter(isTipId) : [];
  return {
    off: typeof r['off'] === 'boolean' ? r['off'] : false,
    seen: seen.filter((id, i) => seen.indexOf(id) === i)
  };
}

/** A tip, what it explains and where the callout points. */
export interface TipSpec {
  /** The hint family whose mark it explains. */
  readonly family: HintFamily;
  /** The pip code and side the mark carries (data-hint, data-hint-side). */
  readonly code: 'win' | 'pair' | 'loot';
  readonly side: 'own' | 'opp';
  /** The rules section of "More in Rules". */
  readonly rules: RuleSectionId;
}

export const TIPS: Readonly<Record<TipId, TipSpec>> = {
  win: { family: 'threats', code: 'win', side: 'own', rules: 'hints' },
  pair: { family: 'science', code: 'pair', side: 'own', rules: 'science' },
  denyPair: { family: 'science', code: 'pair', side: 'opp', rules: 'science' },
  winOpp: { family: 'threats', code: 'win', side: 'opp', rules: 'hints' },
  loot: { family: 'military', code: 'loot', side: 'own', rules: 'military' }
};

/** The CSS selector of the mark a tip points at (HintPip's attributes). */
export function tipSelector(id: TipId): string {
  const s = TIPS[id];
  return `[data-hint="${s.code}"][data-hint-side="${s.side}"]`;
}

/**
 * The tips whose mark the board shows now for `seat` (the screen's own
 * seat), in priority order: a face-up card whose hints carry the tip's
 * code for that seat (own) or the other one (opp), in a family the screen
 * shows, while hint marks are on the board. None once the game is over.
 */
export function tipsInView(
  view: Pick<View, 'structure' | 'result'>,
  seat: 0 | 1,
  families: ReadonlySet<HintFamily>,
  boardMarks: boolean
): TipId[] {
  if (!boardMarks || view.result) return [];
  const has = (code: TipSpec['code'], who: 0 | 1): boolean =>
    view.structure.some(
      (slot) => slot.state === 'faceUp' && !!slot.hints?.[who].codes.includes(code)
    );
  const other = seat === 0 ? 1 : 0;
  return TIP_IDS.filter((id) => {
    const s = TIPS[id];
    return families.has(s.family) && has(s.code, s.side === 'own' ? seat : other);
  });
}

/** The tip to show next: the first of `inView` not seen yet, or null (also when tips are off). */
export function dueTip(inView: readonly TipId[], data: TipsData): TipId | null {
  if (data.off) return null;
  return inView.find((id) => !data.seen.includes(id)) ?? null;
}

/** The part of Storage this store uses. */
export type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;

function defaultStorage(): StorageLike | null {
  try {
    return globalThis.localStorage ?? null;
  } catch {
    return null;
  }
}

function read(storage: StorageLike | null): unknown {
  try {
    const raw = storage?.getItem(TIPS_KEY);
    return raw ? (JSON.parse(raw) as unknown) : null;
  } catch {
    return null; // blocked storage or bad JSON reads as nothing seen
  }
}

/** This device's tips: whether they are on, and which were seen. */
export class TipsStore {
  #storage: StorageLike | null;
  value = $state.raw<TipsData>(NO_TIPS_SEEN);

  constructor(storage: StorageLike | null = defaultStorage()) {
    this.#storage = storage;
    this.value = parseTips(read(storage));
  }

  /** Records that a tip was shown, so it never shows again. */
  markSeen(id: TipId): void {
    if (this.value.seen.includes(id)) return;
    this.#save({ ...this.value, seen: [...this.value.seen, id] });
  }

  /** Turns the tips off, or on again. */
  setOff(off: boolean): void {
    this.#save({ ...this.value, off });
  }

  /** Forgets every tip seen, so each shows once more. */
  resetSeen(): void {
    this.#save({ ...this.value, seen: [] });
  }

  #save(next: TipsData): void {
    this.value = next;
    try {
      this.#storage?.setItem(TIPS_KEY, JSON.stringify(next));
    } catch {
      // Blocked or full: the choice still holds for this page.
    }
  }
}

/** The app's one tips store. */
export const tips = new TipsStore();
