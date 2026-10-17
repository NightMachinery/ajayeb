// DOM helpers of the animation runner (runner.ts): rectangles, inert
// copies of elements, and placing a copy in the FX layer. Reads and copies
// only; nothing here changes an element of the page.

export interface Box {
  x: number;
  y: number;
  w: number;
  h: number;
}

export function boxOf(el: Element): Box {
  const r = el.getBoundingClientRect();
  return { x: r.left, y: r.top, w: r.width, h: r.height };
}

export function shown(box: Box): boolean {
  return box.w > 0 && box.h > 0;
}

/** A box of size w x h centred on (x, y). */
export function boxAround(x: number, y: number, w: number, h: number): Box {
  return { x: x - w / 2, y: y - h / 2, w, h };
}

export function centreOf(box: Box): { x: number; y: number } {
  return { x: box.x + box.w / 2, y: box.y + box.h / 2 };
}

/** A copy of `el` that nothing can find, click or read aloud. */
export function inertClone(el: Element): HTMLElement {
  const clone = el.cloneNode(true) as HTMLElement;
  makeInert(clone);
  return clone;
}

/** Strips what tests, drivers and assistive technology find an element by. */
export function makeInert(clone: HTMLElement): void {
  const strip = (node: Element) => {
    for (const name of [
      'data-card',
      'data-act',
      'data-slot',
      'data-pick',
      'id',
      'tabindex',
      'aria-describedby'
    ]) {
      node.removeAttribute(name);
    }
  };
  strip(clone);
  for (const node of clone.querySelectorAll('*')) strip(node);
  // ArtImage shows its art from its own load handler, which a copy does not
  // get: art not yet loaded when it was copied (a card just turned face up)
  // would stay transparent through the whole flip, so the copy shows it.
  for (const img of clone.querySelectorAll<HTMLImageElement>('img.art-image:not(.loaded)')) {
    const show = () => img.classList.add('loaded');
    img.loading = 'eager';
    if (img.complete && img.naturalWidth > 0) show();
    else img.addEventListener('load', show, { once: true });
  }
  clone.setAttribute('aria-hidden', 'true');
  clone.setAttribute('inert', '');
  clone.setAttribute('data-fx-clone', '');
}

/** Places a clone at the layer's origin, sized like its box; transforms move it. */
export function place(clone: HTMLElement, box: Box): void {
  const s = clone.style;
  s.position = 'absolute';
  s.insetBlockStart = '0';
  s.insetInlineStart = '0';
  s.margin = '0';
  s.width = `${box.w}px`;
  s.height = `${box.h}px`;
  s.setProperty('--cw', `${box.w}px`);
  s.setProperty('--ch', `${box.h}px`);
  s.transform = `translate(${box.x}px, ${box.y}px)`;
  s.zIndex = '';
}

/** A drawn element of the FX layer (not a copy of the page): inert, unread. */
export function drawn(tag: string, className: string): HTMLElement {
  const el = document.createElement(tag);
  el.className = className;
  el.setAttribute('aria-hidden', 'true');
  el.setAttribute('data-fx-clone', '');
  return el;
}

export function cssEscape(value: string): string {
  return typeof CSS !== 'undefined' && CSS.escape ? CSS.escape(value) : value;
}

/** Keeps a point inside a box (a destination scrolled out of its panel). */
export function clampInto(x: number, y: number, box: Box): { x: number; y: number } {
  return {
    x: Math.min(Math.max(x, box.x), box.x + box.w),
    y: Math.min(Math.max(y, box.y), box.y + box.h)
  };
}

/** The first element of `selectors`, in order, that is on screen inside `scope`. */
export function findShown(
  scope: ParentNode,
  selectors: readonly string[],
  keep: (el: HTMLElement) => boolean = () => true
): { el: HTMLElement; box: Box } | null {
  for (const sel of selectors) {
    for (const el of scope.querySelectorAll<HTMLElement>(sel)) {
      if (!keep(el)) continue;
      const box = boxOf(el);
      if (shown(box)) return { el, box };
    }
  }
  return null;
}
