// The tutorial's anchors (the tutorial design, section 3.3): the elements a
// coach step points at, found by selector. A step names a list of
// selectors; the first match that is visible wins (a box with area, not
// visibility: hidden, not hidden by a move animation), because some
// anchors exist twice: the full military track and the compact one of the
// card sheet, the city panel's wonders and the phone strip's wonder button.
//
// New data-tut attributes exist only where no stable attribute did:
//
//   structure    CardStructure's root
//   turn-bar     TurnBar's root
//   home, away   GameScreen's area-home and area-away
//   track        MilitaryTrack's root (full and compact); its pawn is [data-pawn]
//   tokens       ProgressTokenRow's root
//   wonders      CityPanel's wonder area and the phone strip's WonderButton
//   draft-offer  the Draft screen's offer grid
//   age          the top bar's age chip
//
// The rest reuse the e2e contract: [data-card], [data-act], [data-pick],
// [data-tally="coins"], [data-control="wonder-step"], [data-card-sheet],
// [data-chain], [data-cost-breakdown], [data-starter-choice], [data-science].

/** The data-tut anchor ids. */
export const TUT_ANCHORS = [
  'structure',
  'turn-bar',
  'home',
  'away',
  'track',
  'tokens',
  'wonders',
  'draft-offer',
  'age'
] as const;
export type TutAnchor = (typeof TUT_ANCHORS)[number];

/** The selector of a data-tut anchor. */
export function tut(id: TutAnchor): string {
  return `[data-tut="${id}"]`;
}

/** The part of an element the visibility test reads. */
export interface Boxed {
  getBoundingClientRect(): { width: number; height: number };
  closest(selector: string): unknown;
}

/** Whether an element is drawn: a box with area, not visibility hidden,
 *  and not hidden while a move animation flies its clone (data-fx-hidden). */
export function isDrawn(
  el: Element & Boxed,
  style: (el: Element) => { visibility: string }
): boolean {
  const r = el.getBoundingClientRect();
  if (r.width <= 0 || r.height <= 0) return false;
  if (el.closest('[data-fx-hidden]')) return false;
  return style(el).visibility !== 'hidden';
}

/**
 * The first visible match of the selectors, in order (each selector's
 * matches in document order), or null.
 */
export function findAnchor(
  selectors: readonly string[],
  root: ParentNode = document,
  visible: (el: Element) => boolean = (el) => isDrawn(el, (e) => getComputedStyle(e))
): HTMLElement | null {
  for (const selector of selectors) {
    for (const el of root.querySelectorAll<HTMLElement>(selector)) {
      if (visible(el)) return el;
    }
  }
  return null;
}

/**
 * Scrolls the target's nearest vertical scroll containers (the desktop city
 * panel, the draft, a tall sheet) just enough to show it, as
 * scrollIntoView({block: 'nearest'}) would, but never the game screen's
 * overflow-hidden boxes nor anything sideways: the board never scrolls
 * horizontally.
 */
export function revealInScroller(el: HTMLElement): void {
  // Leave room for the target's outline and shadow, including its entrance.
  const margin = 12;
  let box = el.getBoundingClientRect();
  for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) {
    const overflow = getComputedStyle(p).overflowY;
    if ((overflow !== 'auto' && overflow !== 'scroll') || p.scrollHeight <= p.clientHeight) {
      continue;
    }
    const r = p.getBoundingClientRect();
    const top = r.top + margin;
    const bottom = r.bottom - margin;
    if (box.top < top) p.scrollTop -= top - box.top;
    else if (box.bottom > bottom) {
      p.scrollTop += Math.min(box.bottom - bottom, Math.max(0, box.top - top));
    }
    box = el.getBoundingClientRect();
  }
}
