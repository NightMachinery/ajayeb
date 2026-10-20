// Where the toasts sit in phone landscape (M11.board-layout-fixes). There
// the own column is a strip's width, so a long sentence at its foot grew up
// over the strip, and the structure fills the height between the top bar
// and the bottom edge, so no fixed place is clear of the cards: its foot
// holds the cards to take for most of an age, its head near the end. The
// toast list therefore tries a few lanes in order and takes the first where
// it covers no card that can be taken (an accessible slot, whoever is to
// move) and neither strip nor the turn bar: the structure's head, its foot,
// then the free foot of the opponent's column and of the own column. When
// none is clear, it lies over the top bar, under the safe area, where it
// hides only the track for its short life.
//
// clear() is the geometry alone; toastLane is the attachment that measures
// the list in each lane and places it, again when its size, the window or
// the frame changes. DOM only, no reactive state.

import type { Attachment } from 'svelte/attachments';

export interface Box {
  top: number;
  bottom: number;
  left: number; // physical-ok: viewport coordinates
  right: number; // physical-ok: viewport coordinates
}

/** Whether two boxes share more than a hairline. */
export function overlaps(a: Box, b: Box, tolerance = 0.5): boolean {
  return (
    Math.min(a.right, b.right) - Math.max(a.left, b.left) > tolerance && // physical-ok
    Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > tolerance
  );
}

/** Whether a box (the list measured in a lane) covers none of the obstacles; an empty list covers nothing. */
export function clear(box: Box, obstacles: readonly Box[]): boolean {
  return box.bottom - box.top <= 0 || !obstacles.some((o) => overlaps(box, o));
}

/** A place for the list: a box it is centred in, against one of its edges. */
interface Lane {
  name: 'head' | 'foot' | 'away' | 'home';
  box: DOMRect;
  edge: 'top' | 'bottom';
  /** The widest the list may be, in px. */
  width: number;
}

export interface LaneTargets {
  structure: HTMLElement | null;
  home: HTMLElement | null;
  away: HTMLElement | null;
  turn: HTMLElement | null;
  /** Reruns the placement when it changes (the frame's version). */
  version: number;
}

const GAP = 4;
const MAX_WIDTH = 384;

/** Places `lane`'s list (the attached element) in `lane`. */
function place(el: HTMLElement, lane: Lane): void {
  const width = Math.max(0, Math.min(lane.width, lane.box.width - 2 * GAP));
  el.style.width = `${width}px`;
  el.style.left = `${lane.box.left + (lane.box.width - width) / 2}px`; // physical-ok: viewport pixels
  if (lane.edge === 'top') {
    el.style.top = `${lane.box.top + GAP}px`;
    el.style.bottom = 'auto';
  } else {
    el.style.top = 'auto';
    el.style.bottom = `${window.innerHeight - lane.box.bottom + GAP}px`;
  }
}

/**
 * The attachment for the toast list's fixed wrapper in phone landscape. It
 * reads the targets lazily, so it can be attached before they mount.
 */
export function toastLane(targets: () => LaneTargets): Attachment<HTMLElement> {
  return (el) => {
    let frame = 0;
    const layout = (): void => {
      frame = 0;
      const { structure, home, away, turn } = targets();
      const list = el.firstElementChild as HTMLElement | null;
      if (!structure || !list) return;
      const lanes: Lane[] = [];
      const s = structure.getBoundingClientRect();
      const wide = Math.min(MAX_WIDTH, s.width);
      lanes.push(
        { name: 'head', box: s, edge: 'top', width: wide },
        { name: 'foot', box: s, edge: 'bottom', width: wide }
      );
      if (away) {
        const box = away.getBoundingClientRect();
        lanes.push({ name: 'away', box, edge: 'bottom', width: box.width });
      }
      if (home) {
        const box = home.getBoundingClientRect();
        lanes.push({ name: 'home', box, edge: 'bottom', width: box.width });
      }
      const obstacles: Box[] = [
        ...[...structure.querySelectorAll('[data-accessible]')].map((c) =>
          c.getBoundingClientRect()
        ),
        ...[home, away].flatMap((column) => {
          const strip = column?.querySelector('.player-strip');
          return strip ? [strip.getBoundingClientRect()] : [];
        }),
        ...(turn ? [turn.getBoundingClientRect()] : [])
      ];
      for (const lane of lanes) {
        place(el, lane);
        if (clear(list.getBoundingClientRect(), obstacles)) {
          el.dataset.toastLane = lane.name;
          return;
        }
      }
      // Over the top bar, under the safe area.
      const vw = document.documentElement.clientWidth;
      const width = Math.min(MAX_WIDTH, vw - 32);
      el.style.width = `${width}px`;
      el.style.left = `${(vw - width) / 2}px`; // physical-ok: viewport pixels
      el.style.top = `calc(var(--safe-t) + ${GAP}px)`;
      el.style.bottom = 'auto';
      el.dataset.toastLane = 'top';
    };
    const schedule = (): void => {
      if (!frame) frame = requestAnimationFrame(layout);
    };
    targets(); // read the version, so a new frame places the list again
    layout();
    const observer = new ResizeObserver(schedule);
    if (el.firstElementChild) observer.observe(el.firstElementChild);
    window.addEventListener('resize', schedule);
    return () => {
      if (frame) cancelAnimationFrame(frame);
      observer.disconnect();
      window.removeEventListener('resize', schedule);
    };
  };
}
