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
//
// Besides the pip tips, five explain other marks (M10.F3): a looting token
// within reach and the dashed ring a selected card's ghost draws around it
// (ghostLoot), the chain badge of a card the seat builds free (freeChain),
// the hourglass of the race for the last wonder (hourglass), the guild
// back of Age III (guildBack) and the laurel round Law while it would be a
// sixth symbol (lawLaurel). The chain badge and the guild back are drawn
// whatever the hint settings, so their tips need no family.

import type { HintFamily, LootHint, View } from '$lib/wire';
import type { RuleSectionId } from '$lib/components/rules/sections';

export const TIPS_KEY = 'ajayeb:tips';

/** Every tip, in priority order: the one that shows when several are due. */
export const TIP_IDS = [
  'win',
  'pair',
  'denyPair',
  'winOpp',
  'loot',
  'lawLaurel',
  'ghostLoot',
  'hourglass',
  'freeChain',
  'guildBack'
] as const;
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
  /** The hint family whose mark it explains; none for a mark the board
   *  always draws (the chain badge, the guild back). */
  readonly family?: HintFamily;
  /** Whether its mark is a hint mark on the board (the local hide's switch
   *  removes it). */
  readonly boardMark: boolean;
  /** A pip tip: the pip code and side the mark carries (data-hint,
   *  data-hint-side). */
  readonly code?: 'win' | 'pair' | 'loot';
  readonly side?: 'own' | 'opp';
  /** The rules section of "More in Rules". */
  readonly rules: RuleSectionId;
}

export const TIPS: Readonly<Record<TipId, TipSpec>> = {
  win: { family: 'threats', boardMark: true, code: 'win', side: 'own', rules: 'hints' },
  pair: { family: 'science', boardMark: true, code: 'pair', side: 'own', rules: 'science' },
  denyPair: { family: 'science', boardMark: true, code: 'pair', side: 'opp', rules: 'science' },
  winOpp: { family: 'threats', boardMark: true, code: 'win', side: 'opp', rules: 'hints' },
  loot: { family: 'military', boardMark: true, code: 'loot', side: 'own', rules: 'military' },
  lawLaurel: { family: 'science', boardMark: true, rules: 'science' },
  ghostLoot: { family: 'military', boardMark: true, rules: 'military' },
  hourglass: { family: 'wonders', boardMark: true, rules: 'wonders' },
  freeChain: { boardMark: false, rules: 'costs' },
  guildBack: { boardMark: false, rules: 'guilds' }
};

/** What the tips read of the view. */
export type TipView = Pick<View, 'structure' | 'result'> &
  Partial<Pick<View, 'players' | 'hints' | 'boardTokens'>>;

/** The looting token a ghostLoot tip points at: within reach of a card in
 *  view, the other seat's first (the seat's own loot), else its own. */
function lootInReach(view: TipView, seat: 0 | 1): LootHint | undefined {
  const can = (who: 0 | 1): boolean =>
    view.structure.some(
      (s) => s.state === 'faceUp' && s.accessible && !!s.hints?.[who].codes.includes('loot')
    );
  // Positive spaces are on seat 1's side: seat 0's shields take those.
  const side = (who: 0 | 1): LootHint | undefined =>
    (view.hints?.looting ?? [])
      .filter((l) => l.wouldLose > 0 && Math.sign(l.at) === (who === 0 ? 1 : -1))
      .sort((a, b) => Math.abs(a.at) - Math.abs(b.at))[0];
  const other = seat === 0 ? 1 : 0;
  return (can(seat) ? side(seat) : undefined) ?? (can(other) ? side(other) : undefined);
}

/**
 * The CSS selector of the mark a tip points at: HintPip's attributes for a
 * pip tip, else the mark's own (the track's looting token for ghostLoot,
 * which needs the view and the seat).
 */
export function tipSelector(id: TipId, view?: TipView, seat: 0 | 1 = 0): string {
  const s = TIPS[id];
  if (s.code && s.side) return `[data-hint="${s.code}"][data-hint-side="${s.side}"]`;
  switch (id) {
    case 'lawLaurel':
      return '.progress-token-row [data-hint="law"]';
    case 'ghostLoot': {
      const loot = view ? lootInReach(view, seat) : undefined;
      return loot ? `[data-loot-at="${loot.at}"]` : '[data-loot-at] [data-present]';
    }
    case 'hourglass':
      return '[data-hint="hourglass"], [data-hourglass]';
    case 'freeChain':
      return '.card-structure [data-afford="chain"]';
    case 'guildBack':
      return '.card-structure [data-back="guild"]';
    default:
      return '[data-first-tip-none]';
  }
}

/** Whether the board shows the mark of a tip that is not a pip's. */
function markInView(id: TipId, view: TipView, seat: 0 | 1): boolean {
  switch (id) {
    case 'lawLaurel':
      return (
        !!view.boardTokens?.includes('law') &&
        !!view.players?.some((p) => p.hints?.lawWins === true)
      );
    case 'ghostLoot':
      return lootInReach(view, seat) !== undefined;
    case 'hourglass':
      return (
        view.hints?.wondersLeft === 1 &&
        !!view.players?.some((p) => p.wonders.some((w) => !w.built && !w.removed))
      );
    case 'freeChain':
      return view.structure.some(
        (s) => s.state === 'faceUp' && s.accessible && s.costs?.[seat].chainFree === true
      );
    case 'guildBack':
      return view.structure.some((s) => s.state === 'faceDown' && s.back === 'guild');
    default:
      return false;
  }
}

/**
 * The tips whose mark the board shows now for `seat` (the screen's own
 * seat), in priority order: a face-up card whose hints carry the tip's
 * code for that seat (own) or the other one (opp), in a family the screen
 * shows, while hint marks are on the board. None once the game is over.
 */
export function tipsInView(
  view: TipView,
  seat: 0 | 1,
  families: ReadonlySet<HintFamily>,
  boardMarks: boolean
): TipId[] {
  if (view.result) return [];
  const has = (code: 'win' | 'pair' | 'loot', who: 0 | 1): boolean =>
    view.structure.some(
      (slot) => slot.state === 'faceUp' && !!slot.hints?.[who].codes.includes(code)
    );
  const other = seat === 0 ? 1 : 0;
  return TIP_IDS.filter((id) => {
    const s = TIPS[id];
    if (s.boardMark && !boardMarks) return false;
    if (s.family && !families.has(s.family)) return false;
    if (s.code && s.side) return has(s.code, s.side === 'own' ? seat : other);
    return markInView(id, view, seat);
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
